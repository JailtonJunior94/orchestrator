package taskloop

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// bubbleTeaTheme centraliza as 10 cores da identidade visual do task-loop TUI.
type bubbleTeaTheme struct {
	primary      color.Color // destaque principal
	secondary    color.Color // destaque secundario
	success      color.Color // done/sucesso
	danger       color.Color // falha/erro
	warning      color.Color // timeout/alerta
	info         color.Color // informativo
	muted        color.Color // texto secundario
	background   color.Color // fundo de paineis
	activeBorder color.Color // borda de painel com foco
	normalBorder color.Color // borda de painel sem foco
}

// defaultTheme retorna o tema padrao com todas as 10 cores definidas.
func defaultTheme() bubbleTeaTheme {
	return bubbleTeaTheme{
		primary:      lipgloss.Color("39"),  // azul brilhante
		secondary:    lipgloss.Color("183"), // lavanda
		success:      lipgloss.Color("10"),  // verde
		danger:       lipgloss.Color("9"),   // vermelho
		warning:      lipgloss.Color("11"),  // amarelo
		info:         lipgloss.Color("14"),  // ciano
		muted:        lipgloss.Color("243"), // cinza
		background:   lipgloss.Color("234"), // fundo escuro sutil
		activeBorder: lipgloss.Color("39"),  // borda azul ativa
		normalBorder: lipgloss.Color("243"), // borda cinza inativa
	}
}

// layoutTier classifica a faixa de responsividade do terminal.
type layoutTier int

const (
	layoutWide    layoutTier = iota // >= 120 colunas
	layoutMedium                    // 80-119 colunas
	layoutCompact                   // < 80 colunas
)

// resolveLayoutTier determina a faixa de responsividade para a largura informada.
func resolveLayoutTier(width int) layoutTier {
	switch {
	case width >= 120:
		return layoutWide
	case width >= 80:
		return layoutMedium
	default:
		return layoutCompact
	}
}

// panelFocus identifica qual painel possui foco visual no momento.
type panelFocus int

const (
	focusActiveTask    panelFocus = iota
	focusQueueSummary
	focusRecentEvents
)

// blockHeights armazena as alturas calculadas de cada bloco do layout de seis paineis.
type blockHeights struct {
	dashboard    int
	progressBar  int
	activeTask   int
	queueSummary int
	events       int
	footer       int
}

// computeBlockHeights calcula as alturas de cada bloco conforme altura total e tier de layout.
// Em modo compacto, o painel de fila/resumo e omitido (queueSummary = 0).
// Em modos wide/medium, o painel de eventos ocupa o espaco restante (minimo de 3 linhas).
func computeBlockHeights(totalHeight int, tier layoutTier) blockHeights {
	const (
		dashboardHeight    = 3
		progressBarHeight  = 1
		activeTaskHeight   = 8
		queueSummaryHeight = 5
		footerHeight       = 1
		minEventsHeight    = 3
	)

	if tier == layoutCompact {
		// Compacto: dashboard(1) + progresso(1) + task(4) + rodape(1) = 7 fixo
		const compactFixed = 7
		return blockHeights{
			dashboard:    1,
			progressBar:  1,
			activeTask:   4,
			queueSummary: 0,
			events:       max(1, totalHeight-compactFixed),
			footer:       1,
		}
	}

	fixedTotal := dashboardHeight + progressBarHeight + activeTaskHeight + queueSummaryHeight + footerHeight
	return blockHeights{
		dashboard:    dashboardHeight,
		progressBar:  progressBarHeight,
		activeTask:   activeTaskHeight,
		queueSummary: queueSummaryHeight,
		events:       max(minEventsHeight, totalHeight-fixedTotal),
		footer:       footerHeight,
	}
}

// newBubbleTeaRendererWithTheme constroi os estilos Lip Gloss tematicos do renderer.
// Chamado por newBubbleTeaRenderer para compor os campos de estilo da camada visual.
func newBubbleTeaRendererWithTheme(theme bubbleTeaTheme) (
	dashboardStyle lipgloss.Style,
	dashboardBorderStyle lipgloss.Style,
	progressStyle lipgloss.Style,
	progressEmptyStyle lipgloss.Style,
	normalPanelStyle lipgloss.Style,
	activePanelStyle lipgloss.Style,
	footerStyle lipgloss.Style,
	sectionLabelStyle lipgloss.Style,
) {
	dashboardStyle = lipgloss.NewStyle().
		Foreground(theme.primary).
		Bold(true)

	dashboardBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.primary)

	progressStyle = lipgloss.NewStyle().
		Foreground(theme.info)

	progressEmptyStyle = lipgloss.NewStyle().
		Foreground(theme.muted)

	normalPanelStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.normalBorder)

	activePanelStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.activeBorder)

	footerStyle = lipgloss.NewStyle().Reverse(true)

	sectionLabelStyle = lipgloss.NewStyle().Bold(true)

	return
}
