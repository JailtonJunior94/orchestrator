package providers

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	// GeminiProviderName identifies the Gemini CLI provider.
	GeminiProviderName = "gemini"
	// CodexProviderName identifies the Codex CLI provider.
	CodexProviderName = "codex"
)

// Provider executes prompts through an external AI CLI.
type Provider interface {
	Name() string
	Execute(ctx context.Context, input ProviderInput) (ProviderOutput, error)
	// ExecuteStream executes the provider and calls onChunk for each stdout fragment.
	// If onChunk is nil, behaviour is equivalent to Execute.
	// Returns the complete ProviderOutput after the subprocess exits.
	ExecuteStream(ctx context.Context, input ProviderInput, onChunk func([]byte)) (ProviderOutput, error)
	Available() error
}

// ProviderInput contains the prompt and optional timeout for an execution.
type ProviderInput struct {
	Prompt  string
	Timeout time.Duration
	// Options holds provider-specific metadata (e.g. sandbox, output_format).
	// Providers that do not use Options safely ignore it; nil map lookups return "".
	Options map[string]string
}

// ProviderOutput contains the raw provider result.
type ProviderOutput struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Profile  string
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

	profile, err := p.selectedProfile(execCtx)
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
		Profile:  profile.name,
	}
	if err != nil {
		return output, fmt.Errorf("executing provider %q: %w", p.name, err)
	}

	return output, nil
}

// ExecuteStream runs the provider and delivers stdout chunks incrementally via
// onChunk. It collects the full output and returns it as ProviderOutput when
// the subprocess exits. If onChunk is nil the behaviour matches Execute.
func (p cliProvider) ExecuteStream(ctx context.Context, input ProviderInput, onChunk func([]byte)) (ProviderOutput, error) {
	if onChunk == nil {
		return p.Execute(ctx, input)
	}

	return p.executeStreamWithInvocation(ctx, input, onChunk, func(profile invocationProfile, prompt string) invocation {
		return profile.build(prompt)
	})
}

func (p cliProvider) executeStreamWithInvocation(
	ctx context.Context,
	input ProviderInput,
	onChunk func([]byte),
	build func(profile invocationProfile, prompt string) invocation,
) (ProviderOutput, error) {
	if onChunk == nil {
		return p.Execute(ctx, input)
	}

	timeout := input.Timeout
	if timeout <= 0 {
		timeout = p.defaultTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	profile, err := p.selectedProfile(execCtx)
	if err != nil {
		return ProviderOutput{}, err
	}

	call := build(profile, input.Prompt)
	stream, err := p.runner.RunStreaming(execCtx, p.binary, call.args, call.stdin)
	if err != nil {
		return ProviderOutput{}, fmt.Errorf("executing provider %q: %w", p.name, err)
	}

	var stdout bytes.Buffer
	buf := make([]byte, 4096)
	for {
		n, readErr := stream.Stdout.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			stdout.Write(chunk)
			onChunk(chunk)
		}
		if readErr != nil {
			if readErr != io.EOF {
				err = readErr
			}
			break
		}
	}

	var stderr bytes.Buffer
	_, _ = io.Copy(&stderr, stream.Stderr)

	waitErr := stream.Wait()
	output := ProviderOutput{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: stream.ExitCode(),
		Duration: time.Since(stream.StartedAt),
		Profile:  profile.name,
	}

	if err != nil {
		return output, fmt.Errorf("executing provider %q: %w", p.name, err)
	}
	if waitErr != nil {
		return output, fmt.Errorf("executing provider %q: %w", p.name, waitErr)
	}

	return output, nil
}

func (p cliProvider) Available() error {
	if _, err := lookPath(p.binary); err != nil {
		return fmt.Errorf("provider %q binary %q not found in PATH: %w", p.name, p.binary, err)
	}
	if _, err := p.selectedProfile(context.Background()); err != nil {
		return err
	}
	return nil
}

func (p cliProvider) ResolveProfile(ctx context.Context) (string, error) {
	profile, err := p.selectedProfile(ctx)
	if err != nil {
		return "", err
	}

	return profile.name, nil
}

func (p cliProvider) selectedProfile(ctx context.Context) (invocationProfile, error) {
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

func firstMatchingProfile(helpText string, profiles []invocationProfile) (invocationProfile, bool) {
	for _, profile := range profiles {
		if containsAllTokens(helpText, profile.probeTokens) {
			return profile, true
		}
	}

	return invocationProfile{}, false
}

func supportedTokenSets(profiles []invocationProfile) []string {
	sets := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		sets = append(sets, strings.Join(profile.probeTokens, " "))
	}

	return sets
}

func probeHelpText(ctx context.Context, runner platform.CommandRunner, binary string) (string, error) {
	result, err := runner.Run(ctx, binary, []string{"--help"}, "")
	if err != nil {
		return "", err
	}

	return strings.ToLower(result.Stdout + "\n" + result.Stderr), nil
}

func containsAllTokens(helpText string, tokens []string) bool {
	for _, token := range tokens {
		if !strings.Contains(helpText, strings.ToLower(token)) {
			return false
		}
	}

	return true
}
