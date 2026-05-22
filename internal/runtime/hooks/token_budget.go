// token_budget.go — hook Go equivalente a .claude/hooks/validate-token-budget.sh
// Replicação semântica exata do shell hook para modo ACP orchestrator.
// Shell hook permanece ativo para modo interativo Claude Code (coexistência).
//
// Semântica replicada de validate-token-budget.sh:
//   Verifica se o tamanho do prompt excede o budget de tokens da ferramenta.
//   Shell hook: soma arquivos .md em .agents/skills + AGENTS.md (budget de governance).
//   Go hook: verifica o prompt completo já construído em PromptBuildEvent.
//   Budget padrão: "claude" (70000 tokens) via internal/metrics.CheckBudget.
//
// Diferença de escopo intencional:
//   Shell hook: verifica tamanho do contexto de governance (arquivos estáticos)
//   Go hook: verifica o prompt runtime completo (contexto dinâmico pós-build)
//   Ambos coexistem — scopes complementares, não duplicados.
package hooks

import (
	"context"
	"errors"
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/metrics"
)

// ErrTokenBudgetExceeded é retornado quando o prompt excede o budget de tokens.
var ErrTokenBudgetExceeded = errors.New("token budget excedido: prompt muito grande para o modelo")

// TokenBudgetHook valida o tamanho do prompt contra o budget da ferramenta.
// Registrar no ponto PointPromptPostBuild.
type TokenBudgetHook struct {
	// Tool é o identificador da ferramenta para lookup do budget (ex: "claude").
	// Default "claude" quando vazio.
	Tool string
}

// NewTokenBudgetHook cria um TokenBudgetHook para a ferramenta dada.
// tool: identificador da ferramenta (ver metrics.ToolBudgets); "" usa "claude".
func NewTokenBudgetHook(tool string) *TokenBudgetHook {
	if tool == "" {
		tool = "claude"
	}
	return &TokenBudgetHook{Tool: tool}
}

// Name retorna o identificador do hook.
func (h *TokenBudgetHook) Name() string { return "token_budget" }

// Run verifica se o prompt em PromptBuildEvent excede o budget da ferramenta.
// Retorna ErrTokenBudgetExceeded (com tokens e limite) se exceder.
func (h *TokenBudgetHook) Run(ctx context.Context, evt Event) error {
	promptEvt, ok := evt.(PromptBuildEvent)
	if !ok {
		// Ponto errado: ignorar silenciosamente.
		return nil
	}

	if promptEvt.Prompt == nil {
		return nil
	}

	tokens, limit, ok := metrics.CheckBudget(*promptEvt.Prompt, h.Tool)
	if !ok {
		return fmt.Errorf("%w: %d tokens (limite %d para %q)",
			ErrTokenBudgetExceeded, tokens, limit, h.Tool)
	}
	return nil
}
