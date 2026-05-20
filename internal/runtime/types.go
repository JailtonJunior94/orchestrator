package runtime

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// Job descreve os parâmetros de uma execução ACP.
type Job struct {
	// Prompt é o texto enviado ao agente como entrada.
	Prompt string
	// WorkDir é o diretório de trabalho do agente.
	WorkDir string
	// EvidenceDir é o diretório onde artefatos de evidência são salvos.
	EvidenceDir string
	// ActivityTimeout é o timeout de inatividade. Zero = desabilitado.
	ActivityTimeout events.ActivityTimeout
	// Quiet suprime o output do renderer para o usuário.
	Quiet bool
}
