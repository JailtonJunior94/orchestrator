package domain

import (
	"errors"
	"testing"
	"time"
)

var testTime = time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)

func mustStepName(t *testing.T, name string) StepName {
	t.Helper()
	sn, err := NewStepName(name)
	if err != nil {
		t.Fatal(err)
	}
	return sn
}

func mustProviderName(t *testing.T, name string) ProviderName {
	t.Helper()
	pn, err := NewProviderName(name)
	if err != nil {
		t.Fatal(err)
	}
	return pn
}

func mustWorkflowName(t *testing.T, name string) WorkflowName {
	t.Helper()
	wn, err := NewWorkflowName(name)
	if err != nil {
		t.Fatal(err)
	}
	return wn
}

func defaultStepDefs(t *testing.T) []StepDefinition {
	t.Helper()
	return []StepDefinition{
		{Name: mustStepName(t, "prd"), Provider: mustProviderName(t, "claude"), Input: "generate prd"},
		{Name: mustStepName(t, "techspec"), Provider: mustProviderName(t, "claude"), Input: "generate techspec"},
	}
}

func newTestRun(t *testing.T) *Run {
	t.Helper()
	r, err := NewRun("run-1", mustWorkflowName(t, "dev"), "build api", defaultStepDefs(t), testTime)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNewRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		id       string
		workflow string
		input    string
		steps    []StepDefinition
		wantErr  error
	}{
		{
			name:     "valid",
			id:       "run-1",
			workflow: "dev",
			input:    "build api",
			steps: []StepDefinition{
				{Name: StepName{value: "prd"}, Provider: ProviderName{value: "claude"}, Input: "gen prd"},
			},
			wantErr: nil,
		},
		{
			name:     "empty id",
			id:       "",
			workflow: "dev",
			input:    "build",
			steps: []StepDefinition{
				{Name: StepName{value: "prd"}, Provider: ProviderName{value: "claude"}, Input: "gen"},
			},
			wantErr: ErrEmptyRunID,
		},
		{
			name:     "empty input",
			id:       "run-1",
			workflow: "dev",
			input:    "",
			steps: []StepDefinition{
				{Name: StepName{value: "prd"}, Provider: ProviderName{value: "claude"}, Input: "gen"},
			},
			wantErr: ErrEmptyInput,
		},
		{
			name:     "no steps",
			id:       "run-1",
			workflow: "dev",
			input:    "build",
			steps:    nil,
			wantErr:  ErrNoSteps,
		},
		{
			name:     "duplicate step name",
			id:       "run-1",
			workflow: "dev",
			input:    "build",
			steps: []StepDefinition{
				{Name: StepName{value: "prd"}, Provider: ProviderName{value: "claude"}, Input: "gen"},
				{Name: StepName{value: "prd"}, Provider: ProviderName{value: "claude"}, Input: "gen2"},
			},
			wantErr: ErrDuplicateStepName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wn, _ := NewWorkflowName(tt.workflow)
			r, err := NewRun(tt.id, wn, tt.input, tt.steps, testTime)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewRun error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if r.Status() != RunPending {
				t.Errorf("initial status = %q, want pending", r.Status())
			}
			if r.ID() != tt.id {
				t.Errorf("id = %q, want %q", r.ID(), tt.id)
			}
		})
	}
}

func TestRunStartAndPause(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	now := testTime.Add(time.Second)

	if err := r.Start(now); err != nil {
		t.Fatal(err)
	}
	if r.Status() != RunRunning {
		t.Errorf("status = %q, want running", r.Status())
	}

	// Double start should fail
	if err := r.Start(now); err == nil {
		t.Error("expected error on double start")
	}

	if err := r.Pause(now); err != nil {
		t.Fatal(err)
	}
	if r.Status() != RunPaused {
		t.Errorf("status = %q, want paused", r.Status())
	}

	if err := r.Resume(now); err != nil {
		t.Fatal(err)
	}
	if r.Status() != RunRunning {
		t.Errorf("status = %q, want running", r.Status())
	}
}

