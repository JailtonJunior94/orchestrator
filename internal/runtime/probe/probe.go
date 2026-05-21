// Package probe implementa a fase EnsureAvailable do RF-03:
// resolve o launcher do claude-agent-acp na ordem (a) binário canônico no PATH;
// (b) npx --yes @agentclientprotocol/claude-agent-acp@<VER>;
// (c) falha com mensagem contendo três remédios.
// Cache em memória por processo via sync.Map + sync.OnceValues para evitar
// re-probing por task na mesma invocação CLI.
package probe

import (
	"context"
	"fmt"
	"sync"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// adrByID mapeia spec.ID para o path do ADR correspondente.
// Decisão D-09 (ADR-012): mapping vive no package probe, não em specs/.
// Evita acoplamento entre specs/ e a estrutura de docs ADRs.
// Para IDs desconhecidos, use o fallback "tasks/adr/" documentado em resolve().
var adrByID = map[string]string{
	"claude":  "tasks/adr/009-acp-protocol-adoption.md",
	"codex":   "tasks/adr/013-codex-cli-acp-native.md",
	"copilot": "tasks/adr/012-copilot-cli-acp-native.md",
}

// formatLauncherUnavailable formata a mensagem de erro quando nenhum launcher está disponível.
// Parametrizado pelo Spec recebido e pelo path do ADR associado ao spec.ID.
// Conforme RF-03 e RF-05 (mensagem contém binário, npm@pin, fallback legacy, referência ADR).
func formatLauncherUnavailable(spec specs.Spec, adrPath string) string {
	return fmt.Sprintf(
		"%s não encontrado. Install %s; OR install %s@%s via npm; OR use --runtime=legacy. Veja %s",
		spec.Command, spec.Command, spec.NPMPackage(), spec.NPMVersion(), adrPath,
	)
}

// cacheEntry armazena o resultado do probe para uma spec.
type cacheEntry struct {
	once     sync.Once
	launcher specs.Launcher
	err      error
}

// cache armazena os resultados de probe por spec ID para evitar re-probing.
var cache sync.Map // map[string]*cacheEntry

// EnsureAvailable resolve o launcher do agente ACP seguindo RF-03.
// Primeira chamada para cada spec.ID resolve e armazena o resultado em cache;
// chamadas subsequentes reutilizam o resultado sem invocar LookPath novamente.
//
// Ordem de resolução:
//  1. spec.Command no PATH via look.LookPath
//  2. Para cada spec.Fallbacks: verifica se o comando (ex: npx) está no PATH
//  3. Falha com ErrLauncherUnavailable + mensagem com três remédios
func EnsureAvailable(ctx context.Context, spec specs.Spec, look LookPather) (specs.Launcher, error) {
	if err := ctx.Err(); err != nil {
		return specs.Launcher{}, err
	}

	entry, _ := cache.LoadOrStore(spec.ID, &cacheEntry{})
	e := entry.(*cacheEntry)

	e.once.Do(func() {
		e.launcher, e.err = resolve(spec, look)
	})

	return e.launcher, e.err
}

// resolve executa a resolução efetiva do launcher (chamado apenas uma vez por spec via once).
func resolve(spec specs.Spec, look LookPather) (specs.Launcher, error) {
	// Passo 1: binário canônico no PATH.
	// FixedArgs do Spec (ex: ["--acp"] para Copilot) são passados ao BinaryLauncher para
	// garantir que o binário seja invocado com os flags corretos (bug fix: sem FixedArgs,
	// copilot seria iniciado sem --acp e entraria em modo legado em vez de ACP server).
	if path, err := look.LookPath(spec.Command); err == nil {
		return specs.NewBinaryLauncher(path, spec.FixedArgs...), nil
	}

	// Passo 2: tentar cada fallback (verifica se o comando do fallback está no PATH).
	for _, fb := range spec.Fallbacks {
		if _, err := look.LookPath(fb.Command); err == nil {
			// fb.Command (ex: npx) está disponível; usar os FixedArgs do fallback.
			return specs.NewNpxLauncher(extractPackage(fb), extractVersion(fb)), nil
		}
	}

	// Passo 3: falhar com mensagem com três remédios (RF-03, RF-05).
	// Lookup do ADR pelo spec.ID; fallback para path raiz quando ID desconhecido.
	adrPath, ok := adrByID[spec.ID]
	if !ok {
		adrPath = "tasks/adr/"
	}
	msg := formatLauncherUnavailable(spec, adrPath)
	return specs.Launcher{}, fmt.Errorf("%s: %w", msg, ErrLauncherUnavailable)
}

// extractPackage extrai o nome do pacote npm do FallbackLauncher.
// Exemplo: FixedArgs = ["--yes", "@agentclientprotocol/claude-agent-acp@0.1.0"]
// retorna "@agentclientprotocol/claude-agent-acp".
func extractPackage(fb specs.FallbackLauncher) string {
	if len(fb.FixedArgs) < 2 {
		return ""
	}
	arg := fb.FixedArgs[1]
	at := findLastAt(arg)
	if at < 0 {
		return arg
	}
	return arg[:at]
}

// extractVersion extrai a versão npm do FallbackLauncher.
func extractVersion(fb specs.FallbackLauncher) string {
	if len(fb.FixedArgs) < 2 {
		return ""
	}
	arg := fb.FixedArgs[1]
	at := findLastAt(arg)
	if at < 0 {
		return ""
	}
	return arg[at+1:]
}

// findLastAt encontra a posição do último '@' em uma string.
// Necessário para separar "@scope/pkg@version" corretamente.
func findLastAt(s string) int {
	// Scoped packages começam com @, então buscamos o último @.
	last := -1
	for i := 1; i < len(s); i++ {
		if s[i] == '@' {
			last = i
		}
	}
	return last
}

// ResetCache limpa o cache interno. Utilizado apenas em testes.
// Exportado para permitir reset entre testes que precisam testar o caching.
func ResetCache() {
	cache.Range(func(key, _ any) bool {
		cache.Delete(key)
		return true
	})
}
