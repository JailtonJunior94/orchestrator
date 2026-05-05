package taskloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LoopReport agrega o resultado da execucao de Service.RunLoop (RF-01, RF-02, RF-05, RF-07).
// Serializado em JSON estavel: ordem dos campos segue a definicao da struct.
type LoopReport struct {
	PRDFolder      string             `json:"prd_folder"`
	StartTime      time.Time          `json:"start_time"`
	EndTime        time.Time          `json:"end_time"`
	TasksCompleted []string           `json:"tasks_completed"`
	FinalReview    *FinalReviewResult `json:"final_review,omitempty"`
	BugfixCycles   int                `json:"bugfix_cycles"`
	Escalated      bool               `json:"escalated"`
	ActionPlan     *ActionPlan        `json:"action_plan,omitempty"`
	StopReason     string             `json:"stop_reason"`
}

// TaskExecutor abstrai a invocacao da skill execute-task em uma unica task.
// Implementacoes devem aplicar as alteracoes da task no filesystem
// (incluindo atualizacao de status para "done") antes de retornar.
type TaskExecutor interface {
	Execute(ctx context.Context, task TaskEntry, taskFile, prdFolder, workDir string) error
}

// TaskExecutorFunc adapta uma funcao a TaskExecutor.
type TaskExecutorFunc func(ctx context.Context, task TaskEntry, taskFile, prdFolder, workDir string) error

// Execute invoca a funcao adaptada.
func (f TaskExecutorFunc) Execute(ctx context.Context, task TaskEntry, taskFile, prdFolder, workDir string) error {
	return f(ctx, task, taskFile, prdFolder, workDir)
}

// RunLoopDeps agrupa as dependencias injetaveis do orquestrador.
// Em producao, o cmd layer monta as implementacoes reais; em testes,
// stubs verificam comportamento de orquestracao isoladamente.
type RunLoopDeps struct {
	Selector      TaskSelector
	Executor      TaskExecutor
	Gate          AcceptanceGate
	Recorder      EvidenceRecorder
	FinalReviewer FinalReviewer
	BugfixInvoker BugfixInvoker
	DiffCapturer  DiffCapturer
	Prompter      Prompter
}

