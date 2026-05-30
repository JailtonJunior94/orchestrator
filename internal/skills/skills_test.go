package skills

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SkillsSuite struct {
	suite.Suite
}

func TestSkillsSuite(t *testing.T) {
	suite.Run(t, new(SkillsSuite))
}

func (s *SkillsSuite) TestParseTool() {
	scenarios := []struct {
		name   string
		input  string
		want   Tool
		wantOK bool
	}{
		{name: "deve aceitar claude", input: "claude", want: ToolClaude, wantOK: true},
		{name: "deve aceitar gemini", input: "gemini", want: ToolGemini, wantOK: true},
		{name: "deve aceitar codex", input: "codex", want: ToolCodex, wantOK: true},
		{name: "deve aceitar copilot", input: "copilot", want: ToolCopilot, wantOK: true},
		{name: "deve rejeitar invalido", input: "invalid", wantOK: false},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			tool, ok := NewCatalog().ParseTool(scenario.input)
			s.Equal(scenario.wantOK, ok, "NewCatalog().ParseTool(%q) ok", scenario.input)
			if scenario.wantOK {
				s.Equal(string(scenario.want), string(tool), "NewCatalog().ParseTool(%q)", scenario.input)
			}
		})
	}
}

func (s *SkillsSuite) TestParseLang() {
	scenarios := []struct {
		name   string
		input  string
		want   Lang
		wantOK bool
	}{
		{name: "deve aceitar go", input: "go", want: LangGo, wantOK: true},
		{name: "deve aceitar node", input: "node", want: LangNode, wantOK: true},
		{name: "deve aceitar python", input: "python", want: LangPython, wantOK: true},
		{name: "deve rejeitar rust", input: "rust", wantOK: false},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			lang, ok := NewCatalog().ParseLang(scenario.input)
			s.Equal(scenario.wantOK, ok, "NewCatalog().ParseLang(%q) ok", scenario.input)
			if scenario.wantOK {
				s.Equal(string(scenario.want), string(lang), "NewCatalog().ParseLang(%q)", scenario.input)
			}
		})
	}
}

func (s *SkillsSuite) TestLangSkills() {
	scenarios := []struct {
		name  string
		langs []Lang
		want  []string
	}{
		{name: "deve mapear go para duas skills", langs: []Lang{LangGo}, want: []string{"go-implementation", "object-calisthenics-go"}},
		{name: "deve mapear node e python para duas skills", langs: []Lang{LangNode, LangPython}, want: []string{"node-implementation", "python-implementation"}},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			got := NewCatalog().LangSkills(scenario.langs)
			s.Equal(scenario.want, got)
		})
	}
}

func (s *SkillsSuite) TestAllSkills() {
	all := NewCatalog().AllSkills([]Lang{LangGo})
	want := len(BaseSkills) + len(ComplementarySkills) + 2
	s.Len(all, want, "NewCatalog().AllSkills(go)")
}

func (s *SkillsSuite) TestBaseSkillsIncludesExecuteAllTasks() {
	s.Contains(BaseSkills, "execute-all-tasks", "BaseSkills deve incluir execute-all-tasks para instalacao em novos projetos")
}

func (s *SkillsSuite) TestComplementarySkills() {
	s.Len(ComplementarySkills, 11, "ComplementarySkills count")
}
