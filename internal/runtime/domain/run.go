package domain

import (
	"fmt"
	"time"
)

// Run is the aggregate root for workflow execution.
type Run struct {
	id        string
	workflow  WorkflowName
	input     string
	status    RunStatus
	steps     []*StepExecution
	createdAt time.Time
	updatedAt time.Time
}

// NewRun creates a Run with validated invariants.
func NewRun(id string, workflow WorkflowName, input string, stepDefs []StepDefinition, now time.Time) (*Run, error) {
	if id == "" {
		return nil, ErrEmptyRunID
	}
	if input == "" {
		return nil, ErrEmptyInput
	}
	if len(stepDefs) == 0 {
		return nil, ErrNoSteps
	}

	seen := make(map[string]bool, len(stepDefs))
	steps := make([]*StepExecution, 0, len(stepDefs))
	for _, def := range stepDefs {
		name := def.Name.String()
		if seen[name] {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateStepName, name)
		}
		seen[name] = true
		steps = append(steps, NewStepExecution(def))
	}

	return &Run{
		id:        id,
		workflow:  workflow,
		input:     input,
		status:    RunPending,
		steps:     steps,
		createdAt: now,
		updatedAt: now,
	}, nil
}

// ID returns the run identifier.
func (r *Run) ID() string { return r.id }

// Workflow returns the workflow name.
func (r *Run) Workflow() WorkflowName { return r.workflow }

// Input returns the run input.
func (r *Run) Input() string { return r.input }

// Status returns the current run status.
func (r *Run) Status() RunStatus { return r.status }

// Steps returns a copy of the step executions.
func (r *Run) Steps() []*StepExecution {
	cp := make([]*StepExecution, len(r.steps))
	copy(cp, r.steps)
	return cp
}

// CreatedAt returns the creation time.
func (r *Run) CreatedAt() time.Time { return r.createdAt }

// UpdatedAt returns the last update time.
func (r *Run) UpdatedAt() time.Time { return r.updatedAt }

// Start transitions the run from pending to running.
func (r *Run) Start(now time.Time) error {
	newStatus, err := r.status.TransitionTo(RunRunning)
	if err != nil {
		return err
	}
	r.status = newStatus
	r.updatedAt = now
	return nil
}

// Pause transitions the run from running to paused.
func (r *Run) Pause(now time.Time) error {
	newStatus, err := r.status.TransitionTo(RunPaused)
	if err != nil {
		return err
	}
	r.status = newStatus
	r.updatedAt = now
	return nil
}

// Resume transitions the run from paused to running.
func (r *Run) Resume(now time.Time) error {
	newStatus, err := r.status.TransitionTo(RunRunning)
	if err != nil {
		return err
	}
	r.status = newStatus
	r.updatedAt = now
	return nil
}

// Cancel transitions the run to cancelled.
func (r *Run) Cancel(now time.Time) error {
	newStatus, err := r.status.TransitionTo(RunCancelled)
	if err != nil {
		return err
	}
	r.status = newStatus
	r.updatedAt = now
	return nil
}

// CurrentStep returns the first step that is not in a terminal state (approved, failed, skipped).
func (r *Run) CurrentStep() (*StepExecution, error) {
	for _, s := range r.steps {
		switch s.Status() {
		case StepApproved, StepFailed, StepSkipped:
			continue
		default:
			return s, nil
		}
	}
	return nil, ErrNoCurrentStep
}

