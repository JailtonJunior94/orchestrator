//go:build integration

// Package parity — testes E2E cross-project (RF-18) e paridade de fallback (RF-19).
// Build tag: integration. Usa t.TempDir() e binários fake via LookPather — sem dependência
// de binário ACP real no PATH. Conforme techspec §"Testes E2E" e task 7.0.
package parity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	internalfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// ── Helpers de integração ────────────────────────────────────────────────────

// newRealInstallService cria um install.Service com OSFileSystem para testes de integracao.
func newRealInstallService(t *testing.T) *install.Service {
	t.Helper()
	fsys := internalfs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	return install.NewService(fsys, printer, mfst, adpt, ctxg)
}

// createMinimalSourceDir cria um diretório fonte com skills mínimas para testes E2E.
func createMinimalSourceDir(t *testing.T, sourceDir string, skillNames []string) {
	t.Helper()
	for _, sk := range skillNames {
		skillDir := filepath.Join(sourceDir, ".agents", "skills", sk)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("criar skill dir %s: %v", sk, err)
		}
		content := []byte("---\nversion: 1.0.0\n---\n# " + sk + "\nconteudo de teste")
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), content, 0o644); err != nil {
			t.Fatalf("criar SKILL.md para %s: %v", sk, err)
		}
	}
}

// ── Testes cross-project (RF-18) ────────────────────────────────────────────

// TestE2EParity_CrossProject_InstallAndValidateInvariants instala o harness em um
// repo temporário (t.TempDir()) e valida invariantes parity via parity.Generate+Run.
// RF-18: paridade cross-project — instalar em repo temporário e validar invariantes.
func TestE2EParity_CrossProject_InstallAndValidateInvariants(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSourceDir(t, sourceDir, []string{"review", "execute-task", "agent-governance"})

	svc := newRealInstallService(t)

	start := time.Now()
	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("install.Execute falhou: %v", err)
	}
	if elapsed > 30*time.Second {
		t.Errorf("bootstrap cross-project demorou %v, limite e 30s (RF-11)", elapsed)
	}
	t.Logf("cross-project install concluido em %v", elapsed)

	// Validar invariantes parity via Generate no projectDir instalado.
	snap, err := Generate(projectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("parity.Generate falhou apos install: %v", err)
	}

	// Executar suite completa de invariantes.
	results := Run(snap, Invariants())
	failures := Failures(results)
	for _, cr := range failures {
		t.Errorf("[%s] FALHOU cross-project (%s): %s — %s",
			cr.Invariant.ID, cr.Invariant.Level,
			cr.Invariant.Description, cr.Result.Reason)
	}

	warnings := Warnings(results)
	for _, cr := range warnings {
		t.Logf("[%s] AVISO cross-project (best-effort): %s — %s",
			cr.Invariant.ID, cr.Invariant.Description, cr.Result.Reason)
	}

	t.Logf("cross-project parity: %d invariantes verificados, %d falhas, %d avisos",
		len(results), len(failures), len(warnings))
}

// TestE2EParity_CrossProject_AllTools instala todas as 4 CLIs em um repo temporário
// e valida a matriz de invariantes para todas.
// RF-18: paridade 4x4 cross-project.
func TestE2EParity_CrossProject_AllTools(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSourceDir(t, sourceDir, []string{"review", "execute-task", "agent-governance"})

	svc := newRealInstallService(t)

	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       skills.AllTools,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("install.Execute (AllTools) falhou: %v", err)
	}

	// Validar invariantes para cada subconjunto de ferramentas.
	toolGroups := [][]skills.Tool{
		{skills.ToolClaude},
		{skills.ToolGemini},
		{skills.ToolCodex},
		{skills.ToolCopilot},
		skills.AllTools,
	}

	for _, tools := range toolGroups {
		tools := tools
		name := toolSubsetName(tools)
		t.Run(name, func(t *testing.T) {
			snap, err := Generate(projectDir, tools, nil, "full")
			if err != nil {
				t.Fatalf("parity.Generate falhou para %s: %v", name, err)
			}

			results := Run(snap, Invariants())
			for _, cr := range Failures(results) {
				t.Errorf("[%s] FALHOU cross-project (%s): %s — %s",
					cr.Invariant.ID, cr.Invariant.Level,
					cr.Invariant.Description, cr.Result.Reason)
			}
		})
	}
}

// TestE2EParity_CrossProject_Idempotency verifica que instalar duas vezes resulta
// em invariantes identicos (idempotencia cross-project — RF-09 + RF-18).
func TestE2EParity_CrossProject_Idempotency(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSourceDir(t, sourceDir, []string{"review", "execute-task"})

	svc := newRealInstallService(t)
	opts := config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	}

	// Primeira instalação.
	if err := svc.Execute(opts); err != nil {
		t.Fatalf("primeira instalacao falhou: %v", err)
	}

	// Segunda instalação (idempotência).
	if err := svc.Execute(opts); err != nil {
		t.Fatalf("segunda instalacao falhou: %v", err)
	}

	// Invariantes devem ser identicos em ambas as instalacoes.
	snap, err := Generate(projectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("parity.Generate falhou: %v", err)
	}

	results := Run(snap, Invariants())
	for _, cr := range Failures(results) {
		t.Errorf("[%s] idempotencia violada (%s): %s — %s",
			cr.Invariant.ID, cr.Invariant.Level,
			cr.Invariant.Description, cr.Result.Reason)
	}
}

