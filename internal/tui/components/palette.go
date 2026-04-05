package components

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

// PaletteItem represents a command or workflow entry in the command palette.
type PaletteItem struct {
	Key         string // internal identifier used by the caller
	Title       string // display label shown in the list
	Description string // short description (not currently rendered)
}

// PaletteModel is the command palette overlay component (FC-04).
// It holds a text input for fuzzy search and a navigable list of items.
type PaletteModel struct {
	visible  bool
	input    textinput.Model
	items    []PaletteItem
	filtered []PaletteItem
	cursor   int
	width    int
	height   int
	theme    theme.Maestro
}

// NewPaletteModel creates a PaletteModel with the supplied items and theme.
func NewPaletteModel(items []PaletteItem, t theme.Maestro) PaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Search commands and workflows..."
	return PaletteModel{
		input:    ti,
		items:    items,
		filtered: items,
		theme:    t,
	}
}

// SetItems replaces the full item list and re-applies the current filter.
func (p *PaletteModel) SetItems(items []PaletteItem) {
	p.items = items
	p.filtered = fuzzyFilter(items, p.input.Value())
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
}

// Toggle shows (or hides) the palette and returns the Focus cmd when opening.
func (p *PaletteModel) Toggle() tea.Cmd {
	p.visible = !p.visible
	if p.visible {
		p.input.Reset()
		p.filtered = p.items
		p.cursor = 0
		return p.input.Focus()
	}
	p.input.Blur()
	return nil
}

// Visible reports whether the palette overlay is currently shown.
func (p PaletteModel) Visible() bool {
	return p.visible
}

// SetSize stores the terminal dimensions used to center the overlay.
func (p *PaletteModel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

// Update handles keyboard and other messages while the palette is visible.
// It returns the updated model, any follow-up command, and a pointer to the
// selected item (non-nil only when the user presses Enter on a result).
func (p PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd, *PaletteItem) {
	kMsg, isKey := msg.(tea.KeyPressMsg)
	if isKey {
		switch kMsg.String() {
		case "esc":
			p.visible = false
			p.input.Blur()
			return p, nil, nil
		case "enter":
			if len(p.filtered) > 0 {
				selected := p.filtered[p.cursor]
				p.visible = false
				p.input.Blur()
				return p, nil, &selected
			}
			return p, nil, nil
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
			return p, nil, nil
		case "down", "j":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
			}
			return p, nil, nil
		}
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	p.filtered = fuzzyFilter(p.items, p.input.Value())
	if p.cursor >= len(p.filtered) {
		p.cursor = 0
	}
	return p, cmd, nil
}

const (
	paletteBoxWidth = 60
	paletteMaxItems = 8
)

// View renders the command palette as a centered overlay.
func (p PaletteModel) View() string {
	boxW := paletteBoxWidth
	if p.width > 0 && p.width < boxW+4 {
		boxW = p.width - 4
	}
	if boxW < 20 {
		boxW = 20
	}
	innerW := boxW - 2 // subtract border

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.theme.Primary).
		Width(innerW)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(p.theme.Primary)
	hintStyle := lipgloss.NewStyle().Faint(true)

	var sb strings.Builder
	sb.WriteString(titleStyle.Render("Command Palette"))
	sb.WriteByte('\n')
	sb.WriteString(p.input.View())
	sb.WriteByte('\n')
	sb.WriteByte('\n')

	for i, item := range p.filtered {
		if i >= paletteMaxItems {
			break
		}
		label := truncate(item.Title, innerW-2)
		if i == p.cursor {
			line := lipgloss.NewStyle().
				Background(p.theme.Primary).
				Foreground(lipgloss.Color("#ffffff")).
				Width(innerW).
				Render(" " + label)
			sb.WriteString(line)
		} else {
			sb.WriteString(" " + label)
		}
		sb.WriteByte('\n')
	}
	if len(p.filtered) == 0 {
		sb.WriteString(hintStyle.Render("  No results"))
		sb.WriteByte('\n')
	}

	sb.WriteByte('\n')
	sb.WriteString(hintStyle.Render("↑/↓ navigate  enter select  esc close"))

	box := boxStyle.Render(sb.String())

	if p.width <= 0 || p.height <= 0 {
		return box
	}
	return lipgloss.Place(p.width, p.height, lipgloss.Center, lipgloss.Center, box)
}

// FuzzyFilter returns the subset of items whose Title contains query
// (case-insensitive). Exported for use in tests and callers.
func FuzzyFilter(items []PaletteItem, query string) []PaletteItem {
	return fuzzyFilter(items, query)
}

func fuzzyFilter(items []PaletteItem, query string) []PaletteItem {
	if query == "" {
		return items
	}
	q := strings.ToLower(query)
	result := make([]PaletteItem, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Title), q) {
			result = append(result, item)
		}
	}
	return result
}
