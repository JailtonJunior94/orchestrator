package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
	"github.com/jailtonjunior/orchestrator/internal/tui/components"
	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

// listModel is the Bubbletea model for the interactive `orq list` command (FC-03).
type listModel struct {
	width, height int
	workflows     []runtimeapp.WorkflowSummary
	filtered      []runtimeapp.WorkflowSummary
	cursor        int
	filter        textinput.Model
	filterActive  bool
	theme         theme.Maestro
	statusBar     components.StatusBar
	selected      string // populated on Enter; empty means user quit
}

// newListModel creates the initial list model.
func newListModel(workflows []runtimeapp.WorkflowSummary) listModel {
	t := theme.New()
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."

	return listModel{
		workflows: workflows,
		filtered:  workflows,
		theme:     t,
		filter:    ti,
		statusBar: components.NewStatusBar(t),
	}
}

// Init returns nil — the list needs no startup commands.
func (m listModel) Init() tea.Cmd {
	return nil
}

// Update handles all messages for the list model.
func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.filterActive {
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.applyFilter()
		return m, cmd
	}
	return m, nil
}

// handleKey processes keyboard input for the list.
func (m listModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch msg.String() {
		case "esc":
			m.filterActive = false
			m.filter.Blur()
			return m, nil
		case "enter":
			m.filterActive = false
			m.filter.Blur()
			if len(m.filtered) > 0 {
				m.selected = m.filtered[m.cursor].Name
				return m, tea.Quit
			}
			return m, nil
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.applyFilter()
		return m, cmd
	}

	switch msg.String() {
	case "/":
		m.filterActive = true
		return m, m.filter.Focus()
	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "enter":
		if len(m.filtered) > 0 {
			m.selected = m.filtered[m.cursor].Name
			return m, tea.Quit
		}
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// applyFilter updates the filtered list and resets the cursor if needed.
func (m *listModel) applyFilter() {
	query := strings.ToLower(m.filter.Value())
	if query == "" {
		m.filtered = m.workflows
	} else {
		result := make([]runtimeapp.WorkflowSummary, 0, len(m.workflows))
		for _, w := range m.workflows {
			if strings.Contains(strings.ToLower(w.Name), query) {
				result = append(result, w)
			}
		}
		m.filtered = result
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

// View renders the list as a dual-pane TUI: names on the left, preview on the right.
func (m listModel) View() tea.View {
	s := m.renderList()
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m listModel) renderList() string {
	leftW := m.width / 3
	rightW := m.width - leftW
	if leftW < 20 {
		leftW = 20
	}
	if rightW < 10 {
		rightW = 10
	}
	// Content height: total - status (1) - borders (2)
	contentH := m.height - 3
	if contentH < 1 {
		contentH = 1
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary)

	innerLeftW := leftW - 4
	if innerLeftW < 1 {
		innerLeftW = 1
	}
	innerRightW := rightW - 4
	if innerRightW < 1 {
		innerRightW = 1
	}

	// Left pane: workflow list
	var leftSB strings.Builder
	if m.filterActive {
		leftSB.WriteString(m.filter.View())
		leftSB.WriteByte('\n')
	} else {
		leftSB.WriteString(lipgloss.NewStyle().Faint(true).Render("/ to filter"))
		leftSB.WriteByte('\n')
	}
	for i, wf := range m.filtered {
		label := truncateStr(fmt.Sprintf("%s (%d steps)", wf.Name, len(wf.StepNames)), innerLeftW-2)
		if i == m.cursor {
			leftSB.WriteString(lipgloss.NewStyle().
				Background(m.theme.Primary).
				Foreground(lipgloss.Color("#ffffff")).
				Width(innerLeftW).
				Render(" " + label))
		} else {
			leftSB.WriteString(" " + label)
		}
		leftSB.WriteByte('\n')
		if wf.Summary != "" {
			leftSB.WriteString(lipgloss.NewStyle().Faint(true).Render("  " + truncateStr(wf.Summary, innerLeftW-2)))
			leftSB.WriteByte('\n')
		}
	}
	if len(m.filtered) == 0 {
		leftSB.WriteString(lipgloss.NewStyle().Faint(true).Render("  No workflows found"))
		leftSB.WriteByte('\n')
	}
	leftPanel := panelStyle.Width(innerLeftW).Height(contentH).Render(leftSB.String())

	// Right pane: preview showing step names and providers
	var rightSB strings.Builder
	if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
		sel := m.filtered[m.cursor]
		rightSB.WriteString(lipgloss.NewStyle().Bold(true).Foreground(m.theme.Primary).Render(sel.Name))
		rightSB.WriteByte('\n')
		rightSB.WriteByte('\n')
		if sel.Description != "" {
			rightSB.WriteString(truncateStr(sel.Description, innerRightW))
			rightSB.WriteByte('\n')
			rightSB.WriteByte('\n')
		}

		if len(sel.StepNames) > 0 {
			rightSB.WriteString(lipgloss.NewStyle().Bold(true).Render("Steps:"))
			rightSB.WriteByte('\n')
			for i, step := range sel.StepNames {
				provider := ""
				if i < len(sel.Providers) {
					provider = sel.Providers[i]
				}
				line := fmt.Sprintf("  %d. %s", i+1, step)
				if provider != "" {
					line += fmt.Sprintf(" (%s)", provider)
				}
				rightSB.WriteString(lipgloss.NewStyle().Faint(true).Render(truncateStr(line, innerRightW)))
				rightSB.WriteByte('\n')
			}
			rightSB.WriteByte('\n')
		}

		if len(sel.Inputs) > 0 || sel.RequiresInput {
			rightSB.WriteString(lipgloss.NewStyle().Bold(true).Render("Inputs:"))
			rightSB.WriteByte('\n')
			if len(sel.Inputs) == 0 {
				rightSB.WriteString(lipgloss.NewStyle().Faint(true).Render("  - input"))
				rightSB.WriteByte('\n')
			}
			for _, input := range sel.Inputs {
				label := input.Label
				if label == "" {
					label = input.Name
				}
				fieldType := input.Type
				if fieldType == "" {
					fieldType = "text"
				}
				line := fmt.Sprintf("  - %s (%s)", label, fieldType)
				if input.Description != "" {
					line += ": " + input.Description
				}
				rightSB.WriteString(lipgloss.NewStyle().Faint(true).Render(truncateStr(line, innerRightW)))
				rightSB.WriteByte('\n')
			}
			rightSB.WriteByte('\n')
		}

		rightSB.WriteString(lipgloss.NewStyle().Faint(true).Render("Keys: j/k navigate  enter run  / filter  q quit"))
	}
	rightPanel := panelStyle.Width(innerRightW).Height(contentH).Render(rightSB.String())

	top := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, rightPanel)

	var sb strings.Builder
	sb.WriteString(top)
	sb.WriteByte('\n')
	sb.WriteString(m.statusBar.View(m.width))
	return sb.String()
}

// truncateStr trims s to maxLen runes with ellipsis.
func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// RunListTUI runs the interactive workflow list. It returns the selected
// workflow name (empty string if the user quit without selecting).
func RunListTUI(workflows []runtimeapp.WorkflowSummary) (string, error) {
	m := newListModel(workflows)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	if lm, ok := final.(listModel); ok {
		return lm.selected, nil
	}
	return "", nil
}
