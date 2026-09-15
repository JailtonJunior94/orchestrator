package evidence

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestResolveContract_ExemptionRequiresSealedContentNotMereExistence(t *testing.T) {
	dir := newGitFixture(t)
	report := filepath.Join(dir, "1.0_execution_report.md")
	sealed := "# Relatorio\n\nConteudo selado no corte.\n"
	if err := os.WriteFile(report, []byte(sealed), 0o600); err != nil {
		t.Fatal(err)
	}
	pinCut(t, commitAllAt(t, dir))

	if contract, findings := ResolveContract(sealed, report); contract != ContractV1 || len(findings) != 0 {
		t.Fatalf("conteudo identico ao selo deve manter a isencao: %v %v", contract, findings)
	}

	tampered := sealed + "linha acrescentada depois do corte\n"
	if err := os.WriteFile(report, []byte(tampered), 0o600); err != nil {
		t.Fatal(err)
	}
	contract, findings := ResolveContract(tampered, report)
	if contract != ContractV2 || !findingsContain(findings, "byte a byte") {
		t.Fatalf("conteudo reescrito depois do corte perde a isencao (o arquivo continua existindo no corte): %v %v", contract, findings)
	}

	if err := os.WriteFile(report, []byte(sealed), 0o600); err != nil {
		t.Fatal(err)
	}
	if contract, _ := ResolveContract(sealed, report); contract != ContractV1 {
		t.Fatalf("restaurar o conteudo selado devolve a isencao, got %v", contract)
	}
}

func TestResolveContract_TamperedTextIsNotLaunderedByPristineFile(t *testing.T) {
	dir := newGitFixture(t)
	report := filepath.Join(dir, "1.0_execution_report.md")
	sealed := "# Relatorio\n\nConteudo selado no corte.\n"
	if err := os.WriteFile(report, []byte(sealed), 0o600); err != nil {
		t.Fatal(err)
	}
	pinCut(t, commitAllAt(t, dir))

	contract, findings := ResolveContract(sealed+"texto adulterado em memoria\n", report)
	if contract != ContractV2 || !findingsContain(findings, "byte a byte") {
		t.Fatalf("o texto validado, nao o arquivo em disco, e o que precisa casar com o selo: %v %v", contract, findings)
	}
}

func TestContractCut_IsPinnedCommitNotMovingRef(t *testing.T) {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(ContractCutCommit) {
		t.Fatalf("o corte precisa ser um commit fixo de 40 hexadecimais, got %q", ContractCutCommit)
	}
	for _, moving := range []string{"HEAD", "head", "@", "ORIG_HEAD", "FETCH_HEAD", "main", "master"} {
		if strings.EqualFold(ContractCutCommit, moving) {
			t.Fatalf("referencia movel proibida como corte: %q", ContractCutCommit)
		}
	}
	if contractCut != ContractCutCommit {
		t.Fatalf("o corte efetivo diverge do commit fixado: %q", contractCut)
	}
}

func TestContractCut_ProductionCodeNeverReassignsTheCut(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	assignment := regexp.MustCompile(`contractCut\s*=`)
	declaration := "var contractCut = ContractCutCommit"
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(body), "\n") {
			if assignment.MatchString(line) && strings.TrimSpace(line) != declaration {
				t.Fatalf("%s reatribui o corte fora de teste: %q", name, strings.TrimSpace(line))
			}
		}
	}
}
