package metrics

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestToolBudgetsLargeCoverageRealRegistry(t *testing.T) {
	t.Parallel()

	if err := NewCatalog().CheckToolBudgetsLargeCoverage(); err != nil {
		t.Fatalf("CheckToolBudgetsLargeCoverage() = %v; want nil", err)
	}
}

func TestToolBudgetsLargeCoverageFailsOnMissingAgent(t *testing.T) {
	t.Parallel()

	registry := specs.NewCatalog().Registry()
	var largeAgentID string
	for _, agent := range registry {
		if agent.LargeBudget() > 0 {
			largeAgentID = agent.ID()
			break
		}
	}
	if largeAgentID == "" {
		t.Fatal("no agent with positive LargeBudget in registry — cannot exercise the missing-agent branch")
	}

	budgets := map[string]int{}
	err := checkToolBudgetsLargeCoverage(registry, budgets)
	if !errors.Is(err, ErrToolBudgetsLargeCoverage) {
		t.Fatalf("checkToolBudgetsLargeCoverage() = %v; want ErrToolBudgetsLargeCoverage", err)
	}
}

func TestToolBudgetsLargeCoverageFailsOnOrphanKey(t *testing.T) {
	t.Parallel()

	registry := specs.NewCatalog().Registry()
	budgets := map[string]int{"nonexistent-agent": 999_999}
	err := checkToolBudgetsLargeCoverage(registry, budgets)
	if !errors.Is(err, ErrToolBudgetsLargeCoverage) {
		t.Fatalf("checkToolBudgetsLargeCoverage() = %v; want ErrToolBudgetsLargeCoverage", err)
	}
}
