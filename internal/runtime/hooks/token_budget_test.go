package hooks_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// Verifica que prompt dentro do budget retorna nil.
func TestTokenBudgetHook_WithinBudget(t *testing.T) {
	h := hooks.NewTokenBudgetHook("claude")

	small := strings.Repeat("a", 100) // ~28 tokens, bem abaixo de 70000
	evt := hooks.PromptBuildEvent{Prompt: &small, Spec: "spec-test"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("prompt pequeno deve estar dentro do budget; got: %v", err)
	}
}

// Verifica que prompt acima do budget retorna ErrTokenBudgetExceeded.
func TestTokenBudgetHook_ExceedsBudget(t *testing.T) {
	// copilot tem limite de 2000 tokens (~7000 chars); usamos string grande
	h := hooks.NewTokenBudgetHook("copilot")

	inflated := strings.Repeat("palavra ", 10000) // ~20000 tokens >> 2000
	evt := hooks.PromptBuildEvent{Prompt: &inflated, Spec: "spec-test"}

	err := h.Run(context.Background(), evt)
	if err == nil {
		t.Fatal("esperava erro para prompt acima do budget; got nil")
	}
	if !errors.Is(err, hooks.ErrTokenBudgetExceeded) {
		t.Errorf("erro = %v; deve envolver ErrTokenBudgetExceeded", err)
	}
}

// Verifica que prompt nil é ignorado silenciosamente.
func TestTokenBudgetHook_NilPrompt(t *testing.T) {
	h := hooks.NewTokenBudgetHook("claude")
	evt := hooks.PromptBuildEvent{Prompt: nil, Spec: "spec-test"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("prompt nil deve ser ignorado; got: %v", err)
	}
}

// Verifica que ferramenta desconhecida sempre retorna nil (sem budget definido).
func TestTokenBudgetHook_UnknownTool(t *testing.T) {
	h := hooks.NewTokenBudgetHook("ferramenta-desconhecida")

	huge := strings.Repeat("x", 1000000) // muito grande mas sem budget definido
	evt := hooks.PromptBuildEvent{Prompt: &huge, Spec: "spec-test"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("ferramenta desconhecida deve retornar nil; got: %v", err)
	}
}

// Verifica que evento de tipo errado é ignorado silenciosamente.
func TestTokenBudgetHook_WrongEventType(t *testing.T) {
	h := hooks.NewTokenBudgetHook("claude")
	evt := hooks.RuntimePreOpenEvent{WorkDir: "/tmp"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("evento errado deve ser ignorado; got: %v", err)
	}
}

// Verifica que "" como tool usa default "claude".
func TestTokenBudgetHook_DefaultTool(t *testing.T) {
	h := hooks.NewTokenBudgetHook("") // deve usar "claude"
	if h.Name() != "token_budget" {
		t.Errorf("Name() = %q; quero token_budget", h.Name())
	}

	small := strings.Repeat("b", 50)
	evt := hooks.PromptBuildEvent{Prompt: &small, Spec: "spec"}
	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("prompt pequeno com default claude: %v", err)
	}
}

// Verifica que WindowLarge usa teto generoso (gemini = 500k tokens).
func TestTokenBudgetHook_WindowLarge_GenerousBudget(t *testing.T) {
	h := hooks.NewTokenBudgetHookWithClass("gemini", specs.WindowLarge)

	// ~140k tokens — acima do budget padrão gemini (4000) mas dentro do Large (500k)
	medium := strings.Repeat("palavra ", 140_000)
	evt := hooks.PromptBuildEvent{Prompt: &medium, Spec: "gemini"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("prompt médio com gemini WindowLarge deve estar dentro do budget Large; got: %v", err)
	}
}

// Verifica que WindowStandard usa teto conservador (gemini padrão = 4000 tokens).
func TestTokenBudgetHook_WindowStandard_ConservativeBudget(t *testing.T) {
	h := hooks.NewTokenBudgetHookWithClass("gemini", specs.WindowStandard)

	// ~35k tokens — excede budget padrão gemini (4000)
	large := strings.Repeat("x ", 50_000)
	evt := hooks.PromptBuildEvent{Prompt: &large, Spec: "gemini"}

	err := h.Run(context.Background(), evt)
	if err == nil {
		t.Fatal("prompt grande com gemini WindowStandard deve exceder budget padrão; got nil")
	}
	if !errors.Is(err, hooks.ErrTokenBudgetExceeded) {
		t.Errorf("erro = %v; deve envolver ErrTokenBudgetExceeded", err)
	}
}

// Verifica que zero-value de WindowClass (WindowStandard) preserva comportamento F1.
func TestTokenBudgetHook_ZeroValue_WindowClass_IsF1(t *testing.T) {
	// NewTokenBudgetHook usa WindowStandard (zero-value) internamente.
	standard := hooks.NewTokenBudgetHook("gemini")
	explicit := hooks.NewTokenBudgetHookWithClass("gemini", specs.WindowStandard)

	small := strings.Repeat("a", 100)
	evtStandard := hooks.PromptBuildEvent{Prompt: &small, Spec: "gemini"}
	evtExplicit := hooks.PromptBuildEvent{Prompt: &small, Spec: "gemini"}

	errStandard := standard.Run(context.Background(), evtStandard)
	errExplicit := explicit.Run(context.Background(), evtExplicit)

	if errStandard != errExplicit {
		t.Errorf("zero-value WindowClass deve ser equivalente a WindowStandard: standard=%v explicit=%v", errStandard, errExplicit)
	}
}

// Verifica integração via Dispatcher.
func TestTokenBudgetHook_ViaDispatcher(t *testing.T) {
	d := hooks.New()
	d.Register(hooks.PointPromptPostBuild, hooks.NewTokenBudgetHook("copilot"))

	// Acima do budget copilot.
	huge := strings.Repeat("z ", 8000)
	evt := hooks.PromptBuildEvent{Prompt: &huge, Spec: "spec"}

	err := d.Dispatch(context.Background(), hooks.PointPromptPostBuild, evt)
	if err == nil {
		t.Fatal("esperava erro via Dispatcher; got nil")
	}
	if !errors.Is(err, hooks.ErrTokenBudgetExceeded) {
		t.Errorf("erro = %v; deve envolver ErrTokenBudgetExceeded", err)
	}
}
