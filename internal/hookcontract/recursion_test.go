package hookcontract

import (
	"errors"
	"testing"
)

func TestRecursionGuard_TopLevelInvocationAllowed(t *testing.T) {
	result, err := CheckRecursionGuard(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision() != DecisionAllow {
		t.Fatalf("Decision() = %v, want DecisionAllow for top-level invocation", result.Decision())
	}
}

func TestRecursionGuard_DirectRecursionBlocked(t *testing.T) {
	envelope := Envelope{InvocationDepth: 0}
	nested := NextInvocationDepth(envelope)

	result, err := CheckRecursionGuard(nested)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision() != DecisionBlock {
		t.Fatalf("Decision() = %v, want DecisionBlock — hook re-invoking itself via a tool must be blocked", result.Decision())
	}
}

func TestRecursionGuard_IndirectRecursionBlocked(t *testing.T) {
	hookA := Envelope{Provider: "provider-x", InvocationDepth: 0}
	toolSpawnedDepth := NextInvocationDepth(hookA)
	hookB := Envelope{Provider: "provider-x", InvocationDepth: toolSpawnedDepth}

	result, err := CheckRecursionGuard(hookB.InvocationDepth)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision() != DecisionBlock {
		t.Fatalf(
			"Decision() = %v, want DecisionBlock — a different hook invoked from a tool spawned by the first hook is still recursion",
			result.Decision())
	}
}

func TestRecursionGuard_NegativeDepthRejected(t *testing.T) {
	if _, err := CheckRecursionGuard(-1); !errors.Is(err, ErrRecursionGuard) {
		t.Fatalf("error = %v, want ErrRecursionGuard", err)
	}
}

func TestNextInvocationDepth_IncrementsFromEnvelope(t *testing.T) {
	envelope := Envelope{InvocationDepth: 3}
	if got := NextInvocationDepth(envelope); got != 4 {
		t.Fatalf("NextInvocationDepth() = %d, want 4", got)
	}
}
