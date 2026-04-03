package domain

import (
	"fmt"
	"slices"
)

// RunStatus represents the status of a run.
type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunPaused    RunStatus = "paused"
	RunFailed    RunStatus = "failed"
	RunCompleted RunStatus = "completed"
	RunCancelled RunStatus = "cancelled"
)

var validRunStatuses = map[RunStatus]bool{
	RunPending:   true,
	RunRunning:   true,
	RunPaused:    true,
	RunFailed:    true,
	RunCompleted: true,
	RunCancelled: true,
}

// runTransitions defines allowed state transitions for RunStatus.
var runTransitions = map[RunStatus][]RunStatus{
	RunPending:   {RunRunning, RunCancelled},
	RunRunning:   {RunPaused, RunFailed, RunCompleted, RunCancelled},
	RunPaused:    {RunRunning, RunCancelled},
	RunFailed:    {},
	RunCompleted: {},
	RunCancelled: {},
}

// ValidateRunStatus checks if a RunStatus value is valid.
func ValidateRunStatus(s RunStatus) error {
	if !validRunStatuses[s] {
		return fmt.Errorf("%w: %q", ErrInvalidRunStatus, s)
	}
	return nil
}

// CanTransitionTo checks if the run status can transition to the target status.
func (s RunStatus) CanTransitionTo(target RunStatus) bool {
	return slices.Contains(runTransitions[s], target)
}

// TransitionTo attempts to transition to the target status, returning an error if invalid.
func (s RunStatus) TransitionTo(target RunStatus) (RunStatus, error) {
	if !s.CanTransitionTo(target) {
		return s, fmt.Errorf("%w: %q -> %q", ErrInvalidTransition, s, target)
	}
	return target, nil
}

// StepStatus represents the status of a step execution.
type StepStatus string

const (
	StepPending         StepStatus = "pending"
	StepRunning         StepStatus = "running"
	StepWaitingApproval StepStatus = "waiting_approval"
	StepApproved        StepStatus = "approved"
	StepRetrying        StepStatus = "retrying"
	StepFailed          StepStatus = "failed"
	StepSkipped         StepStatus = "skipped"
)

var validStepStatuses = map[StepStatus]bool{
	StepPending:         true,
	StepRunning:         true,
	StepWaitingApproval: true,
	StepApproved:        true,
	StepRetrying:        true,
	StepFailed:          true,
	StepSkipped:         true,
}

// stepTransitions defines allowed state transitions for StepStatus.
var stepTransitions = map[StepStatus][]StepStatus{
	StepPending:         {StepRunning, StepSkipped},
	StepRunning:         {StepWaitingApproval, StepFailed},
	StepWaitingApproval: {StepApproved, StepRetrying, StepFailed},
	StepApproved:        {},
	StepRetrying:        {StepRunning},
	StepFailed:          {},
	StepSkipped:         {},
}

// ValidateStepStatus checks if a StepStatus value is valid.
func ValidateStepStatus(s StepStatus) error {
	if !validStepStatuses[s] {
		return fmt.Errorf("%w: %q", ErrInvalidStepStatus, s)
	}
	return nil
}

// CanTransitionTo checks if the step status can transition to the target status.
func (s StepStatus) CanTransitionTo(target StepStatus) bool {
	return slices.Contains(stepTransitions[s], target)
}

// TransitionTo attempts to transition to the target status, returning an error if invalid.
func (s StepStatus) TransitionTo(target StepStatus) (StepStatus, error) {
	if !s.CanTransitionTo(target) {
		return s, fmt.Errorf("%w: %q -> %q", ErrInvalidTransition, s, target)
	}
	return target, nil
}

// ValidationStatus represents the result of structured output validation.
type ValidationStatus string

const (
	ValidationNotApplicable ValidationStatus = "not_applicable"
	ValidationPending       ValidationStatus = "pending"
	ValidationPassed        ValidationStatus = "passed"
	ValidationCorrected     ValidationStatus = "corrected"
	ValidationFailed        ValidationStatus = "failed"
)

var validValidationStatuses = map[ValidationStatus]bool{
	ValidationNotApplicable: true,
	ValidationPending:       true,
	ValidationPassed:        true,
	ValidationCorrected:     true,
	ValidationFailed:        true,
}

// ValidateValidationStatus checks if a ValidationStatus value is valid.
func ValidateValidationStatus(s ValidationStatus) error {
	if !validValidationStatuses[s] {
		return fmt.Errorf("invalid validation status: %q", s)
	}
	return nil
}

// WorkflowName is a validated workflow name.
type WorkflowName struct {
	value string
}

// NewWorkflowName creates a validated WorkflowName.
func NewWorkflowName(name string) (WorkflowName, error) {
	if name == "" {
		return WorkflowName{}, ErrEmptyWorkflowName
	}
	return WorkflowName{value: name}, nil
}

// String returns the workflow name value.
func (w WorkflowName) String() string { return w.value }

// StepName is a validated step name.
type StepName struct {
	value string
}

// NewStepName creates a validated StepName.
func NewStepName(name string) (StepName, error) {
	if name == "" {
		return StepName{}, ErrEmptyStepName
	}
	return StepName{value: name}, nil
}

// String returns the step name value.
func (s StepName) String() string { return s.value }

// ProviderName is a validated provider name.
type ProviderName struct {
	value string
}

// NewProviderName creates a validated ProviderName.
func NewProviderName(name string) (ProviderName, error) {
	if name == "" {
		return ProviderName{}, ErrEmptyProviderName
	}
	return ProviderName{value: name}, nil
}

// String returns the provider name value.
func (p ProviderName) String() string { return p.value }
