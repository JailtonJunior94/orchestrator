package taskloop

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
)

type Severity string

const (
	SeverityCritical   Severity = "Critical"
	SeverityHigh       Severity = "High"
	SeverityImportant  Severity = "Important"
	SeveritySuggestion Severity = "Suggestion"
)

type Finding struct {
	Severity Severity
	File     string
	Line     int
	Message  string
}

type ReviewVerdict string

const (
	VerdictApproved            ReviewVerdict = "APPROVED"
	VerdictApprovedWithRemarks ReviewVerdict = "APPROVED_WITH_REMARKS"
	VerdictRejected            ReviewVerdict = "REJECTED"
	VerdictBlocked             ReviewVerdict = "BLOCKED"
)

type FinalReviewResult struct {
	Verdict   ReviewVerdict
	Findings  []Finding
	RawOutput string
}

var ErrReviewRejected = errors.New("taskloop: review reprovou diff consolidado")

var ErrReviewBlocked = errors.New("taskloop: review bloqueada por falta de contexto ou evidencia")

type FinalReviewer interface {
	ReviewConsolidated(ctx context.Context, diff string) (FinalReviewResult, error)
}

const maxDiffPartitionSize = 100_000

type defaultFinalReviewer struct {
	invoker AgentInvoker
	workDir string
	model   string
	maxDiff int
}

var _ FinalReviewer = (*defaultFinalReviewer)(nil)

func NewFinalReviewer(invoker AgentInvoker, workDir, model string) FinalReviewer {
	return &defaultFinalReviewer{
		invoker: invoker,
		workDir: workDir,
		model:   model,
		maxDiff: maxDiffPartitionSize,
	}
}

func (r *defaultFinalReviewer) ReviewConsolidated(ctx context.Context, diff string) (FinalReviewResult, error) {
	partitions := NewCatalog().partitionDiff(diff, r.maxDiff)

	var allFindings []Finding
	var rawParts []string
	worstVerdict := VerdictApproved

	for _, part := range partitions {
		prompt := NewCatalog().buildConsolidatedReviewPrompt(part)
		stdout, _, _, err := r.invoker.Invoke(ctx, prompt, r.workDir, r.model)
		if err != nil {
			return FinalReviewResult{}, fmt.Errorf("taskloop: erro ao invocar review: %w", err)
		}

		result := NewCatalog().parseReviewOutput(stdout)
		if NewCatalog().hasBlockingFinding(result.Findings) {
			result.Verdict = VerdictRejected
		}
		allFindings = append(allFindings, result.Findings...)
		rawParts = append(rawParts, result.RawOutput)

		if NewCatalog().verdictWeight(result.Verdict) > NewCatalog().verdictWeight(worstVerdict) {
			worstVerdict = result.Verdict
		}
	}

	return FinalReviewResult{
		Verdict:   worstVerdict,
		Findings:  allFindings,
		RawOutput: strings.Join(rawParts, "\n\n---\n\n"),
	}, nil
}

func (c *Catalog) verdictWeight(v ReviewVerdict) int {
	switch v {
	case VerdictBlocked:
		return 3
	case VerdictRejected:
		return 2
	case VerdictApprovedWithRemarks:
		return 1
	default:
		return 0
	}
}

