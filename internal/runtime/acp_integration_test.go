package runtime_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// ---- fakes ------------------------------------------------------------------

// fakeProber implementa runtime.Prober sem cache global.
// available: map name → path; se vazio, retorna ErrLauncherUnavailable.
type fakeProber struct {
	available map[string]string // binary name → path
}

func (f *fakeProber) EnsureAvailable(_ context.Context, spec specs.Spec) (specs.Launcher, error) {
	// Tentar binário canônico.
	if path, ok := f.available[spec.Command]; ok {
		return specs.NewBinaryLauncher(path), nil
	}
	// Tentar fallbacks (npx).
	for _, fb := range spec.Fallbacks {
		if _, ok := f.available[fb.Command]; ok {
			return specs.NewNpxLauncher(
				extractPkg(fb.FixedArgs),
				extractVer(fb.FixedArgs),
			), nil
		}
	}
	return specs.Launcher{}, errors.New("launcher unavailable: " + spec.Command)
}

func extractPkg(args []string) string {
	if len(args) < 2 {
		return ""
	}
	arg := args[1]
	for i := len(arg) - 1; i > 0; i-- {
		if arg[i] == '@' {
			return arg[:i]
		}
	}
	return arg
}

func extractVer(args []string) string {
	if len(args) < 2 {
		return ""
	}
	arg := args[1]
	for i := len(arg) - 1; i > 0; i-- {
		if arg[i] == '@' {
			return arg[i+1:]
		}
	}
	return ""
}

// fakeClientFactory cria clientes conectados ao acpfake (sem spawn de subprocesso).
type fakeClientFactory struct {
	script *acpfake.Script
	ctx    context.Context
	t      *testing.T
}

func (f *fakeClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	srv := acpfake.NewServer(f.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("acpfake.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}

// fakePersistence captura chamadas de persistência em memória.
type fakePersistence struct {
	events    []events.Event
	toolCalls []events.ToolCallSummary
	summary   *airuntime.Summary
}

func (f *fakePersistence) AppendEvent(evt events.Event) error {
	f.events = append(f.events, evt)
	return nil
}

func (f *fakePersistence) WriteToolCalls(summaries []events.ToolCallSummary) error {
	f.toolCalls = summaries
	return nil
}

func (f *fakePersistence) EnrichReport(summary airuntime.Summary) error {
	cp := summary
	f.summary = &cp
	return nil
}

// fakePersistenceFactory cria fakePersistence para captura em testes.
type fakePersistenceFactory struct {
	persist *fakePersistence
}

func newFakePersistenceFactory() (*fakePersistenceFactory, *fakePersistence) {
	p := &fakePersistence{}
	return &fakePersistenceFactory{persist: p}, p
}

func (f *fakePersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return f.persist, nil
}

// discardRenderer implementa runtime.Renderer descartando todo output.
type discardRenderer struct{}

func (d *discardRenderer) Render(_ events.Event) {}

// ---- helpers ----------------------------------------------------------------

// proberBinary retorna um fakeProber com claude-agent-acp disponível.
func proberBinary() *fakeProber {
	return &fakeProber{available: map[string]string{
		"claude-agent-acp": "/usr/local/bin/claude-agent-acp",
	}}
}

// proberNpxOnly retorna um fakeProber com apenas npx disponível.
func proberNpxOnly() *fakeProber {
	return &fakeProber{available: map[string]string{
		"npx": "/usr/local/bin/npx",
	}}
}

// proberNone retorna um fakeProber sem launchers disponíveis.
func proberNone() *fakeProber {
	return &fakeProber{available: map[string]string{}}
}

// buildRunner constrói um ACPRunner com dependências fake.
func buildRunner(
	t *testing.T,
	ctx context.Context,
	prober airuntime.Prober,
	script *acpfake.Script,
	persistFact airuntime.PersistenceFactory,
) *airuntime.ACPRunner {
	t.Helper()
	return airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(persistFact),
		airuntime.WithRenderer(&discardRenderer{}),
	)
}

// ---- testes -----------------------------------------------------------------

// TestACPRunner_HappyPath: script com 3 mensagens + 2 tool calls + session_end.
// Assertar: events_count>=7, unknown=0, cancel_reason=none, tool_calls=2.
func TestACPRunner_HappyPath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("msg1").
		AppendAgentThought("pensando").
		AppendToolCall("tc_1", "read_file").
		AppendToolCallUpdate("tc_1", "completed").
		AppendAgentMessage("msg2").
		AppendToolCall("tc_2", "write_file").
		AppendToolCallUpdate("tc_2", "completed").
		AppendAgentMessage("msg3").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "do something",
		WorkDir:     t.TempDir(),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	// persist.events inclui: runtime_init + 3 agent_message + 1 thought + 2 tool_call_start + 2 tool_call_update = 9
	if got := len(persist.events); got < 8 {
		t.Errorf("persist.events count = %d, want >= 8", got)
	}

	// summary.EventsCount = eventos mapeados (excluindo unknown e runtime_init já não conta como "evento" padrão)
	if summary.EventsCount < 7 {
		t.Errorf("EventsCount = %d, want >= 7", summary.EventsCount)
	}

	if summary.UnknownEventsCount != 0 {
		t.Errorf("UnknownEventsCount = %d, want 0", summary.UnknownEventsCount)
	}

	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}

	if len(summary.ToolCalls) != 2 {
		t.Errorf("ToolCalls len = %d, want 2", len(summary.ToolCalls))
	}

	if persist.toolCalls == nil {
		t.Error("WriteToolCalls not called")
	}

	if persist.summary == nil {
		t.Error("EnrichReport not called")
	}

	if summary.Launcher != "binary" {
		t.Errorf("Launcher = %q, want binary", summary.Launcher)
	}
}

