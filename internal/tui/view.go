package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// View renders the complete TUI layout, dispatching to either horizontal
// dual-pane or vertical stack depending on terminal width.
func (m model) View() tea.View {
	var s string
	switch {
	case m.mode == modePalette:
		s = m.viewWithPalette()
	case m.width < 100:
		s = m.viewVerticalStack()
	default:
		s = m.viewHorizontal()
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

// viewWithPalette renders the command palette overlay (centered on screen).
func (m model) viewWithPalette() string {
	m.palette.SetSize(m.width, m.height)
	return m.palette.View()
}

// viewHorizontal renders the dual-pane horizontal layout (1:2 weight split).
func (m model) viewHorizontal() string {
	leftW, rightW, contentH := layoutDimensions(m.width, m.height, m.mode == modeWaitingApproval)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	// Left panel: step list
	m.stepList.SetWidth(leftW)
	leftInner := leftW - 4
	if leftInner < 1 {
		leftInner = 1
	}
	leftContent := m.stepList.View()
	leftPanel := panelStyle.
		Width(leftInner).
		Height(contentH).
		Render(leftContent)

	// Right panel: output view
	rightInner := rightW - 4
	if rightInner < 1 {
		rightInner = 1
	}
	m.outputView.SetSize(rightInner, contentH)
	rightPanel := panelStyle.
		Width(rightInner).
		Height(contentH).
		Render(m.outputView.View())

	top := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	return m.assembleFrame(top)
}

// viewVerticalStack renders panels stacked vertically for narrow terminals.
func (m model) viewVerticalStack() string {
	_, _, contentH := layoutDimensions(m.width, m.height, m.mode == modeWaitingApproval)

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	halfH := contentH / 2
	if halfH < 1 {
		halfH = 1
	}

	innerW := m.width - 4
	if innerW < 1 {
		innerW = 1
	}

	m.stepList.SetWidth(m.width)
	topPanel := panelStyle.
		Width(innerW).
		Height(halfH).
		Render(m.stepList.View())

	m.outputView.SetSize(innerW, contentH-halfH)
	bottomPanel := panelStyle.
		Width(innerW).
		Height(contentH - halfH).
		Render(m.outputView.View())

	top := lipgloss.JoinVertical(lipgloss.Left, topPanel, bottomPanel)

	return m.assembleFrame(top)
}

// assembleFrame appends the HITL bar (when visible) and the status bar.
// When a success or flash animation is active, the status bar text is tinted.
func (m model) assembleFrame(mainContent string) string {
	var sb strings.Builder
	sb.WriteString(mainContent)
	sb.WriteByte('\n')

	if m.mode == modeWaitingApproval && m.hitlBar.Visible() {
		sb.WriteString(m.hitlBar.View())
		sb.WriteByte('\n')
	}

	statusBar := m.statusBar
	if !m.runStartedAt.IsZero() {
		statusBar.Duration = time.Since(m.runStartedAt)
	}
	statusContent := statusBar.View(m.width)
	switch {
	case m.successTicks > 0:
		statusContent = lipgloss.NewStyle().Foreground(m.theme.Success).Render(statusContent)
	case m.flashTicks > 0:
		statusContent = lipgloss.NewStyle().Foreground(m.theme.Error).Render(statusContent)
	}
	sb.WriteString(statusContent)
	return sb.String()
}
