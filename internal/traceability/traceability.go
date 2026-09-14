package traceability

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/evidence"
)

type Catalog struct{}

func NewCatalog() *Catalog {
	return &Catalog{}
}

type Criterion struct {
	Task     string
	Text     string
	Evidence string
}

func (c Criterion) HasEvidence() bool {
	evidence := strings.TrimSpace(c.Evidence)
	return fileLineEvidenceRegex.MatchString(evidence) || testEvidenceRegex.MatchString(evidence) || commandEvidenceRegex.MatchString(evidence)
}

type ViolationKind string

const (
	ViolationRequirementWithoutTask   ViolationKind = "requirement_without_task"
	ViolationTaskWithoutReport        ViolationKind = "task_without_report"
	ViolationTaskWithoutCriteria      ViolationKind = "task_without_criteria"
	ViolationCriterionWithoutEvidence ViolationKind = "criterion_without_evidence"
	ViolationGateVacuous              ViolationKind = "gate_vacuous"
)

type NoticeKind string

const (
	NoticeTaskBlocked        NoticeKind = "task_blocked"
	NoticeHistoricalEvidence NoticeKind = "historical_evidence_contract"
)

type Notice struct {
	Kind    NoticeKind
	Subject string
	Detail  string
}

func (n Notice) String() string {
	return fmt.Sprintf("%s: %s — %s", n.Kind, n.Subject, n.Detail)
}

type Violation struct {
	Kind    ViolationKind
	Subject string
	Detail  string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s: %s — %s", v.Kind, v.Subject, v.Detail)
}

type Map struct {
	Requirements   []string
	TaskCoverage   map[string][]string
	TaskStatuses   map[string]string
	RequirementMap map[string][]string
	Criteria       map[string][]Criterion
	MissingReports map[string]bool
	Contracts      map[string]evidence.Contract
}

func (c *Catalog) ParseTaskStatuses(tasksContent []byte) map[string]string {
	lines := strings.Split(string(tasksContent), "\n")
	statuses := make(map[string]string)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		columns := splitTableRow(trimmed)
		if len(columns) < 3 || isSeparatorRow(columns) || !taskIDRegex.MatchString(columns[0]) {
			continue
		}
		statuses[columns[0]] = strings.ToLower(columns[2])
	}
	return statuses
}

var requirementIDRegex = regexp.MustCompile(`(?i)RF-\d+`)

var taskIDRegex = regexp.MustCompile(`^\d+\.\d+$`)

var (
	fileLineEvidenceRegex = regexp.MustCompile(`^[^\s:]+:[0-9]+$`)
	testEvidenceRegex     = regexp.MustCompile(`^(Test[A-Za-z0-9_]+|teste\s+.+)\s+->\s+.+$`)
	commandEvidenceRegex  = regexp.MustCompile(`^(go (test|build|vet)\b|gotestsum\b|bash\b|sh\b|make\b|grep\b|python\b|pytest\b|npm\b|pnpm\b|yarn\b|cargo\b|dotnet\b|\./).+\s+->\s+.+$`)
)

func (c *Catalog) ExtractRequirementIDs(prdContent []byte) []string {
	matches := requirementIDRegex.FindAllString(string(prdContent), -1)
	seen := make(map[string]struct{})
	var out []string
	for _, m := range matches {
		upper := strings.ToUpper(m)
		if _, ok := seen[upper]; ok {
			continue
		}
		seen[upper] = struct{}{}
		out = append(out, upper)
	}
	return out
}

func splitTableRow(row string) []string {
	trimmed := strings.Trim(row, "|")
	parts := strings.Split(trimmed, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

func isSeparatorRow(cols []string) bool {
	for _, c := range cols {
		stripped := strings.Trim(c, "-: ")
		if stripped != "" {
			return false
		}
	}
	return true
}

func (c *Catalog) ParseCoverageTable(tasksContent []byte) map[string][]string {
	lines := strings.Split(string(tasksContent), "\n")
	inSection := false
	headerSeen := false
	result := make(map[string][]string)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if inSection {
				break
			}
			inSection = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), "Cobertura de Requisitos")
			headerSeen = false
			continue
		}
		if !inSection || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cols := splitTableRow(trimmed)
		if len(cols) < 2 {
			continue
		}
		if !headerSeen {
			headerSeen = true
			continue
		}
		if isSeparatorRow(cols) {
			continue
		}
		task := cols[0]
		if task == "" {
			continue
		}
		rfs := requirementIDRegex.FindAllString(cols[1], -1)
		for _, rf := range rfs {
			result[task] = append(result[task], strings.ToUpper(rf))
		}
	}
	return result
}

var acceptanceCriteriaHeading = "Critérios de Aceite"

var fencedCodeBlockDelimiter = "```"

func (c *Catalog) ParseAcceptanceCriteria(task string, reportContent []byte) []Criterion {
	lines := strings.Split(string(reportContent), "\n")
	inSection := false
	inFence := false
	var out []Criterion

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, fencedCodeBlockDelimiter) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			if inSection {
				break
			}
			inSection = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")), acceptanceCriteriaHeading)
			continue
		}
		if !inSection || !strings.HasPrefix(line, "- ") {
			continue
		}
		item := strings.TrimPrefix(trimmed, "- ")
		criterionText, evidenceText, hasArrow := strings.Cut(item, "->")
		crit := Criterion{Task: task, Text: strings.TrimSpace(criterionText)}
		if hasArrow {
			crit.Evidence = strings.TrimSpace(evidenceText)
		}
		out = append(out, crit)
	}
	return out
}

