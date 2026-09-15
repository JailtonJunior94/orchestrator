package evidence

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ── Task ─────────────────────────────────────────────────────────────────────

const taskComplete = `# Tarefa
- ID: 1.0
- Arquivo: task-1.0.md

# Contexto Carregado
PRD: sim
TechSpec: sim
RF-01, REQ-02

# Comandos Executados
go test ./...

# Arquivos Alterados
internal/foo.go

# Resultados de Validacao
Estado: done
Testes: pass
Lint: pass
Veredito do Revisor: APPROVED

# Suposicoes
nenhuma

# Riscos Residuais
nenhum

# Criterios de Aceite
- Criterio unico -> comprovado: internal/foo.go:1

# Diff Reviewed
sha=5e2268d17a23d82b6a1b07694c060245a33d76fe89d7942ef6fd4d2be29326df
verdict=APPROVED
tool=claude

# Coverage
delta=+0.5%
`

const taskFileOneCriterion = `# Tarefa 1.0

## Criterios de Sucesso

- Criterio unico.
`

func validateTaskReport(t *testing.T, body string) Result {
	t.Helper()
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "task-1.0.md")
	if err := os.WriteFile(taskPath, []byte(taskFileOneCriterion), 0o600); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(dir, "1.0_execution_report.md")
	content := []byte(ContractMarkerV2 + "\n" + body)
	if err := os.WriteFile(reportPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	return NewValidator().ValidateReport(content, reportPath, KindTask, nil)
}

const taskEmpty = ``

const taskPartial = `# Contexto Carregado
PRD: sim
RF-01

# Comandos Executados
`

func TestValidateTask_Complete(t *testing.T) {
	r := validateTaskReport(t, taskComplete)
	if !r.Pass {
		t.Errorf("esperado Pass=true, findings: %v", r.Findings)
	}
	if len(r.Findings) != 0 {
		t.Errorf("esperado 0 findings, got %d: %v", len(r.Findings), r.Findings)
	}
}

func TestValidateTask_Empty(t *testing.T) {
	r := NewValidator().Validate([]byte(taskEmpty), KindTask, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio vazio")
	}
	if len(r.Findings) == 0 {
		t.Error("esperado findings nao vazios para relatorio vazio")
	}
}

func TestValidateTask_Partial(t *testing.T) {
	r := NewValidator().Validate([]byte(taskPartial), KindTask, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio parcial")
	}
	// deve ter findings faltantes (Arquivos Alterados, Validacao, Suposicoes, etc.)
	if len(r.Findings) == 0 {
		t.Error("esperado findings para relatorio parcial")
	}
	// nao deve ter finding de Contexto Carregado (existe) nem Comandos Executados (existe)
	for _, f := range r.Findings {
		if f.Label == "secao Contexto Carregado" {
			t.Error("nao esperado finding de 'secao Contexto Carregado' — secao existe")
		}
		if f.Label == "secao Comandos Executados" {
			t.Error("nao esperado finding de 'secao Comandos Executados' — secao existe")
		}
	}
}

func TestValidateTask_TraceabilityRequired(t *testing.T) {
	// PRD mencionado mas sem RF-nn ou REQ-nn
	content := `# Contexto Carregado
PRD: sim
TechSpec: sim

# Comandos Executados
# Arquivos Alterados
# Resultados de Validacao
Estado: done
Testes: pass
Lint: pass
Veredito do Revisor: APPROVED
# Suposicoes
# Riscos Residuais
`
	r := NewValidator().Validate([]byte(content), KindTask, nil)
	found := false
	for _, f := range r.Findings {
		if f.Label == "rastreabilidade RF-nn ou REQ-nn" {
			found = true
		}
	}
	if !found {
		t.Error("esperado finding de rastreabilidade quando PRD mencionado sem RF-nn/REQ-nn")
	}
}

// ── Bugfix ────────────────────────────────────────────────────────────────────

const bugfixComplete = `# Bugs
- ID: Bug-01
- Severidade: major
- Origem: issue #1
- Estado: fixed
- Causa raiz: nil pointer
- Arquivos alterados: internal/a.go
- Teste de regressao: adicionado
- Validacao: ok

# Comandos Executados
go test ./...

- Total de bugs no escopo: 1
- Corrigidos: 1
- Testes de regressao adicionados: 1
- Estado final: done

# Riscos Residuais
nenhum
`

const bugfixEmpty = ``

const bugfixPartial = `# Bugs
Bug-01

# Comandos Executados
Estado: fixed
Causa raiz: nil pointer
`

