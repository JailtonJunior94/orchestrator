//go:build capability_instrumentation

package capability

import (
	"errors"
	"os"
	"testing"
)

func TestDispatchProof_DiscriminatesByCell(t *testing.T) {
	mustHaveSyntacticMethod(t)
	dir := writeFixtureModule(t, "")

	proven := dispatchProvenFromTests([]byte(fakeParitySource), evidenceTestSuite, EvidenceTest, dir, "./...", cellProof{Provider: "claude", Capability: "C01"})

	genuineCell := proven("claude", "C01")
	if !genuineCell {
		t.Fatal("fixture setup invalid: a genuinely executed and passing test must prove its own declared cell")
	}

	unrelatedCell := proven("nonexistent-provider", "CAPABILITY-WITHOUT-ANY-ASSOCIATED-TEST")
	if unrelatedCell == genuineCell {
		t.Fatalf(
			"dispatchProvenFromTests must discriminate by cell: the unrelated cell (%q, %q), never "+
				"declared in the proven-cells set, was declared proved (%v) by the same closure that "+
				"proves the genuinely mapped cell",
			"nonexistent-provider", "CAPABILITY-WITHOUT-ANY-ASSOCIATED-TEST", unrelatedCell,
		)
	}
}

func TestDispatchProof_ReportsEnvironmentFailure(t *testing.T) {
	repoRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	outsideRepo := t.TempDir()

	if chdirErr := os.Chdir(outsideRepo); chdirErr != nil {
		t.Fatalf("chdir para fora do repositorio: %v", chdirErr)
	}
	_, envErr := DispatchProvenFromParityTests([]byte(fakeParitySource))
	if chdirErr := os.Chdir(repoRoot); chdirErr != nil {
		t.Fatalf("restaurar diretorio de trabalho apos simular falha de ambiente: %v", chdirErr)
	}
	t.Cleanup(func() {
		_ = os.Chdir(repoRoot)
	})

	if envErr == nil {
		t.Fatal("expected DispatchProvenFromParityTests to report a distinct environment error when repo root cannot be resolved, got nil")
	}
	if !errors.Is(envErr, ErrDispatchProofEnvironment) {
		t.Fatalf("expected error to wrap ErrDispatchProofEnvironment, got: %v", envErr)
	}

	mustHaveSyntacticMethod(t)
	fixtureDir := writeFixtureModule(t, "")
	missingEvidenceProof := dispatchProvenFromTests([]byte(fakeParitySource), evidenceTestSuite, "TestParitySuite/TestParity_DoesNotExist", fixtureDir, "./...", cellProof{Provider: "claude", Capability: "C01"})
	missingEvidenceResult := missingEvidenceProof("claude", "C01")

	if missingEvidenceResult {
		t.Fatal("a nonexistent test method must never prove a cell")
	}
}
