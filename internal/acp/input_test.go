package acp_test

import (
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/acp"
)

func TestACPInput_Validate(t *testing.T) {
	tests := []struct {
		name    string
		input   acp.ACPInput
		wantErr bool
	}{
		{
			name:    "valid input",
			input:   acp.ACPInput{Prompt: "do something useful", WorkDir: "/tmp"},
			wantErr: false,
		},
		{
			name:    "empty prompt",
			input:   acp.ACPInput{Prompt: "", WorkDir: "/tmp"},
			wantErr: true,
		},
		{
			name:    "prompt with resume session",
			input:   acp.ACPInput{Prompt: "continue", SessionID: "abc-123"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
