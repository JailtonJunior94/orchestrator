package evidence

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func findingsContain(findings []Finding, want string) bool {
	for _, f := range findings {
		if strings.Contains(strings.ToLower(f.Label), strings.ToLower(want)) {
			return true
		}
	}
	return false
}

func TestDetectContract_MarkerSelectsStrictRules(t *testing.T) {
	if contract, _ := DetectContract("# Report"); contract != ContractV1 {
		t.Fatalf("sem marcador = v1, got %v", contract)
	}
	if contract, _ := DetectContract(ContractMarkerV2 + "\n# Report"); contract != ContractV2 {
		t.Fatalf("marcador v2 = v2, got %v", contract)
	}
	contract, findings := DetectContract("<!-- evidence-contract: v9 -->")
	if contract != ContractV2 || !findingsContain(findings, "desconhecida") {
		t.Fatalf("versao desconhecida deve reprovar fechado: %v %v", contract, findings)
	}
}

func TestValidateTask_PortsShellDiffReviewedGates(t *testing.T) {
	result := NewValidator().Validate([]byte("Estado: done\n"), KindTask, nil)
	if result.Pass {
		t.Fatal("relatorio sem evidencia nao pode passar")
	}
	for _, want := range []string{"diff sha", "veredito do reviewer", "tool nao canonica", "coverage delta"} {
		if !findingsContain(result.Findings, want) {
			t.Errorf("faltou gate %q em %v", want, result.Findings)
		}
	}
}

func TestHasHeading_FoldsAccents(t *testing.T) {
	text := "## Validações Executadas\n## Suposições\n"
	if !NewValidator().hasHeading(text, "Validac") {
		t.Error("heading acentuado deve casar com Validac")
	}
	if !NewValidator().hasHeading(text, "Suposic") {
		t.Error("heading acentuado deve casar com Suposic")
	}
}

func TestEvidenceFormProblem_RejectsTrivialRecords(t *testing.T) {
	rejected := []string{"make it work -> done", "make build -> ok", "go test ./... -> n/a", "frobnicate tudo -> saida", "TestFoo -> talvez"}
	for _, evidence := range rejected {
		if evidenceFormProblem(evidence) == "" {
			t.Errorf("evidencia trivial aceita: %q", evidence)
		}
	}
	accepted := []string{"go test ./... -> exit 0", "TestFoo -> pass", "internal/foo.go:42", "bash scripts/x.sh -> 3 arquivos verificados"}
	for _, evidence := range accepted {
		if problem := evidenceFormProblem(evidence); problem != "" {
			t.Errorf("evidencia legitima recusada: %q (%s)", evidence, problem)
		}
	}
}

func TestResolveContract_CutRuleRejectsNewWorkPosingAsHistorical(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.invalid"}, {"config", "user.name", "T"}, {"config", "commit.gpgsign", "false"},
	} {
		if err := exec.Command("git", append([]string{"-C", dir}, args...)...).Run(); err != nil {
			t.Skipf("git indisponivel: %v", err)
		}
	}

	historical := filepath.Join(dir, "historical.md")
	if err := os.WriteFile(historical, []byte("# Report"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "commit", "-qm", "base").Run(); err != nil {
		t.Fatal(err)
	}

	if contract, findings := ResolveContract("# Report", historical); contract != ContractV1 || len(findings) != 0 {
		t.Errorf("relatorio versionado no ref de corte e historico: %v %v", contract, findings)
	}

	fresh := filepath.Join(dir, "fresh.md")
	if err := os.WriteFile(fresh, []byte("# Report"), 0o600); err != nil {
		t.Fatal(err)
	}
	contract, findings := ResolveContract("# Report", fresh)
	if contract != ContractV2 || !findingsContain(findings, "nao e evidencia historica") {
		t.Errorf("relatorio novo sem marcador deve cair em v2: %v %v", contract, findings)
	}
}

