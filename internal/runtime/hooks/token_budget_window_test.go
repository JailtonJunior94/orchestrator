package hooks_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

const openCodeReferenceWindow = specs.OpenCodeConservativeWindow

func openCodePromptEvent(chars int) hooks.PromptBuildEvent {
	prompt := strings.Repeat("a", chars)
	return hooks.PromptBuildEvent{Prompt: &prompt, Spec: "opencode"}
}

func TestTokenBudgetScalesWithResolvedWindow(t *testing.T) {
	evt := openCodePromptEvent(40_000)

	conservative := hooks.NewTokenBudgetHookWithWindow("opencode",
		specs.ContextWindow{MaxTokens: specs.OpenCodeConservativeWindow}, openCodeReferenceWindow)
	if err := conservative.Run(context.Background(), evt); !errors.Is(err, hooks.ErrTokenBudgetExceeded) {
		t.Fatalf("janela conservadora deveria estourar o teto; got: %v", err)
	}

	wide := hooks.NewTokenBudgetHookWithWindow("opencode",
		specs.ContextWindow{MaxTokens: 400_000}, openCodeReferenceWindow)
	if err := wide.Run(context.Background(), evt); err != nil {
		t.Fatalf("janela de 400k deveria acomodar o mesmo prompt (RF-17); got: %v", err)
	}
}

func TestTokenBudgetZeroWindowPreservesTabledLimit(t *testing.T) {
	evt := openCodePromptEvent(40_000)

	legacy := hooks.NewTokenBudgetHookWithClass("opencode", specs.WindowStandard)
	zeroWindow := hooks.NewTokenBudgetHookWithWindow("opencode", specs.ContextWindow{}, 0)

	legacyErr := legacy.Run(context.Background(), evt)
	zeroErr := zeroWindow.Run(context.Background(), evt)
	if (legacyErr == nil) != (zeroErr == nil) {
		t.Fatalf("zero-value de janela alterou o comportamento F1: legacy=%v zero=%v", legacyErr, zeroErr)
	}
}

func TestTokenBudgetNominalWindowPreservesTabledLimit(t *testing.T) {
	prompt := strings.Repeat("a", 300_000)
	evt := hooks.PromptBuildEvent{Prompt: &prompt, Spec: "claude"}

	legacy := hooks.NewTokenBudgetHookWithClass("claude", specs.WindowStandard)
	scaled := hooks.NewTokenBudgetHookWithWindow("claude",
		specs.ContextWindow{MaxTokens: specs.ClaudeMaxTokens}, specs.ClaudeMaxTokens)

	if (legacy.Run(context.Background(), evt) == nil) != (scaled.Run(context.Background(), evt) == nil) {
		t.Error("janela nominal deveria reproduzir o teto tabelado exato (F1)")
	}
}
