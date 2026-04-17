package components

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// outputKind classifies the type of a content segment in the OutputView.
type outputKind int

const (
	outputText      outputKind = iota // plain accumulated text
	outputNamedLine                   // in-place updatable line (tool calls)
)

// outputItem represents one content segment in the OutputView.
type outputItem struct {
	kind   outputKind
	text   string // pre-styled content
	lineID string // non-empty for outputNamedLine; used for in-place updates
}

// OutputView wraps a bubbles viewport for displaying incremental provider output.
type OutputView struct {
	vp         viewport.Model
	items      []outputItem
	width      int
	autoScroll bool
}

// NewOutputView returns an OutputView with the given dimensions.
func NewOutputView(width, height int) OutputView {
	vp := viewport.New(
		viewport.WithWidth(width),
		viewport.WithHeight(height),
	)
	vp.SoftWrap = true
	return OutputView{vp: vp, width: width, autoScroll: true}
}

// AppendNamedLine appends a new in-place-updatable line identified by id.
// If a line with the same id already exists, it is updated instead (same
// semantics as UpdateNamedLine).
func (o *OutputView) AppendNamedLine(id, content string) {
	for i, item := range o.items {
		if item.kind == outputNamedLine && item.lineID == id {
			o.items[i].text = content
			o.refresh()
			if o.autoScroll {
				o.scrollToBottom()
			}
			return
		}
	}
	o.items = append(o.items, outputItem{kind: outputNamedLine, lineID: id, text: content})
	o.refresh()
	if o.autoScroll {
		o.scrollToBottom()
	}
}

// UpdateNamedLine updates the content of an existing named line by id.
// Returns true if the line was found and updated, false otherwise.
func (o *OutputView) UpdateNamedLine(id, content string) bool {
	for i, item := range o.items {
		if item.kind == outputNamedLine && item.lineID == id {
			o.items[i].text = content
			o.refresh()
			if o.autoScroll {
				o.scrollToBottom()
			}
			return true
		}
	}
	return false
}

// SetSize updates the viewport dimensions.
func (o *OutputView) SetSize(width, height int) {
	o.width = width
	o.vp.SetWidth(width)
	o.vp.SetHeight(height)
	o.refresh()
}

// AppendContent adds new output content and scrolls to the bottom when
// auto-scroll is active (i.e., the viewport is already at the bottom).
func (o *OutputView) AppendContent(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	// Extend the last text item, or add a new one.
	if len(o.items) > 0 && o.items[len(o.items)-1].kind == outputText {
		o.items[len(o.items)-1].text += string(chunk)
	} else {
		o.items = append(o.items, outputItem{kind: outputText, text: string(chunk)})
	}
	o.refresh()
	if o.autoScroll {
		o.scrollToBottom()
	}
}

// SetContent replaces all content with a single plain-text segment.
func (o *OutputView) SetContent(content string) {
	o.items = []outputItem{{kind: outputText, text: content}}
	o.refresh()
	if o.autoScroll {
		o.scrollToBottom()
	}
}

// Update handles Bubbletea messages, forwarding them to the inner viewport.
func (o *OutputView) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	o.vp, cmd = o.vp.Update(msg)
	// If user scrolled up, disable auto-scroll; if at bottom, re-enable.
	o.autoScroll = o.vp.AtBottom()
	return cmd
}

// View renders the viewport content as a string.
func (o OutputView) View() string {
	return o.vp.View()
}

// refresh re-sets viewport content from all accumulated items. Plain-text
// segments are passed through formatOutputForViewport for markdown-table
// detection; named lines (tool calls) are included as-is.
func (o *OutputView) refresh() {
	if len(o.items) == 0 {
		o.vp.SetContent("")
		return
	}

	// Fast path: single text item — preserve existing behaviour.
	if len(o.items) == 1 && o.items[0].kind == outputText {
		o.vp.SetContent(formatOutputForViewport(o.items[0].text, o.width))
		return
	}

	// Mixed items: apply markdown formatting to plain-text runs, preserve
	// named lines verbatim.
	var sb strings.Builder
	var textRun strings.Builder
	flushTextRun := func() {
		if textRun.Len() > 0 {
			sb.WriteString(formatOutputForViewport(textRun.String(), o.width))
			textRun.Reset()
		}
	}
	for _, item := range o.items {
		if item.kind == outputText {
			textRun.WriteString(item.text)
		} else {
			flushTextRun()
			sb.WriteString(item.text)
			sb.WriteByte('\n')
		}
	}
	flushTextRun()
	o.vp.SetContent(sb.String())
}

