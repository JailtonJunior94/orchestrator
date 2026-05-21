package agents

// Scope indica o escopo de descoberta de um agente declarativo.
type Scope int

const (
	// ScopeGlobal indica um agente descoberto em ~/.ai-harness/agents/.
	ScopeGlobal Scope = iota
	// ScopeWorkspace indica um agente descoberto em <workdir>/.ai-harness/agents/.
	ScopeWorkspace
)

// String retorna a representacao textual do escopo.
func (s Scope) String() string {
	switch s {
	case ScopeWorkspace:
		return "workspace"
	default:
		return "global"
	}
}

// Metadata contem os metadados descritivos de um agente.
// E um value object imutavel (R-DDD-001).
type Metadata struct {
	// Description e a descricao funcional do agente.
	Description string
	// Version e a versao SemVer validada do agente.
	Version string
}

// RuntimeDefaults contem os valores padrao de configuracao de runtime declarados no AGENT.md.
// Todos os campos sao opcionais; valores vazios indicam que o harness deve usar seus proprios defaults.
// E um value object imutavel (R-DDD-001).
type RuntimeDefaults struct {
	// IDE e a ferramenta alvo: claude | codex | gemini | copilot.
	IDE string
	// Model e o modelo desejado (ex.: claude-opus-4-7). Validado em tarefa 8.0.
	Model string
	// ReasoningEffort e o esforco de raciocinio: low | medium | high.
	ReasoningEffort string
	// AccessMode e o modo de acesso: bypass-permissions | review | readonly.
	AccessMode string
}

// ResolvedAgent representa um agente declarativo completamente resolvido e validado.
// E produzido por ValidateAgentFrontmatter e pelo Registry. E um value object imutavel (R-DDD-001).
type ResolvedAgent struct {
	// Name e o identificador do agente, coincide com o nome do diretorio pai (RF-06).
	Name string
	// Scope indica se o agente e global ou de workspace.
	Scope Scope
	// Path e o caminho absoluto do arquivo AGENT.md.
	Path string
	// Metadata contem descricao e versao do agente.
	Metadata Metadata
	// Runtime contem os defaults de configuracao de runtime declarados.
	Runtime RuntimeDefaults
	// Prompt e o corpo do AGENT.md apos o bloco frontmatter.
	Prompt string
}
