package components

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
	"github.com/jailtonjunior/orchestrator/internal/output"
	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

// HITLActionMsg is emitted when the user selects an action in the HITL bar.
type HITLActionMsg struct {
	Action hitl.Action
	Output string
}

type answerOption struct {
	Label string
	Value string
}

// HITLBar is the prompt bar rendered at the bottom of the TUI when a step
// requires human-in-the-loop approval.
type HITLBar struct {
	stepName string
	output   string
	visible  bool
	theme    theme.Maestro
	width    int
	options  []answerOption
	cursor   int
	custom   textinput.Model
	entering bool
}

// NewHITLBar returns a new HITLBar with the given theme.
func NewHITLBar(t theme.Maestro) HITLBar {
	custom := textinput.New()
	custom.Placeholder = "Type your answer..."
	custom.Prompt = "> "
	return HITLBar{theme: t, custom: custom}
}

// Show makes the bar visible for the given step and provider output.
func (h *HITLBar) Show(stepName, output string) {
	h.stepName = stepName
	h.output = output
	h.visible = true
	h.options = buildAnswerOptions(output)
	h.cursor = 0
	h.entering = false
	h.custom.Reset()
	h.custom.Blur()
}

// Hide dismisses the bar.
func (h *HITLBar) Hide() {
	h.visible = false
	h.entering = false
	h.custom.Reset()
	h.custom.Blur()
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
		if h.entering {
			var cmd tea.Cmd
			h.custom, cmd = h.custom.Update(msg)
			return h, cmd, -1
		}
		return h, nil, -1
	}

	if h.entering {
		switch keyMsg.String() {
		case "esc":
			h.entering = false
			h.custom.Reset()
			h.custom.Blur()
			return h, nil, -1
		case "enter":
			answer := strings.TrimSpace(h.custom.Value())
			if answer == "" {
				return h, nil, -1
			}
			h.visible = false
			h.entering = false
			return h, func() tea.Msg {
				return HITLActionMsg{Action: hitl.ActionEdit, Output: answer}
			}, int(hitl.ActionEdit)
		default:
			var cmd tea.Cmd
			h.custom, cmd = h.custom.Update(msg)
			return h, cmd, -1
		}
	}

	if len(h.options) > 0 {
		switch keyMsg.String() {
		case "up", "k":
			if h.cursor > 0 {
				h.cursor--
			}
			return h, nil, -1
		case "down", "j":
			if h.cursor < len(h.options) {
				h.cursor++
			}
			return h, nil, -1
		case "enter":
			if h.cursor == len(h.options) {
				h.entering = true
				return h, h.custom.Focus(), -1
			}
			h.visible = false
			selected := h.options[h.cursor]
			return h, func() tea.Msg {
				return HITLActionMsg{Action: hitl.ActionEdit, Output: selected.Value}
			}, int(hitl.ActionEdit)
		}
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
	selectedStyle := lipgloss.NewStyle().
		Background(h.theme.Primary).
		Foreground(lipgloss.Color("#ffffff"))
	key := func(k, label string) string {
		return accent.Render("["+k+"]") + dim.Render(label)
	}

	label := lipgloss.NewStyle().
		Foreground(h.theme.Warning).
		Bold(true).
		Render(fmt.Sprintf("Step: %s", h.stepName))
	actions := fmt.Sprintf("%s %s %s %s",
		key("a", "pprove raw output"),
		key("e", "dit in editor"),
		key("r", "edo"),
		key("q", "uit"),
	)

	lines := []string{label}
	if len(h.options) > 0 {
		lines = append(lines, dim.Render("Choose how to answer this prompt:"))
		for i, option := range h.options {
			line := fmt.Sprintf(" %d. %s", i+1, option.Label)
			if i == h.cursor && !h.entering {
				line = selectedStyle.Render(line)
			}
			lines = append(lines, line)
		}
		customLine := " " + fmt.Sprintf("%d. %s", len(h.options)+1, "Custom answer")
		if h.cursor == len(h.options) && !h.entering {
			customLine = selectedStyle.Render(customLine)
		}
		lines = append(lines, customLine)
		if h.entering {
			lines = append(lines, dim.Render("Type a custom answer and press enter. Esc returns to the choices."))
			lines = append(lines, h.custom.View())
		} else {
			lines = append(lines, dim.Render("↑/↓ navigate  enter select"))
		}
	}
	lines = append(lines, dim.Render(actions))

	return lipgloss.NewStyle().
		Width(h.width).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(h.theme.Primary).
		Render(strings.Join(lines, "\n"))
}

