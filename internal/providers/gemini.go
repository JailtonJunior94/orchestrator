package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

const (
	geminiInstallHint = "Install: https://github.com/google-gemini/gemini-cli"
)

// NewGeminiProvider creates the Gemini CLI adapter.
func NewGeminiProvider(runner platform.CommandRunner) Provider {
	return geminiProvider{
		cliProvider: cliProvider{
			name:           GeminiProviderName,
			binary:         GeminiProviderName,
			defaultTimeout: 5 * time.Minute,
			runner:         runner,
			profiles: []invocationProfile{
				{
					name:        "prompt-output-format",
					probeTokens: []string{"-p", "--output-format"},
					build: func(prompt string) invocation {
						return invocation{
							args: []string{"-p", prompt},
						}
					},
				},
				{
					name:        "prompt-short-output",
					probeTokens: []string{"-p", "-o"},
					build: func(prompt string) invocation {
						return invocation{
							args: []string{"-p", prompt},
						}
					},
				},
			},
		},
	}
}

type geminiProvider struct {
	cliProvider
}

// Execute invokes Gemini CLI, explicitly selecting the output format.
func (g geminiProvider) Execute(ctx context.Context, input ProviderInput) (ProviderOutput, error) {
	format := "text"
	if input.Options["output_format"] == "json" {
		format = "json"
	}

	return g.executeWithFormat(ctx, input, format)
}

// ExecuteStream invokes Gemini CLI with the same output-format semantics as
// Execute while still emitting incremental stdout chunks.
func (g geminiProvider) ExecuteStream(ctx context.Context, input ProviderInput, onChunk func([]byte)) (ProviderOutput, error) {
	if onChunk == nil {
		return g.Execute(ctx, input)
	}

	format := "text"
	if input.Options["output_format"] == "json" {
		format = "json"
	}

	streamingInput := input
	streamingInput.Options = nil
	return g.executeStreamWithInvocation(
		ctx,
		streamingInput,
		onChunk,
		func(profile invocationProfile, prompt string) invocation {
			call := profile.build(prompt)
			switch profile.name {
			case "prompt-short-output":
				call.args = append(call.args, "-o", format)
			default:
				call.args = append(call.args, "--output-format", format)
			}
			return call
		},
	)
}

func (g geminiProvider) executeWithFormat(ctx context.Context, input ProviderInput, format string) (ProviderOutput, error) {
	timeout := input.Timeout
	if timeout <= 0 {
		timeout = g.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	profile, err := g.selectedProfile(execCtx)
	if err != nil {
		return ProviderOutput{}, err
	}

	call := profile.build(input.Prompt)
	switch profile.name {
	case "prompt-short-output":
		call.args = append(call.args, "-o", format)
	default:
		call.args = append(call.args, "--output-format", format)
	}

	result, err := g.runner.Run(execCtx, g.binary, call.args, call.stdin)
	output := ProviderOutput{
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
		Duration: result.Duration,
		Profile:  profile.name,
	}
	if err != nil {
		return output, fmt.Errorf("executing provider %q: %w", g.name, err)
	}
	return output, nil
}

// Available verifies the binary is in PATH and supports at least one
// non-interactive prompt profile with configurable output formatting.
func (g geminiProvider) Available() error {
	if _, err := lookPath(g.binary); err != nil {
		return fmt.Errorf("provider %q binary %q not found in PATH — %s: %w",
			g.name, g.binary, geminiInstallHint, err)
	}

	helpText, err := probeHelpText(context.Background(), g.runner, g.binary)
	if err != nil {
		return fmt.Errorf("provider %q: cannot probe binary: %w", g.name, err)
	}
	if _, ok := firstMatchingProfile(helpText, g.profiles); !ok {
		return fmt.Errorf(
			"provider %q: incompatible CLI help output, expected one of the supported token sets %q; update Gemini CLI — %s",
			g.name,
			supportedTokenSets(g.profiles),
			geminiInstallHint,
		)
	}
	return nil
}
