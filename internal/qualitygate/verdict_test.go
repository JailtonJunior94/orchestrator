package qualitygate

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
)

func TestVerdict_OverrideOfDeterministicResultAlwaysFails(t *testing.T) {
	deterministic, err := hookcontract.NewResult(hookcontract.DecisionBlock, "required check failed", "quality-gate:feature:high", GateID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	verdict := NewVerdict(deterministic)

	llmProposedOverride := hookcontract.NewAllow()
	if err := verdict.Override(llmProposedOverride); !errors.Is(err, ErrVerdictSealed) {
		t.Fatalf("expected ErrVerdictSealed, got %v", err)
	}

	if verdict.Result().Decision() != hookcontract.DecisionBlock {
		t.Fatalf("deterministic result must remain BLOCK after override attempt, got %v", verdict.Result().Decision())
	}
}

func TestVerdict_ResultReturnsOriginal(t *testing.T) {
	original := hookcontract.NewAllow()
	verdict := NewVerdict(original)

	if verdict.Result().Decision() != hookcontract.DecisionAllow {
		t.Fatalf("Result() = %v, want ALLOW", verdict.Result().Decision())
	}
}
