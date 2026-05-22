package runtime_test

// runner_autoreview_test.go: testes unitários para F5-Claude (auto-review opt-in).
// T-REV-01..T-REV-04 + testes de helpers parseReviewStatus e buildReviewPrompt.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// ---- helpers de setup -------------------------------------------------------

// fakeProberForReview implementa Prober retornando launcher fixo.
type fakeProberForReview struct{}

func (p *fakeProberForReview) EnsureAvailable(_ context.Context, spec specs.Spec) (specs.Launcher, error) {
	return specs.NewBinaryLauncher("/usr/local/bin/" + spec.Command), nil
}

// fakeClientFactoryForReview cria clientes ACP a partir de acpfake.Script.
type fakeClientFactoryForReview struct {
	script *acpfake.Script
	ctx    context.Context
	t      *testing.T
}

func (f *fakeClientFactoryForReview) New(workDir string) client.Client {
	f.t.Helper()
	srv := acpfake.NewServer(f.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("acpfake.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}

// fakePersistenceForReview descarta persistência em testes unitários.
type fakePersistenceForReview struct{}

func (p *fakePersistenceForReview) AppendEvent(_ events.Event) error               { return nil }
func (p *fakePersistenceForReview) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *fakePersistenceForReview) EnrichReport(_ airuntime.Summary) error         { return nil }

type fakePersistenceFactoryForReview struct{}

func (f *fakePersistenceFactoryForReview) New(_ string) (airuntime.Persistence, error) {
	return &fakePersistenceForReview{}, nil
}

// fakeRendererForReview descarta output.
type fakeRendererForReview struct{}

func (r *fakeRendererForReview) Render(_ events.Event) {}

// buildRunnerWithReviewFn constrói ACPRunner com reviewOutputFn injetado.
// workDir deve conter AGENTS.md (governance hook).
func buildRunnerWithReviewFn(
	t *testing.T,
	ctx context.Context,
	script *acpfake.Script,
	reviewFn airuntime.ReviewOutputFn,
) *airuntime.ACPRunner {
	t.Helper()
	return airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(&fakeProberForReview{}),
		airuntime.WithClientFactory(&fakeClientFactoryForReview{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(&fakePersistenceFactoryForReview{}),
		airuntime.WithRenderer(&fakeRendererForReview{}),
		airuntime.WithReviewOutputFn(reviewFn),
	)
}

// workDirWithAgentsMDForReview cria TempDir com AGENTS.md.
func workDirWithAgentsMDForReview(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("criar AGENTS.md: %v", err)
	}
	// Criar skill de review mínima para que readReviewSkill não falhe.
	skillDir := filepath.Join(dir, ".agents", "skills", "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("criar skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Review Skill\n\nRevise o diff.\n"), 0o644); err != nil {
		t.Fatalf("criar SKILL.md: %v", err)
	}
	return dir
}

// ---- T-REV-01: AutoReview=false → runAutoReview não chamado ----------------

// TestAutoReviewSkipsWhenFlagFalse — T-REV-01.
// Quando AutoReview=false (default), a sessão não dispara review.
// Critério: Summary.ReviewStatus == "" e Summary.ReviewPath == "".
func TestAutoReviewSkipsWhenFlagFalse(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("executando tarefa sem auto-review").
		AppendSessionEnd()

	reviewCalled := false
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		reviewCalled = true
		return "review output", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	job := airuntime.Job{
		Prompt:      "tarefa sem auto-review",
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AutoReview:  false, // T-REV-01: flag false = review não dispara
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-REV-01: Run falhou: %v", err)
	}

	if reviewCalled {
		t.Error("T-REV-01: reviewOutputFn foi chamada com AutoReview=false — não deveria")
	}
	if summary.ReviewStatus != "" {
		t.Errorf("T-REV-01: ReviewStatus = %q, quero vazio", summary.ReviewStatus)
	}
	if summary.ReviewPath != "" {
		t.Errorf("T-REV-01: ReviewPath = %q, quero vazio", summary.ReviewPath)
	}
}

// ---- T-REV-02: review com [HARD] → ReviewStatus="blocked" ------------------

// TestAutoReviewBlocksOnHardIssue — T-REV-02.
// Quando review output contém "[HARD] eval() detected", ReviewStatus = "blocked".
func TestAutoReviewBlocksOnHardIssue(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("tarefa com código suspeito").
		AppendSessionEnd()

	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return "[HARD] eval() detected — uso de eval() é proibido por segurança", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := workDirWithAgentsMDForReview(t)
	job := airuntime.Job{
		Prompt:      "tarefa com eval()",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AutoReview:  true, // T-REV-02: auto-review ativo
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-REV-02: Run falhou: %v", err)
	}

	if summary.ReviewStatus != "blocked" {
		t.Errorf("T-REV-02: ReviewStatus = %q, quero blocked", summary.ReviewStatus)
	}
	if summary.ReviewPath == "" {
		t.Error("T-REV-02: ReviewPath vazio — deve apontar para review.md")
	}
}

// ---- T-REV-03: sem marcadores hard → ReviewStatus="ok" ---------------------

// TestAutoReviewOkWhenNoHardMarkers — T-REV-03.
// Quando review output não contém marcadores hard, ReviewStatus = "ok".
func TestAutoReviewOkWhenNoHardMarkers(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("tarefa limpa").
		AppendSessionEnd()

	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return "Nenhum issue encontrado. Código aprovado.", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	job := airuntime.Job{
		Prompt:      "tarefa limpa",
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AutoReview:  true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-REV-03: Run falhou: %v", err)
	}

	if summary.ReviewStatus != "ok" {
		t.Errorf("T-REV-03: ReviewStatus = %q, quero ok", summary.ReviewStatus)
	}
}

// ---- T-REV-04: child Job tem AutoReview=false (recursão bloqueada) ----------

// TestAutoReviewDoesntRecurse — T-REV-04.
// HARD: a função reviewFn recebe um Job com AutoReview=false mesmo quando
// o parent tem AutoReview=true. Garante que não há recursão infinita.
func TestAutoReviewDoesntRecurse(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("tarefa pai").
		AppendSessionEnd()

	var childAutoReview *bool
	reviewFn := func(_ context.Context, childJob airuntime.Job) (string, error) {
		v := childJob.AutoReview
		childAutoReview = &v
		return "review ok — sem issues", nil
	}

	runner := buildRunnerWithReviewFn(t, ctx, script, reviewFn)

	job := airuntime.Job{
		Prompt:      "tarefa pai com auto-review",
		WorkDir:     workDirWithAgentsMDForReview(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AutoReview:  true, // parent tem AutoReview=true
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-REV-04: Run falhou: %v", err)
	}

	// HARD: child Job deve ter AutoReview=false.
	if childAutoReview == nil {
		t.Fatal("T-REV-04: reviewOutputFn não foi chamada — auto-review não disparou")
	}
	if *childAutoReview != false {
		t.Errorf("T-REV-04: child Job.AutoReview = true — recursão não bloqueada (HARD violado)")
	}
}

// ---- Testes de helpers puros ------------------------------------------------

// TestParseReviewStatus_AllCases valida todos os casos de parseReviewStatus.
func TestParseReviewStatus_AllCases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{"[HARD] presente", "[HARD] eval() detectado", "blocked"},
		{"BLOQUEADO presente", "BLOQUEADO por política de segurança", "blocked"},
		{"CRÍTICO presente", "issue CRÍTICO encontrado no código", "blocked"},
		{"sem marcadores hard", "Nenhum issue. Código limpo.", "ok"},
		{"vazio", "", "ok"},
		{"[HARD] minúsculo — não deve bloquear", "[hard] algo", "ok"}, // case-sensitive
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := airuntime.ParseReviewStatusForTest(tc.input)
			if got != tc.expect {
				t.Errorf("parseReviewStatus(%q) = %q, quero %q", tc.input, got, tc.expect)
			}
		})
	}
}

// TestBuildReviewPrompt_ContainsSkillAndDiff valida que o prompt contém skill body e diff.
func TestBuildReviewPrompt_ContainsSkillAndDiff(t *testing.T) {
	t.Parallel()

	skillBody := "# Review Skill\n\nRevise o código."
	gitDiff := "diff --git a/foo.go b/foo.go\n+func foo() {}"

	prompt := airuntime.BuildReviewPromptForTest(skillBody, gitDiff)

	if !strings.Contains(prompt, skillBody) {
		t.Error("prompt deve conter skill body")
	}
	if !strings.Contains(prompt, gitDiff) {
		t.Error("prompt deve conter git diff")
	}
	if !strings.Contains(prompt, "## Diff a Revisar") {
		t.Error("prompt deve conter seção '## Diff a Revisar'")
	}
	if !strings.Contains(prompt, "[HARD]") {
		t.Error("prompt deve mencionar marcador [HARD] na instrução")
	}
}