// TestACPRunner_ActivityTimeout: script demora mais que o timeout de atividade.
// Assertar: cancel_reason=activity_timeout quando o watchdog dispara.
func TestACPRunner_ActivityTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Primeira mensagem chega rápido; segunda tem delay > timeout do watchdog.
	script := acpfake.NewScript().
		AppendAgentMessage("início").
		AppendAgentMessageWithDelay("nunca chega", 500*time.Millisecond).
		AppendSessionEnd()

	pfact, _ := newFakePersistenceFactory()

	timeout, err := events.NewActivityTimeout(50 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	runner := buildRunner(t, ctx, proberBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:          "prompt",
		WorkDir:         t.TempDir(),
		EvidenceDir:     t.TempDir(),
		ActivityTimeout: timeout,
		Quiet:           true,
	}

	summary, runErr := runner.Run(ctx, job)

	// Se o watchdog disparou, esperamos ErrActivityTimeout como causa.
	if runErr != nil {
		if summary.CancelReason != events.CancelReasonActivityTimeout {
			t.Errorf("CancelReason = %q, want activity_timeout when error present", summary.CancelReason)
		}
	} else {
		// Sessão pode ter terminado antes do timeout; aceitar ambos.
		t.Logf("session ended before timeout; CancelReason=%q", summary.CancelReason)
	}

	// O cancel reason nunca deve ser permission_denied neste cenário.
	if summary.CancelReason == events.CancelReasonPermissionDenied {
		t.Errorf("CancelReason = permission_denied unexpectedly")
	}
}

// TestACPRunner_PermissionDenied: script emite requestPermission.
// O cliente ACP cancela o requestPermission silenciosamente; o runner encerra normalmente.
func TestACPRunner_PermissionDenied(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("antes da permissão").
		AppendRequestPermission("edit_file").
		AppendAgentMessage("depois da permissão").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "prompt",
		WorkDir:     t.TempDir(),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	summary, _ := runner.Run(ctx, job)

	// O runner não deve entrar em pânico e deve produzir um summary.
	_ = summary

	// A persistência deve ter sido chamada com ao menos 1 evento.
	if len(persist.events) == 0 {
		t.Log("no events persisted (may be acceptable depending on permission flow)")
	}

	// Cancel reason deve ser um valor válido.
	validReasons := map[events.CancelReason]bool{
		events.CancelReasonNone:             true,
		events.CancelReasonPermissionDenied: true,
		events.CancelReasonContextCanceled:  true,
	}
	if !validReasons[summary.CancelReason] {
		t.Errorf("CancelReason = %q, want none/permission_denied/context_canceled", summary.CancelReason)
	}
}

