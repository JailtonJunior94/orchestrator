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
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// --- FinalReviewer types ---

// Severity classifica a gravidade de um achado de revisao.
type Severity string

const (
	SeverityCritical   Severity = "Critical"
	SeverityImportant  Severity = "Important"
	SeveritySuggestion Severity = "Suggestion"
)

// Finding representa um achado individual da revisao.
type Finding struct {
	Severity Severity
	File     string
	Line     int
	Message  string
}

// ReviewVerdict e o veredito final emitido pelo FinalReviewer.
type ReviewVerdict string

const (
	VerdictApproved            ReviewVerdict = "APPROVED"
	VerdictApprovedWithRemarks ReviewVerdict = "APPROVED_WITH_REMARKS"
	VerdictRejected            ReviewVerdict = "REJECTED"
	VerdictBlocked             ReviewVerdict = "BLOCKED"
)

// FinalReviewResult agrega veredito, achados e saida bruta da revisao consolidada.
type FinalReviewResult struct {
	Verdict   ReviewVerdict
	Findings  []Finding
	RawOutput string
}

// ErrReviewRejected indica que a revisao final reprovou o diff consolidado.
var ErrReviewRejected = errors.New("taskloop: review reprovou diff consolidado")

// ErrReviewBlocked indica que a revisao nao conseguiu emitir um veredito conclusivo.
var ErrReviewBlocked = errors.New("taskloop: review bloqueada por falta de contexto ou evidencia")

// FinalReviewer invoca a skill review uma unica vez sobre o diff consolidado.
type FinalReviewer interface {
	ReviewConsolidated(ctx context.Context, diff string) (FinalReviewResult, error)
}

// maxDiffPartitionSize e o limite em bytes para cada particao de diff enviada ao reviewer.
const maxDiffPartitionSize = 100_000

type defaultFinalReviewer struct {
	invoker AgentInvoker
	workDir string
	model   string
	maxDiff int
}

// NewFinalReviewer cria um FinalReviewer que usa o AgentInvoker fornecido.
func NewFinalReviewer(invoker AgentInvoker, workDir, model string) FinalReviewer {
	return &defaultFinalReviewer{
		invoker: invoker,
		workDir: workDir,
		model:   model,
		maxDiff: maxDiffPartitionSize,
	}
}

// ReviewConsolidated invoca a skill review sobre o diff consolidado.
// Quando o diff excede maxDiff, particiona por arquivo antes de enviar.
// Agrega vereditos (o mais grave prevalece) e une os achados de todas as particoes.
func (r *defaultFinalReviewer) ReviewConsolidated(ctx context.Context, diff string) (FinalReviewResult, error) {
	partitions := partitionDiff(diff, r.maxDiff)

	var allFindings []Finding
	var rawParts []string
	worstVerdict := VerdictApproved

	for _, part := range partitions {
		prompt := buildConsolidatedReviewPrompt(part)
		stdout, _, _, err := r.invoker.Invoke(ctx, prompt, r.workDir, r.model)
		if err != nil {
			return FinalReviewResult{}, fmt.Errorf("taskloop: erro ao invocar review: %w", err)
		}

		result := parseReviewOutput(stdout)
		allFindings = append(allFindings, result.Findings...)
		rawParts = append(rawParts, result.RawOutput)

		if verdictWeight(result.Verdict) > verdictWeight(worstVerdict) {
			worstVerdict = result.Verdict
		}
	}

	return FinalReviewResult{
		Verdict:   worstVerdict,
		Findings:  allFindings,
		RawOutput: strings.Join(rawParts, "\n\n---\n\n"),
	}, nil
}

