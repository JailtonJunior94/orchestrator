package taskloop

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

func rawVerdict(v ReviewVerdict) string {
	return "Verdict: " + string(v)
}

func TestStubReviewerRawOutputMatchesDeclaredVerdict(t *testing.T) {
	translator := approval.NewTranslator()

	cases := []struct {
		name    string
		verdict ReviewVerdict
	}{
		{"approved", VerdictApproved},
		{"approved_with_remarks", VerdictApprovedWithRemarks},
		{"rejected", VerdictRejected},
		{"blocked", VerdictBlocked},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := rawVerdict(tc.verdict)
			if got := translator.Translate(raw).String(); got != string(tc.verdict) {
				t.Fatalf("Translate(%q).String() = %q, want %q", raw, got, tc.verdict)
			}
		})
	}
}

func TestStubReviewerDefaultResultCarriesRawText(t *testing.T) {
	reviewer := &stubReviewer{}

	result, err := reviewer.ReviewConsolidated(t.Context(), "diff")
	if err != nil {
		t.Fatalf("ReviewConsolidated: %v", err)
	}
	if result.RawOutput == "" {
		t.Fatal("default stub result has empty RawOutput")
	}
	translator := approval.NewTranslator()
	if got := translator.Translate(result.RawOutput).String(); got != string(result.Verdict) {
		t.Fatalf("Translate(RawOutput).String() = %q, want %q", got, result.Verdict)
	}
}
