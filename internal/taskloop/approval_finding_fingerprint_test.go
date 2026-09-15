package taskloop

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

func TestTranslateReviewFindingsFingerprintIgnoresLineNumber(t *testing.T) {
	calculator := approval.NewFingerprintCalculator()

	first, err := translateReviewFindings([]Finding{{Severity: SeverityCritical, File: "internal/x/fix.go:12", Message: "mesmo achado"}})
	if err != nil {
		t.Fatalf("translateReviewFindings: %v", err)
	}
	second, err := translateReviewFindings([]Finding{{Severity: SeverityCritical, File: "internal/x/fix.go:47", Message: "mesmo achado deslocado"}})
	if err != nil {
		t.Fatalf("translateReviewFindings: %v", err)
	}

	if first[0].File() != "internal/x/fix.go" {
		t.Errorf("File() = %q, quero caminho sem numero de linha", first[0].File())
	}
	if got, want := calculator.Compute(first).String(), calculator.Compute(second).String(); got != want {
		t.Errorf("fingerprint %s != %s: RF-37/RF-43 exigem paridade entre os dois adaptadores", got, want)
	}
}
