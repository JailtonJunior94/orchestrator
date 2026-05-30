// governance.go — hook Go equivalente a .claude/hooks/validate-governance.sh
// Replicação semântica exata do shell hook para modo ACP orchestrator.
// Shell hook permanece ativo para modo interativo Claude Code (coexistência).
//
// Semântica replicada de validate-governance.sh:
//
//	Valida que AGENTS.md existe no WorkDir da sessão.
//	Ausência do arquivo aborta a sessão (exit 1 equivalente = retornar erro).
//
// Diferença de escopo:
//
//	Shell hook: reage a edições (PostToolUse matcher Edit|Write)
//	Go hook: valida presença no ponto runtime.pre_open (antes de abrir sessão)
package hooks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrAgentsMDMissing é retornado quando AGENTS.md não é encontrado no WorkDir.
var ErrAgentsMDMissing = errors.New("AGENTS.md ausente no WorkDir: sessão abortada")

// GovernanceHook valida a presença de AGENTS.md em RuntimePreOpenEvent.WorkDir.
// Registrar no ponto PointRuntimePreOpen.
type GovernanceHook struct{}

var _ Hook = (*GovernanceHook)(nil)

// NewGovernanceHook cria um GovernanceHook pronto para registro.
func NewGovernanceHook() *GovernanceHook {
	return &GovernanceHook{}
}

// Name retorna o identificador do hook.
func (h *GovernanceHook) Name() string { return "governance" }

// Run verifica que AGENTS.md existe em evt.WorkDir.
// Retorna ErrAgentsMDMissing (wrapped com o caminho) se o arquivo não existir.
func (h *GovernanceHook) Run(ctx context.Context, evt Event) error {
	preOpen, ok := evt.(RuntimePreOpenEvent)
	if !ok {
		// Ponto errado: ignorar silenciosamente (hook registrado no ponto correto pelo caller).
		return nil
	}

	agentsMDPath := filepath.Join(preOpen.WorkDir, "AGENTS.md")
	if _, err := os.Stat(agentsMDPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrAgentsMDMissing, agentsMDPath)
		}
		return fmt.Errorf("verificar AGENTS.md: %w", err)
	}
	return nil
}
