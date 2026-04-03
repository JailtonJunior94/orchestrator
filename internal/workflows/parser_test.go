package workflows

import (
	"context"
	"testing"
)

func TestParser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, wf *WorkflowDefinition)
	}{
		{
			name: "valid workflow",
			input: `
name: test-wf
steps:
  - name: step1
    provider: claude
    input: do something
`,
			check: func(t *testing.T, wf *WorkflowDefinition) {
				t.Helper()
				if wf.Name != "test-wf" {
					t.Errorf("name = %q", wf.Name)
				}
				if len(wf.Steps) != 1 {
					t.Fatalf("steps = %d", len(wf.Steps))
				}
				if wf.Steps[0].Name != "step1" {
					t.Errorf("step name = %q", wf.Steps[0].Name)
				}
			},
		},
		{
			name: "multiple steps",
			input: `
name: multi
steps:
  - name: a
    provider: claude
    input: input a
  - name: b
    provider: copilot
    input: input b
`,
			check: func(t *testing.T, wf *WorkflowDefinition) {
				t.Helper()
				if len(wf.Steps) != 2 {
					t.Fatalf("steps = %d, want 2", len(wf.Steps))
				}
			},
		},
		{
			name:    "invalid yaml",
			input:   ":\n  :\n  - [invalid",
			wantErr: true,
		},
		{
			name:  "empty yaml",
			input: "",
			check: func(t *testing.T, wf *WorkflowDefinition) {
				t.Helper()
				if wf.Name != "" {
					t.Errorf("expected empty name, got %q", wf.Name)
				}
			},
		},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			wf, err := p.Parse(context.Background(), []byte(tt.input))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				tt.check(t, wf)
			}
		})
	}
}
