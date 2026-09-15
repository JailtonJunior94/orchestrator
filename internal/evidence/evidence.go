package evidence

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/reviewverdict"
)

// ReportKind identifica o tipo de relatorio.
type ReportKind string

const (
	KindTask     ReportKind = "task"
	KindBugfix   ReportKind = "bugfix"
	KindRefactor ReportKind = "refactor"
	KindReview   ReportKind = "review"
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

func (r1 *Validator) Validate(content []byte, kind ReportKind, rfIDs []string) Result {
	return NewValidator().ValidateReport(content, "", kind, rfIDs)
}

func (r1 *Validator) ValidateReport(content []byte, reportPath string, kind ReportKind, rfIDs []string) Result {
	text := string(content)
	var findings []Finding

	switch kind {
	case KindTask:
		findings = NewValidator().validateTask(text, reportPath)
	case KindBugfix:
		findings = NewValidator().validateBugfix(text, rfIDs)
	case KindRefactor:
		findings = NewValidator().validateRefactor(text)
	case KindReview:
		findings = NewValidator().validateReview(text, reportPath)
	}

	return Result{
		Kind:     kind,
		Findings: findings,
		Pass:     len(findings) == 0,
	}
}

func (r1 *Validator) hasHeading(text, heading string) bool {
	for _, line := range strings.Split(text, "\n") {
		trimmed := fold(strings.TrimLeft(line, "#"))
		if trimmed == fold(heading) || strings.Contains(trimmed, fold(heading)) {
			return true
		}
	}
	return false
}

func (r1 *Validator) matchesRegex(text, pattern string) bool {
	re := regexp.MustCompile(`(?i)` + pattern)
	return re.MatchString(text)
}

func (r1 *Validator) validateTask(text, reportPath string) []Finding {
	contract, findings := ResolveContract(text, reportPath)

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
		if !NewValidator().hasHeading(text, h.pattern) {
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
		if !NewValidator().matchesRegex(text, p.pattern) {
			findings = append(findings, Finding{Label: p.label})
		}
	}

	if NewValidator().matchesRegex(text, `PRD`) {
		if !NewValidator().matchesRegex(text, `RF-\d+|REQ-\d+`) {
			findings = append(findings, Finding{Label: "rastreabilidade RF-nn ou REQ-nn"})
		}
	}

	findings = append(findings, NewValidator().strongTestProofFindings(text)...)
	findings = append(findings, NewValidator().diffReviewedFindings(text)...)
	return append(findings, NewValidator().acceptanceCriteriaFindings(text, reportPath, contract)...)
}

func (r1 *Validator) validateBugfix(text string, rfIDs []string) []Finding {
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
		if !NewValidator().hasHeading(text, h.pattern) {
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
		{"Estado done/blocked/failed/needs_input", `Estado(?: final)?:\s*(done|blocked|failed|needs_input)`},
	}

	for _, p := range requiredPatterns {
		if !NewValidator().matchesRegex(text, p.pattern) {
			findings = append(findings, Finding{Label: p.label})
		}
	}

	blocks := NewValidator().bugBlocks(text)
	if len(blocks) == 0 {
		findings = append(findings, Finding{Label: "blocos individuais de bugs"})
	} else {
		fixed := 0
		tests := 0
		for _, block := range blocks {
			id := block["id"]
			for _, field := range []string{"severidade", "origem", "estado", "causa raiz", "arquivos alterados", "teste de regressao", "validacao"} {
				if strings.TrimSpace(block[field]) == "" {
					findings = append(findings, Finding{Label: field + " no bloco " + id})
				}
			}
			if block["estado"] == "fixed" {
				fixed++
			}
			if block["teste de regressao"] != "" {
				tests++
			}
		}
		findings = append(findings, NewValidator().reconcileBugfixTotals(text, len(blocks), fixed, tests)...)
	}

	// rastreabilidade: cada rfID deve aparecer no relatorio
	for _, id := range rfIDs {
		if !strings.Contains(text, id) {
			findings = append(findings, Finding{Label: "rastreabilidade " + id})
		}
	}

	return findings
}

func (r1 *Validator) bugBlocks(text string) []map[string]string {
	lines := strings.Split(text, "\n")
	var blocks []map[string]string
	var current map[string]string
	inBugs := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(strings.TrimSpace(strings.TrimLeft(trimmed, "#")), "Bugs") {
			inBugs = true
			continue
		}
		if inBugs && strings.HasPrefix(trimmed, "#") {
			break
		}
		if !inBugs || !strings.HasPrefix(trimmed, "-") {
			continue
		}
		field, value, found := strings.Cut(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), ":")
		if !found {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(field))
		if key == "id" {
			current = make(map[string]string)
			blocks = append(blocks, current)
		}
		if current != nil {
			current[key] = strings.TrimSpace(value)
		}
	}
	return blocks
}

