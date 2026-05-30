package specs_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestCodexDefaults (T-01) valida que Codex() retorna Spec com ID e Command corretos.
func TestCodexDefaults(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Codex()

	if got, want := spec.ID, "codex"; got != want {
		t.Errorf("Codex().ID = %q; want %q", got, want)
	}
	if got, want := spec.Command, "codex-acp"; got != want {
		t.Errorf("Codex().Command = %q; want %q (não 'codex' — CLI legacy OpenAI)", got, want)
	}
	if spec.FixedArgs != nil {
		t.Errorf("Codex().FixedArgs = %v; want nil (configuração via BootstrapArgs)", spec.FixedArgs)
	}
	if got, want := spec.AccessModeFlag, ""; got != want {
		t.Errorf("Codex().AccessModeFlag = %q; want %q (ADR-013 D-07)", got, want)
	}
}

// TestCodexMetadata (T-02) valida que Spec carrega metadata SDK/NPM corretas.
func TestCodexMetadata(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Codex()

	if got, want := spec.SDKVersion(), specs.CodexSDKVersion; got != want {
		t.Errorf("Codex().SDKVersion() = %q; want %q", got, want)
	}
	if got, want := spec.NPMVersion(), specs.CodexNpmVersion; got != want {
		t.Errorf("Codex().NPMVersion() = %q; want %q", got, want)
	}
	if got, want := spec.NPMPackage(), specs.CodexNpmPackage; got != want {
		t.Errorf("Codex().NPMPackage() = %q; want %q", got, want)
	}
}

