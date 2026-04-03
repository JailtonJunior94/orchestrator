package workflows

import (
	"context"
	"errors"
	"testing"
)

func TestValidator(t *testing.T) {
	t.Parallel()

	v := NewValidator([]string{"claude", "copilot"})

	tests := []struct {
		name    string
		wf      WorkflowDefinition
		wantErr error
	}{
		{
			name: "valid workflow",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "do {{input}}"},
				},
			},
		},
		{
			name: "valid with step reference",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "do {{input}}"},
					{Name: "s2", Provider: "copilot", Input: "use {{steps.s1.output}}"},
				},
			},
		},
		{
			name: "valid timeout and schema",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "do {{input}}", Timeout: "30s", Schema: `{"type":"object"}`},
				},
			},
		},
		{
			name:    "empty name",
			wf:      WorkflowDefinition{Steps: []StepDefinition{{Name: "s", Provider: "claude", Input: "x"}}},
			wantErr: ErrEmptyWorkflowName,
		},
		{
			name:    "no steps",
			wf:      WorkflowDefinition{Name: "test"},
			wantErr: ErrNoSteps,
		},
		{
			name: "empty step name",
			wf: WorkflowDefinition{
				Name:  "test",
				Steps: []StepDefinition{{Name: "", Provider: "claude", Input: "x"}},
			},
			wantErr: ErrEmptyStepName,
		},
		{
			name: "empty provider",
			wf: WorkflowDefinition{
				Name:  "test",
				Steps: []StepDefinition{{Name: "s", Provider: "", Input: "x"}},
			},
			wantErr: ErrEmptyProvider,
		},
		{
			name: "empty input",
			wf: WorkflowDefinition{
				Name:  "test",
				Steps: []StepDefinition{{Name: "s", Provider: "claude", Input: ""}},
			},
			wantErr: ErrEmptyInput,
		},
		{
			name: "duplicate step name",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "x"},
					{Name: "s1", Provider: "claude", Input: "y"},
				},
			},
			wantErr: ErrDuplicateStepName,
		},
		{
			name: "unknown provider",
			wf: WorkflowDefinition{
				Name:  "test",
				Steps: []StepDefinition{{Name: "s", Provider: "gemini", Input: "x"}},
			},
			wantErr: ErrInvalidProvider,
		},
		{
			name: "reference to unknown step",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "use {{steps.nonexistent.output}}"},
				},
			},
			wantErr: ErrInvalidStepReference,
		},
		{
			name: "self reference",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "use {{steps.s1.output}}"},
				},
			},
			wantErr: ErrInvalidStepReference,
		},
		{
			name: "forward reference",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "use {{steps.s2.output}}"},
					{Name: "s2", Provider: "claude", Input: "x"},
				},
			},
			wantErr: ErrInvalidStepReference,
		},
		{
			name: "invalid timeout",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "x", Timeout: "soon"},
				},
			},
			wantErr: ErrInvalidTimeout,
		},
		{
			name: "invalid schema",
			wf: WorkflowDefinition{
				Name: "test",
				Steps: []StepDefinition{
					{Name: "s1", Provider: "claude", Input: "x", Schema: `{invalid}`},
				},
			},
			wantErr: ErrInvalidSchema,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := v.Validate(context.Background(), &tt.wf)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
