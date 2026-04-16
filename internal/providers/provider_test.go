package providers

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func withLookup(provider Provider, lookup binaryLookupFunc) Provider {
	switch p := provider.(type) {
	case cliProvider:
		p.lookPathFn = lookup
		return p
	case geminiProvider:
		p.lookPathFn = lookup
		return p
	case codexProvider:
		p.lookPathFn = lookup
		return p
	default:
		return provider
	}
}

func TestFactory(t *testing.T) {
	t.Parallel()

	factory := NewFactory(platform.FakeCommandRunner{})
	for _, name := range []string{ClaudeProviderName, CopilotProviderName, GeminiProviderName, CodexProviderName} {
		if _, err := factory.Get(name); err != nil {
			t.Fatalf("Get(%q): %v", name, err)
		}
	}
	if _, err := factory.Get("unknown"); err == nil {
		t.Fatal("expected unknown provider error")
	}
}

func TestClaudeProviderExecute(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	callCount := 0
	provider := NewClaudeProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, name string, args []string, stdin string) (platform.CommandResult, error) {
			if name != "claude" {
				t.Fatalf("name = %q", name)
			}
			callCount++
			if len(args) == 1 && args[0] == "--help" {
				return platform.CommandResult{
					Stdout: "--output-format -p",
				}, nil
			}
			if stdin != "" {
				t.Fatalf("stdin = %q", stdin)
			}
			capturedArgs = args
			return platform.CommandResult{
				Stdout:   "{}",
				ExitCode: 0,
				Duration: time.Second,
			}, nil
		},
	})

	result, err := provider.Execute(context.Background(), ProviderInput{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "{}" {
		t.Fatalf("stdout = %q", result.Stdout)
	}
	wantArgs := []string{"-p", "hello", "--output-format", "json"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v", capturedArgs)
	}
	for idx := range wantArgs {
		if capturedArgs[idx] != wantArgs[idx] {
			t.Fatalf("args = %v", capturedArgs)
		}
	}
	if callCount != 2 {
		t.Fatalf("call count = %d", callCount)
	}
}

func TestClaudeProviderExecuteUsesConfiguredBinaryOverride(t *testing.T) {
	t.Setenv(claudeBinaryEnvVar, "claude")

	var capturedName string
	provider := NewClaudeProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, name string, args []string, _ string) (platform.CommandResult, error) {
			capturedName = name
			if len(args) == 1 && args[0] == "--help" {
				return platform.CommandResult{Stdout: "--output-format -p"}, nil
			}
			return platform.CommandResult{Stdout: "{}"}, nil
		},
	})

	_, err := provider.Execute(context.Background(), ProviderInput{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if capturedName != "claude" {
		t.Fatalf("binary = %q, want %q", capturedName, "claude")
	}
}

// --- Task 4.0 tests: ExecuteStream ---

func TestExecuteStreamDeliversChunksAndFullOutput(t *testing.T) {
	t.Parallel()

	const fakeOutput = "hello streaming world"
	runner := platform.FakeCommandRunner{
		// Run is called for --help probe during profile selection.
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return platform.CommandResult{Stdout: "--output-format -p"}, nil
			}
			return platform.CommandResult{}, nil
		},
		RunStreamingFunc: func(_ context.Context, _ string, _ []string, _ string) (*platform.StreamResult, error) {
			return platform.NewFakeStreamResult(fakeOutput, ""), nil
		},
	}

	provider := NewClaudeProvider(runner)

	var collected strings.Builder
	result, err := provider.ExecuteStream(context.Background(), ProviderInput{Prompt: "test"}, func(chunk []byte) {
		collected.Write(chunk)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != fakeOutput {
		t.Fatalf("Stdout = %q, want %q", result.Stdout, fakeOutput)
	}
	if collected.String() != fakeOutput {
		t.Fatalf("chunks = %q, want %q", collected.String(), fakeOutput)
	}
}

func TestExecuteStreamNilOnChunkBehavesLikeExecute(t *testing.T) {
	t.Parallel()

	runner := platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return platform.CommandResult{Stdout: "--output-format -p"}, nil
			}
			return platform.CommandResult{Stdout: "result"}, nil
		},
	}

	provider := NewClaudeProvider(runner)
	result, err := provider.ExecuteStream(context.Background(), ProviderInput{Prompt: "test"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "result" {
		t.Fatalf("Stdout = %q, want %q", result.Stdout, "result")
	}
}

func TestExecuteStreamContextCancelInterruptsStreaming(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	runner := platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return platform.CommandResult{Stdout: "--output-format -p"}, nil
			}
			return platform.CommandResult{}, nil
		},
		RunStreamingFunc: func(_ context.Context, _ string, _ []string, _ string) (*platform.StreamResult, error) {
			return nil, context.Canceled
		},
	}

	provider := NewClaudeProvider(runner)
	_, err := provider.ExecuteStream(ctx, ProviderInput{Prompt: "test"}, func(_ []byte) {})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestExecuteIncludesProviderFailureDetails(t *testing.T) {
	t.Parallel()

	provider := NewClaudeProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return platform.CommandResult{Stdout: "--output-format -p"}, nil
			}
			return platform.CommandResult{
				Stdout: `{"type":"result","result":"Failed to authenticate"}`,
			}, errors.New("exit status 1")
		},
	})

	_, err := provider.Execute(context.Background(), ProviderInput{Prompt: "test"})
	if err == nil {
		t.Fatal("expected execution error")
	}
	if !strings.Contains(err.Error(), "Failed to authenticate") {
		t.Fatalf("expected provider output in error, got: %v", err)
	}
}

