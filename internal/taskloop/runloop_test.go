package taskloop

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// --- Stubs para RunLoop ---

type stubSelector struct {
	queue []TaskEntry
	calls int
}

func (s *stubSelector) Next(ctx context.Context, prdFolder string) (*TaskEntry, error) {
	s.calls++
	if len(s.queue) == 0 {
		return nil, ErrNoEligibleTask
	}
	t := s.queue[0]
	s.queue = s.queue[1:]
	return &t, nil
}

type stubExecutor struct {
	calls int
	err   error
}

func (e *stubExecutor) Execute(ctx context.Context, task TaskEntry, taskFile, prdFolder, workDir string) error {
	e.calls++
	return e.err
}

type stubGate struct {
	report AcceptanceReport
	err    error
	calls  int
}

func (g *stubGate) Verify(ctx context.Context, task TaskEntry, taskFile string) (AcceptanceReport, error) {
	g.calls++
	r := g.report
	r.TaskID = task.ID
	r.Passed = g.err == nil
	return r, g.err
}

type stubRecorder struct {
	calls int
	err   error
}

func (r *stubRecorder) Append(ctx context.Context, taskFile string, report AcceptanceReport) error {
	r.calls++
	return r.err
}

type stubReviewer struct {
	results []FinalReviewResult
	err     error
	calls   int
	diffs   []string
}

func (r *stubReviewer) ReviewConsolidated(ctx context.Context, diff string) (FinalReviewResult, error) {
	r.calls++
	r.diffs = append(r.diffs, diff)
	if r.err != nil {
		return FinalReviewResult{}, r.err
	}
	if len(r.results) == 0 {
		return FinalReviewResult{Verdict: VerdictApproved}, nil
	}
	out := r.results[0]
	if len(r.results) > 1 {
		r.results = r.results[1:]
	}
	return out, nil
}

type runloopBugfixInvoker struct {
	calls int
	diffs []string
}

func (b *runloopBugfixInvoker) InvokeBugfix(ctx context.Context, findings []Finding, diff string) (string, error) {
	b.calls++
	b.diffs = append(b.diffs, diff)
	return "applied bugfix", nil
}

type runloopDiffCapturer struct{}

func (d *runloopDiffCapturer) CaptureDiff(ctx context.Context) (string, error) {
	return "diff-after-bugfix", nil
}

type runloopPrompter struct {
	action    ResolutionAction
	rationale string
}

func (p *runloopPrompter) Ask(ctx context.Context, f Finding) (ResolutionAction, string, error) {
	return p.action, p.rationale, nil
}

type runloopPromptResponse struct {
	action    ResolutionAction
	rationale string
}

type sequenceRunloopPrompter struct {
	responses []runloopPromptResponse
	calls     int
}

func (p *sequenceRunloopPrompter) Ask(ctx context.Context, f Finding) (ResolutionAction, string, error) {
	if p.calls >= len(p.responses) {
		return "", "", errors.New("stub: sem resposta configurada")
	}
	resp := p.responses[p.calls]
	p.calls++
	return resp.action, resp.rationale, nil
}

// --- Helper FS ---

func setupRunLoopFS(taskIDs []string) (*taskfs.FakeFileSystem, string) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")

	var rows strings.Builder
	rows.WriteString("| # | Título | Status | Dependências | Paralelizável |\n")
	rows.WriteString("|---|--------|--------|--------------|---------------|\n")
	for _, id := range taskIDs {
		rows.WriteString("| " + id + " | T " + id + " | done | — | Não |\n")
		fsys.Files[prd+"/task-"+id+"-t.md"] = []byte("**Status:** done\n\n## Definition of Done\n\n- [x] feito\n")
	}
	fsys.Files[prd+"/tasks.md"] = []byte(rows.String())
	return fsys, prd
}

