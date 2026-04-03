package hitl

import (
	"context"
	"io"
)

// FakePrompter returns pre-programmed actions for tests.
type FakePrompter struct {
	results []PromptResult
	index   int
}

// NewFakePrompter creates a fake prompter with a deterministic sequence.
func NewFakePrompter(results ...PromptResult) *FakePrompter {
	return &FakePrompter{results: results}
}

// Prompt returns the next queued result.
func (p *FakePrompter) Prompt(_ context.Context, _ string) (PromptResult, error) {
	if p.index >= len(p.results) {
		return PromptResult{}, io.EOF
	}

	result := p.results[p.index]
	p.index++
	return result, nil
}
