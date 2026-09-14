package taskloop

import (
	"context"
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
)

const runLoopRoundOneEvidence = "/fake/project/evidence/runloop/review/round-1/review.md"

func runLoopApprovedDeps(raw string) RunLoopDeps {
	return RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0", Status: "pending"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: &stubReviewer{results: []FinalReviewResult{{Verdict: VerdictApproved, RawOutput: raw}}},
	}
}

func reviewRequestForRound(t *testing.T, round int, descriptions ...string) approval.ReviewRequest {
	t.Helper()
	task, err := approval.NewTaskIdentity("task-4.3")
	if err != nil {
		t.Fatalf("task identity: %v", err)
	}
	agent, err := approval.NewAgentIdentity("agent-x")
	if err != nil {
		t.Fatalf("agent identity: %v", err)
	}
	var criteria []approval.AcceptanceCriterion
	for _, description := range descriptions {
		criterion, criterionErr := approval.NewAcceptanceCriterion(description)
		if criterionErr != nil {
			t.Fatalf("criterion %q: %v", description, criterionErr)
		}
		criteria = append(criteria, criterion)
	}
	request, err := approval.NewReviewRequest(task, agent, round, approval.NewReviewTarget("diff"), criteria)
	if err != nil {
		t.Fatalf("review request round %d: %v", round, err)
	}
	return request
}

func TestRunLoopApprovedConfrontsCriteriaMapBeforeClosingTheBatch(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := newCycleTestService(fsys, newTestPrinter())

	deps := runLoopApprovedDeps("Verdict: APPROVED\n")
	deps.BugfixInvoker = &runloopBugfixInvoker{}
	deps.DiffCapturer = &runloopDiffCapturer{}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd}, deps)
	if err == nil {
		t.Fatal("APPROVED sem mapa 1:1 confrontavel nao pode encerrar o lote como concluido")
	}
	if !errors.Is(err, ErrBugfixExhausted) {
		t.Fatalf("erro = %v, want %v", err, ErrBugfixExhausted)
	}
	if !report.Escalated {
		t.Error("lote sem prova de aceite deve escalar")
	}
	if report.CycleStopReason == "" {
		t.Error("ciclo nao foi conduzido no ramo APPROVED: CycleStopReason vazio")
	}
	if len(report.CycleRounds) == 0 {
		t.Error("ciclo nao registrou rodadas no ramo APPROVED")
	}
}

func TestRunLoopApprovedWithCompleteCriteriaMapClosesThroughTheCycle(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := newCycleTestService(fsys, newTestPrinter())

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd}, runLoopApprovedDeps(rawVerdict(VerdictApproved)))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if report.CycleStopReason != "approved" {
		t.Errorf("CycleStopReason = %q, want approved", report.CycleStopReason)
	}
	if len(report.CycleRounds) != 1 {
		t.Fatalf("CycleRounds = %d, want 1", len(report.CycleRounds))
	}
	if report.CycleRounds[0].Verdict != "APPROVED" {
		t.Errorf("CycleRounds[0].Verdict = %q, want APPROVED", report.CycleRounds[0].Verdict)
	}
	if report.Escalated {
		t.Error("lote aprovado com prova completa nao deve escalar")
	}
}

func TestRunLoopWritesNumberedRoundEvidence(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := newCycleTestService(fsys, newTestPrinter())

	if _, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd}, runLoopApprovedDeps(rawVerdict(VerdictApproved))); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	data, ok := fsys.Files[runLoopRoundOneEvidence]
	if !ok {
		t.Fatalf("evidencia por rodada ausente em %s", runLoopRoundOneEvidence)
	}
	if string(data) != rawVerdict(VerdictApproved) {
		t.Errorf("conteudo da evidencia = %q, want a saida bruta da revisao", data)
	}
}

func TestRunLoopRoundEvidenceAddressIsImmutable(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := newCycleTestService(fsys, newTestPrinter())

	if _, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd}, runLoopApprovedDeps(rawVerdict(VerdictApproved))); err != nil {
		t.Fatalf("primeira execucao: %v", err)
	}
	if _, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd}, runLoopApprovedDeps(rawVerdict(VerdictApproved))); err == nil {
		t.Fatal("reescrever a evidencia da rodada 1 deve falhar")
	}
	if string(fsys.Files[runLoopRoundOneEvidence]) != rawVerdict(VerdictApproved) {
		t.Error("evidencia da rodada 1 foi mutada")
	}
}

func TestReviewerPortRoundEvidenceAddressesAreDistinctPerRound(t *testing.T) {
	fsys := taskfs.NewFakeFileSystem()
	writer := airuntime.NewRoundEvidenceWriterWithSink("/fake/evidence/task-1.0", fsys)
	reviewer := &stubFinalReviewer{results: []FinalReviewResult{
		{RawOutput: "Verdict: REJECTED\n"},
		{RawOutput: "Verdict: APPROVED\n"},
	}}
	port := newReviewerPort(reviewer, writer, nil)

	for round := 1; round <= 2; round++ {
		if _, err := port.Review(context.Background(), reviewRequestForRound(t, round, "builds green")); err != nil {
			t.Fatalf("rodada %d: %v", round, err)
		}
	}

	const first = "/fake/evidence/task-1.0/review/round-1/review.md"
	const second = "/fake/evidence/task-1.0/review/round-2/review.md"
	if string(fsys.Files[first]) != "Verdict: REJECTED\n" {
		t.Errorf("rodada 1 = %q", fsys.Files[first])
	}
	if string(fsys.Files[second]) != "Verdict: APPROVED\n" {
		t.Errorf("rodada 2 = %q", fsys.Files[second])
	}

	rewrite := newReviewerPort(&stubFinalReviewer{results: []FinalReviewResult{{RawOutput: "overwrite"}}}, writer, nil)
	if _, err := rewrite.Review(context.Background(), reviewRequestForRound(t, 1, "builds green")); err == nil {
		t.Fatal("reescrever o endereco da rodada 1 deve falhar")
	}
	if string(fsys.Files[first]) != "Verdict: REJECTED\n" {
		t.Errorf("evidencia da rodada 1 foi mutada: %q", fsys.Files[first])
	}
}