// TestRunLoopApprovedDirect — caminho feliz: aprovacao direta do reviewer.
func TestRunLoopApprovedDirect(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0", "2.0"})
	svc := NewService(fsys, newTestPrinter())

	deps := RunLoopDeps{
		Selector: &stubSelector{queue: []TaskEntry{
			{ID: "1.0", Title: "T 1.0", Status: "pending"},
			{ID: "2.0", Title: "T 2.0", Status: "pending"},
		}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: &stubReviewer{results: []FinalReviewResult{{Verdict: VerdictApproved}}},
	}

	report, err := svc.RunLoop(context.Background(), Options{
		PRDFolder:  prd,
		ReportPath: prd + "/loop-report.json",
		Timeout:    5 * time.Second,
	}, deps)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got := len(report.TasksCompleted); got != 2 {
		t.Fatalf("TasksCompleted=%d, want 2", got)
	}
	rev := deps.FinalReviewer.(*stubReviewer)
	if rev.calls != 1 {
		t.Errorf("FinalReviewer chamado %d vezes, want 1", rev.calls)
	}
	if len(rev.diffs) != 1 {
		t.Fatalf("payloads da review = %d, want 1", len(rev.diffs))
	}
	for _, want := range []string{
		"Contexto da revisao consolidada:",
		prd + "/prd.md",
		prd + "/techspec.md",
		prd + "/tasks.md",
		prd + "/task-2.0-t.md",
		"Tasks executadas neste lote: 1.0, 2.0",
	} {
		if !strings.Contains(rev.diffs[0], want) {
			t.Fatalf("payload da review nao contem %q:\n%s", want, rev.diffs[0])
		}
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictApproved {
		t.Errorf("verdict final inesperado: %+v", report.FinalReview)
	}
	if report.BugfixCycles != 0 || report.Escalated || report.ActionPlan != nil {
		t.Errorf("estado pos-aprovado inesperado: cycles=%d escalated=%v plan=%v",
			report.BugfixCycles, report.Escalated, report.ActionPlan)
	}

	data, err := fsys.ReadFile(prd + "/loop-report.json")
	if err != nil {
		t.Fatalf("LoopReport nao escrito: %v", err)
	}
	var parsed LoopReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("LoopReport JSON invalido: %v", err)
	}
	if len(parsed.TasksCompleted) != 2 {
		t.Errorf("JSON.TasksCompleted=%v", parsed.TasksCompleted)
	}
}

// TestRunLoopApprovedWithRemarks — gera ActionPlan a partir do prompter (3 decisoes).
func TestRunLoopApprovedWithRemarks(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	findings := []Finding{
		{Severity: SeverityImportant, File: "a.go", Line: 1, Message: "x"},
		{Severity: SeverityImportant, File: "b.go", Line: 2, Message: "y"},
		{Severity: SeveritySuggestion, File: "c.go", Line: 3, Message: "z"},
	}

	cases := []struct {
		name   string
		action ResolutionAction
		rat    string
	}{
		{"implement", ActionImplement, ""},
		{"document", ActionDocument, "follow-up"},
		{"discard", ActionDiscard, "fora de escopo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results := []FinalReviewResult{{Verdict: VerdictApprovedWithRemarks, Findings: findings}}
			if tc.action == ActionImplement {
				// ActionImplement reentra o BugfixLoop; reviewer subsequente aprova.
				results = append(results, FinalReviewResult{Verdict: VerdictApproved})
			}
			deps := RunLoopDeps{
				Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
				Executor:      &stubExecutor{},
				Gate:          &stubGate{},
				Recorder:      &stubRecorder{},
				FinalReviewer: &stubReviewer{results: results},
				BugfixInvoker: &runloopBugfixInvoker{},
				DiffCapturer:  &runloopDiffCapturer{},
				Prompter:      &runloopPrompter{action: tc.action, rationale: tc.rat},
			}
			report, err := svc.RunLoop(context.Background(), Options{
				PRDFolder:  prd,
				ReportPath: prd + "/loop-report.json",
			}, deps)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if report.ActionPlan == nil {
				t.Fatalf("ActionPlan ausente")
			}
			if got := len(report.ActionPlan.Decisions); got != len(findings) {
				t.Errorf("decisoes=%d, want %d", got, len(findings))
			}
			for _, d := range report.ActionPlan.Decisions {
				if d.Action != tc.action {
					t.Errorf("acao=%s, want %s", d.Action, tc.action)
				}
			}
			taskContent, err := fsys.ReadFile(prd + "/task-1.0-t.md")
			if err != nil {
				t.Fatalf("task file: %v", err)
			}
			if !strings.Contains(string(taskContent), "## Plano de Ação") {
				t.Fatalf("plano de acao nao persistido no arquivo da task:\n%s", taskContent)
			}
			if tc.action == ActionDocument {
				tasksContent, err := fsys.ReadFile(prd + "/tasks.md")
				if err != nil {
					t.Fatalf("tasks.md: %v", err)
				}
				if !strings.Contains(string(tasksContent), "Follow-up:") {
					t.Fatalf("follow-up nao anexado ao tasks.md:\n%s", tasksContent)
				}
			}
		})
	}
}

