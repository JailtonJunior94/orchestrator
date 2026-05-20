package probe_test

import (
	"context"
	"errors"
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
			wantErrMsg:      "tasks/adr/009-acp-protocol-adoption.md",
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
	requiredParts := []string{
		"claude-agent-acp não encontrado",
		"Install claude-agent-acp",
		"@agentclientprotocol/claude-agent-acp@",
		"via npm",
		"--runtime=legacy",
		"tasks/adr/009-acp-protocol-adoption.md",
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