func TestValidateBugfix_Complete(t *testing.T) {
	r := NewValidator().Validate([]byte(bugfixComplete), KindBugfix, nil)
	if !r.Pass {
		t.Errorf("esperado Pass=true, findings: %v", r.Findings)
	}
}

func TestValidateBugfix_Empty(t *testing.T) {
	r := NewValidator().Validate([]byte(bugfixEmpty), KindBugfix, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio vazio")
	}
	if len(r.Findings) == 0 {
		t.Error("esperado findings nao vazios")
	}
}

func TestValidateBugfix_Partial(t *testing.T) {
	r := NewValidator().Validate([]byte(bugfixPartial), KindBugfix, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio parcial")
	}
	for _, f := range r.Findings {
		if f.Label == "secao Bugs" {
			t.Error("nao esperado finding de 'secao Bugs' — secao existe")
		}
		if f.Label == "secao Comandos Executados" {
			t.Error("nao esperado finding de 'secao Comandos Executados' — secao existe")
		}
	}
}

func TestValidateBugfix_Traceability(t *testing.T) {
	r := NewValidator().Validate([]byte(bugfixComplete), KindBugfix, []string{"RF-01"})
	found := false
	for _, f := range r.Findings {
		if f.Label == "rastreabilidade RF-01" {
			found = true
		}
	}
	if !found {
		t.Error("esperado finding de rastreabilidade RF-01 ausente no relatorio")
	}

	// com RF-01 presente no conteudo
	withRF := bugfixComplete + "\nRF-01\n"
	r2 := NewValidator().Validate([]byte(withRF), KindBugfix, []string{"RF-01"})
	for _, f := range r2.Findings {
		if f.Label == "rastreabilidade RF-01" {
			t.Error("nao esperado finding de rastreabilidade RF-01 — ID presente no relatorio")
		}
	}
}

func TestValidateBugfixRejectsIncompleteBlockAndWrongTotals(t *testing.T) {
	content := strings.Replace(bugfixComplete, "- Causa raiz: nil pointer\n", "", 1)
	content = strings.Replace(content, "- Corrigidos: 1", "- Corrigidos: 0", 1)
	result := NewValidator().Validate([]byte(content), KindBugfix, nil)
	if result.Pass {
		t.Fatal("bloco incompleto e totalizador divergente deveriam falhar")
	}
	labels := make(map[string]bool, len(result.Findings))
	for _, finding := range result.Findings {
		labels[finding.Label] = true
	}
	if !labels["causa raiz no bloco Bug-01"] || !labels["totalizador Corrigidos diverge dos blocos"] {
		t.Fatalf("findings inesperados: %#v", result.Findings)
	}
}

// ── Refactor ──────────────────────────────────────────────────────────────────

const refactorComplete = `# Escopo
refator do modulo X

# Invariantes
sem quebra de API

# Mudancas
renomear funcoes

# Comandos Executados
go test ./...

# Resultados de Validacao
Modo: advisory
Estado: done
Testes: pass
Lint: pass

# Riscos Residuais
nenhum
`

const refactorExecution = `# Escopo
refator do modulo X

# Invariantes
sem quebra de API

# Mudancas
renomear funcoes

# Comandos Executados
go test ./...

# Resultados de Validacao
Modo: execution
Estado: done
Testes: pass
Lint: pass
Veredito do Revisor: APPROVED

# Riscos Residuais
nenhum
`

const refactorEmpty = ``

const refactorMissingVeredito = `# Escopo
# Invariantes
# Mudancas
# Comandos Executados
# Resultados de Validacao
Modo: execution
Estado: done
Testes: pass
Lint: pass
# Riscos Residuais
`

func TestValidateRefactor_Complete(t *testing.T) {
	r := NewValidator().Validate([]byte(refactorComplete), KindRefactor, nil)
	if !r.Pass {
		t.Errorf("esperado Pass=true, findings: %v", r.Findings)
	}
}

func TestValidateRefactor_Execution_Complete(t *testing.T) {
	r := NewValidator().Validate([]byte(refactorExecution), KindRefactor, nil)
	if !r.Pass {
		t.Errorf("esperado Pass=true para execution com veredito, findings: %v", r.Findings)
	}
}

func TestValidateRefactor_Empty(t *testing.T) {
	r := NewValidator().Validate([]byte(refactorEmpty), KindRefactor, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio vazio")
	}
	if len(r.Findings) == 0 {
		t.Error("esperado findings nao vazios")
	}
}

