package workflows

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Parser parses raw bytes into a WorkflowDefinition.
type Parser interface {
	Parse(ctx context.Context, data []byte) (*WorkflowDefinition, error)
}

type yamlParser struct{}

// NewParser creates a YAML-based Parser.
func NewParser() Parser {
	return &yamlParser{}
}

func (p *yamlParser) Parse(_ context.Context, data []byte) (*WorkflowDefinition, error) {
	var wf WorkflowDefinition
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("parsing workflow yaml: %w", err)
	}
	return &wf, nil
}
