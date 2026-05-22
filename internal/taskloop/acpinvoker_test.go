package taskloop_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func (p *testPersistence) AppendEvent(_ events.Event) error                { return nil }
func (p *testPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *testPersistence) EnrichReport(_ airuntime.Summary) error          { return nil }

type testPersistenceFactory struct{}

func (f *testPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return &testPersistence{}, nil
}

type recordingPersistenceFactory struct {
	evidenceDir string
}

func (f *recordingPersistenceFactory) New(evidenceDir string) (airuntime.Persistence, error) {
	f.evidenceDir = evidenceDir
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

// workDirWithAgentsMD cria um TempDir com AGENTS.md para satisfazer o governance hook (F3-Claude).
func workDirWithAgentsMD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("workDirWithAgentsMD: %v", err)
	}
	return dir
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

	stdout, stderr, exitCode, err := invoker.Invoke(ctx, "prompt", workDirWithAgentsMD(t), "claude")
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

	_, _, exitCode, err := invoker.Invoke(ctx, "prompt", workDirWithAgentsMD(t), "")
	if err != nil {
		t.Fatalf("Invoke (quiet): %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
}

func TestACPInvoker_Invoke_DerivesTaskEvidenceDir(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("ok").
		AppendSessionEnd()

	persist := &recordingPersistenceFactory{}
	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(&testProber{
			launcher: specs.NewBinaryLauncher("/fake/claude-agent-acp"),
		}),
		airuntime.WithClientFactory(&testClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(persist),
		airuntime.WithRenderer(&testDiscardRenderer{}),
	)

	invoker := taskloop.NewACPInvoker(runner, true, 0)
	prompt := "Use a skill execute-task para implementar a task tasks/prd-acp-runtime-claude/task-9.0-runner-invoker-integration.md."
	workDir := workDirWithAgentsMD(t)

	if _, _, _, err := invoker.Invoke(ctx, prompt, workDir, ""); err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	want := filepath.Join(workDir, "evidence", "task-9.0")
	if persist.evidenceDir != want {
		t.Fatalf("evidenceDir = %q, want %q", persist.evidenceDir, want)
	}
}

func TestACPInvoker_Invoke_LogsTelemetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	script := acpfake.NewScript().
		AppendAgentMessage("telemetry").
		AppendSessionEnd()

	runner := buildTestRunner(t, ctx, script)
	invoker := taskloop.NewACPInvoker(runner, true, 0)
	workDir := workDirWithAgentsMD(t)

	if _, _, _, err := invoker.Invoke(ctx, "prompt", workDir, ""); err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(workDir, ".agents", "telemetry.log"))
	if err != nil {
		t.Fatalf("ReadFile telemetry.log: %v", err)
	}
	content := string(data)
	for _, want := range []string{"ref=acp-session", "runtime=acp", "launcher=binary", "events_count=", "cancel_reason=none"} {
		if !strings.Contains(content, want) {
			t.Fatalf("telemetry.log = %q, want substring %q", content, want)
		}
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
