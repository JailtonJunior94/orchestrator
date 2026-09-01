package taskloop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	BugfixAttempts []BugfixIteration  `json:"bugfix_attempts,omitempty"`
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

var _ TaskExecutor = TaskExecutorFunc(nil)

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

	workDir, err := NewCatalog().resolveWorkDir(absFolder, s.fsys)
	if err != nil {
		return nil, fmt.Errorf("taskloop: erro ao resolver diretorio de trabalho: %w", err)
	}

	report := &LoopReport{
		PRDFolder:      opts.PRDFolder,
		StartTime:      time.Now(),
		TasksCompleted: []string{},
	}
	var lastTaskFile string

	// Normalizar parâmetros de concorrência (ADR-018, RF-05).
	// Concurrent <=0 ou 1 ⇒ sequencial idêntico ao comportamento atual (F1 default).
	concurrent := opts.Concurrent
	if concurrent <= 0 {
		concurrent = 1
	}
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = 1
	}

	for {
		if err := ctx.Err(); err != nil {
			return s.finalizeReport(report, opts, "contexto cancelado"), err
		}

		if concurrent <= 1 && batchSize <= 1 {
			// Caminho sequencial: comportamento byte-equivalente ao atual (F1 default, RF-05).
			next, err := deps.Selector.Next(ctx, absFolder)
			if errors.Is(err, ErrNoEligibleTask) {
				break
			}
			if err != nil {
				return s.finalizeReport(report, opts, "erro na selecao de task"), err
			}

			taskFile, err := NewCatalog().ResolveTaskFile(absFolder, *next, s.fsys)
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
				NewCatalog().emitTelemetry("acceptance_failed", next.ID)
				return s.finalizeReport(report, opts, "criterios de aceite nao atendidos"),
					fmt.Errorf("taskloop: aceite da task %s: %w", next.ID, accErr)
			}

			if err := deps.Recorder.Append(ctx, taskFile, accReport); err != nil {
				return s.finalizeReport(report, opts, "erro ao registrar evidencia"),
					fmt.Errorf("taskloop: evidencia da task %s: %w", next.ID, err)
			}
			NewCatalog().emitTelemetry("task_completed", next.ID)
			report.TasksCompleted = append(report.TasksCompleted, next.ID)
		} else {
			// Caminho concorrente: selecionar até batchSize tasks e executar até concurrent em paralelo.
			// Dependências são respeitadas pelo seletor (apenas tasks com deps completas são elegíveis).
			batch, selErr := NewCatalog().selectBatch(ctx, deps.Selector, absFolder, batchSize)
			if selErr != nil && !errors.Is(selErr, ErrNoEligibleTask) {
				return s.finalizeReport(report, opts, "erro na selecao de batch"), selErr
			}
			if len(batch) == 0 {
				break
			}

			completedIDs, lastFile, batchErr := s.executeBatch(ctx, batch, absFolder, workDir, concurrent, deps)
			if lastFile != "" {
				lastTaskFile = lastFile
			}
			report.TasksCompleted = append(report.TasksCompleted, completedIDs...)
			if batchErr != nil {
				return s.finalizeReport(report, opts, "erro na execucao do batch"), batchErr
			}
		}
	}

	diff := NewCatalog().captureGitDiff(ctx, workDir)
	reviewInput := NewCatalog().buildFinalReviewInput(absFolder, lastTaskFile, report.TasksCompleted, diff)
	rev, err := deps.FinalReviewer.ReviewConsolidated(ctx, reviewInput)
	if err != nil {
		return s.finalizeReport(report, opts, "erro na revisao final"),
			fmt.Errorf("taskloop: revisao final: %w", err)
	}
	report.FinalReview = &rev

	switch rev.Verdict {
	case VerdictApproved:
		NewCatalog().emitTelemetry("final_review_verdict", string(rev.Verdict))

	case VerdictApprovedWithRemarks:
		plan, err := s.resolveActionPlan(ctx, absFolder, lastTaskFile, opts, deps, rev.Findings)
		if err != nil {
			return s.finalizeReport(report, opts, "erro no planner de ressalvas"),
				err
		}
		report.ActionPlan = &plan
		NewCatalog().emitTelemetry("final_review_verdict", string(rev.Verdict))
		if stop, err := s.applyImplementDecisions(ctx, absFolder, lastTaskFile, plan, reviewInput, opts, deps, report); stop {
			return s.finalizeReport(report, opts, NewCatalog().stopReasonForImplement(err, report)), err
		}

	case VerdictBlocked:
		NewCatalog().emitTelemetry("final_review_verdict", string(rev.Verdict))
		return s.finalizeReport(report, opts, "revisao final bloqueada"),
			fmt.Errorf("%w: %s", ErrReviewBlocked, NewCatalog().blockedReviewReason(rev.RawOutput))

	case VerdictRejected:
		if deps.BugfixInvoker == nil || deps.DiffCapturer == nil {
			return s.finalizeReport(report, opts, "bugfix loop nao configurado"),
				fmt.Errorf("taskloop: review reprovou mas BugfixInvoker/DiffCapturer ausentes")
		}
		bf := NewBugfixLoop(deps.BugfixInvoker, deps.FinalReviewer, deps.DiffCapturer, opts.MaxBugfixIterations)
		bfReport, bfErr := bf.Run(ctx, rev.Findings, reviewInput)
		report.BugfixCycles = len(bfReport.Iterations)
		report.BugfixAttempts = append(report.BugfixAttempts, bfReport.Iterations...)
		report.Escalated = bfReport.Escalated
		if bfReport.FinalReview != nil {
			report.FinalReview = bfReport.FinalReview
		}
		for _, it := range bfReport.Iterations {
			NewCatalog().emitTelemetry("bugfix_iteration", fmt.Sprintf("%d:%s", it.Sequence, it.ReviewVerdict))
		}
		if errors.Is(bfErr, ErrBugfixExhausted) {
			if report.FinalReview != nil {
				NewCatalog().emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
			}
			NewCatalog().emitTelemetry("escalated", "bugfix_exhausted")
			return s.finalizeReport(report, opts, "escalonamento humano apos 3 iteracoes"), bfErr
		}
		if bfErr != nil {
			return s.finalizeReport(report, opts, "erro no bugfix loop"), bfErr
		}
		if report.FinalReview != nil {
			switch report.FinalReview.Verdict {
			case VerdictApproved:
				NewCatalog().emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
			case VerdictApprovedWithRemarks:
				plan, err := s.resolveActionPlan(ctx, absFolder, lastTaskFile, opts, deps, report.FinalReview.Findings)
				if err != nil {
					return s.finalizeReport(report, opts, "erro no planner de ressalvas"),
						err
				}
				report.ActionPlan = &plan
				NewCatalog().emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
				latestDiff := NewCatalog().buildFinalReviewInput(absFolder, lastTaskFile, report.TasksCompleted, diff)
				if dc := deps.DiffCapturer; dc != nil {
					if d, derr := dc.CaptureDiff(ctx); derr == nil {
						latestDiff = NewCatalog().buildFinalReviewInput(absFolder, lastTaskFile, report.TasksCompleted, d)
					}
				}
				if stop, err := s.applyImplementDecisions(ctx, absFolder, lastTaskFile, plan, latestDiff, opts, deps, report); stop {
					return s.finalizeReport(report, opts, NewCatalog().stopReasonForImplement(err, report)), err
				}
			case VerdictBlocked:
				NewCatalog().emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
				return s.finalizeReport(report, opts, "revisao final bloqueada"),
					fmt.Errorf("%w: %s", ErrReviewBlocked, NewCatalog().blockedReviewReason(report.FinalReview.RawOutput))
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
	implFindings := NewCatalog().findingsForImplement(plan)
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
		NewCatalog().emitTelemetry("implement_promoted", loc)
	}

	bf := NewBugfixLoop(deps.BugfixInvoker, deps.FinalReviewer, deps.DiffCapturer, opts.MaxBugfixIterations)
	bfReport, bfErr := bf.Run(ctx, implFindings, diff)
	report.BugfixCycles += len(bfReport.Iterations)
	report.BugfixAttempts = append(report.BugfixAttempts, bfReport.Iterations...)
	if bfReport.Escalated {
		report.Escalated = true
	}
	if bfReport.FinalReview != nil {
		report.FinalReview = bfReport.FinalReview
	}
	for _, it := range bfReport.Iterations {
		NewCatalog().emitTelemetry("bugfix_iteration", fmt.Sprintf("%d:%s", it.Sequence, it.ReviewVerdict))
	}
	if errors.Is(bfErr, ErrBugfixExhausted) {
		if report.FinalReview != nil {
			NewCatalog().emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
		}
		NewCatalog().emitTelemetry("escalated", "bugfix_exhausted")
		return true, bfErr
	}
	if bfErr != nil {
		if report.FinalReview != nil {
			NewCatalog().emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
		}
		return true, bfErr
	}
	if report.FinalReview != nil {
		NewCatalog().emitTelemetry("final_review_verdict", string(report.FinalReview.Verdict))
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
			nextDiff = NewCatalog().buildFinalReviewInput(prdFolder, taskFile, report.TasksCompleted, capturedDiff)
		}
		return s.applyImplementDecisions(ctx, prdFolder, taskFile, nextPlan, nextDiff, opts, deps, report)
	case VerdictBlocked:
		return true, fmt.Errorf("%w: %s", ErrReviewBlocked, NewCatalog().blockedReviewReason(report.FinalReview.RawOutput))
	default:
		return false, nil
	}
}

