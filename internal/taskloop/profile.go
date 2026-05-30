package taskloop

import (
	"errors"
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// Erros sentinela do dominio de perfil de execucao.
var (
	// ErrToolInvalida indica ferramenta nao reconhecida.
	ErrToolInvalida = errors.New("ferramenta nao suportada")

	// ErrRoleInvalido indica papel fora do vocabulario aceito.
	ErrRoleInvalido = errors.New("papel invalido — aceitos: executor, reviewer")

	// ErrFlagsConflitantes indica uso simultaneo de --tool e --executor-tool.
	ErrFlagsConflitantes = errors.New("flags de modo simples e avancado sao mutuamente exclusivas")
)

// _toolProviderMap mapeia ferramenta para seu provider.
var _toolProviderMap = map[string]string{
	"claude":  "anthropic",
	"codex":   "openai",
	"gemini":  "google",
	"copilot": "github",
}

// inferProvider retorna o provider para a ferramenta informada.
// Retorna string vazia se a ferramenta nao for reconhecida.
func (c *Catalog) inferProvider(tool string) string {
	return _toolProviderMap[tool]
}

// ExecutionProfile representa a configuracao completa de um papel no task-loop.
// Value Object: imutavel, auto-validante, fail fast no construtor.
type ExecutionProfile struct {
	role     string // "executor" ou "reviewer"
	tool     string // claude, codex, gemini, copilot
	provider string // anthropic, openai, google, github (inferido)
	model    string // modelo especifico ou "" (default da ferramenta)
}

// NewExecutionProfile cria um perfil validado.
// Retorna ErrRoleInvalido se role nao for "executor" ou "reviewer".
// Retorna ErrToolInvalida se tool nao pertencer ao conjunto valido.
func NewExecutionProfile(role, tool, model string) (ExecutionProfile, error) {
	if role != "executor" && role != "reviewer" {
		return ExecutionProfile{}, fmt.Errorf("%w: %q", ErrRoleInvalido, role)
	}

	provider, ok := _toolProviderMap[tool]
	if !ok {
		return ExecutionProfile{}, fmt.Errorf("%w: %q — opcoes: claude, codex, gemini, copilot", ErrToolInvalida, tool)
	}

	return ExecutionProfile{
		role:     role,
		tool:     tool,
		provider: provider,
		model:    model,
	}, nil
}

// Role retorna o papel do perfil.
func (p ExecutionProfile) Role() string { return p.role }

// Tool retorna a ferramenta do perfil.
func (p ExecutionProfile) Tool() string { return p.tool }

// Provider retorna o provider inferido para a ferramenta.
func (p ExecutionProfile) Provider() string { return p.provider }

// Model retorna o modelo configurado (pode ser vazio).
func (p ExecutionProfile) Model() string { return p.model }

// ProfileConfig agrupa perfis do executor e reviewer para uma execucao.
type ProfileConfig struct {
	Mode     string // "simples" ou "avancado"
	Executor ExecutionProfile
	Reviewer *ExecutionProfile // nil = reviewer nao configurado
}

// mapIDEToSpec converte o campo runtime.ide de um AGENT.md em uma specs.Spec concreta.
// Nesta fase, apenas "claude" e suportado em ACP; demais IDEs retornam erro acionavel.
// Multi-IDE via ACP e trabalho futuro (Fase 5 do PRD).
func (c *Catalog) mapIDEToSpec(ide string) (specs.Spec, error) {
	switch ide {
	case "claude":
		return specs.NewCatalog().Claude(), nil
	default:
		return specs.Spec{}, fmt.Errorf("ide %q ainda nao suportado em ACP — use --tool legado", ide)
	}
}

// ResolveProfileFromAgent deriva um ProfileConfig a partir de um ResolvedAgent e overrides CLI.
// Aplica a precedencia CLI > AGENT.md > defaults (D-05).
// Retorna erro quando o runtime.ide declarado nao e suportado em ACP nesta fase.
// allowUnknownModel controla se a validacao de compatibilidade runtime.model x CompatibilityTable
// e pulada (paridade com --allow-unknown-model existente).
func (c *Catalog) ResolveProfileFromAgent(agent agents.ResolvedAgent, override agents.RuntimeOverride, allowUnknownModel bool) (*ProfileConfig, error) {
	agents.
		// Aplicar precedencia: override CLI preenche campos nao setados com defaults do agente.
		NewCatalog().
		ApplyRuntimePrecedence(&override, agent.Runtime)

	// IDE final: usa override apos precedencia, ou fallback para "claude" como default ACP.
	ide := override.IDE
	if ide == "" {
		ide = "claude"
	}

	_, err := NewCatalog().mapIDEToSpec(ide)
	if err != nil {
		return nil, err
	}

	// Validar runtime.model contra CompatibilityTable (RF-09, T-13).
	// Modelo vazio e sempre aceito; --allow-unknown-model bypassa a validacao.
	if err := NewCatalog().ValidateModelForIDE(ide, override.Model, allowUnknownModel); err != nil {
		return nil, fmt.Errorf("agente %q: %w", agent.Name, err)
	}

	// Mapear para ExecutionProfile usando a ferramenta derivada do IDE.
	tool := ide
	model := override.Model

	executor, err := NewExecutionProfile("executor", tool, model)
	if err != nil {
		return nil, fmt.Errorf("perfil de agente invalido: %w", err)
	}

	return &ProfileConfig{
		Mode:     "agente",
		Executor: executor,
		Reviewer: nil, // reviewer nao e configurado via AGENT.md nesta fase
	}, nil
}

// ResolveProfiles converte flags CLI em ProfileConfig.
// Retorna nil quando tool != "" e execTool == "" (modo simples — caller usa Tool diretamente).
// Retorna ProfileConfig com Mode="avancado" quando execTool != "".
func (c *Catalog) ResolveProfiles(tool, execTool, execModel, revTool, revModel string) (*ProfileConfig, error) {
	// Regra 1: flags mutuamente exclusivas
	if tool != "" && (execTool != "" || revTool != "") {
		return nil, fmt.Errorf("%w", ErrFlagsConflitantes)
	}

	// Regra 5: model de executor sem tool de executor
	if execModel != "" && execTool == "" {
		return nil, fmt.Errorf("--executor-model requer --executor-tool")
	}

	// Regra 6: model de reviewer sem tool de reviewer
	if revModel != "" && revTool == "" {
		return nil, fmt.Errorf("--reviewer-model requer --reviewer-tool")
	}

	// Regra 2: modo simples — caller usa Tool diretamente
	if tool != "" {
		return nil, nil
	}

	// Regra 3: modo avancado — construir perfil do executor
	executor, err := NewExecutionProfile("executor", execTool, execModel)
	if err != nil {
		return nil, err
	}

	// Regra 4: reviewer herda executor quando omitido
	var reviewer *ExecutionProfile
	if revTool == "" {
		inherited, inheritErr := NewExecutionProfile("reviewer", executor.Tool(), executor.Model())
		if inheritErr != nil {
			return nil, inheritErr
		}
		reviewer = &inherited
	} else {
		revProfile, revErr := NewExecutionProfile("reviewer", revTool, revModel)
		if revErr != nil {
			return nil, revErr
		}
		reviewer = &revProfile
	}

	return &ProfileConfig{
		Mode:     "avancado",
		Executor: executor,
		Reviewer: reviewer,
	}, nil
}
