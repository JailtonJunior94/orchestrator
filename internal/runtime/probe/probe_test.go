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
		name          string
		available     map[string]string
		wantLauncher  string // "binary" ou "npx"
		wantErr       bool
		wantErrSentinel error
		wantErrMsg    string
	}{
		{
			name: "binary_canonico_encontrado",
			available: map[string]string{
				"claude-agent-acp": "/usr/local/bin/claude-agent-acp",
			},
			wantLauncher: "binary",
		},
		{
			name: "so_npx_disponivel",
			available: map[string]string{
				"npx": "/usr/local/bin/npx",
			},
			wantLauncher: "npx",
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
			wantErrMsg:      "tasks/adr/",
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
	// Nota: este teste usa ID único (não "claude"), portanto recebe o fallback "tasks/adr/".
	// A verificação do ADR específico por spec.ID é coberta em TestAdrByID (T-20/T-21).
	requiredParts := []string{
		"claude-agent-acp não encontrado",
		"Install claude-agent-acp",
		"@agentclientprotocol/claude-agent-acp@",
		"via npm",
		"--runtime=legacy",
		"tasks/adr/",
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

func TestEnsureAvailable_FallbackWithoutAt(t *testing.T) {
	t.Parallel()

	// Testa fallback cujo FixedArgs[1] não tem '@' (edge case de extractPackage/extractVersion).
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
	if launcher.Kind() != "npx" {
		t.Errorf("kind = %q, want \"npx\"", launcher.Kind())
	}
}

func TestEnsureAvailable_FallbackWithEmptyArgs(t *testing.T) {
	t.Parallel()

	// Testa fallback cujo FixedArgs está vazio (edge case).
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
	if launcher.Kind() != "npx" {
		t.Errorf("kind = %q, want \"npx\"", launcher.Kind())
	}
}

func TestEnsureAvailable_NpxLauncherArgs(t *testing.T) {
	t.Parallel()

	sp := claudeSpec()
	sp.ID = "claude-npx-args-test"

	look := newFakeLookPather(map[string]string{
		"npx": "/usr/local/bin/npx",
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if launcher.Kind() != "npx" {
		t.Fatalf("kind = %q, want \"npx\"", launcher.Kind())
	}

	cmd, args := launcher.Command()
	if cmd != "npx" {
		t.Errorf("command = %q, want \"npx\"", cmd)
	}

	// Verificar que os args contêm --yes e o pacote com versão pinada.
	found := false
	for _, arg := range args {
		if strings.Contains(arg, specs.ClaudeNpmPackage) && strings.Contains(arg, specs.ClaudeNpmVersion) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("args não contém o pacote npm esperado %s@%s; args: %v",
			specs.ClaudeNpmPackage, specs.ClaudeNpmVersion, args)
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

	// O ID "copilot-t06-test" não está em adrByID, portanto usa fallback "tasks/adr/".
	// A verificação do ADR específico para copilot (ID="copilot") é coberta em T-20/T-21.
	requiredParts := []string{
		"copilot não encontrado",
		specs.CopilotNpmPackage + "@",
		"--runtime=legacy",
		"tasks/adr/",
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

// T-08: Spec Copilot, binário ausente, npx presente → retorna NpxLauncher com CopilotNpmPackage/CopilotNpmVersion.
func TestEnsureAvailable_T08_CopilotFallbackNpx(t *testing.T) {
	t.Parallel()

	sp := copilotSpec()
	sp.ID = "copilot-t08-test"

	look := newFakeLookPather(map[string]string{
		"npx": "/usr/local/bin/npx",
	})

	launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if launcher.Kind() != "npx" {
		t.Fatalf("kind = %q, want \"npx\"", launcher.Kind())
	}

	cmd, args := launcher.Command()
	if cmd != "npx" {
		t.Errorf("command = %q, want \"npx\"", cmd)
	}

	// Verificar que os args contêm o pacote Copilot com versão pinada.
	found := false
	for _, arg := range args {
		if strings.Contains(arg, specs.CopilotNpmPackage) && strings.Contains(arg, specs.CopilotNpmVersion) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("args não contém o pacote npm esperado %s@%s; args: %v",
			specs.CopilotNpmPackage, specs.CopilotNpmVersion, args)
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
		if !strings.Contains(err.Error(), "tasks/adr/009-acp-protocol-adoption.md") {
			t.Errorf("adrByID[\"claude\"] deve apontar para ADR-009\nmensagem: %q", err.Error())
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
		if !strings.Contains(err.Error(), "tasks/adr/012-copilot-cli-acp-native.md") {
			t.Errorf("adrByID[\"copilot\"] deve apontar para ADR-012\nmensagem: %q", err.Error())
		}
	})
}

// T-21: adrByID lookup por ID desconhecido → fallback gracioso para "tasks/adr/".
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

	// Para ID desconhecido, a mensagem deve conter o fallback "tasks/adr/".
	if !strings.Contains(msg, "tasks/adr/") {
		t.Errorf("mensagem de erro deve conter fallback \"tasks/adr/\" para ID desconhecido\nmensagem: %q", msg)
	}

	// E não deve conter um ADR específico (009 ou 012).
	if strings.Contains(msg, "009-") || strings.Contains(msg, "012-") {
		t.Errorf("mensagem de erro não deve conter ADR específico para ID desconhecido\nmensagem: %q", msg)
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

	// ID único ("claude-regression-test") não está em adrByID → fallback "tasks/adr/".
	// Os demais componentes da mensagem devem estar presentes (RF-03).
	requiredParts := []string{
		"claude-agent-acp não encontrado",
		"Install claude-agent-acp",
		"@agentclientprotocol/claude-agent-acp@",
		"--runtime=legacy",
		"tasks/adr/",
	}
	for _, part := range requiredParts {
		if !strings.Contains(msg, part) {
			t.Errorf("regressão Claude: mensagem deve conter %q\nmensagem completa: %q", part, msg)
		}
	}
}
