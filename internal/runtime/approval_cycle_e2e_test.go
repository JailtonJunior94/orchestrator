package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type sequencedCall struct {
	script *acpfake.Script
	before func(workDir string)
}

type sequencedClientFactory struct {
	t     *testing.T
	ctx   context.Context
	calls []sequencedCall
	idx   int
}

func newSequencedClientFactory(t *testing.T, ctx context.Context, calls ...sequencedCall) *sequencedClientFactory {
	t.Helper()
	return &sequencedClientFactory{t: t, ctx: ctx, calls: calls}
}

func (f *sequencedClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	i := f.idx
	if i >= len(f.calls) {
		f.t.Fatalf("sequencedClientFactory: call %d exceeds planned script (%d entries)", i, len(f.calls))
	}
	f.idx++
	call := f.calls[i]
	if call.before != nil {
		call.before(workDir)
	}
	srv := acpfake.NewServer(call.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("acpfake.Start (call %d): %v", i, err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}

func (f *sequencedClientFactory) callCount() int {
	return f.idx
}

func reviewScript(text string) *acpfake.Script {
	return acpfake.NewScript().AppendAgentMessage(text).AppendSessionEnd()
}

func writeRealFileChange(t *testing.T, workDir, content string) func(string) {
	t.Helper()
	return func(_ string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(workDir, "base.txt"), []byte(content), 0o644); err != nil {
			t.Fatalf("writeRealFileChange: %v", err)
		}
	}
}

func buildSequencedRunner(t *testing.T, factory *sequencedClientFactory) *airuntime.ACPRunner {
	t.Helper()
	return airuntime.NewACPRunner(specs.NewCatalog().
		Claude(), airuntime.NewCatalog().WithProber(&fakeProberForReview{}), airuntime.NewCatalog().
		WithClientFactory(factory), airuntime.NewCatalog().
		WithPersistenceFactory(&fakePersistenceFactoryForReview{}), airuntime.NewCatalog().
		WithRenderer(&fakeRendererForReview{}),
	)
}

func newE2ECycleJob(workDir, tasksDir, evidenceDir string) airuntime.Job {
	return airuntime.Job{
		Prompt:       "implement task x",
		WorkDir:      workDir,
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}
}

func TestE2EApprovalCycle_ApprovesFirstRound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript("no outstanding issues\n\nVerdict: APPROVED\n")},
	)
	runner := buildSequencedRunner(t, factory)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	summary, err := runner.Run(ctx, newE2ECycleJob(workDir, tasksDir, t.TempDir()))
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary.CycleStopReason != "approved" {
		t.Errorf("CycleStopReason = %q, want approved", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 1 {
		t.Fatalf("CycleRounds len = %d, want 1", len(summary.CycleRounds))
	}
	if summary.CycleRounds[0].Verdict != "APPROVED" {
		t.Errorf("CycleRounds[0].Verdict = %q, want APPROVED", summary.CycleRounds[0].Verdict)
	}
	if got := factory.callCount(); got != 2 {
		t.Errorf("ACP sessions opened = %d, want 2 (main session + 1 review round through the fake ACP server)", got)
	}
}

func TestE2EApprovalCycle_ApprovesThirdRoundAfterTwoFixes(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript("[high] fix.go:1 initial issue\n\nVerdict: REJECTED\n")},
		sequencedCall{
			script: reviewScript("fix round 1 applied"),
			before: writeRealFileChange(t, workDir, "base\nround1\n"),
		},
		sequencedCall{script: reviewScript("[high] fix.go:2 remaining issue\n\nVerdict: REJECTED\n")},
		sequencedCall{
			script: reviewScript("fix round 2 applied"),
			before: writeRealFileChange(t, workDir, "base\nround1\nround2\n"),
		},
		sequencedCall{script: reviewScript("no outstanding issues\n\nVerdict: APPROVED\n")},
	)
	runner := buildSequencedRunner(t, factory)

	summary, err := runner.Run(ctx, newE2ECycleJob(workDir, tasksDir, t.TempDir()))
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary.CycleStopReason != "approved" {
		t.Errorf("CycleStopReason = %q, want approved", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 3 {
		t.Fatalf("CycleRounds len = %d, want 3 (two fixes before approval)", len(summary.CycleRounds))
	}
	wantVerdicts := []string{"REJECTED", "REJECTED", "APPROVED"}
	for i, want := range wantVerdicts {
		if summary.CycleRounds[i].Verdict != want {
			t.Errorf("CycleRounds[%d].Verdict = %q, want %q", i, summary.CycleRounds[i].Verdict, want)
		}
	}
	if got := factory.callCount(); got != 6 {
		t.Errorf("ACP sessions opened = %d, want 6 (main + 3 reviews + 2 fixes)", got)
	}
}

func TestE2EApprovalCycle_AbortsOnRepeatedFingerprintWithoutSpendingNextRound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript("[high] fix.go:1 same issue\n\nVerdict: REJECTED\n")},
		sequencedCall{
			script: reviewScript("fix attempt"),
			before: writeRealFileChange(t, workDir, "base\nattempt\n"),
		},
		sequencedCall{script: reviewScript("[high] fix.go:1 same issue\n\nVerdict: REJECTED\n")},
	)
	runner := buildSequencedRunner(t, factory)

	summary, err := runner.Run(ctx, newE2ECycleJob(workDir, tasksDir, t.TempDir()))
	if err != nil {
		t.Fatalf("Run failed (a terminal state is not an infrastructure error, RF-45): %v", err)
	}
	if summary.CycleStopReason != "no_convergence" {
		t.Errorf("CycleStopReason = %q, want no_convergence", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 2 {
		t.Fatalf("CycleRounds len = %d, want 2 (repeated fingerprint aborts on detection, no extra review round)", len(summary.CycleRounds))
	}
	if got := factory.callCount(); got != 4 {
		t.Errorf("ACP sessions opened = %d, want 4 (main + review1 + fix + review2); must not spend the next round", got)
	}
}

func TestE2EApprovalCycle_AbortsOnEmptyDiffAfterFix(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript("[high] fix.go:1 issue\n\nVerdict: REJECTED\n")},
		sequencedCall{script: reviewScript("fix attempted without touching any file")},
		sequencedCall{script: reviewScript("[high] fix.go:2 distinct issue\n\nVerdict: REJECTED\n")},
	)
	runner := buildSequencedRunner(t, factory)

	summary, err := runner.Run(ctx, newE2ECycleJob(workDir, tasksDir, t.TempDir()))
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary.CycleStopReason != "empty_diff" {
		t.Errorf("CycleStopReason = %q, want empty_diff (a fix without diff aborts the cycle)", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 2 {
		t.Fatalf("CycleRounds len = %d, want 2", len(summary.CycleRounds))
	}
}

func TestE2EApprovalCycle_RemediationWithoutDiffStillProducesRoundEvidence(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript("[high] fix.go:1 issue\n\nVerdict: REJECTED\n")},
		sequencedCall{script: reviewScript("fix attempted without touching any file")},
		sequencedCall{script: reviewScript("[high] fix.go:2 distinct issue\n\nVerdict: REJECTED\n")},
	)
	runner := buildSequencedRunner(t, factory)

	summary, err := runner.Run(ctx, newE2ECycleJob(workDir, tasksDir, t.TempDir()))
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary.CycleStopReason != "empty_diff" {
		t.Errorf("CycleStopReason = %q, want empty_diff", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 2 {
		t.Fatalf("CycleRounds len = %d, want 2 (remediation was attempted before the abort)", len(summary.CycleRounds))
	}
	for i, round := range summary.CycleRounds {
		if round.Number != i+1 {
			t.Errorf("CycleRounds[%d].Number = %d, want %d", i, round.Number, i+1)
		}
		if round.Verdict != "REJECTED" {
			t.Errorf("CycleRounds[%d].Verdict = %q, want REJECTED", i, round.Verdict)
		}
		if round.FindingsBySeverity["high"] != 1 {
			t.Errorf("CycleRounds[%d].FindingsBySeverity[high] = %d, want 1 (round finding preserved in the evidence)",
				i, round.FindingsBySeverity["high"])
		}
	}
	if got := factory.callCount(); got != 4 {
		t.Errorf("ACP sessions opened = %d, want 4 (remediation was attempted once before the abort)", got)
	}
}

func TestE2EApprovalCycle_ApprovedWithRemarksFeedsBackToFix(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	factory := newSequencedClientFactory(t, ctx,
		sequencedCall{script: reviewScript("main session")},
		sequencedCall{script: reviewScript("[high] fix.go:1 remark to fix\n\nVerdict: APPROVED_WITH_REMARKS\n")},
		sequencedCall{
			script: reviewScript("remark addressed"),
			before: writeRealFileChange(t, workDir, "base\nfixed\n"),
		},
		sequencedCall{script: reviewScript("no outstanding issues\n\nVerdict: APPROVED\n")},
	)
	runner := buildSequencedRunner(t, factory)

	summary, err := runner.Run(ctx, newE2ECycleJob(workDir, tasksDir, t.TempDir()))
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if summary.CycleStopReason != "approved" {
		t.Errorf("CycleStopReason = %q, want approved", summary.CycleStopReason)
	}
	if len(summary.CycleRounds) != 2 {
		t.Fatalf("CycleRounds len = %d, want 2 (the remark fed back into the fix)", len(summary.CycleRounds))
	}
	if summary.CycleRounds[0].Verdict != "APPROVED_WITH_REMARKS" {
		t.Errorf("CycleRounds[0].Verdict = %q, want APPROVED_WITH_REMARKS", summary.CycleRounds[0].Verdict)
	}
	if summary.CycleRounds[0].FindingsBySeverity["high"] != 1 {
		t.Errorf("CycleRounds[0].FindingsBySeverity[high] = %d, want 1", summary.CycleRounds[0].FindingsBySeverity["high"])
	}
	if summary.CycleRounds[1].Verdict != "APPROVED" {
		t.Errorf("CycleRounds[1].Verdict = %q, want APPROVED", summary.CycleRounds[1].Verdict)
	}
	if got := factory.callCount(); got != 4 {
		t.Errorf("ACP sessions opened = %d, want 4 (main + review1 + remark fix + review2)", got)
	}
}
