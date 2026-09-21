//go:build hook_dispatch_proof

package capability

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestHookCapabilityMatrix_RealMatrixEvidenceGateIsClean(t *testing.T) {
	m, err := GenerateHooks()
	if err != nil {
		t.Fatalf("GenerateHooks: %v", err)
	}
	proof, err := HookDispatchProvenFromIntegrationTests(m)
	if err != nil {
		t.Fatalf("HookDispatchProvenFromIntegrationTests: %v", err)
	}
	violations := specs.ValidateCapabilityMatrixEvidence(EvidenceHookCells(m), proof)
	if len(violations) != 0 {
		t.Errorf("current hook capability matrix has unresolved dispatch proof: %v", violations)
	}
}