// scrollToBottom moves the viewport to the last line.
func (o *OutputView) scrollToBottom() {
	o.vp.SetYOffset(int(^uint(0) >> 1)) // MaxInt — viewport clamps internally
}

func formatOutputForViewport(content string, width int) string {
	if strings.TrimSpace(content) == "" {
		return content
	}

	lines := strings.Split(content, "\n")
	rendered := make([]string, 0, len(lines))

	for i := 0; i < len(lines); {
		if i+1 < len(lines) && isMarkdownTableHeader(lines[i], lines[i+1]) {
			tableLines := []string{lines[i], lines[i+1]}
			i += 2
			for i < len(lines) && isMarkdownTableRow(lines[i]) {
				tableLines = append(tableLines, lines[i])
				i++
			}
			rendered = append(rendered, renderMarkdownTable(tableLines, width)...)
			continue
		}

		rendered = append(rendered, lines[i])
		i++
	}

	return strings.Join(rendered, "\n")
}

func isMarkdownTableHeader(header, separator string) bool {
	return isMarkdownTableRow(header) && isMarkdownTableSeparator(separator)
}

func isMarkdownTableRow(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.Count(trimmed, "|") < 2 {
		return false
	}

	cells := parseMarkdownTableRow(trimmed)
	return len(cells) >= 2
}

func isMarkdownTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.Count(trimmed, "|") < 2 {
		return false
	}

	for _, cell := range parseMarkdownTableRow(trimmed) {
		if cell == "" {
			continue
		}
		for _, r := range cell {
			if r != '-' && r != ':' {
				return false
			}
		}
	}

	return true
}

func parseMarkdownTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")

	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, normalizeMarkdownCell(part))
	}

	return cells
}

func normalizeMarkdownCell(cell string) string {
	return strings.TrimSpace(strings.ReplaceAll(cell, `\|`, "|"))
}

func renderMarkdownTable(lines []string, width int) []string {
	if len(lines) < 2 {
		return lines
	}

	headers := parseMarkdownTableRow(lines[0])
	rows := make([][]string, 0, len(lines)-2)
	for _, line := range lines[2:] {
		row := parseMarkdownTableRow(line)
		if len(row) == 0 {
			continue
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return []string{lines[0]}
	}

	out := make([]string, 0, len(rows)*(len(headers)+2)+1)
	if width >= 24 {
		out = append(out, "Table view:")
	}

	for idx, row := range rows {
		out = append(out, fmt.Sprintf("[%d]", idx+1))
		for colIdx, header := range headers {
			value := ""
			if colIdx < len(row) {
				value = row[colIdx]
			}
			label := header
			if label == "" {
				label = fmt.Sprintf("Column %d", colIdx+1)
			}
			out = append(out, formatTableCellLine(label, value, width))
		}
		if idx < len(rows)-1 {
			out = append(out, "")
		}
	}

	return out
}

func formatTableCellLine(label, value string, width int) string {
	line := fmt.Sprintf("%s: %s", label, value)
	if width <= 0 || len([]rune(line)) <= width {
		return line
	}

	prefix := label + ": "
	wrapped := wrapText(value, max(width-len([]rune(prefix)), 12))
	if len(wrapped) == 0 {
		return prefix
	}

	lines := make([]string, 0, len(wrapped))
	for idx, part := range wrapped {
		if idx == 0 {
			lines = append(lines, prefix+part)
			continue
		}
		lines = append(lines, strings.Repeat(" ", len([]rune(prefix)))+part)
	}

	return strings.Join(lines, "\n")
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, len(words))
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if len([]rune(candidate)) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	lines = append(lines, current)
	return lines
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
