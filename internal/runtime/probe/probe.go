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

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// errMsgTemplate é a mensagem de erro para quando nenhum launcher está disponível.
// Literal exato conforme RF-03.
const errMsgTemplate = "claude-agent-acp não encontrado. Install claude-agent-acp; OR install %s@%s via npm; OR use --runtime=legacy. Veja tasks/adr/009-acp-protocol-adoption.md"

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
		e.launcher, e.err = resolve(ctx, spec, look)
	})

	return e.launcher, e.err
}

// resolve executa a resolução efetiva do launcher (chamado apenas uma vez por spec via once).
func resolve(ctx context.Context, spec specs.Spec, look LookPather) (specs.Launcher, error) {
	// Passo 1: binário canônico no PATH.
	if path, err := look.LookPath(spec.Command); err == nil {
		return specs.NewBinaryLauncher(path), nil
	}

	// Passo 2: tentar cada fallback (verifica se o comando do fallback está no PATH).
	for _, fb := range spec.Fallbacks {
		if _, err := look.LookPath(fb.Command); err == nil {
			// fb.Command (ex: npx) está disponível; usar os FixedArgs do fallback.
			return specs.NewNpxLauncher(extractPackage(fb), extractVersion(fb)), nil
		}
	}

	// Passo 3: falhar com mensagem com três remédios (RF-03).
	npmPkg := specs.ClaudeNpmPackage
	npmVer := specs.ClaudeNpmVersion
	if len(spec.Fallbacks) > 0 && len(spec.Fallbacks[0].FixedArgs) > 1 {
		// Extrai do FixedArgs o pacote e versão.
		arg := spec.Fallbacks[0].FixedArgs[1]
		if at := findLastAt(arg); at >= 0 {
			npmPkg = arg[:at]
			npmVer = arg[at+1:]
		}
	}

	msg := fmt.Sprintf(errMsgTemplate, npmPkg, npmVer)
	return specs.Launcher{}, fmt.Errorf("%s: %w", msg, airuntime.ErrLauncherUnavailable)
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
