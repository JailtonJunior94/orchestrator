package specs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestDeclaredProviderLimitationsAreWellFormed(t *testing.T) {
	t.Parallel()

	for _, limitation := range specs.DeclaredProviderLimitations {
		if limitation.Provider == "" || limitation.Area == "" || limitation.Description == "" || limitation.Evidence == "" {
			t.Errorf("declared limitation has an empty field: %+v", limitation)
		}
	}
}

func TestDeclaredProviderLimitationsCoverEveryRequiredArea(t *testing.T) {
	t.Parallel()

	required := []string{
		"preToolUse-timeout",
		"trustedFolders-precondition",
		"after-tool-non-blocking",
		"permission-request-out-of-scope",
		"hook-trust-silent-skip",
	}

	byArea := make(map[string]bool, len(specs.DeclaredProviderLimitations))
	for _, limitation := range specs.DeclaredProviderLimitations {
		byArea[limitation.Area] = true
	}
	for _, area := range required {
		if !byArea[area] {
			t.Errorf("no declared limitation for required area %q", area)
		}
	}
}
