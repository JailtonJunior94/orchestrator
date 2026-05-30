package metrics

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type FlowSuite struct {
	suite.Suite
}

func TestFlowSuite(t *testing.T) {
	suite.Run(t, new(FlowSuite))
}

// repeatedBytes retorna um slice de n bytes para simular artefatos de tamanho controlado.
func repeatedBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return b
}

func (s *FlowSuite) TestMeasureFlowCompactVsStandard() {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/repo/AGENTS.compact.md"] = repeatedBytes(5000)
	ffs.Files["/repo/AGENTS.standard.md"] = repeatedBytes(10000)
	ffs.Files["/repo/.agents/skills/agent-governance/SKILL.md"] = repeatedBytes(3000)
	ffs.Files["/repo/.agents/skills/execute-task/SKILL.md"] = repeatedBytes(3000)

	svc := NewService(ffs, silentPrinter(), nil)

	compactArtifacts := []string{
		"/repo/AGENTS.compact.md",
		"/repo/.agents/skills/agent-governance/SKILL.md",
		"/repo/.agents/skills/execute-task/SKILL.md",
	}
	standardArtifacts := []string{
		"/repo/AGENTS.standard.md",
		"/repo/.agents/skills/agent-governance/SKILL.md",
		"/repo/.agents/skills/execute-task/SKILL.md",
	}

	compact := svc.MeasureFlow("claude", GovernanceProfileCompact, SkillProfileLean, FlowExecution, compactArtifacts)
	standard := svc.MeasureFlow("claude", GovernanceProfileStandard, SkillProfileFull, FlowExecution, standardArtifacts)

	s.True(compact.TokensLimit < standard.TokensLimit, "compact.TokensLimit (%d) deve ser menor que standard.TokensLimit (%d)", compact.TokensLimit, standard.TokensLimit)
	s.True(compact.TokensEst < standard.TokensEst, "compact.TokensEst (%d) deve ser menor que standard.TokensEst (%d)", compact.TokensEst, standard.TokensEst)
}

func (s *FlowSuite) TestFlowBudgetProfiles() {
	scenarios := []struct {
		name   string
		tool   string
		gov    GovernanceProfile
		skill  SkillProfile
		flow   FlowKind
		expect func(limit int, ok bool)
	}{
		{
			name:  "deve negar budget lean para planejamento",
			tool:  "codex",
			gov:   GovernanceProfileCompact,
			skill: SkillProfileLean,
			flow:  FlowPlanning,
			expect: func(limit int, ok bool) {
				s.False(ok, "lean profile nao deve ter budget para FlowPlanning — planejamento requer full profile")
			},
		},
		{
			name:  "deve definir budget full para planejamento",
			tool:  "codex",
			gov:   GovernanceProfileStandard,
			skill: SkillProfileFull,
			flow:  FlowPlanning,
			expect: func(limit int, ok bool) {
				s.True(ok, "full profile deve ter budget para FlowPlanning")
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			limit, ok := NewCatalog().FlowBudget(scenario.tool, scenario.gov, scenario.skill, scenario.flow)

			scenario.expect(limit, ok)
		})
	}
}

func (s *FlowSuite) TestFlowBudgetTwoTools() {
	claudeLimit, claudeOk := NewCatalog().FlowBudget("claude", GovernanceProfileStandard, SkillProfileFull, FlowExecution)
	codexLimit, codexOk := NewCatalog().FlowBudget("codex", GovernanceProfileStandard, SkillProfileFull, FlowExecution)

	s.True(claudeOk, "claude deve ter budget definido para execution/standard/full")
	s.True(codexOk, "codex deve ter budget definido para execution/standard/full")
	s.True(claudeLimit > codexLimit, "claude (%d) deve ter limite maior que codex (%d) — janela de contexto maior", claudeLimit, codexLimit)
}

func (s *FlowSuite) TestMeasureFlowBudgetStatus() {
	scenarios := []struct {
		name   string
		tool   string
		gov    GovernanceProfile
		skill  SkillProfile
		flow   FlowKind
		files  map[string][]byte
		paths  []string
		expect func(m FlowMeasurement)
	}{
		{
			name:  "deve aceitar artefatos leves dentro do budget",
			tool:  "claude",
			gov:   GovernanceProfileStandard,
			skill: SkillProfileFull,
			flow:  FlowExecution,
			files: map[string][]byte{
				"/repo/AGENTS.md": repeatedBytes(3000),
				"/repo/.agents/skills/agent-governance/SKILL.md": repeatedBytes(3000),
			},
			paths: []string{
				"/repo/AGENTS.md",
				"/repo/.agents/skills/agent-governance/SKILL.md",
			},
			expect: func(m FlowMeasurement) {
				s.True(m.WithinBudget, "artefatos leves devem estar dentro do budget: est=%d limit=%d", m.TokensEst, m.TokensLimit)
			},
		},
		{
			name:  "deve rejeitar artefatos acima do budget",
			tool:  "codex",
			gov:   GovernanceProfileCompact,
			skill: SkillProfileLean,
			flow:  FlowExecution,
			files: map[string][]byte{
				"/repo/AGENTS.md": repeatedBytes(20000),
			},
			paths: []string{"/repo/AGENTS.md"},
			expect: func(m FlowMeasurement) {
				s.False(m.WithinBudget, "artefatos grandes devem exceder o budget do codex compact lean: est=%d limit=%d", m.TokensEst, m.TokensLimit)
			},
		},
		{
			name:  "deve aceitar combinacao sem budget definido",
			tool:  "ferramenta-desconhecida",
			gov:   GovernanceProfileStandard,
			skill: SkillProfileFull,
			flow:  FlowExecution,
			files: map[string][]byte{
				"/repo/AGENTS.md": repeatedBytes(100000),
			},
			paths: []string{"/repo/AGENTS.md"},
			expect: func(m FlowMeasurement) {
				s.True(m.WithinBudget, "combinacao sem budget definido deve retornar WithinBudget=true")
				s.Zero(m.TokensLimit, "combinacao sem budget deve ter TokensLimit=0")
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			for path, content := range scenario.files {
				ffs.Files[path] = content
			}
			svc := NewService(ffs, silentPrinter(), nil)

			m := svc.MeasureFlow(scenario.tool, scenario.gov, scenario.skill, scenario.flow, scenario.paths)

			scenario.expect(m)
		})
	}
}
