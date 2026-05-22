package specs

// Constantes do runtime GitHub Copilot CLI (ACP).
// Política de atualização (PRD §"Restrições Técnicas" + ADR-012 D-06):
//   - CopilotNpmVersion e CopilotSDKVersion são constantes Go pinadas. Nunca usar @latest.
//   - CopilotNpmVersion só é alterada via processo audit/ (tasks/templates/skill-upgrade-decision.md).
//   - CopilotSDKVersion é mantida em sincronia com go.mod por scripts/sync-acp-sdk-version.sh.
const (
	// CopilotNpmPackage é o nome do pacote npm canônico do Copilot CLI ACP.
	CopilotNpmPackage = "@github/copilot"

	// CopilotNpmVersion é a versão npm pinada do @github/copilot.
	// Pinada conforme ADR-012 D-06: constante Go atualizada somente via audit/.
	// Não alterar para @latest. Para atualizar: registrar decisão em audit/ e atualizar manualmente.
	// Confirmada via `npm view @github/copilot versions` em 2026-05-21: versão estável mais recente.
	CopilotNpmVersion = "1.0.51"

	// CopilotSDKVersion é a versão do coder/acp-go-sdk sincronizada com go.mod.
	// Mantida em sincronia por scripts/sync-acp-sdk-version.sh.
	// Não editar manualmente — use make sync-acp-sdk-version.
	CopilotSDKVersion = "v0.13.0" // mesmo SDK do Claude (mesma versão em go.mod)

	// CopilotMinCLIVersion é a versão mínima do binário `copilot` que expõe --acp.
	// Confirmada via `copilot --help` em 2026-05-21: flag "--acp Start as Agent Client Protocol server" presente.
	// Referência: https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server
	CopilotMinCLIVersion = "1.0.51"
)

// Copilot retorna a Spec do runtime GitHub Copilot CLI via ACP nativo.
// Binário canônico: "copilot" com FixedArgs=["--acp"].
// Fallback: npx --yes @github/copilot@<CopilotNpmVersion> --acp.
// AccessModeFlag vazio (D-07: sem flag análoga a --bypass-permissions do Claude no v0 do Copilot CLI).
// CopilotMaxTokens é a janela de contexto do Copilot CLI (GPT-4o): 64 000 tokens (ADR-023).
// Valor estático versionado; sobreposto por config de projeto quando necessário.
const CopilotMaxTokens = 64_000

func Copilot() Spec {
	return newSpec(
		"copilot",
		"GitHub Copilot CLI (ACP)",
		"copilot",
		[]string{"--acp"},
		[]FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{"--yes", CopilotNpmPackage + "@" + CopilotNpmVersion, "--acp"},
			},
		},
		"", // AccessModeFlag: vazio (ADR-012 D-07 — Copilot v0 sem flag análoga a --bypass-permissions)
		CopilotSDKVersion,
		CopilotNpmVersion,
		CopilotNpmPackage,
		ContextWindow{MaxTokens: CopilotMaxTokens},
	)
}