func verdictWeight(v ReviewVerdict) int {
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

// buildConsolidatedReviewPrompt constroi o prompt para revisao consolidada do diff.
func buildConsolidatedReviewPrompt(diff string) string {
	return fmt.Sprintf(`First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/review/SKILL.md

Use a skill review para revisar o diff consolidado abaixo. Esta e uma revisao final sobre todas as tasks implementadas.

Focos obrigatorios:
- corretude: a implementacao atende todos os RFs e criterios de aceite?
- regressao: alguma mudanca quebra contrato publico ou comportamento existente?
- seguranca: ha injecao de dependencia insegura, dado sensivel exposto ou validacao faltando?
- testes: todos os cenarios de criterio de pronto estao cobertos?
- divida tecnica introduzida: o que precisara de refactor futuro?

Saidas esperadas:
- lista de achados por categoria: [Critical], [Important], [Suggestion]
- para cada achado: [arquivo:linha] descricao e correcao sugerida
- veredicto final em linha propria: APPROVED / APPROVED_WITH_REMARKS / REJECTED

Do NOT modify any files. Review in read-only mode and report findings only via stdout.

Diff consolidado:
`+"```"+`
%s
`+"```", diff)
}

// parseReviewOutput extrai veredito e achados da saida bruta da skill review.
func parseReviewOutput(raw string) FinalReviewResult {
	return FinalReviewResult{
		Verdict:   parseVerdict(raw),
		Findings:  parseFindings(raw),
		RawOutput: raw,
	}
}

// verdictLineRe casa uma linha dedicada de veredito do tipo
// "Verdict: APPROVED_WITH_REMARKS" / "Veredito - REJECTED" / "**Veredito final:** APPROVED".
// Ancora a deteccao em linha propria para evitar falso positivo quando o corpo
// menciona palavras-chave (ex.: "CI was blocked earlier").
var verdictLineRe = regexp.MustCompile(`(?im)^\s*[*_>\s-]*(?:final\s+)?(?:verdict|veredic?to|vereditto|veredicto)(?:\s+final)?\s*[:\-–]\s*[*_` + "`" + `]*\s*(APPROVED_WITH_REMARKS|APPROVED WITH REMARKS|APPROVED|APROVADO\s+COM\s+RESSALVAS|APROVADO|REJECTED|REPROVADO|REJEITADO|BLOCKED|BLOQUEAD[OA])\b`)

// parseVerdict extrai o veredito da saida bruta. Verifica do mais especifico ao mais geral.
func parseVerdict(raw string) ReviewVerdict {
	if m := verdictLineRe.FindStringSubmatch(raw); len(m) > 1 {
		token := strings.ToUpper(strings.Join(strings.Fields(m[1]), " "))
		switch {
		case strings.HasPrefix(token, "APPROVED_WITH_REMARKS"),
			strings.HasPrefix(token, "APPROVED WITH REMARKS"),
			strings.HasPrefix(token, "APROVADO COM RESSALVAS"):
			return VerdictApprovedWithRemarks
		case strings.HasPrefix(token, "BLOCKED"), strings.HasPrefix(token, "BLOQUEAD"):
			return VerdictBlocked
		case strings.HasPrefix(token, "REJECTED"), strings.HasPrefix(token, "REPROVADO"), strings.HasPrefix(token, "REJEITADO"):
			return VerdictRejected
		case strings.HasPrefix(token, "APPROVED"), strings.HasPrefix(token, "APROVADO"):
			return VerdictApproved
		}
	}

	lower := strings.ToLower(raw)

	if containsAnyPattern(lower,
		"approved_with_remarks", "aprovado com ressalvas", "approved with remarks",
		"approved_with_observations", "aprovado com observacoes", "aprovado com observações",
		"approved with observations",
	) {
		return VerdictApprovedWithRemarks
	}
	if containsAnyPattern(lower, "blocked", "bloqueado", "bloqueada") {
		return VerdictBlocked
	}
	if containsAnyPattern(lower, "rejected", "reprovado", "rejeitado") {
		return VerdictRejected
	}
	if containsAnyPattern(lower, "approved", "aprovado") {
		return VerdictApproved
	}
	return VerdictBlocked
}

// parseFindings extrai achados individuais da saida bruta.
// Reconhece marcadores [Critical], [Important], [Suggestion] e variantes PT-BR.
func parseFindings(raw string) []Finding {
	var findings []Finding
	for _, line := range strings.Split(raw, "\n") {
		sev, ok := extractSeverity(line)
		if !ok {
			continue
		}
		file, lineNum := extractFileLine(line)
		findings = append(findings, Finding{
			Severity: sev,
			File:     file,
			Line:     lineNum,
			Message:  strings.TrimSpace(line),
		})
	}
	return findings
}

// extractSeverity detecta severidade na linha. Retorna (severity, true) se encontrada.
func extractSeverity(line string) (Severity, bool) {
	lower := strings.ToLower(line)
	switch {
	case containsAnyPattern(lower, "[critical]", "[critico]", "[crítico]"):
		return SeverityCritical, true
	case containsAnyPattern(lower, "[important]", "[importante]"):
		return SeverityImportant, true
	case containsAnyPattern(lower, "[suggestion]", "[sugestao]", "[sugestão]"):
		return SeveritySuggestion, true
	default:
		return "", false
	}
}

// extractFileLine tenta extrair arquivo e numero de linha de padroes como [file.go:42].
func extractFileLine(line string) (file string, lineNum int) {
	start := strings.Index(line, "[")
	for start >= 0 {
		end := strings.Index(line[start:], "]")
		if end < 0 {
			break
		}
		inner := line[start+1 : start+end]
		if colon := strings.LastIndex(inner, ":"); colon > 0 {
			candidate := inner[colon+1:]
			n, err := strconv.Atoi(strings.TrimSpace(candidate))
			if err == nil {
				return strings.TrimSpace(inner[:colon]), n
			}
		}
		start = start + end + 1
		if start >= len(line) {
			break
		}
		next := strings.Index(line[start:], "[")
		if next < 0 {
			break
		}
		start = start + next
	}
	return "", 0
}

// partitionDiff divide o diff em particoes que cabem em maxSize bytes.
// Tenta manter arquivos inteiros em cada particao (divide em cabecalhos "diff --git").
func partitionDiff(diff string, maxSize int) []string {
	if len(diff) <= maxSize {
		return []string{diff}
	}

	// Split diff into per-file sections at "diff --git" boundaries.
	// Trim the trailing empty element produced by a trailing newline.
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
			partitions = append(partitions, splitFileSection(s, maxSize)...)
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

// splitFileSection quebra uma seccao "diff --git" maior que maxSize em
// sub-particoes alinhadas em hunks "@@". O cabecalho do arquivo (linhas ate
// o primeiro hunk) e replicado em cada sub-particao, sinalizando truncamento
// com o marcador "# (continuacao truncada de <arquivo>)".
// Quando ate um unico hunk ainda excede maxSize, aplica truncamento explicito
// apendando "\n# ... [truncado: secao excede maxDiffPartitionSize]\n".
func splitFileSection(section string, maxSize int) []string {
	lines := strings.Split(section, "\n")
	headerEnd := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "@@") {
			headerEnd = i
			break
		}
	}
	if headerEnd <= 0 {
		// Sem hunks "@@" para subdividir: nao ha granularidade segura — devolve
		// a seccao integra preservando conteudo. Caller ciente do trade-off.
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
			out = append(out, truncateOversize(header+piece, maxSize))
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
		return []string{truncateOversize(section, maxSize)}
	}
	return out
}