func TestValidateRefactor_Execution_MissingVeredito(t *testing.T) {
	r := NewValidator().Validate([]byte(refactorMissingVeredito), KindRefactor, nil)
	found := false
	for _, f := range r.Findings {
		if f.Label == "Veredito do Revisor obrigatorio em Modo execution" {
			found = true
		}
	}
	if !found {
		t.Error("esperado finding de Veredito do Revisor quando Modo: execution sem veredito")
	}
}

func TestValidateRefactor_Advisory_NoVeredito(t *testing.T) {
	r := NewValidator().Validate([]byte(refactorComplete), KindRefactor, nil)
	for _, f := range r.Findings {
		if f.Label == "Veredito do Revisor obrigatorio em Modo execution" {
			t.Error("nao esperado finding de veredito em Modo advisory")
		}
	}
	_ = r
}

// ── Review — mapa 1:1 criterio -> evidencia (RF-47, RF-48, RF-49, RF-51, RF-52) ──

const reviewFixtureTaskFile = "testdata/task-review-fixture.md"

func reviewWithMap(mapSection string) string {
	return `# Relatorio de Review
- Veredito: APPROVED
- Alvo revisado: diff
- Task file: ` + reviewFixtureTaskFile + `
` + mapSection + `
## Achados
Sem achados.
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
## Validacoes Executadas
- go test ./... -> ok
`
}

func hasFindingContaining(findings []Finding, needle string) bool {
	for _, f := range findings {
		if strings.Contains(f.Label, needle) {
			return true
		}
	}
	return false
}

func TestValidateReview_CriteriaMap(t *testing.T) {
	validMap := `## Mapa de Criterios de Aceite
- [atendido] Criterio um -> go test ./... -> PASS
- [atendido] Criterio dois -> internal/foo.go:42`

	cases := []struct {
		name    string
		content string
		want    bool
		needle  string
	}{
		{
			name:    "mapa ausente",
			content: reviewWithMap(""),
			want:    false,
			needle:  "secao Mapa de Criterios de Aceite",
		},
		{
			name: "mapa incompleto criterio sem evidencia",
			content: reviewWithMap(`## Mapa de Criterios de Aceite
- [atendido] Criterio um -> internal/foo.go:1
- [atendido] Criterio dois`),
			want:   false,
			needle: "criterio sem linha de evidencia",
		},
		{
			name: "criterio nao verificavel",
			content: reviewWithMap(`## Mapa de Criterios de Aceite
- [nao verificavel] Criterio um -> internal/foo.go:1`),
			want:   false,
			needle: "nao verificavel proibe APPROVED",
		},
		{
			name: "linha de evidencia invalida",
			content: reviewWithMap(`## Mapa de Criterios de Aceite
- [atendido] Criterio um -> porque confio no autor da mudanca`),
			want:   false,
			needle: "fora das tres formas de RF-48",
		},
		{
			name:    "mapa completo e valido",
			content: reviewWithMap(validMap),
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewValidator().Validate([]byte(tc.content), KindReview, nil)
			if r.Pass != tc.want {
				t.Fatalf("Pass=%v, esperado %v; findings: %v", r.Pass, tc.want, r.Findings)
			}
			if tc.needle != "" && !hasFindingContaining(r.Findings, tc.needle) {
				t.Fatalf("esperado finding contendo %q; findings: %v", tc.needle, r.Findings)
			}
		})
	}
}

func reviewReport(verdict, target, findingsSection, mapSection string) string {
	return `# Relatorio de Review
- Veredito: ` + verdict + `
` + target + `
- Task file: ` + reviewFixtureTaskFile + `
` + mapSection + `
## Achados
` + findingsSection + `
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
## Validacoes Executadas
- go test ./... -> ok
`
}

func TestValidateReview_ApprovedWithUnmetCriterion(t *testing.T) {
	cases := []struct {
		name    string
		verdict string
		marker  string
		want    bool
	}{
		{"APPROVED com criterio nao atendido", "APPROVED", "nao atendido", false},
		{"APPROVED com criterio nao atendido acentuado", "APPROVED", "não atendido", false},
		{"APPROVED com criterio nao verificavel", "APPROVED", "nao verificavel", false},
		{"APPROVED com criterio atendido", "APPROVED", "atendido", true},
		{"REJECTED com criterio nao atendido", "REJECTED", "nao atendido", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findingsSection := "Sem achados."
			if tc.verdict == "REJECTED" {
				findingsSection = "- Severidade: high\n- criterio um nao coberto"
			}
			content := reviewReport(
				tc.verdict,
				"- Alvo revisado: diff",
				findingsSection,
				"## Mapa de Criterios de Aceite\n- ["+tc.marker+"] Criterio um -> internal/foo.go:42",
			)
			r := NewValidator().Validate([]byte(content), KindReview, nil)
			if r.Pass != tc.want {
				t.Fatalf("Pass=%v, esperado %v; findings: %v", r.Pass, tc.want, r.Findings)
			}
		})
	}
}

