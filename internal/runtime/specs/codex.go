// Package specs — runtime Codex via ACP adapter @zed-industries/codex-acp.
//
// DISTINÇÃO IMPORTANTE: "codex-acp" vs "codex"
//
//   - "codex-acp"  → binário canônico desta Spec. Pacote npm @zed-industries/codex-acp
//     (Zed Industries). Adapter ACP nativo. Este arquivo. ADR-013 D-01.
//   - "codex"      → CLI legado da OpenAI (codexInvoker em internal/taskloop/agent.go).
//     Modo legado depreciado, será removido em 2 versões minor. Ver ADR-013.
//
// Não trocar um pelo outro. A mensagem de erro do probe também documenta essa distinção.
package specs

import "strconv"

// Constantes do runtime Codex via ACP adapter @zed-industries/codex-acp.
// Política de atualização (ADR-009 + ADR-013):
//   - CodexNpmVersion e CodexSDKVersion são constantes Go pinadas. Nunca usar @latest.
//   - CodexNpmVersion só é alterada via processo audit/ (.specs/templates/skill-upgrade-decision.md).
//   - CodexSDKVersion é mantida em sincronia com go.mod por scripts/sync-acp-sdk-version.sh.
const (
	// CodexNpmPackage é o nome do pacote npm canônico do Codex ACP adapter.
	// NÃO confundir com `codex` (CLI legacy da OpenAI). codex-acp é o adapter da Zed Industries.
	// ADR-013 D-01.
	CodexNpmPackage = "@zed-industries/codex-acp"

	// CodexNpmVersion é a versão npm pinada do @zed-industries/codex-acp.
	// Pinada conforme ADR-013 D-06: constante Go atualizada somente via audit/.
	// Validada via `npm view @zed-industries/codex-acp versions` em 2026-05-21:
	// 0.14.0 é o último stable; 0.12.0 é o mínimo para gpt-5.5 (gating do compozy).
	CodexNpmVersion = "0.14.0"

	// CodexMinNpmVersion é a versão mínima do codex-acp que suporta gpt-5.5
	// (gating documentado por compozy/internal/core/agent/registry_compat.go::codexModelRequirements).
	// Informacional; probe não valida versão runtime. ADR-013 D-06.
	CodexMinNpmVersion = "0.12.0"

	// CodexSDKVersion é a versão do coder/acp-go-sdk sincronizada com go.mod.
	// Mesma do Claude/Copilot. Não editar manualmente — use make sync-acp-sdk-version.
	CodexSDKVersion = "v0.13.0"

	// DefaultCodexModel é o modelo default quando --model não é passado.
	// Espelha compozy/internal/core/model/constants.go:15.
	DefaultCodexModel = "gpt-5.5"
)

// Codex retorna a Spec do runtime Codex via codex-acp adapter.
//
// Binário canônico: "codex-acp" (NÃO "codex" — esse é o CLI legacy da OpenAI; ADR-013 D-01).
// Fallback: npx --yes @zed-industries/codex-acp@<CodexNpmVersion>.
// FixedArgs vazio — toda configuração via BootstrapArgs em tempo de spawn (ADR-013 D-07).
// AccessModeFlag vazio — Codex passa access via -c approval_policy=..., não flag dedicada (D-07).
// CodexMaxTokens é a janela de contexto do Codex (gpt-5.5): 128 000 tokens (ADR-023).
// Valor estático versionado; sobreposto por config de projeto quando necessário.
const CodexMaxTokens = 128_000

func Codex() Spec {
	return newSpecWithBootstrap(
		"codex",
		"Codex (ACP)",
		"codex-acp",
		nil, // FixedArgs vazio — configuração via BootstrapArgs
		[]FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{"--yes", CodexNpmPackage + "@" + CodexNpmVersion},
			},
		},
		"", // AccessModeFlag vazio (ADR-013 D-07)
		CodexSDKVersion,
		CodexNpmVersion,
		CodexNpmPackage,
		codexBootstrapArgs,
		ContextWindow{MaxTokens: CodexMaxTokens},
	)
}

// codexBootstrapArgs replica compozy/internal/core/agent/registry_specs.go:247-278.
// Emite pares -c key="value" para model, reasoning, feature toggles e sandbox.
//
// Ordem garantida (espelhando compozy):
//  1. model (se não-vazio)
//  2. model_reasoning_effort (se não-vazio)
//  3. features.code_mode=false
//  4. features.code_mode_only=false
//  5. (só AccessModeFull) approval_policy, sandbox_mode, web_search
//
// strconv.Quote escapa os values de model e reasoning — proteção contra injeção (R-SEC-001).
// Os overrides de sandbox/approval/web_search usam literais com aspas já corretas.
func codexBootstrapArgs(model, reasoning string, _ []string, mode AccessMode) []string {
	args := make([]string, 0, 14)
	if model != "" {
		args = appendCodexOverrides(args, "model="+strconv.Quote(model))
	}
	if reasoning != "" {
		args = appendCodexOverrides(args, "model_reasoning_effort="+strconv.Quote(reasoning))
	}
	args = appendCodexOverrides(args,
		"features.code_mode=false",
		"features.code_mode_only=false",
	)
	if mode == AccessModeFull {
		args = appendCodexOverrides(args,
			`approval_policy="never"`,
			`sandbox_mode="danger-full-access"`,
			`web_search="live"`,
		)
	}
	return args
}

// appendCodexOverrides adiciona cada override como par "-c <override>".
// Emitir pares garante que o adapter codex-acp receba flags bem formadas.
func appendCodexOverrides(args []string, overrides ...string) []string {
	for _, o := range overrides {
		args = append(args, "-c", o)
	}
	return args
}
