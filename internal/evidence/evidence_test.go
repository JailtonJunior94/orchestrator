package evidence

import (
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
	r := Validate([]byte(taskComplete), KindTask, nil)
	if !r.Pass {
		t.Errorf("esperado Pass=true, findings: %v", r.Findings)
	}
	if len(r.Findings) != 0 {
		t.Errorf("esperado 0 findings, got %d: %v", len(r.Findings), r.Findings)
	}
}

func TestValidateTask_Empty(t *testing.T) {
	r := Validate([]byte(taskEmpty), KindTask, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio vazio")
	}
	if len(r.Findings) == 0 {
		t.Error("esperado findings nao vazios para relatorio vazio")
	}
}

func TestValidateTask_Partial(t *testing.T) {
	r := Validate([]byte(taskPartial), KindTask, nil)
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
	r := Validate([]byte(content), KindTask, nil)
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
Bug-01: crash no startup

# Comandos Executados
go test ./...

Estado: fixed
Causa raiz: nil pointer
Teste de regressao: adicionado
Validacao: ok
Corrigidos: 1
Estado: done

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
	r := Validate([]byte(bugfixComplete), KindBugfix, nil)
	if !r.Pass {
		t.Errorf("esperado Pass=true, findings: %v", r.Findings)
	}
}

func TestValidateBugfix_Empty(t *testing.T) {
	r := Validate([]byte(bugfixEmpty), KindBugfix, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio vazio")
	}
	if len(r.Findings) == 0 {
		t.Error("esperado findings nao vazios")
	}
}

func TestValidateBugfix_Partial(t *testing.T) {
	r := Validate([]byte(bugfixPartial), KindBugfix, nil)
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
	r := Validate([]byte(bugfixComplete), KindBugfix, []string{"RF-01"})
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
	r2 := Validate([]byte(withRF), KindBugfix, []string{"RF-01"})
	for _, f := range r2.Findings {
		if f.Label == "rastreabilidade RF-01" {
			t.Error("nao esperado finding de rastreabilidade RF-01 — ID presente no relatorio")
		}
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
	r := Validate([]byte(refactorComplete), KindRefactor, nil)
	if !r.Pass {
		t.Errorf("esperado Pass=true, findings: %v", r.Findings)
	}
}

func TestValidateRefactor_Execution_Complete(t *testing.T) {
	r := Validate([]byte(refactorExecution), KindRefactor, nil)
	if !r.Pass {
		t.Errorf("esperado Pass=true para execution com veredito, findings: %v", r.Findings)
	}
}

func TestValidateRefactor_Empty(t *testing.T) {
	r := Validate([]byte(refactorEmpty), KindRefactor, nil)
	if r.Pass {
		t.Error("esperado Pass=false para relatorio vazio")
	}
	if len(r.Findings) == 0 {
		t.Error("esperado findings nao vazios")
	}
}

func TestValidateRefactor_Execution_MissingVeredito(t *testing.T) {
	r := Validate([]byte(refactorMissingVeredito), KindRefactor, nil)
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
	r := Validate([]byte(refactorComplete), KindRefactor, nil)
	for _, f := range r.Findings {
		if f.Label == "Veredito do Revisor obrigatorio em Modo execution" {
			t.Error("nao esperado finding de veredito em Modo advisory")
		}
	}
	_ = r
}

// ── Kind check ────────────────────────────────────────────────────────────────

func TestValidate_KindPreserved(t *testing.T) {
	r := Validate([]byte(taskComplete), KindTask, nil)
	if r.Kind != KindTask {
		t.Errorf("esperado Kind=%s, got %s", KindTask, r.Kind)
	}
}
