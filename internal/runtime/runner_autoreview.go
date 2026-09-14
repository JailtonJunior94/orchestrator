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
	"github.com/JailtonJunior94/ai-spec-harness/internal/invocation"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

const (
	// _reviewSkillPath é o caminho relativo da skill de review lida em runtime.
	// Falha de leitura gera erro claro — não spawna review sem skill.
	_reviewSkillPath = ".agents/skills/review/SKILL.md"

	// _reviewDiffMaxBytes é o limite default de bytes do diff (5 MB).
	// Configurável via env AISPEC_REVIEW_DIFF_MAX.
	_reviewDiffMaxBytes = 5 * 1024 * 1024

	envReviewPriorSHA = "AI_REVIEW_PRIOR_SHA"

	roundReviewEvidenceFile = "review.md"
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

const reviewDiffUnavailable = "(sem diff disponível)"

const envReviewBaseRef = "AISPEC_REVIEW_BASE_REF"

func ResolveReviewBaseRef(ctx context.Context, workDir string) string {
	if candidate := strings.TrimSpace(os.Getenv(envReviewBaseRef)); candidate != "" {
		if resolved, ok := verifyGitRef(ctx, workDir, candidate); ok {
			return resolved
		}
	}
	if resolved, ok := verifyGitRef(ctx, workDir, "@{upstream}"); ok {
		return resolved
	}
	if base, ok := gitOutput(ctx, workDir, "merge-base", "HEAD", "origin/HEAD"); ok {
		return base
	}
	return ""
}

func verifyGitRef(ctx context.Context, workDir, ref string) (string, bool) {
	return gitOutput(ctx, workDir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
}

func gitOutput(ctx context.Context, workDir string, args ...string) (string, bool) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", false
	}
	value := strings.TrimSpace(out.String())
	return value, value != ""
}

func (c *Catalog) collectGitDiff(workDir string) string {
	return NewCatalog().collectGitDiffContext(context.Background(), workDir)
}

func (c *Catalog) collectGitDiffContext(ctx context.Context, workDir string) string {
	maxBytes := _reviewDiffMaxBytes
	if envVal := os.Getenv("AISPEC_REVIEW_DIFF_MAX"); envVal != "" {
		var n int
		if _, err := fmt.Sscan(envVal, &n); err == nil && n > 0 {
			maxBytes = n
		}
	}

	var sb strings.Builder

	if base := ResolveReviewBaseRef(ctx, workDir); base != "" {
		committed, err := NewCatalog().runGitDiff(workDir, base, "HEAD")
		if err == nil && strings.TrimSpace(committed) != "" {
			sb.WriteString(committed)
		}
	}

	staged, err := NewCatalog().runGitDiff(workDir, "--staged")
	if err == nil && staged != "" {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(staged)
	}

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
		result = reviewDiffUnavailable
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

type autoReviewOutputFn func(ctx context.Context, j Job) (string, error)

func (r *ACPRunner) runAutoReview(ctx context.Context, j Job) (ReviewResult, error) {
	return r.runAutoReviewRound(ctx, j, 1, "")
}

func (r *ACPRunner) runAutoReviewRound(ctx context.Context, j Job, round int, priorSHA string) (ReviewResult, error) {
	if round < 1 {
		return ReviewResult{}, fmt.Errorf("runAutoReviewRound: round %d below one", round)
	}

	skillBody, err := r.readReviewSkill(j.WorkDir)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("runAutoReviewRound: %w", err)
	}

	gitDiff := NewCatalog().collectGitDiff(j.WorkDir)
	prompt := NewCatalog().buildReviewPrompt(skillBody, gitDiff)

	evidenceWriter := NewRoundEvidenceWriter(j.EvidenceDir)
	reviewEvidenceDir := evidenceWriter.Dir(round)

	childJob := Job{
		Prompt:      prompt,
		WorkDir:     j.WorkDir,
		EvidenceDir: reviewEvidenceDir,
		RuntimeConfig: RuntimeConfig{
			Timeout: NewCatalog().mustReviewTimeout(),
		},
		Quiet:          true,
		TasksDir:       j.TasksDir,
		TaskFileName:   j.TaskFileName,
		DisableHooks:   j.DisableHooks,
		SkipDriftGuard: j.SkipDriftGuard,
		AutoReview:     false,
	}

	restoreEnv := NewCatalog().applyRoundReviewEnv(round, priorSHA)
	reviewOutput, runErr := r.spawnReviewSession(ctx, childJob)
	restoreEnv()

	status := NewCatalog().translateReviewStatus(reviewOutput)
	hardIssues := NewCatalog().extractHardIssues(reviewOutput)

	evidencePath, writeErr := evidenceWriter.Write(round, reviewOutput)
	if writeErr != nil {
		return ReviewResult{}, writeErr
	}

	return ReviewResult{
		Status:     status,
		Path:       evidencePath,
		Output:     reviewOutput,
		HardIssues: hardIssues,
	}, runErr
}

func (c *Catalog) roundReviewEvidenceDir(evidenceDir string, round int) string {
	return filepath.Join(evidenceDir, "review", fmt.Sprintf("round-%d", round))
}

func (c *Catalog) writeRoundReviewEvidence(roundDir, content string) (string, error) {
	if err := os.MkdirAll(roundDir, 0o755); err != nil {
		return "", fmt.Errorf("write round review evidence: %w", err)
	}
	path := filepath.Join(roundDir, roundReviewEvidenceFile)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("write round review evidence at %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString(content); err != nil {
		return "", fmt.Errorf("write round review evidence at %q: %w", path, err)
	}
	return path, nil
}

func ApplyRoundReviewEnv(round int, priorSHA string) func() {
	return NewCatalog().applyRoundReviewEnv(round, priorSHA)
}

func (c *Catalog) applyRoundReviewEnv(round int, priorSHA string) func() {
	restoreDepth := invocation.NewGuard().ResetDepth()
	restorePrior := c.applyPriorReviewSHA(round, priorSHA)
	return func() {
		restorePrior()
		restoreDepth()
	}
}

func (c *Catalog) applyPriorReviewSHA(round int, priorSHA string) func() {
	previous, had := os.LookupEnv(envReviewPriorSHA)
	if round > 1 && priorSHA != "" {
		_ = os.Setenv(envReviewPriorSHA, priorSHA)
	} else {
		_ = os.Unsetenv(envReviewPriorSHA)
	}
	return func() {
		if had {
			_ = os.Setenv(envReviewPriorSHA, previous)
			return
		}
		_ = os.Unsetenv(envReviewPriorSHA)
	}
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

// mustReviewTimeout retorna 5*time.Minute como ActivityTimeout para sessões de review.
// Timeout reduzido vs sessões normais (review é tipicamente rápido; evita hang longo).
func (c *Catalog) mustReviewTimeout() events.ActivityTimeout {
	t, _ := events.NewActivityTimeout(5 * time.Minute)
	return t
}
