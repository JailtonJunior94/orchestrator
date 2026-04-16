package providers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

const (
	codexInstallHint    = "Install: https://github.com/openai/codex"
	codexDefaultSandbox = "read-only"
)

// NewCodexProvider creates the Codex CLI adapter.
func NewCodexProvider(runner platform.CommandRunner) Provider {
	return codexProvider{
		cliProvider: cliProvider{
			name:           CodexProviderName,
			binary:         CodexProviderName,
			defaultTimeout: 5 * time.Minute,
			runner:         runner,
			profiles: []invocationProfile{
				{
					name:        "exec-sandbox-json",
					probeTokens: []string{"exec", "--sandbox", "--json"},
					build: func(prompt string) invocation {
						return invocation{
							args: []string{"exec", "--sandbox", codexDefaultSandbox, prompt},
						}
					},
				},
				{
					name:        "exec-short-sandbox-json",
					probeTokens: []string{"exec", "-s", "--json"},
					build: func(prompt string) invocation {
						return invocation{
							args: []string{"exec", prompt, "-s", codexDefaultSandbox},
						}
					},
				},
			},
		},
	}
}

type codexProvider struct {
	cliProvider
}

// Execute invokes Codex CLI, adding --json when output_format is "jsonl".
func (c codexProvider) Execute(ctx context.Context, input ProviderInput) (ProviderOutput, error) {
	sandbox := input.Options["sandbox"]
	if sandbox == "" {
		sandbox = codexDefaultSandbox
	}

	timeout := input.Timeout
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	profile, err := c.selectedProfile(execCtx)
	if err != nil {
		return ProviderOutput{}, err
	}

	call := buildCodexInvocation(profile, input.Prompt, sandbox)
	if input.Options["output_format"] == "jsonl" {
		call.args = append(call.args, "--json")
	}

	result, err := c.runner.Run(execCtx, c.binary, call.args, call.stdin)
	out := ProviderOutput{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		Duration: result.Duration,
		Profile:  profile.name,
	}
	if err != nil {
		return out, fmt.Errorf("executing provider %q: %w", c.name, err)
	}
	return out, nil
}

// ExecuteStream invokes Codex CLI with the same sandbox and JSON options as
// Execute while still emitting incremental stdout chunks.
func (c codexProvider) ExecuteStream(ctx context.Context, input ProviderInput, onChunk func([]byte)) (ProviderOutput, error) {
	if onChunk == nil {
		return c.Execute(ctx, input)
	}

	sandbox := input.Options["sandbox"]
	if sandbox == "" {
		sandbox = codexDefaultSandbox
	}

	streamingInput := input
	streamingInput.Options = nil
	return c.executeStreamWithInvocation(
		ctx,
		streamingInput,
		onChunk,
		func(profile invocationProfile, prompt string) invocation {
			call := buildCodexInvocation(profile, prompt, sandbox)
			if input.Options["output_format"] == "jsonl" {
				call.args = append(call.args, "--json")
			}
			return call
		},
	)
}

// buildCodexInvocation constructs the invocation with the given sandbox mode.
func buildCodexInvocation(profile invocationProfile, prompt, sandbox string) invocation {
	call := profile.build(prompt)
	// Replace placeholder sandbox in args built by the profile.
	for i, arg := range call.args {
		if arg == codexDefaultSandbox {
			call.args[i] = sandbox
		}
	}
	return call
}

// Available verifies the binary is in PATH and supports at least one
// non-interactive execution profile with sandboxing and JSON output.
func (c codexProvider) Available() error {
	if _, err := c.lookupBinaryPath(c.binary); err != nil {
		return fmt.Errorf("provider %q binary %q not found in PATH — %s: %w",
			c.name, c.binary, codexInstallHint, err)
	}

	helpText, err := probeCodexHelpText(context.Background(), c.runner, c.binary)
	if err != nil {
		return fmt.Errorf("provider %q: cannot probe binary: %w", c.name, err)
	}
	if _, ok := firstMatchingProfile(helpText, c.profiles); !ok {
		return fmt.Errorf(
			"provider %q: incompatible CLI help output, expected one of the supported token sets %q; update Codex CLI — %s",
			c.name,
			supportedTokenSets(c.profiles),
			codexInstallHint,
		)
	}
	return nil
}

func probeCodexHelpText(ctx context.Context, runner platform.CommandRunner, binary string) (string, error) {
	helpText, err := probeHelpText(ctx, runner, binary)
	if err != nil {
		return "", err
	}

	execHelp, err := runner.Run(ctx, binary, []string{"exec", "--help"}, "")
	if err != nil {
		return helpText, nil
	}

	return strings.Join([]string{
		helpText,
		strings.ToLower(execHelp.Stdout + "\n" + execHelp.Stderr),
	}, "\n"), nil
}
