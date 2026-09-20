package specs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

var mandatoryParityAgents = []string{"claude", "codex", "copilot", "opencode"}

func repoScriptResolver(t *testing.T) specs.ScriptResolver {
	t.Helper()
	root := repoRoot(t)
	return func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(relPath)))
	}
}

func allCellsDispatchProven(string, specs.CanonicalPoint) bool { return true }

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

func TestParityGateFailsWhenAgentLosesCanonicalPoint(t *testing.T) {
	t.Parallel()

	cells := []specs.AgentEnforcement{
		{Agent: "claude", Enforcement: mustEnforcementFor(t, "claude")},
		{Agent: "codex", Enforcement: specs.Enforcement{}},
		{Agent: "copilot", Enforcement: mustEnforcementFor(t, "copilot")},
		{Agent: "opencode", Enforcement: mustEnforcementFor(t, "opencode")},
	}

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, nil, repoScriptResolver(t))
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
			mutated, mkErr := catalog.NewPointCoverage("codex", specs.PointPreTool, cov.NativeKey(), ".agents/hooks/some-other-validator.sh", cov.ArtifactPath())
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

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, allCellsDispatchProven, repoScriptResolver(t))
	found := false
	for _, v := range violations {
		if v.Agent == "codex" && v.Point == specs.PointPreTool && strings.Contains(v.Reason, "validator diverges") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected parity gate to flag codex pre-tool validator divergence; got %v", violations)
	}
	for _, v := range violations {
		if v.Reason == "no dispatch proof test associated" {
			t.Fatalf("dispatch proof was supplied for every cell; divergence must be the only violation: %v", violations)
		}
	}
}

func replaceCoverage(t *testing.T, agentID string, point specs.CanonicalPoint, scriptPath, artifactPath string) specs.Enforcement {
	t.Helper()
	catalog := specs.NewCatalog()
	agent, err := catalog.AgentByID(agentID)
	if err != nil {
		t.Fatalf("AgentByID(%s): %v", agentID, err)
	}
	coverage := make([]specs.PointCoverage, 0, 3)
	for _, cov := range agent.Enforcement().Coverage() {
		if cov.Point() != point {
			coverage = append(coverage, cov)
			continue
		}
		mutated, mkErr := catalog.NewPointCoverage(agentID, point, cov.NativeKey(), scriptPath, artifactPath)
		if mkErr != nil {
			t.Fatalf("NewPointCoverage: %v", mkErr)
		}
		coverage = append(coverage, mutated)
	}
	enf, err := catalog.NewEnforcement(coverage, agent.Enforcement().Preconditions()...)
	if err != nil {
		t.Fatalf("NewEnforcement: %v", err)
	}
	return enf
}

func TestParityGateFailsWhenDeclaredValidatorDoesNotExistOnDisk(t *testing.T) {
	t.Parallel()

	const ghost = ".agents/scripts/NOPE.sh"
	cells := make([]specs.AgentEnforcement, 0, len(mandatoryParityAgents))
	for _, id := range mandatoryParityAgents {
		cells = append(cells, specs.AgentEnforcement{
			Agent:       id,
			Enforcement: replaceCoverage(t, id, specs.PointPreTool, ghost, mustArtifactPath(t, id, specs.PointPreTool)),
		})
	}

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, allCellsDispatchProven, repoScriptResolver(t))
	flagged := map[string]bool{}
	for _, v := range violations {
		if v.Point == specs.PointPreTool && strings.Contains(v.Reason, "does not exist on disk") {
			flagged[v.Agent] = true
		}
	}
	for _, id := range mandatoryParityAgents {
		if !flagged[id] {
			t.Fatalf("agent %q: a canonical validator path that exists nowhere on disk must be a violation, not a trivially satisfied string comparison; got %v", id, violations)
		}
	}
}

func TestParityGateFailsWhenArtifactNeitherMirrorsNorExecutesCanonical(t *testing.T) {
	t.Parallel()

	cells := mandatoryParityCells(t)
	for i := range cells {
		if cells[i].Agent != "claude" {
			continue
		}
		cells[i].Enforcement = replaceCoverage(t, "claude", specs.PointPreTool, ".agents/hooks/validate-preload.sh", ".claude/hooks/validate-token-budget.sh")
	}

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, allCellsDispatchProven, repoScriptResolver(t))
	found := false
	for _, v := range violations {
		if v.Agent == "claude" && v.Point == specs.PointPreTool && strings.Contains(v.Reason, "neither mirrors nor executes") {
			found = true
		}
	}
	if !found {
		t.Fatalf("an installed artifact unrelated to the canonical validator must be flagged; got %v", violations)
	}
}

func TestParityGateFailsWithoutFilesystemConfrontation(t *testing.T) {
	t.Parallel()

	violations := specs.ValidateParityMatrix(mandatoryParityCells(t), mandatoryParityAgents, allCellsDispatchProven, nil)
	if len(violations) == 0 {
		t.Fatal("without a script resolver the gate must refuse to certify parity: the declared chain was never confronted with disk")
	}
}

func mustArtifactPath(t *testing.T, agentID string, point specs.CanonicalPoint) string {
	t.Helper()
	path, ok := specs.InstalledArtifactPath(agentID, point)
	if !ok {
		t.Fatalf("no installed artifact declared for %s point %s", agentID, point)
	}
	return path
}

func TestParityGateFailsWhenCellHasNoDispatchProof(t *testing.T) {
	t.Parallel()
	cells := mandatoryParityCells(t)

	violations := specs.ValidateParityMatrix(cells, mandatoryParityAgents, nil, repoScriptResolver(t))
	found := false
	for _, v := range violations {
		if v.Agent == "opencode" && v.Point == specs.PointBeforeComplete && v.Reason == "no dispatch proof test associated" {
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
