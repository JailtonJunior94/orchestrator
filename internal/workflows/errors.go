package workflows

import "errors"

var (
	ErrEmptyWorkflowName    = errors.New("workflow name must not be empty")
	ErrNoSteps              = errors.New("workflow must have at least one step")
	ErrEmptyStepName        = errors.New("step name must not be empty")
	ErrEmptyProvider        = errors.New("step provider must not be empty")
	ErrEmptyInput           = errors.New("step input must not be empty")
	ErrDuplicateStepName    = errors.New("duplicate step name")
	ErrInvalidProvider      = errors.New("unknown provider")
	ErrInvalidStepReference = errors.New("step references unknown step")
	ErrInvalidTimeout       = errors.New("step timeout must be a valid duration")
	ErrInvalidSchema        = errors.New("step schema must be valid json")
	ErrUnresolvedVariable   = errors.New("unresolved template variable")
	ErrWorkflowNotFound     = errors.New("workflow not found")
)
