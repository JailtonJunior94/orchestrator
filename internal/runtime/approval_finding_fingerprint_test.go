package runtime

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

func TestParseCycleFindingsFingerprintIgnoresLineNumber(t *testing.T) {
	calculator := approval.NewFingerprintCalculator()

	first := parseCycleFindings("[HIGH] internal/x/fix.go:12 mesmo achado\n")
	second := parseCycleFindings("[HIGH] internal/x/fix.go:47 mesmo achado deslocado\n")

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("findings = %d/%d, quero 1/1", len(first), len(second))
	}
	if first[0].File() != "internal/x/fix.go" {
		t.Errorf("File() = %q, quero caminho sem numero de linha", first[0].File())
	}
	if first[0].Line() != 12 || second[0].Line() != 47 {
		t.Errorf("Line() = %d/%d, quero 12/47 preservados fora da identidade", first[0].Line(), second[0].Line())
	}
	if got, want := calculator.Compute(first).String(), calculator.Compute(second).String(); got != want {
		t.Errorf("fingerprint %s != %s: RF-37 exige identidade estavel sob mudanca de linha", got, want)
	}
}

func TestParseCycleFindingsFingerprintStillDistinguishesFiles(t *testing.T) {
	calculator := approval.NewFingerprintCalculator()

	first := parseCycleFindings("[HIGH] internal/x/fix.go:12 achado\n")
	second := parseCycleFindings("[HIGH] internal/x/other.go:12 achado\n")

	if calculator.Compute(first).Equal(calculator.Compute(second)) {
		t.Error("fingerprint identica para arquivos distintos: identidade perdeu discriminacao")
	}
}
