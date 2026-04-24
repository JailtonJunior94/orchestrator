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
	defaultBubbleTeaWidth  = 100
	defaultBubbleTeaHeight = 24
	minBubbleTeaWidth      = 36
	minBubbleTeaHeight     = 8
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

	// Campos de controle operacional adicionados na task 13.0
	focus  panelFocus
	paused bool
}

type bubbleTeaRenderer struct {
	// Estilos legados do layout de 2 paineis (preservados para compatibilidade)
	panelStyle        lipgloss.Style
	statusStyle       lipgloss.Style
	sectionLabelStyle lipgloss.Style
	criticalStyle     lipgloss.Style
	warnStyle         lipgloss.Style
	infoStyle         lipgloss.Style

	// Estilos do layout de 6 blocos (Fase 2)
	dashboardStyle       lipgloss.Style
	dashboardBorderStyle lipgloss.Style
	progressStyle        lipgloss.Style
	progressEmptyStyle   lipgloss.Style
	normalPanelStyle     lipgloss.Style
	activePanelStyle     lipgloss.Style
	footerStyle          lipgloss.Style
	// criticalStyle, warnStyle e infoStyle sao consumidos pelo render de task ativa (task 10.0)
	// e por formatacao de eventos criticos (tasks 12.0/14.0) — mantidos no renderer legado por ora
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
	theme := defaultTheme()
	dash, dashBorder, prog, progEmpty, normal, active, footer, sectionLabel := newBubbleTeaRendererWithTheme(theme)
	return bubbleTeaRenderer{
		panelStyle:        lipgloss.NewStyle(),
		statusStyle:       lipgloss.NewStyle().Reverse(true),
		sectionLabelStyle: sectionLabel,
		criticalStyle:     lipgloss.NewStyle().Bold(true).Foreground(theme.danger),
		warnStyle:         lipgloss.NewStyle().Bold(true).Foreground(theme.warning),
		infoStyle:         lipgloss.NewStyle().Bold(true).Foreground(theme.success),

		dashboardStyle:       dash,
		dashboardBorderStyle: dashBorder,
		progressStyle:        prog,
		progressEmptyStyle:   progEmpty,
		normalPanelStyle:     normal,
		activePanelStyle:     active,
		footerStyle:          footer,
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
		switch resolveKeyAction(typed.String()) {
		case keyQuit:
			return m, tea.Quit
		case keyPause:
			m.paused = !m.paused
		case keySkip:
			// visual apenas nesta fase — sem sinal ao Service (ADR-005)
		case keyTabFocus:
			m.focus = (m.focus + 1) % 3
		}
	}
	return m, nil
}

func (m bubbleTeaModel) View() tea.View {
	view := tea.NewView(m.renderer.Render(m.width, m.height, m.snapshot, m.summary, m.focus, m.paused))
	view.AltScreen = m.altScreen
	return view
}

