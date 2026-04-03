package application

import (
	"context"

	"github.com/jailtonjunior/orchestrator/internal/runtime"
)

// WorkflowCatalog lists the workflows available to the runtime application.
type WorkflowCatalog interface {
	List() ([]string, error)
}

// Service exposes the runtime use cases consumed by the CLI layer.
type Service interface {
	Run(ctx context.Context, workflowName string, input string) (*runtime.RunResult, error)
	Continue(ctx context.Context, runID string) (*runtime.RunResult, error)
	ListWorkflows(ctx context.Context) ([]string, error)
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
