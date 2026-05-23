package specs

// Constantes do runtime Claude.
// Política de atualização (PRD §"Restrições Técnicas" + ADR-009):
//   - ClaudeNpmVersion e ClaudeSDKVersion são constantes Go pinadas. Nunca usar @latest.
//   - ClaudeNpmVersion só é alterada via processo audit/ (tasks/templates/skill-upgrade-decision.md).
//   - ClaudeSDKVersion é mantida em sincronia com go.mod por scripts/sync-acp-sdk-version.sh.
const (
	// ClaudeNpmPackage é o nome do pacote npm canônico do agente Claude ACP.
	ClaudeNpmPackage = "@agentclientprotocol/claude-agent-acp"

	// ClaudeNpmVersion é a versão npm pinada do claude-agent-acp.
	// Pinada conforme ADR-009 §"Decisão": constante Go atualizada somente via audit/.
	// Não alterar para @latest. Para atualizar: registrar decisão em audit/ e rodar
	// make sync-acp-sdk-version (que só atualiza ClaudeSDKVersion via go.mod).
	// 0.1.0 nunca foi publicada no npm (versões reais começam em 0.24.0) — o fallback npx
	// falhava com ETARGET. Corrigido para versão publicada válida; revisar via audit/.
	ClaudeNpmVersion = "0.37.0"

	// ClaudeSDKVersion é a versão do coder/acp-go-sdk sincronizada com go.mod.
	// Mantida em sincronia por scripts/sync-acp-sdk-version.sh.
	// Não editar manualmente — use make sync-acp-sdk-version.
	ClaudeSDKVersion = "v0.13.0"
)

// Claude retorna a Spec do runtime Claude ACP.
// Binário canônico: "claude-agent-acp".
// Fallback: npx --yes @agentclientprotocol/claude-agent-acp@<ClaudeNpmVersion>.
// AccessModeFlag: "--bypass-permissions" (permite que o agente opere sem prompts interativos).
// ClaudeMaxTokens é a janela de contexto do Claude (claude-3.x e 4.x): 200 000 tokens (ADR-023).
// Valor estático versionado; sobreposto por config de projeto quando necessário.
const ClaudeMaxTokens = 200_000

func Claude() Spec {
	return newSpec(
		"claude",
		"Claude (ACP)",
		"claude-agent-acp",
		nil,
		[]FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{"--yes", ClaudeNpmPackage + "@" + ClaudeNpmVersion},
			},
		},
		"--bypass-permissions",
		ClaudeSDKVersion,
		ClaudeNpmVersion,
		ClaudeNpmPackage,
		ContextWindow{MaxTokens: ClaudeMaxTokens},
	)
}
