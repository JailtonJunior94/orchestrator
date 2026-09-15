package traceability_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/evidence"
	"github.com/JailtonJunior94/ai-spec-harness/internal/traceability"
)

func TestExtractRequirementIDs_UniqueOrdered(t *testing.T) {
	prd := []byte("RF-01 e RF-02 devem existir. RF-01 aparece de novo. rf-03 em minusculas.")

	got := traceability.NewCatalog().ExtractRequirementIDs(prd)

	want := []string{"RF-01", "RF-02", "RF-03"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("got[%d]=%s, want %s", i, got[i], w)
		}
	}
}

func TestParseCoverageTable_MapsTaskToRequirements(t *testing.T) {
	tasks := []byte(`# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|--------|-------------------|
| 1.0 | RF-01, RF-02 |
| 2.0 | RF-03 |

## Outra Secao

| Tarefa | Ignorado |
|---|---|
| 9.9 | RF-99 |
`)

	got := traceability.NewCatalog().ParseCoverageTable(tasks)

	if len(got["1.0"]) != 2 {
		t.Fatalf("task 1.0 = %v, want 2 RFs", got["1.0"])
	}
	if len(got["2.0"]) != 1 || got["2.0"][0] != "RF-03" {
		t.Fatalf("task 2.0 = %v, want [RF-03]", got["2.0"])
	}
	if _, ok := got["9.9"]; ok {
		t.Fatal("table outside 'Cobertura de Requisitos' must not be parsed")
	}
}

func TestParseAcceptanceCriteria_ArrowSeparatesTextFromEvidence(t *testing.T) {
	report := []byte(`# Report

## Critérios de Aceite
- criterio um -> go test ./... -> PASS
- criterio dois -> TestCriterion -> PASS
  - (a) sub-item ilustrativo sem contar como critério próprio
- criterio sem evidencia

## Outra Secao
- item fora da secao -> nao deve ser contado
`)

	got := traceability.NewCatalog().ParseAcceptanceCriteria("1.0", report)

	if len(got) != 3 {
		t.Fatalf("got %d criteria, want 3: %+v", len(got), got)
	}
	if !got[0].HasEvidence() || got[0].Evidence != "go test ./... -> PASS" {
		t.Fatalf("got[0]=%+v", got[0])
	}
	if !got[1].HasEvidence() {
		t.Fatalf("got[1]=%+v, want evidence", got[1])
	}
	if got[2].HasEvidence() {
		t.Fatalf("got[2]=%+v, want no evidence (no '->')", got[2])
	}
}

func writeTraceFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestBuildMapAndValidate_FullChainPasses(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir. RF-02 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01, RF-02 |
`)
	writeTraceFixture(t, dir, "task-1.0.md", `# Task 1.0

## Critérios de Sucesso
- criterio um
`)
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

## Tarefa
- Arquivo: task-1.0.md

## Critérios de Aceite
- criterio um -> go test ./... -> PASS
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	violations := m.Validate()
	if len(violations) != 0 {
		t.Fatalf("expected zero violations, got %v", violations)
	}
	if m.VerifiedTaskCount() != 1 {
		t.Fatalf("VerifiedTaskCount = %d, want 1", m.VerifiedTaskCount())
	}
}

func TestValidate_RequirementWithoutTask(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir. RF-02 deve existir mas nenhuma tarefa cobre.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)
	writeTraceFixture(t, dir, "task-1.0.md", "# Task\n\n## Critérios de Sucesso\n- criterio um\n")
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

## Tarefa
- Arquivo: task-1.0.md

## Critérios de Aceite
- criterio um -> comprovado: log-1
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	violations := m.Validate()

	found := false
	for _, v := range violations {
		if v.Kind == traceability.ViolationRequirementWithoutTask && v.Subject == "RF-02" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected requirement_without_task for RF-02, got %v", violations)
	}
}

func TestValidate_TaskWithoutReport(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	violations := m.Validate()

	found := false
	for _, v := range violations {
		if v.Kind == traceability.ViolationTaskWithoutReport && v.Subject == "1.0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected task_without_report for 1.0, got %v", violations)
	}
}

func TestValidate_TaskWithoutCriteria(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)
	writeTraceFixture(t, dir, "1.0_execution_report.md", "# Report sem a secao de criterios\n")

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	violations := m.Validate()

	found := false
	for _, v := range violations {
		if v.Kind == traceability.ViolationTaskWithoutCriteria && v.Subject == "1.0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected task_without_criteria for 1.0, got %v", violations)
	}
}

func TestValidate_CriterionWithoutEvidence(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)
	writeTraceFixture(t, dir, "task-1.0.md", "# Task\n\n## Critérios de Sucesso\n- criterio um\n")
	writeTraceFixture(t, dir, "1.0_execution_report.md", `<!-- evidence-contract: v2 -->
# Report

## Tarefa
- Arquivo: task-1.0.md

## Critérios de Aceite
- criterio sem seta de evidencia
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	violations := m.Validate()

	found := false
	for _, v := range violations {
		if v.Kind == traceability.ViolationCriterionWithoutEvidence {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected criterion_without_evidence, got %v", violations)
	}
}

func TestValidate_BlockedTaskWithWrittenReportIsStillCharged(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

| # | Título | Status |
|---|---|---|
| 1.0 | Native proof | blocked |

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)
	writeTraceFixture(t, dir, "task-1.0.md", "# Task\n\n## Critérios de Sucesso\n- criterio um\n")
	writeTraceFixture(t, dir, "1.0_execution_report.md", `<!-- evidence-contract: v2 -->
# Report

## Tarefa
- Arquivo: task-1.0.md

## Critérios de Aceite
- Bloqueado aguardando execução nativa.
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	violations, notices := m.Report()

	found := false
	for _, v := range violations {
		if v.Kind == traceability.ViolationCriterionWithoutEvidence {
			found = true
		}
	}
	if !found {
		t.Fatalf("blocked task with a written report must still be charged for evidence, got %v", violations)
	}
	for _, n := range notices {
		if n.Kind == traceability.NoticeTaskBlocked && strings.Contains(n.Detail, "fora do escopo") {
			t.Fatalf("task_blocked must not read as an exemption: %v", n)
		}
	}
	if m.VerifiedTaskCount() != 1 {
		t.Fatalf("VerifiedTaskCount = %d, want 1 (blocked task is in the confronted universe)", m.VerifiedTaskCount())
	}
}

func TestValidate_HistoricalContractDoesNotExemptCriteriaMap(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)
	writeTraceFixture(t, dir, "task-1.0.md", "# Task\n\n## Critérios de Sucesso\n- criterio um\n- criterio dois\n")
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

## Tarefa
- Arquivo: task-1.0.md

## Critérios de Aceite
- criterio um -> comprovado: prosa livre aceita sob v1
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	m.Contracts = map[string]evidence.Contract{"1.0": evidence.ContractV1}

	violations, notices := m.Report()

	found := false
	for _, v := range violations {
		if v.Kind == traceability.ViolationCriteriaMapIncomplete && v.Subject == "1.0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("v1 exemption must not cover the 1:1 criteria map, got %v", violations)
	}
	for _, n := range notices {
		if n.Kind == traceability.NoticeHistoricalEvidence {
			t.Fatalf("map breach must be a violation, not swallowed by the historical notice: %v", n)
		}
	}
}

func TestValidate_HistoricalContractKeepsEvidenceFormExempt(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)
	writeTraceFixture(t, dir, "task-1.0.md", "# Task\n\n## Critérios de Sucesso\n- criterio um\n")
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

## Tarefa
- Arquivo: task-1.0.md

## Critérios de Aceite
- criterio um -> comprovado: prosa livre aceita sob v1
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	m.Contracts = map[string]evidence.Contract{"1.0": evidence.ContractV1}

	violations, notices := m.Report()
	for _, v := range violations {
		if v.Kind == traceability.ViolationCriterionWithoutEvidence {
			t.Fatalf("v1 exemption must still cover the strict evidence form: %v", v)
		}
	}
	found := false
	for _, n := range notices {
		if n.Kind == traceability.NoticeHistoricalEvidence && n.Subject == "1.0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected historical_evidence_contract notice, got %v", notices)
	}
	if m.VerifiedTaskCount() != 0 {
		t.Fatalf("VerifiedTaskCount = %d, want 0 (evidence form was never confronted)", m.VerifiedTaskCount())
	}
}

func TestValidate_UnresolvableTaskFileBreaksTheMap(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		reference  string
		taskFile   string
		wantDetail string
	}{
		{name: "no reference at all", reference: "", wantDetail: "nao declara task file resolvivel"},
		{name: "reference points nowhere", reference: "- Arquivo: task-inexistente.md\n", wantDetail: "nao existe"},
		{name: "task file declares no criteria", reference: "- Arquivo: task-1.0.md\n", taskFile: "# Task\n\nSem secao de criterios.\n", wantDetail: "nao declara nenhum criterio"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
			writeTraceFixture(t, dir, "tasks.md", "# Tasks\n\n## Cobertura de Requisitos\n\n| Tarefa | Requisitos cobertos |\n|---|---|\n| 1.0 | RF-01 |\n")
			if tc.taskFile != "" {
				writeTraceFixture(t, dir, "task-1.0.md", tc.taskFile)
			}
			writeTraceFixture(t, dir, "1.0_execution_report.md",
				"<!-- evidence-contract: v2 -->\n# Report\n\n## Tarefa\n"+tc.reference+"\n## Critérios de Aceite\n- criterio um -> go test ./... -> PASS\n")

			m, err := traceability.NewCatalog().BuildMap(dir)
			if err != nil {
				t.Fatalf("BuildMap: %v", err)
			}
			violations := m.Validate()

			found := false
			for _, v := range violations {
				if v.Kind == traceability.ViolationCriteriaMapIncomplete && strings.Contains(v.Detail, tc.wantDetail) {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected criteria_map_incomplete containing %q, got %v", tc.wantDetail, violations)
			}
			if m.VerifiedTaskCount() != 0 {
				t.Fatalf("VerifiedTaskCount = %d, want 0 (map was not confrontable)", m.VerifiedTaskCount())
			}
			foundVacuous := false
			for _, v := range violations {
				if v.Kind == traceability.ViolationGateVacuous {
					foundVacuous = true
				}
			}
			if !foundVacuous {
				t.Fatalf("a universe with no confronted task must also report gate_vacuous, got %v", violations)
			}
		})
	}
}

func TestCountTaskCriteria_OnlyInsideCriteriaSection(t *testing.T) {
	t.Parallel()

	task := []byte("# Task\n\n## Contexto\n- nao e criterio\n\n## Critérios de Sucesso\n- [ ] criterio um\n- criterio dois\n  - sub-item tambem conta\n\n```\n- dentro de fence nao conta\n```\n\n## Notas\n- nao e criterio\n")

	if got := traceability.NewCatalog().CountTaskCriteria(task); got != 3 {
		t.Fatalf("CountTaskCriteria = %d, want 3", got)
	}
}

func TestValidate_UniverseFullyExemptIsNotVerification(t *testing.T) {
	dir := t.TempDir()

	writeTraceFixture(t, dir, "prd.md", "RF-01 deve existir.")
	writeTraceFixture(t, dir, "tasks.md", `# Tasks

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
`)
	writeTraceFixture(t, dir, "task-1.0.md", "# Task\n\n## Critérios de Sucesso\n- criterio\n")
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

## Tarefa
- Arquivo: task-1.0.md

## Critérios de Aceite
- criterio -> go test ./... -> PASS
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}

	m.Contracts = map[string]evidence.Contract{"1.0": evidence.ContractV1}
	violations := m.Validate()
	if len(violations) != 1 || violations[0].Kind != traceability.ViolationGateVacuous {
		t.Fatalf("universo integralmente isento deve reprovar como vacuo, got %v", violations)
	}
	if m.VerifiedTaskCount() != 0 {
		t.Fatalf("nenhuma tarefa foi confrontada, VerifiedTaskCount = %d", m.VerifiedTaskCount())
	}

	m.Contracts = map[string]evidence.Contract{"1.0": evidence.ContractV2}
	if violations := m.Validate(); len(violations) != 0 {
		t.Fatalf("tarefa confrontada nao e gate vacuo: %v", violations)
	}
}

func TestValidate_CriterionRejectsArbitraryEvidenceProse(t *testing.T) {
	m := traceability.Map{
		TaskCoverage:   map[string][]string{"1.0": {"RF-01"}},
		Criteria:       map[string][]traceability.Criterion{"1.0": {{Task: "1.0", Text: "criterion", Evidence: "trust me"}}},
		MissingReports: map[string]bool{},
		TaskSources:    map[string]traceability.TaskSource{"1.0": {Reference: "task-1.0.md", Path: "task-1.0.md", CriteriaCount: 1}},
	}
	violations := m.Validate()
	if len(violations) != 1 || violations[0].Kind != traceability.ViolationCriterionWithoutEvidence {
		t.Fatalf("violations = %v, want criterion_without_evidence", violations)
	}
}

func TestCriterionHasEvidenceAcceptsOnlyRF48Forms(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		evidence string
		want     bool
	}{
		{name: "command with output", evidence: "go test ./... -> PASS", want: true},
		{name: "file line", evidence: "internal/foo.go:42", want: true},
		{name: "named test with result", evidence: "TestCriterion -> PASS", want: true},
		{name: "prose with spaces", evidence: "trust me completely", want: false},
		{name: "unnamed command-looking prose", evidence: "it worked -> PASS", want: false},
		{name: "command without output", evidence: "go test ./...", want: false},
		{name: "test without result", evidence: "TestCriterion", want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			criterion := traceability.Criterion{Evidence: tc.evidence}
			if got := criterion.HasEvidence(); got != tc.want {
				t.Errorf("HasEvidence(%q) = %t, want %t", tc.evidence, got, tc.want)
			}
		})
	}
}

func TestBuildMap_MissingPRDFile(t *testing.T) {
	dir := t.TempDir()
	writeTraceFixture(t, dir, "tasks.md", "# Tasks\n")

	_, err := traceability.NewCatalog().BuildMap(dir)
	if err == nil {
		t.Fatal("expected error when prd.md is absent")
	}
}

func TestFullPRD_HarnessQuatroClisLoopAprovacao_ReportsRealState(t *testing.T) {
	dir := "../../.specs/prd-harness-quatro-clis-loop-aprovacao"
	if _, err := os.Stat(filepath.Join(dir, "prd.md")); err != nil {
		t.Skip("PRD directory not present in this checkout")
	}

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	if len(m.Requirements) != 63 {
		t.Fatalf("expected 63 requirements in prd.md, got %d", len(m.Requirements))
	}

	blockedWithReport := 0
	for task, status := range m.TaskStatuses {
		if status == "blocked" && !m.MissingReports[task] {
			blockedWithReport++
		}
	}
	if blockedWithReport == 0 {
		t.Fatal("fixture drift: this PRD is expected to carry blocked tasks with written reports")
	}

	violations := m.Validate()
	kinds := make(map[traceability.ViolationKind]int)
	for _, v := range violations {
		kinds[v.Kind]++
	}
	for _, v := range violations {
		if v.Kind != traceability.ViolationGateVacuous && v.Kind != traceability.ViolationCriteriaMapIncomplete {
			t.Errorf("unexpected traceability violation: %s", v.String())
		}
	}
	if kinds[traceability.ViolationGateVacuous] != 1 {
		t.Errorf("every report here is contract v1, so no task has its evidence form confronted: gate_vacuous = %d, want 1", kinds[traceability.ViolationGateVacuous])
	}
	if m.VerifiedTaskCount() != 0 {
		t.Errorf("VerifiedTaskCount = %d, want 0 while the whole universe is contract v1", m.VerifiedTaskCount())
	}
	t.Logf("estado real: %d tarefa(s) no escopo, %d blocked com relatorio, %d confrontada(s), violacoes=%v",
		len(m.TaskCoverage), blockedWithReport, m.VerifiedTaskCount(), kinds)
}