// TestRunLoopRejectedThenBugfixApproves — primeira revisao reprova, bugfix faz reviewer aprovar.
func TestRunLoopRejectedThenBugfixApproves(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	critical := []Finding{{Severity: SeverityCritical, File: "x.go", Line: 1, Message: "bug"}}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictRejected, Findings: critical},    // 1a chamada (RunLoop)
		{Verdict: VerdictApproved, Findings: []Finding{}}, // dentro do bugfix loop
	}}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if report.Escalated {
		t.Error("nao deveria escalar quando bugfix corrige")
	}
	if report.BugfixCycles != 1 {
		t.Errorf("BugfixCycles=%d, want 1", report.BugfixCycles)
	}
	if deps.BugfixInvoker.(*runloopBugfixInvoker).calls != 1 {
		t.Errorf("bugfix invoker chamado %d vezes, want 1", deps.BugfixInvoker.(*runloopBugfixInvoker).calls)
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictApproved {
		t.Fatalf("verdict final do report = %+v, want APPROVED", report.FinalReview)
	}
}

// TestRunLoopRejectedThenBugfixWithRemarks — correcao elimina criticos, mas
// ressalvas remanescentes ainda precisam virar ActionPlan persistido.
func TestRunLoopRejectedThenBugfixWithRemarks(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	critical := []Finding{{Severity: SeverityCritical, File: "x.go", Line: 1, Message: "bug"}}
	remarks := []Finding{{Severity: SeverityImportant, File: "a.go", Line: 2, Message: "documentar follow-up"}}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictRejected, Findings: critical},
		{Verdict: VerdictApprovedWithRemarks, Findings: remarks},
	}}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
	}

	report, err := svc.RunLoop(context.Background(), Options{
		PRDFolder:      prd,
		NonInteractive: true,
	}, deps)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictApprovedWithRemarks {
		t.Fatalf("verdict final = %+v, want APPROVED_WITH_REMARKS", report.FinalReview)
	}
	if report.ActionPlan == nil || len(report.ActionPlan.Decisions) != 1 {
		t.Fatalf("ActionPlan invalido: %+v", report.ActionPlan)
	}
	taskContent, err := fsys.ReadFile(prd + "/task-1.0-t.md")
	if err != nil {
		t.Fatalf("task file: %v", err)
	}
	if !strings.Contains(string(taskContent), "## Plano de Ação") {
		t.Fatalf("plano de acao nao persistido apos bugfix:\n%s", taskContent)
	}
}

// TestRunLoopRejectedEscalated — bugfix exausto apos 3 iteracoes.
func TestRunLoopRejectedEscalated(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	critical := []Finding{{Severity: SeverityCritical, File: "x.go", Line: 1, Message: "bug"}}
	// Reviewer sempre retorna criticos.
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictRejected, Findings: critical},
		{Verdict: VerdictRejected, Findings: critical},
		{Verdict: VerdictRejected, Findings: critical},
		{Verdict: VerdictRejected, Findings: critical},
	}}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if !errors.Is(err, ErrBugfixExhausted) {
		t.Fatalf("err=%v, want ErrBugfixExhausted", err)
	}
	if !report.Escalated {
		t.Error("Escalated=false, want true")
	}
	if report.BugfixCycles != 3 {
		t.Errorf("BugfixCycles=%d, want 3", report.BugfixCycles)
	}
	if report.StopReason == "" {
		t.Error("StopReason vazio em escalonamento")
	}
}

