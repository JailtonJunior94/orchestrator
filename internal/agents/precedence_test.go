package agents

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PrecedenceSuite struct {
	suite.Suite
}

func TestPrecedenceSuite(t *testing.T) {
	suite.Run(t, new(PrecedenceSuite))
}

func (s *PrecedenceSuite) TestApplyRuntimePrecedence() {
	scenarios := []struct {
		name     string
		cfg      *RuntimeOverride
		defaults RuntimeDefaults
		expect   func(cfg *RuntimeOverride)
	}{
		{
			name: "deve preservar model explicitado pela CLI e preencher demais campos",
			cfg: &RuntimeOverride{
				Model:         "cli-model",
				ExplicitModel: true,
			},
			defaults: RuntimeDefaults{
				IDE:             "claude",
				Model:           "agent-model",
				ReasoningEffort: "high",
				AccessMode:      "bypass-permissions",
			},
			expect: func(cfg *RuntimeOverride) {
				s.Equal("cli-model", cfg.Model)
				s.Equal("claude", cfg.IDE)
				s.Equal("high", cfg.ReasoningEffort)
				s.Equal("bypass-permissions", cfg.AccessMode)
			},
		},
		{
			name: "deve preencher model do agente quando CLI nao explicita",
			cfg:  &RuntimeOverride{},
			defaults: RuntimeDefaults{
				IDE:   "codex",
				Model: "agent-model",
			},
			expect: func(cfg *RuntimeOverride) {
				s.Equal("codex", cfg.IDE)
				s.Equal("agent-model", cfg.Model)
			},
		},
		{
			name:     "deve manter override vazio quando CLI e agente estao vazios",
			cfg:      &RuntimeOverride{},
			defaults: RuntimeDefaults{},
			expect: func(cfg *RuntimeOverride) {
				s.Empty(cfg.IDE)
				s.Empty(cfg.Model)
				s.Empty(cfg.ReasoningEffort)
				s.Empty(cfg.AccessMode)
			},
		},
		{
			name: "deve preservar todos os campos explicitados pela CLI",
			cfg: &RuntimeOverride{
				IDE:                     "gemini",
				Model:                   "gemini-pro",
				ReasoningEffort:         "low",
				AccessMode:              "readonly",
				ExplicitIDE:             true,
				ExplicitModel:           true,
				ExplicitReasoningEffort: true,
				ExplicitAccessMode:      true,
			},
			defaults: RuntimeDefaults{
				IDE:             "claude",
				Model:           "claude-opus-4-7",
				ReasoningEffort: "high",
				AccessMode:      "bypass-permissions",
			},
			expect: func(cfg *RuntimeOverride) {
				s.Equal("gemini", cfg.IDE)
				s.Equal("gemini-pro", cfg.Model)
				s.Equal("low", cfg.ReasoningEffort)
				s.Equal("readonly", cfg.AccessMode)
			},
		},
		{
			name: "deve preservar campos parciais da CLI e preencher restantes pelo agente",
			cfg: &RuntimeOverride{
				IDE:         "copilot",
				ExplicitIDE: true,
			},
			defaults: RuntimeDefaults{
				IDE:             "claude",
				Model:           "claude-opus-4-7",
				ReasoningEffort: "medium",
			},
			expect: func(cfg *RuntimeOverride) {
				s.Equal("copilot", cfg.IDE)
				s.Equal("claude-opus-4-7", cfg.Model)
				s.Equal("medium", cfg.ReasoningEffort)
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			NewCatalog().applyRuntimePrecedence(scenario.cfg, scenario.defaults)

			scenario.expect(scenario.cfg)
		})
	}
}
