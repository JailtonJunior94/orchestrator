package workflows

// WorkflowDefinition represents a workflow loaded from YAML.
type WorkflowDefinition struct {
	Name  string           `yaml:"name"`
	Steps []StepDefinition `yaml:"steps"`
}

// StepDefinition represents a single step in a workflow.
type StepDefinition struct {
	Name         string               `yaml:"name"`
	Provider     string               `yaml:"provider"`
	Input        string               `yaml:"input"`
	Timeout      string               `yaml:"timeout,omitempty"`
	Schema       string               `yaml:"schema,omitempty"`
	Output       StepOutputDefinition `yaml:"output,omitempty"`
	Capabilities map[string]string    `yaml:"capabilities,omitempty"`
}

// StepOutputDefinition configures the output contract of a workflow step.
type StepOutputDefinition struct {
	Markdown   string `yaml:"markdown,omitempty"`
	JSONSchema string `yaml:"json_schema,omitempty"`
}

// RequiresStructuredOutput reports whether the step must produce structured JSON.
func (s StepDefinition) RequiresStructuredOutput() bool {
	return s.Schema != "" || s.Output.JSONSchema != ""
}