func TestValidateTask_HistoricalExemptionCoversFormNotOutcome(t *testing.T) {
	dir := newGitFixture(t)
	taskPath := filepath.Join(dir, "task-1.0.md")
	if err := os.WriteFile(taskPath, []byte(taskFileOneCriterion), 0o600); err != nil {
		t.Fatal(err)
	}

	noMap := strings.Replace(taskComplete, "# Criterios de Aceite\n- Criterio unico -> comprovado: internal/foo.go:1\n", "", 1)
	remarks := strings.Replace(noMap, "verdict=APPROVED", "verdict=APPROVED_WITH_REMARKS", 1)

	approvedPath := filepath.Join(dir, "1.0_execution_report.md")
	remarksPath := filepath.Join(dir, "2.0_execution_report.md")
	if err := os.WriteFile(approvedPath, []byte(noMap), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(remarksPath, []byte(remarks), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir)

	result := NewValidator().ValidateReport([]byte(noMap), approvedPath, KindTask, nil)
	if !result.Pass {
		t.Errorf("v1 isenta o mapa 1:1 de criterios: %v", result.Findings)
	}

	result = NewValidator().ValidateReport([]byte(remarks), remarksPath, KindTask, nil)
	if result.Pass || !findingsContain(result.Findings, "RF-33") {
		t.Errorf("v1 NAO isenta o desfecho: APPROVED_WITH_REMARKS sem severidade declarada deve reprovar (RF-33 fail-closed): %v", result.Findings)
	}
}

func TestValidateTask_RemarksCloseOnlyWithoutBlockingSeverity(t *testing.T) {
	dir := newGitFixture(t)
	taskPath := filepath.Join(dir, "task-1.0.md")
	if err := os.WriteFile(taskPath, []byte(taskFileOneCriterion), 0o600); err != nil {
		t.Fatal(err)
	}

	remarks := "<!-- evidence-contract: v2 -->\n" + strings.Replace(taskComplete, "verdict=APPROVED", "verdict=APPROVED_WITH_REMARKS", 1)
	withMedium := remarks + "\n# Achados\n- [MEDIUM] internal/foo.go:1 nomenclatura\n"
	withHigh := remarks + "\n# Achados\n- [HIGH] internal/foo.go:1 corrida de dados\n"
	withCritical := remarks + "\n# Achados\n- [CRITICAL] internal/foo.go:1 vazamento de segredo\n"

	cases := []struct {
		name string
		body string
		pass bool
		want string
	}{
		{"medium only closes", withMedium, true, ""},
		{"high blocks", withHigh, false, "high/critical"},
		{"critical blocks", withCritical, false, "high/critical"},
		{"no declared severity fails closed", remarks, false, "fail-closed"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, "1.0_execution_report.md")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			result := NewValidator().ValidateReport([]byte(tc.body), path, KindTask, nil)
			if result.Pass != tc.pass {
				t.Fatalf("Pass=%v, want %v: %v", result.Pass, tc.pass, result.Findings)
			}
			if tc.want != "" && !findingsContain(result.Findings, tc.want) {
				t.Fatalf("achados %v nao contem %q", result.Findings, tc.want)
			}
		})
	}
}

func TestResolveContract_AbsenceOfInformationIsNotExemption(t *testing.T) {
	if contract, findings := ResolveContract("# Report", ""); contract != ContractV2 || !findingsContain(findings, "nao e evidencia historica") {
		t.Errorf("caminho vazio deve cair em v2 estrito, got %v %v", contract, findings)
	}

	outside := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(outside, []byte("# Report"), 0o600); err != nil {
		t.Fatal(err)
	}
	if contract, findings := ResolveContract("# Report", outside); contract != ContractV2 || !findingsContain(findings, "nao e evidencia historica") {
		t.Errorf("diretorio sem repositorio git deve cair em v2 estrito, got %v %v", contract, findings)
	}
}

func TestResolveContract_CutRefIsNotEnvironmentControlled(t *testing.T) {
	dir := newGitFixture(t)
	baseline := filepath.Join(dir, "baseline.md")
	if err := os.WriteFile(baseline, []byte("# Report"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir)

	tagged, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "tag", "cut-fixture", strings.TrimSpace(string(tagged))).Run(); err != nil {
		t.Fatal(err)
	}

	fresh := filepath.Join(dir, "fresh.md")
	if err := os.WriteFile(fresh, []byte("# Report"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir)

	t.Setenv("AI_EVIDENCE_CONTRACT_CUT_REF", "cut-fixture")
	if contract, _ := ResolveContract("# Report", fresh); contract != ContractV1 {
		t.Fatalf("o ref de corte e HEAD fixo; o env nao pode estreita-lo, got %v", contract)
	}

	staged := filepath.Join(dir, "staged.md")
	if err := os.WriteFile(staged, []byte("# Report"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "add", "staged.md").Run(); err != nil {
		t.Fatal(err)
	}
	if contract, _ := ResolveContract("# Report", staged); contract != ContractV2 {
		t.Fatalf("relatorio apenas indexado e trabalho novo, nao historico, got %v", contract)
	}
}

func newGitFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"}, {"config", "user.email", "t@example.invalid"}, {"config", "user.name", "T"}, {"config", "commit.gpgsign", "false"},
	} {
		if err := exec.Command("git", append([]string{"-C", dir}, args...)...).Run(); err != nil {
			t.Skipf("git indisponivel: %v", err)
		}
	}
	return dir
}

func commitAll(t *testing.T, dir string) {
	t.Helper()
	if err := exec.Command("git", "-C", dir, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "commit", "-qm", "fixture").Run(); err != nil {
		t.Fatal(err)
	}
}
