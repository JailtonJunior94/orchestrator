package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
	"github.com/jailtonjunior/orchestrator/internal/runtime/domain"
)

const schemaVersion = 1

var (
	// ErrRunNotFound indicates persisted state for the requested run does not exist.
	ErrRunNotFound = errors.New("run not found")
	// ErrNoPendingRuns indicates continue cannot resume any persisted run.
	ErrNoPendingRuns = errors.New("no pending runs found")
)

// Artifact stores the persisted outputs of a step.
type Artifact struct {
	RawOutput        []byte
	ApprovedMarkdown []byte
	StructuredJSON   []byte
	ValidationReport []byte
}

// Store persists workflow runs and their artifacts.
type Store interface {
	SaveRun(ctx context.Context, run *domain.Run) error
	LoadRun(ctx context.Context, runID string) (*domain.Run, error)
	FindLatestPending(ctx context.Context) (*domain.Run, error)
	SaveArtifact(ctx context.Context, runID string, stepName string, artifact Artifact) error
	LoadArtifact(ctx context.Context, runID string, stepName string) (*Artifact, error)
	AppendLog(ctx context.Context, runID string, entry []byte) error
}

// FileStore persists runs under .orq/runs/<run-id>.
type FileStore struct {
	baseDir string
	fs      platform.FileSystem
}

// NewFileStore creates the filesystem-backed state store.
func NewFileStore(baseDir string, fileSystem platform.FileSystem) *FileStore {
	return &FileStore{
		baseDir: baseDir,
		fs:      fileSystem,
	}
}

type runDTO struct {
	RunID         string    `json:"run_id"`
	Workflow      string    `json:"workflow"`
	Input         string    `json:"input"`
	Status        string    `json:"status"`
	CreatedAt     string    `json:"created_at"`
	UpdatedAt     string    `json:"updated_at"`
	CurrentStep   string    `json:"current_step,omitempty"`
	SchemaVersion int       `json:"schema_version"`
	Steps         []stepDTO `json:"steps"`
}

type stepDTO struct {
	Name     string        `json:"name"`
	Provider string        `json:"provider"`
	Status   string        `json:"status"`
	Input    string        `json:"input,omitempty"`
	Attempts int           `json:"attempts"`
	Result   stepResultDTO `json:"result,omitempty"`
	Error    string        `json:"error,omitempty"`
}

type stepResultDTO struct {
	RawOutputRef        string `json:"raw_output_ref,omitempty"`
	ApprovedMarkdownRef string `json:"approved_markdown_ref,omitempty"`
	StructuredJSONRef   string `json:"structured_json_ref,omitempty"`
	ValidationReportRef string `json:"validation_report_ref,omitempty"`
	SchemaName          string `json:"schema_name,omitempty"`
	SchemaVersion       string `json:"schema_version,omitempty"`
	ValidationStatus    string `json:"validation_status,omitempty"`
	EditedByHuman       bool   `json:"edited_by_human,omitempty"`
}

// SaveRun persists a state.json snapshot and required directories.
func (s *FileStore) SaveRun(_ context.Context, run *domain.Run) error {
	snapshot := run.Snapshot()
	runDir := s.runDir(snapshot.ID)
	if err := s.fs.MkdirAll(filepath.Join(runDir, "logs"), 0o755); err != nil {
		return fmt.Errorf("creating run directory: %w", err)
	}

	dto, err := toRunDTO(snapshot)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(dto, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state json: %w", err)
	}

	if err := s.fs.WriteFile(filepath.Join(runDir, "state.json"), data, 0o644); err != nil {
		return fmt.Errorf("writing state json: %w", err)
	}

	return nil
}

// LoadRun reconstructs a run aggregate from state.json.
func (s *FileStore) LoadRun(_ context.Context, runID string) (*domain.Run, error) {
	data, err := s.fs.ReadFile(filepath.Join(s.runDir(runID), "state.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrRunNotFound
		}
		return nil, fmt.Errorf("reading state json: %w", err)
	}

	var dto runDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("decoding state json: %w", err)
	}

	snapshot, err := dto.toSnapshot(func(ref string) (string, error) {
		if ref == "" {
			return "", nil
		}

		data, readErr := s.fs.ReadFile(filepath.Join(s.runDir(runID), ref))
		if readErr != nil {
			return "", readErr
		}
		return string(data), nil
	})
	if err != nil {
		return nil, err
	}

	return domain.NewRunFromSnapshot(snapshot)
}

