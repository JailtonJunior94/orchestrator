package runtime

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

func TestBuildCycleOutcomeCarriesFingerprint(t *testing.T) {
	finding, err := approval.NewFinding(approval.SeverityCritical, "main.go", "review-finding", "uninitialized variable")
	if err != nil {
		t.Fatalf("NewFinding: %v", err)
	}
	fingerprint := approval.NewFingerprintCalculator().Compute([]approval.Finding{finding})

	round, err := approval.NewRound(1)
	if err != nil {
		t.Fatalf("NewRound: %v", err)
	}
	completed, err := round.Complete(approval.VerdictRejected, []approval.Finding{finding}, fingerprint)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	result, err := approval.NewClosedResult(approval.ReasonNoConvergence, []approval.Round{completed})
	if err != nil {
		t.Fatalf("NewClosedResult: %v", err)
	}

	outcome := buildCycleOutcome(Job{EvidenceDir: "evidence"}, result)
	if len(outcome.cycleRounds) != 1 {
		t.Fatalf("cycleRounds=%d, want 1", len(outcome.cycleRounds))
	}
	summary := outcome.cycleRounds[0]
	if summary.Fingerprint != fingerprint.String() {
		t.Errorf("Fingerprint=%q, want %q", summary.Fingerprint, fingerprint.String())
	}
	if summary.Verdict == "" {
		t.Error("veredito da rodada ausente no relatorio consolidado")
	}
	if summary.FindingsBySeverity["critical"] != 1 {
		t.Errorf("contagem por severidade=%v, want critical=1", summary.FindingsBySeverity)
	}
	if outcome.cycleStopReason != approval.ReasonNoConvergence.String() {
		t.Errorf("cycleStopReason=%q, want %q", outcome.cycleStopReason, approval.ReasonNoConvergence.String())
	}
}