func TestValidateReview_CoherenceRules(t *testing.T) {
	validMap := "## Mapa de Criterios de Aceite\n- [atendido] Criterio um -> internal/foo.go:42"

	cases := []struct {
		name    string
		content string
		want    bool
		needle  string
	}{
		{
			name:    "sem referencia ao alvo revisado",
			content: "# Relatorio de Review\n- Veredito: APPROVED\n" + validMap + "\n## Achados\nSem achados.\n## Riscos Residuais\n- nenhum\n## Validacoes Executadas\n- go test -> ok\n",
			want:    false,
			needle:  "referencia ao alvo revisado",
		},
		{
			name:    "achados sem severidade canonica",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "- algo estranho no handler", validMap),
			want:    false,
			needle:  "severidade canonica",
		},
		{
			name:    "achados com severidade canonica",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "- Severidade: low\n- nota menor", validMap),
			want:    true,
		},
		{
			name:    "REJECTED sem achado critical ou high",
			content: reviewReport("REJECTED", "- Alvo revisado: diff", "- Severidade: low\n- nota menor", validMap),
			want:    false,
			needle:  "REJECTED exige ao menos um achado",
		},
		{
			name:    "REJECTED com achado high",
			content: reviewReport("REJECTED", "- Alvo revisado: diff", "- Severidade: high\n- bug real", validMap),
			want:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewValidator().Validate([]byte(tc.content), KindReview, nil)
			if r.Pass != tc.want {
				t.Fatalf("Pass=%v, esperado %v; findings: %v", r.Pass, tc.want, r.Findings)
			}
			if tc.needle != "" && !hasFindingContaining(r.Findings, tc.needle) {
				t.Fatalf("esperado finding contendo %q; findings: %v", tc.needle, r.Findings)
			}
		})
	}
}

func TestValidateReview_ParityWithShellValidator(t *testing.T) {
	shell, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash indisponivel")
	}
	validator := filepath.Join("..", "..", ".agents", "scripts", "validate-review-evidence.sh")
	if _, err := os.Stat(validator); err != nil {
		t.Skipf("validador shell ausente: %v", err)
	}

	validMap := `## Mapa de Criterios de Aceite
- [atendido] Criterio um -> go test ./... -> PASS
- [atendido] Criterio dois -> internal/foo.go:42`

	scenarios := []struct {
		name          string
		content       string
		wantGo        bool
		shellKnownGap bool
	}{
		{
			name:    "mapa ausente",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "Sem achados.", ""),
			wantGo:  false,
		},
		{
			name:    "mapa incompleto",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "Sem achados.", "## Mapa de Criterios de Aceite\n- [atendido] Criterio um -> internal/foo.go:1\n- [atendido] Criterio dois"),
			wantGo:  false,
		},
		{
			name:    "criterio nao verificavel",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "Sem achados.", "## Mapa de Criterios de Aceite\n- [nao verificavel] Criterio um -> internal/foo.go:1"),
			wantGo:  false,
		},
		{
			name:    "evidencia invalida",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "Sem achados.", "## Mapa de Criterios de Aceite\n- [atendido] Criterio um -> porque confio no autor"),
			wantGo:  false,
		},
		{
			name:    "mapa valido com veredito APPROVED",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "Sem achados.", validMap),
			wantGo:  true,
		},
		{
			name:    "achados sem severidade canonica",
			content: reviewReport("APPROVED", "- Alvo revisado: diff", "- algo estranho no handler", validMap),
			wantGo:  false,
		},
		{
			name:    "REJECTED sem achado critical ou high",
			content: reviewReport("REJECTED", "- Alvo revisado: diff", "- Severidade: low\n- nota menor", validMap),
			wantGo:  false,
		},
		{
			name:    "REJECTED com achado high",
			content: reviewReport("REJECTED", "- Alvo revisado: diff", "- Severidade: high\n- bug real", validMap),
			wantGo:  true,
		},
		{
			name:    "APPROVED_WITH_REMARKS com achado medium",
			content: reviewReport("APPROVED_WITH_REMARKS", "- Alvo revisado: branch feat/x", "- Severidade: medium\n- nota", validMap),
			wantGo:  true,
		},
		{
			name:    "sem veredito canonico",
			content: reviewReport("talvez", "- Alvo revisado: diff", "Sem achados.", validMap),
			wantGo:  false,
		},
		{
			name:          "APPROVED com criterio nao atendido (gap conhecido do espelho shell)",
			content:       reviewReport("APPROVED", "- Alvo revisado: diff", "Sem achados.", "## Mapa de Criterios de Aceite\n- [nao atendido] Criterio um -> internal/foo.go:42"),
			wantGo:        false,
			shellKnownGap: true,
		},
	}

	dir := t.TempDir()
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			path := filepath.Join(dir, "review.md")
			if err := os.WriteFile(path, []byte(sc.content), 0o644); err != nil {
				t.Fatal(err)
			}

			result := NewValidator().Validate([]byte(sc.content), KindReview, nil)
			if result.Pass != sc.wantGo {
				t.Fatalf("validador Go pass=%v, esperado %v; findings: %v", result.Pass, sc.wantGo, result.Findings)
			}

			cmd := exec.Command(shell, validator, path)
			cmd.Env = append(os.Environ(), "LC_ALL=C")
			shellPass := cmd.Run() == nil

			if result.Pass && !shellPass {
				t.Fatalf("paridade quebrada: go aceita relatorio que o shell reprova")
			}
			if !sc.shellKnownGap && result.Pass != shellPass {
				t.Fatalf("paridade quebrada: go pass=%v, shell pass=%v", result.Pass, shellPass)
			}
		})
	}
}

