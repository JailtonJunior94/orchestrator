package domain

import "fmt"

// StepDefinition holds the static definition of a workflow step.
type StepDefinition struct {
	Name     StepName
	Provider ProviderName
	Input    string
}

// StepExecution represents the runtime state of a step.
type StepExecution struct {
	name      StepName
	provider  ProviderName
	status    StepStatus
	input     string
	result    StepResult
	attempts  int
	errMsg    string
	sessionID string // ACP session identifier for resume
}

// StepResult captures the persisted and in-memory outputs of a step.
type StepResult struct {
	Output              string
	RawOutputRef        string
	ApprovedMarkdownRef string
	StructuredJSONRef   string
	ValidationReportRef string
	SchemaName          string
	SchemaVersion       string
	ValidationStatus    ValidationStatus
	EditedByHuman       bool
}

// NewStepExecution creates a StepExecution from a StepDefinition.
func NewStepExecution(def StepDefinition) *StepExecution {
	return &StepExecution{
		name:     def.Name,
		provider: def.Provider,
		status:   StepPending,
		input:    def.Input,
		attempts: 0,
	}
}

// Name returns the step name.
func (s *StepExecution) Name() StepName { return s.name }

// Provider returns the provider name.
func (s *StepExecution) Provider() ProviderName { return s.provider }

// Status returns the current step status.
func (s *StepExecution) Status() StepStatus { return s.status }

// Input returns the step input.
func (s *StepExecution) Input() string { return s.input }

// Output returns the step output.
func (s *StepExecution) Output() string { return s.result.Output }

// Attempts returns the number of execution attempts.
func (s *StepExecution) Attempts() int { return s.attempts }

// Error returns the error message, if any.
func (s *StepExecution) Error() string { return s.errMsg }

// Result returns a copy of the step result metadata.
func (s *StepExecution) Result() StepResult { return s.result }

// Start transitions the step to running and increments the attempt counter.
func (s *StepExecution) Start() error {
	newStatus, err := s.status.TransitionTo(StepRunning)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.name.String(), err)
	}
	s.status = newStatus
	s.attempts++
	return nil
}

// MarkWaitingApproval transitions the step to waiting_approval with output.
func (s *StepExecution) MarkWaitingApproval(result StepResult) error {
	newStatus, err := s.status.TransitionTo(StepWaitingApproval)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.name.String(), err)
	}
	s.status = newStatus
	s.result = result
	return nil
}

// Approve transitions the step to approved.
func (s *StepExecution) Approve() error {
	newStatus, err := s.status.TransitionTo(StepApproved)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.name.String(), err)
	}
	s.status = newStatus
	return nil
}

// Retry transitions the step to retrying.
func (s *StepExecution) Retry() error {
	newStatus, err := s.status.TransitionTo(StepRetrying)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.name.String(), err)
	}
	s.status = newStatus
	return nil
}

// MarkFailed transitions the step to failed with an error message.
func (s *StepExecution) MarkFailed(errMsg string) error {
	newStatus, err := s.status.TransitionTo(StepFailed)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.name.String(), err)
	}
	s.status = newStatus
	s.errMsg = errMsg
	return nil
}

// Skip transitions the step to skipped.
func (s *StepExecution) Skip() error {
	newStatus, err := s.status.TransitionTo(StepSkipped)
	if err != nil {
		return fmt.Errorf("step %q: %w", s.name.String(), err)
	}
	s.status = newStatus
	return nil
}

// SessionID returns the ACP session identifier for resume.
func (s *StepExecution) SessionID() string { return s.sessionID }

// SetSessionID updates the ACP session identifier.
func (s *StepExecution) SetSessionID(id string) { s.sessionID = id }

// SetInput updates the step input (used for edit/redo).
func (s *StepExecution) SetInput(input string) {
	s.input = input
}

// SetOutput updates the step output while preserving the current status.
func (s *StepExecution) SetOutput(output string) {
	s.result.Output = output
	s.result.EditedByHuman = true
}

// UpdateResultMetadata replaces step result metadata while preserving the current output.
func (s *StepExecution) UpdateResultMetadata(result StepResult) {
	s.result = result
}

// StepSnapshot is the serializable representation of a step execution.
type StepSnapshot struct {
	Name      string
	Provider  string
	Status    StepStatus
	Input     string
	Result    StepResult
	Attempts  int
	Error     string
	SessionID string
}

func (s *StepExecution) snapshot() StepSnapshot {
	return StepSnapshot{
		Name:      s.name.String(),
		Provider:  s.provider.String(),
		Status:    s.status,
		Input:     s.input,
		Result:    s.result,
		Attempts:  s.attempts,
		Error:     s.errMsg,
		SessionID: s.sessionID,
	}
}

func newStepExecutionFromSnapshot(snapshot StepSnapshot) (*StepExecution, error) {
	name, err := NewStepName(snapshot.Name)
	if err != nil {
		return nil, err
	}

	provider, err := NewProviderName(snapshot.Provider)
	if err != nil {
		return nil, err
	}

	if err := ValidateStepStatus(snapshot.Status); err != nil {
		return nil, err
	}
	if snapshot.Result.ValidationStatus != "" {
		if err := ValidateValidationStatus(snapshot.Result.ValidationStatus); err != nil {
			return nil, err
		}
	}

	return &StepExecution{
		name:      name,
		provider:  provider,
		status:    snapshot.Status,
		input:     snapshot.Input,
		result:    snapshot.Result,
		attempts:  snapshot.Attempts,
		errMsg:    snapshot.Error,
		sessionID: snapshot.SessionID,
	}, nil
}
