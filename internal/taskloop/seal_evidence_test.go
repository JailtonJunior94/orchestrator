package taskloop

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

// gitOut e o par de runGit para comandos cujo valor esta na saida.
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

// sealFixture monta um repositorio com trabalho ja commitado e devolve o
// resultado de execucao correspondente, pronto para ser selado.
func sealFixture(t *testing.T) (string, string, sdd.ExecutionResult) {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "teste@example.com")
	runGit(t, dir, "config", "user.name", "Teste")

	prdDir := filepath.Join(dir, ".specs", "prd-x")
	for _, sub := range []string{prdDir, filepath.Join(dir, "evidence")} {
		if err := os.MkdirAll(sub, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, content string) {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("rastreado.txt", "base\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "-c", "commit.gpgSign=false", "commit", "-m", "base")
	baseSHA := gitOut(t, dir, "rev-parse", "HEAD")

	write("rastreado.txt", "alterado\n")
	write("novo.go", "package novo\n")
	write("evidence/test.log", "PASS\n")
	write("evidence/patch.diff", "patch\n")
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "-c", "commit.gpgSign=false", "commit", "-m", "trabalho da tarefa")

	result := sdd.ExecutionResult{
		SchemaVersion: 2, RunID: "r", TaskID: "1.0", Attempt: 1, Status: sdd.StatusDone,
		BaseSHA: baseSHA, PatchSHA256: strings.Repeat("0", 64),
		PatchRef: "evidence/patch.diff", FinalStateSHA256: strings.Repeat("0", 64),
		Tests:    []sdd.TestProof{{Command: "go test", ExitCode: 0, OutputSHA256: strings.Repeat("0", 64)}},
		Criteria: []sdd.CriterionProof{{ID: "AC", EvidenceRef: "evidence/test.log"}},
		Evidence: []string{"evidence/test.log"}, ReviewVerdict: "approved",
	}
	return dir, prdDir, result
}

// TestSealEvidenceSobreviveAoAvancoDoRepositorio e o teste central de RF-14: a
// prova de fechamento depende da arvore viva e evapora no commit; o selo tem de
// continuar verificavel depois que o repositorio segue em frente.
func TestSealEvidenceSobreviveAoAvancoDoRepositorio(t *testing.T) {
	dir, prdDir, result := sealFixture(t)
	orchestrator := NewOrchestrator(sdd.NewStore())

	sealed, err := orchestrator.SealEvidence(prdDir, result, "HEAD")
	if err != nil {
		t.Fatalf("SealEvidence retornou erro: %v", err)
	}
	if sealed.CommitSHA == "" || len(sealed.CommitPatchSHA256) != 64 {
		t.Fatalf("selo incompleto: %+v", sealed)
	}

	// O repositorio avanca e a arvore fica suja: a prova de fechamento nao
	// sobreviveria a isso, o selo precisa sobreviver.
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

	if err := orchestrator.VerifySealedEvidence(prdDir, sealed); err != nil {
		t.Fatalf("evidencia selada deixou de verificar apos o repositorio avancar: %v", err)
	}
}

func TestSealEvidenceRejeitaDigestAdulterado(t *testing.T) {
	_, prdDir, result := sealFixture(t)
	orchestrator := NewOrchestrator(sdd.NewStore())

	sealed, err := orchestrator.SealEvidence(prdDir, result, "HEAD")
	if err != nil {
		t.Fatalf("SealEvidence retornou erro: %v", err)
	}
	sealed.CommitPatchSHA256 = strings.Repeat("f", 64)

	if err := orchestrator.VerifySealedEvidence(prdDir, sealed); err == nil {
		t.Fatal("digest adulterado deveria falhar a verificacao")
	}
}

func TestSealEvidenceRejeitaCommitForaDaLinhaDaBase(t *testing.T) {
	dir, prdDir, result := sealFixture(t)
	orchestrator := NewOrchestrator(sdd.NewStore())

	main := gitOut(t, dir, "rev-parse", "--abbrev-ref", "HEAD")
	runGit(t, dir, "checkout", "--orphan", "solto")
	runGit(t, dir, "rm", "-rf", "--cached", ".")
	if err := os.WriteFile(filepath.Join(dir, "orfao.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "orfao.txt")
	runGit(t, dir, "-c", "commit.gpgSign=false", "commit", "-m", "orfao")
	orphan := gitOut(t, dir, "rev-parse", "HEAD")
	runGit(t, dir, "checkout", "-f", main)

	if _, err := orchestrator.SealEvidence(prdDir, result, orphan); err == nil {
		t.Fatal("commit que nao descende da base deveria ser recusado")
	}
}

func TestSealEvidenceRecusaReselagemEStatusNaoDone(t *testing.T) {
	_, prdDir, result := sealFixture(t)
	orchestrator := NewOrchestrator(sdd.NewStore())

	sealed, err := orchestrator.SealEvidence(prdDir, result, "HEAD")
	if err != nil {
		t.Fatalf("SealEvidence retornou erro: %v", err)
	}
	if _, err := orchestrator.SealEvidence(prdDir, sealed, "HEAD"); err == nil {
		t.Fatal("resselagem deveria ser recusada")
	}

	naoDone := result
	naoDone.Status = sdd.StatusBlocked
	if _, err := orchestrator.SealEvidence(prdDir, naoDone, "HEAD"); err == nil {
		t.Fatal("status diferente de done nao pode ser selado")
	}
}

func TestVerifySealedEvidenceExigeSelo(t *testing.T) {
	_, prdDir, result := sealFixture(t)
	if err := NewOrchestrator(sdd.NewStore()).VerifySealedEvidence(prdDir, result); err == nil {
		t.Fatal("resultado sem selo deveria falhar a verificacao")
	}
}
