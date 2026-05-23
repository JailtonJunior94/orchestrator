package probe_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// fakeLookPather é um mock de LookPather para testes.
type fakeLookPather struct {
	available map[string]string // nome -> path resolvido
	calls     atomic.Int64
}

func newFakeLookPather(available map[string]string) *fakeLookPather {
	return &fakeLookPather{available: available}
}

func (f *fakeLookPather) LookPath(name string) (string, error) {
	f.calls.Add(1)
	if path, ok := f.available[name]; ok {
		return path, nil
	}
	return "", errors.New("not found: " + name)
}

// claudeSpec retorna uma spec de teste baseada na spec Claude.
func claudeSpec() specs.Spec {
	return specs.Claude()
}

func TestEnsureAvailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		available       map[string]string
		wantLauncher    string // "binary" ou "npx"
		wantErr         bool
		wantErrSentinel error
		wantErrMsg      string
	}{
		{
			name: "binary_canonico_encontrado",
			available: map[string]string{
				"claude-agent-acp": "/usr/local/bin/claude-agent-acp",
			},
			wantLauncher: "binary",
		},
		{
			// ADR-017: fallback genérico — npx é um BinaryLauncher com FixedArgs literais.
			name: "so_npx_disponivel",
			available: map[string]string{
				"npx": "/usr/local/bin/npx",
			},
			wantLauncher: "binary",
		},
		{
			name:            "nenhum_launcher_disponivel",
			available:       map[string]string{},
			wantErr:         true,
			wantErrSentinel: airuntime.ErrLauncherUnavailable,
			wantErrMsg:      "claude-agent-acp não encontrado",
		},
		{
			name:            "mensagem_contem_tres_remedios",
			available:       map[string]string{},
			wantErr:         true,
			wantErrSentinel: airuntime.ErrLauncherUnavailable,
			wantErrMsg:      "OR use --runtime=legacy",
		},
		{
			name:            "mensagem_contem_referencia_adr",
			available:       map[string]string{},
			wantErr:         true,
			wantErrSentinel: airuntime.ErrLauncherUnavailable,
			wantErrMsg:      ".specs/adr/",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Cada sub-teste usa uma spec com ID diferente para evitar colisão de cache.
			sp := claudeSpec()
			sp.ID = "claude-test-" + tc.name
			look := newFakeLookPather(tc.available)

			launcher, err := probe.EnsureAvailable(context.Background(), sp, look)

			if tc.wantErr {
				if err == nil {
					t.Fatal("esperava erro, mas não houve")
				}
				if tc.wantErrSentinel != nil && !errors.Is(err, tc.wantErrSentinel) {
					t.Errorf("erro esperado Is(%v), got %v", tc.wantErrSentinel, err)
				}
				if tc.wantErrMsg != "" && !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("mensagem de erro deve conter %q, got %q", tc.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if launcher.Kind() != tc.wantLauncher {
				t.Errorf("kind = %q, want %q", launcher.Kind(), tc.wantLauncher)
			}
		})
	}
}

func TestEnsureAvailable_Cache(t *testing.T) {
	t.Parallel()

	// Usa ID único para isolar do cache de outros testes.
	sp := claudeSpec()
	sp.ID = "claude-cache-test"

	look := newFakeLookPather(map[string]string{
		"claude-agent-acp": "/usr/local/bin/claude-agent-acp",
	})

	// Limpa cache antes do teste de caching para garantir estado limpo.
	probe.ResetCache()

	// Primeira chamada: deve resolver.
	launcher1, err1 := probe.EnsureAvailable(context.Background(), sp, look)
	if err1 != nil {
		t.Fatalf("primeira chamada falhou: %v", err1)
	}

	// Segunda chamada com a mesma spec.
	launcher2, err2 := probe.EnsureAvailable(context.Background(), sp, look)
	if err2 != nil {
		t.Fatalf("segunda chamada falhou: %v", err2)
	}

	// Ambas devem retornar o mesmo launcher kind.
	if launcher1.Kind() != launcher2.Kind() {
		t.Errorf("launcher kinds divergem: %q vs %q", launcher1.Kind(), launcher2.Kind())
	}

	// LookPath deve ter sido invocado apenas uma vez (cache ativo).
	if got := look.calls.Load(); got != 1 {
		t.Errorf("LookPath foi chamado %d vez(es); esperava exatamente 1", got)
	}
}