// TestCodexFallback (T-03) valida que Spec declara fallback npx único com package correto.
func TestCodexFallback(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Codex()

	if got := len(spec.Fallbacks); got != 1 {
		t.Fatalf("len(Codex().Fallbacks) = %d; want 1", got)
	}
	fb := spec.Fallbacks[0]
	if got, want := fb.Command, "npx"; got != want {
		t.Errorf("Fallbacks[0].Command = %q; want %q", got, want)
	}
	// Verifica que os FixedArgs contêm o package@version correto.
	wantPkg := "@zed-industries/codex-acp@" + specs.CodexNpmVersion
	found := false
	for _, arg := range fb.FixedArgs {
		if arg == wantPkg {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Fallbacks[0].FixedArgs = %v; want to contain %q", fb.FixedArgs, wantPkg)
	}
	// Verifica --yes presente
	foundYes := false
	for _, arg := range fb.FixedArgs {
		if arg == "--yes" {
			foundYes = true
			break
		}
	}
	if !foundYes {
		t.Errorf("Fallbacks[0].FixedArgs = %v; want to contain %q", fb.FixedArgs, "--yes")
	}
}

// TestCodexVersionPinned (T-04) valida que a versão npm não é "latest" e que
// CodexMinNpmVersion <= CodexNpmVersion (ordem lexicográfica de semver para versões < 1.0).
func TestCodexVersionPinned(t *testing.T) {
	t.Parallel()

	if specs.CodexNpmVersion == "" {
		t.Error("CodexNpmVersion não pode ser vazio")
	}
	if specs.CodexNpmVersion == "latest" {
		t.Error("CodexNpmVersion não pode ser @latest (ADR-013 D-06)")
	}
	if specs.CodexMinNpmVersion == "" {
		t.Error("CodexMinNpmVersion não pode ser vazio")
	}
	// Ordem semver lexicográfica: válida para versões 0.x.y com mesmo número de dígitos.
	if specs.CodexMinNpmVersion > specs.CodexNpmVersion {
		t.Errorf("CodexMinNpmVersion (%s) > CodexNpmVersion (%s); mínimo deve ser <= pinada",
			specs.CodexMinNpmVersion, specs.CodexNpmVersion)
	}
}

// TestCodexBootstrapArgsEmptyModelRestricted (T-06) valida o caso base sem model/reasoning.
// Resultado esperado: apenas os feature toggles obrigatórios.
func TestCodexBootstrapArgsEmptyModelRestricted(t *testing.T) {
	t.Parallel()

	got := specs.NewCatalog().Codex().BootstrapArgs("", "", nil, specs.AccessModeRestricted)
	want := []string{
		"-c", "features.code_mode=false",
		"-c", "features.code_mode_only=false",
	}
	assertSliceEqual(t, "T-06", got, want)
	// Garantir ausência de sandbox flags
	assertNotContains(t, "T-06", got, "sandbox_mode")
	assertNotContains(t, "T-06", got, "approval_policy")
	assertNotContains(t, "T-06", got, "web_search")
}

// TestCodexBootstrapArgsModelReasoningRestricted (T-07) valida model+reasoning com modo restrito.
func TestCodexBootstrapArgsModelReasoningRestricted(t *testing.T) {
	t.Parallel()

	got := specs.NewCatalog().Codex().BootstrapArgs("gpt-5.5", "medium", nil, specs.AccessModeRestricted)

	assertContainsPair(t, "T-07", got, "-c", `model="gpt-5.5"`)
	assertContainsPair(t, "T-07", got, "-c", `model_reasoning_effort="medium"`)
	assertContainsPair(t, "T-07", got, "-c", "features.code_mode=false")
	assertContainsPair(t, "T-07", got, "-c", "features.code_mode_only=false")
	// Sem sandbox flags
	assertNotContains(t, "T-07", got, "sandbox_mode")
	assertNotContains(t, "T-07", got, "approval_policy")
	assertNotContains(t, "T-07", got, "web_search")
}

// TestCodexBootstrapArgsFullAccess (T-08) valida que AccessModeFull acrescenta sandbox flags.
func TestCodexBootstrapArgsFullAccess(t *testing.T) {
	t.Parallel()

	got := specs.NewCatalog().Codex().BootstrapArgs("gpt-5.5", "high", nil, specs.AccessModeFull)

	assertContainsPair(t, "T-08", got, "-c", `model="gpt-5.5"`)
	assertContainsPair(t, "T-08", got, "-c", `model_reasoning_effort="high"`)
	assertContainsPair(t, "T-08", got, "-c", "features.code_mode=false")
	assertContainsPair(t, "T-08", got, "-c", "features.code_mode_only=false")
	assertContainsPair(t, "T-08", got, "-c", `approval_policy="never"`)
	assertContainsPair(t, "T-08", got, "-c", `sandbox_mode="danger-full-access"`)
	assertContainsPair(t, "T-08", got, "-c", `web_search="live"`)
}

// TestCodexBootstrapArgsReasoningLow (T-09) valida reasoning="low" é aceito sem erro.
func TestCodexBootstrapArgsReasoningLow(t *testing.T) {
	t.Parallel()

	got := specs.NewCatalog().Codex().BootstrapArgs("gpt-5.5", "low", nil, specs.AccessModeRestricted)
	assertContainsPair(t, "T-09", got, "-c", `model_reasoning_effort="low"`)
}

// TestCodexBootstrapArgsDelegates (T-12) valida que Spec.BootstrapArgs delega para
// codexBootstrapArgs — resultado idêntico ao obtido via BootstrapArgs diretamente.
func TestCodexBootstrapArgsDelegates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		model     string
		reasoning string
		mode      specs.AccessMode
	}{
		{"empty-restricted", "", "", specs.AccessModeRestricted},
		{"model-medium-restricted", "gpt-5.5", "medium", specs.AccessModeRestricted},
		{"model-high-full", "gpt-5.5", "high", specs.AccessModeFull},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Resultado via Spec.BootstrapArgs
			viaSpec := specs.NewCatalog().Codex().BootstrapArgs(tc.model, tc.reasoning, nil, tc.mode)
			// Resultado via segundo Codex() — deve ser idêntico
			viaSpec2 := specs.NewCatalog().Codex().BootstrapArgs(tc.model, tc.reasoning, nil, tc.mode)
			assertSliceEqual(t, "T-12/"+tc.name, viaSpec, viaSpec2)
		})
	}
}

// --- helpers de asserção ---

func assertSliceEqual(t *testing.T, testID string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: len(args) = %d; want %d\ngot:  %v\nwant: %v", testID, len(got), len(want), got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s: args[%d] = %q; want %q\ngot:  %v\nwant: %v", testID, i, got[i], want[i], got, want)
		}
	}
}

// assertContainsPair verifica que o par (flag, value) aparece consecutivamente em args.
func assertContainsPair(t *testing.T, testID string, args []string, flag, value string) {
	t.Helper()
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return
		}
	}
	t.Errorf("%s: par (%q, %q) não encontrado em args: %v", testID, flag, value, args)
}

// assertNotContains verifica que nenhum elemento de args contém a substring substr.
func assertNotContains(t *testing.T, testID string, args []string, substr string) {
	t.Helper()
	for _, arg := range args {
		if strings.Contains(arg, substr) {
			t.Errorf("%s: args contém substring proibida %q (em %q): %v", testID, substr, arg, args)
		}
	}
}
