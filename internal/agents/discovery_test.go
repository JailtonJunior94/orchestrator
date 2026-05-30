package agents_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type DiscoverySuite struct {
	suite.Suite
}

func TestDiscoverySuite(t *testing.T) {
	suite.Run(t, new(DiscoverySuite))
}

// validAgentMD retorna um AGENT.md minimo valido com o nome indicado.
func validAgentMD(name string) string {
	return "---\nname: " + name + "\ndescription: Agente de teste\nversion: 1.0.0\n---\n\nCorpo do agente.\n"
}

// setupFakeFS retorna um FakeFileSystem com agentes nos escopos indicados.
// globalAgents e workspaceAgents sao listas de nomes de agentes a criar.
func setupFakeFS(globalRoot, workspaceRoot string, globalAgents, workspaceAgents []string) *fs.FakeFileSystem {
	fake := fs.NewFakeFileSystem()

	for _, name := range globalAgents {
		path := globalRoot + "/.ai-harness/agents/" + name + "/AGENT.md"
		fake.Files[path] = []byte(validAgentMD(name))
	}

	for _, name := range workspaceAgents {
		path := workspaceRoot + "/.ai-harness/agents/" + name + "/AGENT.md"
		fake.Files[path] = []byte(validAgentMD(name))
	}

	return fake
}

func (s *DiscoverySuite) TestDiscoverAgents() {
	scenarios := []struct {
		name   string
		fsys   func() *fs.FakeFileSystem
		scope  agents.Scope
		root   string
		expect func(result []agents.ResolvedAgent, err error)
	}{
		{
			name:  "deve retornar vazio quando nenhum agente existe",
			fsys:  fs.NewFakeFileSystem,
			scope: agents.ScopeGlobal,
			root:  "/home/user",
			expect: func(result []agents.ResolvedAgent, err error) {
				s.NoError(err)
				s.Empty(result)
			},
		},
		{
			name: "deve listar apenas agente global",
			fsys: func() *fs.FakeFileSystem {
				return setupFakeFS("/home/user", "/workspace", []string{"agente-alpha"}, nil)
			},
			scope: agents.ScopeGlobal,
			root:  "/home/user",
			expect: func(result []agents.ResolvedAgent, err error) {
				s.NoError(err)
				if !s.Len(result, 1) {
					return
				}
				s.Equal("agente-alpha", result[0].Name)
				s.Equal(agents.ScopeGlobal, result[0].Scope)
				s.False(result[0].Path == "")
			},
		},
		{
			name: "deve listar apenas agente workspace",
			fsys: func() *fs.FakeFileSystem {
				return setupFakeFS("/home/user", "/workspace", nil, []string{"agente-beta"})
			},
			scope: agents.ScopeWorkspace,
			root:  "/workspace",
			expect: func(result []agents.ResolvedAgent, err error) {
				s.NoError(err)
				if !s.Len(result, 1) {
					return
				}
				s.Equal("agente-beta", result[0].Name)
				s.Equal(agents.ScopeWorkspace, result[0].Scope)
			},
		},
		{
			name: "deve retornar erro parcial quando nome diverge do diretorio e continuar descoberta",
			fsys: func() *fs.FakeFileSystem {
				fake := fs.NewFakeFileSystem()
				fake.Files["/home/user/.ai-harness/agents/meu-agente/AGENT.md"] = []byte("---\nname: nome-diferente\ndescription: Teste\nversion: 1.0.0\n---\n")
				fake.Files["/home/user/.ai-harness/agents/agente-valido/AGENT.md"] = []byte(validAgentMD("agente-valido"))
				return fake
			},
			scope: agents.ScopeGlobal,
			root:  "/home/user",
			expect: func(result []agents.ResolvedAgent, err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrNameDirMismatch))
				if !s.Len(result, 1) {
					return
				}
				s.Equal("agente-valido", result[0].Name)
			},
		},
		{
			name: "deve continuar descoberta quando frontmatter e invalido",
			fsys: func() *fs.FakeFileSystem {
				fake := fs.NewFakeFileSystem()
				fake.Files["/home/user/.ai-harness/agents/invalido/AGENT.md"] = []byte("---\nname: invalido\nversion: 1.0.0\n---\n")
				fake.Files["/home/user/.ai-harness/agents/valido/AGENT.md"] = []byte(validAgentMD("valido"))
				return fake
			},
			scope: agents.ScopeGlobal,
			root:  "/home/user",
			expect: func(result []agents.ResolvedAgent, err error) {
				s.Error(err)
				if !s.Len(result, 1) {
					return
				}
				s.Equal("valido", result[0].Name)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			result, err := agents.ExportDiscoverAgents(scenario.fsys(), scenario.scope, scenario.root)

			scenario.expect(result, err)
		})
	}
}

func (s *DiscoverySuite) TestMergeWithShadowing() {
	scenarios := []struct {
		name      string
		global    []agents.ResolvedAgent
		workspace []agents.ResolvedAgent
		expect    func(merged, shadowed []agents.ResolvedAgent)
	}{
		{
			name: "deve manter workspace quando ha colisao com global",
			global: []agents.ResolvedAgent{
				{Name: "agente-x", Scope: agents.ScopeGlobal},
				{Name: "agente-y", Scope: agents.ScopeGlobal},
			},
			workspace: []agents.ResolvedAgent{{Name: "agente-x", Scope: agents.ScopeWorkspace}},
			expect: func(merged, shadowed []agents.ResolvedAgent) {
				found := map[string]agents.Scope{}
				for _, agent := range merged {
					found[agent.Name] = agent.Scope
				}
				s.Equal(agents.ScopeWorkspace, found["agente-x"])
				_, ok := found["agente-y"]
				s.True(ok)
				s.Len(merged, 2)
				if !s.Len(shadowed, 1) {
					return
				}
				s.Equal("agente-x", shadowed[0].Name)
				s.Equal(agents.ScopeGlobal, shadowed[0].Scope)
			},
		},
		{
			name:      "deve manter todos agentes quando nao ha colisao",
			global:    []agents.ResolvedAgent{{Name: "agente-g", Scope: agents.ScopeGlobal}},
			workspace: []agents.ResolvedAgent{{Name: "agente-w", Scope: agents.ScopeWorkspace}},
			expect: func(merged, shadowed []agents.ResolvedAgent) {
				s.Len(merged, 2)
				s.Empty(shadowed)
			},
		},
		{
			name: "deve retornar merged em ordem lexicografica",
			global: []agents.ResolvedAgent{
				{Name: "zebra", Scope: agents.ScopeGlobal},
				{Name: "alpha", Scope: agents.ScopeGlobal},
			},
			expect: func(merged, shadowed []agents.ResolvedAgent) {
				if !s.Len(merged, 2) {
					return
				}
				s.Equal("alpha", merged[0].Name)
				s.Equal("zebra", merged[1].Name)
				s.Empty(shadowed)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			merged, shadowed := agents.ExportMergeWithShadowing(scenario.global, scenario.workspace)

			scenario.expect(merged, shadowed)
		})
	}
}