func TestEnsureAvailable_ContextCanceled(t *testing.T) {
	t.Parallel()

	// Usa ID único para isolar do cache de outros testes.
	sp := claudeSpec()
	sp.ID = "claude-ctx-cancel-test"

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelar antes da chamada

	look := newFakeLookPather(map[string]string{
		"claude-agent-acp": "/usr/local/bin/claude-agent-acp",
	})

	_, err := probe.EnsureAvailable(ctx, sp, look)
	if err == nil {
		t.Fatal("esperava erro de contexto cancelado, mas não houve")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("esperava context.Canceled, got %v", err)
	}
}

func TestEnsureAvailable_ErrorMessageExact(t *testing.T) {
	t.Parallel()

	sp := claudeSpec()
	sp.ID = "claude-msg-exact-test"

	look := newFakeLookPather(map[string]string{})

	_, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err == nil {
		t.Fatal("esperava erro, mas não houve")
	}

	msg := err.Error()

	// Verificar todos os componentes obrigatórios da mensagem de erro (RF-03).
	// Nota: este teste usa ID único (não "claude"), portanto recebe o fallback ".specs/adr/".
	// A verificação do ADR específico por spec.ID é coberta em TestAdrByID (T-20/T-21).
	requiredParts := []string{
		"claude-agent-acp não encontrado",
		"Install claude-agent-acp",
		"@agentclientprotocol/claude-agent-acp@",
		"via npm",
		"--runtime=legacy",
		".specs/adr/",
	}

	for _, part := range requiredParts {
		if !strings.Contains(msg, part) {
			t.Errorf("mensagem de erro deve conter %q\nmensagem completa: %q", part, msg)
		}
	}
}