// FindLatestPending returns the most recently updated pending or paused run.
func (s *FileStore) FindLatestPending(ctx context.Context) (*domain.Run, error) {
	entries, err := s.fs.ReadDir(s.runsDir())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoPendingRuns
		}
		return nil, fmt.Errorf("reading runs directory: %w", err)
	}

	type candidate struct {
		run *domain.Run
	}

	candidates := make([]candidate, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		run, err := s.LoadRun(ctx, entry.Name())
		if err != nil {
			return nil, err
		}

		status := run.Status()
		if status == domain.RunPaused || status == domain.RunPending {
			candidates = append(candidates, candidate{run: run})
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoPendingRuns
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].run.UpdatedAt().After(candidates[j].run.UpdatedAt())
	})

	return candidates[0].run, nil
}

// SaveArtifact persists step markdown and optional json payloads.
func (s *FileStore) SaveArtifact(_ context.Context, runID string, stepName string, artifact Artifact) error {
	stepDir := filepath.Join(s.runDir(runID), "artifacts", stepName)
	if err := s.fs.MkdirAll(stepDir, 0o755); err != nil {
		return fmt.Errorf("creating artifact directory: %w", err)
	}

	if len(artifact.RawOutput) > 0 {
		if err := s.fs.WriteFile(filepath.Join(stepDir, "raw.md"), artifact.RawOutput, 0o644); err != nil {
			return fmt.Errorf("writing raw artifact: %w", err)
		}
	}

	if len(artifact.ApprovedMarkdown) > 0 {
		if err := s.fs.WriteFile(filepath.Join(stepDir, "approved.md"), artifact.ApprovedMarkdown, 0o644); err != nil {
			return fmt.Errorf("writing markdown artifact: %w", err)
		}
	}

	if len(artifact.StructuredJSON) > 0 {
		if err := s.fs.WriteFile(filepath.Join(stepDir, "structured.json"), artifact.StructuredJSON, 0o644); err != nil {
			return fmt.Errorf("writing json artifact: %w", err)
		}
	}

	if len(artifact.ValidationReport) > 0 {
		if err := s.fs.WriteFile(filepath.Join(stepDir, "validation.json"), artifact.ValidationReport, 0o644); err != nil {
			return fmt.Errorf("writing validation artifact: %w", err)
		}
	}

	return nil
}

// LoadArtifact loads persisted step artifacts.
func (s *FileStore) LoadArtifact(_ context.Context, runID string, stepName string) (*Artifact, error) {
	stepDir := filepath.Join(s.runDir(runID), "artifacts", stepName)
	artifact := &Artifact{}

	rawOutput, err := s.fs.ReadFile(filepath.Join(stepDir, "raw.md"))
	if err == nil {
		artifact.RawOutput = rawOutput
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading raw artifact: %w", err)
	}

	markdown, err := s.fs.ReadFile(filepath.Join(stepDir, "approved.md"))
	if err == nil {
		artifact.ApprovedMarkdown = markdown
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading markdown artifact: %w", err)
	}

	jsonData, err := s.fs.ReadFile(filepath.Join(stepDir, "structured.json"))
	if err == nil {
		artifact.StructuredJSON = jsonData
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading json artifact: %w", err)
	}

	validation, err := s.fs.ReadFile(filepath.Join(stepDir, "validation.json"))
	if err == nil {
		artifact.ValidationReport = validation
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("reading validation artifact: %w", err)
	}

	return artifact, nil
}

// AppendLog appends a JSON line to logs/run.log for the run.
func (s *FileStore) AppendLog(_ context.Context, runID string, entry []byte) error {
	logDir := filepath.Join(s.runDir(runID), "logs")
	if err := s.fs.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}

	logPath := filepath.Join(logDir, "run.log")
	existing, err := s.fs.ReadFile(logPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("reading run log: %w", err)
	}

	data := append(existing, entry...)
	data = append(data, '\n')
	if err := s.fs.WriteFile(logPath, data, 0o644); err != nil {
		return fmt.Errorf("writing run log: %w", err)
	}

	return nil
}

