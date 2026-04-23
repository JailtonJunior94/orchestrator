package taskloop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	defaultBubbleTeaWidth      = 100
	defaultBubbleTeaHeight     = 24
	minBubbleTeaWidth          = 36
	minBubbleTeaHeight         = 8
	compactBubbleTeaWidth      = 64
	bubbleTeaPanelBorderHeight = 2
	bubbleTeaStatusHeight      = 1
)

// BubbleTeaPresenter renderiza o task-loop em modo TUI sem acoplar execucao ao dominio.
type BubbleTeaPresenter struct {
	capabilities     TerminalCapabilities
	snapshotProvider func() SessionSnapshot
	programFactory   bubbleTeaProgramFactory
	renderer         bubbleTeaRenderer

	mu      sync.Mutex
	started bool
	program bubbleTeaProgram
	done    chan error
	cancel  context.CancelFunc
}

type bubbleTeaProgram interface {
	Send(tea.Msg)
	Run() (tea.Model, error)
}

type bubbleTeaProgramFactory func(model tea.Model, options ...tea.ProgramOption) bubbleTeaProgram

type bubbleTeaSnapshotMsg struct {
	snapshot SessionSnapshot
}

type bubbleTeaSummaryMsg struct {
	summary FinalSummary
}

type bubbleTeaModel struct {
	renderer  bubbleTeaRenderer
	snapshot  SessionSnapshot
	summary   *FinalSummary
	width     int
	height    int
	altScreen bool
}

type bubbleTeaRenderer struct {
	panelStyle        lipgloss.Style
	statusStyle       lipgloss.Style
	sectionLabelStyle lipgloss.Style
	criticalStyle     lipgloss.Style
	warnStyle         lipgloss.Style
	infoStyle         lipgloss.Style
}

// NewBubbleTeaPresenter cria o presenter TUI canônico do modo iterativo.
func NewBubbleTeaPresenter(capabilities TerminalCapabilities, snapshotProvider func() SessionSnapshot) *BubbleTeaPresenter {
	return &BubbleTeaPresenter{
		capabilities:     capabilities,
		snapshotProvider: snapshotProvider,
		programFactory:   defaultBubbleTeaProgramFactory,
		renderer:         newBubbleTeaRenderer(),
	}
}

func defaultBubbleTeaProgramFactory(model tea.Model, options ...tea.ProgramOption) bubbleTeaProgram {
	return tea.NewProgram(model, options...)
}

func newBubbleTeaRenderer() bubbleTeaRenderer {
	return bubbleTeaRenderer{
		panelStyle: lipgloss.NewStyle(),
		statusStyle: lipgloss.NewStyle().
			Reverse(true),
		sectionLabelStyle: lipgloss.NewStyle().Bold(true),
		criticalStyle:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")),
		warnStyle:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")),
		infoStyle:         lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")),
	}
}

func (p *BubbleTeaPresenter) Start(snapshot SessionSnapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return nil
	}

	width, height := normalizeBubbleTeaDimensions(p.capabilities.Width, p.capabilities.Height)
	ctx, cancel := context.WithCancel(context.Background())
	options := []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithWindowSize(width, height),
	}

	model := bubbleTeaModel{
		renderer:  p.renderer,
		snapshot:  snapshot,
		width:     width,
		height:    height,
		altScreen: p.capabilities.SupportsAltScreen,
	}

	p.program = p.programFactory(model, options...)
	p.done = make(chan error, 1)
	p.cancel = cancel
	p.started = true

	go func(program bubbleTeaProgram, done chan<- error) {
		_, err := program.Run()
		done <- err
	}(p.program, p.done)

	return nil
}

func (p *BubbleTeaPresenter) Consume(event LoopEvent) error {
	p.mu.Lock()
	program := p.program
	done := p.done
	snapshot := p.snapshot()
	p.mu.Unlock()

	if err := consumeBubbleTeaRunResult(done, false); err != nil {
		return err
	}
	if program == nil {
		return nil
	}

	program.Send(bubbleTeaSnapshotMsg{snapshot: snapshot})
	return nil
}

func (p *BubbleTeaPresenter) Finish(summary FinalSummary) error {
	p.mu.Lock()
	program := p.program
	done := p.done
	cancel := p.cancel
	p.program = nil
	p.done = nil
	p.cancel = nil
	p.started = false
	p.mu.Unlock()

	if program == nil || done == nil {
		if cancel != nil {
			cancel()
		}
		return nil
	}

	program.Send(bubbleTeaSummaryMsg{summary: summary})
	err := consumeBubbleTeaRunResult(done, true)
	if cancel != nil {
		cancel()
	}
	return err
}

func (p *BubbleTeaPresenter) snapshot() SessionSnapshot {
	if p.snapshotProvider == nil {
		return SessionSnapshot{}
	}
	return p.snapshotProvider()
}

func consumeBubbleTeaRunResult(done <-chan error, wait bool) error {
	if done == nil {
		return nil
	}
	if wait {
		return <-done
	}
	select {
	case err := <-done:
		return err
	default:
		return nil
	}
}