func (r1 *Validator) reconcileBugfixTotals(text string, total, fixed, tests int) []Finding {
	var findings []Finding
	for _, expected := range []struct {
		label string
		value int
	}{
		{label: "Total de bugs no escopo", value: total},
		{label: "Corrigidos", value: fixed},
		{label: "Testes de regressao adicionados", value: tests},
	} {
		match := regexp.MustCompile(`(?im)^-\s*` + regexp.QuoteMeta(expected.label) + `\s*:\s*(\d+)\s*$`).FindStringSubmatch(text)
		if len(match) != 2 {
			findings = append(findings, Finding{Label: "totalizador " + expected.label})
			continue
		}
		value, err := strconv.Atoi(match[1])
		if err != nil || value != expected.value {
			findings = append(findings, Finding{Label: "totalizador " + expected.label + " diverge dos blocos"})
		}
	}
	return findings
}

var (
	reviewTargetRe       = regexp.MustCompile(`(?i)(diff|branch|commit|arquivos? revisad)`)
	reviewNoFindingsRe   = regexp.MustCompile(`(?i)sem achados`)
	reviewSeverityRe     = regexp.MustCompile(`(?i)(severidade\s*:\s*(critical|high|medium|low|cr(i|í)tico|alta|m(e|é)dia|baixa)|severity\s*:\s*(critical|high|medium|low))`)
	reviewHighSeverityRe = regexp.MustCompile(`(?i)(severidade\s*:\s*(critical|high|cr(i|í)tico|alta)|severity\s*:\s*(critical|high))`)
)

func (r1 *Validator) reviewVerdict(text string) string {
	verdict, declared := reviewverdict.ParseDocument(text)
	if !declared {
		return ""
	}
	return verdict
}

func (r1 *Validator) validateReview(text, reportPath string) []Finding {
	var findings []Finding

	verdict := NewValidator().reviewVerdict(text)
	if verdict == "" {
		findings = append(findings, Finding{Label: "veredito canonico do review"})
	}

	requiredHeadings := []struct {
		label   string
		pattern string
	}{
		{"secao Achados", "Achados"},
		{"secao Arquivos Revisados", "Arquivos Revisados"},
		{"secao Riscos Residuais", "Riscos Residuais"},
		{"secao Validacoes Executadas", "Validac"},
	}
	for _, h := range requiredHeadings {
		if !NewValidator().hasHeading(text, h.pattern) {
			findings = append(findings, Finding{Label: h.label})
		}
	}

	findings = append(findings, NewValidator().validateReviewCoherence(text, verdict)...)
	return append(findings, NewValidator().validateCriteriaMap(text, verdict, reportPath)...)
}

func (r1 *Validator) validateReviewCoherence(text, verdict string) []Finding {
	var findings []Finding

	if !reviewTargetRe.MatchString(text) {
		findings = append(findings, Finding{Label: "referencia ao alvo revisado (diff/branch/commit/arquivos)"})
	}
	if !reviewNoFindingsRe.MatchString(text) && !reviewSeverityRe.MatchString(text) {
		findings = append(findings, Finding{Label: "severidade canonica em ao menos um achado (critical|high|medium|low) ou declaracao 'Sem achados'"})
	}
	if verdict == "REJECTED" && !reviewHighSeverityRe.MatchString(text) {
		findings = append(findings, Finding{Label: "veredito REJECTED exige ao menos um achado de severidade critical ou high comprovado"})
	}

	return findings
}

var (
	criteriaMapHeadingRe = regexp.MustCompile(`(?i)mapa de crit(e|é)rios de aceite`)
	criteriaLineRe       = regexp.MustCompile(`^-\s*\[`)
	criteriaMarkerRe     = regexp.MustCompile(`^-\s*\[([^\]]*)\]`)
	evidenceFileLineRe   = regexp.MustCompile(`[A-Za-z0-9_./-]+:[0-9]+`)
	reviewedHeadingRe    = regexp.MustCompile(`(?i)^#+\s+arquivos revisados`)
	pathTokenRe          = regexp.MustCompile(`[A-Za-z0-9_][A-Za-z0-9_./-]*\.[A-Za-z0-9]+`)
)

func (r1 *Validator) reviewedFilePaths(lines []string) map[string]struct{} {
	paths := make(map[string]struct{})
	capture := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if reviewedHeadingRe.MatchString(trimmed) {
			capture = true
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			capture = false
			continue
		}
		if !capture {
			continue
		}
		for _, token := range pathTokenRe.FindAllString(trimmed, -1) {
			if normalized := normalizeReviewedPath(token); normalized != "" {
				paths[normalized] = struct{}{}
			}
		}
	}
	return paths
}

