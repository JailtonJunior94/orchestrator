package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type BudgetSuite struct {
	suite.Suite
}

func TestBudgetSuite(t *testing.T) {
	suite.Run(t, new(BudgetSuite))
}

func (s *BudgetSuite) TestToolBudgetsDefined() {
	scenarios := []struct {
		name string
		tool string
	}{
		{name: "deve definir budget para claude", tool: "claude"},
		{name: "deve definir budget para gemini", tool: "gemini"},
		{name: "deve definir budget para codex", tool: "codex"},
		{name: "deve definir budget para copilot", tool: "copilot"},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			_, ok := ToolBudgets[scenario.tool]
			s.True(ok, "ToolBudgets: budget nao definido para %q", scenario.tool)
		})
	}
}

func (s *BudgetSuite) TestCheckBudget() {
	scenarios := []struct {
		name    string
		content string
		tool    string
		expect  func(tokens, limit int, ok bool)
	}{
		{
			name:    "deve aceitar conteudo dentro do limite",
			content: strings.Repeat("a", 100),
			tool:    "codex",
			expect: func(tokens, limit int, ok bool) {
				s.True(ok, "conteudo pequeno deve estar dentro do budget codex (%d tokens, limite %d)", tokens, limit)
			},
		},
		{
			name:    "deve rejeitar conteudo acima do limite",
			content: strings.Repeat("palavra ", 8000),
			tool:    "copilot",
			expect: func(tokens, limit int, ok bool) {
				s.False(ok, "conteudo inflado deve exceder budget copilot (%d tokens, limite %d)", tokens, limit)
				s.True(tokens > limit, "tokens (%d) devem ser maiores que o limite (%d)", tokens, limit)
			},
		},
		{
			name:    "deve aceitar ferramenta desconhecida",
			content: strings.Repeat("x", 1000000),
			tool:    "ferramenta-desconhecida",
			expect: func(tokens, limit int, ok bool) {
				s.True(ok, "ferramenta desconhecida deve sempre retornar ok=true")
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			tokens, limit, ok := NewCatalog().CheckBudget(scenario.content, scenario.tool)

			scenario.expect(tokens, limit, ok)
		})
	}
}

func (s *BudgetSuite) TestGeneratedGovernanceWithinBudget() {
	scenarios := []struct {
		name string
		tool skills.Tool
		key  string
	}{
		{name: "deve manter governanca claude dentro do budget", tool: skills.ToolClaude, key: "claude"},
		{name: "deve manter governanca gemini dentro do budget", tool: skills.ToolGemini, key: "gemini"},
		{name: "deve manter governanca codex dentro do budget", tool: skills.ToolCodex, key: "codex"},
		{name: "deve manter governanca copilot dentro do budget", tool: skills.ToolCopilot, key: "copilot"},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			ffs.Dirs["/project"] = true
			ffs.Dirs["/source"] = true
			g := contextgen.NewGenerator(ffs, output.New(false))

			err := g.Generate("/source", "/project", []skills.Tool{scenario.tool}, nil, "full", false)
			s.NoError(err)

			agentsData := string(ffs.Files["/project/AGENTS.md"])
			tokens, limit, ok := NewCatalog().CheckBudget(agentsData, "claude")
			s.True(ok, "AGENTS.md (%d tokens) excede budget claude (%d)", tokens, limit)
		})
	}
}
