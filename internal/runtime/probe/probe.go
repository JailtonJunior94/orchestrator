// Package probe implementa a fase EnsureAvailable: resolve o launcher do agente
// ACP seguindo uma cadeia ordenada de candidatos — (a) binário canônico no PATH;
// (b) cada FallbackLauncher da spec em ordem, materializado como BinaryLauncher
// genérico com seus FixedArgs literais; (c) falha com mensagem contendo três
// remédios. Conforme ADR-017: o fallback npx é apenas um caso particular de
// FallbackLauncher{Command:"npx", FixedArgs:[...]}, sem tratamento especial.
// Cache em memória por processo via sync.Map + sync.Once para evitar
// re-probing por task na mesma invocação CLI.
package probe

import (
	"context"
	"fmt"
	"sync"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// _adrByID mapeia spec.ID para o path do ADR correspondente.
// Decisão D-09 (ADR-012): mapping vive no package probe, não em specs/.
// Evita acoplamento entre specs/ e a estrutura de docs ADRs.
// Para IDs desconhecidos, use o fallback ".specs/adr/" documentado em resolve().
var _adrByID = map[string]string{
	"claude":  ".specs/adr/009-acp-protocol-adoption.md",
	"codex":   ".specs/adr/013-codex-cli-acp-native.md",
	"copilot": ".specs/adr/012-copilot-cli-acp-native.md",
	"gemini":  ".specs/adr/015-gemini-cli-acp-native.md",
}

// formatLauncherUnavailable formata a mensagem de erro quando nenhum launcher está disponível.
// Parametrizado pelo Spec recebido e pelo path do ADR associado ao spec.ID.
// Conforme RF-03 e RF-05 (mensagem contém binário, npm@pin, fallback legacy, referência ADR).
func (c *Catalog) formatLauncherUnavailable(spec specs.Spec, adrPath string) string {
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

// _cache armazena os resultados de probe por spec ID para evitar re-probing.
var _cache sync.Map // map[string]*cacheEntry

// EnsureAvailable resolve o launcher do agente ACP seguindo RF-03.
// Primeira chamada para cada spec.ID resolve e armazena o resultado em cache;
// chamadas subsequentes reutilizam o resultado sem invocar LookPath novamente.
//
// Ordem de resolução:
//  1. spec.Command no PATH via look.LookPath
//  2. Para cada spec.Fallbacks: verifica se o comando (ex: npx) está no PATH
//  3. Falha com ErrLauncherUnavailable + mensagem com três remédios
func (c *Catalog) EnsureAvailable(ctx context.Context, spec specs.Spec, look LookPather) (specs.Launcher, error) {
	if err := ctx.Err(); err != nil {
		return specs.Launcher{}, err
	}

	entry, _ := _cache.LoadOrStore(spec.ID, &cacheEntry{})
	e, ok := entry.(*cacheEntry)
	if !ok {
		return specs.Launcher{}, fmt.Errorf("cache de launcher corrompido para %q: tipo inesperado %T", spec.ID, entry)
	}

	e.once.Do(func() {
		e.launcher, e.err = NewCatalog().resolve(spec, look)
	})

	return e.launcher, e.err
}

// resolve executa a resolução efetiva do launcher (chamado apenas uma vez por spec via once).
// ADR-017: cada FallbackLauncher é tratado como launcher de comando genérico —
// materializado via NewBinaryLauncher(path, fb.FixedArgs...) sem semântica npx-only.
func (c *Catalog) resolve(spec specs.Spec, look LookPather) (specs.Launcher, error) {
	// Passo 1: binário canônico no PATH.
	// FixedArgs do Spec (ex: ["--acp"] para Copilot/Gemini) são passados ao BinaryLauncher para
	// garantir que o binário seja invocado com os flags corretos.
	if path, err := look.LookPath(spec.Command); err == nil {
		return specs.NewBinaryLauncher(path, spec.FixedArgs...), nil
	}

	// Passo 2: tentar cada fallback em ordem (ADR-017 D-2).
	// O primeiro cujo Command exista no PATH vence; FixedArgs são preservados literalmente.
	// npx é apenas um caso particular de FallbackLauncher — sem tratamento especial.
	for _, fb := range spec.Fallbacks {
		if path, err := look.LookPath(fb.Command); err == nil {
			return specs.NewBinaryLauncher(path, fb.FixedArgs...), nil
		}
	}

	// Passo 3: falhar com mensagem com três remédios (RF-03, RF-05).
	// Lookup do ADR pelo spec.ID; fallback para path raiz quando ID desconhecido.
	adrPath, ok := _adrByID[spec.ID]
	if !ok {
		adrPath = ".specs/adr/"
	}
	msg := NewCatalog().formatLauncherUnavailable(spec, adrPath)
	return specs.Launcher{}, fmt.Errorf("%s: %w", msg, ErrLauncherUnavailable)
}

// ResetCache limpa o cache interno. Utilizado apenas em testes.
// Exportado para permitir reset entre testes que precisam testar o caching.
func (c *Catalog) ResetCache() {
	_cache.Range(func(key, _ any) bool {
		_cache.Delete(key)
		return true
	})
}
