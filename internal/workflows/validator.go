package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// Validator validates a WorkflowDefinition before execution.
type Validator interface {
	Validate(ctx context.Context, workflow *WorkflowDefinition) error
}

type workflowValidator struct {
	knownProviders map[string]bool
}

// NewValidator creates a Validator with the given known providers.
func NewValidator(providers []string) Validator {
	known := make(map[string]bool, len(providers))
	for _, p := range providers {
		known[p] = true
	}
	return &workflowValidator{knownProviders: known}
}

var stepRefPattern = regexp.MustCompile(`\{\{steps\.([^.}]+)\.output\}\}`)

func (v *workflowValidator) Validate(_ context.Context, wf *WorkflowDefinition) error {
	if wf.Name == "" {
		return ErrEmptyWorkflowName
	}
	if len(wf.Steps) == 0 {
		return ErrNoSteps
	}

	seen := make(map[string]bool, len(wf.Steps))
	for i, step := range wf.Steps {
		if step.Name == "" {
			return fmt.Errorf("step %d: %w", i, ErrEmptyStepName)
		}
		if step.Provider == "" {
			return fmt.Errorf("step %q: %w", step.Name, ErrEmptyProvider)
		}
		if step.Input == "" {
			return fmt.Errorf("step %q: %w", step.Name, ErrEmptyInput)
		}
		if step.Timeout != "" {
			if _, err := time.ParseDuration(step.Timeout); err != nil {
				return fmt.Errorf("step %q: %w: %v", step.Name, ErrInvalidTimeout, err)
			}
		}
		if step.Schema != "" && !json.Valid([]byte(step.Schema)) {
			return fmt.Errorf("step %q: %w", step.Name, ErrInvalidSchema)
		}
		if seen[step.Name] {
			return fmt.Errorf("%w: %q", ErrDuplicateStepName, step.Name)
		}
		seen[step.Name] = true

		if !v.knownProviders[step.Provider] {
			return fmt.Errorf("step %q: %w: %q", step.Name, ErrInvalidProvider, step.Provider)
		}

		// Check that step references point to previously defined steps.
		matches := stepRefPattern.FindAllStringSubmatch(step.Input, -1)
		for _, match := range matches {
			ref := match[1]
			if !seen[ref] || ref == step.Name {
				return fmt.Errorf("step %q: %w: %q", step.Name, ErrInvalidStepReference, ref)
			}
		}
	}

	return nil
}
