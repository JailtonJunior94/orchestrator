package taskloop

import (
	"fmt"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// TextPresenter traduz snapshots e eventos do loop para saida textual proporcional.
type TextPresenter struct {
	printer          *output.Printer
	snapshotProvider func() SessionSnapshot
	started          bool
	lastSnapshot     SessionSnapshot
	lastProgressLine string
}

// NewTextPresenter cria o presenter textual canonico do modo plain.
func NewTextPresenter(printer *output.Printer, snapshotProvider func() SessionSnapshot) *TextPresenter {
	return &TextPresenter{
		printer:          printer,
		snapshotProvider: snapshotProvider,
	}
}

func (p *TextPresenter) Start(snapshot SessionSnapshot) error {
	p.started = true
	p.lastSnapshot = snapshot
	return nil
}

func (p *TextPresenter) Consume(event LoopEvent) error {
	snapshot := p.snapshot()
	if !p.started {
		if err := p.Start(snapshot); err != nil {
			return err
		}
	}
	p.lastSnapshot = snapshot

	switch event.Kind {
	case EventIterationSelected:
		p.printer.Step("%s", p.renderIterationHeader(snapshot))
	case EventProgressUpdated:
		line := renderProgress(snapshot.Progress)
		if line == "" || line == p.lastProgressLine {
			return nil
		}
		p.lastProgressLine = line
		p.printer.Info("%s", line)
	case EventPhaseChanged, EventOutputObserved, EventHeartbeatObserved:
		p.printer.Info("%s", p.renderIterationState(snapshot, event))
	case EventFailureObserved:
		p.printer.Error("%s", p.renderFailure(snapshot, event))
	case EventSessionStarted, EventSessionFinished:
		// Inicio e fim sao comunicados pelo resumo final agregado.
	}

	return nil
}

func (p *TextPresenter) Finish(summary FinalSummary) error {
	p.printer.Info(
		"resumo final: stop=%s iteracoes=%d lote=%s report=%s",
		firstNonEmpty(summary.StopReason, "execucao encerrada"),
		summary.IterationsRun,
		formatBatchProgress(summary.Progress),
		firstNonEmpty(summary.ReportPath, "n/a"),
	)
	if summary.LastFailure != nil {
		p.printer.Error("falha final: %s", renderFailureMessage(summary.LastFailure))
	}
	return nil
}

func (p *TextPresenter) snapshot() SessionSnapshot {
	if p.snapshotProvider == nil {
		return p.lastSnapshot
	}
	return p.snapshotProvider()
}

func (p *TextPresenter) renderIterationHeader(snapshot SessionSnapshot) string {
	active := snapshot.ActiveIteration
	if active == nil {
		return "iteracao selecionada"
	}
	return fmt.Sprintf(
		"%s task %s (%s) tool=%s role=%s phase=%s",
		renderIterationCounter(active.Sequence, snapshot.MaxIterations),
		active.TaskID,
		firstNonEmpty(active.Title, "sem titulo"),
		firstNonEmpty(string(active.Tool), string(snapshot.CurrentTool)),
		firstNonEmpty(string(active.Role), string(snapshot.CurrentRole)),
		phaseLabel(active.Phase),
	)
}

func (p *TextPresenter) renderIterationState(snapshot SessionSnapshot, event LoopEvent) string {
	active := snapshot.ActiveIteration
	if active == nil {
		return strings.TrimSpace(event.Message)
	}

	message := strings.TrimSpace(event.Message)
	if message == "" && len(snapshot.RecentEvents) > 0 {
		message = snapshot.RecentEvents[len(snapshot.RecentEvents)-1].Message
	}
	message = truncateForLine(message, 120)

	return fmt.Sprintf(
		"%s task %s tool=%s role=%s phase=%s elapsed=%s evento=%s",
		renderIterationCounter(active.Sequence, snapshot.MaxIterations),
		active.TaskID,
		firstNonEmpty(string(active.Tool), string(snapshot.CurrentTool)),
		firstNonEmpty(string(active.Role), string(snapshot.CurrentRole)),
		phaseLabel(snapshot.CurrentPhase),
		snapshot.Elapsed.Truncate(time.Second),
		firstNonEmpty(message, "sem detalhe adicional"),
	)
}

func (p *TextPresenter) renderFailure(snapshot SessionSnapshot, event LoopEvent) string {
	active := snapshot.ActiveIteration
	taskID := event.Task.ID
	if taskID == "" && active != nil {
		taskID = active.TaskID
	}
	tool := string(event.Tool)
	if tool == "" && active != nil {
		tool = string(active.Tool)
	}
	failure := event.Failure
	if failure == nil && snapshot.LastError != nil {
		failure = snapshot.LastError
	}

	return fmt.Sprintf(
		"%s task %s tool=%s phase=%s falha=%s",
		renderIterationCounter(event.Iteration, snapshot.MaxIterations),
		firstNonEmpty(taskID, "n/a"),
		firstNonEmpty(tool, "n/a"),
		phaseLabel(snapshot.CurrentPhase),
		renderFailureMessage(failure),
	)
}

func renderProgress(progress BatchProgress) string {
	if progress.Total == 0 {
		return ""
	}
	return fmt.Sprintf(
		"progresso do lote: total=%d done=%d failed=%d blocked=%d needs_input=%d pending=%d in_progress=%d",
		progress.Total,
		progress.Done,
		progress.Failed,
		progress.Blocked,
		progress.NeedsInput,
		progress.Pending,
		progress.InProgress,
	)
}

func renderFailureMessage(failure *LoopFailure) string {
	if failure == nil {
		return "falha nao detalhada"
	}
	base := firstNonEmpty(strings.TrimSpace(failure.Message), failure.Code.defaultMessage())
	hint := failureHint(failure.Code)
	if hint == "" {
		return base
	}
	return fmt.Sprintf("%s; hint: %s", base, hint)
}

func failureHint(code ErrorCode) string {
	switch code {
	case ErrorToolBinaryMissing:
		return "verifique se o binario da ferramenta esta instalado e disponivel no PATH"
	case ErrorToolAuthRequired:
		return "verifique login, sessao local ou credenciais da ferramenta ativa"
	case ErrorToolTimeout:
		return "revise possivel travamento da CLI ou aumente o timeout configurado"
	case ErrorToolExecutionFailed:
		return "inspecione stderr, diff local e report da iteracao"
	case ErrorTaskIsolationViolation:
		return "revise alteracoes fora da task alvo antes de repetir a execucao"
	case ErrorInteractiveUnavailable:
		return "use o modo plain quando a TUI nao estiver disponivel"
	default:
		return ""
	}
}

func renderIterationCounter(current, max int) string {
	switch {
	case current <= 0:
		return "iteracao"
	case max > 0:
		return fmt.Sprintf("iteracao %d/%d", current, max)
	default:
		return fmt.Sprintf("iteracao %d", current)
	}
}

func phaseLabel(phase AgentPhase) string {
	switch phase {
	case PhasePreparing:
		return "preparando"
	case PhaseRunning:
		return "executando"
	case PhaseStreaming:
		return "recebendo saida"
	case PhaseReviewing:
		return "revisando"
	case PhaseDone:
		return "concluido"
	case PhaseFailed:
		return "falhou"
	case PhaseTimeout:
		return "timeout"
	case PhaseAuthRequired:
		return "autenticacao pendente"
	case PhaseIdle:
		return "ocioso"
	default:
		if phase == "" {
			return "desconhecida"
		}
		return string(phase)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func truncateForLine(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}
