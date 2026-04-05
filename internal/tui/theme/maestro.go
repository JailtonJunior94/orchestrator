// Package theme defines the Maestro visual identity for the orq TUI.
package theme

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Maestro holds the Maestro color palette used across all TUI components.
type Maestro struct {
	Primary color.Color
	Success color.Color
	Warning color.Color
	Error   color.Color
}

// New returns a Maestro theme. When the terminal does not support true-color,
// the palette falls back to the nearest ANSI 256 codes.
func New() Maestro {
	if hasTrueColor() {
		return trueColorPalette()
	}
	return fallback256()
}

// trueColorPalette returns the Maestro palette in 24-bit hex colors.
func trueColorPalette() Maestro {
	return Maestro{
		Primary: lipgloss.Color("#5f5fd7"),
		Success: lipgloss.Color("#00af87"),
		Warning: lipgloss.Color("#dfaf5f"),
		Error:   lipgloss.Color("#df5f87"),
	}
}

// fallback256 returns the nearest 256-color ANSI approximations for each
// Maestro palette entry.
func fallback256() Maestro {
	return Maestro{
		Primary: lipgloss.Color("63"),  // xterm: SlateBlue1
		Success: lipgloss.Color("36"),  // xterm: DarkCyan
		Warning: lipgloss.Color("179"), // xterm: LightGoldenrod3
		Error:   lipgloss.Color("168"), // xterm: HotPink3
	}
}

// hasTrueColor reports whether the terminal connected to stdout supports
// 24-bit color. It uses a heuristic based on the COLORTERM environment
// variable, which is the most reliable cross-platform indicator.
func hasTrueColor() bool {
	ct := os.Getenv("COLORTERM")
	if ct == "truecolor" || ct == "24bit" {
		return true
	}
	// Accept if stdout is a terminal and COLORTERM is set at all.
	fd := int(os.Stdout.Fd())
	return term.IsTerminal(fd) && ct != ""
}