func (r *Run) findStep(name StepName) (*StepExecution, error) {
	for _, s := range r.steps {
		if s.Name().String() == name.String() {
			return s, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrStepNotFound, name.String())
}

// ApproveStep approves a step that is waiting for approval.
func (r *Run) ApproveStep(name StepName, now time.Time) error {
	if r.status != RunRunning {
		return ErrRunNotRunning
	}
	step, err := r.findStep(name)
	if err != nil {
		return err
	}
	if err := step.Approve(); err != nil {
		return err
	}
	r.updatedAt = now

	// If all steps are done, complete the run.
	if r.allStepsTerminal() {
		r.status = RunCompleted
	}
	return nil
}

// RetryStep marks a step for retry.
func (r *Run) RetryStep(name StepName, now time.Time) error {
	if r.status != RunRunning {
		return ErrRunNotRunning
	}
	step, err := r.findStep(name)
	if err != nil {
		return err
	}
	if err := step.Retry(); err != nil {
		return err
	}
	r.updatedAt = now
	return nil
}

// MarkStepFailed marks a step as failed.
func (r *Run) MarkStepFailed(name StepName, errMsg string, now time.Time) error {
	if r.status != RunRunning {
		return ErrRunNotRunning
	}
	step, err := r.findStep(name)
	if err != nil {
		return err
	}
	if err := step.MarkFailed(errMsg); err != nil {
		return err
	}
	r.status = RunFailed
	r.updatedAt = now
	return nil
}

// MarkStepCompleted transitions a running step to waiting_approval with output.
func (r *Run) MarkStepCompleted(name StepName, result StepResult, now time.Time) error {
	if r.status != RunRunning {
		return ErrRunNotRunning
	}
	step, err := r.findStep(name)
	if err != nil {
		return err
	}
	if err := step.MarkWaitingApproval(result); err != nil {
		return err
	}
	r.updatedAt = now
	return nil
}

// StartStep transitions a step from pending (or retrying) to running.
func (r *Run) StartStep(name StepName, now time.Time) error {
	if r.status != RunRunning {
		return ErrRunNotRunning
	}
	step, err := r.findStep(name)
	if err != nil {
		return err
	}
	if err := step.Start(); err != nil {
		return err
	}
	r.updatedAt = now
	return nil
}

func (r *Run) allStepsTerminal() bool {
	for _, s := range r.steps {
		switch s.Status() {
		case StepApproved, StepFailed, StepSkipped:
			continue
		default:
			return false
		}
	}
	return true
}

// RunSnapshot is the serializable representation of a run aggregate.
type RunSnapshot struct {
	ID        string
	Workflow  string
	Input     string
	Status    RunStatus
	Steps     []StepSnapshot
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Snapshot returns a serializable representation of the run.
func (r *Run) Snapshot() RunSnapshot {
	steps := make([]StepSnapshot, 0, len(r.steps))
	for _, step := range r.steps {
		steps = append(steps, step.snapshot())
	}

	return RunSnapshot{
		ID:        r.id,
		Workflow:  r.workflow.String(),
		Input:     r.input,
		Status:    r.status,
		Steps:     steps,
		CreatedAt: r.createdAt,
		UpdatedAt: r.updatedAt,
	}
}

// NewRunFromSnapshot restores a run aggregate from persisted state.
func NewRunFromSnapshot(snapshot RunSnapshot) (*Run, error) {
	if snapshot.ID == "" {
		return nil, ErrEmptyRunID
	}

	workflow, err := NewWorkflowName(snapshot.Workflow)
	if err != nil {
		return nil, err
	}

	if err := ValidateRunStatus(snapshot.Status); err != nil {
		return nil, err
	}

	if len(snapshot.Steps) == 0 {
		return nil, ErrNoSteps
	}

	steps := make([]*StepExecution, 0, len(snapshot.Steps))
	for _, stepSnapshot := range snapshot.Steps {
		step, err := newStepExecutionFromSnapshot(stepSnapshot)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}

	return &Run{
		id:        snapshot.ID,
		workflow:  workflow,
		input:     snapshot.Input,
		status:    snapshot.Status,
		steps:     steps,
		createdAt: snapshot.CreatedAt,
		updatedAt: snapshot.UpdatedAt,
	}, nil
}

// UpdateStepOutput updates the output of an existing step.
func (r *Run) UpdateStepOutput(name StepName, output string, now time.Time) error {
	step, err := r.findStep(name)
	if err != nil {
		return err
	}

	step.SetOutput(output)
	r.updatedAt = now
	return nil
}

// UpdateStepResult updates the result metadata of an existing step.
func (r *Run) UpdateStepResult(name StepName, result StepResult, now time.Time) error {
	step, err := r.findStep(name)
	if err != nil {
		return err
	}

	step.UpdateResultMetadata(result)
	r.updatedAt = now
	return nil
}
