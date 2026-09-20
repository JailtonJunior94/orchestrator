package hookcontract

import "testing"

func TestExitCodeTranslator_PreservesCurrentSemantics(t *testing.T) {
	translator := NewExitCodeTranslator()

	tests := []struct {
		name     string
		origin   string
		code     int
		critical bool
		want     Decision
	}{
		{"git-operation-gate.sh GIT_OPERATION_BLOCK_EXIT", ".agents/scripts/git-operation-gate.sh:5", 2, true, DecisionBlock},
		{"validate-preload.sh PRELOAD_BLOCK_EXIT", ".agents/hooks/validate-preload.sh:5", 2, true, DecisionBlock},
		{"validate-session-end.sh SESSION_END_BLOCK_EXIT", ".agents/hooks/validate-session-end.sh:4", 2, true, DecisionBlock},
		{"validate-governance.sh literal exit 1", ".agents/hooks/validate-governance.sh:45", 1, true, DecisionBlock},
		{"success exit 0", "n/a", 0, true, DecisionAllow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translator.ToDecision(tt.code, "", tt.critical)
			if result.Decision() != tt.want {
				t.Fatalf("ToDecision(%d) [%s] = %v, want %v", tt.code, tt.origin, result.Decision(), tt.want)
			}
		})
	}
}

func TestExitCodeTranslator_CriticalUnmappedIsError(t *testing.T) {
	translator := NewExitCodeTranslator()

	result := translator.ToDecision(127, "command not found", true)
	if result.Decision() != DecisionError {
		t.Fatalf("Decision() = %v, want DecisionError", result.Decision())
	}
}

func TestExitCodeTranslator_NonCriticalUnmappedIsWarn(t *testing.T) {
	translator := NewExitCodeTranslator()

	result := translator.ToDecision(127, "command not found", false)
	if result.Decision() != DecisionWarn {
		t.Fatalf("Decision() = %v, want DecisionWarn", result.Decision())
	}
}

func TestExitCodeTranslator_ToExitCodeRoundTrip(t *testing.T) {
	translator := NewExitCodeTranslator()

	tests := []struct {
		result Result
		want   int
	}{
		{NewAllow(), 0},
		{NewNotApplicable("skip"), 0},
		{mustResult(t, DecisionWarn, "warn", "", "G-1"), 0},
		{mustResult(t, DecisionBlock, "blocked", "P-1", ""), 2},
		{mustResult(t, DecisionError, "error", "", ""), 2},
	}

	for _, tt := range tests {
		if got := translator.ToExitCode(tt.result); got != tt.want {
			t.Fatalf("ToExitCode(%v) = %d, want %d", tt.result.Decision(), got, tt.want)
		}
	}
}

func mustResult(t *testing.T, decision Decision, reason, policyID, gateID string) Result {
	t.Helper()
	result, err := NewResult(decision, reason, policyID, gateID)
	if err != nil {
		t.Fatalf("NewResult unexpected error: %v", err)
	}
	return result
}
