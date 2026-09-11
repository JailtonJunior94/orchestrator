package traceability_test

import (
	"os"
	"path/filepath"
	"testing"

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
- criterio um -> comprovado: log-1
- criterio dois -> comprovado: log-2
  - (a) sub-item ilustrativo sem contar como critério próprio
- criterio sem evidencia

## Outra Secao
- item fora da secao -> nao deve ser contado
`)

	got := traceability.NewCatalog().ParseAcceptanceCriteria("1.0", report)

	if len(got) != 3 {
		t.Fatalf("got %d criteria, want 3: %+v", len(got), got)
	}
	if !got[0].HasEvidence() || got[0].Evidence != "comprovado: log-1" {
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
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

## Critérios de Aceite
- criterio um -> comprovado: log-1
`)

	m, err := traceability.NewCatalog().BuildMap(dir)
	if err != nil {
		t.Fatalf("BuildMap: %v", err)
	}
	violations := m.Validate()
	if len(violations) != 0 {
		t.Fatalf("expected zero violations, got %v", violations)
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
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

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
	writeTraceFixture(t, dir, "1.0_execution_report.md", `# Report

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

func TestBuildMap_MissingPRDFile(t *testing.T) {
	dir := t.TempDir()
	writeTraceFixture(t, dir, "tasks.md", "# Tasks\n")

	_, err := traceability.NewCatalog().BuildMap(dir)
	if err == nil {
		t.Fatal("expected error when prd.md is absent")
	}
}

func TestFullPRD_HarnessQuatroClisLoopAprovacao_ChainIsClosedExceptTaskUnderExecution(t *testing.T) {
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

	violations := m.Validate()
	for _, v := range violations {
		if v.Kind == traceability.ViolationTaskWithoutReport && v.Subject == "11.0" {
			continue
		}
		t.Errorf("unexpected traceability violation: %s", v.String())
	}
	if t.Failed() {
		t.Logf("all violations: %v", violations)
	}
}
