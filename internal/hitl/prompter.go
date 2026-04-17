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

// PermissionDecision captures the decision taken for an ACP permission request.
type PermissionDecision string

const (
	// PermissionAllow approves the requested ACP operation.
	PermissionAllow PermissionDecision = "allow"
	// PermissionDeny rejects the requested ACP operation.
	PermissionDeny PermissionDecision = "deny"
	// PermissionCancel aborts the permission flow without choosing allow/deny.
	PermissionCancel PermissionDecision = "cancel"
)

// PermissionOption mirrors one ACP permission option exposed by the agent.
type PermissionOption struct {
	ID   string
	Name string
	Kind string
}

// PermissionRequest describes an ACP permission request in UI-friendly terms.
type PermissionRequest struct {
	Provider   string
	Workflow   string
	ToolCallID string
	Title      string
	ToolKind   string
	Details    string
	Options    []PermissionOption
}

// PermissionResult captures the permission decision taken by the user.
type PermissionResult struct {
	Decision PermissionDecision
}

// PermissionPrompter is implemented by UIs that can explicitly handle ACP
// permission requests without routing them through the generic step HITL flow.
type PermissionPrompter interface {
	PromptPermission(ctx context.Context, request PermissionRequest) (PermissionResult, error)
}
