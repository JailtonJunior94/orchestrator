package specs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestValidateCapabilityMatrixEvidence_EmptyMatrix(t *testing.T) {
	t.Parallel()
	violations := specs.ValidateCapabilityMatrixEvidence(nil, nil)
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for an empty matrix, got %d: %v", len(violations), violations)
	}
	if violations[0].Reason != "capability matrix has zero cells" {
		t.Errorf("unexpected reason: %q", violations[0].Reason)
	}
}

func TestValidateCapabilityMatrixEvidence_SupportedWithoutDispatchProof(t *testing.T) {
	t.Parallel()
	cells := []specs.CapabilityCell{
		{Provider: "claude", Capability: "C01", Supported: true},
		{Provider: "codex", Capability: "C01", Supported: false},
	}
	violations := specs.ValidateCapabilityMatrixEvidence(cells, func(string, string) bool { return false })
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation, got %d: %v", len(violations), violations)
	}
	if violations[0].Reason != "no dispatch proof test associated" {
		t.Errorf("literal reason must be preserved from ValidateParityMatrix, got %q", violations[0].Reason)
	}
	if violations[0].Agent != "claude" {
		t.Errorf("expected violation attributed to provider claude, got %q", violations[0].Agent)
	}
}

func TestValidateCapabilityMatrixEvidence_NilDispatchProvenFailsClosed(t *testing.T) {
	t.Parallel()
	cells := []specs.CapabilityCell{{Provider: "claude", Capability: "C01", Supported: true}}
	violations := specs.ValidateCapabilityMatrixEvidence(cells, nil)
	if len(violations) != 1 {
		t.Fatalf("expected fail-closed behaviour with nil dispatchProven, got %d violations", len(violations))
	}
}

func TestValidateCapabilityMatrixEvidence_ResolvedProofPasses(t *testing.T) {
	t.Parallel()
	cells := []specs.CapabilityCell{
		{Provider: "claude", Capability: "C01", Supported: true},
		{Provider: "codex", Capability: "C01", Supported: true},
	}
	violations := specs.ValidateCapabilityMatrixEvidence(cells, func(string, string) bool { return true })
	if len(violations) != 0 {
		t.Errorf("expected no violations, got %v", violations)
	}
}
