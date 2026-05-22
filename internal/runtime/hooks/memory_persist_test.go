package hooks_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
)

// newTestStore cria um memory.Store para testes usando um diretório temporário.
func newTestStore(t *testing.T) (memory.Store, string) {
	t.Helper()
	tasksDir := t.TempDir()
	limits := memory.DefaultLimits()
	return memory.New(tasksDir, limits), tasksDir
}

// TestMemoryPersistHook_Name valida o identificador do hook.
func TestMemoryPersistHook_Name(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	h := hooks.NewMemoryPersistHook(store)

	if h.Name() != "memory_persist" {
		t.Errorf("Name() = %q; quero memory_persist", h.Name())
	}
}

// TestMemoryPersistHook_WritesMEMORYMD valida que o hook escreve MEMORY.md corretamente.
func TestMemoryPersistHook_WritesMEMORYMD(t *testing.T) {
	t.Parallel()

	store, tasksDir := newTestStore(t)
	h := hooks.NewMemoryPersistHook(store)

	summary := hooks.SessionSummary{
		TaskFileName: "task-6.0-runner-integrate-f3.md",
		ExitStatus:   "ok",
		EventsCount:  42,
		ToolCalls:    7,
	}

	evt := hooks.SessionPostEndEvent{Summary: summary}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Fatalf("Run() erro inesperado: %v", err)
	}

	memoryPath := filepath.Join(tasksDir, "memory", "MEMORY.md")
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		t.Fatalf("MEMORY.md nao foi criado em %s: %v", memoryPath, err)
	}

	content := string(data)

	// Verificar estrutura canônica do MEMORY.md (formato Compozy).
	if !strings.Contains(content, "# Workflow Memory") {
		t.Error("MEMORY.md deve conter '# Workflow Memory'")
	}
	if !strings.Contains(content, "## Last Session Summary") {
		t.Error("MEMORY.md deve conter '## Last Session Summary'")
	}
	if !strings.Contains(content, "task-6.0-runner-integrate-f3.md") {
		t.Error("MEMORY.md deve conter o TaskFileName")
	}
	if !strings.Contains(content, "ok") {
		t.Error("MEMORY.md deve conter o ExitStatus")
	}
	if !strings.Contains(content, "42") {
		t.Error("MEMORY.md deve conter EventsCount")
	}
	if !strings.Contains(content, "7") {
		t.Error("MEMORY.md deve conter ToolCalls")
	}

	// Verificar que está dentro do limite de 150 linhas.
	lineCount := strings.Count(content, "\n")
	if lineCount > memory.DefaultWorkflowLineLimit {
		t.Errorf("MEMORY.md tem %d linhas; limite é %d", lineCount, memory.DefaultWorkflowLineLimit)
	}
}

// TestMemoryPersistHook_IgnoresWrongEventType valida que hooks com ponto errado são ignorados.
func TestMemoryPersistHook_IgnoresWrongEventType(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	h := hooks.NewMemoryPersistHook(store)

	// Evento de ponto errado: RuntimePreOpenEvent
	wrongEvt := hooks.RuntimePreOpenEvent{
		WorkDir:  "/tmp",
		SpecID:   "claude",
		Launcher: "claude",
	}

	if err := h.Run(context.Background(), wrongEvt); err != nil {
		t.Errorf("Run() com evento errado deve retornar nil; got: %v", err)
	}
}

// TestMemoryPersistHook_IgnoresNilSummary valida que Summary com tipo errado é ignorado.
func TestMemoryPersistHook_IgnoresNilSummary(t *testing.T) {
	t.Parallel()

	store, _ := newTestStore(t)
	h := hooks.NewMemoryPersistHook(store)

	// Summary com tipo errado (string em vez de SessionSummary).
	evt := hooks.SessionPostEndEvent{Summary: "wrong-type"}

	if err := h.Run(context.Background(), evt); err != nil {
		t.Errorf("Run() com Summary tipo errado deve retornar nil; got: %v", err)
	}
}

// TestMemoryPersistHook_ViaDispatcher valida registro + dispatch via Dispatcher.
func TestMemoryPersistHook_ViaDispatcher(t *testing.T) {
	t.Parallel()

	store, tasksDir := newTestStore(t)
	h := hooks.NewMemoryPersistHook(store)

	d := hooks.New()
	d.Register(hooks.PointSessionPostEnd, h)

	summary := hooks.SessionSummary{
		TaskFileName: "task-1.0.md",
		ExitStatus:   "done",
		EventsCount:  10,
		ToolCalls:    3,
	}

	evt := hooks.SessionPostEndEvent{Summary: summary}

	if err := d.Dispatch(context.Background(), hooks.PointSessionPostEnd, evt); err != nil {
		t.Fatalf("Dispatch erro: %v", err)
	}

	// Verificar que MEMORY.md foi criado.
	memoryPath := filepath.Join(tasksDir, "memory", "MEMORY.md")
	if _, err := os.Stat(memoryPath); err != nil {
		t.Errorf("MEMORY.md deve existir após dispatch: %v", err)
	}
}

// TestMemoryPersistHook_OverwritesPreviousContent valida que WriteModeReplace sobrescreve conteúdo anterior.
func TestMemoryPersistHook_OverwritesPreviousContent(t *testing.T) {
	t.Parallel()

	store, tasksDir := newTestStore(t)
	h := hooks.NewMemoryPersistHook(store)

	// Primeira sessão.
	evt1 := hooks.SessionPostEndEvent{Summary: hooks.SessionSummary{
		TaskFileName: "task-1.0.md",
		ExitStatus:   "done",
		EventsCount:  5,
		ToolCalls:    2,
	}}
	if err := h.Run(context.Background(), evt1); err != nil {
		t.Fatalf("primeira Run() erro: %v", err)
	}

	// Segunda sessão — deve sobrescrever.
	evt2 := hooks.SessionPostEndEvent{Summary: hooks.SessionSummary{
		TaskFileName: "task-2.0.md",
		ExitStatus:   "done",
		EventsCount:  20,
		ToolCalls:    8,
	}}
	if err := h.Run(context.Background(), evt2); err != nil {
		t.Fatalf("segunda Run() erro: %v", err)
	}

	memoryPath := filepath.Join(tasksDir, "memory", "MEMORY.md")
	data, err := os.ReadFile(memoryPath)
	if err != nil {
		t.Fatalf("MEMORY.md nao encontrado: %v", err)
	}

	content := string(data)

	// Deve conter dados da segunda sessão.
	if !strings.Contains(content, "task-2.0.md") {
		t.Error("MEMORY.md deve conter TaskFileName da segunda sessão")
	}
	// Não deve conter dados da primeira sessão (sobrescrito).
	if strings.Contains(content, "task-1.0.md") {
		t.Error("MEMORY.md não deve conter TaskFileName da primeira sessão (deve ter sido sobrescrito)")
	}
}
