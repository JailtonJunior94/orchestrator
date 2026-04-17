package acp

import (
	"errors"
	"time"
)

// ACPInput encapsulates a prompt request to an ACP agent.
type ACPInput struct {
	// Prompt is the resolved prompt text to send to the agent.
	Prompt string

	// SessionID is the ACP session to resume. Empty means create a new session.
	SessionID string

	// Timeout is the per-operation deadline for this execution.
	Timeout time.Duration

	// WorkDir is the working directory for the session. Required when creating a new session.
	WorkDir string

	// RunID identifies the workflow run owning this ACP execution.
	RunID string

	// WorkflowName identifies the workflow owning this ACP execution.
	WorkflowName string

	// StepName identifies the workflow step owning this ACP execution.
	StepName string

	// ProviderName identifies the ACP provider executing this request.
	ProviderName string

	// PermissionPolicy overrides the permission policy for this execution.
	PermissionPolicy PermissionPolicy
}

// Validate reports an error if the input is not usable.
func (i ACPInput) Validate() error {
	if i.Prompt == "" {
		return errors.New("acp input: prompt must not be empty")
	}
	return nil
}
