package specs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestSpecAccessors (T-19) valida que SDKVersion(), NPMVersion() e NPMPackage()
// retornam exatamente os valores passados em newSpec via Claude().
func TestSpecAccessors(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Claude()

	if got, want := spec.SDKVersion(), specs.ClaudeSDKVersion; got != want {
		t.Errorf("SDKVersion() = %q; want %q", got, want)
	}

	if got, want := spec.NPMVersion(), specs.ClaudeNpmVersion; got != want {
		t.Errorf("NPMVersion() = %q; want %q", got, want)
	}

	if got, want := spec.NPMPackage(), specs.ClaudeNpmPackage; got != want {
		t.Errorf("NPMPackage() = %q; want %q", got, want)
	}
}

// TestSpecAccessorsNonEmpty valida que os acessores retornam strings não-vazias para Claude.
func TestSpecAccessorsNonEmpty(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Claude()

	if spec.SDKVersion() == "" {
		t.Error("SDKVersion() deve ser não-vazio para Claude")
	}

	if spec.NPMVersion() == "" {
		t.Error("NPMVersion() deve ser não-vazio para Claude")
	}

	if spec.NPMPackage() == "" {
		t.Error("NPMPackage() deve ser não-vazio para Claude")
	}
}

// TestBootstrapArgsNoOpClaude (T-10) valida que Claude().BootstrapArgs retorna nil (no-op default).
// Claude não injeta BootstrapArgsFunc — bootstrapArgs == nil → retorna nil.
func TestBootstrapArgsNoOpClaude(t *testing.T) {
	t.Parallel()

	got := specs.NewCatalog().Claude().BootstrapArgs("any", "any", nil, specs.AccessModeFull)
	if got != nil {
		t.Errorf("Claude().BootstrapArgs() = %v; want nil (no-op)", got)
	}
}

// TestBootstrapArgsNoOpCopilot (T-11) valida que Copilot().BootstrapArgs retorna nil (no-op default).
// Copilot não injeta BootstrapArgsFunc — bootstrapArgs == nil → retorna nil.
func TestBootstrapArgsNoOpCopilot(t *testing.T) {
	t.Parallel()

	got := specs.NewCatalog().Copilot().BootstrapArgs("any", "any", nil, specs.AccessModeFull)
	if got != nil {
		t.Errorf("Copilot().BootstrapArgs() = %v; want nil (no-op)", got)
	}
}

// TestSpec_ContextWindow_Catalog verifica que todos os catálogos populam ContextWindow (ADR-023).
func TestSpec_ContextWindow_Catalog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		spec      specs.Spec
		wantClass specs.WindowClass
		wantPos   bool // MaxTokens deve ser positivo
	}{
		{"claude", specs.NewCatalog().Claude(), specs.WindowStandard, true},
		{"codex", specs.NewCatalog().Codex(), specs.WindowStandard, true},
		{"copilot", specs.NewCatalog().Copilot(), specs.WindowStandard, true},
		{"gemini", specs.NewCatalog().Gemini(), specs.WindowLarge, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cw := tc.spec.ContextWindow()

			if tc.wantPos && cw.MaxTokens <= 0 {
				t.Errorf("%s: MaxTokens = %d; deve ser positivo", tc.name, cw.MaxTokens)
			}

			got := cw.Class()
			if got != tc.wantClass {
				t.Errorf("%s: Class() = %v; want %v", tc.name, got, tc.wantClass)
			}
		})
	}
}

// TestSpec_Gemini_IsWindowLarge garante que Gemini retorna WindowLarge (invariante ADR-023).
func TestSpec_Gemini_IsWindowLarge(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Gemini()
	cw := spec.ContextWindow()

	if cw.Class() != specs.WindowLarge {
		t.Errorf("Gemini ContextWindow.Class() = %v; want WindowLarge (janela ≥1M)", cw.Class())
	}

	if cw.MaxTokens < 1_000_000 {
		t.Errorf("Gemini MaxTokens = %d; deve ser ≥1_000_000", cw.MaxTokens)
	}
}

// TestSpec_ClaudeCodexCopilot_IsWindowStandard garante regressão F1 para as 3 CLIs standard.
func TestSpec_ClaudeCodexCopilot_IsWindowStandard(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		spec specs.Spec
	}{
		{"claude", specs.NewCatalog().Claude()},
		{"codex", specs.NewCatalog().Codex()},
		{"copilot", specs.NewCatalog().Copilot()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.spec.ContextWindow().Class()
			if got != specs.WindowStandard {
				t.Errorf("%s: Class() = %v; want WindowStandard (F1 invariant)", tc.name, got)
			}
		})
	}
}

// TestSpec_DriverID_Catalog verifica que DriverID() retorna o VO correto para os 4 catálogos.
func TestSpec_DriverID_Catalog(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec specs.Spec
		want string
	}{
		{"claude", specs.NewCatalog().Claude(), "claude"},
		{"codex", specs.NewCatalog().Codex(), "codex"},
		{"copilot", specs.NewCatalog().Copilot(), "copilot"},
		{"gemini", specs.NewCatalog().Gemini(), "gemini"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.spec.DriverID().String()
			if got != tc.want {
				t.Errorf("DriverID() = %q; want %q", got, tc.want)
			}
		})
	}
}
