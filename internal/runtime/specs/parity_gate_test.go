package specs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

var mandatoryParityAgents = []string{"claude", "codex", "copilot", "opencode"}

func mandatoryParityCells(t *testing.T) []specs.AgentEnforcement {
	t.Helper()
	catalog := specs.NewCatalog()
	cells := make([]specs.AgentEnforcement, 0, len(mandatoryParityAgents))
	for _, id := range mandatoryParityAgents {
		agent, err := catalog.AgentByID(id)
		if err != nil {
			t.Fatalf("agent %q not in registry: %v", id, err)
		}
		cells = append(cells, specs.AgentEnforcement{Agent: id, Enforcement: agent.Enforcement()})
	}
	return cells
}

func TestParityGatePassesForMandatoryMatrix(t *testing.T) {
	t.Parallel()
	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, specs.DispatchProven)
	if len(violations) != 0 {
		t.Fatalf("expected no parity violations for the mandatory 4-agent matrix; got %v", violations)
	}
}

func TestParityGateFailsWhenAgentLosesCanonicalPoint(t *testing.T) {
	t.Parallel()

	cells := []specs.AgentEnforcement{
		{Agent: "claude", Enforcement: mustEnforcementFor(t, "claude")},
		{Agent: "codex", Enforcement: specs.Enforcement{}},
		{Agent: "copilot", Enforcement: mustEnforcementFor(t, "copilot")},
		{Agent: "opencode", Enforcement: mustEnforcementFor(t, "opencode")},
	}

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, specs.DispatchProven)
	found := false
	for _, v := range violations {
		if v.Agent == "codex" && v.Reason == "invalid enforcement" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected parity gate to flag codex with missing/invalid enforcement; got %v", violations)
	}
}

func TestParityGateFailsWhenValidatorDiverges(t *testing.T) {
	t.Parallel()
	catalog := specs.NewCatalog()

	claude, err := catalog.AgentByID("claude")
	if err != nil {
		t.Fatalf("AgentByID(claude): %v", err)
	}
	codex, err := catalog.AgentByID("codex")
	if err != nil {
		t.Fatalf("AgentByID(codex): %v", err)
	}

	divergentCoverage := make([]specs.PointCoverage, 0, 3)
	for _, cov := range codex.Enforcement().Coverage() {
		if cov.Point() == specs.PointPreTool {
			mutated, mkErr := catalog.NewPointCoverage(specs.PointPreTool, cov.NativeKey(), ".agents/scripts/some-other-validator.sh")
			if mkErr != nil {
				t.Fatalf("NewPointCoverage: %v", mkErr)
			}
			divergentCoverage = append(divergentCoverage, mutated)
			continue
		}
		divergentCoverage = append(divergentCoverage, cov)
	}
	divergentEnforcement, err := catalog.NewEnforcement(divergentCoverage, codex.Enforcement().Preconditions()...)
	if err != nil {
		t.Fatalf("NewEnforcement: %v", err)
	}

	cells := []specs.AgentEnforcement{
		{Agent: "claude", Enforcement: claude.Enforcement()},
		{Agent: "codex", Enforcement: divergentEnforcement},
		{Agent: "copilot", Enforcement: mustEnforcementFor(t, "copilot")},
		{Agent: "opencode", Enforcement: mustEnforcementFor(t, "opencode")},
	}

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, specs.DispatchProven)
	found := false
	for _, v := range violations {
		if v.Agent == "codex" && v.Point == specs.PointPreTool {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected parity gate to flag codex pre-tool validator divergence; got %v", violations)
	}
}

func TestParityGateFailsWhenCellHasNoDispatchProof(t *testing.T) {
	t.Parallel()
	cells := mandatoryParityCells(t)

	noProof := func(agentID string, point specs.CanonicalPoint) bool {
		if agentID == "opencode" && point == specs.PointSessionEnd {
			return false
		}
		return specs.DispatchProven(agentID, point)
	}

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, noProof)
	found := false
	for _, v := range violations {
		if v.Agent == "opencode" && v.Point == specs.PointSessionEnd && v.Reason == "no dispatch proof test associated" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected parity gate to flag opencode session-end without dispatch proof; got %v", violations)
	}
}

func mustEnforcementFor(t *testing.T, id string) specs.Enforcement {
	t.Helper()
	agent, err := specs.NewCatalog().AgentByID(id)
	if err != nil {
		t.Fatalf("AgentByID(%s): %v", id, err)
	}
	return agent.Enforcement()
}
