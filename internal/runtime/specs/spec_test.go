package specs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestSpecAccessors (T-19) valida que SDKVersion(), NPMVersion() e NPMPackage()
// retornam exatamente os valores passados em newSpec via Claude().
func TestSpecAccessors(t *testing.T) {
	t.Parallel()

	spec := specs.Claude()

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

	spec := specs.Claude()

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
