package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

// Panel returns a rounded-border panel style using the primary color.
func Panel(t theme.Maestro) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary)
}

// Title returns a bold title style.
func Title(t theme.Maestro) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Primary)
}

// Metadata returns a dim style for secondary metadata.
func Metadata(_ theme.Maestro) lipgloss.Style {
	return lipgloss.NewStyle().Faint(true)
}

// StepStyle returns the Lipgloss style for a step item given its status.
func StepStyle(t theme.Maestro, status string) lipgloss.Style {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
	case "approved", "waiting_approval":
		return lipgloss.NewStyle().Foreground(t.Success)
	case "failed":
		return lipgloss.NewStyle().Foreground(t.Error)
	case "skipped":
		return lipgloss.NewStyle().Faint(true)
	default: // pending
		return lipgloss.NewStyle()
	}
}

// ThoughtStyle returns a faint style for agent chain-of-thought output.
func ThoughtStyle(_ theme.Maestro) lipgloss.Style {
	return lipgloss.NewStyle().Faint(true)
}

// ToolCallStyle returns a style for tool call progress indicators.
func ToolCallStyle(t theme.Maestro) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Warning)
}

// PermissionStyle returns a style for ACP permission request notifications.
func PermissionStyle(t theme.Maestro) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Warning).Bold(true)
}

// StepIcon returns the icon character for a step status.
func StepIcon(status string) string {
	switch status {
	case "running":
		return iconRunning
	case "approved":
		return iconApproved
	case "failed":
		return iconFailed
	case "skipped":
		return iconSkipped
	case "waiting_approval":
		return iconWaitingApproval
	default: // pending
		return iconPending
	}
}
