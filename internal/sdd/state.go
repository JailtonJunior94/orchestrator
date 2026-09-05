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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd/tasks"

	"github.com/JailtonJunior94/ai-spec-harness/internal/specdigest"
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

var _migrationRunIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

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

type RequirementState struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type DAGNode struct {
	TaskID       string   `json:"task_id"`
	Dependencies []string `json:"dependencies"`
}

type AttemptSnapshot struct {
	BaseSHA          string `json:"base_sha"`
	PatchSHA256      string `json:"patch_sha256"`
	FinalStateSHA256 string `json:"final_state_sha256"`
	Dirty            bool   `json:"dirty"`
}

type TaskState struct {
	ID           string           `json:"id"`
	Status       Status           `json:"status"`
	Requirements []string         `json:"requirements"`
	Dependencies []string         `json:"dependencies"`
	Ownership    []string         `json:"ownership"`
	Parallel     bool             `json:"parallel"`
	Attempt      int              `json:"attempt"`
	Snapshot     *AttemptSnapshot `json:"snapshot,omitempty"`
	EvidenceRefs []string         `json:"evidence_refs"`
}

type MigrationState struct {
	RunID     string `json:"run_id"`
	BackupRef string `json:"backup_ref,omitempty"`
}

type State struct {
	SchemaVersion int                        `json:"schema_version"`
	RunID         string                     `json:"run_id"`
	Artifacts     map[Artifact]ArtifactState `json:"artifacts"`
	Requirements  []RequirementState         `json:"requirements"`
	DAG           []DAGNode                  `json:"dag"`
	Tasks         map[string]TaskState       `json:"tasks"`
	EvidenceRefs  []string                   `json:"evidence_refs"`
	Migration     *MigrationState            `json:"migration,omitempty"`
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
	PatchRef           string           `json:"patch_ref,omitempty"`
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
		Requirements: make([]RequirementState, 0),
		DAG:          make([]DAGNode, 0),
		Tasks:        make(map[string]TaskState),
		EvidenceRefs: make([]string, 0),
	}
	state.Events = append(state.Events, Event{At: s.now().UTC(), RunID: runID, Action: "initialized"})
	if err := s.Save(prdDir, state); err != nil {
		return State{}, err
	}
	return state, nil
}

// Migrate constrói o estado v2 a partir dos artefatos atuais. Dry-run não escreve.
func (s *Store) Migrate(prdDir, runID string, dryRun bool) (State, error) {
	if !_migrationRunIDPattern.MatchString(runID) {
		return State{}, fmt.Errorf("%w: run_id de migracao invalido", ErrInvalidState)
	}
	path := s.StatePath(prdDir)
	previous, readErr := os.ReadFile(path)
	if readErr != nil && !os.IsNotExist(readErr) {
		return State{}, fmt.Errorf("ler estado legado: %w", readErr)
	}
	backupRef := ""
	if readErr == nil {
		backupRef = ".sdd-state." + runID + ".legacy.json"
	}
	state := State{
		SchemaVersion: SchemaVersion,
		RunID:         runID,
		Artifacts:     make(map[Artifact]ArtifactState, 3),
		Requirements:  make([]RequirementState, 0),
		DAG:           make([]DAGNode, 0),
		Tasks:         make(map[string]TaskState),
		EvidenceRefs:  make([]string, 0),
		Migration:     &MigrationState{RunID: runID, BackupRef: backupRef},
	}
	for _, artifact := range []Artifact{ArtifactPRD, ArtifactTechSpec, ArtifactTasks} {
		artifactPath, err := s.artifactPath(prdDir, artifact)
		if err != nil {
			return State{}, err
		}
		digest, err := s.digestFile(artifactPath)
		if err != nil {
			return State{}, err
		}
		state.Artifacts[artifact] = ArtifactState{Status: StatusApproved, SHA256: digest, Approved: true}
	}
	if err := s.populateOperationalModel(prdDir, &state); err != nil {
		return State{}, err
	}
	state.Events = append(state.Events, Event{At: s.now().UTC(), RunID: runID, Action: "migrated"})
	if err := s.Validate(state); err != nil {
		return State{}, err
	}
	if dryRun {
		return state, nil
	}
	if len(previous) > 0 {
		if err := s.writeExclusive(filepath.Join(prdDir, backupRef), previous); err != nil {
			return State{}, err
		}
	}
	if err := s.Save(prdDir, state); err != nil {
		return State{}, err
	}
	return state, nil
}