// Render compoe o layout de 6 blocos verticais usando lipgloss.JoinVertical.
// Quando summary != nil, renderiza a tela de encerramento em vez do layout de 6 blocos.
// O primeiro frame com snapshot vazio renderiza todos os paineis com estados iniciais.
func (r bubbleTeaRenderer) Render(width, height int, snapshot SessionSnapshot, summary *FinalSummary, focus panelFocus, paused bool) string {
	width, height = normalizeBubbleTeaDimensions(width, height)

	if summary != nil {
		return r.renderFinalSummary(width, height, *summary)
	}

	tier := resolveLayoutTier(width)
	heights := computeBlockHeights(height, tier)

	blocks := make([]string, 0, 6)
	blocks = append(blocks, r.renderDashboard(width, snapshot, tier))
	blocks = append(blocks, r.renderProgressBar(width, snapshot.Progress))
	blocks = append(blocks, r.renderActiveTask(width, snapshot, tier))
	if heights.queueSummary > 0 {
		blocks = append(blocks, r.renderQueueSummary(width, snapshot.Progress, tier))
	}
	blocks = append(blocks, r.renderRecentEvents(width, heights.events, snapshot.RecentEvents,
		focus == focusRecentEvents, tier))
	blocks = append(blocks, r.renderFooter(width, paused, focus, snapshot.Mode))

	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

// renderFinalSummary renderiza a tela de encerramento quando summary != nil.
// Exibe: motivo de encerramento, iteracoes executadas, caminho do relatorio,
// progresso do lote e ultima falha (se existir).
func (r bubbleTeaRenderer) renderFinalSummary(width, height int, summary FinalSummary) string {
	lines := []string{
		r.sectionLabelStyle.Render("Execucao Finalizada"),
		"",
		fmt.Sprintf("Motivo: %s", firstNonEmpty(summary.StopReason, "execucao encerrada")),
		fmt.Sprintf("Iteracoes: %d", summary.IterationsRun),
		fmt.Sprintf("Relatorio: %s", firstNonEmpty(summary.ReportPath, "n/a")),
		fmt.Sprintf("Lote: %s", formatBatchProgress(summary.Progress)),
	}
	if summary.LastFailure != nil {
		lines = append(lines, fmt.Sprintf("Falha final: %s", renderFailureMessage(summary.LastFailure)))
	}
	innerHeight := max(len(lines), height-2)
	return r.renderStyledPanel(width, innerHeight, lines, r.normalPanelStyle)
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

func maxInt(values ...int) int {
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}

// renderActiveTask renderiza o painel principal com informacoes completas da task em execucao.
// Em wide/medium: 6 linhas (label, ID+titulo, iteracao, ferramenta+papel, fase, tempo) com borda.
// Em compact: 2 linhas densas com informacao essencial.
// Quando ActiveIteration == nil, exibe mensagem de espera com altura preservada.
func (r bubbleTeaRenderer) renderActiveTask(width int, snapshot SessionSnapshot, tier layoutTier) string {
	active := snapshot.ActiveIteration
	contentWidth := max(1, width-4)

	if active == nil {
		innerH := 6
		if tier == layoutCompact {
			innerH = 2
		}
		return r.renderStyledPanel(width, innerH, []string{
			r.sectionLabelStyle.Render("Task Ativa"),
			"Aguardando selecao de task...",
		}, r.normalPanelStyle)
	}

	phaseDisplay := phaseStatusLabel(active.Phase)
	elapsed := time.Since(active.StartedAt).Truncate(time.Second).String()

	if tier == layoutCompact {
		lines := []string{
			truncateBubbleTeaLine(fmt.Sprintf("%s %s", active.TaskID, active.Title), contentWidth),
			truncateBubbleTeaLine(fmt.Sprintf("%s | %s | %s | %s",
				renderIterationCounter(active.Sequence, snapshot.MaxIterations),
				string(active.Tool), string(active.Role), phaseDisplay), contentWidth),
		}
		return r.renderStyledPanel(width, 2, lines, r.normalPanelStyle)
	}

	lines := []string{
		r.sectionLabelStyle.Render("Task Ativa"),
		truncateBubbleTeaLine(fmt.Sprintf("ID: %s — %s", active.TaskID,
			firstNonEmpty(active.Title, "sem titulo")), contentWidth),
		truncateBubbleTeaLine(fmt.Sprintf("Iteracao: %s",
			renderIterationCounter(active.Sequence, snapshot.MaxIterations)), contentWidth),
		truncateBubbleTeaLine(fmt.Sprintf("Ferramenta: %s   Papel: %s",
			firstNonEmpty(string(active.Tool), "n/a"),
			firstNonEmpty(string(active.Role), "n/a")), contentWidth),
		truncateBubbleTeaLine(fmt.Sprintf("Fase: %s", phaseDisplay), contentWidth),
		truncateBubbleTeaLine(fmt.Sprintf("Tempo: %s", elapsed), contentWidth),
	}
	return r.renderStyledPanel(width, 6, lines, r.normalPanelStyle)
}

// renderStyledPanel renderiza um painel com borda e altura fixa usando o estilo Lip Gloss fornecido.
// As linhas sao normalizadas para innerHeight e truncadas a contentWidth = width - 4.
// O estilo deve ter borda configurada; a largura total do painel sera width.
func (r bubbleTeaRenderer) renderStyledPanel(width, innerHeight int, lines []string, style lipgloss.Style) string {
	contentWidth := max(1, width-4)
	normalized := normalizePanelLines(lines, innerHeight, contentWidth)
	content := strings.Join(normalized, "\n")
	return style.Width(max(1, width-2)).Render(content)
}

// renderDashboard renderiza o bloco de cabecalho com identidade visual e contexto operacional.
// Em wide/medium: painel com borda exibindo titulo, modo, ferramenta ativa e limite de iteracoes.
// Em compact: linha unica truncada com separadores "|".
func (r bubbleTeaRenderer) renderDashboard(width int, snapshot SessionSnapshot, tier layoutTier) string {
	contentWidth := max(1, width-4)

	title := "ai-spec task-loop"
	mode := "Modo: " + firstNonEmpty(snapshot.Mode, "simples")
	tool := "Tool: " + firstNonEmpty(string(snapshot.CurrentTool), "n/a")
	maxIter := fmt.Sprintf("Max: %d", snapshot.MaxIterations)

	if tier == layoutCompact {
		line := truncateBubbleTeaLine(
			fmt.Sprintf("%s | %s | %s | %s", title, mode, tool, maxIter), contentWidth)
		return r.dashboardStyle.Width(max(1, width)).Render(line)
	}

	line := fmt.Sprintf("%s   %s   %s   %s", title, mode, tool, maxIter)
	return r.renderStyledPanel(width, 1, []string{
		truncateBubbleTeaLine(line, contentWidth),
	}, r.dashboardBorderStyle)
}

// renderQueueSummary renderiza o painel de fila com 6 contadores do lote por estado.
// Em compact: retorna string vazia (dados visiveis na barra de progresso).
// Em wide/medium: painel com borda, label "Fila" e 2 linhas de contadores.
func (r bubbleTeaRenderer) renderQueueSummary(width int, progress BatchProgress, tier layoutTier) string {
	if tier == layoutCompact {
		return "" // omitido em compacto; dados visiveis na barra de progresso
	}

	contentWidth := max(1, width-4)
	lines := []string{
		r.sectionLabelStyle.Render("Fila"),
		truncateBubbleTeaLine(fmt.Sprintf("Total: %d   Pendentes: %d   Em execucao: %d",
			progress.Total, progress.Pending, progress.InProgress), contentWidth),
		truncateBubbleTeaLine(fmt.Sprintf("Concluidas: %d   Falhadas: %d   Bloqueadas: %d",
			progress.Done, progress.Failed, progress.Blocked), contentWidth),
	}
	return r.renderStyledPanel(width, 3, lines, r.normalPanelStyle)
}

// renderFooter renderiza o rodape com atalhos de teclado, modo efetivo e status da UI.
// O label de pause alterna entre "p pausar" e "p retomar" conforme o estado paused.
// Trunca explicitamente quando a largura for insuficiente.
func (r bubbleTeaRenderer) renderFooter(width int, paused bool, focus panelFocus, mode string) string {
	pauseLabel := "p pausar"
	if paused {
		pauseLabel = "p retomar"
	}

	_ = focus // foco sera usado na composicao final (task 14.0)
	shortcuts := fmt.Sprintf(" q sair | %s (visual) | s skip (visual) | tab foco | UI: tui | Modo: %s",
		pauseLabel, firstNonEmpty(mode, "n/a"))

	return r.footerStyle.Width(max(1, width)).Render(
		truncateBubbleTeaLine(shortcuts, max(1, width-2)))
}

// renderRecentEvents renderiza o painel de eventos recentes com altura dinamica.
// A borda usa activePanelStyle quando focus=true e normalPanelStyle caso contrario.
// Exibe os N eventos mais recentes em ordem reversa (mais recente primeiro).
// Capacidade visual: height - 2 (bordas) - 1 (label) linhas de eventos.
// Estado vazio: "Nenhum evento registrado".
func (r bubbleTeaRenderer) renderRecentEvents(width, height int, events []RecentEvent, focus bool, tier layoutTier) string {
	innerHeight := max(1, height-2)
	capacity := max(0, innerHeight-1) // reserva 1 linha para o label

	style := r.normalPanelStyle
	if focus {
		style = r.activePanelStyle
	}

	lines := []string{r.sectionLabelStyle.Render("Eventos recentes")}

	if len(events) == 0 {
		lines = append(lines, "Nenhum evento registrado")
		return r.renderStyledPanel(width, innerHeight, lines, style)
	}

	// Janela deslizante: exibe apenas os N ultimos eventos
	start := 0
	if capacity > 0 && len(events) > capacity {
		start = len(events) - capacity
	}
	window := events[start:]

	// Ordem reversa: mais recente primeiro
	for i := len(window) - 1; i >= 0; i-- {
		lines = append(lines, formatRecentEvent(window[i], width))
	}

	return r.renderStyledPanel(width, innerHeight, lines, style)
}

// renderProgressBar renderiza a barra de progresso horizontal com caracteres Unicode.
// Exibe percentual e contadores "N/M tasks". Quando Total == 0, exibe mensagem vazia.
func (r bubbleTeaRenderer) renderProgressBar(width int, progress BatchProgress) string {
	if progress.Total == 0 {
		return r.progressEmptyStyle.Width(max(1, width)).Render("sem tarefas conhecidas")
	}

	pct := float64(progress.Done) / float64(progress.Total)
	label := fmt.Sprintf(" %d/%d tasks (%d%%)", progress.Done, progress.Total, int(pct*100))

	barWidth := max(1, width-len(label)-2)
	filled := min(barWidth, int(float64(barWidth)*pct))
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return r.progressStyle.Width(max(1, width)).Render(bar + label)
}
