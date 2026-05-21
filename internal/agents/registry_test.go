package agents_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// countingFakeFS e um FakeFileSystem com contador de leituras de arquivo para T-14.
type countingFakeFS struct {
	*fs.FakeFileSystem
	mu        sync.Mutex
	readCount int
}

func newCountingFakeFS() *countingFakeFS {
	return &countingFakeFS{
		FakeFileSystem: fs.NewFakeFileSystem(),
	}
}

func (c *countingFakeFS) ReadFile(path string) ([]byte, error) {
	c.mu.Lock()
	c.readCount++
	c.mu.Unlock()
	return c.FakeFileSystem.ReadFile(path)
}

func (c *countingFakeFS) getReadCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readCount
}

// --- T-14: Cache — 2 chamadas a Resolve leem disco 1 vez ---

func TestRegistry_T14_CacheDiscoChamadoUmaVez(t *testing.T) {
	t.Parallel()

	cfs := newCountingFakeFS()

	// Criar agente "foo" no workspace.
	const workdir = "/workspace"
	const home = "/home/user"
	agentPath := workdir + "/.ai-harness/agents/foo/AGENT.md"
	cfs.Files[agentPath] = []byte(validAgentMD("foo"))

	registry := agents.NewDefaultRegistry(cfs, workdir, home)

	// Primeira chamada.
	a1, err := registry.Resolve("foo")
	if err != nil {
		t.Fatalf("primeira Resolve: erro inesperado: %v", err)
	}
	if a1.Name != "foo" {
		t.Errorf("Resolve retornou agente errado: %q", a1.Name)
	}

	reads1 := cfs.getReadCount()
	if reads1 == 0 {
		t.Error("esperado pelo menos 1 leitura de disco na primeira chamada")
	}

	// Segunda chamada — nao deve ler disco novamente.
	a2, err := registry.Resolve("foo")
	if err != nil {
		t.Fatalf("segunda Resolve: erro inesperado: %v", err)
	}
	if a2.Name != "foo" {
		t.Errorf("segunda Resolve retornou agente errado: %q", a2.Name)
	}

	reads2 := cfs.getReadCount()
	if reads2 != reads1 {
		t.Errorf("T-14: disco lido mais de uma vez: %d leituras vs %d apos segunda chamada", reads1, reads2)
	}
}

// --- T-19: ErrAgentNotFound lista candidatos descobertos ---

func TestRegistry_T19_AgentNaoEncontradoListaCandidatos(t *testing.T) {
	t.Parallel()

	fake := setupFakeFS("/home/user", "/workspace",
		[]string{"agente-global-a"},
		[]string{"agente-workspace-b"},
	)

	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	_, err := registry.Resolve("agente-inexistente")

	if err == nil {
		t.Fatal("esperado erro ErrAgentNotFound, obteve nil")
	}
	if !errors.Is(err, agents.ErrAgentNotFound) {
		t.Errorf("erro esperado wrapping ErrAgentNotFound, obteve: %v", err)
	}

	// Mensagem deve listar candidatos descobertos (RF-17).
	errMsg := err.Error()
	if !strings.Contains(errMsg, "agente-global-a") && !strings.Contains(errMsg, "agente-workspace-b") {
		t.Errorf("mensagem de erro nao lista candidatos descobertos: %q", errMsg)
	}
	// Workspace prevalece: agente-global-a pode ser shadowed se mesmo nome; aqui sao diferentes.
	if !strings.Contains(errMsg, "candidatos") {
		t.Errorf("mensagem de erro deve conter 'candidatos', obteve: %q", errMsg)
	}
}

// TestRegistry_T19_SemCandidatos verifica mensagem acionavel quando nenhum agente e descoberto.
func TestRegistry_T19_SemCandidatos(t *testing.T) {
	t.Parallel()

	fake := fs.NewFakeFileSystem()
	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	_, err := registry.Resolve("foo")

	if !errors.Is(err, agents.ErrAgentNotFound) {
		t.Errorf("esperado ErrAgentNotFound, obteve: %v", err)
	}
	if !strings.Contains(err.Error(), "nenhum agente descoberto") {
		t.Errorf("mensagem esperada 'nenhum agente descoberto', obteve: %q", err.Error())
	}
}

// --- Concorrencia: 100 goroutines chamando Discover em paralelo ---

func TestRegistry_Concorrencia_100Goroutines(t *testing.T) {
	t.Parallel()

	fake := setupFakeFS("/home/user", "/workspace",
		[]string{"agente-alpha", "agente-beta"},
		[]string{"agente-gamma"},
	)

	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	const n = 100
	results := make([][]agents.ResolvedAgent, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	wg.Add(n)

	for idx := range n {
		go func() {
			defer wg.Done()
			list, err := registry.Discover(context.Background())
			results[idx] = list
			errs[idx] = err
		}()
	}

	wg.Wait()

	// Todos devem retornar o mesmo resultado sem race.
	first := results[0]
	for i, r := range results {
		if errs[i] != nil {
			t.Errorf("goroutine %d: erro inesperado: %v", i, errs[i])
		}
		if len(r) != len(first) {
			t.Errorf("goroutine %d: tamanho diferente: %d vs %d", i, len(r), len(first))
		}
	}
}

// --- Testes adicionais de Discover ---

func TestRegistry_Discover_Vazio(t *testing.T) {
	t.Parallel()

	fake := fs.NewFakeFileSystem()
	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	list, err := registry.Discover(context.Background())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("esperado catalogo vazio, obteve %d agentes", len(list))
	}
}

func TestRegistry_Discover_WorkspacePrevaleceEmColisao(t *testing.T) {
	t.Parallel()

	fake := setupFakeFS("/home/user", "/workspace",
		[]string{"agente-x"}, // global
		[]string{"agente-x"}, // workspace: mesmo nome → workspace prevalece
	)

	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	list, err := registry.Discover(context.Background())
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if len(list) != 1 {
		t.Fatalf("esperado 1 agente apos shadowing, obteve %d", len(list))
	}
	if list[0].Scope != agents.ScopeWorkspace {
		t.Errorf("agente deve ter ScopeWorkspace apos shadowing, obteve %v", list[0].Scope)
	}
}

func TestRegistry_Resolve_WorkspacePrevaleceEmColisao(t *testing.T) {
	t.Parallel()

	fake := setupFakeFS("/home/user", "/workspace",
		[]string{"agente-x"}, // global
		[]string{"agente-x"}, // workspace: prevalece
	)

	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	a, err := registry.Resolve("agente-x")
	if err != nil {
		t.Fatalf("Resolve: erro inesperado: %v", err)
	}
	if a.Scope != agents.ScopeWorkspace {
		t.Errorf("Resolve deve retornar workspace agent em colisao, obteve %v", a.Scope)
	}
}
