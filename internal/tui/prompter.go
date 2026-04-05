package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
)

// Compile-time interface conformance check.
var _ hitl.Prompter = (*TUIPrompter)(nil)

// promptRequest carries the step output waiting for user action.
type promptRequest struct {
	output string
}

// promptResponseMsg is a Bubbletea message delivered when the engine
// sends a new HITL request through the TUIPrompter.
type promptResponseMsg struct {
	req promptRequest
}

// TUIPrompter implements hitl.Prompter via Bubbletea message passing.
// The engine goroutine calls Prompt, which blocks until the TUI user
// acts via the HITL bar.
type TUIPrompter struct {
	requestCh  chan promptRequest
	responseCh chan hitl.PromptResult
	done       chan struct{}
}

// NewTUIPrompter returns an initialised TUIPrompter.
func NewTUIPrompter() *TUIPrompter {
	p := &TUIPrompter{}
	p.Reset()
	return p
}

// Reset prepares the prompter for a new TUI session while preserving the same
// object identity already injected into the engine.
func (p *TUIPrompter) Reset() {
	p.requestCh = make(chan promptRequest, 1)
	p.responseCh = make(chan hitl.PromptResult, 1)
	p.done = make(chan struct{})
}

// Close unblocks any goroutine blocked in WaitForRequest when the TUI program
// exits. Safe to call multiple times — subsequent calls are no-ops.
func (p *TUIPrompter) Close() {
	select {
	case <-p.done:
	default:
		close(p.done)
	}
}

// Prompt blocks the calling goroutine until the TUI user selects an
// action or the context is cancelled.
func (p *TUIPrompter) Prompt(ctx context.Context, output string) (hitl.PromptResult, error) {
	req := promptRequest{output: output}
	select {
	case p.requestCh <- req:
	case <-ctx.Done():
		return hitl.PromptResult{}, ctx.Err()
	}
	select {
	case result := <-p.responseCh:
		return result, nil
	case <-ctx.Done():
		return hitl.PromptResult{}, ctx.Err()
	}
}

// WaitForRequest returns a tea.Cmd that reads the next HITL request from
// the engine goroutine without blocking the Bubbletea event loop.
// The goroutine is unblocked when Close is called.
func (p *TUIPrompter) WaitForRequest() tea.Cmd {
	return func() tea.Msg {
		select {
		case req := <-p.requestCh:
			return promptResponseMsg{req: req}
		case <-p.done:
			return nil
		}
	}
}

// Respond sends the user's action back to the blocked engine goroutine.
func (p *TUIPrompter) Respond(result hitl.PromptResult) {
	p.responseCh <- result
}