// ── Paridade de fallback launcher (RF-19) — testes E2E ──────────────────────

// TestE2EParity_FallbackLauncher_ArgvIdentical valida RF-19 end-to-end:
// para cada uma das 4 specs, o argv resolvido via fallback (binário direto ausente)
// é idêntico ao que seria resolvido se o fallback fosse o único launcher disponível.
// Usa LookPather fake — sem dependência de binário real no PATH (techspec §"Testes E2E").
func TestE2EParity_FallbackLauncher_ArgvIdentical(t *testing.T) {
	const npxPath = "/usr/local/bin/npx"

	tests := []struct {
		name     string
		spec     specs.Spec
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "claude_rf19",
			spec:     specs.Claude(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.ClaudeNpmPackage + "@" + specs.ClaudeNpmVersion},
		},
		{
			name:     "codex_rf19",
			spec:     specs.Codex(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.CodexNpmPackage + "@" + specs.CodexNpmVersion},
		},
		{
			name:     "gemini_rf19",
			spec:     specs.Gemini(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.GeminiNpmPackage + "@" + specs.GeminiNpmVersion, "--acp"},
		},
		{
			name:     "copilot_rf19",
			spec:     specs.Copilot(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.CopilotNpmPackage + "@" + specs.CopilotNpmVersion, "--acp"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sp := tc.spec
			sp.ID = tc.name + "-e2e"

			// Apenas npx disponível — binário direto ausente (RF-19).
			look := newFakeLookPather(map[string]string{"npx": npxPath})

			launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
			if err != nil {
				t.Fatalf("probe.EnsureAvailable falhou (RF-19): %v", err)
			}
			if launcher.Kind() != "binary" {
				t.Errorf("kind = %q, want \"binary\" (ADR-017: BinaryLauncher genérico)", launcher.Kind())
			}
			cmd, args := launcher.Command()
			if cmd != tc.wantCmd {
				t.Errorf("command = %q, want %q", cmd, tc.wantCmd)
			}
			if !slices.Equal(args, tc.wantArgs) {
				t.Errorf("args = %v, want %v (FixedArgs preservados literalmente — ADR-017)", args, tc.wantArgs)
			}
		})
	}
}