var listItemPattern = regexp.MustCompile(`^(?:[-*+]|[0-9]+[.)])\s+(.+)$`)

func buildAnswerOptions(rawOutput string) []answerOption {
	output := normalizePromptOutput(rawOutput)
	questions := extractOpenQuestions(output)
	if len(questions) > 0 {
		return canonicalQuestionOptions(questions)
	}

	if choices := extractQuestionChoices(output); len(choices) > 0 {
		options := make([]answerOption, 0, len(choices))
		for _, choice := range choices {
			options = append(options, answerOption{
				Label: choice,
				Value: choice,
			})
		}
		return options
	}

	return nil
}

func extractQuestionChoices(output string) []string {
	lines := strings.Split(normalizePromptOutput(output), "\n")
	for i := 0; i < len(lines); i++ {
		match := listItemPattern.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if match == nil {
			continue
		}

		start := i
		choices := make([]string, 0, 4)
		for ; i < len(lines); i++ {
			itemMatch := listItemPattern.FindStringSubmatch(strings.TrimSpace(lines[i]))
			if itemMatch == nil {
				break
			}
			choices = append(choices, strings.TrimSpace(itemMatch[1]))
		}

		if len(choices) < 2 {
			continue
		}
		if allQuestions(choices) {
			continue
		}
		if hasQuestionContext(lines, start) {
			return choices
		}
	}

	return nil
}

func extractOpenQuestions(output string) []string {
	lines := strings.Split(normalizePromptOutput(output), "\n")
	questions := make([]string, 0, 4)

	for _, line := range lines {
		trimmed := sanitizePromptLine(line)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "?") {
			questions = append(questions, trimmed)
		}
	}

	if len(questions) == 0 {
		return nil
	}
	if !hasClarificationContext(lines) {
		return nil
	}

	return questions
}

func hasClarificationContext(lines []string) bool {
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		for _, marker := range []string{
			"question:",
			"questions:",
			"pergunta:",
			"perguntas:",
			"choose",
			"pick one",
			"select",
			"escolha",
			"selecione",
			"responda",
			"answer what you know",
			"antes de gerar",
			"before generating",
		} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}

	return false
}

func allQuestions(items []string) bool {
	for _, item := range items {
		if !strings.Contains(strings.TrimSpace(item), "?") {
			return false
		}
	}

	return len(items) > 0
}

func hasQuestionContext(lines []string, listStart int) bool {
	for i := listStart - 1; i >= 0 && i >= listStart-3; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		if strings.Contains(line, "?") {
			return true
		}
		for _, marker := range []string{
			"question:",
			"pergunta:",
			"choose",
			"pick one",
			"select",
			"escolha",
			"selecione",
			"responda",
		} {
			if strings.Contains(lower, marker) {
				return true
			}
		}
	}

	return false
}

func sanitizePromptLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}

	if match := listItemPattern.FindStringSubmatch(trimmed); match != nil {
		trimmed = strings.TrimSpace(match[1])
	}

	trimmed = strings.Trim(trimmed, "\"")
	trimmed = strings.TrimPrefix(trimmed, "**")
	trimmed = strings.TrimSuffix(trimmed, "**")
	trimmed = strings.TrimSpace(trimmed)
	return trimmed
}

func canonicalQuestionOptions(questions []string) []answerOption {
	subject := "the open question"
	if len(questions) > 1 {
		subject = "the open questions"
	}

	return []answerOption{
		{
			Label: "I don't know yet",
			Value: fmt.Sprintf("I don't know yet. Continue with explicit assumptions for %s and mark unknown details as TBD.", subject),
		},
		{
			Label: "Use reasonable assumptions",
			Value: fmt.Sprintf("Use reasonable assumptions for %s, call them out clearly, and continue.", subject),
		},
		{
			Label: "Mark as TBD and continue",
			Value: fmt.Sprintf("These details are still TBD. Leave clear placeholders for %s and continue.", subject),
		},
	}
}

func normalizePromptOutput(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}

	if json.Valid([]byte(text)) {
		if extracted, err := output.ExtractProviderJSONResponse(text); err == nil {
			text = extracted
		}
	}

	replacer := strings.NewReplacer(
		`\r\n`, "\n",
		`\n`, "\n",
		`\t`, "\t",
	)

	return replacer.Replace(text)
}
