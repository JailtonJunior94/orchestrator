package workflows

import (
	"context"
	"embed"
	"fmt"
)

//go:embed embedded/*.yaml
var embeddedFS embed.FS

// Catalog provides access to built-in workflows.
type Catalog struct {
	parser Parser
}

// NewCatalog creates a Catalog with the given parser.
func NewCatalog(parser Parser) *Catalog {
	return &Catalog{parser: parser}
}

// Load returns a built-in workflow by name.
func (c *Catalog) Load(ctx context.Context, name string) (*WorkflowDefinition, error) {
	data, err := embeddedFS.ReadFile("embedded/" + name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrWorkflowNotFound, name)
	}
	return c.parser.Parse(ctx, data)
}

// LoadSteps returns the step names and providers for a workflow by name.
// This satisfies the runtimeapp.WorkflowCatalog interface extension without
// requiring workflows to import runtimeapp.
func (c *Catalog) LoadSteps(ctx context.Context, name string) (stepNames []string, providers []string, err error) {
	def, err := c.Load(ctx, name)
	if err != nil {
		return nil, nil, err
	}
	stepNames = make([]string, 0, len(def.Steps))
	providers = make([]string, 0, len(def.Steps))
	for _, s := range def.Steps {
		stepNames = append(stepNames, s.Name)
		providers = append(providers, s.Provider)
	}
	return stepNames, providers, nil
}

// List returns the names of all available built-in workflows.
func (c *Catalog) List() ([]string, error) {
	entries, err := embeddedFS.ReadDir("embedded")
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if len(n) > 5 && n[len(n)-5:] == ".yaml" {
			names = append(names, n[:len(n)-5])
		}
	}
	return names, nil
}
