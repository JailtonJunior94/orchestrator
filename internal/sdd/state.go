// Package sdd implementa o estado operacional versionado do fluxo SDD.
package sdd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const SchemaVersion = 2

type Status string

const (
	StatusDraft      Status = "draft"
	StatusApproved   Status = "approved"
	StatusStale      Status = "stale"
	StatusExecuting  Status = "executing"
	StatusBlocked    Status = "blocked"
	StatusNeedsInput Status = "needs_input"
	StatusFailed     Status = "failed"
	StatusDone       Status = "done"
)

type Artifact string

const (
	ArtifactPRD      Artifact = "prd"
	ArtifactTechSpec Artifact = "techspec"
	ArtifactTasks    Artifact = "tasks"
)

var ErrInvalidState = errors.New("estado SDD invalido")

type ArtifactState struct {
	Status   Status `json:"status"`
	SHA256   string `json:"sha256,omitempty"`
	Approved bool   `json:"approved"`
}

type Event struct {
	At      time.Time `json:"at"`
	RunID   string    `json:"run_id,omitempty"`
	TaskID  string    `json:"task_id,omitempty"`
	Attempt int       `json:"attempt,omitempty"`
	Action  string    `json:"action"`
	Detail  string    `json:"detail,omitempty"`
}

type State struct {
	SchemaVersion int                        `json:"schema_version"`
	RunID         string                     `json:"run_id"`
	Artifacts     map[Artifact]ArtifactState `json:"artifacts"`
	Events        []Event                    `json:"events"`
}

// TestProof liga um comando executado ao seu resultado verificável.
type TestProof struct {
	Command      string `json:"command"`
	ExitCode     int    `json:"exit_code"`
	OutputSHA256 string `json:"output_sha256"`
}

// CriterionProof exige evidência individual, evitando uma afirmação genérica de sucesso.
type CriterionProof struct {
	ID          string `json:"id"`
	EvidenceRef string `json:"evidence_ref"`
}

// ExecutionResult é o contrato compartilhado por executor, checkpoint e estado.
type ExecutionResult struct {
	SchemaVersion      int              `json:"schema_version"`
	RunID              string           `json:"run_id"`
	TaskID             string           `json:"task_id"`
	Attempt            int              `json:"attempt"`
	Status             Status           `json:"status"`
	BaseSHA            string           `json:"base_sha"`
	PatchSHA256        string           `json:"patch_sha256"`
	FinalStateSHA256   string           `json:"final_state_sha256"`
	CoverageRegression bool             `json:"coverage_regression"`
	Tests              []TestProof      `json:"tests"`
	Criteria           []CriterionProof `json:"criteria"`
	Evidence           []string         `json:"evidence"`
	ReviewVerdict      string           `json:"review_verdict"`
}

type Store struct {
	now func() time.Time
}

func NewStore() *Store {
	return &Store{now: time.Now}
}

func (s *Store) StatePath(prdDir string) string {
	return filepath.Join(prdDir, "sdd-state.json")
}

