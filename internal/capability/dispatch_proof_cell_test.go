//go:build capability_instrumentation

package capability

import (
	"os"
	"testing"
)

func TestDispatchProof_DiscriminatesByCell(t *testing.T) {
	mustHaveSyntacticMethod(t)
	dir := writeFixtureModule(t, "")

	proven := dispatchProvenFromTests([]byte(fakeParitySource), evidenceTestSuite, EvidenceTest, dir, "./...")

	genuineCell := proven("claude", "C01")
	if !genuineCell {
		t.Fatal("fixture setup invalid: a genuinely executed and passing test must prove its own cell")
	}

	unrelatedCell := proven("nonexistent-provider", "CAPABILITY-WITHOUT-ANY-ASSOCIATED-TEST")
	if unrelatedCell == genuineCell {
		t.Fatalf(
			"evidence.go:147-149 devolve uma closure que ignora provider e capabilityID: a celula "+
				"nao relacionada (%q, %q), sem nenhum teste proprio, foi declarada provada (%v) apenas "+
				"porque um teste diferente e nao relacionado passou; a prova ainda nao discrimina por "+
				"celula. Sucessor: tarefa 9.0",
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
	brokenEnvironmentProof := DispatchProvenFromParityTests([]byte(fakeParitySource))
	brokenEnvironmentResult := brokenEnvironmentProof("claude", "C01")

	if chdirErr := os.Chdir(repoRoot); chdirErr != nil {
		t.Fatalf("restaurar diretorio de trabalho apos simular falha de ambiente: %v", chdirErr)
	}
	t.Cleanup(func() {
		_ = os.Chdir(repoRoot)
	})

	mustHaveSyntacticMethod(t)
	fixtureDir := writeFixtureModule(t, "")
	missingEvidenceProof := dispatchProvenFromTests([]byte(fakeParitySource), evidenceTestSuite, "TestParitySuite/TestParity_DoesNotExist", fixtureDir, "./...")
	missingEvidenceResult := missingEvidenceProof("claude", "C01")

	if brokenEnvironmentResult != missingEvidenceResult {
		return
	}
	t.Fatalf(
		"falha de resolucao da raiz do repositorio (evidence.go:169-173, repoRootFromWorkingDir sem "+
			"go.mod ancestral) colapsa para o mesmo booleano (%v) que a ausencia legitima de prova de "+
			"uma celula; DispatchProvenFromParityTests precisa reportar falha de ambiente distintamente "+
			"de celula nao provada, nao apenas devolver false silenciosamente. Sucessor: tarefa 9.0",
		brokenEnvironmentResult,
	)
}
