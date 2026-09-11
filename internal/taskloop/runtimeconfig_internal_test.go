package taskloop

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
)

func TestOptionsToConfigOverrides_MaxBugfixIterationsDefaultDoesNotOverrideConfig(t *testing.T) {
	t.Parallel()

	got := NewCatalog().optionsToConfigOverrides(Options{MaxBugfixIterations: 5})

	if got.MaxBugfixIterations != 0 {
		t.Fatalf("default do Cobra nao deve virar override; got %d", got.MaxBugfixIterations)
	}
}

func TestOptionsToConfigOverrides_MaxBugfixIterationsExplicitOverridesConfig(t *testing.T) {
	t.Parallel()

	got := NewCatalog().optionsToConfigOverrides(Options{
		MaxBugfixIterations:    9,
		MaxBugfixIterationsSet: true,
	})

	if got.MaxBugfixIterations != 9 {
		t.Fatalf("flag explicita deve virar override; got %d", got.MaxBugfixIterations)
	}
}

func TestResolveRuntimeConfig_MaxBugfixIterationsFromWorkspaceConfigOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatalf("mkdir .claude: %v", err)
	}
	yaml := "max_bugfix_iterations: 4\n"
	if err := os.WriteFile(filepath.Join(claudeDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}

	opts := Options{}
	rc, err := NewCatalog().resolveRuntimeConfig(dir, NewCatalog().optionsToConfigOverrides(opts))
	if err != nil {
		t.Fatalf("resolveRuntimeConfig: %v", err)
	}
	if rc.MaxBugfixIterations != 4 {
		t.Fatalf("MaxBugfixIterations resolvido do arquivo de workspace = %d, want 4", rc.MaxBugfixIterations)
	}

	resolvedOpts := NewCatalog().applyResolvedMaxBugfixIterations(opts, rc)
	if resolvedOpts.MaxBugfixIterations != 4 {
		t.Fatalf("Options.MaxBugfixIterations apos resolucao de config = %d, want 4", resolvedOpts.MaxBugfixIterations)
	}
}

func TestApplyResolvedMaxBugfixIterations_ZeroValuePreservesOptions(t *testing.T) {
	t.Parallel()

	opts := Options{MaxBugfixIterations: 0}
	got := NewCatalog().applyResolvedMaxBugfixIterations(opts, airuntime.RuntimeConfig{})
	if got.MaxBugfixIterations != 0 {
		t.Fatalf("Options.MaxBugfixIterations deveria permanecer 0; got %d", got.MaxBugfixIterations)
	}
}

func TestOptionsToConfigOverrides_ActivityTimeoutDefaultDoesNotOverrideConfig(t *testing.T) {
	t.Parallel()

	got := NewCatalog().optionsToConfigOverrides(Options{
		ActivityTimeout: 120 * time.Second,
	})

	if got.Timeout != "" {
		t.Fatalf("timeout default do Cobra nao deve virar override; got %q", got.Timeout)
	}
}

func TestOptionsToConfigOverrides_ActivityTimeoutExplicitOverridesConfig(t *testing.T) {
	t.Parallel()

	got := NewCatalog().optionsToConfigOverrides(Options{
		ActivityTimeout:    90 * time.Second,
		ActivityTimeoutSet: true,
	})

	if got.Timeout != "1m30s" {
		t.Fatalf("timeout explicito deve virar override; got %q", got.Timeout)
	}
}

// TestOptionsToConfigOverrides_ActivityTimeoutExplicitZeroOverrides garante que
// --activity-timeout=0 (explicito) propaga "0s" para desabilitar o watchdog (flag help: 0=desabilitado).
func TestOptionsToConfigOverrides_ActivityTimeoutExplicitZeroOverrides(t *testing.T) {
	t.Parallel()

	got := NewCatalog().optionsToConfigOverrides(Options{
		ActivityTimeout:    0,
		ActivityTimeoutSet: true,
	})

	if got.Timeout != "0s" {
		t.Fatalf("--activity-timeout=0 explicito deve propagar \"0s\"; got %q", got.Timeout)
	}
}

// TestResolveRuntimeConfig_DefaultWatchdogIs120s é a regressão da correção do BF-01: sem flag
// explícita e sem config, o watchdog ACP deve usar o default F1 de 120s (não 0/desabilitado).
func TestResolveRuntimeConfig_DefaultWatchdogIs120s(t *testing.T) {
	t.Parallel()

	dir := t.TempDir() // sem config.yaml
	// Default do Cobra (120s) sem flag explícita: ActivityTimeoutSet=false.
	opts := Options{ActivityTimeout: 120 * time.Second, ActivityTimeoutSet: false}

	rc, err := NewCatalog().resolveRuntimeConfig(dir, NewCatalog().optionsToConfigOverrides(opts))
	if err != nil {
		t.Fatalf("resolveRuntimeConfig: %v", err)
	}
	if rc.Timeout.Duration() != 120*time.Second {
		t.Fatalf("watchdog default deve ser 120s (F1); got %v", rc.Timeout.Duration())
	}
}

// TestResolveRuntimeConfig_ExplicitZeroDisablesWatchdog garante que --activity-timeout=0
// explícito desabilita o watchdog mesmo com o default F1 de 120s presente.
func TestResolveRuntimeConfig_ExplicitZeroDisablesWatchdog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	opts := Options{ActivityTimeout: 0, ActivityTimeoutSet: true}

	rc, err := NewCatalog().resolveRuntimeConfig(dir, NewCatalog().optionsToConfigOverrides(opts))
	if err != nil {
		t.Fatalf("resolveRuntimeConfig: %v", err)
	}
	if !rc.Timeout.Disabled() {
		t.Fatalf("--activity-timeout=0 explícito deve desabilitar o watchdog; got %v", rc.Timeout.Duration())
	}
}

// TestResolveRuntimeConfig_ExplicitValueWins garante que um valor explícito de flag prevalece
// sobre o default F1.
func TestResolveRuntimeConfig_ExplicitValueWins(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	opts := Options{ActivityTimeout: 45 * time.Second, ActivityTimeoutSet: true}

	rc, err := NewCatalog().resolveRuntimeConfig(dir, NewCatalog().optionsToConfigOverrides(opts))
	if err != nil {
		t.Fatalf("resolveRuntimeConfig: %v", err)
	}
	if rc.Timeout.Duration() != 45*time.Second {
		t.Fatalf("valor explícito deve prevalecer sobre default; got %v", rc.Timeout.Duration())
	}
}
