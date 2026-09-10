package runtime

// runner_autoreview.go implementa os helpers de auto-review para F5-Claude.
// Separado de runner.go para manter o application service legível (OC heurística).

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

const (
	// _reviewSkillPath é o caminho relativo da skill de review lida em runtime.
	// Falha de leitura gera erro claro — não spawna review sem skill.
	_reviewSkillPath = ".agents/skills/review/SKILL.md"

	// _reviewDiffMaxBytes é o limite default de bytes do diff (5 MB).
	// Configurável via env AISPEC_REVIEW_DIFF_MAX.
	_reviewDiffMaxBytes = 5 * 1024 * 1024
)

// ReviewResult agrega o resultado do auto-review.
type ReviewResult struct {
	// Status é "ok" ou "blocked".
	Status string
	// Path é o caminho de evidence/<task>/review.md.
	Path string
	// Output é o texto completo da sessão de review.
	Output string
	// HardIssues lista as linhas que contêm [HARD]/BLOQUEADO/CRÍTICO.
	HardIssues []string
}

// buildReviewPrompt constrói o prompt para a sessão de review.
// skillBody é o conteúdo de .agents/skills/review/SKILL.md.
// gitDiff é a saída de git diff (staged + unstaged).
func (c *Catalog) buildReviewPrompt(skillBody, gitDiff string) string {
	return fmt.Sprintf(
		"%s\n\n## Diff a Revisar\n\n```diff\n%s\n```\n\n## Instrução\n"+
			"Revise o diff acima conforme as regras da skill. Reporte issues por severidade.\n"+
			"Para issues `hard`/`CRÍTICO`/`BLOQUEADO`, prefixar a linha com [HARD].\n",
		skillBody, gitDiff,
	)
}

// parseReviewStatus analisa a saída do review e retorna "blocked" ou "ok".
// Regras (documentadas aqui por legibilidade — não duplicar no caller):
//   - Contém "[HARD]"    → blocked (marcador explícito de issue hard)
//   - Contém "BLOQUEADO" → blocked (português; paridade Compozy review)
//   - Contém "CRÍTICO"   → blocked (sinônimo de hard em PT-BR)
//   - Caso contrário     → ok
func (c *Catalog) parseReviewStatus(reviewOutput string) string {
	if strings.Contains(reviewOutput, "[HARD]") ||
		strings.Contains(reviewOutput, "BLOQUEADO") ||
		strings.Contains(reviewOutput, "CRÍTICO") {
		return "blocked"
	}
	return "ok"
}

// extractHardIssues retorna as linhas do review output que contêm marcadores críticos.
func (c *Catalog) extractHardIssues(reviewOutput string) []string {
	var issues []string
	for line := range strings.SplitSeq(reviewOutput, "\n") {
		if strings.Contains(line, "[HARD]") ||
			strings.Contains(line, "BLOQUEADO") ||
			strings.Contains(line, "CRÍTICO") {
			issues = append(issues, strings.TrimSpace(line))
		}
	}
	return issues
}

// collectGitDiff coleta git diff (staged + unstaged) no workDir.
// Limita saída a reviewDiffMaxBytes (ou AISPEC_REVIEW_DIFF_MAX env).
// Trunca com warning prefixado quando excede o limite.
func (c *Catalog) collectGitDiff(workDir string) string {
	maxBytes := _reviewDiffMaxBytes
	if envVal := os.Getenv("AISPEC_REVIEW_DIFF_MAX"); envVal != "" {
		var n int
		if _, err := fmt.Sscan(envVal, &n); err == nil && n > 0 {
			maxBytes = n
		}
	}

	var sb strings.Builder

	// git diff --staged (staged changes)
	staged, err := NewCatalog().runGitDiff(workDir, "--staged")
	if err == nil && staged != "" {
		sb.WriteString(staged)
	}

	// git diff (unstaged changes)
	unstaged, err := NewCatalog().runGitDiff(workDir)
	if err == nil && unstaged != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(unstaged)
	}

	result := sb.String()
	if len(result) > maxBytes {
		truncMsg := fmt.Sprintf("# [AVISO: diff truncado em %d bytes de %d totais]\n", maxBytes, len(result))
		result = truncMsg + result[:maxBytes]
	}

	if result == "" {
		result = "(sem diff disponível)"
	}

	return result
}

// runGitDiff executa `git diff [args...]` no workDir e retorna a saída.
func (c *Catalog) runGitDiff(workDir string, args ...string) (string, error) {
	cmdArgs := append([]string{"diff"}, args...) //nolint:gocritic
	cmd := exec.Command("git", cmdArgs...)       //nolint:gosec
	cmd.Dir = workDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return out.String(), nil
}

// autoReviewOutputFn é injetável para testes (evitar spawn real de ACPRunner em testes unitários).
// Em produção é nil; em testes pode ser substituída via campo do runner (não exposto — testado via mock).
type autoReviewOutputFn func(ctx context.Context, j Job) (string, error)

