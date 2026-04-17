package state

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
	"github.com/jailtonjunior/orchestrator/internal/runtime/domain"
)

func TestFileStoreRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewFileStore(t.TempDir(), platform.NewFileSystem())
	run := mustRun(t, "run-1", "wf")
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	if err := run.Start(now); err != nil {
		t.Fatal(err)
	}
	if err := run.StartStep(mustStepName(t, "prd"), now); err != nil {
		t.Fatal(err)
	}
	if err := run.MarkStepCompleted(mustStepName(t, "prd"), domain.StepResult{
		Output:              "markdown",
		RawOutputRef:        filepath.Join("artifacts", "prd", "raw.md"),
		ApprovedMarkdownRef: filepath.Join("artifacts", "prd", "approved.md"),
		StructuredJSONRef:   filepath.Join("artifacts", "prd", "structured.json"),
		ValidationReportRef: filepath.Join("artifacts", "prd", "validation.json"),
		ValidationStatus:    domain.ValidationPassed,
	}, now); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveArtifact(context.Background(), run.ID(), "prd", Artifact{
		RawOutput:        []byte("raw"),
		ApprovedMarkdown: []byte("markdown"),
		StructuredJSON:   []byte(`{"ok":true}`),
		ValidationReport: []byte(`{"validation_status":"passed"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.LoadRun(context.Background(), run.ID())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID() != run.ID() {
		t.Fatalf("id = %q", loaded.ID())
	}
	if loaded.Steps()[0].Output() != "markdown" {
		t.Fatalf("output = %q", loaded.Steps()[0].Output())
	}
	if loaded.Steps()[0].Result().StructuredJSONRef == "" {
		t.Fatal("expected structured json ref")
	}
}

func TestFileStoreFindLatestPending(t *testing.T) {
	t.Parallel()

	store := NewFileStore(t.TempDir(), platform.NewFileSystem())
	older := mustRun(t, "run-older", "wf")
	newer := mustRun(t, "run-newer", "wf")

	if err := older.Start(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := older.Pause(time.Date(2026, 4, 3, 10, 5, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := newer.Start(time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := newer.Pause(time.Date(2026, 4, 3, 11, 5, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRun(context.Background(), older); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRun(context.Background(), newer); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.FindLatestPending(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID() != "run-newer" {
		t.Fatalf("loaded id = %q", loaded.ID())
	}
}

func TestFileStoreArtifacts(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := NewFileStore(baseDir, platform.NewFileSystem())
	if err := store.SaveArtifact(context.Background(), "run-1", "prd", Artifact{
		RawOutput:        []byte("raw"),
		ApprovedMarkdown: []byte("md"),
		StructuredJSON:   []byte(`{"ok":true}`),
		ValidationReport: []byte(`{"validation_status":"passed"}`),
	}); err != nil {
		t.Fatal(err)
	}

	artifact, err := store.LoadArtifact(context.Background(), "run-1", "prd")
	if err != nil {
		t.Fatal(err)
	}
	if string(artifact.ApprovedMarkdown) != "md" || string(artifact.StructuredJSON) != `{"ok":true}` {
		t.Fatalf("artifact = %+v", artifact)
	}

	if _, err := platform.NewFileSystem().Stat(filepath.Join(baseDir, ".orq", "runs", "run-1", "artifacts", "prd", "approved.md")); err != nil {
		t.Fatal(err)
	}
}

func TestFileStoreAppendLog(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store := NewFileStore(baseDir, platform.NewFileSystem())
	if err := store.AppendLog(context.Background(), "run-1", []byte(`{"event":"started"}`)); err != nil {
		t.Fatal(err)
	}

	data, err := platform.NewFileSystem().ReadFile(filepath.Join(baseDir, ".orq", "runs", "run-1", "logs", "run.log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{\"event\":\"started\"}\n" {
		t.Fatalf("log = %q", string(data))
	}
}

func mustRun(t *testing.T, runID string, workflow string) *domain.Run {
	t.Helper()
	wf, err := domain.NewWorkflowName(workflow)
	if err != nil {
		t.Fatal(err)
	}
	stepName, err := domain.NewStepName("prd")
	if err != nil {
		t.Fatal(err)
	}
	provider, err := domain.NewProviderName("claude")
	if err != nil {
		t.Fatal(err)
	}
	run, err := domain.NewRun(runID, wf, "input", []domain.StepDefinition{{
		Name:     stepName,
		Provider: provider,
		Input:    "input",
	}}, time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return run
}

// TestFileStoreSessionIDRoundTrip verifies that sessionID is persisted to
// state.json and restored correctly on LoadRun.
func TestFileStoreSessionIDRoundTrip(t *testing.T) {
	t.Parallel()

	store := NewFileStore(t.TempDir(), platform.NewFileSystem())
	run := mustRun(t, "run-sess", "wf")
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	if err := run.Start(now); err != nil {
		t.Fatal(err)
	}

	// Set sessionID on the step before completing it.
	prdName := mustStepName(t, "prd")
	if err := run.StartStep(prdName, now); err != nil {
		t.Fatal(err)
	}
	// Access the current step directly to set sessionID, as the engine does.
	currentStep, stepErr := run.CurrentStep()
	if stepErr != nil {
		t.Fatal(stepErr)
	}
	currentStep.SetSessionID("acp-session-test-42")

	if err := run.MarkStepCompleted(prdName, domain.StepResult{
		Output: "result",
	}, now); err != nil {
		t.Fatal(err)
	}

	if err := store.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.LoadRun(context.Background(), "run-sess")
	if err != nil {
		t.Fatal(err)
	}
	steps := loaded.Steps()
	if len(steps) == 0 {
		t.Fatal("expected at least one step")
	}
	if got := steps[0].SessionID(); got != "acp-session-test-42" {
		t.Errorf("sessionID = %q, want %q", got, "acp-session-test-42")
	}
}

func mustStepName(t *testing.T, name string) domain.StepName {
	t.Helper()
	stepName, err := domain.NewStepName(name)
	if err != nil {
		t.Fatal(err)
	}
	return stepName
}
