package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

const (
	// ClaudeProviderName identifies the Claude CLI provider.
	ClaudeProviderName = "claude"
	// CopilotProviderName identifies the Copilot CLI provider.
	CopilotProviderName = "copilot"
)

// Provider executes prompts through an external AI CLI.
type Provider interface {
	Name() string
	Execute(ctx context.Context, input ProviderInput) (ProviderOutput, error)
	Available() error
}

// ProviderInput contains the prompt and optional timeout for an execution.
type ProviderInput struct {
	Prompt  string
	Timeout time.Duration
}

// ProviderOutput contains the raw provider result.
type ProviderOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Factory resolves providers by their workflow names.
type Factory interface {
	Get(name string) (Provider, error)
}

var lookPath = exec.LookPath

type cliProvider struct {
	name           string
	binary         string
	defaultTimeout time.Duration
	profiles       []invocationProfile
	runner         platform.CommandRunner
}

type invocationProfile struct {
	name        string
	probeTokens []string
	build       func(prompt string) invocation
}

type invocation struct {
	args  []string
	stdin string
}

func (p cliProvider) Name() string {
	return p.name
}

func (p cliProvider) Execute(ctx context.Context, input ProviderInput) (ProviderOutput, error) {
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = p.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	profile, err := p.resolveProfile(execCtx)
	if err != nil {
		return ProviderOutput{}, err
	}

	call := profile.build(input.Prompt)
	result, err := p.runner.Run(execCtx, p.binary, call.args, call.stdin)
	output := ProviderOutput{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		Duration: result.Duration,
	}
	if err != nil {
		return output, fmt.Errorf("executing provider %q: %w", p.name, err)
	}

	return output, nil
}

func (p cliProvider) Available() error {
	if _, err := lookPath(p.binary); err != nil {
		return fmt.Errorf("provider %q binary %q not found in PATH: %w", p.name, p.binary, err)
	}
	if _, err := p.resolveProfile(context.Background()); err != nil {
		return err
	}
	return nil
}

func (p cliProvider) resolveProfile(ctx context.Context) (invocationProfile, error) {
	if len(p.profiles) == 0 {
		return invocationProfile{}, fmt.Errorf("provider %q has no invocation profiles configured", p.name)
	}

	result, err := p.runner.Run(ctx, p.binary, []string{"--help"}, "")
	if err != nil {
		return p.profiles[0], nil
	}

	helpText := strings.ToLower(result.Stdout + "\n" + result.Stderr)
	for _, profile := range p.profiles {
		matched := true
		for _, token := range profile.probeTokens {
			if !strings.Contains(helpText, strings.ToLower(token)) {
				matched = false
				break
			}
		}
		if matched {
			return profile, nil
		}
	}

	return p.profiles[0], nil
}