func TestRunCancel(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	now := testTime.Add(time.Second)

	if err := r.Cancel(now); err != nil {
		t.Fatal(err)
	}
	if r.Status() != RunCancelled {
		t.Errorf("status = %q, want cancelled", r.Status())
	}

	// Cannot cancel again (terminal)
	if err := r.Cancel(now); err == nil {
		t.Error("expected error on double cancel")
	}
}

func TestRunApproveStep(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	now := testTime.Add(time.Second)

	// Cannot approve when not running
	prd := mustStepName(t, "prd")
	if err := r.ApproveStep(prd, now); !errors.Is(err, ErrRunNotRunning) {
		t.Errorf("expected ErrRunNotRunning, got %v", err)
	}

	_ = r.Start(now)
	_ = r.StartStep(prd, now)
	_ = r.MarkStepCompleted(prd, StepResult{Output: "prd output"}, now)

	if err := r.ApproveStep(prd, now); err != nil {
		t.Fatal(err)
	}

	// Step not found
	unknown := mustStepName(t, "unknown")
	if err := r.ApproveStep(unknown, now); !errors.Is(err, ErrStepNotFound) {
		t.Errorf("expected ErrStepNotFound, got %v", err)
	}
}

func TestRunRetryStep(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	now := testTime.Add(time.Second)

	_ = r.Start(now)
	prd := mustStepName(t, "prd")
	_ = r.StartStep(prd, now)
	_ = r.MarkStepCompleted(prd, StepResult{Output: "output"}, now)

	if err := r.RetryStep(prd, now); err != nil {
		t.Fatal(err)
	}

	// Step should be retrying, can start again
	step, _ := r.findStep(prd)
	if step.Status() != StepRetrying {
		t.Errorf("status = %q, want retrying", step.Status())
	}
}

func TestRunMarkStepFailed(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	now := testTime.Add(time.Second)

	_ = r.Start(now)
	prd := mustStepName(t, "prd")
	_ = r.StartStep(prd, now)

	if err := r.MarkStepFailed(prd, "provider timeout", now); err != nil {
		t.Fatal(err)
	}

	if r.Status() != RunFailed {
		t.Errorf("run status = %q, want failed", r.Status())
	}
}

func TestRunCurrentStep(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	now := testTime.Add(time.Second)

	current, err := r.CurrentStep()
	if err != nil {
		t.Fatal(err)
	}
	if current.Name().String() != "prd" {
		t.Errorf("current step = %q, want prd", current.Name().String())
	}

	// Approve first step, current should be second
	_ = r.Start(now)
	_ = r.StartStep(mustStepName(t, "prd"), now)
	_ = r.MarkStepCompleted(mustStepName(t, "prd"), StepResult{Output: "done"}, now)
	_ = r.ApproveStep(mustStepName(t, "prd"), now)

	current, err = r.CurrentStep()
	if err != nil {
		t.Fatal(err)
	}
	if current.Name().String() != "techspec" {
		t.Errorf("current step = %q, want techspec", current.Name().String())
	}
}

func TestRunCompletesWhenAllStepsApproved(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	now := testTime.Add(time.Second)

	_ = r.Start(now)

	for _, name := range []string{"prd", "techspec"} {
		sn := mustStepName(t, name)
		_ = r.StartStep(sn, now)
		_ = r.MarkStepCompleted(sn, StepResult{Output: name + " output"}, now)
		_ = r.ApproveStep(sn, now)
	}

	if r.Status() != RunCompleted {
		t.Errorf("status = %q, want completed", r.Status())
	}
}

func TestRunStepsReturnsCopy(t *testing.T) {
	t.Parallel()
	r := newTestRun(t)
	steps := r.Steps()
	steps[0] = nil
	if r.Steps()[0] == nil {
		t.Error("Steps() should return a copy")
	}
}