// TestRunLoopValidatesDeps — dependencias obrigatorias ausentes geram erro descritivo.
func TestRunLoopValidatesDeps(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	cases := []struct {
		name string
		deps RunLoopDeps
		want string
	}{
		{"sem selector", RunLoopDeps{}, "Selector"},
		{"sem executor", RunLoopDeps{Selector: &stubSelector{}}, "Executor"},
		{"sem gate", RunLoopDeps{Selector: &stubSelector{}, Executor: &stubExecutor{}}, "Gate"},
		{"sem recorder", RunLoopDeps{Selector: &stubSelector{}, Executor: &stubExecutor{}, Gate: &stubGate{}}, "Recorder"},
		{"sem reviewer", RunLoopDeps{Selector: &stubSelector{}, Executor: &stubExecutor{}, Gate: &stubGate{}, Recorder: &stubRecorder{}}, "FinalReviewer"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd}, tc.deps)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err=%v, deveria mencionar %q", err, tc.want)
			}
		})
	}
}

// TestRunLoopApprovedWithRemarksImplementReentersBugfix — regressao RF-08(a):
// quando o operador escolhe ActionImplement em uma ressalva, o BugfixLoop deve
// ser reentrado com o finding correspondente (promovido a Critical), o
// reviewer subsequente roda e LoopReport reflete os ciclos adicionais.
func TestRunLoopApprovedWithRemarksImplementReentersBugfix(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	findings := []Finding{
		{Severity: SeverityImportant, File: "a.go", Line: 1, Message: "ressalva 1"},
		{Severity: SeverityImportant, File: "b.go", Line: 2, Message: "ressalva 2"},
	}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictApprovedWithRemarks, Findings: findings},
		{Verdict: VerdictApproved, Findings: nil},
	}}
	bf := &runloopBugfixInvoker{}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: bf,
		DiffCapturer:  &runloopDiffCapturer{},
		Prompter:      &runloopPrompter{action: ActionImplement, rationale: ""},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if bf.calls != 1 {
		t.Errorf("BugfixInvoker chamado %d vezes, want 1 (Implement deve reentrar bugfix)", bf.calls)
	}
	if report.BugfixCycles != 1 {
		t.Errorf("BugfixCycles=%d, want 1", report.BugfixCycles)
	}
	if report.Escalated {
		t.Error("nao deveria escalar quando bugfix de Implement aprova")
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictApproved {
		t.Fatalf("FinalReview=%+v, want APPROVED apos Implement", report.FinalReview)
	}
}

// TestRunLoopApprovedWithRemarksImplementExhaustsEscalates — regressao RF-08(a)
// + ADR-003: Implement entra no BugfixLoop e respeita o limite rigido de 3
// iteracoes; ao exaurir, escalonamento humano e sinalizado.
func TestRunLoopApprovedWithRemarksImplementExhaustsEscalates(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	findings := []Finding{{Severity: SeverityImportant, File: "a.go", Line: 1, Message: "ressalva persistente"}}
	// Cinco resultados: 1 RunLoop inicial + 3 ciclos de bugfix sem aprovar.
	results := []FinalReviewResult{
		{Verdict: VerdictApprovedWithRemarks, Findings: findings},
	}
	for i := 0; i < 4; i++ {
		results = append(results, FinalReviewResult{
			Verdict:  VerdictRejected,
			Findings: []Finding{{Severity: SeverityCritical, File: "a.go", Line: 1, Message: "ressalva persistente"}},
		})
	}
	reviewer := &stubReviewer{results: results}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
		Prompter:      &runloopPrompter{action: ActionImplement, rationale: ""},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if !errors.Is(err, ErrBugfixExhausted) {
		t.Fatalf("err=%v, want ErrBugfixExhausted", err)
	}
	if !report.Escalated {
		t.Error("Escalated=false apos exaurir Implement, want true")
	}
	if report.BugfixCycles != 3 {
		t.Errorf("BugfixCycles=%d, want 3", report.BugfixCycles)
	}
}

