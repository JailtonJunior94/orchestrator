package hooks_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
)

// T-HOOK-04: hook governance retorna erro claro quando AGENTS.md ausente.
func TestGovernanceHook_AgentsMDAbsent(t *testing.T) {
	dir := t.TempDir() // diretório temporário sem AGENTS.md

	h := hooks.NewGovernanceHook()
	evt := hooks.RuntimePreOpenEvent{WorkDir: dir, SpecID: "spec-test"}

	err := h.Run(context.Background(), evt)
	if err == nil {
		t.Fatal("esperava erro quando AGENTS.md ausente; got nil")
	}
	if !errors.Is(err, hooks.ErrAgentsMDMissing) {
		t.Errorf("erro = %v; deve envolver ErrAgentsMDMissing", err)
	}
}

// T-HOOK-04b: hook governance retorna nil quando AGENTS.md presente.
func TestGovernanceHook_AgentsMDPresent(t *testing.T) {
	dir := t.TempDir()
	// Criar AGENTS.md mínimo.
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatalf("criar AGENTS.md: %v", err)
	}

	h := hooks.NewGovernanceHook()
	evt := hooks.RuntimePreOpenEvent{WorkDir: dir, SpecID: "spec-test"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("esperava nil quando AGENTS.md presente; got: %v", err)
	}
}

// Verifica que evento de tipo errado é ignorado silenciosamente.
func TestGovernanceHook_WrongEventType(t *testing.T) {
	h := hooks.NewGovernanceHook()
	prompt := "prompt qualquer"
	evt := hooks.PromptBuildEvent{Prompt: &prompt, Spec: "spec"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("evento errado deve ser ignorado; got: %v", err)
	}
}

// Verifica que o hook pode ser registrado e despachado via Dispatcher.
func TestGovernanceHook_ViaDispatcher_Absent(t *testing.T) {
	dir := t.TempDir() // sem AGENTS.md

	d := hooks.New()
	d.Register(hooks.PointRuntimePreOpen, hooks.NewGovernanceHook())

	evt := hooks.RuntimePreOpenEvent{WorkDir: dir}
	err := d.Dispatch(context.Background(), hooks.PointRuntimePreOpen, evt)
	if err == nil {
		t.Fatal("esperava erro via Dispatcher; got nil")
	}
	if !errors.Is(err, hooks.ErrAgentsMDMissing) {
		t.Errorf("erro = %v; deve envolver ErrAgentsMDMissing", err)
	}
}

func TestGovernanceHook_ViaDispatcher_Present(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatalf("criar AGENTS.md: %v", err)
	}

	d := hooks.New()
	d.Register(hooks.PointRuntimePreOpen, hooks.NewGovernanceHook())

	evt := hooks.RuntimePreOpenEvent{WorkDir: dir}
	if err := d.Dispatch(context.Background(), hooks.PointRuntimePreOpen, evt); err != nil {
		t.Fatalf("esperava nil via Dispatcher; got: %v", err)
	}
}
