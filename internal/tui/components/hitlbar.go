package components

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

// HITLActionMsg is emitted when the user selects an action in the HITL bar.
type HITLActionMsg struct {
	Action hitl.Action
}

// HITLBar is the prompt bar rendered at the bottom of the TUI when a step
// requires human-in-the-loop approval.
type HITLBar struct {
	stepName string
	output   string
	visible  bool
	theme    theme.Maestro
	width    int
}

// NewHITLBar returns a new HITLBar with the given theme.
func NewHITLBar(t theme.Maestro) HITLBar {
	return HITLBar{theme: t}
}

// Show makes the bar visible for the given step and provider output.
func (h *HITLBar) Show(stepName, output string) {
	h.stepName = stepName
	h.output = output
	h.visible = true
}

// Hide dismisses the bar.
func (h *HITLBar) Hide() {
	h.visible = false
}

// Visible reports whether the bar is currently shown.
func (h HITLBar) Visible() bool {
	return h.visible
}

// SetWidth sets the available width for rendering.
func (h *HITLBar) SetWidth(w int) {
	h.width = w
}

// Update handles key events while the HITL bar is visible.
// Returns (updated model, cmd, action selected). action is -1 when no action
// was taken (key not handled or bar not visible).
func (h HITLBar) Update(msg tea.Msg) (HITLBar, tea.Cmd, int) {
	if !h.visible {
		return h, nil, -1
	}
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return h, nil, -1
	}
	switch keyMsg.String() {
	case "a":
		h.visible = false
		return h, func() tea.Msg { return HITLActionMsg{Action: hitl.ActionApprove} }, int(hitl.ActionApprove)
	case "e":
		h.visible = false
		// Suspend the terminal so the external editor has full control, then
		// resume and notify the model that an edit was requested (RF-02.3).
		return h, tea.Batch(
			func() tea.Msg { return tea.SuspendMsg{} },
			func() tea.Msg { return HITLActionMsg{Action: hitl.ActionEdit} },
		), int(hitl.ActionEdit)
	case "r":
		h.visible = false
		return h, func() tea.Msg { return HITLActionMsg{Action: hitl.ActionRedo} }, int(hitl.ActionRedo)
	case "q":
		h.visible = false
		return h, func() tea.Msg { return HITLActionMsg{Action: hitl.ActionExit} }, int(hitl.ActionExit)
	}
	return h, nil, -1
}

// View renders the HITL bar. Returns an empty string when not visible.
func (h HITLBar) View() string {
	if !h.visible {
		return ""
	}

	accent := lipgloss.NewStyle().Foreground(h.theme.Primary).Bold(true)
	dim := lipgloss.NewStyle().Faint(true)
	key := func(k, label string) string {
		return accent.Render("["+k+"]") + dim.Render(label)
	}

	actions := fmt.Sprintf(" %s %s %s %s",
		key("a", "pprove"),
		key("e", "dit"),
		key("r", "edo"),
		key("q", "uit"),
	)

	label := lipgloss.NewStyle().
		Foreground(h.theme.Warning).
		Bold(true).
		Render(fmt.Sprintf("Step: %s", h.stepName))

	separator := dim.Render(" │")
	line := label + separator + actions

	return lipgloss.NewStyle().
		Width(h.width).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(h.theme.Primary).
		Render(line)
}