func (c *Catalog) BuildMap(dir string) (Map, error) {
	prdContent, err := os.ReadFile(filepath.Join(dir, "prd.md"))
	if err != nil {
		return Map{}, fmt.Errorf("ler prd.md: %w", err)
	}
	tasksContent, err := os.ReadFile(filepath.Join(dir, "tasks.md"))
	if err != nil {
		return Map{}, fmt.Errorf("ler tasks.md: %w", err)
	}

	requirements := c.ExtractRequirementIDs(prdContent)
	taskCoverage := c.ParseCoverageTable(tasksContent)
	taskStatuses := c.ParseTaskStatuses(tasksContent)

	requirementMap := make(map[string][]string)
	for task, rfs := range taskCoverage {
		for _, rf := range rfs {
			requirementMap[rf] = append(requirementMap[rf], task)
		}
	}

	criteria := make(map[string][]Criterion)
	missingReports := make(map[string]bool)
	contracts := make(map[string]evidence.Contract)
	for task := range taskCoverage {
		reportPath := filepath.Join(dir, fmt.Sprintf("%s_execution_report.md", task))
		reportContent, err := os.ReadFile(reportPath)
		if err != nil {
			missingReports[task] = true
			continue
		}
		contract, _ := evidence.ResolveContract(string(reportContent), reportPath)
		contracts[task] = contract
		criteria[task] = NewCatalog().ParseAcceptanceCriteria(task, reportContent)
	}

	return Map{
		Requirements:   requirements,
		TaskCoverage:   taskCoverage,
		TaskStatuses:   taskStatuses,
		RequirementMap: requirementMap,
		Criteria:       criteria,
		MissingReports: missingReports,
		Contracts:      contracts,
	}, nil
}

func (m Map) Validate() []Violation {
	violations, _ := m.Report()
	return violations
}

func (m Map) VerifiedTaskCount() int {
	count := 0
	for task := range m.TaskCoverage {
		if m.taskIsVerifiable(task) {
			count++
		}
	}
	return count
}

func (m Map) taskIsVerifiable(task string) bool {
	if m.MissingReports[task] || m.TaskStatuses[task] == "blocked" {
		return false
	}
	if len(m.Criteria[task]) == 0 {
		return false
	}
	return m.Contracts[task] != evidence.ContractV1
}

func (m Map) Report() ([]Violation, []Notice) {
	var violations []Violation
	var notices []Notice

	for _, rf := range m.Requirements {
		if len(m.RequirementMap[rf]) == 0 {
			violations = append(violations, Violation{
				Kind:    ViolationRequirementWithoutTask,
				Subject: rf,
				Detail:  "nenhuma tarefa na tabela de Cobertura de Requisitos cobre este RF",
			})
		}
	}

	tasks := make([]string, 0, len(m.TaskCoverage))
	for task := range m.TaskCoverage {
		tasks = append(tasks, task)
	}

	for _, task := range tasks {
		if m.MissingReports[task] {
			violations = append(violations, Violation{
				Kind:    ViolationTaskWithoutReport,
				Subject: task,
				Detail:  "relatorio de execucao ausente (<tarefa>_execution_report.md)",
			})
			continue
		}
		if m.TaskStatuses[task] == "blocked" {
			notices = append(notices, Notice{
				Kind:    NoticeTaskBlocked,
				Subject: task,
				Detail:  "tarefa bloqueada — criterios de aceite fora do escopo do gate ate a execucao concluir; os RF cobertos por ela seguem sem prova",
			})
			continue
		}
		items := m.Criteria[task]
		if len(items) == 0 {
			violations = append(violations, Violation{
				Kind:    ViolationTaskWithoutCriteria,
				Subject: task,
				Detail:  "secao '## Criterios de Aceite' ausente ou vazia no relatorio de execucao",
			})
			continue
		}
		if m.Contracts[task] == evidence.ContractV1 {
			notices = append(notices, Notice{
				Kind:    NoticeHistoricalEvidence,
				Subject: task,
				Detail:  "relatorio sob contrato de evidencia v1 (historico) — forma estrita de evidencia por criterio nao cobrada (RF-04: evidencia de execucao e historica e nao e reescrita)",
			})
			continue
		}
		for _, item := range items {
			if !item.HasEvidence() {
				violations = append(violations, Violation{
					Kind:    ViolationCriterionWithoutEvidence,
					Subject: fmt.Sprintf("%s: %s", task, item.Text),
					Detail:  "criterio sem linha de evidencia apos o separador '->'",
				})
			}
		}
	}

	if verified := m.VerifiedTaskCount(); verified == 0 {
		violations = append(violations, Violation{
			Kind:    ViolationGateVacuous,
			Subject: fmt.Sprintf("%d tarefa(s) no escopo, 0 verificada(s)", len(m.TaskCoverage)),
			Detail: "gate vacuo: nenhuma tarefa teve seus criterios confrontados contra a forma estrita de evidencia " +
				"(escopo vazio, tarefa bloqueada ou isencao de contrato v1 em todo o universo). Universo integralmente " +
				"isento nao e cadeia verificada e nao pode ser lido como aprovacao (RF-55/RF-56)",
		})
	}

	return violations, notices
}
