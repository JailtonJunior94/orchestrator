package domain

import (
	"errors"
	"testing"
)

func TestStepExecutionLifecycle(t *testing.T) {
	t.Parallel()

	def := StepDefinition{
		Name:     StepName{value: "prd"},
		Provider: ProviderName{value: "claude"},
		Input:    "generate prd",
	}
	s := NewStepExecution(def)

	if s.Status() != StepPending {
		t.Fatalf("initial status = %q", s.Status())
	}
	if s.Attempts() != 0 {
		t.Fatalf("initial attempts = %d", s.Attempts())
	}

	// Start
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if s.Status() != StepRunning {
		t.Errorf("status = %q, want running", s.Status())
	}
	if s.Attempts() != 1 {
		t.Errorf("attempts = %d, want 1", s.Attempts())
	}

	// MarkWaitingApproval
	if err := s.MarkWaitingApproval(StepResult{Output: "output text"}); err != nil {
		t.Fatal(err)
	}
	if s.Output() != "output text" {
		t.Errorf("output = %q", s.Output())
	}

	// Retry
	if err := s.Retry(); err != nil {
		t.Fatal(err)
	}
	if s.Status() != StepRetrying {
		t.Errorf("status = %q, want retrying", s.Status())
	}

	// Start again after retry
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if s.Attempts() != 2 {
		t.Errorf("attempts = %d, want 2", s.Attempts())
	}

	// Fail
	if err := s.MarkWaitingApproval(StepResult{Output: "output2"}); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFailed("timeout"); err != nil {
		t.Fatal(err)
	}
	if s.Error() != "timeout" {
		t.Errorf("error = %q", s.Error())
	}
}

func TestStepExecutionSkip(t *testing.T) {
	t.Parallel()

	s := NewStepExecution(StepDefinition{
		Name:     StepName{value: "skip-me"},
		Provider: ProviderName{value: "claude"},
		Input:    "input",
	})

	if err := s.Skip(); err != nil {
		t.Fatal(err)
	}
	if s.Status() != StepSkipped {
		t.Errorf("status = %q, want skipped", s.Status())
	}
}

func TestStepExecutionInvalidTransitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action func(s *StepExecution) error
	}{
		{"approve from pending", func(s *StepExecution) error { return s.Approve() }},
		{"retry from pending", func(s *StepExecution) error { return s.Retry() }},
		{"fail from pending", func(s *StepExecution) error { return s.MarkFailed("err") }},
		{"mark waiting from pending", func(s *StepExecution) error { return s.MarkWaitingApproval(StepResult{Output: "out"}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := NewStepExecution(StepDefinition{
				Name:     StepName{value: "test"},
				Provider: ProviderName{value: "claude"},
				Input:    "in",
			})
			err := tt.action(s)
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("expected ErrInvalidTransition, got %v", err)
			}
		})
	}
}

func TestStepExecutionSetInput(t *testing.T) {
	t.Parallel()

	s := NewStepExecution(StepDefinition{
		Name:     StepName{value: "test"},
		Provider: ProviderName{value: "claude"},
		Input:    "original",
	})
	s.SetInput("edited")
	if s.Input() != "edited" {
		t.Errorf("input = %q, want edited", s.Input())
	}
}

func TestStepExecutionGetters(t *testing.T) {
	t.Parallel()

	s := NewStepExecution(StepDefinition{
		Name:     StepName{value: "prd"},
		Provider: ProviderName{value: "copilot"},
		Input:    "gen",
	})

	if s.Name().String() != "prd" {
		t.Errorf("name = %q", s.Name().String())
	}
	if s.Provider().String() != "copilot" {
		t.Errorf("provider = %q", s.Provider().String())
	}
}
