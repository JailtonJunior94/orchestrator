// Package specs — runtime Gemini CLI via ACP nativo.
//
// Divergência intencional do Compozy (D-05 em ADR-015):
//
//	Compozy usa BootstrapArgs: nil para Gemini (sem mapeamento de AccessMode).
//	Este harness introduz geminiBootstrapArgs que mapeia AccessMode → --approval-mode,
//	espelhando o comportamento Codex (D-07 ADR-013) e preservando paridade de UX.
//	Model, reasoning e addDirs são ignorados por motivos técnicos documentados abaixo.
//
// Referências: ADR-015 D-01..D-05.
package specs

// Constantes do runtime Gemini CLI via ACP nativo.
// Política de atualização (ADR-015 D-02):
//   - GeminiNpmVersion e GeminiSDKVersion são constantes Go pinadas. Nunca usar @latest.
//   - GeminiNpmVersion só é alterada via processo audit/ (.specs/templates/skill-upgrade-decision.md).
//   - GeminiSDKVersion é mantida em sincronia com go.mod por scripts/sync-acp-sdk-version.sh.
//   - Validadas via `npm view @google/gemini-cli version dist-tags` em 2026-05-22:
//     dist-tag latest = 0.43.0. Flag --acp confirmada estável via
//     `npx --yes @google/gemini-cli@0.43.0 --acp --help`.
const (
	// GeminiNpmPackage é o nome do pacote npm canônico do Gemini CLI ACP.
	// ADR-015 D-01.
	GeminiNpmPackage = "@google/gemini-cli"

	// GeminiNpmVersion é a versão npm pinada do @google/gemini-cli.
	// Pinada conforme ADR-015 D-02: constante Go atualizada somente via audit/.
	// Não alterar para @latest.
	GeminiNpmVersion = "0.43.0"

	// GeminiSDKVersion é a versão do coder/acp-go-sdk sincronizada com go.mod.
	// Mesma versão de Claude/Codex/Copilot. Não editar manualmente.
	GeminiSDKVersion = "v0.13.0"

	// DefaultGeminiModel é o modelo default quando --model não é passado.
	// gemini-2.5-pro possui janela de contexto 1M+ tokens. ADR-015 D-03.
	DefaultGeminiModel = "gemini-2.5-pro"
)

// Gemini retorna a Spec do runtime Gemini CLI via ACP nativo.
//
// Binário canônico: "gemini" com FixedArgs=["--acp"].
// Fallback: npx --yes @google/gemini-cli@<GeminiNpmVersion> --acp.
// AccessModeFlag vazio — ADR-015 D-04: Gemini 0.43.0 não expõe flag análoga a --bypass-permissions;
// o controle de acesso é feito exclusivamente via --approval-mode em geminiBootstrapArgs (D-05).
// GeminiMaxTokens é a janela de contexto do Gemini CLI (gemini-2.5-pro): 1 000 000 tokens (ADR-023).
// Valor ≥ largeTokenThreshold (1M) ⇒ WindowLarge ⇒ limites ampliados de token_budget e memória.
// Valor estático versionado; sobreposto por config de projeto quando necessário.
const GeminiMaxTokens = 1_000_000

func Gemini() Spec {
	return newSpecWithBootstrap(
		"gemini",
		"Gemini (ACP)",
		"gemini",
		[]string{"--acp"},
		[]FallbackLauncher{{
			Command:   "npx",
			FixedArgs: []string{"--yes", GeminiNpmPackage + "@" + GeminiNpmVersion, "--acp"},
		}},
		"", // AccessModeFlag vazio — ADR-015 D-04
		GeminiSDKVersion, GeminiNpmVersion, GeminiNpmPackage,
		geminiBootstrapArgs,
		ContextWindow{MaxTokens: GeminiMaxTokens},
	)
}

// geminiBootstrapArgs implementa o mapeamento literal D-05 de ADR-015.
// Mapeia AccessMode em flag --approval-mode do Gemini CLI 0.43.0.
//
// Parâmetros ignorados intencionalmente (divergência Compozy — ADR-015 D-05):
//   - model: propagado via --model separado pelo ACPRunner, não via BootstrapArgs.
//   - reasoning: Gemini 2.5 não expõe controle programático de esforço de raciocínio.
//   - addDirs: CLI Gemini 0.43.0 não expõe flag equivalente a Claude --add-dir.
//
// Mapeamento:
//
//	AccessModeFull        → --approval-mode yolo   (acesso completo sem aprovação)
//	AccessModeRestricted  → --approval-mode default (modo padrão com aprovação)
//	"" (vazio)            → --approval-mode default (fallback seguro)
//	qualquer outro valor  → --approval-mode default (fallback seguro)
func geminiBootstrapArgs(_, _ string, _ []string, mode AccessMode) []string {
	switch mode {
	case AccessModeFull:
		return []string{"--approval-mode", "yolo"}
	case AccessModeRestricted, "":
		return []string{"--approval-mode", "default"}
	default:
		return []string{"--approval-mode", "default"}
	}
}
