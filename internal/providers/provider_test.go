package providers

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func TestFactory(t *testing.T) {
	t.Parallel()

	factory := NewFactory(platform.FakeCommandRunner{})
	if _, err := factory.Get(ClaudeProviderName); err != nil {
		t.Fatal(err)
	}
	if _, err := factory.Get(CopilotProviderName); err != nil {
		t.Fatal(err)
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
