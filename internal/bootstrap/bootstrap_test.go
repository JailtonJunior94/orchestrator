package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/providers"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

func TestBootstrapValidatorAcceptsFourProviders(t *testing.T) {
	t.Parallel()

	validator := workflows.NewValidator([]string{
		providers.ClaudeProviderName,
		providers.CopilotProviderName,
		providers.GeminiProviderName,
		providers.CodexProviderName,
	})

	for _, providerName := range []string{
		providers.ClaudeProviderName,
		providers.CopilotProviderName,
		providers.GeminiProviderName,
		providers.CodexProviderName,
	} {
		wf := &workflows.WorkflowDefinition{
			Name: "test-workflow",
			Steps: []workflows.StepDefinition{
				{Name: "step1", Provider: providerName, Input: "do something"},
			},
		}
		if err := validator.Validate(context.Background(), wf); err != nil {
			t.Fatalf("provider %q should be valid, got: %v", providerName, err)
		}
	}
}

func TestBootstrapValidatorRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	validator := workflows.NewValidator([]string{
		providers.ClaudeProviderName,
		providers.CopilotProviderName,
		providers.GeminiProviderName,
		providers.CodexProviderName,
	})

	wf := &workflows.WorkflowDefinition{
		Name: "test-workflow",
		Steps: []workflows.StepDefinition{
			{Name: "step1", Provider: "openrouter", Input: "do something"},
		},
	}
	err := validator.Validate(context.Background(), wf)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "openrouter") {
		t.Fatalf("error should mention unknown provider name, got: %v", err)
	}
}