func truncateOversize(s string, maxSize int) string {
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

// ErrTemplateInvalido indica erro no parsing ou execucao do template de revisao.
var ErrTemplateInvalido = errors.New("template de revisao invalido")

// defaultReviewTemplate e o template embutido via go:embed.
//
//go:embed review_template.tmpl
var defaultReviewTemplate string

// defaultBugfixTemplate e o template embutido via go:embed para o prompt de bugfix.
//
//go:embed bugfix_template.tmpl
var defaultBugfixTemplate string

// ReviewTemplateData agrupa os placeholders do template de revisao.
type ReviewTemplateData struct {
	TaskFile       string
	PRDFolder      string
	TechSpec       string
	TasksFile      string
	Diff           string
	CompletedTasks string // Lista de tasks executadas no bundle ate o momento
	RiskAreas      string // Areas de risco detectadas (performance, seguranca, contratos, concorrencia)
}

// BuildReviewPrompt constroi o prompt de revisao a partir do template.
// Se templatePath != "", carrega template customizado do disco via fsys.
// Se templatePath == "", usa defaultReviewTemplate embutido.
// Retorna erro wrappado com ErrTemplateInvalido em caso de falha de parsing ou execucao.
func BuildReviewPrompt(templatePath string, data ReviewTemplateData, fsys fs.FileSystem) (string, error) {
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

// captureGitDiff agrega o diff do working tree atual: staged, unstaged e arquivos
// untracked. Se o diretorio nao for um repo git valido ou o working tree estiver
// limpo, retorna "(diff indisponivel)". Nao e erro bloqueante.
func captureGitDiff(ctx context.Context, workDir string) string {
	if !isGitWorkTree(ctx, workDir) {
		return "(diff indisponivel)"
	}

	var sections []string

	if diff, ok := commandDiff(ctx, workDir, false, "git", "diff", "--binary", "--cached", "--"); ok {
		sections = append(sections, diff)
	}
	if diff, ok := commandDiff(ctx, workDir, false, "git", "diff", "--binary", "--"); ok {
		sections = append(sections, diff)
	}

	untracked, err := commandOutputLines(ctx, workDir, "git", "ls-files", "--others", "--exclude-standard", "--")
	if err != nil {
		if len(strings.TrimSpace(strings.Join(sections, "\n"))) > 0 {
			return strings.Join(sections, "\n") + "\n"
		}
		return "(diff indisponivel)"
	}
	for _, file := range untracked {
		if file == "" {
			continue
		}
		if diff, ok := commandDiff(ctx, workDir, true, "git", "diff", "--binary", "--no-index", "--", os.DevNull, file); ok {
			sections = append(sections, diff)
		}
	}

	combined := strings.TrimSpace(strings.Join(sections, "\n"))
	if combined == "" {
		return "(diff indisponivel)"
	}
	return combined + "\n"
}

func isGitWorkTree(ctx context.Context, workDir string) bool {
	out, err := commandOutput(ctx, workDir, "git", "rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func commandOutput(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Output()
}

func commandOutputLines(ctx context.Context, dir, name string, args ...string) ([]string, error) {
	out, err := commandOutput(ctx, dir, name, args...)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	return strings.Split(trimmed, "\n"), nil
}

func commandDiff(ctx context.Context, dir string, allowExitOne bool, name string, args ...string) (string, bool) {
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

// detectRiskAreas analisa o conteudo combinado de techspec e diff para detectar
// areas de risco relevantes para a revisao.
func detectRiskAreas(prdFolder, workDir string, diff string, fsys fs.FileSystem) string {
	techspecPath := filepath.Join(workDir, prdFolder, "techspec.md")
	techspec, _ := fsys.ReadFile(techspecPath)
	combined := strings.ToLower(string(techspec) + "\n" + diff)

	var areas []string

	if containsAnyPattern(combined, "performance", "latencia", "latency", "benchmark", "cache", "pool", "buffer") {
		areas = append(areas, "performance")
	}
	if containsAnyPattern(combined, "seguranca", "security", "auth", "credential", "token", "injection", "xss", "csrf") {
		areas = append(areas, "seguranca")
	}
	if containsAnyPattern(combined, "interface ", "contrato", "contract", "assinatura publica", "public api", "breaking change") {
		areas = append(areas, "contratos")
	}
	if containsAnyPattern(combined, "goroutine", "mutex", "channel", "sync.", "concurren", "race", "deadlock", "lock") {
		areas = append(areas, "concorrencia")
	}
	if containsAnyPattern(combined, "migra", "schema", "database", "sql", "query") {
		areas = append(areas, "persistencia")
	}

	if len(areas) == 0 {
		areas = append(areas, "contratos", "seguranca")
	}
	return strings.Join(areas, ", ")
}

// BugfixTemplateData agrupa os placeholders do template de bugfix.
type BugfixTemplateData struct {
	TaskFile       string
	PRDFolder      string
	TechSpec       string
	TasksFile      string
	ReviewFindings string // saida bruta do reviewer (achados criticos)
	Diff           string // diff original que disparou os achados
}

// ErrBugfixTemplateInvalido indica erro no parsing ou execucao do template de bugfix.
var ErrBugfixTemplateInvalido = errors.New("template de bugfix invalido")

// BuildBugfixPrompt constroi o prompt de bugfix a partir do template embutido.
// Retorna erro wrappado com ErrBugfixTemplateInvalido em caso de falha de parsing ou execucao.
func BuildBugfixPrompt(data BugfixTemplateData) (string, error) {
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

// formatCompletedTasks formata a lista de tasks ja executadas no bundle
// a partir das iteracoes anteriores do report.
func formatCompletedTasks(iterations []IterationResult, currentTaskID string) string {
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
