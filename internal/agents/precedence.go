package agents

// RuntimeOverride contem os valores de configuracao de runtime vindos da CLI.
// Os campos Explicit* indicam se o campo correspondente foi explicitamente passado via CLI,
// distinguindo "nao setado" de "setado vazio". Espelha applyRuntimePrecedence do Compozy
// (internal/core/agents/execution.go) para paridade semantica (D-05).
type RuntimeOverride struct {
	// IDE e a ferramenta alvo (ex.: claude, codex, gemini, copilot).
	IDE string
	// Model e o modelo desejado (ex.: claude-opus-4-7).
	Model string
	// ReasoningEffort e o esforco de raciocinio: low | medium | high.
	ReasoningEffort string
	// AccessMode e o modo de acesso: bypass-permissions | review | readonly.
	AccessMode string

	// ExplicitIDE indica que IDE foi passado explicitamente via CLI.
	ExplicitIDE bool
	// ExplicitModel indica que Model foi passado explicitamente via CLI.
	ExplicitModel bool
	// ExplicitReasoningEffort indica que ReasoningEffort foi passado explicitamente via CLI.
	ExplicitReasoningEffort bool
	// ExplicitAccessMode indica que AccessMode foi passado explicitamente via CLI.
	ExplicitAccessMode bool
}

// ApplyRuntimePrecedence e a versao exportada de applyRuntimePrecedence para uso por pacotes externos
// (ex: internal/taskloop). Aplica a hierarquia CLI > AGENT.md > defaults.
func (c *Catalog) ApplyRuntimePrecedence(cfg *RuntimeOverride, defaults RuntimeDefaults) {
	NewCatalog().applyRuntimePrecedence(cfg, defaults)
}

// applyRuntimePrecedence aplica a hierarquia de precedencia:
// CLI flags > AGENT.md defaults > harness defaults.
// Campos marcados como Explicit* preservam o valor CLI.
// Campos nao marcados sao preenchidos com os defaults do agente quando disponiveis.
// Funcao pura: sem IO, sem cache, sem estado externo (D-05).
func (c *Catalog) applyRuntimePrecedence(cfg *RuntimeOverride, defaults RuntimeDefaults) {
	if !cfg.ExplicitIDE && defaults.IDE != "" {
		cfg.IDE = defaults.IDE
	}
	if !cfg.ExplicitModel && defaults.Model != "" {
		cfg.Model = defaults.Model
	}
	if !cfg.ExplicitReasoningEffort && defaults.ReasoningEffort != "" {
		cfg.ReasoningEffort = defaults.ReasoningEffort
	}
	if !cfg.ExplicitAccessMode && defaults.AccessMode != "" {
		cfg.AccessMode = defaults.AccessMode
	}
}
