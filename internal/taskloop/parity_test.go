package taskloop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

type parityOutcome struct {
	approved       bool
	reviewRounds   int
	fixInvocations int
	stopReason     string
}

func extractCycleStopReason(report string) string {
	const marker = "**Cycle Stop Reason:**"
	idx := strings.Index(report, marker)
	if idx == -1 {
		return ""
	}
	rest := report[idx+len(marker):]
	if nl := strings.IndexByte(rest, '\n'); nl != -1 {
		rest = rest[:nl]
	}
	return strings.TrimSpace(rest)
}

func driveServiceExecuteParity(t *testing.T) parityOutcome {
	t.Helper()

	fsys, prd, _ := setupCycleFS(t, "- [ ] build green")

	var reviewerCalls int
	var executorCalls int

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{binary: "claude", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				executorCalls++
				if executorCalls == 1 {
					fsys.Files[filepath.Join(prd, "tasks.md")] = tasksContent("1.0", "Cycle Task", "done")
					return "executor output", "", 0, nil
				}
				return "root cause: nil deref\nFail-before: go test ./... FAIL\nPass-after: go test ./... ok\n", "", 0, nil
			}}, nil
		case "codex":
			return &callbackInvoker{binary: "codex", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				reviewerCalls++
				if reviewerCalls == 1 {
					return "[Critical] [x.go:1] uninitialized variable\n\nVerdict: REJECTED\n", "", 1, nil
				}
				return "clean\n\nVerdict: APPROVED\n", "", 0, nil
			}}, nil
		default:
			return nil, fmt.Errorf("tool not configured in test: %s", tool)
		}
	}

	if err := svc.Execute(cycleOptions(prd)); err != nil {
		t.Fatalf("Service.Execute: unexpected error: %v", err)
	}

	report := readFileString(t, fsys, filepath.Join(prd, "report.md"))
	return parityOutcome{
		approved:       !strings.Contains(report, "blocked") && !strings.Contains(report, "escalated"),
		reviewRounds:   reviewerCalls,
		fixInvocations: executorCalls - 1,
		stopReason:     extractCycleStopReason(report),
	}
}

func driveRunLoopParity(t *testing.T) parityOutcome {
	t.Helper()

	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	critical := []Finding{{Severity: SeverityCritical, File: "x.go", Line: 1, Message: "bug"}}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictRejected, Findings: critical, RawOutput: rawVerdict(VerdictRejected)},
		{Verdict: VerdictApproved, Findings: []Finding{}, RawOutput: rawVerdict(VerdictApproved)},
	}}
	bugfixInvoker := &runloopBugfixInvoker{}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: bugfixInvoker,
		DiffCapturer:  &runloopDiffCapturer{},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if err != nil {
		t.Fatalf("RunLoop: unexpected error: %v", err)
	}

	return parityOutcome{
		approved:       !report.Escalated && report.FinalReview != nil && report.FinalReview.Verdict == VerdictApproved,
		reviewRounds:   reviewer.calls,
		fixInvocations: bugfixInvoker.calls,
		stopReason:     report.CycleStopReason,
	}
}

type parityProber struct{}

func (p *parityProber) EnsureAvailable(_ context.Context, spec specs.Spec) (specs.Launcher, error) {
	return specs.NewBinaryLauncher("/usr/local/bin/" + spec.Command), nil
}

type parityPersistence struct{}

func (p *parityPersistence) AppendEvent(_ events.Event) error                { return nil }
func (p *parityPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *parityPersistence) EnrichReport(_ airuntime.Summary) error          { return nil }

type parityPersistenceFactory struct{}

func (f *parityPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return &parityPersistence{}, nil
}

type parityRenderer struct{}

func (r *parityRenderer) Render(_ events.Event) {}

type paritySequencedCall struct {
	script *acpfake.Script
	before func(workDir string)
}

type paritySequencedClientFactory struct {
	t     *testing.T
	ctx   context.Context
	calls []paritySequencedCall
	idx   int
}

