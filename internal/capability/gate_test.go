package capability

import (
	"os"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestCapabilityMatrix_GateFailsOnEmptyMatrix(t *testing.T) {
	violations := specs.ValidateCapabilityMatrixEvidence(nil, nil)
	if len(violations) == 0 {
		t.Fatal("expected a violation for an empty capability matrix, got none")
	}
	found := false
	for _, v := range violations {
		if v.Reason == "capability matrix has zero cells" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected violation reason %q, got %v", "capability matrix has zero cells", violations)
	}
}

func TestCapabilityMatrix_GateFailsOnSupportedCellWithoutTest(t *testing.T) {
	cells := []specs.CapabilityCell{{Provider: "claude", Capability: "C01", Supported: true}}
	violations := specs.ValidateCapabilityMatrixEvidence(cells, func(provider, capability string) bool { return false })
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation, got %d: %v", len(violations), violations)
	}
	if violations[0].Reason != "no dispatch proof test associated" {
		t.Errorf("expected literal reason %q, got %q", "no dispatch proof test associated", violations[0].Reason)
	}
}

func TestCapabilityMatrix_GatePassesWhenDispatchProven(t *testing.T) {
	cells := []specs.CapabilityCell{{Provider: "claude", Capability: "C01", Supported: true}}
	violations := specs.ValidateCapabilityMatrixEvidence(cells, func(provider, capability string) bool { return true })
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %v", violations)
	}
}

func TestCapabilityMatrix_GateIgnoresUnsupportedCells(t *testing.T) {
	cells := []specs.CapabilityCell{{Provider: "claude", Capability: "C01", Supported: false}}
	violations := specs.ValidateCapabilityMatrixEvidence(cells, func(provider, capability string) bool { return false })
	if len(violations) != 0 {
		t.Errorf("unsupported cells must never require dispatch proof, got %v", violations)
	}
}

func TestCapabilityMatrix_RealMatrixEvidenceGateIsClean(t *testing.T) {
	m, err := Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	source, err := os.ReadFile("../parity/parity_test.go")
	if err != nil {
		t.Fatalf("read parity_test.go: %v", err)
	}
	violations := specs.ValidateCapabilityMatrixEvidence(EvidenceCells(m), DispatchProvenFromParityTests(source))
	if len(violations) != 0 {
		t.Errorf("current capability matrix has unresolved dispatch proof: %v", violations)
	}
}

func TestCapabilityMatrix_GoldenGateDetectsJSONOnlyDrift(t *testing.T) {
	golden, err := os.ReadFile(capabilityMatrixJSONPath())
	if err != nil {
		t.Fatalf("read golden JSON: %v", err)
	}
	tampered := append(append([]byte{}, golden...), []byte("tampered\n")...)

	matches, _, err := goldenMatches(capabilityMatrixJSONPath(), tampered)
	if err != nil {
		t.Fatalf("goldenMatches: %v", err)
	}
	if matches {
		t.Fatal("golden-file gate must fail when the committed JSON drifts from the generated one, but it passed")
	}
}

func TestCapabilityMatrix_GoldenGateDetectsMarkdownOnlyDrift(t *testing.T) {
	golden, err := os.ReadFile(capabilityMatrixMarkdownPath())
	if err != nil {
		t.Fatalf("read golden Markdown: %v", err)
	}
	tampered := append(append([]byte{}, golden...), []byte("tampered\n")...)

	matches, _, err := goldenMatches(capabilityMatrixMarkdownPath(), tampered)
	if err != nil {
		t.Fatalf("goldenMatches: %v", err)
	}
	if matches {
		t.Fatal("golden-file gate must fail when the committed Markdown drifts from the generated one, but it passed")
	}
}

func TestCapabilityMatrix_GoldenGateApprovesExactMatch(t *testing.T) {
	golden, err := os.ReadFile(capabilityMatrixJSONPath())
	if err != nil {
		t.Fatalf("read golden JSON: %v", err)
	}
	matches, diff, err := goldenMatches(capabilityMatrixJSONPath(), golden)
	if err != nil {
		t.Fatalf("goldenMatches: %v", err)
	}
	if !matches {
		t.Fatalf("golden-file gate must pass when content is byte-identical to the committed artifact; diff=%s", diff)
	}
}
