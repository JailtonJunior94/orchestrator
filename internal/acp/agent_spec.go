package acp

import "errors"

// AgentSpec describes an ACP-compatible agent provider declaratively.
// It contains enough information for the registry and factory to spawn and
// connect to the agent without any provider-specific logic in the engine.
type AgentSpec struct {
	// Name is the unique identifier for this provider (e.g. "claude", "codex").
	Name string

	// Binary is the preferred executable name looked up in PATH (e.g. "claude-agent-acp").
	Binary string

	// FallbackCmd is an alternative launcher when Binary is not found (e.g. "npx").
	FallbackCmd string

	// FallbackArgs are the arguments passed to FallbackCmd (e.g. ["--yes", "@agentclientprotocol/claude-agent-acp"]).
	FallbackArgs []string

	// DefaultModel is the model identifier used when the workflow does not specify one.
	DefaultModel string

	// SupportsAddDirs indicates whether the agent supports the add_dirs capability.
	SupportsAddDirs bool

	// FullAccessModeID is the identifier for the agent's full-access permission mode, if any.
	FullAccessModeID string

	// ExtraArgs are additional command-line arguments appended when spawning the agent.
	ExtraArgs []string

	// ExtraEnv are additional environment variables injected into the agent process.
	ExtraEnv []string
}

// Validate reports an error when the spec cannot be used to spawn an agent.
// A spec is valid when it has a Name and at least one of Binary or FallbackCmd.
func (s AgentSpec) Validate() error {
	if s.Name == "" {
		return errors.New("agent spec: name must not be empty")
	}
	if s.Binary == "" && s.FallbackCmd == "" {
		return errors.New("agent spec: at least one of Binary or FallbackCmd must be set")
	}
	return nil
}
