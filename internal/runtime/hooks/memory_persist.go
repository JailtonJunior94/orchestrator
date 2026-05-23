// memory_persist.go — hook de persistência de memória (F3-Claude).
// Implementa MemoryPersistHook completo: em PointSessionPostEnd escreve MEMORY.md
// com o resumo da sessão e emite métricas via log.
//
// Ponto: PointSessionPostEnd (session.post_end)
// Responsabilidade: escrever MEMORY.md em .specs/<prd>/memory/ via memory.Store.
// Formato: ≤150 linhas (conforme DefaultWorkflowLineLimit).
//
// HARD: não modificar .claude/hooks/*.sh — shell hooks coexistem (modo interativo).
package hooks

import (
	"context"
	"fmt"
	"log"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
)

// SessionSummary é o tipo concreto esperado em SessionPostEndEvent.Summary (F3-Claude).
// Exposto para desacoplar memory_persist do pacote runtime (evitar import cíclico).
type SessionSummary struct {
	TaskFileName string
	ExitStatus   string
	EventsCount  int
	ToolCalls    int
}

// MemoryPersistHook persiste o resumo da sessão em MEMORY.md.
// Registrar no ponto PointSessionPostEnd.
// Em PointSessionPostEnd: lê SessionPostEndEvent.Summary (tipo SessionSummary),
// constrói conteúdo MEMORY.md e grava via store.WriteWorkflow.
type MemoryPersistHook struct {
	store memory.Store
}

// NewMemoryPersistHook cria um MemoryPersistHook com o store fornecido.
// store deve ser instanciado via memory.New(tasksDir, limits) pelo runner.
func NewMemoryPersistHook(store memory.Store) *MemoryPersistHook {
	return &MemoryPersistHook{store: store}
}

// Name retorna o identificador do hook.
func (h *MemoryPersistHook) Name() string { return "memory_persist" }

// Run persiste o resumo da sessão em MEMORY.md.
// Evento esperado: SessionPostEndEvent com Summary do tipo SessionSummary.
// Evento inesperado: ignorado silenciosamente (ponto errado ou tipo errado).
func (h *MemoryPersistHook) Run(ctx context.Context, evt Event) error {
	postEnd, ok := evt.(SessionPostEndEvent)
	if !ok {
		return nil
	}

	summary, ok := postEnd.Summary.(SessionSummary)
	if !ok {
		// Summary ainda não tipado (pré-wiring): ignorar silenciosamente.
		return nil
	}

	content := buildMemoryContent(summary)

	if err := h.store.WriteWorkflow(ctx, content, memory.WriteModeReplace); err != nil {
		return fmt.Errorf("memory_persist: escrever MEMORY.md: %w", err)
	}

	// Emitir métricas para captura pelo execution_report.md.
	// Formato reconhecido pelo validate-evidence script.
	log.Printf("[memory_persist] Memory Compaction Requested: false")
	log.Printf("[memory_persist] Memory Workflow Bytes: %d", len(content))
	log.Printf("[memory_persist] Task: %s | Events: %d | Tool Calls: %d",
		summary.TaskFileName, summary.EventsCount, summary.ToolCalls)

	return nil
}

// buildMemoryContent constrói o conteúdo MEMORY.md a partir do resumo da sessão.
// Formato fixo ≤150 linhas (DefaultWorkflowLineLimit).
// Texto exato de cabeçalho compatível com Compozy memory format.
func buildMemoryContent(s SessionSummary) string {
	return fmt.Sprintf(`# Workflow Memory

Keep only durable, cross-task context here. Do not duplicate facts that are obvious from the repository, PRD documents, or git history.

## Last Session Summary

- Task: %s
- Exit Status: %s
- Events: %d
- Tool Calls: %d
`, s.TaskFileName, s.ExitStatus, s.EventsCount, s.ToolCalls)
}
