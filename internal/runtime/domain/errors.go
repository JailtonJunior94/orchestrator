package domain

import "errors"

var (
	ErrEmptyWorkflowName  = errors.New("workflow name must not be empty")
	ErrEmptyStepName      = errors.New("step name must not be empty")
	ErrEmptyProviderName  = errors.New("provider name must not be empty")
	ErrInvalidRunStatus   = errors.New("invalid run status")
	ErrInvalidStepStatus  = errors.New("invalid step status")
	ErrInvalidTransition  = errors.New("invalid state transition")
	ErrNoSteps            = errors.New("run must have at least one step")
	ErrStepNotFound       = errors.New("step not found")
	ErrNoCurrentStep      = errors.New("no current step available")
	ErrRunNotRunning      = errors.New("run is not in running state")
	ErrRunNotPaused       = errors.New("run is not in paused state")
	ErrDuplicateStepName  = errors.New("duplicate step name")
	ErrEmptyRunID         = errors.New("run id must not be empty")
	ErrEmptyInput         = errors.New("input must not be empty")
)
