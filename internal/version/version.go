package version

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// Set via ldflags at build time.
//
// Em producao, ldflags grava Version uma unica vez antes de main e nao
// concorre com leituras. Em testes paralelos, leitores chamam Get() (atomic
// load, lock-free) e escritores usam SetForTest (serializados via _writerMu).
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"

	_versionAtomic atomic.Pointer[string]
	_writerMu      sync.Mutex
)

// Provider resolve a versao do binario.
type Provider struct{}

// NewProvider cria um Provider stateless.
func NewProvider() *Provider {
	return &Provider{}
}

// Get retorna o valor corrente de Version via atomic load.
// Lock-free; seguro para uso concorrente com SetForTest.
func (p *Provider) Get() string {
	if current := _versionAtomic.Load(); current != nil {
		return *current
	}
	return Version
}

// SetForTest substitui o valor lido por Get pelo valor informado e
// retorna uma funcao de restauracao a ser registrada via t.Cleanup.
//
// Serializa multiplos chamadores via writerMu (escritores nao se
// sobrepoem), mas nao bloqueia leitores em Get — leituras permanecem
// lock-free via atomic. Reentrante a partir do mesmo teste apenas se
// restore for chamado antes de novo SetForTest.
//
// Destinada exclusivamente a testes.
func (p *Provider) SetForTest(value string) (restore func()) {
	_writerMu.Lock()
	prev := p.Get()
	_versionAtomic.Store(&value)
	return func() {
		_versionAtomic.Store(&prev)
		_writerMu.Unlock()
	}
}

// ReadVersionFile le o arquivo VERSION de um diretorio e retorna a versao.
// Retorna "unknown" se o arquivo nao existir ou nao puder ser lido.
func (p *Provider) ReadVersionFile(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "VERSION"))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

// Resolve retorna a versao do binario. Prioridade:
//  1. Versao injetada via ldflags (releases via GoReleaser)
//  2. Arquivo VERSION no diretorio informado com sufixo "-dev" (builds locais)
//  3. "dev" como fallback final
func (p *Provider) Resolve(dir string) string {
	if v := p.Get(); v != "dev" {
		return v
	}
	if v := p.ReadVersionFile(dir); v != "unknown" {
		return v + "-dev"
	}
	return "dev"
}

// ResolveFromExecutable localiza o VERSION file adjacente ao binario,
// resolvendo symlinks antes de extrair o diretorio.
// Fallback chain: ldflags > VERSION adjacente ao executavel resolvido > "dev"
func (p *Provider) ResolveFromExecutable() string {
	if v := p.Get(); v != "dev" {
		return v // ldflags injetado pelo GoReleaser tem prioridade maxima
	}
	exe, err := os.Executable()
	if err != nil {
		return "dev"
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "dev"
	}
	dir := filepath.Dir(resolved)
	if v := p.ReadVersionFile(dir); v != "unknown" {
		return v + "-dev"
	}
	return "dev"
}
