package runtime

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
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

	// Model é o identificador do modelo a ser usado pelo agente.
	// Para Codex: ex. "gpt-5.5" (default specs.DefaultCodexModel).
	// Default "" (string vazia) é tratado como no-op por codexBootstrapArgs.
	// Ignorado por Claude/Copilot via no-op em Spec.BootstrapArgs. ADR-013 D-02.
	Model string

	// ReasoningEffort define o nível de esforço de raciocínio do agente Codex.
	// Valores aceitos: "low", "medium", "high". Default "" omite o flag
	// (Codex usa o padrão do modelo). Ignorado por Claude/Copilot via no-op
	// em Spec.BootstrapArgs. ADR-013 D-02.
	ReasoningEffort string

	// AccessMode define o nível de acesso ao sistema de arquivos solicitado ao
	// agente Codex. Default specs.AccessModeRestricted ("restricted").
	// specs.AccessModeFull ("full") deve ser usado apenas em ambientes isolados.
	// Ignorado por Claude/Copilot via no-op em Spec.BootstrapArgs. ADR-013 D-02.
	AccessMode specs.AccessMode

	// AddDirs lista diretórios adicionais que o agente Codex pode acessar além
	// do WorkDir. Requer SupportsAddDirs=true no compozy. Default nil (sem
	// diretórios extras). Ignorado por Claude/Copilot via no-op em
	// Spec.BootstrapArgs. ADR-013 D-02.
	AddDirs []string
}