func TestRunLoopApprovedWithRemarksImplementThenRemarksNeedsNewPlan(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	initial := []Finding{{Severity: SeverityImportant, File: "a.go", Line: 1, Message: "implementar ajuste"}}
	secondRound := []Finding{{Severity: SeveritySuggestion, File: "b.go", Line: 2, Message: "documentar follow-up"}}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictApprovedWithRemarks, Findings: initial},
		{Verdict: VerdictApprovedWithRemarks, Findings: secondRound},
	}}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
		Prompter: &sequenceRunloopPrompter{responses: []runloopPromptResponse{
			{action: ActionImplement},
			{action: ActionDocument, rationale: "registrar follow-up"},
		}},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictApprovedWithRemarks {
		t.Fatalf("FinalReview=%+v, want APPROVED_WITH_REMARKS", report.FinalReview)
	}
	if report.ActionPlan == nil || len(report.ActionPlan.Decisions) != 1 {
		t.Fatalf("ActionPlan invalido: %+v", report.ActionPlan)
	}
	if report.ActionPlan.Decisions[0].Action != ActionDocument {
		t.Fatalf("acao final = %s, want %s", report.ActionPlan.Decisions[0].Action, ActionDocument)
	}
	taskContent, err := fsys.ReadFile(prd + "/task-1.0-t.md")
	if err != nil {
		t.Fatalf("task file: %v", err)
	}
	if !strings.Contains(string(taskContent), "documentar follow-up") {
		t.Fatalf("novo plano de acao nao persistido apos Implement:\n%s", taskContent)
	}
	tasksContent, err := fsys.ReadFile(prd + "/tasks.md")
	if err != nil {
		t.Fatalf("tasks.md: %v", err)
	}
	if !strings.Contains(string(tasksContent), "Follow-up:") {
		t.Fatalf("follow-up nao anexado apos segunda rodada de remarks:\n%s", tasksContent)
	}
}

// TestRunLoopRemarksImplementRecursionPreservesContext — regressao: na recursao
// de applyImplementDecisions disparada por APPROVED_WITH_REMARKS+Implement, o
// diff recapturado deve ser reembrulhado por buildFinalReviewInput, mantendo
// PRD/TechSpec/Tasks no payload entregue ao reviewer em todas as iteracoes.
// Adicionalmente, o BugfixInvoker deve receber apenas o diff puro.
func TestRunLoopRemarksImplementRecursionPreservesContext(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	first := []Finding{{Severity: SeverityImportant, File: "a.go", Line: 1, Message: "ajuste 1"}}
	second := []Finding{{Severity: SeverityImportant, File: "b.go", Line: 2, Message: "ajuste 2"}}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictApprovedWithRemarks, Findings: first},
		{Verdict: VerdictApprovedWithRemarks, Findings: second},
		{Verdict: VerdictApproved},
	}}
	invoker := &runloopBugfixInvoker{}
	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: invoker,
		DiffCapturer:  &runloopDiffCapturer{},
		Prompter: &sequenceRunloopPrompter{responses: []runloopPromptResponse{
			{action: ActionImplement},
			{action: ActionImplement},
		}},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictApproved {
		t.Fatalf("FinalReview=%+v, want APPROVED", report.FinalReview)
	}

	// Reviewer foi chamado 3x: inicial + 2 iteracoes Implement; todas devem
	// preservar o cabecalho de contexto consolidado.
	if len(reviewer.diffs) != 3 {
		t.Fatalf("reviewer.diffs len = %d, want 3", len(reviewer.diffs))
	}
	for i, got := range reviewer.diffs {
		if !strings.HasPrefix(got, "Contexto da revisao consolidada:") {
			t.Fatalf("reviewer chamada %d perdeu o contexto:\n%s", i+1, got)
		}
		if !strings.Contains(got, prd+"/prd.md") {
			t.Fatalf("reviewer chamada %d sem PRD path:\n%s", i+1, got)
		}
	}

	// Invoker deve receber diff puro em todas as chamadas (Bug 2).
	if len(invoker.diffs) == 0 {
		t.Fatalf("invoker nao foi chamado")
	}
	for i, got := range invoker.diffs {
		if strings.Contains(got, "Contexto da revisao consolidada:") {
			t.Fatalf("invoker chamada %d recebeu cabecalho de contexto indevido:\n%s", i+1, got)
		}
	}
}

