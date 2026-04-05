package providers

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func codexHelpWithSandboxAndJSON() platform.CommandResult {
	return platform.CommandResult{Stdout: "exec run a task\n--sandbox mode\n--json output events\n"}
}

func codexHelpWithShortSandboxAndJSON() platform.CommandResult {
	return platform.CommandResult{Stdout: "exec run a task\n-s mode\n--json output events\n"}
}

func codexHelpWithoutJSON() platform.CommandResult {
	return platform.CommandResult{Stdout: "exec run a task\n--sandbox mode\n"}
}

func codexHelpWithoutExec() platform.CommandResult {
	return platform.CommandResult{Stdout: "--sandbox mode\n--json output events\n"}
}

func TestCodexProviderExecute_execSandbox(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	var capturedStdin string
	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, stdin string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return codexHelpWithSandboxAndJSON(), nil
			}
			capturedArgs = args
			capturedStdin = stdin
			return platform.CommandResult{Stdout: "Codex answer"}, nil
		},
	})

	result, err := provider.Execute(context.Background(), ProviderInput{Prompt: "fix bug"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "Codex answer" {
		t.Fatalf("stdout = %q", result.Stdout)
	}
	wantArgs := []string{"exec", "--sandbox", "read-only", "fix bug"}
	if len(capturedArgs) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", capturedArgs, wantArgs)
	}
	for i := range wantArgs {
		if capturedArgs[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q, want %q", i, capturedArgs[i], wantArgs[i])
		}
	}
	if capturedStdin != "" {
		t.Fatalf("stdin = %q", capturedStdin)
	}
}

func TestCodexProviderExecute_withJSONOutput(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return codexHelpWithSandboxAndJSON(), nil
			}
			capturedArgs = args
			return platform.CommandResult{Stdout: `{"type":"message","content":"ok"}`}, nil
		},
	})

	_, err := provider.Execute(context.Background(), ProviderInput{
		Prompt:  "test",
		Options: map[string]string{"output_format": "jsonl"},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, arg := range capturedArgs {
		if arg == "--json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --json in args, got %v", capturedArgs)
	}
}

func TestCodexProviderExecuteStream_withJSONOutputAndSandbox(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return codexHelpWithSandboxAndJSON(), nil
			}
			return platform.CommandResult{}, nil
		},
		RunStreamingFunc: func(_ context.Context, _ string, args []string, _ string) (*platform.StreamResult, error) {
			capturedArgs = append([]string(nil), args...)
			return platform.NewFakeStreamResult(`{"type":"message","content":"ok"}`+"\n", ""), nil
		},
	})

	_, err := provider.ExecuteStream(context.Background(), ProviderInput{
		Prompt: "test",
		Options: map[string]string{
			"output_format": "jsonl",
			"sandbox":       "read-only",
		},
	}, func(_ []byte) {})
	if err != nil {
		t.Fatal(err)
	}

	foundJSON := false
	foundSandbox := false
	for i, arg := range capturedArgs {
		if arg == "--json" {
			foundJSON = true
		}
		if arg == "--sandbox" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "read-only" {
			foundSandbox = true
		}
	}
	if !foundJSON {
		t.Fatalf("expected --json in streaming args, got %v", capturedArgs)
	}
	if !foundSandbox {
		t.Fatalf("expected --sandbox read-only in streaming args, got %v", capturedArgs)
	}
}

func TestCodexProviderExecute_sandboxAlwaysReadOnly(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return codexHelpWithSandboxAndJSON(), nil
			}
			capturedArgs = args
			return platform.CommandResult{Stdout: "ok"}, nil
		},
	})

	_, err := provider.Execute(context.Background(), ProviderInput{
		Prompt:  "do it",
		Options: map[string]string{"sandbox": "read-only"},
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i, arg := range capturedArgs {
		if arg == "--sandbox" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "read-only" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --sandbox read-only in args, got %v", capturedArgs)
	}
}

func TestCodexProviderAvailable_jsonMissing(t *testing.T) {
	t.Parallel()

	original := lookPath
	t.Cleanup(func() { lookPath = original })
	lookPath = func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	}

	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return codexHelpWithoutJSON(), nil
		},
	})

	err := provider.Available()
	if err == nil {
		t.Fatal("expected error when --json is missing")
	}
	if !strings.Contains(err.Error(), "--json") {
		t.Fatalf("error should mention --json, got: %v", err)
	}
}

func TestCodexProviderAvailable_shortSandboxFlagSupported(t *testing.T) {
	t.Parallel()

	original := lookPath
	t.Cleanup(func() { lookPath = original })
	lookPath = func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	}

	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return codexHelpWithShortSandboxAndJSON(), nil
		},
	})

	if err := provider.Available(); err != nil {
		t.Fatalf("unexpected error for short sandbox profile: %v", err)
	}
}

func TestCodexProviderAvailable_execMissing(t *testing.T) {
	// Not parallel: modifies the package-level lookPath variable.
	original := lookPath
	t.Cleanup(func() { lookPath = original })
	lookPath = func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	}

	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return codexHelpWithoutExec(), nil
		},
	})

	err := provider.Available()
	if err == nil {
		t.Fatal("expected error when exec is missing")
	}
	if !strings.Contains(err.Error(), "exec") {
		t.Fatalf("error should mention missing tokens, got: %v", err)
	}
}

func TestCodexProviderAvailable_binaryNotFound(t *testing.T) {
	t.Parallel()

	original := lookPath
	t.Cleanup(func() { lookPath = original })
	lookPath = func(_ string) (string, error) {
		return "", errors.New("not found")
	}

	if err := NewCodexProvider(platform.FakeCommandRunner{}).Available(); err == nil {
		t.Fatal("expected error when binary missing")
	}
}

func TestCodexProviderAvailable_success(t *testing.T) {
	t.Parallel()

	original := lookPath
	t.Cleanup(func() { lookPath = original })
	lookPath = func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	}

	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return codexHelpWithSandboxAndJSON(), nil
		},
	})

	if err := provider.Available(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCodexProviderResolveProfileFallsBackToShortSandboxFlag(t *testing.T) {
	t.Parallel()

	provider := NewCodexProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return codexHelpWithShortSandboxAndJSON(), nil
		},
	})

	resolver, ok := provider.(interface {
		ResolveProfile(context.Context) (string, error)
	})
	if !ok {
		t.Fatal("provider does not expose ResolveProfile")
	}

	profile, err := resolver.ResolveProfile(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if profile != "exec-short-sandbox-json" {
		t.Fatalf("profile = %q, want %q", profile, "exec-short-sandbox-json")
	}
}