// RunLoop encadeia: select → execute → acceptance → evidence → (loop ate esgotar)
// → final review (uma unica vez) → bugfix loop ou reservation planner conforme veredito.
// Retorna LoopReport mesmo em caminhos de erro parciais para preservar auditoria;
// os erros sao propagados via segundo retorno.
func (s *Service) RunLoop(ctx context.Context, opts Options, deps RunLoopDeps) (*LoopReport, error) {
	if deps.Selector == nil {
		return nil, fmt.Errorf("taskloop: RunLoop requer Selector")
	}
	if deps.Executor == nil {
		return nil, fmt.Errorf("taskloop: RunLoop requer Executor")
	}
	if deps.Gate == nil {
		return nil, fmt.Errorf("taskloop: RunLoop requer Gate")
	}
	if deps.Recorder == nil {
		return nil, fmt.Errorf("taskloop: RunLoop requer Recorder")
	}
	if deps.FinalReviewer == nil {
		return nil, fmt.Errorf("taskloop: RunLoop requer FinalReviewer")
	}

	absFolder, err := filepath.Abs(opts.PRDFolder)
	if err != nil {
		return nil, fmt.Errorf("taskloop: caminho invalido %q: %w", opts.PRDFolder, err)
	}
	for _, required := range []string{"tasks.md", "prd.md", "techspec.md"} {
		if !s.fsys.Exists(filepath.Join(absFolder, required)) {
			return nil, fmt.Errorf("taskloop: arquivo obrigatorio nao encontrado: %s",
				filepath.Join(absFolder, required))
		}
	}

	workDir, err := resolveWorkDir(absFolder, s.fsys)
	if err != nil {
		return nil, fmt.Errorf("taskloop: erro ao resolver diretorio de trabalho: %w", err)
	}

	report := &LoopReport{
		PRDFolder:      opts.PRDFolder,
		StartTime:      time.Now(),
		TasksCompleted: []string{},
	}
	var lastTaskFile string

	for {
		if err := ctx.Err(); err != nil {
			return s.finalizeReport(report, opts, "contexto cancelado"), err
		}

		next, err := deps.Selector.Next(ctx, absFolder)
		if errors.Is(err, ErrNoEligibleTask) {
			break
		}
		if err != nil {
			return s.finalizeReport(report, opts, "erro na selecao de task"), err
		}

		taskFile, err := ResolveTaskFile(absFolder, *next, s.fsys)
		if err != nil {
			return s.finalizeReport(report, opts, "erro ao resolver arquivo da task"), err
		}
		lastTaskFile = taskFile

		if err := deps.Executor.Execute(ctx, *next, taskFile, absFolder, workDir); err != nil {
			return s.finalizeReport(report, opts, "erro na execucao da task"),
				fmt.Errorf("taskloop: execucao da task %s: %w", next.ID, err)
		}

		accReport, accErr := deps.Gate.Verify(ctx, *next, taskFile)
		if accErr != nil {
			emitTelemetry("acceptance_failed", next.ID)
			return s.finalizeReport(report, opts, "criterios de aceite nao atendidos"),
				fmt.Errorf("taskloop: aceite da task %s: %w", next.ID, accErr)
		}

		if err := deps.Recorder.Append(ctx, taskFile, accReport); err != nil {
			return s.finalizeReport(report, opts, "erro ao registrar evidencia"),
				fmt.Errorf("taskloop: evidencia da task %s: %w", next.ID, err)
		}

		emitTelemetry("task_completed", next.ID)
		report.TasksCompleted = append(report.TasksCompleted, next.ID)
	}

	diff := captureGitDiff(ctx, workDir)
	rev, err := deps.FinalReviewer.ReviewConsolidated(ctx, diff)
	if err != nil {
		return s.finalizeReport(report, opts, "erro na revisao final"),
			fmt.Errorf("taskloop: revisao final: %w", err)
	}
	report.FinalReview = &rev

	switch rev.Verdict {
	case VerdictApproved:
		emitTelemetry("final_review_verdict", string(rev.Verdict))

	case VerdictApprovedWithRemarks:
		plan, err := s.resolveActionPlan(ctx, absFolder, lastTaskFile, opts, deps, rev.Findings)
		if err != nil {
			return s.finalizeReport(report, opts, "erro no planner de ressalvas"),
				err
		}
		report.ActionPlan = &plan
		emitTelemetry("final_review_verdict", string(rev.Verdict))
		if stop, err := s.applyImplementDecisions(ctx, absFolder, lastTaskFile, plan, diff, opts, deps, report); stop {
			return s.finalizeReport(report, opts, stopReasonForImplement(err, report)), err
		}

	case VerdictBlocked:
		emitTelemetry("final_review_verdict", string(rev.Verdict))
		return s.finalizeReport(report, opts, "revisao final bloqueada"),
			fmt.Errorf("%w: %s", ErrReviewBlocked, blockedReviewReason(rev.RawOutput))

	case VerdictRejected:
		if deps.BugfixInvoker == nil || deps.DiffCapturer == nil {
			return s.finalizeReport(report, opts, "bugfix loop nao configurado"),
				fmt.Errorf("taskloop: review reprovou mas BugfixInvoker/DiffCapturer ausentes")
		}
		bf := NewBugfixLoop(deps.BugfixInvoker, deps.FinalReviewer, deps.DiffCapturer, opts.MaxBugfixIterations)
		bfReport, bfErr := bf.Run(ctx, rev.Findings, diff)
		report.BugfixCycles = len(bfReport.Iterations)
		report.Escalated = bfReport.Escalated
		if bfReport.FinalReview != nil {
			report.FinalReview = bfReport.FinalReview
		}
		for _, it := range bfReport.Iterations {
			emitTelemetry("bugfix_iteration", fmt.Sprintf("%d:%s", it.Sequence, it.ReviewVerdict))
		}
		if errors.Is(bfErr, ErrBugfixExhausted) {
			if report.FinalReview != nil {
				emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
			}
			emitTelemetry("escalated", "bugfix_exhausted")
			return s.finalizeReport(report, opts, "escalonamento humano apos 3 iteracoes"), bfErr
		}
		if bfErr != nil {
			return s.finalizeReport(report, opts, "erro no bugfix loop"), bfErr
		}
		if report.FinalReview != nil {
			switch report.FinalReview.Verdict {
			case VerdictApproved:
				emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
			case VerdictApprovedWithRemarks:
				plan, err := s.resolveActionPlan(ctx, absFolder, lastTaskFile, opts, deps, report.FinalReview.Findings)
				if err != nil {
					return s.finalizeReport(report, opts, "erro no planner de ressalvas"),
						err
				}
				report.ActionPlan = &plan
				emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
				latestDiff := diff
				if dc := deps.DiffCapturer; dc != nil {
					if d, derr := dc.CaptureDiff(ctx); derr == nil {
						latestDiff = d
					}
				}
				if stop, err := s.applyImplementDecisions(ctx, absFolder, lastTaskFile, plan, latestDiff, opts, deps, report); stop {
					return s.finalizeReport(report, opts, stopReasonForImplement(err, report)), err
				}
			case VerdictBlocked:
				emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
				return s.finalizeReport(report, opts, "revisao final bloqueada"),
					fmt.Errorf("%w: %s", ErrReviewBlocked, blockedReviewReason(report.FinalReview.RawOutput))
			}
		}
	}

	return s.finalizeReport(report, opts, "concluido"), nil
}