func TestRunLoopApprovedWithRemarksImplementThenBlocked(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	initial := []Finding{{Severity: SeverityImportant, File: "a.go", Line: 1, Message: "implementar ajuste"}}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictApprovedWithRemarks, Findings: initial},
		{Verdict: VerdictBlocked, RawOutput: "BLOCKED: faltou diff para validar follow-up"},
	}}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
		Prompter:      &runloopPrompter{action: ActionImplement},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	if !errors.Is(err, ErrReviewBlocked) {
		t.Fatalf("err=%v, want ErrReviewBlocked", err)
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictBlocked {
		t.Fatalf("FinalReview=%+v, want BLOCKED", report.FinalReview)
	}
}

// errorBugfixInvoker erra na primeira chamada — usado para validar telemetria
// no caminho bfErr nao-exhausted.
type errorBugfixInvoker struct{ calls int }

func (b *errorBugfixInvoker) InvokeBugfix(ctx context.Context, findings []Finding, diff string) (string, error) {
	b.calls++
	return "", errors.New("invoker explodiu")
}

// TestRunLoopImplementEmitsTelemetry — regressao Suggestion 1 e 4 da review:
// (S4) cada finding promovido para Implement emite "implement_promoted";
// (S1) caminho bfErr nao-exhausted emite "final_review_verdict" preservando
// paridade com o ramo VerdictRejected original.
func TestRunLoopImplementEmitsTelemetry(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	findings := []Finding{
		{Severity: SeverityImportant, File: "a.go", Line: 10, Message: "ressalva A"},
		{Severity: SeveritySuggestion, File: "b.go", Line: 0, Message: "ressalva B"},
	}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictApprovedWithRemarks, Findings: findings},
	}}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &errorBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
		Prompter:      &runloopPrompter{action: ActionImplement, rationale: ""},
	}

	var runErr error
	stderr := captureStderr(t, func() {
		_, runErr = svc.RunLoop(context.Background(), Options{PRDFolder: prd, MaxBugfixIterations: 3}, deps)
	})

	if runErr == nil {
		t.Fatal("esperava erro propagado do BugfixInvoker, recebeu nil")
	}
	if errors.Is(runErr, ErrBugfixExhausted) {
		t.Fatal("erro nao deveria ser ErrBugfixExhausted neste caminho")
	}

	// S4: cada finding promovido emite implement_promoted com localizacao.
	if !strings.Contains(stderr, "event=implement_promoted value=a.go:10") {
		t.Errorf("evento implement_promoted ausente para a.go:10\nstderr:\n%s", stderr)
	}
	if !strings.Contains(stderr, "event=implement_promoted value=b.go") {
		t.Errorf("evento implement_promoted ausente para b.go (sem linha)\nstderr:\n%s", stderr)
	}
	// S1: bfErr nao-exhausted ainda emite final_review_verdict preservando
	// o veredito anterior (APPROVED_WITH_REMARKS) presente em report.FinalReview.
	if !strings.Contains(stderr, "event=final_review_verdict value=APPROVED_WITH_REMARKS") {
		t.Errorf("final_review_verdict nao emitido no caminho bfErr nao-exhausted\nstderr:\n%s", stderr)
	}
}

func TestRunLoopBlockedReviewer(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, newTestPrinter())

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: &stubReviewer{results: []FinalReviewResult{{Verdict: VerdictBlocked, RawOutput: "BLOCKED: faltou evidencia"}}},
	}

	report, err := svc.RunLoop(context.Background(), Options{PRDFolder: prd}, deps)
	if !errors.Is(err, ErrReviewBlocked) {
		t.Fatalf("err=%v, want ErrReviewBlocked", err)
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictBlocked {
		t.Fatalf("FinalReview=%+v, want BLOCKED", report.FinalReview)
	}
}
