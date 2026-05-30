package agents_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type RegistrySuite struct {
	suite.Suite
}

func TestRegistrySuite(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}

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

func (s *RegistrySuite) TestResolveCacheDiscoChamadoUmaVez() {
	cfs := newCountingFakeFS()

	const workdir = "/workspace"
	const home = "/home/user"
	agentPath := workdir + "/.ai-harness/agents/foo/AGENT.md"
	cfs.Files[agentPath] = []byte(validAgentMD("foo"))

	registry := agents.NewDefaultRegistry(cfs, workdir, home)

	a1, err := registry.Resolve("foo")
	s.NoError(err)
	s.Equal("foo", a1.Name)

	reads1 := cfs.getReadCount()
	s.True(reads1 > 0)

	a2, err := registry.Resolve("foo")
	s.NoError(err)
	s.Equal("foo", a2.Name)

	reads2 := cfs.getReadCount()
	s.Equal(reads1, reads2)
}

func (s *RegistrySuite) TestResolveAgentNaoEncontrado() {
	scenarios := []struct {
		name   string
		fsys   func() *fs.FakeFileSystem
		expect func(err error)
	}{
		{
			name: "deve listar candidatos descobertos",
			fsys: func() *fs.FakeFileSystem {
				return setupFakeFS("/home/user", "/workspace", []string{"agente-global-a"}, []string{"agente-workspace-b"})
			},
			expect: func(err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrAgentNotFound))
				errMsg := err.Error()
				s.True(strings.Contains(errMsg, "agente-global-a") || strings.Contains(errMsg, "agente-workspace-b"))
				s.True(strings.Contains(errMsg, "candidatos"))
			},
		},
		{
			name: "deve informar quando nenhum candidato foi descoberto",
			fsys: fs.NewFakeFileSystem,
			expect: func(err error) {
				s.True(errors.Is(err, agents.ErrAgentNotFound))
				s.True(strings.Contains(err.Error(), "nenhum agente descoberto"))
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			registry := agents.NewDefaultRegistry(scenario.fsys(), "/workspace", "/home/user")

			_, err := registry.Resolve("agente-inexistente")

			scenario.expect(err)
		})
	}
}

func (s *RegistrySuite) TestDiscoverConcorrencia() {
	fake := setupFakeFS("/home/user", "/workspace", []string{"agente-alpha", "agente-beta"}, []string{"agente-gamma"})
	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	const totalGoroutines = 100
	results := make([][]agents.ResolvedAgent, totalGoroutines)
	errs := make([]error, totalGoroutines)

	var wg sync.WaitGroup
	wg.Add(totalGoroutines)

	for idx := range totalGoroutines {
		idx := idx
		go func() {
			defer wg.Done()
			list, err := registry.Discover(context.Background())
			results[idx] = list
			errs[idx] = err
		}()
	}

	wg.Wait()

	first := results[0]
	for idx, result := range results {
		s.NoError(errs[idx])
		s.Len(result, len(first))
	}
}

func (s *RegistrySuite) TestDiscover() {
	scenarios := []struct {
		name   string
		fsys   func() *fs.FakeFileSystem
		expect func(list []agents.ResolvedAgent, err error)
	}{
		{
			name: "deve retornar catalogo vazio",
			fsys: fs.NewFakeFileSystem,
			expect: func(list []agents.ResolvedAgent, err error) {
				s.NoError(err)
				s.Empty(list)
			},
		},
		{
			name: "deve manter workspace quando ha colisao no discover",
			fsys: func() *fs.FakeFileSystem {
				return setupFakeFS("/home/user", "/workspace", []string{"agente-x"}, []string{"agente-x"})
			},
			expect: func(list []agents.ResolvedAgent, err error) {
				s.NoError(err)
				if !s.Len(list, 1) {
					return
				}
				s.Equal(agents.ScopeWorkspace, list[0].Scope)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			registry := agents.NewDefaultRegistry(scenario.fsys(), "/workspace", "/home/user")

			list, err := registry.Discover(context.Background())

			scenario.expect(list, err)
		})
	}
}

func (s *RegistrySuite) TestResolveWorkspacePrevaleceEmColisao() {
	fake := setupFakeFS("/home/user", "/workspace", []string{"agente-x"}, []string{"agente-x"})
	registry := agents.NewDefaultRegistry(fake, "/workspace", "/home/user")

	agent, err := registry.Resolve("agente-x")

	s.NoError(err)
	s.Equal(agents.ScopeWorkspace, agent.Scope)
}