func (f *paritySequencedClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	if f.idx >= len(f.calls) {
		f.t.Fatalf("paritySequencedClientFactory: call %d exceeds planned script (%d entries)", f.idx, len(f.calls))
	}
	call := f.calls[f.idx]
	f.idx++
	if call.before != nil {
		call.before(workDir)
	}
	srv := acpfake.NewServer(call.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("acpfake.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}

func parityReviewScript(text string) *acpfake.Script {
	return acpfake.NewScript().AppendAgentMessage(text).AppendSessionEnd()
}

func newParityGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init")
	run("config", "user.email", "parity@example.com")
	run("config", "user.name", "Parity")
	run("config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("write AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base.txt: %v", err)
	}
	skillDir := filepath.Join(dir, ".agents", "skills", "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Review Skill\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	run("add", ".")
	run("commit", "-m", "base")
	return dir
}

func driveACPRunnerParity(t *testing.T) parityOutcome {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workDir := newParityGitRepo(t)
	tasksDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tasksDir, "task-x.md"),
		[]byte("# Task\n\n## Definition of Done\n\n- [ ] build green\n"), 0o644); err != nil {
		t.Fatalf("write task file: %v", err)
	}

	factory := &paritySequencedClientFactory{t: t, ctx: ctx, calls: []paritySequencedCall{
		{script: parityReviewScript("main session")},
		{script: parityReviewScript("[critical] x.go:1 uninitialized variable\n\nVerdict: REJECTED\n")},
		{script: parityReviewScript("fix applied"), before: func(_ string) {
			if err := os.WriteFile(filepath.Join(workDir, "base.txt"), []byte("base\nfixed\n"), 0o644); err != nil {
				t.Fatalf("write base.txt fix: %v", err)
			}
		}},
		{script: parityReviewScript("clean\n\nVerdict: APPROVED\n")},
	}}

	runner := airuntime.NewACPRunner(specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(&parityProber{}),
		airuntime.NewCatalog().WithClientFactory(factory),
		airuntime.NewCatalog().WithPersistenceFactory(&parityPersistenceFactory{}),
		airuntime.NewCatalog().WithRenderer(&parityRenderer{}),
	)

	job := airuntime.Job{
		Prompt:       "implement task x",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("ACPRunner.Run: unexpected error: %v", err)
	}

	return parityOutcome{
		approved:       summary.CycleStopReason == "approved",
		reviewRounds:   len(summary.CycleRounds),
		fixInvocations: max(len(summary.CycleRounds)-1, 0),
		stopReason:     summary.CycleStopReason,
	}
}

func TestApprovalCycleParityAcrossThreeProductionPaths(t *testing.T) {
	execOutcome := driveServiceExecuteParity(t)
	runLoopOutcome := driveRunLoopParity(t)
	acpOutcome := driveACPRunnerParity(t)

	paths := map[string]parityOutcome{
		"Service.Execute": execOutcome,
		"RunLoop":         runLoopOutcome,
		"ACPRunner":       acpOutcome,
	}

	for name, outcome := range paths {
		if !outcome.approved {
			t.Errorf("%s: approved = false, want true", name)
		}
		if outcome.reviewRounds != 2 {
			t.Errorf("%s: reviewRounds = %d, want 2", name, outcome.reviewRounds)
		}
		if outcome.fixInvocations != 1 {
			t.Errorf("%s: fixInvocations = %d, want 1", name, outcome.fixInvocations)
		}
		if outcome.stopReason != "approved" {
			t.Errorf("%s: stopReason = %q, want approved", name, outcome.stopReason)
		}
	}

	if execOutcome != runLoopOutcome || runLoopOutcome != acpOutcome {
		t.Errorf("parity mismatch across the three paths: Service.Execute=%+v RunLoop=%+v ACPRunner=%+v",
			execOutcome, runLoopOutcome, acpOutcome)
	}
}