func (c *Catalog) buildConsolidatedReviewPrompt(diff string) string {
	languages := strings.Join(NewCatalog().detectReviewLanguages(diff), ", ")
	return fmt.Sprintf(`First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/review/SKILL.md

Use a skill review para revisar o diff consolidado abaixo. Esta e uma revisao final sobre todas as tasks implementadas.
Se a secao "Contexto da revisao consolidada" listar caminhos de arquivos, leia esses arquivos antes de concluir o veredito.

Focos obrigatorios:
- corretude: a implementacao atende todos os RFs e criterios de aceite?
- regressao: alguma mudanca quebra contrato publico ou comportamento existente?
- seguranca: ha injecao de dependencia insegura, dado sensivel exposto ou validacao faltando?
- testes: todos os cenarios de criterio de pronto estao cobertos?
- divida tecnica introduzida: o que precisara de refactor futuro?
- linguagens detectadas no patch: %s. Aplique as convencoes e riscos de cada uma.

Saidas esperadas:
- lista de achados por categoria: [Critical], [Important], [Suggestion]
- para cada achado: [arquivo:linha] descricao e correcao sugerida
- veredicto final em linha propria: APPROVED / APPROVED_WITH_REMARKS / REJECTED / BLOCKED

Do NOT modify any files. Review in read-only mode and report findings only via stdout.

Diff consolidado:
`+"```"+`
%s
`+"```", languages, diff)
}

func (c *Catalog) detectReviewLanguages(diff string) []string {
	seen := make(map[string]bool)
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "+++ b/") && !strings.HasPrefix(line, "diff --git ") {
			continue
		}
		name := line
		if strings.HasPrefix(line, "+++ b/") {
			name = strings.TrimPrefix(line, "+++ b/")
		}
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".go":
			seen["Go"] = true
		case ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs":
			seen["Node/TypeScript"] = true
		case ".py":
			seen["Python"] = true
		case ".cs", ".csproj", ".sln":
			seen[".NET/C#"] = true
		}
	}
	if len(seen) == 0 {
		return []string{"geral"}
	}
	ordered := []string{"Go", "Node/TypeScript", "Python", ".NET/C#"}
	out := make([]string, 0, len(seen))
	for _, language := range ordered {
		if seen[language] {
			out = append(out, language)
		}
	}
	return out
}

func (c *Catalog) hasBlockingFinding(findings []Finding) bool {
	for _, finding := range findings {
		if finding.Severity == SeverityCritical || finding.Severity == SeverityHigh {
			return true
		}
	}
	return false
}

func (c *Catalog) parseReviewOutput(raw string) FinalReviewResult {
	return FinalReviewResult{
		Verdict:   NewCatalog().parseVerdict(raw),
		Findings:  NewCatalog().parseFindings(raw),
		RawOutput: raw,
	}
}

func (c *Catalog) parseVerdict(raw string) ReviewVerdict {
	return ReviewVerdict(approval.NewTranslator().Translate(raw).String())
}

func (c *Catalog) parseFindings(raw string) []Finding {
	var findings []Finding
	for _, finding := range approval.ParseReviewFindings(raw) {
		findings = append(findings, Finding{
			Severity: reverseSeverity(finding.Severity()),
			File:     finding.File(),
			Line:     finding.Line(),
			Message:  finding.Description(),
		})
	}
	return findings
}

func (c *Catalog) partitionDiff(diff string, maxSize int) []string {
	if len(diff) <= maxSize {
		return []string{diff}
	}

	lines := strings.Split(diff, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var fileSections []string
	var section strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") && section.Len() > 0 {
			fileSections = append(fileSections, section.String())
			section.Reset()
		}
		section.WriteString(line)
		section.WriteByte('\n')
	}
	if section.Len() > 0 {
		fileSections = append(fileSections, section.String())
	}

	if len(fileSections) == 0 {
		return []string{diff}
	}

	var partitions []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			partitions = append(partitions, current.String())
			current.Reset()
		}
	}

	for _, s := range fileSections {
		if len(s) > maxSize {
			flush()
			partitions = append(partitions, NewCatalog().splitFileSection(s, maxSize)...)
			continue
		}
		if current.Len() > 0 && current.Len()+len(s) > maxSize {
			flush()
		}
		current.WriteString(s)
	}
	flush()

	return partitions
}

