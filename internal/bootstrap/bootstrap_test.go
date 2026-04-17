package bootstrap

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/acp"
	"github.com/jailtonjunior/orchestrator/internal/providers"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

func TestBootstrapValidatorAcceptsFourProviders(t *testing.T) {
	t.Parallel()

	registry := acp.NewRegistry(nil)
	validator := workflows.NewValidator(agentSpecNames(registry.List()))

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

	registry := acp.NewRegistry(nil)
	validator := workflows.NewValidator(agentSpecNames(registry.List()))

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

func TestAgentSpecNames_ReturnsSortedProviderNames(t *testing.T) {
	t.Parallel()

	names := agentSpecNames([]acp.AgentSpec{
		{Name: providers.GeminiProviderName},
		{Name: providers.ClaudeProviderName},
		{Name: providers.CodexProviderName},
	})

	if !slices.Equal(names, []string{providers.ClaudeProviderName, providers.CodexProviderName, providers.GeminiProviderName}) {
		t.Fatalf("names = %#v", names)
	}
}

func TestPermissionPolicyFromEnv_UsesProviderOverrides(t *testing.T) {
	t.Setenv("ORQ_ACP_PERMISSION_POLICY_CLAUDE", "deny")
	t.Setenv("ORQ_ACP_PERMISSION_POLICY_CODEX", "cancel")

	policy := permissionPolicyFromEnv([]acp.AgentSpec{
		{Name: providers.ClaudeProviderName},
		{Name: providers.CodexProviderName},
	})

	if policy.ProviderDecisions[providers.ClaudeProviderName] != "deny" {
		t.Fatalf("claude policy = %q", policy.ProviderDecisions[providers.ClaudeProviderName])
	}
	if policy.ProviderDecisions[providers.CodexProviderName] != "cancel" {
		t.Fatalf("codex policy = %q", policy.ProviderDecisions[providers.CodexProviderName])
	}
}
