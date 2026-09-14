package taskloop

import (
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestCompatibilityTableAcceptsProviderQualifiedModel(t *testing.T) {
	table := NewCompatibilityTable()
	for _, model := range []string{
		"anthropic/claude-sonnet-4",
		"google/gemini-2.5-pro",
		"openai/gpt-5",
		"claude-sonnet-4",
	} {
		if !table.IsSupported("opencode", model) {
			t.Errorf("IsSupported(opencode, %q) = false; want true", model)
		}
	}
	if table.IsSupported("opencode", "anthropic/modelo-inexistente") {
		t.Error("modelo desconhecido qualificado nao deve ser suportado")
	}
}

func TestNewAgentInvokerOpenCodeRequiresACP(t *testing.T) {
	invoker, err := NewAgentInvoker("opencode")
	if err == nil {
		t.Fatalf("esperado erro para opencode no runtime legacy, obteve invoker=%v", invoker)
	}
	var acpOnly *ACPOnlyToolError
	if !errors.As(err, &acpOnly) {
		t.Fatalf("erro deve ser *ACPOnlyToolError, obteve %T: %v", err, err)
	}
	if acpOnly.Tool != "opencode" {
		t.Errorf("ACPOnlyToolError.Tool = %q; want opencode", acpOnly.Tool)
	}
	if !errors.Is(err, ErrToolRequiresACP) {
		t.Error("erro deve satisfazer errors.Is(err, ErrToolRequiresACP)")
	}
	if !strings.Contains(err.Error(), "--runtime acp") {
		t.Errorf("mensagem deve orientar --runtime acp, obteve: %v", err)
	}

	if _, genericErr := NewAgentInvoker("tool-inexistente"); genericErr == nil {
		t.Fatal("esperado erro para ferramenta desconhecida")
	} else {
		var asACPOnly *ACPOnlyToolError
		if errors.As(genericErr, &asACPOnly) {
			t.Error("ferramenta desconhecida generica nao deve virar ACPOnlyToolError")
		}
	}
}

func TestNewExecutionProfileRemovedAgentIsTyped(t *testing.T) {
	for _, role := range []string{"executor", "reviewer"} {
		_, err := NewExecutionProfile(role, "gemini", "")
		if err == nil {
			t.Fatalf("role %s: esperado erro para gemini", role)
		}
		var removed *skills.RemovedAgentError
		if !errors.As(err, &removed) {
			t.Fatalf("role %s: erro deve ser *skills.RemovedAgentError, obteve %T: %v", role, err, err)
		}
		if removed.Agent != "gemini" {
			t.Errorf("role %s: RemovedAgentError.Agent = %q; want gemini", role, removed.Agent)
		}
	}

	_, err := NewExecutionProfile("executor", "ferramenta-inexistente", "")
	var removed *skills.RemovedAgentError
	if errors.As(err, &removed) {
		t.Error("ferramenta generica invalida nao deve virar RemovedAgentError")
	}
	if !errors.Is(err, ErrToolInvalida) {
		t.Errorf("ferramenta generica invalida deve manter ErrToolInvalida, obteve: %v", err)
	}
}
