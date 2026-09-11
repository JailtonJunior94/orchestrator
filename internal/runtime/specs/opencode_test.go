package specs_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestOpenCodeArgvSubcommandBeforeFlags(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().OpenCode()
	if len(spec.FixedArgs) == 0 || spec.FixedArgs[0] != "acp" {
		t.Fatalf("FixedArgs = %v; want first item %q", spec.FixedArgs, "acp")
	}

	bootstrap := spec.BootstrapArgs("", "", nil, specs.AccessModeRestricted, "/repo")
	argv := append(append([]string{}, spec.FixedArgs...), bootstrap...)

	want := []string{"acp", "--cwd", "/repo", "--log-level", "ERROR"}
	if len(argv) != len(want) {
		t.Fatalf("argv = %v; want %v", argv, want)
	}
	for i := range want {
		if argv[i] != want[i] {
			t.Fatalf("argv = %v; want %v", argv, want)
		}
	}

	if err := specs.ValidateFixedArgsFormat(spec.FixedArgs); err != nil {
		t.Fatalf("ValidateFixedArgsFormat(spec.FixedArgs) = %v; want nil", err)
	}
}

func TestOpenCodeFallbackPinnedVersion(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().OpenCode()
	if len(spec.Fallbacks) == 0 {
		t.Fatal("OpenCode() has no fallback launcher")
	}
	fb := spec.Fallbacks[0]
	wantPkg := specs.OpenCodeNpmPackage + "@" + specs.OpenCodeNpmVersion
	found := false
	for _, arg := range fb.FixedArgs {
		if arg == wantPkg {
			found = true
		}
		if strings.Contains(arg, "@latest") {
			t.Fatalf("fallback launcher uses @latest: %v", fb.FixedArgs)
		}
	}
	if !found {
		t.Fatalf("fallback FixedArgs = %v; want pinned package %q", fb.FixedArgs, wantPkg)
	}
}

func TestValidateFixedArgsFormatAcceptsSubcommandPrefix(t *testing.T) {
	t.Parallel()

	if err := specs.ValidateFixedArgsFormat([]string{"acp"}); err != nil {
		t.Fatalf("ValidateFixedArgsFormat([acp]) = %v; want nil", err)
	}
	if err := specs.ValidateFixedArgsFormat([]string{"--acp"}); err != nil {
		t.Fatalf("ValidateFixedArgsFormat([--acp]) = %v; want nil", err)
	}
	if err := specs.ValidateFixedArgsFormat(nil); err != nil {
		t.Fatalf("ValidateFixedArgsFormat(nil) = %v; want nil", err)
	}
}

func TestValidateFixedArgsFormatRejectsPositionalAfterFlag(t *testing.T) {
	t.Parallel()

	err := specs.ValidateFixedArgsFormat([]string{"--cwd", "acp"})
	if err == nil {
		t.Fatal("ValidateFixedArgsFormat([--cwd, acp]) = nil; want error")
	}
	if !strings.Contains(err.Error(), "acp") {
		t.Fatalf("error = %v; want message mentioning offending item", err)
	}
}

func TestResolveOpenCodeWindowExactMatch(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().OpenCode()
	got := spec.ResolveWindow("gemini-2.5-pro")
	if got.Class() != specs.WindowLarge {
		t.Fatalf("ResolveWindow(gemini-2.5-pro).Class() = %v; want WindowLarge", got.Class())
	}
}

func TestResolveOpenCodeWindowLongestPrefixMatch(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().OpenCode()
	got := spec.ResolveWindow("gemini-2.5-pro-20260101")
	if got.Class() != specs.WindowLarge {
		t.Fatalf("ResolveWindow(dated model).Class() = %v; want WindowLarge", got.Class())
	}
}

func TestResolveOpenCodeWindowUnknownModelFallsBackConservative(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().OpenCode()
	got := spec.ResolveWindow("totally-unknown-model-xyz")
	if got.MaxTokens != specs.OpenCodeConservativeWindow {
		t.Fatalf("ResolveWindow(unknown).MaxTokens = %d; want %d", got.MaxTokens, specs.OpenCodeConservativeWindow)
	}
	if got.Class() != specs.WindowStandard {
		t.Fatalf("ResolveWindow(unknown).Class() = %v; want WindowStandard", got.Class())
	}
}

func TestResolveOpenCodeWindowEmptyModelFallsBackConservative(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().OpenCode()
	got := spec.ResolveWindow("")
	if got.MaxTokens != specs.OpenCodeConservativeWindow {
		t.Fatalf("ResolveWindow(\"\").MaxTokens = %d; want %d", got.MaxTokens, specs.OpenCodeConservativeWindow)
	}
}

func TestResolveOpenCodeWindowNeverExceedsTableEntry(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().OpenCode()
	for _, model := range []string{"claude-opus-4", "claude-opus-4-20260305", "gpt-5", "gpt-5-turbo-20260101"} {
		got := spec.ResolveWindow(model)
		exact := spec.ResolveWindow(strings.SplitN(model, "-2026", 2)[0])
		if got.MaxTokens > exact.MaxTokens {
			t.Fatalf("ResolveWindow(%q).MaxTokens = %d exceeds base entry %d", model, got.MaxTokens, exact.MaxTokens)
		}
	}
}

func TestResolveWindowNilResolverIsByteIdenticalForExistingAgents(t *testing.T) {
	t.Parallel()

	for _, spec := range []specs.Spec{
		specs.NewCatalog().Claude(),
		specs.NewCatalog().Codex(),
		specs.NewCatalog().Copilot(),
	} {
		for _, model := range []string{"", "any-model", "gpt-5.5"} {
			got := spec.ResolveWindow(model)
			want := spec.ContextWindow()
			if got != want {
				t.Fatalf("spec %q: ResolveWindow(%q) = %+v; want static ContextWindow() = %+v", spec.ID, model, got, want)
			}
		}
	}
}
