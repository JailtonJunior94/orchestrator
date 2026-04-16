package providers

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/platform"
)

func helpWithOutputFormat() platform.CommandResult {
	return platform.CommandResult{Stdout: "-p prompt\n--output-format json\n"}
}

func helpWithShortOutputFlag() platform.CommandResult {
	return platform.CommandResult{Stdout: "-p prompt\n-o json\n"}
}

func helpWithoutOutputFormat() platform.CommandResult {
	return platform.CommandResult{Stdout: "-p prompt\n"}
}

func helpWithoutPromptFlag() platform.CommandResult {
	return platform.CommandResult{Stdout: "--output-format json\n"}
}

func TestGeminiProviderExecute_promptArgOutputFormat(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	var capturedStdin string
	provider := NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, name string, args []string, stdin string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return helpWithOutputFormat(), nil
			}
			capturedArgs = args
			capturedStdin = stdin
			return platform.CommandResult{Stdout: "Gemini answer"}, nil
		},
	})

	result, err := provider.Execute(context.Background(), ProviderInput{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Stdout != "Gemini answer" {
		t.Fatalf("stdout = %q", result.Stdout)
	}
	wantArgs := []string{"-p", "hello", "--output-format", "text"}
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

func TestGeminiProviderExecute_withOutputFormatJSON(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	provider := NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return helpWithOutputFormat(), nil
			}
			capturedArgs = args
			return platform.CommandResult{Stdout: `{"answer":"ok"}`}, nil
		},
	})

	_, err := provider.Execute(context.Background(), ProviderInput{
		Prompt:  "test",
		Options: map[string]string{"output_format": "json"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(capturedArgs) < 2 || capturedArgs[len(capturedArgs)-2] != "--output-format" || capturedArgs[len(capturedArgs)-1] != "json" {
		t.Fatalf("expected --output-format json in args, got %v", capturedArgs)
	}
}

func TestGeminiProviderExecuteStream_withOutputFormatJSON(t *testing.T) {
	t.Parallel()

	var capturedArgs []string
	provider := NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, args []string, _ string) (platform.CommandResult, error) {
			if len(args) == 1 && args[0] == "--help" {
				return helpWithOutputFormat(), nil
			}
			return platform.CommandResult{}, nil
		},
		RunStreamingFunc: func(_ context.Context, _ string, args []string, _ string) (*platform.StreamResult, error) {
			capturedArgs = append([]string(nil), args...)
			return platform.NewFakeStreamResult(`{"answer":"ok"}`, ""), nil
		},
	})

	_, err := provider.ExecuteStream(context.Background(), ProviderInput{
		Prompt:  "test",
		Options: map[string]string{"output_format": "json"},
	}, func(_ []byte) {})
	if err != nil {
		t.Fatal(err)
	}
	if len(capturedArgs) < 2 || capturedArgs[len(capturedArgs)-2] != "--output-format" || capturedArgs[len(capturedArgs)-1] != "json" {
		t.Fatalf("expected --output-format json in streaming args, got %v", capturedArgs)
	}
}

func TestGeminiProviderAvailable_outputFormatMissing(t *testing.T) {
	t.Parallel()

	provider := withLookup(NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return helpWithoutOutputFormat(), nil
		},
	}), func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	})

	err := provider.Available()
	if err == nil {
		t.Fatal("expected error when --output-format missing")
	}
	if !strings.Contains(err.Error(), "--output-format") {
		t.Fatalf("error should mention --output-format, got: %v", err)
	}
}

func TestGeminiProviderAvailable_shortOutputFlagSupported(t *testing.T) {
	t.Parallel()

	provider := withLookup(NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return helpWithShortOutputFlag(), nil
		},
	}), func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	})

	if err := provider.Available(); err != nil {
		t.Fatalf("unexpected error for short output flag profile: %v", err)
	}
}

func TestGeminiProviderAvailable_promptFlagMissing(t *testing.T) {
	t.Parallel()

	provider := withLookup(NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return helpWithoutPromptFlag(), nil
		},
	}), func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	})

	err := provider.Available()
	if err == nil {
		t.Fatal("expected error when -p is missing")
	}
	if !strings.Contains(err.Error(), "-p") {
		t.Fatalf("error should mention missing tokens, got: %v", err)
	}
}

func TestGeminiProviderAvailable_binaryNotFound(t *testing.T) {
	t.Parallel()

	provider := withLookup(NewGeminiProvider(platform.FakeCommandRunner{}), func(_ string) (string, error) {
		return "", errors.New("not found")
	})
	err := provider.Available()
	if err == nil {
		t.Fatal("expected error when binary missing")
	}
}

func TestGeminiProviderAvailable_success(t *testing.T) {
	t.Parallel()

	provider := withLookup(NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return helpWithOutputFormat(), nil
		},
	}), func(file string) (string, error) {
		return filepath.Join("/usr/bin", file), nil
	})

	if err := provider.Available(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGeminiProviderResolveProfileFallsBackToShortOutputFlag(t *testing.T) {
	t.Parallel()

	provider := NewGeminiProvider(platform.FakeCommandRunner{
		RunFunc: func(_ context.Context, _ string, _ []string, _ string) (platform.CommandResult, error) {
			return helpWithShortOutputFlag(), nil
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
	if profile != "prompt-short-output" {
		t.Fatalf("profile = %q, want %q", profile, "prompt-short-output")
	}
}
