package bootstrap

import (
	"bytes"
	"context"
	"log/slog"
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

func TestNewLoggerWritesToConfiguredOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := newLogger(&buf)

	logger.Info("workflow step", slog.String("step", "prd"))

	if got := buf.String(); !strings.Contains(got, "workflow step") {
		t.Fatalf("expected log output to contain message, got %q", got)
	}
	if !strings.Contains(buf.String(), "step=prd") {
		t.Fatalf("expected log output to contain structured field, got %q", buf.String())
	}
}

func TestNewLoggerUsesDiscardWhenOutputIsNil(t *testing.T) {
	t.Parallel()

	logger := newLogger(nil)

	logger.Info("workflow step", slog.String("step", "prd"))
}