// runAutoReview spawna nova ACPRunner com prompt de review e retorna ReviewResult.
// HARD: child Job tem AutoReview=false forçado (anti-recursão — T-REV-04 valida).
// HARD: falha de leitura da skill → erro claro, não spawna review.
func (r *ACPRunner) runAutoReview(ctx context.Context, j Job) (ReviewResult, error) {
	// Ler skill de review em runtime (não embutir inline).
	skillBody, err := r.readReviewSkill(j.WorkDir)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("runAutoReview: %w", err)
	}

	gitDiff := NewCatalog().collectGitDiff(j.WorkDir)
	prompt := NewCatalog().buildReviewPrompt(skillBody, gitDiff)

	reviewEvidenceDir := filepath.Join(j.EvidenceDir, "review")
	reviewPath := filepath.Join(j.EvidenceDir, "review.md")

	// HARD: child Job tem AutoReview=false (anti-recursão).
	childJob := Job{
		Prompt:      prompt,
		WorkDir:     j.WorkDir,
		EvidenceDir: reviewEvidenceDir,
		// RuntimeConfig: apenas Timeout para o review; demais campos inertes (F1).
		RuntimeConfig: RuntimeConfig{
			Timeout: NewCatalog().mustReviewTimeout(),
		},
		Quiet:          true,
		TasksDir:       j.TasksDir,
		TaskFileName:   j.TaskFileName,
		DisableHooks:   j.DisableHooks,
		SkipDriftGuard: j.SkipDriftGuard, // herdar bypass do parent (consistência do guard)
		AutoReview:     false,            // HARD: recursão bloqueada
	}

	reviewOutput, err := r.spawnReviewSession(ctx, childJob)

	status := NewCatalog().translateReviewStatus(reviewOutput)
	hardIssues := NewCatalog().extractHardIssues(reviewOutput)

	reviewPointerContent := NewCatalog().buildReviewPointer(reviewEvidenceDir, status)
	_ = os.MkdirAll(filepath.Dir(reviewPath), 0o755)
	_ = os.WriteFile(reviewPath, []byte(reviewPointerContent), 0o644)

	return ReviewResult{
		Status:     status,
		Path:       reviewPath,
		Output:     reviewOutput,
		HardIssues: hardIssues,
	}, err
}

// readReviewSkill lê .agents/skills/review/SKILL.md relativo ao workDir ou ao cwd.
func (r *ACPRunner) readReviewSkill(workDir string) (string, error) {
	// Tentar relativo ao workDir primeiro.
	p := filepath.Join(workDir, _reviewSkillPath)
	body, err := os.ReadFile(p)
	if err == nil {
		return string(body), nil
	}
	// Fallback: relativo ao cwd (útil em desenvolvimento local).
	body, err = os.ReadFile(_reviewSkillPath)
	if err != nil {
		return "", fmt.Errorf("ler skill review %q: %w", _reviewSkillPath, err)
	}
	return string(body), nil
}

func (r *ACPRunner) spawnReviewSession(ctx context.Context, childJob Job) (string, error) {
	if r.reviewOutputFn != nil {
		return r.reviewOutputFn(ctx, childJob)
	}

	capture := &reviewOutputCapture{}
	reviewRunner := NewACPRunner(r.spec, NewCatalog().WithClock(r.clock), NewCatalog().WithProber(r.prober), NewCatalog().WithClientFactory(r.factory), NewCatalog().WithPersistenceFactory(&reviewCaptureFactory{inner: r.persistenceFactory, capture: capture}))

	_, runErr := reviewRunner.Run(ctx, childJob)
	return capture.String(), runErr
}

func (c *Catalog) translateReviewStatus(reviewOutput string) string {
	if approval.NewTranslator().Translate(reviewOutput).Approves() {
		return "ok"
	}
	return "blocked"
}

type reviewOutputCapture struct {
	inner Persistence
	buf   strings.Builder
}

func (r *reviewOutputCapture) AppendEvent(evt events.Event) error {
	if evt.Kind() == events.KindAgentMessage {
		if msg := evt.AgentMessage(); msg != nil {
			if r.buf.Len() > 0 {
				r.buf.WriteByte('\n')
			}
			r.buf.WriteString(msg.Text())
		}
	}
	if r.inner != nil {
		return r.inner.AppendEvent(evt)
	}
	return nil
}

func (r *reviewOutputCapture) WriteToolCalls(summary []events.ToolCallSummary) error {
	if r.inner != nil {
		return r.inner.WriteToolCalls(summary)
	}
	return nil
}

func (r *reviewOutputCapture) EnrichReport(summary Summary) error {
	if r.inner != nil {
		return r.inner.EnrichReport(summary)
	}
	return nil
}

func (r *reviewOutputCapture) String() string {
	return r.buf.String()
}

type reviewCaptureFactory struct {
	inner   PersistenceFactory
	capture *reviewOutputCapture
}

func (f *reviewCaptureFactory) New(evidenceDir string) (Persistence, error) {
	if f.inner != nil {
		inner, err := f.inner.New(evidenceDir)
		if err != nil {
			return nil, err
		}
		f.capture.inner = inner
	}
	return f.capture, nil
}

// buildReviewPointer cria o conteúdo do arquivo review.md (apontador conveniente ~3 linhas).
func (c *Catalog) buildReviewPointer(evidenceDir, status string) string {
	return fmt.Sprintf("# Auto-Review\n\nReviewStatus: %s\n\nRelatório completo: %s/execution_report.md\n",
		status, evidenceDir)
}

// mustReviewTimeout retorna 5*time.Minute como ActivityTimeout para sessões de review.
// Timeout reduzido vs sessões normais (review é tipicamente rápido; evita hang longo).
func (c *Catalog) mustReviewTimeout() events.ActivityTimeout {
	t, _ := events.NewActivityTimeout(5 * time.Minute)
	return t
}

// ParseReviewStatusForTest expõe parseReviewStatus para testes externos.
// Não usar em produção.
func (c *Catalog) ParseReviewStatusForTest(output string) string {
	return NewCatalog().parseReviewStatus(output)
}

// BuildReviewPromptForTest expõe buildReviewPrompt para testes externos.
// Não usar em produção.
func (c *Catalog) BuildReviewPromptForTest(skillBody, gitDiff string) string {
	return NewCatalog().buildReviewPrompt(skillBody, gitDiff)
}
