package evidence

import (
	"strings"
	"testing"
)

// ── Task ─────────────────────────────────────────────────────────────────────

const taskComplete = `# Contexto Carregado
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
`

const taskEmpty = ``

const taskPartial = `# Contexto Carregado
PRD: sim
RF-01

# Comandos Executados
`

func TestValidateTask_Complete(t *testing.T) {
	r := NewValidator().Validate([]byte(taskComplete), KindTask, nil)
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
	// Relatório completo SEM a seção de métricas → deve passar (ausência não bloqueia).
	r := NewValidator().Validate([]byte(taskComplete), KindTask, nil)
	if !r.Pass {
		t.Errorf("Pass deve ser true sem seção Métricas Claude-2026; findings: %v", r.Findings)
	}
}

func TestValidateTask_ClaudeMetricsSection_Present_DoesNotBlock(t *testing.T) {
	// Relatório completo COM a seção de métricas → deve continuar passando.
	withMetrics := taskComplete + `
## Métricas Claude-2026
| Métrica | Valor |
|---|---|
| cache_read_tokens | 150 |
| cache_creation_tokens | 300 |
| thinking_tokens | 42 |
| tool_calls_normalized | 5 |
`
	r := NewValidator().Validate([]byte(withMetrics), KindTask, nil)
	if !r.Pass {
		t.Errorf("Pass deve ser true com seção Métricas Claude-2026; findings: %v", r.Findings)
	}
}

// ── Métricas Gemini-2026 — validador permissivo (F4-Gemini, RF-20) ────────────
// A seção "Métricas Gemini-2026" é OPCIONAL no execution_report.md.
// Presença ou ausência não deve alterar o resultado de Pass=true para relatório completo.

// T-37: TestEvidenceRendersGeminiMetricsSection — relatório com Gemini* não-zero renderiza tabela
func TestEvidenceRendersGeminiMetricsSection_Present_DoesNotBlock(t *testing.T) {
	// Relatório completo COM a seção de métricas Gemini → deve continuar passando.
	withGeminiMetrics := taskComplete + `
## Métricas Gemini-2026
| Métrica | Valor |
|---|---|
| cache_read_tokens | 100 |
| effective_context_tokens | 200 |
| prompt_tokens_billed | 300 |
| thoughts_tokens | 0 |
`
	r := NewValidator().Validate([]byte(withGeminiMetrics), KindTask, nil)
	if !r.Pass {
		t.Errorf("Pass deve ser true com seção Métricas Gemini-2026; findings: %v", r.Findings)
	}
}

// T-38: TestEvidenceMissingGeminiMetricsDoesNotBlock — ausência da seção Gemini não bloqueia
func TestEvidenceMissingGeminiMetricsDoesNotBlock(t *testing.T) {
	// Relatório completo SEM a seção de métricas Gemini → deve passar (ausência não bloqueia).
	r := NewValidator().Validate([]byte(taskComplete), KindTask, nil)
	if !r.Pass {
		t.Errorf("Pass deve ser true sem seção Métricas Gemini-2026; findings: %v", r.Findings)
	}
	// Garantir que nenhum finding menciona Gemini.
	for _, f := range r.Findings {
		if f.Label == "secao Métricas Gemini-2026" {
			t.Error("nao esperado finding de 'secao Métricas Gemini-2026' — seção é opcional (RF-20)")
		}
	}
}
