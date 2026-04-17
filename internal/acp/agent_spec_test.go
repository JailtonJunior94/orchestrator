package acp_test

import (
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/acp"
)

func TestAgentSpec_Validate(t *testing.T) {
	tests := []struct {
		name    string
		spec    acp.AgentSpec
		wantErr bool
	}{
		{
			name: "valid with binary",
			spec: acp.AgentSpec{
				Name:   "claude",
				Binary: "claude-agent-acp",
			},
			wantErr: false,
		},
		{
			name: "valid with fallback only",
			spec: acp.AgentSpec{
				Name:        "codex",
				FallbackCmd: "npx",
				FallbackArgs: []string{"--yes", "@zed-industries/codex-acp"},
			},
			wantErr: false,
		},
		{
			name:    "missing name",
			spec:    acp.AgentSpec{Binary: "claude-agent-acp"},
			wantErr: true,
		},
		{
			name:    "missing binary and fallback",
			spec:    acp.AgentSpec{Name: "claude"},
			wantErr: true,
		},
		{
			name:    "empty spec",
			spec:    acp.AgentSpec{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
