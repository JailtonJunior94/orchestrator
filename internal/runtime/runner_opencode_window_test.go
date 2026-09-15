package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func openCodeWorkDir(t *testing.T, configModel string) string {
	t.Helper()
	dir := t.TempDir()
	if configModel == "" {
		return dir
	}
	content := []byte(`{"model": "` + configModel + `"}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, specs.OpenCodeConfigFileName), content, 0o644); err != nil {
		t.Fatalf("setup opencode.json: %v", err)
	}
	return dir
}

func TestResolveEffectiveModelUsesOpenCodeConfigModel(t *testing.T) {
	runner := NewACPRunner(specs.NewCatalog().OpenCode())
	job := Job{WorkDir: openCodeWorkDir(t, "openai/gpt-5")}

	effective, err := runner.resolveEffectiveModel(job)
	if err != nil {
		t.Fatalf("resolveEffectiveModel: %v", err)
	}
	if effective != "openai/gpt-5" {
		t.Fatalf("modelo efetivo = %q, quero openai/gpt-5", effective)
	}
	if got := runner.spec.ResolveWindow(effective).MaxTokens; got != 400_000 {
		t.Errorf("janela = %d, quero 400000 derivada de opencode.json (RF-17)", got)
	}
}

func TestResolveEffectiveModelRejectsDivergentFlag(t *testing.T) {
	runner := NewACPRunner(specs.NewCatalog().OpenCode())
	job := Job{WorkDir: openCodeWorkDir(t, "openai/gpt-5"), Model: "anthropic/claude-sonnet-4-5"}

	if _, err := runner.resolveEffectiveModel(job); !errors.Is(err, specs.ErrOpenCodeModelMismatch) {
		t.Fatalf("erro = %v, quero ErrOpenCodeModelMismatch", err)
	}
}

func TestRunRejectsDivergentOpenCodeModelBeforeLaunching(t *testing.T) {
	runner := NewACPRunner(specs.NewCatalog().OpenCode())
	job := Job{WorkDir: openCodeWorkDir(t, "openai/gpt-5"), Model: "anthropic/claude-sonnet-4-5"}

	if _, err := runner.Run(context.Background(), job); !errors.Is(err, specs.ErrOpenCodeModelMismatch) {
		t.Fatalf("Run erro = %v, quero ErrOpenCodeModelMismatch", err)
	}
}

func TestResolveEffectiveModelAgreeingFlagIsAccepted(t *testing.T) {
	runner := NewACPRunner(specs.NewCatalog().OpenCode())
	job := Job{WorkDir: openCodeWorkDir(t, "openai/gpt-5"), Model: "openai/gpt-5"}

	effective, err := runner.resolveEffectiveModel(job)
	if err != nil {
		t.Fatalf("resolveEffectiveModel: %v", err)
	}
	if effective != "openai/gpt-5" {
		t.Errorf("modelo efetivo = %q, quero openai/gpt-5", effective)
	}
}

func TestResolveEffectiveModelWithoutConfigKeepsJobModel(t *testing.T) {
	runner := NewACPRunner(specs.NewCatalog().OpenCode())
	job := Job{WorkDir: openCodeWorkDir(t, ""), Model: "openai/gpt-5"}

	effective, err := runner.resolveEffectiveModel(job)
	if err != nil {
		t.Fatalf("resolveEffectiveModel: %v", err)
	}
	if effective != "openai/gpt-5" {
		t.Errorf("modelo efetivo = %q, quero openai/gpt-5 (fallback para a flag)", effective)
	}
}

func TestResolveEffectiveModelOtherSpecsIgnoreOpenCodeConfig(t *testing.T) {
	runner := NewACPRunner(specs.NewCatalog().Claude())
	job := Job{WorkDir: openCodeWorkDir(t, "openai/gpt-5"), Model: "claude-sonnet-4-6"}

	effective, err := runner.resolveEffectiveModel(job)
	if err != nil {
		t.Fatalf("resolveEffectiveModel: %v", err)
	}
	if effective != "claude-sonnet-4-6" {
		t.Errorf("modelo efetivo = %q, quero claude-sonnet-4-6 (spec nao-opencode intocada)", effective)
	}
}

func TestPrepareHooksDispatcherScalesBudgetFromJobWindow(t *testing.T) {
	prompt := strings.Repeat("a", 40_000)
	evt := hooks.PromptBuildEvent{Prompt: &prompt, Spec: specs.OpenCodeSpecID}

	conservative := NewCatalog().prepareHooksDispatcher(
		Job{WindowMaxTokens: specs.OpenCodeConservativeWindow, WorkDir: t.TempDir()},
		specs.OpenCodeSpecID, nil, specs.OpenCodeConservativeWindow, nil, nil, nil)
	if err := conservative.Dispatch(context.Background(), hooks.PointPromptPostBuild, evt); !errors.Is(err, hooks.ErrTokenBudgetExceeded) {
		t.Fatalf("janela conservadora no Job deveria estourar o teto; got: %v", err)
	}

	wide := NewCatalog().prepareHooksDispatcher(
		Job{WindowMaxTokens: 400_000, WorkDir: t.TempDir()},
		specs.OpenCodeSpecID, nil, specs.OpenCodeConservativeWindow, nil, nil, nil)
	if err := wide.Dispatch(context.Background(), hooks.PointPromptPostBuild, evt); err != nil {
		t.Fatalf("Job com janela de 400k deveria acomodar o mesmo prompt (RF-17); got: %v", err)
	}
}
