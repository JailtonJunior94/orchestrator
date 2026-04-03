package domain

import (
	"errors"
	"testing"
)

func TestValidateRunStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  RunStatus
		wantErr bool
	}{
		{"valid pending", RunPending, false},
		{"valid running", RunRunning, false},
		{"valid paused", RunPaused, false},
		{"valid failed", RunFailed, false},
		{"valid completed", RunCompleted, false},
		{"valid cancelled", RunCancelled, false},
		{"invalid empty", RunStatus(""), true},
		{"invalid unknown", RunStatus("unknown"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateRunStatus(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRunStatus(%q) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrInvalidRunStatus) {
				t.Errorf("expected ErrInvalidRunStatus, got %v", err)
			}
		})
	}
}

func TestRunStatusTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		from    RunStatus
		to      RunStatus
		wantOk  bool
	}{
		// pending transitions
		{"pending -> running", RunPending, RunRunning, true},
		{"pending -> cancelled", RunPending, RunCancelled, true},
		{"pending -> paused", RunPending, RunPaused, false},
		{"pending -> failed", RunPending, RunFailed, false},
		{"pending -> completed", RunPending, RunCompleted, false},
		// running transitions
		{"running -> paused", RunRunning, RunPaused, true},
		{"running -> failed", RunRunning, RunFailed, true},
		{"running -> completed", RunRunning, RunCompleted, true},
		{"running -> cancelled", RunRunning, RunCancelled, true},
		{"running -> pending", RunRunning, RunPending, false},
		// paused transitions
		{"paused -> running", RunPaused, RunRunning, true},
		{"paused -> cancelled", RunPaused, RunCancelled, true},
		{"paused -> failed", RunPaused, RunFailed, false},
		{"paused -> completed", RunPaused, RunCompleted, false},
		// terminal states
		{"failed -> running", RunFailed, RunRunning, false},
		{"completed -> running", RunCompleted, RunRunning, false},
		{"cancelled -> running", RunCancelled, RunRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.from.CanTransitionTo(tt.to)
			if got != tt.wantOk {
				t.Errorf("CanTransitionTo(%q -> %q) = %v, want %v", tt.from, tt.to, got, tt.wantOk)
			}

			result, err := tt.from.TransitionTo(tt.to)
			if tt.wantOk {
				if err != nil {
					t.Errorf("TransitionTo(%q -> %q) unexpected error: %v", tt.from, tt.to, err)
				}
				if result != tt.to {
					t.Errorf("TransitionTo result = %q, want %q", result, tt.to)
				}
			} else {
				if err == nil {
					t.Errorf("TransitionTo(%q -> %q) expected error", tt.from, tt.to)
				}
				if !errors.Is(err, ErrInvalidTransition) {
					t.Errorf("expected ErrInvalidTransition, got %v", err)
				}
			}
		})
	}
}

func TestValidateStepStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  StepStatus
		wantErr bool
	}{
		{"valid pending", StepPending, false},
		{"valid running", StepRunning, false},
		{"valid waiting_approval", StepWaitingApproval, false},
		{"valid approved", StepApproved, false},
		{"valid retrying", StepRetrying, false},
		{"valid failed", StepFailed, false},
		{"valid skipped", StepSkipped, false},
		{"invalid empty", StepStatus(""), true},
		{"invalid unknown", StepStatus("bogus"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateStepStatus(tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStepStatus(%q) error = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
		})
	}
}

func TestStepStatusTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		from   StepStatus
		to     StepStatus
		wantOk bool
	}{
		// pending
		{"pending -> running", StepPending, StepRunning, true},
		{"pending -> skipped", StepPending, StepSkipped, true},
		{"pending -> failed", StepPending, StepFailed, false},
		// running
		{"running -> waiting_approval", StepRunning, StepWaitingApproval, true},
		{"running -> failed", StepRunning, StepFailed, true},
		{"running -> approved", StepRunning, StepApproved, false},
		// waiting_approval
		{"waiting -> approved", StepWaitingApproval, StepApproved, true},
		{"waiting -> retrying", StepWaitingApproval, StepRetrying, true},
		{"waiting -> failed", StepWaitingApproval, StepFailed, true},
		{"waiting -> running", StepWaitingApproval, StepRunning, false},
		// retrying
		{"retrying -> running", StepRetrying, StepRunning, true},
		{"retrying -> failed", StepRetrying, StepFailed, false},
		// terminal
		{"approved -> running", StepApproved, StepRunning, false},
		{"failed -> running", StepFailed, StepRunning, false},
		{"skipped -> running", StepSkipped, StepRunning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.from.CanTransitionTo(tt.to)
			if got != tt.wantOk {
				t.Errorf("CanTransitionTo(%q -> %q) = %v, want %v", tt.from, tt.to, got, tt.wantOk)
			}

			_, err := tt.from.TransitionTo(tt.to)
			if tt.wantOk && err != nil {
				t.Errorf("TransitionTo unexpected error: %v", err)
			}
			if !tt.wantOk && err == nil {
				t.Errorf("TransitionTo expected error")
			}
		})
	}
}

func TestWorkflowName(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		wn, err := NewWorkflowName("dev-workflow")
		if err != nil {
			t.Fatal(err)
		}
		if wn.String() != "dev-workflow" {
			t.Errorf("got %q", wn.String())
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		_, err := NewWorkflowName("")
		if !errors.Is(err, ErrEmptyWorkflowName) {
			t.Errorf("expected ErrEmptyWorkflowName, got %v", err)
		}
	})
}

func TestStepName(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		sn, err := NewStepName("prd")
		if err != nil {
			t.Fatal(err)
		}
		if sn.String() != "prd" {
			t.Errorf("got %q", sn.String())
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		_, err := NewStepName("")
		if !errors.Is(err, ErrEmptyStepName) {
			t.Errorf("expected ErrEmptyStepName, got %v", err)
		}
	})
}

func TestProviderName(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		pn, err := NewProviderName("claude")
		if err != nil {
			t.Fatal(err)
		}
		if pn.String() != "claude" {
			t.Errorf("got %q", pn.String())
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		_, err := NewProviderName("")
		if !errors.Is(err, ErrEmptyProviderName) {
			t.Errorf("expected ErrEmptyProviderName, got %v", err)
		}
	})
}