// applyImplementDecisions reentra o BugfixLoop quando o ActionPlan possui ao
// menos uma decisao ActionImplement (RF-08(a)). Os findings selecionados sao
// repassados ao BugfixLoop com o mesmo limite de iteracoes; LoopReport e
// atualizado com ciclos adicionais, escalonamento e veredito final.
//
// Retorna (stop, err): stop=true sinaliza que RunLoop deve encerrar
// imediatamente — usado para ErrBugfixExhausted ou outros erros do loop.
func (s *Service) applyImplementDecisions(
	ctx context.Context,
	prdFolder string,
	taskFile string,
	plan ActionPlan,
	diff string,
	opts Options,
	deps RunLoopDeps,
	report *LoopReport,
) (bool, error) {
	implFindings := findingsForImplement(plan)
	if len(implFindings) == 0 {
		return false, nil
	}
	if deps.BugfixInvoker == nil || deps.DiffCapturer == nil {
		return true, fmt.Errorf("taskloop: ActionImplement requer BugfixInvoker e DiffCapturer configurados")
	}

	for _, f := range implFindings {
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		if loc == "" {
			loc = "(sem localizacao)"
		}
		emitTelemetry("implement_promoted", loc)
	}

	bf := NewBugfixLoop(deps.BugfixInvoker, deps.FinalReviewer, deps.DiffCapturer, opts.MaxBugfixIterations)
	bfReport, bfErr := bf.Run(ctx, implFindings, diff)
	report.BugfixCycles += len(bfReport.Iterations)
	if bfReport.Escalated {
		report.Escalated = true
	}
	if bfReport.FinalReview != nil {
		report.FinalReview = bfReport.FinalReview
	}
	for _, it := range bfReport.Iterations {
		emitTelemetry("bugfix_iteration", fmt.Sprintf("%d:%s", it.Sequence, it.ReviewVerdict))
	}
	if errors.Is(bfErr, ErrBugfixExhausted) {
		if report.FinalReview != nil {
			emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
		}
		emitTelemetry("escalated", "bugfix_exhausted")
		return true, bfErr
	}
	if bfErr != nil {
		if report.FinalReview != nil {
			emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
		}
		return true, bfErr
	}
	if report.FinalReview != nil {
		emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
	}
	if report.FinalReview == nil {
		return false, nil
	}

	switch report.FinalReview.Verdict {
	case VerdictApproved:
		return false, nil
	case VerdictApprovedWithRemarks:
		nextPlan, err := s.resolveActionPlan(ctx, prdFolder, taskFile, opts, deps, report.FinalReview.Findings)
		if err != nil {
			return true, err
		}
		report.ActionPlan = &nextPlan

		nextDiff := diff
		capturedDiff, err := deps.DiffCapturer.CaptureDiff(ctx)
		if err != nil {
			return true, fmt.Errorf("taskloop: capturar diff apos ressalvas Implement: %w", err)
		}
		if capturedDiff != "" {
			nextDiff = capturedDiff
		}
		return s.applyImplementDecisions(ctx, prdFolder, taskFile, nextPlan, nextDiff, opts, deps, report)
	case VerdictBlocked:
		return true, fmt.Errorf("%w: %s", ErrReviewBlocked, blockedReviewReason(report.FinalReview.RawOutput))
	default:
		return false, nil
	}
}

