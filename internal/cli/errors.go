package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jailtonjunior/orchestrator/internal/state"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

func translateError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, workflows.ErrWorkflowNotFound):
		return fmt.Errorf("workflow não encontrado. Use `orq list` para ver os workflows disponíveis")
	case errors.Is(err, state.ErrNoPendingRuns):
		return fmt.Errorf("nenhuma execução pausada ou pendente foi encontrada em `.orq/runs`")
	case errors.Is(err, state.ErrRunNotFound):
		return fmt.Errorf("run não encontrada. Verifique o `--run-id` informado ou use `orq continue` sem argumentos")
	case strings.Contains(err.Error(), "not found in PATH"):
		return fmt.Errorf("provider indisponível: %s. Instale o CLI correspondente e confirme que ele está no PATH", err)
	case strings.Contains(err.Error(), "unknown provider"):
		return fmt.Errorf("workflow inválido: %s", err)
	default:
		return err
	}
}
