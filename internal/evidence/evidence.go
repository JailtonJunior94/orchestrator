package evidence

import (
	"regexp"
	"strings"
)

// ReportKind identifica o tipo de relatorio.
type ReportKind string

const (
	KindTask     ReportKind = "task"
	KindBugfix   ReportKind = "bugfix"
	KindRefactor ReportKind = "refactor"
)

// Finding representa uma secao ou padrao faltante.
type Finding struct {
	Label string // ex: "secao Comandos Executados"
}

// Result e o resultado da validacao.
type Result struct {
	Kind     ReportKind
	Findings []Finding
	Pass     bool
}

// Validate valida um relatorio Markdown conforme o tipo.
// rfIDs e opcional — lista de RF-nn/REQ-nn para rastreabilidade (usado em bugfix).
func Validate(content []byte, kind ReportKind, rfIDs []string) Result {
	text := string(content)
	var findings []Finding

	switch kind {
	case KindTask:
		findings = validateTask(text)
	case KindBugfix:
		findings = validateBugfix(text, rfIDs)
	case KindRefactor:
		findings = validateRefactor(text)
	}

	return Result{
		Kind:     kind,
		Findings: findings,
		Pass:     len(findings) == 0,
	}
}

func hasHeading(text, heading string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, "#")
		trimmed = strings.TrimSpace(trimmed)
		if strings.EqualFold(trimmed, heading) {
			return true
		}
		// partial match for headings with accents/variations
		if strings.Contains(strings.ToLower(trimmed), strings.ToLower(heading)) {
			return true
		}
	}
	return false
}

func matchesRegex(text, pattern string) bool {
	re := regexp.MustCompile(`(?i)` + pattern)
	return re.MatchString(text)
}

func validateTask(text string) []Finding {
	var findings []Finding

	requiredHeadings := []struct {
		label   string
		pattern string
	}{
		{"secao Contexto Carregado", "Contexto Carregado"},
		{"secao Comandos Executados", "Comandos Executados"},
		{"secao Arquivos Alterados", "Arquivos Alterados"},
		{"secao Resultados de Validacao", "Validac"},
		{"secao Suposicoes", "Suposic"},
		{"secao Riscos Residuais", "Riscos Residuais"},
	}

	for _, h := range requiredHeadings {
		if !hasHeading(text, h.pattern) {
			findings = append(findings, Finding{Label: h.label})
		}
	}

	requiredPatterns := []struct {
		label   string
		pattern string
	}{
		{"referencia PRD", `PRD\s*:`},
		{"referencia TechSpec", `TechSpec\s*:`},
		{"estado blocked/failed/done", `estado:\s*(blocked|failed|done)`},
		{"testes pass/fail/blocked", `testes:\s*(pass|fail|blocked)`},
		{"lint pass/fail/blocked", `lint:\s*(pass|fail|blocked)`},
		{"veredito do revisor", `veredito do revisor:\s*(APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED)`},
	}

	for _, p := range requiredPatterns {
		if !matchesRegex(text, p.pattern) {
			findings = append(findings, Finding{Label: p.label})
		}
	}

	// rastreabilidade condicional: se PRD mencionado, RF-nn ou REQ-nn obrigatorio
	if matchesRegex(text, `PRD`) {
		if !matchesRegex(text, `RF-\d+|REQ-\d+`) {
			findings = append(findings, Finding{Label: "rastreabilidade RF-nn ou REQ-nn"})
		}
	}

	return findings
}

func validateBugfix(text string, rfIDs []string) []Finding {
	var findings []Finding

	requiredHeadings := []struct {
		label   string
		pattern string
	}{
		{"secao Bugs", "Bugs"},
		{"secao Comandos Executados", "Comandos Executados"},
		{"secao Riscos Residuais", "Riscos Residuais"},
	}

	for _, h := range requiredHeadings {
		if !hasHeading(text, h.pattern) {
			findings = append(findings, Finding{Label: h.label})
		}
	}

	requiredPatterns := []struct {
		label   string
		pattern string
	}{
		{"Estado fixed/blocked/skipped/failed", `Estado:\s*(fixed|blocked|skipped|failed)`},
		{"Causa raiz", `Causa raiz:`},
		{"Teste de regressao", `Teste de regress`},
		{"Validacao", `Valida`},
		{"Corrigidos contagem", `Corrigidos:\s*\d+`},
		{"Estado done/blocked/failed/needs_input", `Estado:\s*(done|blocked|failed|needs_input)`},
	}

	for _, p := range requiredPatterns {
		if !matchesRegex(text, p.pattern) {
			findings = append(findings, Finding{Label: p.label})
		}
	}

	// rastreabilidade: cada rfID deve aparecer no relatorio
	for _, id := range rfIDs {
		if !strings.Contains(text, id) {
			findings = append(findings, Finding{Label: "rastreabilidade " + id})
		}
	}

	return findings
}

func validateRefactor(text string) []Finding {
	var findings []Finding

	requiredHeadings := []struct {
		label   string
		pattern string
	}{
		{"secao Escopo", "Escopo"},
		{"secao Invariantes", "Invariantes"},
		{"secao Mudancas", "Mudan"},
		{"secao Comandos Executados", "Comandos Executados"},
		{"secao Resultados de Validacao", "Validac"},
		{"secao Riscos Residuais", "Riscos Residuais"},
	}

	for _, h := range requiredHeadings {
		if !hasHeading(text, h.pattern) {
			findings = append(findings, Finding{Label: h.label})
		}
	}

	requiredPatterns := []struct {
		label   string
		pattern string
	}{
		{"Modo advisory/execution", `Modo:\s*(advisory|execution)`},
		{"Estado needs_input/blocked/failed/done", `Estado:\s*(needs_input|blocked|failed|done)`},
		{"Testes pass/fail/blocked/n/a", `Testes:\s*(pass|fail|blocked|n/a)`},
		{"Lint pass/fail/blocked/n/a", `Lint:\s*(pass|fail|blocked|n/a)`},
	}

	for _, p := range requiredPatterns {
		if !matchesRegex(text, p.pattern) {
			findings = append(findings, Finding{Label: p.label})
		}
	}

	// condicional: Modo execution exige Veredito do Revisor
	if matchesRegex(text, `Modo:\s*execution`) {
		if !matchesRegex(text, `Veredito do Revisor:\s*(APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED|n/a)`) {
			findings = append(findings, Finding{Label: "Veredito do Revisor obrigatorio em Modo execution"})
		}
	}

	return findings
}
