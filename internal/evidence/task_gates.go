package evidence

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	diffShaRe       = regexp.MustCompile(`(?m)^sha=\s*([0-9a-fA-F]{40}(?:[0-9a-fA-F]{24})?)\s*$`)
	diffVerdictRe   = regexp.MustCompile(`(?m)^verdict=\s*(APPROVED_WITH_REMARKS|APPROVED|REJECTED|BLOCKED)\s*$`)
	diffToolRe      = regexp.MustCompile(`(?m)^tool=\s*(claude|codex|copilot|opencode)\s*$`)
	coverageDeltaRe = regexp.MustCompile(`(?m)^delta=\s*([+-]?[0-9]+(?:\.[0-9]+)?)%\s*$`)
	taskFileRefRe   = regexp.MustCompile(`(?mi)^-\s*Arquivo\s*:\s*(.+?)\s*$`)
	reviewTaskRefRe = regexp.MustCompile(`(?mi)^-\s*(?:task\s*file|arquivo da task)\s*:\s*(.+?)\s*$`)
	stateDoneRe     = regexp.MustCompile(`(?i)estado\s*:\s*done`)
	testesPassRe    = regexp.MustCompile(`(?i)testes\s*:\s*pass`)
	testCommandRe   = regexp.MustCompile(`(?i)(go test|gotestsum|pytest|unittest|npm (run )?test|yarn test|pnpm test|jest|vitest|mocha|make test|make integration|cargo test|dotnet test|ctest|rspec|phpunit)`)
	criteriaHeadRe  = regexp.MustCompile(`(?i)^#+\s+crit(?:e|é)rios de aceite`)
	taskCriteriaRe  = regexp.MustCompile(`(?i)^#+\s+(?:crit(?:e|é)rios de (?:sucesso|aceite)|definition of done|acceptance criteria)`)
	provenRe        = regexp.MustCompile(`(?i)->\s*comprovado\s*:\s*\S`)
	emptyProofRe    = regexp.MustCompile(`(?i)comprovado\s*:\s*(\[ev|\[evid|\[\]\s*$)`)
)

func (r1 *Validator) diffReviewedFindings(text string) []Finding {
	var findings []Finding

	if diffShaRe.FindStringSubmatch(text) == nil {
		findings = append(findings, Finding{Label: "missing diff sha imutavel (sha= deve ter 40 ou 64 hexadecimais)"})
	}

	switch verdict := diffVerdictRe.FindStringSubmatch(text); {
	case verdict == nil:
		findings = append(findings, Finding{Label: "veredito do reviewer no bloco Diff Reviewed"})
	case verdict[1] != "APPROVED":
		findings = append(findings, Finding{Label: "veredito do reviewer nao encerra o ciclo de aprovacao: " + verdict[1] + " (RF-53: a isencao historica cobre a forma da evidencia, nunca o desfecho; somente APPROVED encerra)"})
	}

	if diffToolRe.FindStringSubmatch(text) == nil {
		findings = append(findings, Finding{Label: "tool nao canonica ou ausente no bloco Diff Reviewed"})
	}

	delta := coverageDeltaRe.FindStringSubmatch(text)
	if delta == nil {
		return append(findings, Finding{Label: "coverage delta ausente ou invalido"})
	}
	if value, err := strconv.ParseFloat(delta[1], 64); err == nil && value < 0 {
		findings = append(findings, Finding{Label: "coverage regression detectada (delta=" + delta[1] + "%)"})
	}
	return findings
}

func (r1 *Validator) strongTestProofFindings(text string) []Finding {
	if !testesPassRe.MatchString(text) {
		return nil
	}
	if testCommandRe.MatchString(sectionBody(text, "comandos executados")) {
		return nil
	}
	return []Finding{{Label: "'Testes: pass' declarado sem comando de teste correspondente em '## Comandos Executados' (prova fraca)"}}
}

func sectionBody(text, heading string) string {
	var body []string
	capture := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			capture = fold(strings.TrimSpace(strings.TrimLeft(trimmed, "#"))) == fold(heading)
			continue
		}
		if capture {
			body = append(body, line)
		}
	}
	return strings.Join(body, "\n")
}

func resolveReviewTaskPath(text, reportPath string) string {
	match := reviewTaskRefRe.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	reference := strings.TrimSpace(match[1])
	if reference == "" || strings.Contains(reference, "<") || strings.HasPrefix(strings.ToLower(reference), "n/a") {
		return ""
	}
	if info, err := os.Stat(reference); err == nil && !info.IsDir() {
		return reference
	}
	if reportPath == "" {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(reportPath), reference)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

func resolveTaskPath(text, reportPath string) string {
	match := taskFileRefRe.FindStringSubmatch(text)
	if match == nil {
		return ""
	}
	reference := strings.TrimSpace(match[1])
	if reference == "" || strings.Contains(reference, "<slug>") || strings.HasPrefix(strings.ToLower(reference), "n/a") {
		return ""
	}
	if info, err := os.Stat(reference); err == nil && !info.IsDir() {
		return reference
	}
	if reportPath == "" {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(reportPath), reference)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

func countTaskCriteria(taskPath string) int {
	content, err := os.ReadFile(taskPath)
	if err != nil {
		return 0
	}
	count := 0
	capture := false
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			capture = taskCriteriaRe.MatchString(trimmed)
			continue
		}
		if !capture || !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		item := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		if idx := strings.Index(item, "]"); strings.HasPrefix(item, "[") && idx > 0 {
			item = strings.TrimSpace(item[idx+1:])
		}
		if item != "" {
			count++
		}
	}
	return count
}

func countProvenCriteria(text string) int {
	count := 0
	capture := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			capture = criteriaHeadRe.MatchString(trimmed)
			continue
		}
		if capture && provenRe.MatchString(trimmed) && !emptyProofRe.MatchString(trimmed) {
			count++
		}
	}
	return count
}

func (r1 *Validator) acceptanceCriteriaFindings(text, reportPath string, contract Contract) []Finding {
	if contract == ContractV1 || !stateDoneRe.MatchString(text) {
		return nil
	}

	reportHasCriteria := false
	for _, line := range strings.Split(text, "\n") {
		if criteriaHeadRe.MatchString(strings.TrimSpace(line)) {
			reportHasCriteria = true
			break
		}
	}

	taskPath := resolveTaskPath(text, reportPath)
	if taskPath == "" {
		if reportHasCriteria {
			return []Finding{{Label: "relatorio declara '## Criterios de Aceite' mas nao ha task file resolvivel para confronto 1:1 (RF-53)"}}
		}
		return []Finding{{Label: "relatorio declara 'done' mas nao ha task file resolvivel (campo 'Arquivo:') para confronto 1:1 dos criterios (RF-51/RF-53)"}}
	}

	defined := countTaskCriteria(taskPath)
	if defined == 0 {
		return []Finding{{Label: "task file (" + taskPath + ") nao declara nenhum criterio de aceite — mapa 1:1 nao confrontavel (RF-53)"}}
	}
	if !reportHasCriteria {
		return []Finding{{Label: "secao '## Criterios de Aceite' no relatorio (task define " + strconv.Itoa(defined) + " criterio(s))"}}
	}
	if proven := countProvenCriteria(text); proven < defined {
		return []Finding{{Label: "criterios de aceite comprovados (" + strconv.Itoa(proven) + ") < definidos na task (" + strconv.Itoa(defined) + ")"}}
	}
	return nil
}