func TestEnsureAvailable_BinaryLauncherCommand(t *testing.T) {
	t.Parallel()

	sp := claudeSpec()
	sp.ID = "claude-binary-cmd-test"

	const expectedPath = "/custom/path/claude-agent-acp"
	look := newFakeLookPather(map[string]string{
		"claude-agent-acp": expectedPath,
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	cmd, _ := launcher.Command()
	if cmd != expectedPath {
		t.Errorf("command = %q, want %q", cmd, expectedPath)
	}
}

// TestEnsureAvailable_FallbackWithoutAt testa fallback cujo FixedArgs não tem '@' de versão npm.
// ADR-017: fallback genérico — FixedArgs são preservados literalmente, sem parsing especial.
func TestEnsureAvailable_FallbackWithoutAt(t *testing.T) {
	t.Parallel()

	sp := specs.Spec{
		ID:      "test-no-at-fallback",
		Command: "nonexistent-binary",
		Fallbacks: []specs.FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{"--yes", "some-package-no-version"},
			},
		},
	}

	look := newFakeLookPather(map[string]string{
		"npx": "/usr/local/bin/npx",
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// ADR-017: fallback é materializado como BinaryLauncher (kind="binary"), não NpxLauncher.
	if launcher.Kind() != "binary" {
		t.Errorf("kind = %q, want \"binary\"", launcher.Kind())
	}
	// FixedArgs literais preservados.
	cmd, args := launcher.Command()
	if cmd != "/usr/local/bin/npx" {
		t.Errorf("command = %q, want \"/usr/local/bin/npx\"", cmd)
	}
	if !slices.Equal(args, []string{"--yes", "some-package-no-version"}) {
		t.Errorf("args = %v, want [\"--yes\", \"some-package-no-version\"]", args)
	}
}

// TestEnsureAvailable_FallbackWithEmptyArgs testa fallback cujo FixedArgs está vazio.
// ADR-017: FixedArgs vazios são preservados — o launcher é iniciado sem args extras.
func TestEnsureAvailable_FallbackWithEmptyArgs(t *testing.T) {
	t.Parallel()

	sp := specs.Spec{
		ID:      "test-empty-args-fallback",
		Command: "nonexistent-binary",
		Fallbacks: []specs.FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{},
			},
		},
	}

	look := newFakeLookPather(map[string]string{
		"npx": "/usr/local/bin/npx",
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// ADR-017: BinaryLauncher genérico, não NpxLauncher.
	if launcher.Kind() != "binary" {
		t.Errorf("kind = %q, want \"binary\"", launcher.Kind())
	}
	cmd, args := launcher.Command()
	if cmd != "/usr/local/bin/npx" {
		t.Errorf("command = %q, want \"/usr/local/bin/npx\"", cmd)
	}
	if len(args) != 0 {
		t.Errorf("args = %v, want []", args)
	}
}

// TestEnsureAvailable_FallbackLauncherArgs valida que o fallback genérico (ADR-017)
// preserva Command e FixedArgs literalmente — sem parsing especial de "@pkg@ver".
// Para a spec Claude, o fallback é {Command:"npx", FixedArgs:["--yes", "@agentclientprotocol/claude-agent-acp@<ver>"]}.
func TestEnsureAvailable_FallbackLauncherArgs(t *testing.T) {
	t.Parallel()

	sp := claudeSpec()
	sp.ID = "claude-fallback-args-test"

	const npxPath = "/usr/local/bin/npx"
	look := newFakeLookPather(map[string]string{
		"npx": npxPath,
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// ADR-017: fallback materializado como BinaryLauncher (kind="binary").
	if launcher.Kind() != "binary" {
		t.Fatalf("kind = %q, want \"binary\"", launcher.Kind())
	}

	cmd, args := launcher.Command()
	// Command é o path resolvido do binário npx.
	if cmd != npxPath {
		t.Errorf("command = %q, want %q", cmd, npxPath)
	}

	// FixedArgs são os da spec, preservados literalmente: ["--yes", "@agentclientprotocol/claude-agent-acp@<ver>"].
	wantArgs := []string{"--yes", specs.ClaudeNpmPackage + "@" + specs.ClaudeNpmVersion}
	if !slices.Equal(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

// copilotSpec retorna uma spec de teste baseada na spec Copilot.
func copilotSpec() specs.Spec {
	return specs.Copilot()
}

// T-06: Spec Copilot, binário ausente, npx ausente → erro com mensagem completa.
// Critérios: contém "copilot não encontrado", "@github/copilot@...", "--runtime=legacy", "012-copilot-cli-acp-native.md".
func TestEnsureAvailable_T06_CopilotNoBinaryNoNpx(t *testing.T) {
	t.Parallel()

	sp := copilotSpec()
	sp.ID = "copilot-t06-test"

	look := newFakeLookPather(map[string]string{})

	_, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err == nil {
		t.Fatal("esperava erro, mas não houve")
	}
	if !errors.Is(err, airuntime.ErrLauncherUnavailable) {
		t.Errorf("esperava ErrLauncherUnavailable, got %v", err)
	}

	msg := err.Error()

	// O ID "copilot-t06-test" não está em adrByID, portanto usa fallback ".specs/adr/".
	// A verificação do ADR específico para copilot (ID="copilot") é coberta em T-20/T-21.
	requiredParts := []string{
		"copilot não encontrado",
		specs.CopilotNpmPackage + "@",
		"--runtime=legacy",
		".specs/adr/",
	}
	for _, part := range requiredParts {
		if !strings.Contains(msg, part) {
			t.Errorf("mensagem de erro deve conter %q\nmensagem completa: %q", part, msg)
		}
	}
}

// T-07: Spec Copilot, binário presente → retorna BinaryLauncher("copilot", ...) com FixedArgs.
// Verifica que spec.FixedArgs (["--acp"]) são incluídos nos args do launcher (RF-01).
// Bug fix: sem FixedArgs, copilot seria iniciado sem --acp em modo legado (não ACP server).
func TestEnsureAvailable_T07_CopilotBinaryPresent(t *testing.T) {
	t.Parallel()

	sp := copilotSpec()
	sp.ID = "copilot-t07-test"

	const expectedPath = "/usr/local/bin/copilot"
	look := newFakeLookPather(map[string]string{
		"copilot": expectedPath,
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if launcher.Kind() != "binary" {
		t.Errorf("kind = %q, want \"binary\"", launcher.Kind())
	}

	cmd, args := launcher.Command()
	if cmd != expectedPath {
		t.Errorf("command = %q, want %q", cmd, expectedPath)
	}

	// Verificar que spec.FixedArgs estão incluídos nos args do launcher.
	// Para Copilot, FixedArgs = ["--acp"]; sem isso o binário roda em modo legado.
	wantArgs := specs.Copilot().FixedArgs
	for _, wantArg := range wantArgs {
		if !slices.Contains(args, wantArg) {
			t.Errorf("args %v não contém FixedArg %q (spec.FixedArgs devem ser incluídos pelo probe)", args, wantArg)
		}
	}
}

// T-08: Spec Copilot, binário ausente, npx presente → retorna BinaryLauncher genérico (ADR-017).
// FixedArgs do fallback incluem "--acp" para manter paridade com o binário direto (RF-05).
func TestEnsureAvailable_T08_CopilotFallbackNpx(t *testing.T) {
	t.Parallel()

	sp := copilotSpec()
	sp.ID = "copilot-t08-test"

	const npxPath = "/usr/local/bin/npx"
	look := newFakeLookPather(map[string]string{
		"npx": npxPath,
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	// ADR-017: BinaryLauncher genérico, não NpxLauncher.
	if launcher.Kind() != "binary" {
		t.Fatalf("kind = %q, want \"binary\"", launcher.Kind())
	}

	cmd, args := launcher.Command()
	if cmd != npxPath {
		t.Errorf("command = %q, want %q", cmd, npxPath)
	}

	// FixedArgs do fallback Copilot: ["--yes", "@github/copilot@<ver>", "--acp"] — preservados literalmente.
	wantArgs := specs.Copilot().Fallbacks[0].FixedArgs
	if !slices.Equal(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

// T-20: adrByID lookup por ID conhecido → mensagem de erro contém path do ADR correto.
// Testado via comportamento de EnsureAvailable; adrByID é var interna ao package.
// Nota: não usa t.Parallel() nos sub-testes que chamam ResetCache para evitar race conditions.
func TestAdrByID_T20_KnownIDs(t *testing.T) {
	// Não paralelo: usa ResetCache() que modifica estado global.

	t.Run("claude_adr009", func(t *testing.T) {
		sp := claudeSpec()
		sp.ID = "claude"
		probe.ResetCache()

		look := newFakeLookPather(map[string]string{})

		_, err := probe.EnsureAvailable(context.Background(), sp, look)
		if err == nil {
			t.Fatal("esperava erro, mas não houve")
		}
		if !strings.Contains(err.Error(), ".specs/adr/009-acp-protocol-adoption.md") {
			t.Errorf("adrByID[\"claude\"] deve apontar para ADR-009\nmensagem: %q", err.Error())
		}
	})

	t.Run("codex_adr013", func(t *testing.T) {
		sp := specs.Codex()
		sp.ID = "codex"
		probe.ResetCache()

		look := newFakeLookPather(map[string]string{})

		_, err := probe.EnsureAvailable(context.Background(), sp, look)
		if err == nil {
			t.Fatal("esperava erro, mas não houve")
		}
		if !strings.Contains(err.Error(), ".specs/adr/013-codex-cli-acp-native.md") {
			t.Errorf("adrByID[\"codex\"] deve apontar para ADR-013\nmensagem: %q", err.Error())
		}
	})

	t.Run("copilot_adr012", func(t *testing.T) {
		sp := copilotSpec()
		sp.ID = "copilot"
		probe.ResetCache()

		look := newFakeLookPather(map[string]string{})

		_, err := probe.EnsureAvailable(context.Background(), sp, look)
		if err == nil {
			t.Fatal("esperava erro, mas não houve")
		}
		if !strings.Contains(err.Error(), ".specs/adr/012-copilot-cli-acp-native.md") {
			t.Errorf("adrByID[\"copilot\"] deve apontar para ADR-012\nmensagem: %q", err.Error())
		}
	})
}

// T-21: adrByID lookup por ID desconhecido → fallback gracioso para ".specs/adr/".
func TestAdrByID_T21_UnknownIDFallback(t *testing.T) {
	t.Parallel()

	sp := specs.Spec{
		ID:      "unknown-runtime-id-t21",
		Command: "unknown-agent",
		Fallbacks: []specs.FallbackLauncher{
			{
				Command:   "npx",
				FixedArgs: []string{"--yes", "@example/agent@1.0.0"},
			},
		},
	}

	look := newFakeLookPather(map[string]string{})

	_, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err == nil {
		t.Fatal("esperava erro, mas não houve")
	}
	if !errors.Is(err, airuntime.ErrLauncherUnavailable) {
		t.Errorf("esperava ErrLauncherUnavailable, got %v", err)
	}

	msg := err.Error()

	// Para ID desconhecido, a mensagem deve conter o fallback ".specs/adr/".
	if !strings.Contains(msg, ".specs/adr/") {
		t.Errorf("mensagem de erro deve conter fallback \".specs/adr/\" para ID desconhecido\nmensagem: %q", msg)
	}

	// E não deve conter um ADR específico (009, 012, 013 ou 015).
	if strings.Contains(msg, "009-") || strings.Contains(msg, "012-") || strings.Contains(msg, "013-") || strings.Contains(msg, "015-") {
		t.Errorf("mensagem de erro não deve conter ADR específico para ID desconhecido\nmensagem: %q", msg)
	}
}

// geminiSpec retorna uma spec de teste baseada na spec Gemini.
func geminiSpec() specs.Spec {
	return specs.Gemini()
}

// TestProbeReferencesADR_Gemini valida que adrByID["gemini"] aponta para ADR-015 (T-13 ext, RF-06).
// Nota: não usa t.Parallel() pois usa ResetCache() que modifica estado global.
func TestProbeReferencesADR_Gemini(t *testing.T) {
	sp := geminiSpec()
	sp.ID = "gemini"
	probe.ResetCache()

	look := newFakeLookPather(map[string]string{})

	_, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err == nil {
		t.Fatal("esperava erro, mas não houve")
	}
	if !strings.Contains(err.Error(), ".specs/adr/015-gemini-cli-acp-native.md") {
		t.Errorf("adrByID[\"gemini\"] deve apontar para ADR-015\nmensagem: %q", err.Error())
	}
}

// TestProbeCacheKey_Gemini valida que o cache key do Gemini funciona corretamente (T-13 ext, RF-29).
// Chamadas subsequentes para a mesma spec.ID retornam resultado em cache sem re-lookup.
func TestProbeCacheKey_Gemini(t *testing.T) {
	t.Parallel()

	sp := geminiSpec()
	sp.ID = "gemini-cache-test"

	look := newFakeLookPather(map[string]string{
		"gemini": "/usr/local/bin/gemini",
	})

	probe.ResetCache()

	launcher1, err1 := probe.EnsureAvailable(context.Background(), sp, look)
	if err1 != nil {
		t.Fatalf("primeira chamada falhou: %v", err1)
	}

	launcher2, err2 := probe.EnsureAvailable(context.Background(), sp, look)
	if err2 != nil {
		t.Fatalf("segunda chamada falhou: %v", err2)
	}

	if launcher1.Kind() != launcher2.Kind() {
		t.Errorf("launcher kinds divergem: %q vs %q", launcher1.Kind(), launcher2.Kind())
	}

	// LookPath deve ter sido invocado apenas uma vez (cache ativo).
	if got := look.calls.Load(); got != 1 {
		t.Errorf("LookPath foi chamado %d vez(es); esperava exatamente 1", got)
	}
}

// Regressão Claude: T-06 equivalente para Claude — mensagem continua com ADR-009 para ID real.
// Nota: não usa t.Parallel() pois usa ResetCache() que modifica estado global.
func TestEnsureAvailable_ClaudeRegressionErrorMessage(t *testing.T) {
	// Usa ID real "claude" para validar lookup em adrByID.
	sp := claudeSpec()
	sp.ID = "claude-regression-test"

	look := newFakeLookPather(map[string]string{})

	_, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err == nil {
		t.Fatal("esperava erro, mas não houve")
	}

	msg := err.Error()

	// ID único ("claude-regression-test") não está em adrByID → fallback ".specs/adr/".
	// Os demais componentes da mensagem devem estar presentes (RF-03).
	requiredParts := []string{
		"claude-agent-acp não encontrado",
		"Install claude-agent-acp",
		"@agentclientprotocol/claude-agent-acp@",
		"--runtime=legacy",
		".specs/adr/",
	}
	for _, part := range requiredParts {
		if !strings.Contains(msg, part) {
			t.Errorf("regressão Claude: mensagem deve conter %q\nmensagem completa: %q", part, msg)
		}
	}
}

// ---------------------------------------------------------------------------
// Testes adicionais ADR-017: cadeia genérica de fallback (Subtask 2.5)
// ---------------------------------------------------------------------------

// TestFallbackChain_GenericLauncher valida que qualquer FallbackLauncher (não só npx) é
// materializado como BinaryLauncher com FixedArgs literais (ADR-017 D-1).
func TestFallbackChain_GenericLauncher(t *testing.T) {
	t.Parallel()

	const bunxPath = "/usr/local/bin/bunx"
	sp := specs.Spec{
		ID:      "generic-fallback-test",
		Command: "nonexistent-acp-binary",
		Fallbacks: []specs.FallbackLauncher{
			{
				Command:   "bunx",
				FixedArgs: []string{"--bun", "@some/acp-agent@2.0.0"},
			},
		},
	}

	look := newFakeLookPather(map[string]string{
		"bunx": bunxPath,
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if launcher.Kind() != "binary" {
		t.Errorf("kind = %q, want \"binary\"", launcher.Kind())
	}
	cmd, args := launcher.Command()
	if cmd != bunxPath {
		t.Errorf("command = %q, want %q", cmd, bunxPath)
	}
	wantArgs := []string{"--bun", "@some/acp-agent@2.0.0"}
	if !slices.Equal(args, wantArgs) {
		t.Errorf("args = %v, want %v", args, wantArgs)
	}
}

// TestFallbackChain_MultipleOrder valida que múltiplos fallbacks são tentados em ordem:
// o primeiro cujo Command exista no PATH vence (ADR-017 D-2).
func TestFallbackChain_MultipleOrder(t *testing.T) {
	t.Parallel()

	const secondPath = "/usr/bin/second-launcher"
	sp := specs.Spec{
		ID:      "multi-fallback-order-test",
		Command: "nonexistent-primary",
		Fallbacks: []specs.FallbackLauncher{
			{Command: "first-launcher", FixedArgs: []string{"--first"}},
			{Command: "second-launcher", FixedArgs: []string{"--second"}},
			{Command: "third-launcher", FixedArgs: []string{"--third"}},
		},
	}

	// Apenas o segundo fallback está disponível.
	look := newFakeLookPather(map[string]string{
		"second-launcher": secondPath,
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if launcher.Kind() != "binary" {
		t.Errorf("kind = %q, want \"binary\"", launcher.Kind())
	}
	cmd, args := launcher.Command()
	if cmd != secondPath {
		t.Errorf("command = %q, want %q", cmd, secondPath)
	}
	if !slices.Equal(args, []string{"--second"}) {
		t.Errorf("args = %v, want [\"--second\"]", args)
	}
}

// TestFallbackChain_CanonicalFirst valida que o binário canônico é sempre tentado antes dos
// fallbacks — mesmo que ambos estejam disponíveis (ADR-017 D-2: canônico primeiro).
func TestFallbackChain_CanonicalFirst(t *testing.T) {
	t.Parallel()

	const canonicalPath = "/usr/local/bin/primary-acp"
	const fallbackPath = "/usr/local/bin/npx"
	sp := specs.Spec{
		ID:        "canonical-first-test",
		Command:   "primary-acp",
		FixedArgs: []string{"--server"},
		Fallbacks: []specs.FallbackLauncher{
			{Command: "npx", FixedArgs: []string{"--yes", "@example/agent@1.0.0"}},
		},
	}

	// Ambos disponíveis — canônico deve vencer.
	look := newFakeLookPather(map[string]string{
		"primary-acp": canonicalPath,
		"npx":         fallbackPath,
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	cmd, args := launcher.Command()
	if cmd != canonicalPath {
		t.Errorf("canônico deve vencer: command = %q, want %q", cmd, canonicalPath)
	}
	if !slices.Equal(args, []string{"--server"}) {
		t.Errorf("args canônico = %v, want [\"--server\"]", args)
	}
}

// TestFallbackChain_ArgvParityPerSpec valida paridade byte-equivalente (RF-05):
// o argv resolvido via fallback genérico é idêntico ao que seria esperado com os
// FixedArgs declarados nas specs atuais (claude/codex/gemini/copilot).
func TestFallbackChain_ArgvParityPerSpec(t *testing.T) {
	t.Parallel()

	const npxPath = "/usr/local/bin/npx"

	tests := []struct {
		name     string
		spec     specs.Spec
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "claude_fallback_argv",
			spec:     specs.Claude(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.ClaudeNpmPackage + "@" + specs.ClaudeNpmVersion},
		},
		{
			name:     "codex_fallback_argv",
			spec:     specs.Codex(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.CodexNpmPackage + "@" + specs.CodexNpmVersion},
		},
		{
			name:     "gemini_fallback_argv",
			spec:     specs.Gemini(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.GeminiNpmPackage + "@" + specs.GeminiNpmVersion, "--acp"},
		},
		{
			name:     "copilot_fallback_argv",
			spec:     specs.Copilot(),
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.CopilotNpmPackage + "@" + specs.CopilotNpmVersion, "--acp"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// ID único para isolar cache entre sub-testes.
			sp := tc.spec
			sp.ID = tc.name + "-parity"

			look := newFakeLookPather(map[string]string{
				"npx": npxPath,
			})

			launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if launcher.Kind() != "binary" {
				t.Errorf("kind = %q, want \"binary\"", launcher.Kind())
			}
			cmd, args := launcher.Command()
			if cmd != tc.wantCmd {
				t.Errorf("command = %q, want %q", cmd, tc.wantCmd)
			}
			if !slices.Equal(args, tc.wantArgs) {
				t.Errorf("args = %v, want %v", args, tc.wantArgs)
			}
		})
	}
}