// findingsForImplement coleta os findings com decisao ActionImplement e
// promove-os a SeverityCritical: a opcao do operador de "implementar agora"
// expressa que o item deve ser tratado pelo BugfixLoop, que so atua sobre
// Critical. A promocao e local — o report original nao e mutado.
func (c *Catalog) findingsForImplement(plan ActionPlan) []Finding {
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

func (c *Catalog) stopReasonForImplement(err error, report *LoopReport) string {
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
	if err := NewCatalog().WriteActionPlanToTaskFile(s.fsys, taskFile, plan); err != nil {
		return ActionPlan{}, err
	}
	if err := NewCatalog().AppendFollowUpTasks(s.fsys, filepath.Join(prdFolder, "tasks.md"), plan); err != nil {
		return ActionPlan{}, err
	}
	return plan, nil
}

func (c *Catalog) blockedReviewReason(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "review nao retornou motivo detalhado"
}

// _reviewContextSeparator delimita o cabecalho de contexto e o diff bruto no payload
// entregue ao reviewer consolidado. Usado por buildFinalReviewInput,
// extractReviewContext e attachReviewContext para preservar/recompor o contexto
// entre iteracoes do BugfixLoop sem poluir o prompt do BugfixInvoker.
const _reviewContextSeparator = "\nDiff consolidado:\n"

func (c *Catalog) buildFinalReviewInput(prdFolder, taskFile string, completed []string, diff string) string {
	var b strings.Builder
	b.WriteString("Contexto da revisao consolidada:\n")
	fmt.Fprintf(&b, "- PRD: %s\n", filepath.Join(prdFolder, "prd.md"))
	fmt.Fprintf(&b, "- TechSpec: %s\n", filepath.Join(prdFolder, "techspec.md"))
	fmt.Fprintf(&b, "- Tasks: %s\n", filepath.Join(prdFolder, "tasks.md"))
	if taskFile != "" {
		fmt.Fprintf(&b, "- Ultima task executada: %s\n", taskFile)
	}
	if len(completed) > 0 {
		fmt.Fprintf(&b, "- Tasks executadas neste lote: %s\n", strings.Join(completed, ", "))
	}
	b.WriteString(_reviewContextSeparator)
	b.WriteString(diff)
	return b.String()
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
func (c *Catalog) emitTelemetry(event, value string) {
	if os.Getenv("GOVERNANCE_TELEMETRY") != "1" {
		return
	}
	fmt.Fprintf(os.Stderr, "[taskloop] event=%s value=%s ts=%s\n",
		event, value, time.Now().UTC().Format(time.RFC3339))
}

// selectBatch seleciona até maxBatch tasks elegíveis sem repetição (ADR-018, RF-04).
// Chama Selector.Next iterativamente e para quando ErrNoEligibleTask ou maxBatch atingido.
// Respeita dependências pois o seletor só retorna tasks com deps completas.
// Nota: tasks já selecionadas nesta rodada são incluídas; o seletor pode retornar a mesma
// task múltiplas vezes quando não há outras elegíveis — deduplicamos por ID.
func (c *Catalog) selectBatch(ctx context.Context, sel TaskSelector, absFolder string, maxBatch int) ([]TaskEntry, error) {
	seen := make(map[string]bool, maxBatch)
	batch := make([]TaskEntry, 0, maxBatch)
	for len(batch) < maxBatch {
		next, err := sel.Next(ctx, absFolder)
		if errors.Is(err, ErrNoEligibleTask) {
			return batch, ErrNoEligibleTask
		}
		if err != nil {
			return batch, err
		}
		if seen[next.ID] {
			// Seletor retornou a mesma task novamente (sem outras elegíveis): parar.
			break
		}
		seen[next.ID] = true
		batch = append(batch, *next)
	}
	return batch, nil
}

// batchResult é o resultado de uma task executada concorrentemente.
type batchResult struct {
	taskID   string
	taskFile string
	err      error
}

// executeBatch executa tasks em paralelo limitado por concurrent (ADR-018, RF-04).
// Para cada task: Execute → Gate.Verify → Recorder.Append (sequencial por task).
// Retorna os IDs completados, o último taskFile e o primeiro erro encontrado.
// Sem erro: todos os completed IDs são incluídos no LoopReport.
// Com erro: para imediatamente (halt-on-first-error por task).
func (s *Service) executeBatch(
	ctx context.Context,
	batch []TaskEntry,
	absFolder, workDir string,
	concurrent int,
	deps RunLoopDeps,
) (completedIDs []string, lastTaskFile string, firstErr error) {
	sem := make(chan struct{}, concurrent)
	results := make([]batchResult, len(batch))
	var wg sync.WaitGroup

	for i, task := range batch {
		taskFile, err := NewCatalog().ResolveTaskFile(absFolder, task, s.fsys)
		if err != nil {
			results[i] = batchResult{taskID: task.ID, err: err}
			continue
		}
		results[i].taskFile = taskFile

		wg.Add(1)
		go func(idx int, t TaskEntry, tf string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if ctx.Err() != nil {
				results[idx].err = ctx.Err()
				return
			}

			execErr := deps.Executor.Execute(ctx, t, tf, absFolder, workDir)
			if execErr != nil {
				results[idx].err = fmt.Errorf("taskloop: execucao da task %s: %w", t.ID, execErr)
				return
			}

			accReport, accErr := deps.Gate.Verify(ctx, t, tf)
			if accErr != nil {
				NewCatalog().emitTelemetry("acceptance_failed", t.ID)
				results[idx].err = fmt.Errorf("taskloop: aceite da task %s: %w", t.ID, accErr)
				return
			}

			if recErr := deps.Recorder.Append(ctx, tf, accReport); recErr != nil {
				results[idx].err = fmt.Errorf("taskloop: evidencia da task %s: %w", t.ID, recErr)
				return
			}
			NewCatalog().emitTelemetry("task_completed", t.ID)
			results[idx].taskID = t.ID
		}(i, task, taskFile)
	}

	wg.Wait()

	// Coletar resultados preservando ordem de declaração das tasks.
	for _, r := range results {
		if r.taskFile != "" {
			lastTaskFile = r.taskFile
		}
		if r.err != nil && firstErr == nil {
			firstErr = r.err
		}
		if r.err == nil && r.taskID != "" {
			completedIDs = append(completedIDs, r.taskID)
		}
	}
	return completedIDs, lastTaskFile, firstErr
}