func normalizeReviewedPath(path string) string {
	trimmed := strings.Trim(strings.TrimSpace(path), "`\"'")
	trimmed = strings.TrimPrefix(trimmed, "./")
	return strings.TrimPrefix(trimmed, "/")
}

func samePathOrSuffix(declared, reference string) bool {
	if declared == reference {
		return true
	}
	return strings.HasSuffix(declared, "/"+reference) || strings.HasSuffix(reference, "/"+declared)
}

func (r1 *Validator) referencedFileIsReviewed(reviewed map[string]struct{}, evidence string) bool {
	reference := evidenceFileLineRe.FindString(evidence)
	if reference == "" {
		return true
	}
	cut := strings.LastIndex(reference, ":")
	if cut <= 0 {
		return false
	}
	target := normalizeReviewedPath(reference[:cut])
	if target == "" {
		return false
	}
	for declared := range reviewed {
		if samePathOrSuffix(declared, target) {
			return true
		}
	}
	return false
}

func (r1 *Validator) validateCriteriaMap(text, verdict, reportPath string) []Finding {
	lines := strings.Split(text, "\n")
	headingAt := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") && criteriaMapHeadingRe.MatchString(trimmed) {
			headingAt = i
			break
		}
	}
	if headingAt == -1 {
		return []Finding{{Label: "secao Mapa de Criterios de Aceite"}}
	}

	var findings []Finding
	reviewed := NewValidator().reviewedFilePaths(lines)
	if len(reviewed) == 0 {
		findings = append(findings, Finding{Label: "secao Arquivos Revisados sem nenhum arquivo listado"})
	}
	criteria := 0
	for _, line := range lines[headingAt+1:] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			break
		}
		if !criteriaLineRe.MatchString(trimmed) {
			continue
		}
		criteria++

		marker := ""
		if m := criteriaMarkerRe.FindStringSubmatch(trimmed); m != nil {
			marker = strings.ToLower(strings.TrimSpace(m[1]))
		}
		evidence := ""
		if idx := strings.Index(trimmed, "->"); idx != -1 {
			evidence = strings.TrimSpace(trimmed[idx+2:])
		}

		switch marker {
		case "atendido":
		case "nao atendido", "não atendido":
			if verdict == "APPROVED" || verdict == "APPROVED_WITH_REMARKS" {
				findings = append(findings, Finding{Label: "criterio nao atendido proibe veredito aprovador: " + trimmed})
			}
		case "nao verificavel", "nao verificável", "não verificavel", "não verificável":
			findings = append(findings, Finding{Label: "criterio nao verificavel proibe APPROVED: " + trimmed})
		default:
			findings = append(findings, Finding{Label: "marcador de criterio invalido no mapa 1:1: " + trimmed})
		}

		if evidence == "" {
			findings = append(findings, Finding{Label: "criterio sem linha de evidencia no mapa 1:1: " + trimmed})
			continue
		}

		if evidenceFileLineRe.MatchString(evidence) {
			if !NewValidator().referencedFileIsReviewed(reviewed, evidence) {
				findings = append(findings, Finding{Label: "evidencia arquivo:linha fora dos arquivos revisados: " + trimmed})
			}
		} else if problem := evidenceFormProblem(evidence); problem != "" {
			findings = append(findings, Finding{Label: problem + ": " + trimmed})
		}
	}

	if criteria == 0 {
		findings = append(findings, Finding{Label: "mapa 1:1 sem nenhuma linha de criterio"})
	}

	taskPath := resolveReviewTaskPath(text, reportPath)
	if taskPath == "" {
		return append(findings, Finding{Label: "task file nao resolvivel para confronto 1:1 do mapa de criterios: declare '- Task file: <caminho>' apontando para a task revisada"})
	}
	declared := countTaskCriteria(taskPath)
	if declared == 0 {
		return append(findings, Finding{Label: "task file (" + taskPath + ") nao declara nenhum criterio de aceite — mapa 1:1 nao confrontavel"})
	}
	if criteria < declared {
		return append(findings, Finding{Label: "mapa 1:1 incompleto — " + strconv.Itoa(criteria) + " linha(s) de criterio no review para " + strconv.Itoa(declared) + " criterio(s) definido(s) em " + taskPath})
	}
	return findings
}

func (r1 *Validator) validateRefactor(text string) []Finding {
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
		if !NewValidator().hasHeading(text, h.pattern) {
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
		if !NewValidator().matchesRegex(text, p.pattern) {
			findings = append(findings, Finding{Label: p.label})
		}
	}

	// condicional: Modo execution exige Veredito do Revisor
	if NewValidator().matchesRegex(text, `Modo:\s*execution`) {
		if !NewValidator().matchesRegex(text, `Veredito do Revisor:\s*(APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED|n/a)`) {
			findings = append(findings, Finding{Label: "Veredito do Revisor obrigatorio em Modo execution"})
		}
	}

	return findings
}
