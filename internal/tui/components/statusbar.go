package components

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

const (
	// narrowTerminalWidth is the column threshold below which less-critical
	// sections (branch, latency) are omitted from the status bar (RF-05.5).
	narrowTerminalWidth = 100

	sectionSep = " │ "
)

// StatusBar renders a single-line status bar with operational context.
type StatusBar struct {
	Branch      string
	Provider    string
	CurrentStep int
	TotalSteps  int
	Duration    time.Duration
	theme       theme.Maestro
}

// NewStatusBar returns a StatusBar with the given theme.
func NewStatusBar(t theme.Maestro) StatusBar {
	return StatusBar{theme: t}
}

// View renders the status bar as a single line truncated to width columns.
func (s StatusBar) View(width int) string {
	provStyle := lipgloss.NewStyle().Foreground(s.theme.Primary).Bold(true)
	stepStyle := lipgloss.NewStyle().Foreground(s.theme.Success)
	dimStyle := lipgloss.NewStyle().Faint(true)

	stepSection := stepStyle.Render(
		fmt.Sprintf("step %d/%d", s.CurrentStep, s.TotalSteps),
	)
	durSection := dimStyle.Render(formatDuration(s.Duration))
	provSection := provStyle.Render(s.Provider)

	var parts []string

	if width >= narrowTerminalWidth && s.Branch != "" {
		parts = append(parts, dimStyle.Render(s.Branch))
	}

	parts = append(parts, provSection, stepSection, durSection)

	line := strings.Join(parts, sectionSep)

	// Pad or truncate to exactly width columns.
	plain := ansiStrip(line)
	if len([]rune(plain)) < width {
		line += strings.Repeat(" ", width-len([]rune(plain)))
	}

	return line
}

// formatDuration converts a duration to a compact human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", m, sec)
}

// ansiStrip removes ANSI escape sequences from s for width calculation.
// This is a minimal implementation sufficient for the status bar.
func ansiStrip(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++ // skip 'm'
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}
