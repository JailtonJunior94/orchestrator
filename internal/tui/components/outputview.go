package components

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
)

// OutputView wraps a bubbles viewport for displaying incremental provider output.
type OutputView struct {
	vp       viewport.Model
	lines    []string
	autoScroll bool
}

// NewOutputView returns an OutputView with the given dimensions.
func NewOutputView(width, height int) OutputView {
	vp := viewport.New(
		viewport.WithWidth(width),
		viewport.WithHeight(height),
	)
	vp.SoftWrap = true
	return OutputView{vp: vp, autoScroll: true}
}

// SetSize updates the viewport dimensions.
func (o *OutputView) SetSize(width, height int) {
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
	o.lines = append(o.lines, strings.Split(string(chunk), "\n")...)
	o.refresh()
	if o.autoScroll {
		o.scrollToBottom()
	}
}

// SetContent replaces all content.
func (o *OutputView) SetContent(content string) {
	o.lines = strings.Split(content, "\n")
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

// refresh re-sets viewport content from the accumulated lines.
func (o *OutputView) refresh() {
	o.vp.SetContent(strings.Join(o.lines, "\n"))
}

// scrollToBottom moves the viewport to the last line.
func (o *OutputView) scrollToBottom() {
	o.vp.SetYOffset(int(^uint(0) >> 1)) // MaxInt — viewport clamps internally
}