// TestE2EParity_FallbackLauncher_ResultIdenticalToDirectBinary valida que o resultado
// (argv resolvido) ao usar fallback é idêntico ao resultado ao usar o binário direto,
// exceto pelo comando (path do binário vs path do npx).
// RF-19: resultado idêntico = mesmos FixedArgs semanticamente aplicados.
func TestE2EParity_FallbackLauncher_ResultIdenticalToDirectBinary(t *testing.T) {
	tests := []struct {
		name           string
		spec           specs.Spec
		directBinary   string
		directPath     string
		fallbackCmd    string
		fallbackPath   string
		// FixedArgs esperados no launcher direto (spec.FixedArgs).
		directFixedArgs []string
		// FixedArgs esperados no fallback (spec.Fallbacks[0].FixedArgs).
		fallbackFixedArgs []string
	}{
		{
			name:              "claude",
			spec:              specs.Claude(),
			directBinary:      "claude-agent-acp",
			directPath:        "/usr/local/bin/claude-agent-acp",
			fallbackCmd:       "npx",
			fallbackPath:      "/usr/local/bin/npx",
			directFixedArgs:   specs.Claude().FixedArgs,
			fallbackFixedArgs: specs.Claude().Fallbacks[0].FixedArgs,
		},
		{
			name:              "codex",
			spec:              specs.Codex(),
			directBinary:      "codex-acp",
			directPath:        "/usr/local/bin/codex-acp",
			fallbackCmd:       "npx",
			fallbackPath:      "/usr/local/bin/npx",
			directFixedArgs:   specs.Codex().FixedArgs,
			fallbackFixedArgs: specs.Codex().Fallbacks[0].FixedArgs,
		},
		{
			name:              "gemini",
			spec:              specs.Gemini(),
			directBinary:      "gemini",
			directPath:        "/usr/local/bin/gemini",
			fallbackCmd:       "npx",
			fallbackPath:      "/usr/local/bin/npx",
			directFixedArgs:   specs.Gemini().FixedArgs,
			fallbackFixedArgs: specs.Gemini().Fallbacks[0].FixedArgs,
		},
		{
			name:              "copilot",
			spec:              specs.Copilot(),
			directBinary:      "copilot",
			directPath:        "/usr/local/bin/copilot",
			fallbackCmd:       "npx",
			fallbackPath:      "/usr/local/bin/npx",
			directFixedArgs:   specs.Copilot().FixedArgs,
			fallbackFixedArgs: specs.Copilot().Fallbacks[0].FixedArgs,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// ── Caso 1: binário direto disponível ──
			spDirect := tc.spec
			spDirect.ID = tc.name + "-direct-e2e"
			lookDirect := newFakeLookPather(map[string]string{
				tc.directBinary: tc.directPath,
			})

			launcherDirect, err := probe.EnsureAvailable(context.Background(), spDirect, lookDirect)
			if err != nil {
				t.Fatalf("probe direto falhou: %v", err)
			}
			cmdDirect, argsDirect := launcherDirect.Command()
			if cmdDirect != tc.directPath {
				t.Errorf("direto: command = %q, want %q", cmdDirect, tc.directPath)
			}
			// FixedArgs do direto devem corresponder a spec.FixedArgs.
			if !slices.Equal(argsDirect, tc.directFixedArgs) {
				t.Errorf("direto: args = %v, want %v (spec.FixedArgs)", argsDirect, tc.directFixedArgs)
			}

			// ── Caso 2: apenas fallback disponível ──
			spFallback := tc.spec
			spFallback.ID = tc.name + "-fallback-e2e"
			lookFallback := newFakeLookPather(map[string]string{
				tc.fallbackCmd: tc.fallbackPath,
			})

			launcherFallback, err := probe.EnsureAvailable(context.Background(), spFallback, lookFallback)
			if err != nil {
				t.Fatalf("probe fallback falhou: %v", err)
			}
			cmdFallback, argsFallback := launcherFallback.Command()
			if cmdFallback != tc.fallbackPath {
				t.Errorf("fallback: command = %q, want %q", cmdFallback, tc.fallbackPath)
			}
			// FixedArgs do fallback devem corresponder a spec.Fallbacks[0].FixedArgs.
			if !slices.Equal(argsFallback, tc.fallbackFixedArgs) {
				t.Errorf("fallback: args = %v, want %v (spec.Fallbacks[0].FixedArgs)", argsFallback, tc.fallbackFixedArgs)
			}

			// ── Invariante RF-19: ambos sao "binary" (tipo identico) ──
			if launcherDirect.Kind() != "binary" || launcherFallback.Kind() != "binary" {
				t.Errorf("RF-19: ambos launchers devem ser 'binary': direto=%q, fallback=%q",
					launcherDirect.Kind(), launcherFallback.Kind())
			}

			t.Logf("RF-19 [%s]: direto(%q, %v) | fallback(%q, %v)",
				tc.name, cmdDirect, argsDirect, cmdFallback, argsFallback)
		})
	}
}

// TestE2EParity_FallbackLauncher_BothUnavailable valida que ErrLauncherUnavailable
// e retornado para todas as 4 specs quando nenhum launcher esta disponivel (RF-19).
func TestE2EParity_FallbackLauncher_BothUnavailable(t *testing.T) {
	allSpecs := []specs.Spec{
		specs.Claude(),
		specs.Codex(),
		specs.Gemini(),
		specs.Copilot(),
	}

	for _, sp := range allSpecs {
		sp := sp
		t.Run(sp.ID+"_unavailable", func(t *testing.T) {
			t.Parallel()

			sp.ID = sp.ID + "-unavail-e2e"
			look := newFakeLookPather(map[string]string{}) // nenhum disponivel

			_, err := probe.EnsureAvailable(context.Background(), sp, look)
			if err == nil {
				t.Fatal("esperava ErrLauncherUnavailable, mas nao houve erro")
			}
			if !errors.Is(err, probe.ErrLauncherUnavailable) {
				t.Errorf("esperava probe.ErrLauncherUnavailable, got %T: %v", err, err)
			}
		})
	}
}

// TestE2EParity_Matrix4x4_AllSubsets_CrossProject executa os invariantes 4x4 em
// um repo temporário instalado com todas as 4 CLIs (RF-18 + RF-19 combinados).
func TestE2EParity_Matrix4x4_AllSubsets_CrossProject(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	createMinimalSourceDir(t, sourceDir, []string{"review", "execute-task", "agent-governance"})

	svc := newRealInstallService(t)
	err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       skills.AllTools,
		LinkMode:    skills.LinkCopy,
		GenerateCtx: false,
	})
	if err != nil {
		t.Fatalf("install.Execute (AllTools) falhou: %v", err)
	}

	// Todos os subconjuntos das 4 CLIs.
	subsets := allToolSubsets()
	for _, tools := range subsets {
		tools := tools
		name := toolSubsetName(tools)
		t.Run(name, func(t *testing.T) {
			snap, err := Generate(projectDir, tools, nil, "full")
			if err != nil {
				t.Fatalf("parity.Generate [%s]: %v", name, err)
			}
			results := Run(snap, Invariants())
			for _, cr := range Failures(results) {
				t.Errorf("[%s][%s] FALHOU cross-project (%s): %s — %s",
					name, cr.Invariant.ID, cr.Invariant.Level,
					cr.Invariant.Description, cr.Result.Reason)
			}
		})
	}
}