func TestValidateReview_KindPreserved(t *testing.T) {
	r := NewValidator().Validate([]byte(reviewWithMap("## Mapa de Criterios de Aceite\n- [atendido] C -> internal/foo.go:1")), KindReview, nil)
	if r.Kind != KindReview {
		t.Errorf("esperado Kind=%s, got %s", KindReview, r.Kind)
	}
}

// ── Kind check ────────────────────────────────────────────────────────────────

func TestValidate_KindPreserved(t *testing.T) {
	r := NewValidator().Validate([]byte(taskComplete), KindTask, nil)
	if r.Kind != KindTask {
		t.Errorf("esperado Kind=%s, got %s", KindTask, r.Kind)
	}
}

// ── Métricas Claude-2026 — validador permissivo (F4-Claude) ──────────────────
// A seção "Métricas Claude-2026" é OPCIONAL no execution_report.md.
// Presença ou ausência não deve alterar o resultado de Pass=true para relatório completo.

func TestValidateTask_ClaudeMetricsSection_Absent_DoesNotBlock(t *testing.T) {
	r := validateTaskReport(t, taskComplete)
	if !r.Pass {
		t.Errorf("Pass deve ser true sem seção Métricas Claude-2026; findings: %v", r.Findings)
	}
}

func TestValidateTask_ClaudeMetricsSection_Present_DoesNotBlock(t *testing.T) {
	withMetrics := taskComplete + `
## Métricas Claude-2026
| Métrica | Valor |
|---|---|
| cache_read_tokens | 150 |
| cache_creation_tokens | 300 |
| thinking_tokens | 42 |
| tool_calls_normalized | 5 |
`
	r := validateTaskReport(t, withMetrics)
	if !r.Pass {
		t.Errorf("Pass deve ser true com seção Métricas Claude-2026; findings: %v", r.Findings)
	}
}

// ── Métricas por driver — validador permissivo ────────────────────────────────
// Uma seção extra de métricas por driver (ex: "Métricas OpenCode") é OPCIONAL no
// execution_report.md. Presença ou ausência não deve alterar o resultado de
// Pass=true para relatório completo.

func TestEvidenceRendersDriverMetricsSection_Present_DoesNotBlock(t *testing.T) {
	withDriverMetrics := taskComplete + `
## Métricas OpenCode
| Métrica | Valor |
|---|---|
| cache_read_tokens | 100 |
`
	r := validateTaskReport(t, withDriverMetrics)
	if !r.Pass {
		t.Errorf("Pass deve ser true com seção de métricas extra; findings: %v", r.Findings)
	}
}

func TestEvidenceMissingDriverMetricsDoesNotBlock(t *testing.T) {
	r := validateTaskReport(t, taskComplete)
	if !r.Pass {
		t.Errorf("Pass deve ser true sem seção de métricas extra; findings: %v", r.Findings)
	}
}
