package providers

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

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

// Verify all providers implement Provider (including ExecuteStream) at compile time.
var (
	_ Provider = (*geminiProvider)(nil)
	_ Provider = (*codexProvider)(nil)
	_ Provider = cliProvider{}
)

func TestProviderAvailable(t *testing.T) {
	t.Parallel()

	original := lookPath
	t.Cleanup(func() { lookPath = original })

	lookPath = func(file string) (string, error) {
		if file == "claude" {
			return filepath.Join("/tmp", file), nil
		}
		return "", errors.New("missing")
	}

	if err := NewClaudeProvider(platform.FakeCommandRunner{}).Available(); err != nil {
		t.Fatal(err)
	}
	if err := NewCopilotProvider(platform.FakeCommandRunner{}).Available(); err == nil {
		t.Fatal("expected error")
	}
}