func (m bubbleTeaModel) Init() tea.Cmd {
	return nil
}

func (m bubbleTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = normalizeBubbleTeaDimensions(typed.Width, typed.Height)
	case bubbleTeaSnapshotMsg:
		m.snapshot = typed.snapshot
	case bubbleTeaSummaryMsg:
		summary := typed.summary
		m.summary = &summary
		return m, tea.Quit
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m bubbleTeaModel) View() tea.View {
	view := tea.NewView(m.renderer.Render(m.width, m.height, m.snapshot, m.summary))
	view.AltScreen = m.altScreen
	return view
}

func (r bubbleTeaRenderer) Render(width, height int, snapshot SessionSnapshot, summary *FinalSummary) string {
	width, height = normalizeBubbleTeaDimensions(width, height)

	status := r.renderStatusLine(width, snapshot, summary)
	if height <= minBubbleTeaHeight {
		return lipgloss.JoinVertical(lipgloss.Left,
			r.renderPanel(width, 2, r.mainLines(snapshot, width, summary)),
			status,
		)
	}

	mainInner, eventsInner := splitBubbleTeaHeights(height, width)
	main := r.renderPanel(width, mainInner, r.mainLines(snapshot, width, summary))
	events := r.renderPanel(width, eventsInner, r.eventLines(snapshot, width, eventsInner))

	return lipgloss.JoinVertical(lipgloss.Left, main, events, status)
}

func (r bubbleTeaRenderer) renderPanel(width, innerHeight int, lines []string) string {
	contentWidth := maxInt(1, width-4)
	border := lipgloss.NormalBorder()
	normalized := normalizePanelLines(lines, innerHeight, contentWidth)
	rendered := make([]string, 0, len(normalized)+2)
	rendered = append(rendered, border.TopLeft+strings.Repeat(border.Top, contentWidth)+border.TopRight)
	for _, line := range normalized {
		rendered = append(rendered, border.Left+padBubbleTeaLine(line, contentWidth)+border.Right)
	}
	rendered = append(rendered, border.BottomLeft+strings.Repeat(border.Bottom, contentWidth)+border.BottomRight)
	return r.panelStyle.Render(strings.Join(rendered, "\n"))
}

func (r bubbleTeaRenderer) mainLines(snapshot SessionSnapshot, width int, summary *FinalSummary) []string {
	lines := []string{r.sectionLabelStyle.Render("Execucao")}
	if summary != nil {
		lines = append(lines,
			fmt.Sprintf("Stop: %s", firstNonEmpty(summary.StopReason, "execucao encerrada")),
			fmt.Sprintf("Iteracoes: %d", summary.IterationsRun),
			fmt.Sprintf("Report: %s", firstNonEmpty(summary.ReportPath, "n/a")),
			fmt.Sprintf("Lote: %s", formatBatchProgress(summary.Progress)),
		)
		if summary.LastFailure != nil {
			lines = append(lines, "Falha final: "+renderFailureMessage(summary.LastFailure))
		}
		return lines
	}

	active := snapshot.ActiveIteration
	if active == nil {
		lines = append(lines, "Task: aguardando selecao", "Fase: "+phaseLabel(snapshot.CurrentPhase))
		lines = append(lines, "Lote: "+formatBatchProgress(snapshot.Progress))
		return lines
	}

	lines = append(lines,
		fmt.Sprintf("Task: %s %s", active.TaskID, firstNonEmpty(active.Title, "sem titulo")),
	)

	if width < compactBubbleTeaWidth {
		lines = append(lines,
			fmt.Sprintf("Iteracao/Execucao: %s | %s / %s", renderIterationCounter(active.Sequence, snapshot.MaxIterations), firstNonEmpty(string(active.Role), "n/a"), firstNonEmpty(string(active.Tool), "n/a")),
			"Fase: "+phaseStatusLabel(active.Phase),
		)
		if snapshot.LastError != nil {
			lines = append(lines, "Falha: "+renderFailureMessage(snapshot.LastError))
		} else {
			lines = append(lines, "Lote: "+formatBatchProgress(snapshot.Progress))
		}
	} else {
		lines = append(lines,
			fmt.Sprintf("Iteracao: %s", renderIterationCounter(active.Sequence, snapshot.MaxIterations)),
			fmt.Sprintf("Papel: %s | Tool: %s", firstNonEmpty(string(active.Role), "n/a"), firstNonEmpty(string(active.Tool), "n/a")),
			"Fase: "+phaseStatusLabel(active.Phase),
			"Lote: "+formatBatchProgress(snapshot.Progress),
		)
		if snapshot.LastError != nil {
			lines = append(lines, "Falha: "+renderFailureMessage(snapshot.LastError))
		}
	}
	return lines
}

