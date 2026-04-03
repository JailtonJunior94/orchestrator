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
