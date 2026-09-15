package taskloop

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

func sealedValidationFixture(t *testing.T) (string, string, sdd.ExecutionResult) {
	t.Helper()
	dir, prdDir, result := sealFixture(t)

	digestOf := func(rel string) string {
		content, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(content)
		return hex.EncodeToString(sum[:])
	}

	result.PatchSHA256 = digestOf("evidence/patch.diff")
	result.Tests = []sdd.TestProof{{Command: "go test", ExitCode: 0, OutputSHA256: digestOf("evidence/test.log")}}
	return dir, prdDir, result
}

func TestValidateExecutionEvidenceAceitaSeloAposRepositorioAvancar(t *testing.T) {
	dir, prdDir, result := sealedValidationFixture(t)
	orchestrator := NewOrchestrator(sdd.NewStore())

	sealed, err := orchestrator.SealEvidence(prdDir, result, "HEAD")
	if err != nil {
		t.Fatalf("SealEvidence retornou erro: %v", err)
	}

	for _, name := range []string{"depois1.txt", "depois2.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		runGit(t, dir, "add", "-A")
		runGit(t, dir, "-c", "commit.gpgSign=false", "commit", "-m", name)
	}
	if err := os.WriteFile(filepath.Join(dir, "rastreado.txt"), []byte("sujeira\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	reportPath := filepath.Join(prdDir, "1.0_execution_report.md")
	if err := os.WriteFile(reportPath, []byte("# relatorio\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := orchestrator.ValidateExecutionEvidence(prdDir, sealed, reportPath); err != nil {
		t.Fatalf("evidencia selada legitima deveria validar apos o HEAD avancar: %v", err)
	}
}

func TestValidateExecutionEvidenceRecusaSeloAdulterado(t *testing.T) {
	_, prdDir, result := sealedValidationFixture(t)
	orchestrator := NewOrchestrator(sdd.NewStore())

	sealed, err := orchestrator.SealEvidence(prdDir, result, "HEAD")
	if err != nil {
		t.Fatalf("SealEvidence retornou erro: %v", err)
	}

	for name, mutate := range map[string]func(sdd.ExecutionResult) sdd.ExecutionResult{
		"commit_patch_sha256": func(r sdd.ExecutionResult) sdd.ExecutionResult {
			r.CommitPatchSHA256 = hex.EncodeToString(make([]byte, 32))
			return r
		},
		"patch_sha256": func(r sdd.ExecutionResult) sdd.ExecutionResult {
			r.PatchSHA256 = hex.EncodeToString(make([]byte, 32))
			return r
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := orchestrator.ValidateExecutionEvidence(prdDir, mutate(sealed)); err == nil {
				t.Fatalf("resultado com %s adulterado deveria reprovar", name)
			}
		})
	}
}

func TestValidateExecutionEvidenceIgnoraExclusoesOperacionaisDoChamador(t *testing.T) {
	_, prdDir, result := sealedValidationFixture(t)
	orchestrator := NewOrchestrator(sdd.NewStore())

	sealed, err := orchestrator.SealEvidence(prdDir, result, "HEAD")
	if err != nil {
		t.Fatalf("SealEvidence retornou erro: %v", err)
	}

	semExclusao := orchestrator.ValidateExecutionEvidence(prdDir, sealed)
	comExclusao := orchestrator.ValidateExecutionEvidence(prdDir, sealed, filepath.Join(prdDir, "1.0_execution_report.md"), "qualquer/outro.md")
	if (semExclusao == nil) != (comExclusao == nil) {
		t.Fatalf("verificacao do selo dependeu das exclusoes do chamador: sem=%v com=%v", semExclusao, comExclusao)
	}
	if semExclusao != nil {
		t.Fatalf("selo legitimo deveria validar: %v", semExclusao)
	}
}