// findingsForImplement coleta os findings com decisao ActionImplement e
// promove-os a SeverityCritical: a opcao do operador de "implementar agora"
// expressa que o item deve ser tratado pelo BugfixLoop, que so atua sobre
// Critical. A promocao e local — o report original nao e mutado.
func findingsForImplement(plan ActionPlan) []Finding {
	out := make([]Finding, 0, len(plan.Decisions))
	for _, d := range plan.Decisions {
		if d.Action == ActionImplement {
			f := d.Finding
			f.Severity = SeverityCritical
			out = append(out, f)
		}
	}
	return out
}

func stopReasonForImplement(err error, report *LoopReport) string {
	if errors.Is(err, ErrBugfixExhausted) {
		return "escalonamento humano apos ressalvas Implement"
	}
	if err != nil {
		return "erro no bugfix loop de ressalvas Implement"
	}
	if report != nil && report.Escalated {
		return "escalonamento humano apos ressalvas Implement"
	}
	return "concluido"
}

func (s *Service) resolveActionPlan(
	ctx context.Context,
	prdFolder string,
	taskFile string,
	opts Options,
	deps RunLoopDeps,
	findings []Finding,
) (ActionPlan, error) {
	planner := NewReservationPlanner(deps.Prompter, opts.NonInteractive)
	plan, err := planner.Resolve(ctx, findings)
	if err != nil {
		return ActionPlan{}, fmt.Errorf("taskloop: ressalvas: %w", err)
	}
	if taskFile == "" {
		return ActionPlan{}, fmt.Errorf("taskloop: plano de acao requer arquivo da task final")
	}
	if err := WriteActionPlanToTaskFile(s.fsys, taskFile, plan); err != nil {
		return ActionPlan{}, err
	}
	if err := AppendFollowUpTasks(s.fsys, filepath.Join(prdFolder, "tasks.md"), plan); err != nil {
		return ActionPlan{}, err
	}
	return plan, nil
}

func blockedReviewReason(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "review nao retornou motivo detalhado"
}

// finalizeReport preenche EndTime/StopReason e persiste o report em JSON.
// Falhas de escrita sao logadas mas nao mascaram o erro de orquestracao.
func (s *Service) finalizeReport(report *LoopReport, opts Options, stopReason string) *LoopReport {
	if report.StopReason == "" {
		report.StopReason = stopReason
	}
	report.EndTime = time.Now()
	if opts.ReportPath != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err == nil {
			if writeErr := s.fsys.WriteFile(opts.ReportPath, data); writeErr != nil && s.printer != nil {
				s.printer.Warn("taskloop: erro ao escrever LoopReport em %s: %v", opts.ReportPath, writeErr)
			}
		}
	}
	return report
}

// emitTelemetry escreve um evento em stderr quando GOVERNANCE_TELEMETRY=1.
// Formato: "[taskloop] event=<name> value=<value>".
func emitTelemetry(event, value string) {
	if os.Getenv("GOVERNANCE_TELEMETRY") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "[taskloop] event=%s value=%s ts=%s\n",
		event, value, time.Now().UTC().Format(time.RFC3339))
}
