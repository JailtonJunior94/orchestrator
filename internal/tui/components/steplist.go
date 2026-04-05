// Package components provides reusable Bubbletea TUI components for the orq CLI.
package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

// StepItem is the view model for a single workflow step.
type StepItem struct {
	Name     string
	Provider string
	Status   string // maps to domain.StepStatus values
}

// StepList renders the left panel listing all workflow steps with status icons.
type StepList struct {
	steps        []StepItem
	activeStep   int
	width        int
	theme        theme.Maestro
	spin         spinner.Model
	noAnimation  bool
}

// NewStepList returns a new StepList with the given theme.
func NewStepList(t theme.Maestro) StepList {
	sp := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	sp.Style = lipgloss.NewStyle().Foreground(t.Primary)
	return StepList{theme: t, spin: sp}
}

// SetNoAnimation controls whether the spinner is rendered or replaced with a
// static icon.
func (s *StepList) SetNoAnimation(disabled bool) {
	s.noAnimation = disabled
}

// UpdateSpinner forwards messages to the spinner so it can advance its frame.
// Returns the tick command produced by the spinner (if any).
func (s *StepList) UpdateSpinner(msg tea.Msg) tea.Cmd {
	if s.noAnimation {
		return nil
	}
	var cmd tea.Cmd
	s.spin, cmd = s.spin.Update(msg)
	return cmd
}

// TickSpinner returns a command that drives the spinner animation.
func (s StepList) TickSpinner() tea.Cmd {
	if s.noAnimation {
		return nil
	}
	return s.spin.Tick
}

// SetSteps replaces the step list.
func (s *StepList) SetSteps(steps []StepItem) {
	s.steps = steps
}

// SetActiveStep sets the index of the currently executing step.
func (s *StepList) SetActiveStep(idx int) {
	s.activeStep = idx
}

// SetWidth sets the available width for rendering (including borders).
func (s *StepList) SetWidth(w int) {
	s.width = w
}

// innerWidth returns the usable inner width (width minus 4 for borders + padding).
func (s StepList) innerWidth() int {
	inner := s.width - 4
	if inner < 1 {
		return 1
	}
	return inner
}

// View renders the step list as a string.
func (s StepList) View() string {
	inner := s.innerWidth()
	var sb strings.Builder
	for i, item := range s.steps {
		icon := s.iconFor(item.Status)
		style := stepStyle(s.theme, item.Status)

		// Format: icon + name truncated
		label := truncate(fmt.Sprintf(" %s %s", icon, item.Name), inner)
		line := style.Render(label)

		// Highlight active step with background accent
		if i == s.activeStep {
			line = lipgloss.NewStyle().
				Background(s.theme.Primary).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Render(truncate(fmt.Sprintf(" %s %s", icon, item.Name), inner))
		}

		sb.WriteString(line)
		sb.WriteByte('\n')

		// Provider line (dim)
		if item.Provider != "" {
			prov := lipgloss.NewStyle().Faint(true).
				Render(truncate("   "+item.Provider, inner))
			sb.WriteString(prov)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// iconFor returns the display icon for a step, using the spinner frame when
// the step is running and animations are enabled.
func (s StepList) iconFor(status string) string {
	if status == "running" && !s.noAnimation {
		return s.spin.View()
	}
	return stepIcon(status)
}

// stepIcon returns the display icon for a step status.
func stepIcon(status string) string {
	switch status {
	case "running":
		return "●"
	case "approved", "completed":
		return "✓"
	case "failed":
		return "✗"
	case "skipped":
		return "→"
	case "waiting_approval":
		return "◎"
	case "retrying":
		return "↺"
	default: // pending
		return "○"
	}
}

// stepStyle returns the Lipgloss style for a step given its status.
func stepStyle(t theme.Maestro, status string) lipgloss.Style {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	case "approved", "waiting_approval":
		return lipgloss.NewStyle().Foreground(t.Success)
	case "failed":
		return lipgloss.NewStyle().Foreground(t.Error)
	case "retrying":
		return lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
	case "skipped":
		return lipgloss.NewStyle().Faint(true)
	default:
		return lipgloss.NewStyle()
	}
}

// truncate trims s to at most maxLen runes, appending "…" if truncated.
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
