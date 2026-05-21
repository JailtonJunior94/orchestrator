package specs_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestCopilotSpecDefaults (T-01) valida ID, Command e FixedArgs da Spec Copilot.
func TestCopilotSpecDefaults(t *testing.T) {
	t.Parallel()

	spec := specs.Copilot()

	if got, want := spec.ID, "copilot"; got != want {
		t.Errorf("ID = %q; want %q", got, want)
	}

	if got, want := spec.Command, "copilot"; got != want {
		t.Errorf("Command = %q; want %q", got, want)
	}

	if got, want := len(spec.FixedArgs), 1; got != want {
		t.Fatalf("len(FixedArgs) = %d; want %d", got, want)
	}

	if got, want := spec.FixedArgs[0], "--acp"; got != want {
		t.Errorf("FixedArgs[0] = %q; want %q", got, want)
	}
}

// TestCopilotSpecMetadata (T-02) valida que SDKVersion, NPMVersion e NPMPackage estão corretos.
func TestCopilotSpecMetadata(t *testing.T) {
	t.Parallel()

	spec := specs.Copilot()

	if got, want := spec.SDKVersion(), specs.CopilotSDKVersion; got != want {
		t.Errorf("SDKVersion() = %q; want %q", got, want)
	}

	if got, want := spec.NPMVersion(), specs.CopilotNpmVersion; got != want {
		t.Errorf("NPMVersion() = %q; want %q", got, want)
	}

	if got, want := spec.NPMPackage(), specs.CopilotNpmPackage; got != want {
		t.Errorf("NPMPackage() = %q; want %q", got, want)
	}
}

// TestCopilotSpecFallback (T-03) valida que existe exatamente um fallback npx com --acp ao final.
func TestCopilotSpecFallback(t *testing.T) {
	t.Parallel()

	spec := specs.Copilot()

	if got, want := len(spec.Fallbacks), 1; got != want {
		t.Fatalf("len(Fallbacks) = %d; want %d", got, want)
	}

	fb := spec.Fallbacks[0]

	if got, want := fb.Command, "npx"; got != want {
		t.Errorf("Fallbacks[0].Command = %q; want %q", got, want)
	}

	if len(fb.FixedArgs) == 0 {
		t.Fatal("Fallbacks[0].FixedArgs deve ser não-vazio")
	}

	// O último argumento do fallback deve ser "--acp"
	lastArg := fb.FixedArgs[len(fb.FixedArgs)-1]
	if lastArg != "--acp" {
		t.Errorf("Fallbacks[0].FixedArgs último arg = %q; want %q", lastArg, "--acp")
	}

	// Valida que o segundo arg contém o pacote npm e a versão pinada
	if len(fb.FixedArgs) < 2 {
		t.Fatalf("Fallbacks[0].FixedArgs deve ter ao menos 2 elementos; got %d", len(fb.FixedArgs))
	}

	wantPkgArg := specs.CopilotNpmPackage + "@" + specs.CopilotNpmVersion
	if fb.FixedArgs[1] != wantPkgArg {
		t.Errorf("Fallbacks[0].FixedArgs[1] = %q; want %q", fb.FixedArgs[1], wantPkgArg)
	}
}

// TestCopilotConstantsPinned (T-04) valida que as constantes de versão não são vazias nem "@latest".
func TestCopilotConstantsPinned(t *testing.T) {
	t.Parallel()

	if specs.CopilotNpmVersion == "" {
		t.Error("CopilotNpmVersion não deve ser vazio")
	}

	if strings.EqualFold(specs.CopilotNpmVersion, "latest") {
		t.Errorf("CopilotNpmVersion = %q; não deve ser 'latest'", specs.CopilotNpmVersion)
	}

	if strings.Contains(specs.CopilotNpmVersion, "@latest") {
		t.Errorf("CopilotNpmVersion = %q; não deve conter '@latest'", specs.CopilotNpmVersion)
	}

	if specs.CopilotMinCLIVersion == "" {
		t.Error("CopilotMinCLIVersion não deve ser vazio")
	}

	if specs.CopilotNpmPackage == "" {
		t.Error("CopilotNpmPackage não deve ser vazio")
	}

	if specs.CopilotSDKVersion == "" {
		t.Error("CopilotSDKVersion não deve ser vazio")
	}
}

// TestCopilotAccessModeFlag valida que AccessModeFlag é vazio (D-07: sem flag análoga a --bypass-permissions).
func TestCopilotAccessModeFlag(t *testing.T) {
	t.Parallel()

	spec := specs.Copilot()

	if spec.AccessModeFlag != "" {
		t.Errorf("AccessModeFlag = %q; want %q (D-07: Copilot v0 sem flag análoga)", spec.AccessModeFlag, "")
	}
}

// TestCopilotDisplayName valida o nome de exibição da Spec Copilot.
func TestCopilotDisplayName(t *testing.T) {
	t.Parallel()

	spec := specs.Copilot()

	if got, want := spec.DisplayName, "GitHub Copilot CLI (ACP)"; got != want {
		t.Errorf("DisplayName = %q; want %q", got, want)
	}
}

// TestCopilotStability garante que Copilot() retorna a mesma estrutura entre chamadas.
func TestCopilotStability(t *testing.T) {
	t.Parallel()

	a := specs.Copilot()
	b := specs.Copilot()

	if a.ID != b.ID {
		t.Errorf("ID inconsistente: %q != %q", a.ID, b.ID)
	}

	if a.Command != b.Command {
		t.Errorf("Command inconsistente: %q != %q", a.Command, b.Command)
	}

	if a.AccessModeFlag != b.AccessModeFlag {
		t.Errorf("AccessModeFlag inconsistente: %q != %q", a.AccessModeFlag, b.AccessModeFlag)
	}

	if len(a.Fallbacks) != len(b.Fallbacks) {
		t.Errorf("len(Fallbacks) inconsistente: %d != %d", len(a.Fallbacks), len(b.Fallbacks))
	}
}
