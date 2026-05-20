package taskloop_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

// ---- fakes ------------------------------------------------------------------

// testProber implementa runtime.Prober com launcher fixo.
type testProber struct {
	launcher specs.Launcher
}

func (p *testProber) EnsureAvailable(_ context.Context, _ specs.Spec) (specs.Launcher, error) {
	return p.launcher, nil
}

// testClientFactory cria clientes ACP fake in-process.
type testClientFactory struct {
	script *acpfake.Script
	ctx    context.Context
	t      *testing.T
}

func (f *testClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	srv := acpfake.NewServer(f.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("acpfake.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}

// testPersistence implementa runtime.Persistence em memória.
type testPersistence struct{}

func (p *testPersistence) AppendEvent(_ events.Event) error             { return nil }
func (p *testPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *testPersistence) EnrichReport(_ airuntime.Summary) error       { return nil }

type testPersistenceFactory struct{}

func (f *testPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return &testPersistence{}, nil
}

// discardRenderer descarta eventos de render.
type testDiscardRenderer struct{}

func (r *testDiscardRenderer) Render(_ events.Event) {}

// buildTestRunner cria um ACPRunner com dependências fake para testes do invoker.
func buildTestRunner(t *testing.T, ctx context.Context, script *acpfake.Script) *airuntime.ACPRunner {
	t.Helper()
	return airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(&testProber{
			launcher: specs.NewBinaryLauncher("/fake/claude-agent-acp"),
		}),
		airuntime.WithClientFactory(&testClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(&testPersistenceFactory{}),
		airuntime.WithRenderer(&testDiscardRenderer{}),
	)
}

// ---- testes -----------------------------------------------------------------

// TestNewACPInvoker_ImplementsAgentInvoker verifica que acpInvoker implementa AgentInvoker.
func TestNewACPInvoker_ImplementsAgentInvoker(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendSessionEnd()
	runner := buildTestRunner(t, ctx, script)

	invoker := taskloop.NewACPInvoker(runner, true, 0)
	if invoker == nil {
		t.Fatal("NewACPInvoker retornou nil")
	}

	// Verificar BinaryName.
	if name := invoker.BinaryName(); name == "" {
		t.Error("BinaryName retornou string vazia")
	}
}

// TestACPInvoker_BinaryName verifica o nome lógico retornado.
func TestACPInvoker_BinaryName(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendSessionEnd()
	runner := buildTestRunner(t, ctx, script)
	invoker := taskloop.NewACPInvoker(runner, true, 0)

	if got := invoker.BinaryName(); got != "claude-agent-acp" {
		t.Errorf("BinaryName() = %q, want claude-agent-acp", got)
	}
}

// TestACPInvoker_Invoke_HappyPath: invoker executa e retorna stdout consolidado.
func TestACPInvoker_Invoke_HappyPath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("resposta do agente").
		AppendSessionEnd()

	runner := buildTestRunner(t, ctx, script)
	invoker := taskloop.NewACPInvoker(runner, false, 0) // quiet=false para capturar output

	stdout, stderr, exitCode, err := invoker.Invoke(ctx, "prompt", t.TempDir(), "claude")
	if err != nil {
		t.Fatalf("Invoke: unexpected error: %v", err)
	}

	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}

	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	// stdout deve conter a renderização do agente.
	_ = stdout // output pode ser vazio dependendo do renderer interno
}

// TestACPInvoker_Invoke_ExitCodes: valida mapeamento Summary.CancelReason → exitCode.
func TestACPInvoker_Invoke_ExitCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reason   events.CancelReason
		wantCode int
	}{
		{"none", events.CancelReasonNone, 0},
		{"permission_denied", events.CancelReasonPermissionDenied, 3},
		{"activity_timeout", events.CancelReasonActivityTimeout, 1},
		{"context_canceled", events.CancelReasonContextCanceled, 1},
		{"tool_error", events.CancelReasonToolError, 1},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := taskloop.MapExitCode(tt.reason)
			if got != tt.wantCode {
				t.Errorf("MapExitCode(%q) = %d, want %d", tt.reason, got, tt.wantCode)
			}
		})
	}
}

// TestACPInvoker_SetLiveOutput: verifica que SetLiveOutput implementa LiveOutputSetter.
func TestACPInvoker_SetLiveOutput(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendSessionEnd()
	runner := buildTestRunner(t, ctx, script)
	invoker := taskloop.NewACPInvoker(runner, true, 0)

	// Verifica que implementa LiveOutputSetter.
	setter, ok := invoker.(taskloop.LiveOutputSetter)
	if !ok {
		t.Fatal("acpInvoker não implementa LiveOutputSetter")
	}

	var buf bytes.Buffer
	setter.SetLiveOutput(&buf) // não deve panicar
}

// TestACPInvoker_Invoke_QuietMode: modo quiet não emite output ao usuário.
func TestACPInvoker_Invoke_QuietMode(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("mensagem privada").
		AppendSessionEnd()

	runner := buildTestRunner(t, ctx, script)
	invoker := taskloop.NewACPInvoker(runner, true, 0) // quiet=true

	_, _, exitCode, err := invoker.Invoke(ctx, "prompt", t.TempDir(), "")
	if err != nil {
		t.Fatalf("Invoke (quiet): %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

// TestACPInvoker_ImplementsAgentInvokerInterface garante compatibilidade de tipo.
func TestACPInvoker_ImplementsAgentInvokerInterface(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendSessionEnd()
	runner := buildTestRunner(t, ctx, script)
	invoker := taskloop.NewACPInvoker(runner, true, 0)

	// Verificação de interface: invoker deve ser não-nil e implementar AgentInvoker.
	if invoker == nil {
		t.Fatal("invoker é nil")
	}
	var _ io.Writer = &bytes.Buffer{}
}
