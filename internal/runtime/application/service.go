package application

import (
	"context"
	"strings"

	"github.com/jailtonjunior/orchestrator/internal/runtime"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

// WorkflowCatalog provides access to available workflow names and their steps.
type WorkflowCatalog interface {
	List() ([]string, error)
	Load(ctx context.Context, name string) (*workflows.WorkflowDefinition, error)
}

// WorkflowInput is the application-layer view of a workflow input field.
type WorkflowInput struct {
	Name        string
	Label       string
	Description string
	Type        string
	Placeholder string
	Required    bool
	Options     []string
}

// WorkflowSummary is a lightweight description of a workflow for display in the list TUI.
type WorkflowSummary struct {
	Name          string
	Summary       string
	Description   string
	Inputs        []WorkflowInput
	RequiresInput bool
	StepNames     []string
	Providers     []string
}

// Service exposes the runtime use cases consumed by the CLI layer.
type Service interface {
	Run(ctx context.Context, workflowName string, input string) (*runtime.RunResult, error)
	Continue(ctx context.Context, runID string) (*runtime.RunResult, error)
	ListWorkflows(ctx context.Context) ([]string, error)
	// ListWorkflowDetails returns workflow summaries (name + steps + providers) for the list TUI.
	ListWorkflowDetails(ctx context.Context) ([]WorkflowSummary, error)
}

type runtimeService struct {
	engine  runtime.Engine
	catalog WorkflowCatalog
}

// NewService creates the runtime application service.
func NewService(engine runtime.Engine, catalog WorkflowCatalog) Service {
	return &runtimeService{
		engine:  engine,
		catalog: catalog,
	}
}

func (s *runtimeService) Run(ctx context.Context, workflowName string, input string) (*runtime.RunResult, error) {
	return s.engine.Run(ctx, workflowName, input)
}

func (s *runtimeService) Continue(ctx context.Context, runID string) (*runtime.RunResult, error) {
	return s.engine.Continue(ctx, runID)
}

func (s *runtimeService) ListWorkflows(_ context.Context) ([]string, error) {
	return s.catalog.List()
}

func (s *runtimeService) ListWorkflowDetails(ctx context.Context) ([]WorkflowSummary, error) {
	names, err := s.catalog.List()
	if err != nil {
		return nil, err
	}
	summaries := make([]WorkflowSummary, 0, len(names))
	for _, name := range names {
		def, loadErr := s.catalog.Load(ctx, name)
		if loadErr != nil {
			// Skip workflows that fail to load rather than aborting the whole list.
			continue
		}
		summaries = append(summaries, WorkflowSummary{
			Name:          def.Name,
			Summary:       def.Summary,
			Description:   def.Description,
			Inputs:        mapWorkflowInputs(def.Inputs),
			RequiresInput: workflowRequiresInput(def),
			StepNames:     workflowStepNames(def.Steps),
			Providers:     workflowProviders(def.Steps),
		})
	}
	return summaries, nil
}

func mapWorkflowInputs(inputs []workflows.WorkflowInputDefinition) []WorkflowInput {
	mapped := make([]WorkflowInput, 0, len(inputs))
	for _, input := range inputs {
		mapped = append(mapped, WorkflowInput{
			Name:        input.Name,
			Label:       input.Label,
			Description: input.Description,
			Type:        input.Type,
			Placeholder: input.Placeholder,
			Required:    input.Required,
			Options:     append([]string(nil), input.Options...),
		})
	}

	return mapped
}

func workflowStepNames(steps []workflows.StepDefinition) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, step.Name)
	}

	return names
}

func workflowProviders(steps []workflows.StepDefinition) []string {
	providers := make([]string, 0, len(steps))
	for _, step := range steps {
		providers = append(providers, step.Provider)
	}

	return providers
}

func workflowRequiresInput(def *workflows.WorkflowDefinition) bool {
	if len(def.Inputs) > 0 {
		return true
	}

	for _, step := range def.Steps {
		if strings.Contains(step.Input, "{{input}}") {
			return true
		}
	}

	return false
}