func (r bubbleTeaRenderer) eventLines(snapshot SessionSnapshot, width, innerHeight int) []string {
	lines := []string{r.sectionLabelStyle.Render("Eventos recentes")}
	if len(snapshot.RecentEvents) == 0 {
		return append(lines, "INFO nenhum evento observado")
	}

	available := maxInt(1, innerHeight-1)
	start := len(snapshot.RecentEvents) - available
	if start < 0 {
		start = 0
	}

	for i := len(snapshot.RecentEvents) - 1; i >= start; i-- {
		lines = append(lines, formatRecentEvent(snapshot.RecentEvents[i], width))
	}
	return lines
}

func (r bubbleTeaRenderer) renderStatusLine(width int, snapshot SessionSnapshot, summary *FinalSummary) string {
	segments := []string{
		"UI: tui",
		"Modo: " + firstNonEmpty(snapshot.Mode, "n/a"),
		"Tempo: " + snapshot.Elapsed.Truncate(time.Second).String(),
		fmt.Sprintf("Lote: %d/%d done", snapshot.Progress.Done, snapshot.Progress.Total),
	}

	if summary != nil {
		segments = append(segments, "Final: "+firstNonEmpty(summary.StopReason, "encerrado"))
	} else {
		segments = append(segments, "Fase: "+phaseStatusLabel(snapshot.CurrentPhase))
	}

	line := truncateBubbleTeaLine(strings.Join(segments, " | "), width)
	return r.statusStyle.Width(maxInt(1, width)).Render(line)
}

func phaseStatusLabel(phase AgentPhase) string {
	base := strings.ToUpper(phaseLabel(phase))
	switch phase {
	case PhaseFailed, PhaseTimeout, PhaseAuthRequired:
		return "CRITICO " + base
	case PhasePreparing, PhaseRunning, PhaseStreaming, PhaseReviewing:
		return "ATIVO " + base
	default:
		return base
	}
}

func formatRecentEvent(event RecentEvent, width int) string {
	label := eventSeverityLabel(event)
	phase := phaseStatusLabel(event.Phase)
	parts := []string{
		event.Timestamp.Format("15:04:05"),
		label,
	}
	if event.Task.ID != "" {
		parts = append(parts, "task="+event.Task.ID)
	}
	if event.Tool != "" {
		parts = append(parts, "tool="+string(event.Tool))
	}
	if event.Phase != "" {
		parts = append(parts, "fase="+phase)
	}
	if msg := strings.TrimSpace(event.Message); msg != "" {
		parts = append(parts, msg)
	}
	return truncateBubbleTeaLine(strings.Join(parts, " "), maxInt(1, width-4))
}

func eventSeverityLabel(event RecentEvent) string {
	switch {
	case event.Phase == PhaseFailed || event.Phase == PhaseTimeout || event.Phase == PhaseAuthRequired:
		return "ERRO"
	case event.Severity == SeverityError:
		return "ERRO"
	case event.Severity == SeverityWarn:
		return "ALERTA"
	default:
		return "INFO"
	}
}

func formatBatchProgress(progress BatchProgress) string {
	if progress.Total == 0 {
		return "sem tarefas conhecidas"
	}
	return fmt.Sprintf(
		"done=%d failed=%d blocked=%d needs_input=%d pending=%d in_progress=%d total=%d",
		progress.Done,
		progress.Failed,
		progress.Blocked,
		progress.NeedsInput,
		progress.Pending,
		progress.InProgress,
		progress.Total,
	)
}

func splitBubbleTeaHeights(totalHeight, width int) (int, int) {
	availableInner := totalHeight - bubbleTeaStatusHeight - (2 * bubbleTeaPanelBorderHeight)
	if availableInner < 2 {
		return 1, 1
	}

	mainInner := 6
	if width < compactBubbleTeaWidth {
		mainInner = 5
	}
	if availableInner <= mainInner {
		mainInner = maxInt(1, availableInner/2)
	}
	eventsInner := maxInt(1, availableInner-mainInner)
	return mainInner, eventsInner
}

func normalizeBubbleTeaDimensions(width, height int) (int, int) {
	if width <= 0 {
		width = defaultBubbleTeaWidth
	}
	if height <= 0 {
		height = defaultBubbleTeaHeight
	}
	if width < minBubbleTeaWidth {
		width = minBubbleTeaWidth
	}
	if height < minBubbleTeaHeight {
		height = minBubbleTeaHeight
	}
	return width, height
}

func normalizePanelLines(lines []string, innerHeight, contentWidth int) []string {
	normalized := make([]string, 0, maxInt(innerHeight, len(lines)))
	for _, line := range lines {
		normalized = append(normalized, truncateBubbleTeaLine(line, contentWidth))
	}
	for len(normalized) < innerHeight {
		normalized = append(normalized, "")
	}
	if len(normalized) > innerHeight {
		return normalized[:innerHeight]
	}
	return normalized
}

func truncateBubbleTeaLine(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func padBubbleTeaLine(value string, limit int) string {
	if len(value) >= limit {
		return value
	}
	return value + strings.Repeat(" ", limit-len(value))
}

func maxInt(values ...int) int {
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}
