package agents

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PromptSuite struct {
	suite.Suite
}

func TestPromptSuite(t *testing.T) {
	suite.Run(t, new(PromptSuite))
}

func makeAgent(name, description, version string) ResolvedAgent {
	return ResolvedAgent{
		Name: name,
		Metadata: Metadata{
			Description: description,
			Version:     version,
		},
	}
}

func (s *PromptSuite) TestBuildAgentBlocks() {
	scenarios := []struct {
		name    string
		active  *ResolvedAgent
		catalog []ResolvedAgent
		expect  func(meta, catalog string)
	}{
		{
			name: "deve ordenar catalogo e marcar agente ativo",
			active: &ResolvedAgent{
				Name: "beta-agent",
				Metadata: Metadata{
					Description: "Beta agent description",
					Version:     "1.0.0",
				},
				Runtime: RuntimeDefaults{
					IDE:   "claude",
					Model: "claude-opus-4-7",
				},
			},
			catalog: []ResolvedAgent{
				makeAgent("zeta-agent", "Zeta agent", "1.0.0"),
				makeAgent("alpha-agent", "Alpha agent", "2.0.0"),
				makeAgent("beta-agent", "Beta agent description", "1.0.0"),
			},
			expect: func(meta, catalog string) {
				s.True(strings.Contains(meta, "### Agente Ativo"))
				s.True(strings.Contains(meta, "beta-agent"))
				s.True(strings.Contains(meta, "Beta agent description"))
				s.True(strings.Contains(meta, "1.0.0"))
				s.True(strings.Contains(meta, "claude"))
				s.True(strings.Contains(catalog, "### Agentes Disponiveis"))
				s.True(strings.Contains(catalog, "`beta-agent` [active]"))

				alphaPos := strings.Index(catalog, "alpha-agent")
				betaPos := strings.Index(catalog, "beta-agent")
				zetaPos := strings.Index(catalog, "zeta-agent")
				s.True(alphaPos >= 0)
				s.True(betaPos >= 0)
				s.True(zetaPos >= 0)
				s.True(alphaPos < betaPos)
				s.True(betaPos < zetaPos)
			},
		},
		{
			name:   "deve truncar catalogo em duzentas entradas",
			active: ptrAgent(makeAgent("agent-000", "Active", "1.0.0")),
			catalog: func() []ResolvedAgent {
				catalog := make([]ResolvedAgent, 250)
				for i := range catalog {
					name := fmt.Sprintf("agent-%03d", i)
					catalog[i] = makeAgent(name, "desc", "1.0.0")
				}
				return catalog
			}(),
			expect: func(meta, catalog string) {
				s.Equal(_catalogLimit, strings.Count(catalog, "\n- `"))
			},
		},
		{
			name:    "deve aceitar agente ativo nil sem panic",
			active:  nil,
			catalog: []ResolvedAgent{makeAgent("alpha-agent", "Alpha", "1.0.0")},
			expect: func(meta, catalog string) {
				s.True(strings.Contains(catalog, "alpha-agent"))
			},
		},
		{
			name:    "deve renderizar headers quando catalogo esta vazio",
			active:  ptrAgent(makeAgent("my-agent", "My agent", "1.0.0")),
			catalog: []ResolvedAgent{},
			expect: func(meta, catalog string) {
				s.True(strings.Contains(meta, "### Agente Ativo"))
				s.True(strings.Contains(catalog, "### Agentes Disponiveis"))
				s.False(strings.Contains(catalog, "`"))
			},
		},
		{
			name: "deve omitir campos de runtime vazios",
			active: &ResolvedAgent{
				Name: "partial-agent",
				Metadata: Metadata{
					Description: "Partial",
					Version:     "1.0.0",
				},
				Runtime: RuntimeDefaults{IDE: "claude"},
			},
			catalog: nil,
			expect: func(meta, catalog string) {
				s.True(strings.Contains(meta, "claude"))
			},
		},
		{
			name: "deve exibir hifen quando runtime esta vazio",
			active: &ResolvedAgent{
				Name: "empty-rt",
				Metadata: Metadata{
					Description: "No runtime",
					Version:     "1.0.0",
				},
			},
			catalog: nil,
			expect: func(meta, catalog string) {
				s.True(strings.Contains(meta, "- **Runtime**: -"))
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			meta, catalog := NewCatalog().BuildAgentBlocks(scenario.active, scenario.catalog)

			scenario.expect(meta, catalog)
		})
	}
}

func ptrAgent(agent ResolvedAgent) *ResolvedAgent {
	return &agent
}
