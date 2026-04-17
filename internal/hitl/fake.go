package hitl

import (
	"context"
	"io"
)

// FakePrompter returns pre-programmed actions for tests.
type FakePrompter struct {
	results            []PromptResult
	permissionResults  []PermissionResult
	index              int
	permissionResultIx int
}

// NewFakePrompter creates a fake prompter with a deterministic sequence.
func NewFakePrompter(results ...PromptResult) *FakePrompter {
	return &FakePrompter{results: results}
}

// NewFakePermissionPrompter creates a fake prompter with deterministic ACP
// permission responses.
func NewFakePermissionPrompter(results ...PermissionResult) *FakePrompter {
	return &FakePrompter{permissionResults: results}
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

// PromptPermission returns the next queued ACP permission decision.
func (p *FakePrompter) PromptPermission(_ context.Context, _ PermissionRequest) (PermissionResult, error) {
	if len(p.permissionResults) == 0 {
		result, err := p.Prompt(context.Background(), "")
		if err != nil {
			return PermissionResult{}, err
		}
		if result.Action == ActionApprove {
			return PermissionResult{Decision: PermissionAllow}, nil
		}
		return PermissionResult{Decision: PermissionDeny}, nil
	}

	if p.permissionResultIx >= len(p.permissionResults) {
		return PermissionResult{}, io.EOF
	}

	result := p.permissionResults[p.permissionResultIx]
	p.permissionResultIx++
	return result, nil
}
