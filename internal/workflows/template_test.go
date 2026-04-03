package workflows

import (
	"context"
	"errors"
	"testing"
)

func TestTemplateResolver(t *testing.T) {
	t.Parallel()

	r := NewTemplateResolver()

	tests := []struct {
		name    string
		tmpl    string
		vars    TemplateVars
		want    string
		wantErr error
	}{
		{
			name: "resolve input",
			tmpl: "Generate PRD for: {{input}}",
			vars: TemplateVars{Input: "login API"},
			want: "Generate PRD for: login API",
		},
		{
			name: "resolve step output",
			tmpl: "Based on: {{steps.prd.output}}",
			vars: TemplateVars{StepOutputs: map[string]string{"prd": "PRD content"}},
			want: "Based on: PRD content",
		},
		{
			name: "resolve multiple variables",
			tmpl: "Input: {{input}}, PRD: {{steps.prd.output}}, Spec: {{steps.techspec.output}}",
			vars: TemplateVars{
				Input:       "test",
				StepOutputs: map[string]string{"prd": "P", "techspec": "T"},
			},
			want: "Input: test, PRD: P, Spec: T",
		},
		{
			name:    "unresolved step reference",
			tmpl:    "Use: {{steps.missing.output}}",
			vars:    TemplateVars{StepOutputs: map[string]string{}},
			wantErr: ErrUnresolvedVariable,
		},
		{
			name:    "unknown variable",
			tmpl:    "Use: {{unknown}}",
			vars:    TemplateVars{},
			wantErr: ErrUnresolvedVariable,
		},
		{
			name: "no variables",
			tmpl: "plain text",
			vars: TemplateVars{},
			want: "plain text",
		},
		{
			name: "input with spaces in key",
			tmpl: "{{ input }}",
			vars: TemplateVars{Input: "trimmed"},
			want: "trimmed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := r.Resolve(context.Background(), tt.tmpl, tt.vars)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
