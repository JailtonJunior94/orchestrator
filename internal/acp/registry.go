package acp

import (
	"fmt"
	"os/exec"
	"strings"
)

// BinaryLookup resolves whether a binary exists on the system PATH.
// It is an interface so tests can substitute a fake without calling exec.LookPath.
type BinaryLookup interface {
	LookPath(file string) (string, error)
}

// execLookup is the production implementation that delegates to exec.LookPath.
type execLookup struct{}

func (execLookup) LookPath(file string) (string, error) { return exec.LookPath(file) }

// Registry holds AgentSpec entries keyed by provider name and resolves
// the best available binary for each spec.
type Registry struct {
	specs  map[string]AgentSpec
	lookup BinaryLookup
}

// NewRegistry returns a Registry pre-populated with the four canonical ACP providers.
// The optional lookup argument is used in tests; pass nil to use exec.LookPath.
func NewRegistry(lookup BinaryLookup) *Registry {
	if lookup == nil {
		lookup = execLookup{}
	}
	r := &Registry{
		specs:  make(map[string]AgentSpec),
		lookup: lookup,
	}
	for _, spec := range defaultSpecs() {
		r.specs[spec.Name] = spec
	}
	return r
}

// defaultSpecs returns the four built-in ACP provider specs.
func defaultSpecs() []AgentSpec {
	return []AgentSpec{
		{
			Name:         "claude",
			Binary:       "claude-agent-acp",
			FallbackCmd:  "npx",
			FallbackArgs: []string{"--yes", "@agentclientprotocol/claude-agent-acp"},
			DefaultModel: "opus",
		},
		{
			Name:         "codex",
			Binary:       "codex-acp",
			FallbackCmd:  "npx",
			FallbackArgs: []string{"--yes", "@zed-industries/codex-acp"},
			DefaultModel: "gpt-5.4",
		},
		{
			Name:         "gemini",
			Binary:       "gemini",
			FallbackCmd:  "npx",
			FallbackArgs: []string{"--yes", "@google/gemini-cli", "--acp"},
			DefaultModel: "gemini-2.5-pro",
			ExtraArgs:    []string{"--acp"},
		},
		{
			Name:         "copilot",
			Binary:       "copilot",
			FallbackCmd:  "npx",
			FallbackArgs: []string{"--yes", "@github/copilot", "--acp"},
			DefaultModel: "claude-sonnet-4.6",
			ExtraArgs:    []string{"--acp"},
		},
	}
}

// Get returns the AgentSpec for the given provider name.
// Returns ErrAgentNotAvailable when the name is not registered.
func (r *Registry) Get(name string) (AgentSpec, error) {
	spec, ok := r.specs[name]
	if !ok {
		return AgentSpec{}, fmt.Errorf("%w: %q is not a registered provider", ErrAgentNotAvailable, name)
	}
	return spec, nil
}

// List returns all registered AgentSpec entries in an unspecified order.
func (r *Registry) List() []AgentSpec {
	out := make([]AgentSpec, 0, len(r.specs))
	for _, s := range r.specs {
		out = append(out, s)
	}
	return out
}

// Available reports whether at least one binary (native or fallback) is
// resolvable for the named provider.
func (r *Registry) Available(name string) bool {
	_, _, err := r.Resolve(name)
	return err == nil
}

// ResolvedBinary holds the resolved command and arguments for an agent.
type ResolvedBinary struct {
	Cmd  string
	Args []string
}

// Resolve returns the command and initial arguments needed to launch the named
// provider. It prefers the native binary; falls back to npx when the native
// binary is not found in PATH. Returns ErrAgentNotAvailable when neither is
// available.
func (r *Registry) Resolve(name string) (string, []string, error) {
	spec, err := r.Get(name)
	if err != nil {
		return "", nil, err
	}

	// Try native binary first.
	if spec.Binary != "" {
		if _, lookErr := r.lookup.LookPath(spec.Binary); lookErr == nil {
			args := append([]string{}, spec.ExtraArgs...)
			return spec.Binary, args, nil
		}
	}

	// Fall back to FallbackCmd (e.g. npx).
	if spec.FallbackCmd != "" {
		if _, lookErr := r.lookup.LookPath(spec.FallbackCmd); lookErr == nil {
			args := append([]string{}, spec.FallbackArgs...)
			return spec.FallbackCmd, args, nil
		}
	}

	return "", nil, fmt.Errorf(
		"%w: %q — install with: %s %s",
		ErrAgentNotAvailable,
		spec.Name,
		spec.FallbackCmd,
		strings.Join(spec.FallbackArgs, " "),
	)
}
