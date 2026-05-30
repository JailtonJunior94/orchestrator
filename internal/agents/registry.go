package agents

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// Registry descreve o contrato de descoberta e resolucao de agentes declarativos.
// Cache valido pelo tempo de vida da instancia; sem TTL; sem reset global (D-04).
type Registry interface {
	// Discover retorna o catalogo completo de agentes (global + workspace, sem shadowed).
	Discover(ctx context.Context) ([]ResolvedAgent, error)
	// Resolve retorna um unico agente pelo nome; workspace prevalece em colisao.
	// Retorna ErrAgentNotFound com mensagem listando candidatos descobertos (RF-17).
	Resolve(name string) (ResolvedAgent, error)
}

// defaultRegistry e a implementacao default de Registry com cache stateful por instancia (D-04).
// Nao ha cache global: instanciar nova Registry em vez de chamar ResetCache.
type defaultRegistry struct {
	fsys    fs.FileSystem
	workdir string
	home    string

	once      sync.Once
	cached    []ResolvedAgent
	cachedErr error
}

var _ Registry = (*defaultRegistry)(nil)

// NewDefaultRegistry cria um Registry com cache proprio.
// Cache valido pelo tempo de vida da instancia — testes instanciam nova registry entre cenarios.
func NewDefaultRegistry(fsys fs.FileSystem, workdir, home string) Registry {
	return &defaultRegistry{
		fsys:    fsys,
		workdir: workdir,
		home:    home,
	}
}

// load popula o cache via sync.Once garantindo leitura de disco apenas uma vez,
// mesmo sob chamadas paralelas (T-14, criterio de concorrencia).
func (r *defaultRegistry) load() {
	r.once.Do(func() {
		global, errG := NewCatalog().discoverAgents(r.fsys, ScopeGlobal, r.home)
		workspace, errW := NewCatalog().discoverAgents(r.fsys, ScopeWorkspace, r.workdir)

		// Priorizar erro de workspace sobre global; ambos registrados no campo de erro.
		if errG != nil || errW != nil {
			var parts []string
			if errG != nil {
				parts = append(parts, errG.Error())
			}
			if errW != nil {
				parts = append(parts, errW.Error())
			}
			r.cachedErr = fmt.Errorf("descoberta parcial: %s", strings.Join(parts, "; "))
		}

		merged, _ := NewCatalog().mergeWithShadowing(global, workspace)
		r.cached = merged
	})
}

// Discover retorna o catalogo completo de agentes (global + workspace, sem shadowed).
// Seguro para chamadas paralelas — sync.Once garante idempotencia (T-14).
func (r *defaultRegistry) Discover(_ context.Context) ([]ResolvedAgent, error) {
	r.load()
	if r.cachedErr != nil {
		// Retornar lista parcial junto com o erro para robustez.
		return r.cached, r.cachedErr
	}
	return r.cached, nil
}

// Resolve retorna um agente pelo nome consumindo o cache.
// Retorna ErrAgentNotFound com mensagem acionavel listando candidatos (RF-17).
func (r *defaultRegistry) Resolve(name string) (ResolvedAgent, error) {
	r.load()

	for _, a := range r.cached {
		if a.Name == name {
			return a, nil
		}
	}

	// Construir mensagem acionavel com candidatos descobertos (RF-17).
	candidates := make([]string, 0, len(r.cached))
	for _, a := range r.cached {
		candidates = append(candidates, a.Name)
	}

	if len(candidates) == 0 {
		return ResolvedAgent{}, fmt.Errorf("%w: %q — nenhum agente descoberto", ErrAgentNotFound, name)
	}

	return ResolvedAgent{}, fmt.Errorf("%w: %q — candidatos descobertos: [%s]", ErrAgentNotFound, name, strings.Join(candidates, ", "))
}