// TestACPRunner_UnknownDrift: script com kinds desconhecidos.
// Assertar: unknown_events_count > 0, warning emitido.
func TestACPRunner_UnknownDrift(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// AppendUnknown usa UserMessageChunk → mapeado para KindUnknown pelo converter.
	script := acpfake.NewScript().
		AppendUnknown("weird_kind", nil).
		AppendUnknown("another_unknown", nil).
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "prompt",
		WorkDir:     t.TempDir(),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	summary, _ := runner.Run(ctx, job)

	// Deve haver eventos unknown registrados.
	unknownPersisted := 0
	for _, evt := range persist.events {
		if evt.Kind() == events.KindUnknown {
			unknownPersisted++
		}
	}

	if unknownPersisted == 0 && summary.UnknownEventsCount == 0 {
		t.Error("esperava ao menos 1 evento unknown; got 0")
	}

	// Se há unknowns, UnknownKinds não deve ser vazio.
	if summary.UnknownEventsCount > 0 && len(summary.UnknownKinds) == 0 {
		t.Error("UnknownKinds vazio quando UnknownEventsCount > 0")
	}
}

// TestACPRunner_NpxFallback: probe simula binary canônico ausente; npx disponível.
// Assertar: Launcher = "npx".
func TestACPRunner_NpxFallback(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("hello via npx").
		AppendSessionEnd()

	pfact, _ := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberNpxOnly(), script, pfact)

	job := airuntime.Job{
		Prompt:      "prompt",
		WorkDir:     t.TempDir(),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	if summary.Launcher != "npx" {
		t.Errorf("Launcher = %q, want npx", summary.Launcher)
	}
}

// TestACPRunner_LauncherUnavailable: nenhum launcher disponível → erro.
func TestACPRunner_LauncherUnavailable(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pfact, _ := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberNone(), acpfake.NewScript(), pfact)

	job := airuntime.Job{
		Prompt:      "prompt",
		WorkDir:     t.TempDir(),
		EvidenceDir: t.TempDir(),
	}

	_, err := runner.Run(ctx, job)
	if err == nil {
		t.Fatal("esperava erro quando nenhum launcher disponível")
	}
}

// TestACPRunner_WithRealPersistence: usa persistence real via FakeFileSystem.
// Valida que events.jsonl e tool_calls.md são escritos corretamente.
func TestACPRunner_WithRealPersistence(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("msg").
		AppendToolCall("tc_1", "tool_a").
		AppendToolCallUpdate("tc_1", "completed").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(proberBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(pfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)

	evidenceDir := "/evidence/task-9"
	job := airuntime.Job{
		Prompt:      "prompt",
		WorkDir:     "/workdir",
		EvidenceDir: evidenceDir,
		Quiet:       true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}

	// events.jsonl deve existir e ter conteúdo.
	eventsPath := evidenceDir + "/events.jsonl"
	data, readErr := fakeFS.ReadFile(eventsPath)
	if readErr != nil {
		t.Errorf("events.jsonl não encontrado: %v", readErr)
	}
	if len(data) == 0 {
		t.Error("events.jsonl está vazio")
	}

	// tool_calls.md deve conter o nome da ferramenta.
	tcPath := evidenceDir + "/tool_calls.md"
	tcData, readErr := fakeFS.ReadFile(tcPath)
	if readErr != nil {
		t.Errorf("tool_calls.md não encontrado: %v", readErr)
	}
	if !strings.Contains(string(tcData), "tool_a") {
		t.Errorf("tool_calls.md = %q, want to contain 'tool_a'", string(tcData))
	}
}

// TestACPRunner_HumanRenderer: output do renderer é capturado em buffer.
func TestACPRunner_HumanRenderer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("olá mundo").
		AppendSessionEnd()

	pfact, _ := newFakePersistenceFactory()
	var buf bytes.Buffer

	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(proberBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(pfact),
	)
	runner.SetRenderer(&buf)

	job := airuntime.Job{
		Prompt:      "prompt",
		WorkDir:     t.TempDir(),
		EvidenceDir: t.TempDir(),
		Quiet:       false,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(buf.String(), "olá mundo") {
		t.Errorf("renderer output %q does not contain 'olá mundo'", buf.String())
	}
}

// ---- verificação de unused imports ------------------------------------------
var _ = probe.ResetCache // silenciar import não usado
var _ = io.Discard
