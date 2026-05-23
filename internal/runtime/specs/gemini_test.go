package specs_test

import (
	"slices"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestGeminiSpecHasCorrectCommandAndFlags (T-14) valida que Gemini() retorna Spec
// com ID, Command e FixedArgs corretos conforme ADR-015 D-01..D-04.
func TestGeminiSpecHasCorrectCommandAndFlags(t *testing.T) {
	t.Parallel()

	spec := specs.Gemini()

	if got, want := spec.ID, "gemini"; got != want {
		t.Errorf("Gemini().ID = %q; want %q", got, want)
	}
	if got, want := spec.Command, "gemini"; got != want {
		t.Errorf("Gemini().Command = %q; want %q (ADR-015 D-01)", got, want)
	}
	if got, want := spec.DisplayName, "Gemini (ACP)"; got != want {
		t.Errorf("Gemini().DisplayName = %q; want %q", got, want)
	}
	// FixedArgs deve conter exatamente ["--acp"]
	if got, want := len(spec.FixedArgs), 1; got != want {
		t.Fatalf("len(Gemini().FixedArgs) = %d; want %d", got, want)
	}
	if got, want := spec.FixedArgs[0], "--acp"; got != want {
		t.Errorf("Gemini().FixedArgs[0] = %q; want %q (ADR-015 D-03)", got, want)
	}
	// AccessModeFlag deve ser vazio (ADR-015 D-04)
	if got, want := spec.AccessModeFlag, ""; got != want {
		t.Errorf("Gemini().AccessModeFlag = %q; want %q (ADR-015 D-04)", got, want)
	}
}

// TestGeminiFallbackResolvesViaNpx (T-15) valida que Spec declara fallback npx
// com @google/gemini-cli@<GeminiNpmVersion> --acp.
func TestGeminiFallbackResolvesViaNpx(t *testing.T) {
	t.Parallel()

	spec := specs.Gemini()

	if got := len(spec.Fallbacks); got != 1 {
		t.Fatalf("len(Gemini().Fallbacks) = %d; want 1", got)
	}
	fb := spec.Fallbacks[0]
	if got, want := fb.Command, "npx"; got != want {
		t.Errorf("Fallbacks[0].Command = %q; want %q", got, want)
	}
	// Verifica --yes presente
	if !slices.Contains(fb.FixedArgs, "--yes") {
		t.Errorf("Fallbacks[0].FixedArgs = %v; want to contain --yes", fb.FixedArgs)
	}
	// Verifica @google/gemini-cli@<version> presente
	wantPkg := specs.GeminiNpmPackage + "@" + specs.GeminiNpmVersion
	if !slices.Contains(fb.FixedArgs, wantPkg) {
		t.Errorf("Fallbacks[0].FixedArgs = %v; want to contain %q", fb.FixedArgs, wantPkg)
	}
	// Verifica --acp presente no fallback
	if !slices.Contains(fb.FixedArgs, "--acp") {
		t.Errorf("Fallbacks[0].FixedArgs = %v; want to contain --acp", fb.FixedArgs)
	}
}

// TestGeminiBootstrapArgsForRestricted (T-16) valida que AccessModeRestricted
// gera --approval-mode default conforme mapeamento literal D-05 de ADR-015.
func TestGeminiBootstrapArgsForRestricted(t *testing.T) {
	t.Parallel()

	got := specs.Gemini().BootstrapArgs("", "", nil, specs.AccessModeRestricted)
	want := []string{"--approval-mode", "default"}
	assertSliceEqual(t, "T-16", got, want)
}

// TestGeminiBootstrapArgsForFull (T-29) valida que AccessModeFull
// gera --approval-mode yolo conforme mapeamento literal D-05 de ADR-015.
func TestGeminiBootstrapArgsForFull(t *testing.T) {
	t.Parallel()

	got := specs.Gemini().BootstrapArgs("", "", nil, specs.AccessModeFull)
	want := []string{"--approval-mode", "yolo"}
	assertSliceEqual(t, "T-29", got, want)
}

// TestGeminiBootstrapArgsIgnoresModelAndReasoning (T-30) valida que model, reasoning
// e addDirs são ignorados intencionalmente — divergência Compozy documentada em D-05 ADR-015.
// Resultado deve ser idêntico independentemente dos valores passados.
func TestGeminiBootstrapArgsIgnoresModelAndReasoning(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		model     string
		reasoning string
		addDirs   []string
		mode      specs.AccessMode
		want      []string
	}{
		{
			name:      "model-set-restricted",
			model:     "gemini-2.5-pro",
			reasoning: "high",
			addDirs:   []string{"/some/dir"},
			mode:      specs.AccessModeRestricted,
			want:      []string{"--approval-mode", "default"},
		},
		{
			name:      "model-set-full",
			model:     "gemini-2.5-flash",
			reasoning: "medium",
			addDirs:   []string{"/a", "/b"},
			mode:      specs.AccessModeFull,
			want:      []string{"--approval-mode", "yolo"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := specs.Gemini().BootstrapArgs(tc.model, tc.reasoning, tc.addDirs, tc.mode)
			assertSliceEqual(t, "T-30/"+tc.name, got, tc.want)
		})
	}
}

// TestGeminiBootstrapArgsDefaultsToRestricted (T-31) valida que valores não mapeados
// de AccessMode (incluindo string vazia) retornam --approval-mode default.
// Garante fallback seguro (principle of least privilege).
func TestGeminiBootstrapArgsDefaultsToRestricted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		mode specs.AccessMode
	}{
		{"empty-string", ""},
		{"unknown-value", "unknown"},
		{"typo-full", "Full"},
		{"typo-restricted", "Restricted"},
	}
	want := []string{"--approval-mode", "default"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := specs.Gemini().BootstrapArgs("", "", nil, tc.mode)
			assertSliceEqual(t, "T-31/"+tc.name, got, want)
		})
	}
}