func (s *FileStore) runsDir() string {
	return filepath.Join(s.baseDir, ".orq", "runs")
}

func (s *FileStore) runDir(runID string) string {
	return filepath.Join(s.runsDir(), runID)
}

func toRunDTO(snapshot domain.RunSnapshot) (runDTO, error) {
	steps := make([]stepDTO, 0, len(snapshot.Steps))
	for _, step := range snapshot.Steps {
		result := stepResultDTO{
			RawOutputRef:        step.Result.RawOutputRef,
			ApprovedMarkdownRef: step.Result.ApprovedMarkdownRef,
			StructuredJSONRef:   step.Result.StructuredJSONRef,
			ValidationReportRef: step.Result.ValidationReportRef,
			SchemaName:          step.Result.SchemaName,
			SchemaVersion:       step.Result.SchemaVersion,
			ValidationStatus:    string(step.Result.ValidationStatus),
			EditedByHuman:       step.Result.EditedByHuman,
		}
		dto := stepDTO{
			Name:     step.Name,
			Provider: step.Provider,
			Status:   string(step.Status),
			Input:    step.Input,
			Attempts: step.Attempts,
			Result:   result,
			Error:    step.Error,
		}
		steps = append(steps, dto)
	}

	currentStep := ""
	for _, step := range snapshot.Steps {
		switch step.Status {
		case domain.StepApproved, domain.StepFailed, domain.StepSkipped:
			continue
		default:
			currentStep = step.Name
			goto done
		}
	}

done:

	return runDTO{
		RunID:         snapshot.ID,
		Workflow:      snapshot.Workflow,
		Input:         snapshot.Input,
		Status:        string(snapshot.Status),
		CreatedAt:     snapshot.CreatedAt.Format(timeLayout),
		UpdatedAt:     snapshot.UpdatedAt.Format(timeLayout),
		CurrentStep:   currentStep,
		SchemaVersion: schemaVersion,
		Steps:         steps,
	}, nil
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

func (dto runDTO) toSnapshot(loadOutput func(ref string) (string, error)) (domain.RunSnapshot, error) {
	createdAt, err := timeFromString(dto.CreatedAt)
	if err != nil {
		return domain.RunSnapshot{}, err
	}

	updatedAt, err := timeFromString(dto.UpdatedAt)
	if err != nil {
		return domain.RunSnapshot{}, err
	}

	steps := make([]domain.StepSnapshot, 0, len(dto.Steps))
	for _, step := range dto.Steps {
		stepStatus := domain.StepStatus(step.Status)
		output := ""
		if loadOutput != nil && step.Result.ApprovedMarkdownRef != "" {
			var loadErr error
			output, loadErr = loadOutput(step.Result.ApprovedMarkdownRef)
			if loadErr != nil {
				return domain.RunSnapshot{}, fmt.Errorf("loading artifact %q: %w", step.Result.ApprovedMarkdownRef, loadErr)
			}
		}
		steps = append(steps, domain.StepSnapshot{
			Name:     step.Name,
			Provider: step.Provider,
			Status:   stepStatus,
			Input:    step.Input,
			Result: domain.StepResult{
				Output:              output,
				RawOutputRef:        step.Result.RawOutputRef,
				ApprovedMarkdownRef: step.Result.ApprovedMarkdownRef,
				StructuredJSONRef:   step.Result.StructuredJSONRef,
				ValidationReportRef: step.Result.ValidationReportRef,
				SchemaName:          step.Result.SchemaName,
				SchemaVersion:       step.Result.SchemaVersion,
				ValidationStatus:    domain.ValidationStatus(step.Result.ValidationStatus),
				EditedByHuman:       step.Result.EditedByHuman,
			},
			Attempts: step.Attempts,
			Error:    step.Error,
		})
	}

	return domain.RunSnapshot{
		ID:        dto.RunID,
		Workflow:  dto.Workflow,
		Input:     dto.Input,
		Status:    domain.RunStatus(dto.Status),
		Steps:     steps,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}, nil
}

func timeFromString(value string) (time.Time, error) {
	parsed, err := time.Parse(timeLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing time %q: %w", value, err)
	}
	return parsed, nil
}
