package hooks_live

const (
	PreToolMissingSkillDiagnostic = "BLOQUEIO: tarefa toca arquivos cuja skill obrigatoria nao esta acessivel."
	PreToolPreloadDiagnostic      = "ERRO: governanca nao carregada para edicao de codigo"
	OpenCodePreToolDiagnostic     = "GOVERNANCE BLOCKED for tool"
	PostToolGovernanceDiagnostic  = "AVISO: arquivo de governanca modificado"
	OpenCodePostToolDiagnostic    = "GOVERNANCE OBSERVED at tool.execute.after"
	SessionEndDiagnostic          = "[session-end] GATE DE ENCERRAMENTO BLOQUEADO — existe tarefa sem veredito APPROVED registrado."
)

const (
	CopilotRepoHooksOptInEnvVar   = "GITHUB_COPILOT_PROMPT_MODE_REPO_HOOKS"
	CopilotRepoHooksOptInEnvValue = "true"
)

type DiagnosticSource struct {
	Diagnostic string
	SourceFile string
}

func CanonicalDiagnosticSources() []DiagnosticSource {
	return []DiagnosticSource{
		{Diagnostic: PreToolMissingSkillDiagnostic, SourceFile: ".agents/scripts/validate-skill-prerequisites.sh"},
		{Diagnostic: PreToolPreloadDiagnostic, SourceFile: ".agents/hooks/validate-preload.sh"},
		{Diagnostic: OpenCodePreToolDiagnostic, SourceFile: ".opencode/plugin/governance.js"},
		{Diagnostic: PostToolGovernanceDiagnostic, SourceFile: ".agents/hooks/validate-governance.sh"},
		{Diagnostic: OpenCodePostToolDiagnostic, SourceFile: ".opencode/plugin/governance.js"},
		{Diagnostic: SessionEndDiagnostic, SourceFile: ".agents/hooks/validate-session-end.sh"},
	}
}

func PreToolDenialDiagnostics(agent string) []string {
	if agent == "opencode" {
		return []string{OpenCodePreToolDiagnostic}
	}
	return []string{PreToolMissingSkillDiagnostic, PreToolPreloadDiagnostic}
}

func PostToolDiagnostics(agent string) []string {
	if agent == "opencode" {
		return []string{OpenCodePostToolDiagnostic}
	}
	return []string{PostToolGovernanceDiagnostic}
}

func SessionEndDiagnostics(agent string) []string {
	return []string{SessionEndDiagnostic}
}