func (c *Catalog) splitFileSection(section string, maxSize int) []string {
	lines := strings.Split(section, "\n")
	headerEnd := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "@@") {
			headerEnd = i
			break
		}
	}
	if headerEnd <= 0 {
		return []string{section}
	}

	header := strings.Join(lines[:headerEnd], "\n") + "\n"
	rest := lines[headerEnd:]

	var hunks []string
	var cur strings.Builder
	for _, l := range rest {
		if strings.HasPrefix(l, "@@") && cur.Len() > 0 {
			hunks = append(hunks, cur.String())
			cur.Reset()
		}
		cur.WriteString(l)
		cur.WriteByte('\n')
	}
	if cur.Len() > 0 {
		hunks = append(hunks, cur.String())
	}

	var out []string
	var part strings.Builder
	part.WriteString(header)
	first := true
	for _, h := range hunks {
		piece := h
		if len(header)+len(piece) > maxSize {
			if part.Len() > len(header) {
				out = append(out, part.String())
			}
			out = append(out, NewCatalog().truncateOversize(header+piece, maxSize))
			part.Reset()
			part.WriteString(header)
			first = true
			continue
		}
		if !first && part.Len()+len(piece) > maxSize {
			out = append(out, part.String())
			part.Reset()
			part.WriteString(header)
		}
		part.WriteString(piece)
		first = false
	}
	if part.Len() > len(header) {
		out = append(out, part.String())
	}
	if len(out) == 0 {
		return []string{NewCatalog().truncateOversize(section, maxSize)}
	}
	return out
}

func (c *Catalog) truncateOversize(s string, maxSize int) string {
	const marker = "\n# ... [truncado: secao excede maxDiffPartitionSize]\n"
	if len(s) <= maxSize {
		return s
	}
	cut := maxSize - len(marker)
	if cut < 0 {
		cut = 0
	}
	return s[:cut] + marker
}

var ErrTemplateInvalido = errors.New("template de revisao invalido")

//go:embed review_template.tmpl
var defaultReviewTemplate string

//go:embed bugfix_template.tmpl
var defaultBugfixTemplate string

type ReviewTemplateData struct {
	TaskFile       string
	PRDFolder      string
	TechSpec       string
	TasksFile      string
	Diff           string
	CompletedTasks string
	RiskAreas      string
}

func (c *Catalog) BuildReviewPrompt(templatePath string, data ReviewTemplateData, fsys fs.FileSystem) (string, error) {
	var tmplContent string
	if templatePath != "" {
		raw, err := fsys.ReadFile(templatePath)
		if err != nil {
			return "", fmt.Errorf("%w: nao foi possivel ler arquivo %q: %v", ErrTemplateInvalido, templatePath, err)
		}
		tmplContent = string(raw)
	} else {
		tmplContent = defaultReviewTemplate
	}

	tmpl, err := template.New("review").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateInvalido, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateInvalido, err)
	}

	return buf.String(), nil
}

const diffUnavailable = "(diff indisponivel)"

func (c *Catalog) captureGitDiff(ctx context.Context, workDir string) string {
	if !NewCatalog().isGitWorkTree(ctx, workDir) {
		return diffUnavailable
	}

	var sections []string

	if base := airuntime.ResolveReviewBaseRef(ctx, workDir); base != "" {
		if diff, ok := NewCatalog().commandDiff(ctx, workDir, false, "git", "diff", "--binary", base, "HEAD", "--"); ok {
			sections = append(sections, diff)
		}
	}

	if diff, ok := NewCatalog().commandDiff(ctx, workDir, false, "git", "diff", "--binary", "--cached", "--"); ok {
		sections = append(sections, diff)
	}
	if diff, ok := NewCatalog().commandDiff(ctx, workDir, false, "git", "diff", "--binary", "--"); ok {
		sections = append(sections, diff)
	}

	untracked, err := NewCatalog().commandOutputLines(ctx, workDir, "git", "ls-files", "--others", "--exclude-standard", "--")
	if err != nil {
		if len(strings.TrimSpace(strings.Join(sections, "\n"))) > 0 {
			return strings.Join(sections, "\n") + "\n"
		}
		return diffUnavailable
	}
	for _, file := range untracked {
		if file == "" {
			continue
		}
		if diff, ok := NewCatalog().commandDiff(ctx, workDir, true, "git", "diff", "--binary", "--no-index", "--", os.DevNull, file); ok {
			sections = append(sections, diff)
		}
	}

	combined := strings.TrimSpace(strings.Join(sections, "\n"))
	if combined == "" {
		return diffUnavailable
	}
	return combined + "\n"
}