// RollbackMigration remove somente o estado criado pelo run_id informado.
func (s *Store) RollbackMigration(prdDir, runID string) error {
	state, err := s.Load(prdDir)
	if err != nil {
		return err
	}
	if state.Migration == nil || state.Migration.RunID != runID {
		return fmt.Errorf("%w: run_id nao criou o estado v2 atual", ErrInvalidState)
	}
	if err := os.Remove(s.StatePath(prdDir)); err != nil {
		return fmt.Errorf("remover estado v2 migrado: %w", err)
	}
	if state.Migration.BackupRef != "" {
		backup := filepath.Join(prdDir, state.Migration.BackupRef)
		if err := os.Rename(backup, s.StatePath(prdDir)); err != nil {
			return fmt.Errorf("restaurar estado anterior: %w", err)
		}
	}
	return nil
}

func (s *Store) writeExclusive(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("criar backup de migracao: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("escrever backup de migracao: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sincronizar backup de migracao: %w", err)
	}
	return nil
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
	if artifact == ArtifactTasks {
		if err := s.populateOperationalModel(prdDir, &state); err != nil {
			return State{}, err
		}
	}
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
	if state.Artifacts[ArtifactTasks].Approved {
		if state.Tasks == nil || state.Requirements == nil || state.DAG == nil || state.EvidenceRefs == nil ||
			len(state.Requirements) == 0 || len(state.Tasks) == 0 || len(state.DAG) != len(state.Tasks) {
			return fmt.Errorf("%w: modelo operacional aprovado incompleto", ErrInvalidState)
		}
	}
	for id, task := range state.Tasks {
		if id == "" || task.ID != id || !s.validStatus(task.Status) {
			return fmt.Errorf("%w: task operacional %q invalida", ErrInvalidState, id)
		}
		if task.Status == StatusExecuting && (task.Attempt < 1 || task.Snapshot == nil) {
			return fmt.Errorf("%w: task %s executando sem tentativa persistida", ErrInvalidState, id)
		}
		if task.Status == StatusDone && len(task.EvidenceRefs) == 0 {
			return fmt.Errorf("%w: task %s done sem evidencia persistida", ErrInvalidState, id)
		}
	}
	return nil
}

