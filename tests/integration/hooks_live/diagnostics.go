package hooks_live

const (
	PreToolMissingSkillDiagnostic         = "BLOQUEIO: tarefa toca arquivos cuja skill obrigatoria nao esta acessivel."
	PreToolPreloadDiagnostic              = "ERRO: governanca nao carregada para edicao de codigo"
	OpenCodePreToolDiagnostic             = "GOVERNANCE BLOCKED for tool"
	PostToolGovernanceDiagnostic          = "AVISO: arquivo de governanca modificado"
	OpenCodePostToolDiagnostic            = "GOVERNANCE OBSERVED at tool.execute.after"
	SessionEndDiagnostic                  = "[session-end] GATE DE ENCERRAMENTO BLOQUEADO — existe tarefa sem veredito que encerre o ciclo (APPROVED, ou APPROVED_WITH_REMARKS sem achado high/critical)."
	CodexPostToolFailedDiagnostic         = "hook: PostToolUse Failed"
	CodexStopBlockedDiagnostic            = "hook: Stop Blocked"
	GitOperationBlockedDiagnostic         = "GOVERNANCE BLOQUEIO: operacao git nao solicitada"
	DestructiveOperationBlockedDiagnostic = "GOVERNANCE BLOQUEIO: comando destrutivo detectado"
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
		{Diagnostic: GitOperationBlockedDiagnostic, SourceFile: ".agents/scripts/git-operation-gate.sh"},
		{Diagnostic: DestructiveOperationBlockedDiagnostic, SourceFile: ".agents/scripts/git-operation-gate.sh"},
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
	if agent == "codex" {
		return []string{PostToolGovernanceDiagnostic, CodexPostToolFailedDiagnostic}
	}
	return []string{PostToolGovernanceDiagnostic}
}

func SessionEndDiagnostics(agent string) []string {
	if agent == "codex" {
		return []string{SessionEndDiagnostic, CodexStopBlockedDiagnostic}
	}
	return []string{SessionEndDiagnostic}
}