func (c *Catalog) isGitWorkTree(ctx context.Context, workDir string) bool {
	out, err := NewCatalog().commandOutput(ctx, workDir, "git", "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func (c *Catalog) commandOutput(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Output()
}

func (c *Catalog) commandOutputLines(ctx context.Context, dir, name string, args ...string) ([]string, error) {
	out, err := NewCatalog().commandOutput(ctx, dir, name, args...)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func (c *Catalog) commandDiff(ctx context.Context, dir string, allowExitOne bool, name string, args ...string) (string, bool) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		diff := strings.TrimSpace(string(out))
		return diff, diff != ""
	}

	var exitErr *exec.ExitError
	if allowExitOne && errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		diff := strings.TrimSpace(string(out))
		return diff, diff != ""
	}

	return "", false
}

func (c *Catalog) detectRiskAreas(prdFolder, workDir string, diff string, fsys fs.FileSystem) string {
	techspecPath := filepath.Join(workDir, prdFolder, "techspec.md")
	techspec, _ := fsys.ReadFile(techspecPath)
	combined := strings.ToLower(string(techspec) + "\n" + diff)

	var areas []string

	if NewCatalog().containsAnyPattern(combined, "performance", "latencia", "latency", "benchmark", "cache", "pool", "buffer") {
		areas = append(areas, "performance")
	}
	if NewCatalog().containsAnyPattern(combined, "seguranca", "security", "auth", "credential", "token", "injection", "xss", "csrf") {
		areas = append(areas, "seguranca")
	}
	if NewCatalog().containsAnyPattern(combined, "interface ", "contrato", "contract", "assinatura publica", "public api", "breaking change") {
		areas = append(areas, "contratos")
	}
	if NewCatalog().containsAnyPattern(combined, "goroutine", "mutex", "channel", "sync.", "concurren", "race", "deadlock", "lock") {
		areas = append(areas, "concorrencia")
	}
	if NewCatalog().containsAnyPattern(combined, "migra", "schema", "database", "sql", "query") {
		areas = append(areas, "persistencia")
	}

	if len(areas) == 0 {
		areas = append(areas, "contratos", "seguranca")
	}
	return strings.Join(areas, ", ")
}

type BugfixTemplateData struct {
	TaskFile       string
	PRDFolder      string
	TechSpec       string
	TasksFile      string
	ReviewFindings string
	Diff           string
}

var ErrBugfixTemplateInvalido = errors.New("template de bugfix invalido")

func (c *Catalog) BuildBugfixPrompt(data BugfixTemplateData) (string, error) {
	tmpl, err := template.New("bugfix").Parse(defaultBugfixTemplate)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBugfixTemplateInvalido, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrBugfixTemplateInvalido, err)
	}

	return buf.String(), nil
}

func (c *Catalog) formatCompletedTasks(iterations []IterationResult, currentTaskID string) string {
	var completed []string
	seen := make(map[string]bool)

	for _, iter := range iterations {
		if iter.PostStatus == "done" && !seen[iter.TaskID] {
			seen[iter.TaskID] = true
			completed = append(completed, fmt.Sprintf("%s (%s)", iter.TaskID, iter.Title))
		}
	}

	if !seen[currentTaskID] {
		completed = append(completed, currentTaskID+" (atual)")
	}

	if len(completed) == 0 {
		return "(nenhuma task concluida anteriormente)"
	}
	return strings.Join(completed, ", ")
}
