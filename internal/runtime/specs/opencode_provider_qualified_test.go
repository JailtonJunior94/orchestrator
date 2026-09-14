package specs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestResolveOpenCodeWindowProviderQualifiedModel(t *testing.T) {
	spec := specs.NewCatalog().OpenCode()
	cases := []struct {
		model     string
		wantMax   int
		wantClass specs.WindowClass
	}{
		{"anthropic/claude-sonnet-4-5", 200_000, specs.WindowStandard},
		{"google/gemini-2.5-pro", 1_000_000, specs.WindowLarge},
		{"openai/gpt-5", 400_000, specs.WindowStandard},
		{"openrouter/google/gemini-2.5-flash", 1_000_000, specs.WindowLarge},
		{"claude-sonnet-4-5", 200_000, specs.WindowStandard},
		{"anthropic/totally-unknown-model", specs.OpenCodeConservativeWindow, specs.WindowStandard},
	}
	for _, tc := range cases {
		got := spec.ResolveWindow(tc.model)
		if got.MaxTokens != tc.wantMax {
			t.Errorf("ResolveWindow(%q).MaxTokens = %d; want %d", tc.model, got.MaxTokens, tc.wantMax)
		}
		if got.Class() != tc.wantClass {
			t.Errorf("ResolveWindow(%q).Class() = %v; want %v", tc.model, got.Class(), tc.wantClass)
		}
	}
}

func TestNormalizeModelID(t *testing.T) {
	cases := map[string]string{
		"":                            "",
		"gpt-5":                       "gpt-5",
		"openai/gpt-5":                "gpt-5",
		"openrouter/openai/gpt-5":     "gpt-5",
		"anthropic/claude-sonnet-4-5": "claude-sonnet-4-5",
	}
	for in, want := range cases {
		if got := specs.NormalizeModelID(in); got != want {
			t.Errorf("NormalizeModelID(%q) = %q; want %q", in, got, want)
		}
	}
}