// ValidateDirectory confronta estado, artefatos atuais e o modelo estrutural.
func (s *Store) ValidateDirectory(prdDir string) (State, error) {
	state, err := s.Load(prdDir)
	if err != nil {
		return State{}, err
	}
	for _, artifact := range []Artifact{ArtifactPRD, ArtifactTechSpec, ArtifactTasks} {
		entry := state.Artifacts[artifact]
		if !entry.Approved {
			continue
		}
		path, pathErr := s.artifactPath(prdDir, artifact)
		if pathErr != nil {
			return State{}, pathErr
		}
		digest, digestErr := s.digestFile(path)
		if digestErr != nil {
			return State{}, digestErr
		}
		if digest != entry.SHA256 {
			return State{}, fmt.Errorf("%w: artefato %s aprovado esta stale", ErrInvalidState, artifact)
		}
	}
	expected := State{Tasks: make(map[string]TaskState)}
	if err := s.populateOperationalModel(prdDir, &expected); err != nil {
		return State{}, err
	}
	if err := s.compareOperationalModel(state, expected); err != nil {
		return State{}, err
	}
	evidenceRoot, err := s.evidenceRoot(prdDir)
	if err != nil {
		return State{}, err
	}
	for _, reference := range state.EvidenceRefs {
		if _, err := s.resolveContainedFile(evidenceRoot, reference); err != nil {
			return State{}, fmt.Errorf("%w: evidencia operacional invalida: %v", ErrInvalidState, err)
		}
	}
	return state, nil
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
	if !s.validBaseSHA(result.BaseSHA) || !s.validDigest(result.PatchSHA256) || !s.validDigest(result.FinalStateSHA256) {
		return fmt.Errorf("%w: digests de patch incompletos", ErrInvalidState)
	}
	if result.CoverageRegression || len(result.Tests) == 0 || len(result.Criteria) == 0 || len(result.Evidence) == 0 {
		return fmt.Errorf("%w: provas de done incompletas", ErrInvalidState)
	}
	if strings.TrimSpace(result.PatchRef) == "" {
		return fmt.Errorf("%w: artefato fisico do patch ausente", ErrInvalidState)
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

func (s *Store) populateOperationalModel(prdDir string, state *State) error {
	prdContent, err := os.ReadFile(filepath.Join(prdDir, "prd.md"))
	if err != nil {
		return fmt.Errorf("ler PRD para modelo operacional: %w", err)
	}
	tasksContent, err := os.ReadFile(filepath.Join(prdDir, "tasks.md"))
	if err != nil {
		return fmt.Errorf("ler tasks para modelo operacional: %w", err)
	}
	document, err := tasks.NewParser().ParseAt(prdDir, prdContent, tasksContent)
	if err != nil {
		return fmt.Errorf("construir modelo operacional: %w", err)
	}

	state.Requirements = make([]RequirementState, 0, len(document.Requirements))
	for _, id := range document.Requirements {
		kind := "functional"
		if strings.HasPrefix(id, "NFR-") {
			kind = "non_functional"
		}
		state.Requirements = append(state.Requirements, RequirementState{ID: id, Kind: kind})
	}
	state.DAG = make([]DAGNode, 0, len(document.Tasks))
	state.Tasks = make(map[string]TaskState, len(document.Tasks))
	state.EvidenceRefs = make([]string, 0)
	for _, parsedTask := range document.Tasks {
		status := s.importedTaskStatus(parsedTask.Status)
		evidence := make([]string, 0, 1)
		if status == StatusDone {
			checkpointStatus, checkpointEvidence, checkpointErr := s.importTaskCheckpoint(prdDir, parsedTask.ID)
			if checkpointErr != nil {
				return fmt.Errorf("importar task %s: %w", parsedTask.ID, checkpointErr)
			}
			status = checkpointStatus
			evidence = checkpointEvidence
			state.EvidenceRefs = append(state.EvidenceRefs, evidence...)
		}
		state.DAG = append(state.DAG, DAGNode{TaskID: parsedTask.ID, Dependencies: parsedTask.Dependencies})
		state.Tasks[parsedTask.ID] = TaskState{
			ID:           parsedTask.ID,
			Status:       status,
			Requirements: document.Coverage[parsedTask.ID],
			Dependencies: parsedTask.Dependencies,
			Ownership:    parsedTask.Ownership,
			Parallel:     parsedTask.Parallel,
			EvidenceRefs: evidence,
		}
	}
	sort.Slice(state.Requirements, func(left, right int) bool {
		return state.Requirements[left].ID < state.Requirements[right].ID
	})
	sort.Slice(state.DAG, func(left, right int) bool { return state.DAG[left].TaskID < state.DAG[right].TaskID })
	sort.Strings(state.EvidenceRefs)
	return nil
}

func (s *Store) importTaskCheckpoint(prdDir, taskID string) (Status, []string, error) {
	content, err := os.ReadFile(filepath.Join(prdDir, ".checkpoints", taskID+".json"))
	if err != nil {
		return StatusDraft, nil, fmt.Errorf("checkpoint v2 ausente: %w", err)
	}
	result, err := NewResultValidator().ValidateCheckpointJSON(content)
	if err != nil {
		return StatusDraft, nil, err
	}
	if result.TaskID != taskID {
		return StatusDraft, nil, fmt.Errorf("checkpoint pertence a task %s", result.TaskID)
	}
	if result.Status != StatusDone {
		return result.Status, append([]string(nil), result.Evidence...), nil
	}
	if !strings.HasPrefix(filepath.ToSlash(result.PatchRef), "evidence/") {
		return StatusDraft, nil, fmt.Errorf("artefato do patch deve estar em evidence/: %q", result.PatchRef)
	}
	root, err := s.evidenceRoot(prdDir)
	if err != nil {
		return StatusDraft, nil, err
	}
	digests := make(map[string]struct{}, len(result.Evidence))
	for _, reference := range result.Evidence {
		path, resolveErr := s.resolveContainedFile(root, reference)
		if resolveErr != nil {
			return StatusDraft, nil, resolveErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return StatusDraft, nil, readErr
		}
		digest := sha256.Sum256(data)
		digests[hex.EncodeToString(digest[:])] = struct{}{}
	}
	evidenceRefs := make(map[string]struct{}, len(result.Evidence))
	for _, reference := range result.Evidence {
		evidenceRefs[filepath.ToSlash(strings.Split(reference, "#")[0])] = struct{}{}
	}
	for _, criterion := range result.Criteria {
		reference := filepath.ToSlash(strings.Split(criterion.EvidenceRef, "#")[0])
		if _, exists := evidenceRefs[reference]; !exists {
			return StatusDraft, nil, fmt.Errorf("criterio %s nao referencia evidencia declarada", criterion.ID)
		}
	}
	for _, proof := range result.Tests {
		if _, exists := digests[proof.OutputSHA256]; !exists {
			return StatusDraft, nil, fmt.Errorf("digest do teste sem evidencia fisica")
		}
	}
	patchPath, err := s.resolveContainedFile(root, result.PatchRef)
	if err != nil {
		return StatusDraft, nil, fmt.Errorf("artefato do patch invalido: %w", err)
	}
	patch, err := os.ReadFile(patchPath)
	if err != nil {
		return StatusDraft, nil, err
	}
	patchDigest := sha256.Sum256(patch)
	if hex.EncodeToString(patchDigest[:]) != result.PatchSHA256 {
		return StatusDraft, nil, fmt.Errorf("digest do patch diverge do artefato fisico")
	}
	return StatusDone, append([]string(nil), result.Evidence...), nil
}

func (s *Store) compareOperationalModel(actual, expected State) error {
	if len(actual.Requirements) != len(expected.Requirements) || len(actual.DAG) != len(expected.DAG) || len(actual.Tasks) != len(expected.Tasks) {
		return fmt.Errorf("%w: modelo operacional diverge dos artefatos atuais", ErrInvalidState)
	}
	for index := range expected.Requirements {
		if actual.Requirements[index] != expected.Requirements[index] {
			return fmt.Errorf("%w: requisitos persistidos divergem do PRD", ErrInvalidState)
		}
	}
	for index := range expected.DAG {
		left, right := actual.DAG[index], expected.DAG[index]
		if left.TaskID != right.TaskID || strings.Join(left.Dependencies, "\x00") != strings.Join(right.Dependencies, "\x00") {
			return fmt.Errorf("%w: DAG persistido diverge de tasks.md", ErrInvalidState)
		}
	}
	for id, expectedTask := range expected.Tasks {
		actualTask, exists := actual.Tasks[id]
		if !exists || strings.Join(actualTask.Requirements, "\x00") != strings.Join(expectedTask.Requirements, "\x00") ||
			strings.Join(actualTask.Dependencies, "\x00") != strings.Join(expectedTask.Dependencies, "\x00") ||
			strings.Join(actualTask.Ownership, "\x00") != strings.Join(expectedTask.Ownership, "\x00") || actualTask.Parallel != expectedTask.Parallel {
			return fmt.Errorf("%w: task %s persistida diverge de tasks.md", ErrInvalidState, id)
		}
	}
	return nil
}

func (s *Store) importedTaskStatus(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "done":
		return StatusDone
	case "blocked":
		return StatusBlocked
	case "failed":
		return StatusFailed
	case "needs_input":
		return StatusNeedsInput
	case "executing", "in_progress":
		return StatusExecuting
	default:
		return StatusDraft
	}
}

func (s *Store) resolveContainedFile(root, reference string) (string, error) {
	path, _, _ := strings.Cut(reference, "#")
	if path == "" || filepath.IsAbs(path) {
		return "", fmt.Errorf("referencia deve ser relativa: %q", reference)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolver raiz de evidencia: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, path))
	if err != nil {
		return "", fmt.Errorf("resolver evidencia %q: %w", reference, err)
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("evidencia escapa da raiz: %q", reference)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("evidencia nao e arquivo regular: %q", reference)
	}
	return resolvedPath, nil
}

func (s *Store) evidenceRoot(prdDir string) (string, error) {
	current, err := filepath.Abs(prdDir)
	if err != nil {
		return "", fmt.Errorf("resolver diretorio do PRD: %w", err)
	}
	for {
		if info, statErr := os.Stat(filepath.Join(current, ".git")); statErr == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Abs(prdDir)
		}
		current = parent
	}
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
	return specdigest.Canonical(data), nil
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

func (s *Store) validBaseSHA(digest string) bool {
	if len(digest) != 40 && len(digest) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}
