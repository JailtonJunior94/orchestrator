package agents_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
)

type AgentSuite struct {
	suite.Suite
}

func TestAgentSuite(t *testing.T) {
	suite.Run(t, new(AgentSuite))
}

func (s *AgentSuite) TestScopeString() {
	scenarios := []struct {
		name  string
		scope agents.Scope
		want  string
	}{
		{name: "deve retornar escopo global", scope: agents.ScopeGlobal, want: "global"},
		{name: "deve retornar escopo workspace", scope: agents.ScopeWorkspace, want: "workspace"},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			s.Equal(scenario.want, scenario.scope.String())
		})
	}
}

func (s *AgentSuite) TestResolvedAgentZeroValue() {
	var agent agents.ResolvedAgent

	s.Empty(agent.Name)
	s.Equal(agents.ScopeGlobal, agent.Scope)
}