func (s *Store) Load(prdDir string) (State, error) {
	data, err := os.ReadFile(s.StatePath(prdDir))
	if err != nil {
		return State{}, fmt.Errorf("ler estado SDD: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decodificar estado SDD: %w", err)
	}
	if err := s.Validate(state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s *Store) Initialize(prdDir, runID string) (State, error) {
	if strings.TrimSpace(runID) == "" {
		return State{}, fmt.Errorf("%w: run_id ausente", ErrInvalidState)
	}
	state := State{
		SchemaVersion: SchemaVersion,
		RunID:         runID,
		Artifacts: map[Artifact]ArtifactState{
			ArtifactPRD:      {Status: StatusDraft},
			ArtifactTechSpec: {Status: StatusDraft},
			ArtifactTasks:    {Status: StatusDraft},
		},
	}
	state.Events = append(state.Events, Event{At: s.now().UTC(), RunID: runID, Action: "initialized"})
	if err := s.Save(prdDir, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s *Store) Save(prdDir string, state State) error {
	if err := s.Validate(state); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar estado SDD: %w", err)
	}
	data = append(data, '\n')
	path := s.StatePath(prdDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("criar diretorio de estado SDD: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".sdd-state-*")
	if err != nil {
		return fmt.Errorf("criar estado temporario: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("escrever estado temporario: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sincronizar estado temporario: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("fechar estado temporario: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("publicar estado SDD: %w", err)
	}
	return nil
}

func (s *Store) Approve(prdDir string, artifact Artifact) (State, error) {
	state, err := s.Load(prdDir)
	if err != nil {
		return State{}, err
	}
	if err := s.validateApproval(state, prdDir, artifact); err != nil {
		return State{}, err
	}
	path, err := s.artifactPath(prdDir, artifact)
	if err != nil {
		return State{}, err
	}
	digest, err := s.digestFile(path)
	if err != nil {
		return State{}, err
	}
	entry := state.Artifacts[artifact]
	entry.SHA256 = digest
	entry.Status = StatusApproved
	entry.Approved = true
	state.Artifacts[artifact] = entry
	state.Events = append(state.Events, Event{At: s.now().UTC(), RunID: state.RunID, Action: "approved", Detail: string(artifact)})
	if err := s.Save(prdDir, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s *Store) Invalidate(prdDir string, from Artifact) (State, error) {
	state, err := s.Load(prdDir)
	if err != nil {
		return State{}, err
	}
	downstream := s.downstream(from)
	if len(downstream) == 0 {
		return State{}, fmt.Errorf("%w: artefato %s nao possui descendentes", ErrInvalidState, from)
	}
	for _, artifact := range downstream {
		entry := state.Artifacts[artifact]
		if !entry.Approved {
			continue
		}
		entry.Status = StatusStale
		entry.Approved = false
		state.Artifacts[artifact] = entry
	}
	state.Events = append(state.Events, Event{At: s.now().UTC(), RunID: state.RunID, Action: "invalidated", Detail: string(from)})
	if err := s.Save(prdDir, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func (s *Store) Validate(state State) error {
	if state.SchemaVersion != SchemaVersion || strings.TrimSpace(state.RunID) == "" {
		return fmt.Errorf("%w: schema_version ou run_id invalido", ErrInvalidState)
	}
	for _, artifact := range []Artifact{ArtifactPRD, ArtifactTechSpec, ArtifactTasks} {
		entry, ok := state.Artifacts[artifact]
		if !ok || !s.validStatus(entry.Status) {
			return fmt.Errorf("%w: artefato %s invalido", ErrInvalidState, artifact)
		}
		if (entry.Status == StatusApproved) != entry.Approved {
			return fmt.Errorf("%w: status e aprovacao de %s inconsistentes", ErrInvalidState, artifact)
		}
		if entry.Approved && (entry.Status != StatusApproved || len(entry.SHA256) != sha256.Size*2) {
			return fmt.Errorf("%w: aprovacao de %s sem digest valido", ErrInvalidState, artifact)
		}
		if entry.SHA256 != "" && !s.validDigest(entry.SHA256) {
			return fmt.Errorf("%w: digest de %s invalido", ErrInvalidState, artifact)
		}
	}
	return nil
}

// ValidateExecutionResult garante que um done tenha todas as provas obrigatórias.
func (s *Store) ValidateExecutionResult(result ExecutionResult) error {
	if result.SchemaVersion != SchemaVersion || result.RunID == "" || result.TaskID == "" || result.Attempt < 1 {
		return fmt.Errorf("%w: identidade de resultado incompleta", ErrInvalidState)
	}
	if !s.validStatus(result.Status) || result.Status == StatusDraft || result.Status == StatusApproved || result.Status == StatusStale || result.Status == StatusExecuting {
		return fmt.Errorf("%w: status de resultado invalido", ErrInvalidState)
	}
	if result.Status != StatusDone {
		return nil
	}
	if len(result.BaseSHA) != sha256.Size*2 || len(result.PatchSHA256) != sha256.Size*2 || len(result.FinalStateSHA256) != sha256.Size*2 {
		return fmt.Errorf("%w: digests de patch incompletos", ErrInvalidState)
	}
	if result.CoverageRegression || len(result.Tests) == 0 || len(result.Criteria) == 0 || len(result.Evidence) == 0 {
		return fmt.Errorf("%w: provas de done incompletas", ErrInvalidState)
	}
	for _, proof := range result.Tests {
		if proof.Command == "" || proof.ExitCode != 0 || len(proof.OutputSHA256) != sha256.Size*2 {
			return fmt.Errorf("%w: teste sem prova de sucesso", ErrInvalidState)
		}
	}
	for _, criterion := range result.Criteria {
		if criterion.ID == "" || criterion.EvidenceRef == "" {
			return fmt.Errorf("%w: criterio sem evidencia individual", ErrInvalidState)
		}
	}
	if result.ReviewVerdict != "approved" {
		return fmt.Errorf("%w: revisao nao aprovada", ErrInvalidState)
	}
	return nil
}

func (s *Store) artifactPath(prdDir string, artifact Artifact) (string, error) {
	switch artifact {
	case ArtifactPRD:
		return filepath.Join(prdDir, "prd.md"), nil
	case ArtifactTechSpec:
		return filepath.Join(prdDir, "techspec.md"), nil
	case ArtifactTasks:
		return filepath.Join(prdDir, "tasks.md"), nil
	default:
		return "", fmt.Errorf("%w: artefato desconhecido %q", ErrInvalidState, artifact)
	}
}

func (s *Store) validateApproval(state State, prdDir string, artifact Artifact) error {
	entry, ok := state.Artifacts[artifact]
	if !ok {
		return fmt.Errorf("%w: artefato desconhecido %q", ErrInvalidState, artifact)
	}
	if entry.Status != StatusDraft && entry.Status != StatusStale {
		return fmt.Errorf("%w: transicao para aprovar %s a partir de %s nao permitida", ErrInvalidState, artifact, entry.Status)
	}
	for _, prerequisite := range s.prerequisites(artifact) {
		prerequisiteEntry := state.Artifacts[prerequisite]
		if !prerequisiteEntry.Approved || prerequisiteEntry.Status != StatusApproved {
			return fmt.Errorf("%w: %s requer %s aprovado", ErrInvalidState, artifact, prerequisite)
		}
		path, err := s.artifactPath(prdDir, prerequisite)
		if err != nil {
			return err
		}
		digest, err := s.digestFile(path)
		if err != nil {
			return err
		}
		if digest != prerequisiteEntry.SHA256 {
			return fmt.Errorf("%w: %s aprovado foi alterado; invalide descendentes antes de aprovar", ErrInvalidState, prerequisite)
		}
	}
	return nil
}

func (s *Store) digestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("ler artefato para aprovar: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// DigestFile calcula o digest canônico usado para vincular um artefato aprovado.
func (s *Store) DigestFile(path string) (string, error) {
	return s.digestFile(path)
}

func (s *Store) downstream(from Artifact) []Artifact {
	switch from {
	case ArtifactPRD:
		return []Artifact{ArtifactTechSpec, ArtifactTasks}
	case ArtifactTechSpec:
		return []Artifact{ArtifactTasks}
	default:
		return nil
	}
}

func (s *Store) prerequisites(artifact Artifact) []Artifact {
	switch artifact {
	case ArtifactTechSpec:
		return []Artifact{ArtifactPRD}
	case ArtifactTasks:
		return []Artifact{ArtifactPRD, ArtifactTechSpec}
	default:
		return nil
	}
}

func (s *Store) validStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusApproved, StatusStale, StatusExecuting, StatusBlocked, StatusNeedsInput, StatusFailed, StatusDone:
		return true
	default:
		return false
	}
}

func (s *Store) validDigest(digest string) bool {
	if len(digest) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}