func TestExecuteStreamDrainsStderrWhileStreamingStdout(t *testing.T) {
	t.Setenv("GO_WANT_PROVIDER_HELPER_PROCESS", "1")

	provider := cliProvider{
		name:           "helper",
		binary:         os.Args[0],
		defaultTimeout: time.Second,
		profiles: []invocationProfile{
			{
				name:        "helper-stream",
				probeTokens: []string{"helper-stream"},
				build: func(prompt string) invocation {
					return invocation{
						args: []string{"-test.run=TestProviderHelperProcess", "--", "stderr-first", prompt},
					}
				},
			},
		},
		runner: providerHelperRunner{},
	}

	var chunks strings.Builder
	result, err := provider.ExecuteStream(context.Background(), ProviderInput{Prompt: "ok"}, func(chunk []byte) {
		chunks.Write(chunk)
	})
	if err != nil {
		t.Fatalf("ExecuteStream error: %v", err)
	}
	if result.Stdout != "stream-complete:ok" {
		t.Fatalf("stdout = %q", result.Stdout)
	}
	if chunks.String() != result.Stdout {
		t.Fatalf("chunks = %q, want %q", chunks.String(), result.Stdout)
	}
	if len(result.Stderr) == 0 {
		t.Fatal("expected stderr to be captured")
	}
}

// Verify all providers implement Provider (including ExecuteStream) at compile time.
var (
	_ Provider = (*geminiProvider)(nil)
	_ Provider = (*codexProvider)(nil)
	_ Provider = cliProvider{}
)

func TestProviderAvailable(t *testing.T) {
	t.Parallel()

	lookup := func(file string) (string, error) {
		if file == "claude" {
			return filepath.Join("/tmp", file), nil
		}
		return "", errors.New("missing")
	}

	if err := withLookup(NewClaudeProvider(platform.FakeCommandRunner{}), lookup).Available(); err != nil {
		t.Fatal(err)
	}
	if err := withLookup(NewCopilotProvider(platform.FakeCommandRunner{}), lookup).Available(); err == nil {
		t.Fatal("expected error")
	}
}

func TestProviderAvailable_lookupIsScopedPerProvider(t *testing.T) {
	t.Parallel()

	claude := withLookup(NewClaudeProvider(platform.FakeCommandRunner{}), func(file string) (string, error) {
		return filepath.Join("/tmp/claude", file), nil
	})
	copilot := withLookup(NewCopilotProvider(platform.FakeCommandRunner{}), func(_ string) (string, error) {
		return "", errors.New("missing")
	})

	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		errs <- claude.Available()
	}()

	go func() {
		defer wg.Done()
		errs <- copilot.Available()
	}()

	wg.Wait()
	close(errs)

	var results []error
	for err := range errs {
		results = append(results, err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0] == nil && results[1] == nil {
		t.Fatal("expected one provider to fail binary lookup")
	}
	if results[0] != nil && results[1] != nil {
		t.Fatalf("expected one provider to remain available, got errors %v", results)
	}
}

type providerHelperRunner struct{}

func (providerHelperRunner) Run(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
	if len(args) == 1 && args[0] == "--help" {
		return platform.CommandResult{Stdout: "helper-stream"}, nil
	}

	return platform.CommandResult{}, nil
}

func (providerHelperRunner) RunStreaming(ctx context.Context, _ string, args []string, stdin string) (*platform.StreamResult, error) {
	return platform.NewCommandRunner().RunStreaming(ctx, os.Args[0], args, stdin)
}

func TestProviderHelperProcess(t *testing.T) {
	t.Helper()
	if os.Getenv("GO_WANT_PROVIDER_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for idx, arg := range args {
		if arg == "--" {
			args = args[idx+1:]
			break
		}
	}

	if len(args) < 2 || args[0] != "stderr-first" {
		return
	}

	payload := strings.Repeat("e", 256*1024)
	if _, err := io.WriteString(os.Stderr, payload); err != nil {
		os.Exit(2)
	}
	if _, err := io.WriteString(os.Stdout, "stream-complete:"+args[1]); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}
