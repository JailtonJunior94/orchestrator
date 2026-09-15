package taskloop

import (
	"context"
	"errors"
	"testing"
)

type scriptedDiffCapturer struct {
	script   []string
	captures int
}

func (d *scriptedDiffCapturer) CaptureDiff(ctx context.Context) (string, error) {
	index := d.captures
	d.captures++
	if index >= len(d.script) {
		return d.script[len(d.script)-1], nil
	}
	return d.script[index], nil
}

func emptyDiffRunLoopDeps(reviewer *stubReviewer, capturer DiffCapturer) RunLoopDeps {
	return RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  capturer,
	}
}

func twoRoundRejectingReviewer() *stubReviewer {
	return &stubReviewer{results: []FinalReviewResult{
		{
			Verdict:   VerdictRejected,
			Findings:  []Finding{{Severity: SeverityCritical, File: "a.go", Line: 10, Message: "achado da rodada 1"}},
			RawOutput: rawVerdict(VerdictRejected),
		},
		{
			Verdict:   VerdictRejected,
			Findings:  []Finding{{Severity: SeverityCritical, File: "b.go", Line: 20, Message: "achado da rodada 2"}},
			RawOutput: rawVerdict(VerdictRejected),
		},
	}}
}

func TestRunLoopClosesCycleWithEmptyDiffWhenFixProducesNoDelta(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := newCycleTestService(fsys, newTestPrinter())

	capturer := &scriptedDiffCapturer{script: []string{"D1", "D1", "D1", "D2", "D2"}}
	deps := emptyDiffRunLoopDeps(twoRoundRejectingReviewer(), capturer)

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 5}, deps)

	if !errors.Is(err, ErrBugfixExhausted) {
		t.Fatalf("err=%v, want ErrBugfixExhausted", err)
	}
	if report.CycleStopReason != "empty_diff" {
		t.Fatalf("CycleStopReason=%q, want empty_diff", report.CycleStopReason)
	}
	if !report.Escalated {
		t.Error("Escalated=false, want true when the cycle closes without approval")
	}
	if capturer.captures != 5 {
		t.Errorf("captures=%d, want 5 (checkpoint, full target, post-fix delta, checkpoint, delta)", capturer.captures)
	}
}

func TestRunLoopDoesNotCloseWithEmptyDiffWhenFixProducesDelta(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := newCycleTestService(fsys, newTestPrinter())

	capturer := &scriptedDiffCapturer{script: []string{"D1", "D1", "D2", "D3", "D4"}}
	deps := emptyDiffRunLoopDeps(twoRoundRejectingReviewer(), capturer)

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 2}, deps)

	if !errors.Is(err, ErrBugfixExhausted) {
		t.Fatalf("err=%v, want ErrBugfixExhausted", err)
	}
	if report.CycleStopReason == "empty_diff" {
		t.Fatalf("CycleStopReason=empty_diff, want a different reason when every fix round changes the diff")
	}
	if report.CycleStopReason != "max_rounds" {
		t.Errorf("CycleStopReason=%q, want max_rounds", report.CycleStopReason)
	}
}
