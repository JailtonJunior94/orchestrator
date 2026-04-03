package hitl

import "context"

// Action is the user decision taken after a step output is displayed.
type Action int

const (
	ActionApprove Action = iota
	ActionEdit
	ActionRedo
	ActionExit
)

// PromptResult captures the selected action and any edited output.
type PromptResult struct {
	Action Action
	Output string
}

// Prompter requests human approval between steps.
type Prompter interface {
	Prompt(ctx context.Context, output string) (PromptResult, error)
}
