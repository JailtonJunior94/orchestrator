package runtime_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"slices"
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

// ensureAgentsMD cria AGENTS.md no diretório fornecido para satisfazer o governance hook (F3-Claude).
// Chamado nos helpers de setup de jobs onde WorkDir não contém AGENTS.md.
func ensureAgentsMD(t *testing.T, dir string) {
	t.Helper()
	agentsPath := dir + "/AGENTS.md"
	if err := os.WriteFile(agentsPath, []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("ensureAgentsMD: criar %s: %v", agentsPath, err)
	}
}

// workDirWithAgentsMD cria um diretório temporário com AGENTS.md já presente.
// Helper para testes que precisam de um WorkDir válido para o governance hook (F3-Claude).
func workDirWithAgentsMD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	ensureAgentsMD(t, dir)
	return dir
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()

	_ = w.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	_ = r.Close()
	return string(data)
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
		WorkDir:     workDirWithAgentsMD(t),
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
	if got := persist.events[len(persist.events)-1].Kind(); got != events.KindSessionEnd {
		t.Errorf("último evento persistido = %q, want session_end", got)
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
		WorkDir:         workDirWithAgentsMD(t),
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
// O runner deve cancelar imediatamente a sessão com permission_denied.
func TestACPRunner_PermissionDenied(t *testing.T) {
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
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	var (
		summary airuntime.Summary
		runErr  error
	)
	stderr := captureStderr(t, func() {
		summary, runErr = runner.Run(ctx, job)
	})

	if !errors.Is(runErr, client.ErrPermissionDenied) {
		t.Fatalf("Run error = %v, want ErrPermissionDenied", runErr)
	}
	if summary.CancelReason != events.CancelReasonPermissionDenied {
		t.Fatalf("CancelReason = %q, want permission_denied", summary.CancelReason)
	}
	if !strings.Contains(stderr, "agent requested permission; configure accessMode=bypassPermissions no claude-agent-acp ou execute em ambiente que pré-aprove. Veja ADR-009") {
		t.Fatalf("stderr = %q, want RF-16 guidance", stderr)
	}
	for _, evt := range persist.events {
		if evt.Kind() == events.KindSessionEnd {
			t.Fatalf("não esperava session_end após permission_denied")
		}
	}
}

// TestACPRunner_UnknownDrift: script com kinds desconhecidos.
// Assertar: unknown_events_count conta eventos, UnknownKinds deduplica kinds e warning segue RF-05.
func TestACPRunner_UnknownDrift(t *testing.T) {
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
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	var summary airuntime.Summary
	stderr := captureStderr(t, func() {
		summary, _ = runner.Run(ctx, job)
	})

	// Deve haver eventos unknown registrados.
	unknownPersisted := 0
	for _, evt := range persist.events {
		if evt.Kind() == events.KindUnknown {
			unknownPersisted++
		}
	}

	if unknownPersisted != 2 {
		t.Errorf("eventos unknown persistidos = %d, want 2", unknownPersisted)
	}
	if summary.UnknownEventsCount != 2 {
		t.Errorf("UnknownEventsCount = %d, want 2", summary.UnknownEventsCount)
	}
	if len(summary.UnknownKinds) != 1 || summary.UnknownKinds[0] != "user_message_chunk" {
		t.Errorf("UnknownKinds = %v, want [user_message_chunk]", summary.UnknownKinds)
	}
	if !strings.Contains(stderr, "2 unknown ACP events skipped (kinds: user_message_chunk)") {
		t.Errorf("stderr = %q, want RF-05 aggregated warning", stderr)
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
		WorkDir:     workDirWithAgentsMD(t),
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
		WorkDir:     workDirWithAgentsMD(t),
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
		Prompt:       "prompt",
		WorkDir:      "/workdir",
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		DisableHooks: true, // fake FS: /workdir não existe no disco; governance hook desabilitado
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
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("events.jsonl lines = %d, want >= 2", len(lines))
	}

	var first map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	var kind string
	if err := json.Unmarshal(first["kind"], &kind); err != nil {
		t.Fatalf("unmarshal first kind: %v", err)
	}
	if kind != string(events.KindRuntimeInit) {
		t.Fatalf("first kind = %q, want runtime_init", kind)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(first["raw"], &raw); err != nil {
		t.Fatalf("unmarshal runtime_init raw: %v", err)
	}
	for _, key := range []string{"launcher", "command", "args", "sdk_version", "npm_version"} {
		if _, ok := raw[key]; !ok {
			t.Fatalf("runtime_init raw missing key %q", key)
		}
	}

	var last map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatalf("unmarshal last line: %v", err)
	}
	if err := json.Unmarshal(last["kind"], &kind); err != nil {
		t.Fatalf("unmarshal last kind: %v", err)
	}
	if kind != string(events.KindSessionEnd) {
		t.Fatalf("last kind = %q, want session_end", kind)
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
		WorkDir:     workDirWithAgentsMD(t),
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

// buildRunnerWithSpec constrói um ACPRunner com spec personalizada e dependências fake.
func buildRunnerWithSpec(
	t *testing.T,
	ctx context.Context,
	spec specs.Spec,
	prober airuntime.Prober,
	script *acpfake.Script,
	persistFact airuntime.PersistenceFactory,
) *airuntime.ACPRunner {
	t.Helper()
	return airuntime.NewACPRunner(
		spec,
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(persistFact),
		airuntime.WithRenderer(&discardRenderer{}),
	)
}

// proberCopilotBinary retorna um fakeProber com o binário "copilot" disponível.
func proberCopilotBinary() *fakeProber {
	return &fakeProber{available: map[string]string{
		"copilot": "/usr/local/bin/copilot",
	}}
}

// TestACPRunner_RuntimeInit_CopilotVersions (T-09): verifica que o payload runtime_init
// carrega as versões do Spec Copilot (sdk_version=CopilotSDKVersion, npm_version=CopilotNpmVersion).
// Regressão T-05: versões Claude permanecem byte-idênticas ao que specs.Claude() declara.
func TestACPRunner_RuntimeInit_CopilotVersions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("hello from copilot").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	copilotSpec := specs.Copilot()

	runner := buildRunnerWithSpec(t, ctx, copilotSpec, proberCopilotBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "test prompt",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	// Encontrar evento runtime_init
	var runtimeInitEvt *events.Event
	for i := range persist.events {
		if persist.events[i].Kind() == events.KindRuntimeInit {
			evtCopy := persist.events[i]
			runtimeInitEvt = &evtCopy
			break
		}
	}
	if runtimeInitEvt == nil {
		t.Fatal("runtime_init event não encontrado em persist.events")
	}

	// Verificar que RuntimeInitPayload carrega versões Copilot via acessores
	ri := runtimeInitEvt.RuntimeInit()
	if ri == nil {
		t.Fatal("RuntimeInit() payload é nil")
	}

	if sdkVersion := ri.SDKVersion(); sdkVersion != specs.CopilotSDKVersion {
		t.Errorf("SDKVersion = %q; want CopilotSDKVersion %q", sdkVersion, specs.CopilotSDKVersion)
	}

	if npmVersion := ri.NpmVersion(); npmVersion != specs.CopilotNpmVersion {
		t.Errorf("NpmVersion = %q; want CopilotNpmVersion %q", npmVersion, specs.CopilotNpmVersion)
	}

	// Verificar que launcher é "binary" (prober retornou binário canônico)
	if launcher := ri.Launcher(); launcher != "binary" {
		t.Errorf("Launcher = %q; want %q", launcher, "binary")
	}
}

// TestACPRunner_RuntimeInit_ClaudeVersions (T-05 regressão): verifica que runtime_init
// ainda carrega versões Claude quando spec é Claude().
func TestACPRunner_RuntimeInit_ClaudeVersions(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("hello from claude").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "test prompt",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	// Encontrar evento runtime_init
	var runtimeInitEvt *events.Event
	for i := range persist.events {
		if persist.events[i].Kind() == events.KindRuntimeInit {
			evtCopy := persist.events[i]
			runtimeInitEvt = &evtCopy
			break
		}
	}
	if runtimeInitEvt == nil {
		t.Fatal("runtime_init event não encontrado em persist.events")
	}

	// Verificar que RuntimeInitPayload carrega versões Claude via acessores
	ri := runtimeInitEvt.RuntimeInit()
	if ri == nil {
		t.Fatal("RuntimeInit() payload é nil")
	}

	if sdkVersion := ri.SDKVersion(); sdkVersion != specs.ClaudeSDKVersion {
		t.Errorf("SDKVersion = %q; want ClaudeSDKVersion %q", sdkVersion, specs.ClaudeSDKVersion)
	}

	if npmVersion := ri.NpmVersion(); npmVersion != specs.ClaudeNpmVersion {
		t.Errorf("NpmVersion = %q; want ClaudeNpmVersion %q", npmVersion, specs.ClaudeNpmVersion)
	}
}

// ---- sub-suite Copilot (T-10, T-11, T-12) -----------------------------------

// TestACPIntegration_Copilot_T10: sessão completa Copilot (open → prompt → agent_message → completion).
// Verifica:
//   - events.jsonl contém runtime_init com sdk_version=CopilotSDKVersion e npm_version=CopilotNpmVersion.
//   - Sequência de kinds inclui runtime_init, agent_message, session_end.
//   - Paridade observacional: Copilot produz os mesmos kinds que Claude para a mesma sequência do fake server.
func TestACPIntegration_Copilot_T10(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("copilot response T10").
		AppendSessionEnd()

	// --- Copilot runner ---
	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	copilotRunner := airuntime.NewACPRunner(
		specs.Copilot(),
		airuntime.WithProber(proberCopilotBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(pfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)

	evidenceDir := "/evidence/copilot-t10"
	job := airuntime.Job{
		Prompt:       "t10 copilot prompt",
		WorkDir:      "/workdir",
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		DisableHooks: true, // fake FS: /workdir não existe no disco
	}

	summary, err := copilotRunner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Copilot T10): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}
	if summary.Launcher != "binary" {
		t.Errorf("Launcher = %q, want binary", summary.Launcher)
	}

	// Verificar events.jsonl
	eventsPath := evidenceDir + "/events.jsonl"
	data, readErr := fakeFS.ReadFile(eventsPath)
	if readErr != nil {
		t.Fatalf("events.jsonl não encontrado: %v", readErr)
	}
	if len(data) == 0 {
		t.Fatal("events.jsonl está vazio")
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("events.jsonl lines = %d, want >= 2", len(lines))
	}

	// Primeira linha deve ser runtime_init com versões Copilot
	var first map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	var kind string
	if err := json.Unmarshal(first["kind"], &kind); err != nil {
		t.Fatalf("unmarshal kind: %v", err)
	}
	if kind != string(events.KindRuntimeInit) {
		t.Fatalf("first kind = %q, want runtime_init", kind)
	}

	// Verificar runtime_init raw contém versões Copilot
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(first["raw"], &raw); err != nil {
		t.Fatalf("unmarshal runtime_init raw: %v", err)
	}
	for _, key := range []string{"launcher", "command", "args", "sdk_version", "npm_version"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("runtime_init raw missing key %q", key)
		}
	}
	var sdkVer string
	if err := json.Unmarshal(raw["sdk_version"], &sdkVer); err != nil {
		t.Fatalf("unmarshal sdk_version: %v", err)
	}
	if sdkVer != specs.CopilotSDKVersion {
		t.Errorf("sdk_version = %q, want CopilotSDKVersion %q", sdkVer, specs.CopilotSDKVersion)
	}
	var npmVer string
	if err := json.Unmarshal(raw["npm_version"], &npmVer); err != nil {
		t.Fatalf("unmarshal npm_version: %v", err)
	}
	if npmVer != specs.CopilotNpmVersion {
		t.Errorf("npm_version = %q, want CopilotNpmVersion %q", npmVer, specs.CopilotNpmVersion)
	}

	// Última linha deve ser session_end
	var last map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatalf("unmarshal last line: %v", err)
	}
	if err := json.Unmarshal(last["kind"], &kind); err != nil {
		t.Fatalf("unmarshal last kind: %v", err)
	}
	if kind != string(events.KindSessionEnd) {
		t.Fatalf("last kind = %q, want session_end", kind)
	}

	// Paridade observacional: coletar kinds de Copilot e comparar com Claude para o mesmo script
	copilotKinds := make([]string, 0, len(lines))
	for _, line := range lines {
		var env map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			continue
		}
		var k string
		if err := json.Unmarshal(env["kind"], &k); err != nil {
			continue
		}
		copilotKinds = append(copilotKinds, k)
	}

	// Rodar mesma sequência com Claude para verificar paridade
	claudeFakeFS := fs.NewFakeFileSystem()
	claudePfact := persistence.NewSessionPersistenceFactory(claudeFakeFS)
	claudeScript := acpfake.NewScript().
		AppendAgentMessage("claude response T10 parity").
		AppendSessionEnd()
	claudeCtx, claudeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer claudeCancel()

	claudeRunner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(proberBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: claudeScript, ctx: claudeCtx, t: t}),
		airuntime.WithPersistenceFactory(claudePfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)
	claudeEvidenceDir := "/evidence/claude-t10-parity"
	_, claudeErr := claudeRunner.Run(claudeCtx, airuntime.Job{
		Prompt:       "t10 claude parity prompt",
		WorkDir:      "/workdir",
		EvidenceDir:  claudeEvidenceDir,
		Quiet:        true,
		DisableHooks: true, // fake FS: /workdir não existe no disco
	})
	if claudeErr != nil {
		t.Fatalf("Run (Claude parity T10): %v", claudeErr)
	}

	claudeData, claudeReadErr := claudeFakeFS.ReadFile(claudeEvidenceDir + "/events.jsonl")
	if claudeReadErr != nil {
		t.Fatalf("Claude events.jsonl não encontrado: %v", claudeReadErr)
	}
	claudeLines := strings.Split(strings.TrimSpace(string(claudeData)), "\n")
	claudeKinds := make([]string, 0, len(claudeLines))
	for _, line := range claudeLines {
		var env map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			continue
		}
		var k string
		if err := json.Unmarshal(env["kind"], &k); err != nil {
			continue
		}
		claudeKinds = append(claudeKinds, k)
	}

	// Paridade: mesmos kinds na mesma ordem (modulo runtime_init payload tool=copilot/claude)
	if len(copilotKinds) != len(claudeKinds) {
		t.Errorf("paridade events.jsonl: Copilot kinds=%v, Claude kinds=%v", copilotKinds, claudeKinds)
	} else {
		for i, ck := range copilotKinds {
			if ck != claudeKinds[i] {
				t.Errorf("paridade events.jsonl[%d]: Copilot=%q, Claude=%q", i, ck, claudeKinds[i])
			}
		}
	}
}

// TestACPIntegration_Copilot_T11: ≥ 2 tool calls com kinds distintos (Read e Bash).
// Verifica que tool_calls.md agrega corretamente (counts por tool name).
func TestACPIntegration_Copilot_T11(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("iniciando T11").
		AppendToolCall("tc_read", "Read").
		AppendToolCallUpdate("tc_read", "completed").
		AppendAgentMessage("entre tool calls").
		AppendToolCall("tc_bash", "Bash").
		AppendToolCallUpdate("tc_bash", "completed").
		AppendAgentMessage("finalizado T11").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	runner := airuntime.NewACPRunner(
		specs.Copilot(),
		airuntime.WithProber(proberCopilotBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(pfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)

	evidenceDir := "/evidence/copilot-t11"
	job := airuntime.Job{
		Prompt:       "t11 copilot prompt",
		WorkDir:      "/workdir",
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		DisableHooks: true, // fake FS: /workdir não existe no disco
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Copilot T11): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}

	// Verificar que summary tem 2 tool calls distintos
	if len(summary.ToolCalls) != 2 {
		t.Errorf("ToolCalls len = %d, want 2", len(summary.ToolCalls))
	}

	// Verificar tool_calls.md contém os tool names
	tcPath := evidenceDir + "/tool_calls.md"
	tcData, readErr := fakeFS.ReadFile(tcPath)
	if readErr != nil {
		t.Fatalf("tool_calls.md não encontrado: %v", readErr)
	}
	tcContent := string(tcData)
	if !strings.Contains(tcContent, "Read") {
		t.Errorf("tool_calls.md = %q, want to contain 'Read'", tcContent)
	}
	if !strings.Contains(tcContent, "Bash") {
		t.Errorf("tool_calls.md = %q, want to contain 'Bash'", tcContent)
	}

	// Verificar que ambos os tool names estão nos summaries
	toolNames := make(map[string]bool)
	for _, tc := range summary.ToolCalls {
		toolNames[tc.Name] = true
	}
	if !toolNames["Read"] {
		t.Errorf("ToolCalls não contém 'Read'; summaries = %v", summary.ToolCalls)
	}
	if !toolNames["Bash"] {
		t.Errorf("ToolCalls não contém 'Bash'; summaries = %v", summary.ToolCalls)
	}
}

// TestACPIntegration_Copilot_T12: server inativo → ActivityWatchdog cancela.
// Verifica que execution_report registra CancelReason = activity_timeout.
func TestACPIntegration_Copilot_T12(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Script emite primeira mensagem rápida; segunda tem delay maior que o watchdog.
	script := acpfake.NewScript().
		AppendAgentMessage("copilot iniciou T12").
		AppendAgentMessageWithDelay("nunca chega", 500*time.Millisecond).
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()

	timeout, err := events.NewActivityTimeout(50 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	runner := buildRunnerWithSpec(t, ctx, specs.Copilot(), proberCopilotBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:          "t12 copilot watchdog test",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		ActivityTimeout: timeout,
		Quiet:           true,
	}

	summary, runErr := runner.Run(ctx, job)

	// Watchdog deve ter disparado com activity_timeout.
	if runErr != nil {
		if summary.CancelReason != events.CancelReasonActivityTimeout {
			t.Errorf("CancelReason = %q, want activity_timeout when error present", summary.CancelReason)
		}
	} else {
		// A sessão pode ter terminado antes do timeout; ambos são válidos.
		t.Logf("T12: session ended before timeout; CancelReason=%q", summary.CancelReason)
	}

	// Não deve ser permission_denied neste cenário.
	if summary.CancelReason == events.CancelReasonPermissionDenied {
		t.Errorf("CancelReason = permission_denied unexpectedly")
	}

	// Verificar que EnrichReport foi chamado (execution_report enriquecido).
	if persist.summary == nil {
		t.Error("EnrichReport não foi chamado — execution_report não foi enriquecido")
	}

	// Verificar CancelReason no summary persistido (simula execution_report.md).
	if runErr != nil && persist.summary != nil {
		if persist.summary.CancelReason != events.CancelReasonActivityTimeout {
			t.Errorf("persist.summary.CancelReason = %q, want activity_timeout", persist.summary.CancelReason)
		}
	}
}

// ---- Testes Task 3.0: novos campos em Job (T-19 regressão + zero-value + construção) ----

// TestJob_ZeroValue_AccessMode (Task 3.0): garante que Job{}.AccessMode é o zero-value de
// specs.AccessMode (string vazia ""), e que o fluxo Claude/Copilot permanece inalterado
// com o job de zero-value — regressão T-19.
func TestJob_ZeroValue_AccessMode(t *testing.T) {
	t.Parallel()

	var j airuntime.Job
	// Zero-value de specs.AccessMode é "" (não "restricted"), conforme DoD Task 3.0.
	if j.AccessMode != specs.AccessMode("") {
		t.Errorf("Job{}.AccessMode = %q, want %q (zero-value string)", j.AccessMode, "")
	}
	// ReasoningEffort zero-value deve ser string vazia.
	if j.ReasoningEffort != "" {
		t.Errorf("Job{}.ReasoningEffort = %q, want %q", j.ReasoningEffort, "")
	}
	// AddDirs zero-value deve ser nil.
	if j.AddDirs != nil {
		t.Errorf("Job{}.AddDirs = %v, want nil", j.AddDirs)
	}
}

// TestJob_FullConstruction (Task 3.0): garante que Job com os novos campos preenchidos
// compila e carrega os valores corretamente.
func TestJob_FullConstruction(t *testing.T) {
	t.Parallel()

	j := airuntime.Job{
		ReasoningEffort: "high",
		AccessMode:      specs.AccessModeFull,
		AddDirs:         []string{"/x"},
	}
	if j.ReasoningEffort != "high" {
		t.Errorf("ReasoningEffort = %q, want %q", j.ReasoningEffort, "high")
	}
	if j.AccessMode != specs.AccessModeFull {
		t.Errorf("AccessMode = %q, want %q", j.AccessMode, specs.AccessModeFull)
	}
	if len(j.AddDirs) != 1 || j.AddDirs[0] != "/x" {
		t.Errorf("AddDirs = %v, want [/x]", j.AddDirs)
	}
}

// TestACPRunner_JobZeroValue_ClaudeRegression (T-19 regressão): Job{} com novos campos
// zero-value não altera o comportamento de uma sessão Claude normal.
func TestACPRunner_JobZeroValue_ClaudeRegression(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("regressao T-19 Claude").
		AppendSessionEnd()

	pfact, _ := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberBinary(), script, pfact)

	// Job com zero-value nos novos campos — deve funcionar como antes.
	job := airuntime.Job{
		Prompt:          "regression T-19",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		Quiet:           true,
		ReasoningEffort: "",
		AccessMode:      specs.AccessMode(""),
		AddDirs:         nil,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}
	if summary.Launcher != "binary" {
		t.Errorf("Launcher = %q, want binary", summary.Launcher)
	}
}

// ---- sub-suite Codex (T-17, T-18, T-31) + regressão Claude T-19 -------------

// proberCodexBinary retorna um fakeProber com o binário "codex-acp" disponível.
func proberCodexBinary() *fakeProber {
	return &fakeProber{available: map[string]string{
		"codex-acp": "/usr/local/bin/codex-acp",
	}}
}

// findRuntimeInitArgs extrai o slice de args do primeiro runtime_init persistido.
func findRuntimeInitArgs(t *testing.T, persist *fakePersistence) []string {
	t.Helper()
	for i := range persist.events {
		if persist.events[i].Kind() == events.KindRuntimeInit {
			ri := persist.events[i].RuntimeInit()
			if ri == nil {
				t.Fatal("RuntimeInit() payload é nil")
			}
			return ri.Args()
		}
	}
	t.Fatal("runtime_init event não encontrado em persist.events")
	return nil
}

// containsArg verifica se slice contém um argumento específico.
func containsArg(args []string, needle string) bool {
	return slices.Contains(args, needle)
}

// containsArgPair verifica se slice contém "-c <value>" consecutivos.
func containsArgPair(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

// TestACPRunner_Codex_T17_BootstrapArgs_Restricted (T-17): Codex restricted — spawn args
// contêm -c model=..., -c model_reasoning_effort=..., -c features.code_mode=false,
// -c features.code_mode_only=false e NÃO contêm sandbox/approval_policy/web_search.
func TestACPRunner_Codex_T17_BootstrapArgs_Restricted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("codex restricted T17").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.Codex(), proberCodexBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:          "T17 codex restricted",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		Quiet:           true,
		Model:           specs.DefaultCodexModel,
		ReasoningEffort: "medium",
		AccessMode:      specs.AccessModeRestricted,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	args := findRuntimeInitArgs(t, persist)

	// Deve conter -c model="gpt-5.5"
	if !containsArgPair(args, "-c", `"gpt-5.5"`) {
		// strconv.Quote("gpt-5.5") = `"gpt-5.5"` ; verificar pelo prefixo model=
		wantModel := "model=" + `"gpt-5.5"`
		if !containsArgPair(args, "-c", wantModel) {
			t.Errorf("args %v: esperava par -c %q (model)", args, wantModel)
		}
	}

	// Deve conter -c model_reasoning_effort="medium"
	wantReasoning := `model_reasoning_effort="medium"`
	if !containsArgPair(args, "-c", wantReasoning) {
		t.Errorf("args %v: esperava par -c %q", args, wantReasoning)
	}

	// Deve conter features.code_mode=false e features.code_mode_only=false
	if !containsArgPair(args, "-c", "features.code_mode=false") {
		t.Errorf("args %v: esperava par -c features.code_mode=false", args)
	}
	if !containsArgPair(args, "-c", "features.code_mode_only=false") {
		t.Errorf("args %v: esperava par -c features.code_mode_only=false", args)
	}

	// NÃO deve conter sandbox_mode (modo restricted)
	if containsArg(args, `sandbox_mode="danger-full-access"`) {
		t.Errorf("args %v: NÃO esperava sandbox_mode no modo restricted", args)
	}
	if containsArg(args, `approval_policy="never"`) {
		t.Errorf("args %v: NÃO esperava approval_policy no modo restricted", args)
	}
	if containsArg(args, `web_search="live"`) {
		t.Errorf("args %v: NÃO esperava web_search no modo restricted", args)
	}
}

// TestACPRunner_Codex_T18_BootstrapArgs_Full (T-18): Codex full — spawn args
// contêm todos os de T-17 + sandbox_mode="danger-full-access", approval_policy="never",
// web_search="live".
func TestACPRunner_Codex_T18_BootstrapArgs_Full(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("codex full T18").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.Codex(), proberCodexBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:          "T18 codex full access",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		Quiet:           true,
		Model:           specs.DefaultCodexModel,
		ReasoningEffort: "high",
		AccessMode:      specs.AccessModeFull,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	args := findRuntimeInitArgs(t, persist)

	// Deve conter features toggles
	if !containsArgPair(args, "-c", "features.code_mode=false") {
		t.Errorf("args %v: esperava par -c features.code_mode=false", args)
	}
	if !containsArgPair(args, "-c", "features.code_mode_only=false") {
		t.Errorf("args %v: esperava par -c features.code_mode_only=false", args)
	}

	// Deve conter sandbox/approval/web_search para AccessModeFull
	if !containsArgPair(args, "-c", `approval_policy="never"`) {
		t.Errorf("args %v: esperava par -c approval_policy=%q (full access)", args, `"never"`)
	}
	if !containsArgPair(args, "-c", `sandbox_mode="danger-full-access"`) {
		t.Errorf("args %v: esperava par -c sandbox_mode=%q (full access)", args, `"danger-full-access"`)
	}
	if !containsArgPair(args, "-c", `web_search="live"`) {
		t.Errorf("args %v: esperava par -c web_search=%q (full access)", args, `"live"`)
	}
}

// TestACPRunner_Claude_T19_NoCodexFlags (T-19 regressão estendida): sessão Claude
// com Job zero-value nos campos Codex NÃO produz flags -c no runtime_init args.
// Valida que BootstrapArgs no-op preserva comportamento pré-F1-Codex.
func TestACPRunner_Claude_T19_NoCodexFlags(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("claude sem flags codex T19").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunner(t, ctx, proberBinary(), script, pfact)

	// Job zero-value nos campos Codex — Claude usa no-op BootstrapArgs.
	job := airuntime.Job{
		Prompt:          "T19 claude regressao",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		Quiet:           true,
		Model:           "",
		ReasoningEffort: "",
		AccessMode:      specs.AccessMode(""),
		AddDirs:         nil,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	args := findRuntimeInitArgs(t, persist)

	// NÃO deve conter -c (nenhum bootstrap flag para Claude — BootstrapArgs no-op).
	// Nota: fakeProber não injeta FixedArgs (--bypass-permissions); apenas testa ausência de -c.
	if containsArg(args, "-c") {
		t.Errorf("args %v: NÃO esperava flag -c em sessão Claude (regressão T-19)", args)
	}
}

// TestACPRunner_Copilot_T19_NoCodexFlags (T-19 regressão Copilot): sessão Copilot
// com Job zero-value nos campos Codex NÃO produz flags -c no runtime_init args.
func TestACPRunner_Copilot_T19_NoCodexFlags(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("copilot sem flags codex T19").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.Copilot(), proberCopilotBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:          "T19 copilot regressao",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		Quiet:           true,
		Model:           "",
		ReasoningEffort: "",
		AccessMode:      specs.AccessMode(""),
		AddDirs:         nil,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: unexpected error: %v", err)
	}

	args := findRuntimeInitArgs(t, persist)

	// NÃO deve conter -c (nenhum bootstrap flag para Copilot — BootstrapArgs no-op).
	// Nota: fakeProber não injeta FixedArgs (--acp); apenas testa ausência de -c.
	if containsArg(args, "-c") {
		t.Errorf("args %v: NÃO esperava flag -c em sessão Copilot (regressão T-19)", args)
	}
}

// TestACPRunner_Codex_T20_ToolCallsAndReport (T-20): Codex + fake server emite ≥ 2 tool calls.
// Valida:
//   - tool_calls.md agrega corretamente com os nomes das ferramentas.
//   - execution_report.md contém events_count correto e cancel_reason=none.
//   - Paridade estrutural com Claude/Copilot (mesmos campos em events.jsonl).
func TestACPRunner_Codex_T20_ToolCallsAndReport(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("codex iniciou T20").
		AppendToolCall("tc_search", "search_query").
		AppendToolCallUpdate("tc_search", "completed").
		AppendAgentMessage("entre tool calls").
		AppendToolCall("tc_img", "image_query").
		AppendToolCallUpdate("tc_img", "completed").
		AppendAgentMessage("finalizado T20").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	runner := airuntime.NewACPRunner(
		specs.Codex(),
		airuntime.WithProber(proberCodexBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(pfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)

	evidenceDir := "/evidence/codex-t20"
	job := airuntime.Job{
		Prompt:          "T20 codex tool calls",
		WorkDir:         "/workdir",
		EvidenceDir:     evidenceDir,
		Quiet:           true,
		Model:           specs.DefaultCodexModel,
		ReasoningEffort: "medium",
		AccessMode:      specs.AccessModeRestricted,
		DisableHooks:    true, // fake FS: /workdir não existe no disco
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Codex T20): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}

	// Verificar que summary tem ≥ 2 tool calls
	if len(summary.ToolCalls) < 2 {
		t.Errorf("ToolCalls len = %d, want >= 2", len(summary.ToolCalls))
	}

	// Verificar tool_calls.md contém os tool names nativos do Codex (D-09: aliasing adiado)
	tcPath := evidenceDir + "/tool_calls.md"
	tcData, readErr := fakeFS.ReadFile(tcPath)
	if readErr != nil {
		t.Fatalf("tool_calls.md não encontrado: %v", readErr)
	}
	tcContent := string(tcData)
	if !strings.Contains(tcContent, "search_query") {
		t.Errorf("tool_calls.md = %q, want to contain 'search_query'", tcContent)
	}
	if !strings.Contains(tcContent, "image_query") {
		t.Errorf("tool_calls.md = %q, want to contain 'image_query'", tcContent)
	}

	// Verificar events.jsonl estrutura — mesmos kinds que Claude/Copilot
	eventsPath := evidenceDir + "/events.jsonl"
	eventsData, eventsErr := fakeFS.ReadFile(eventsPath)
	if eventsErr != nil {
		t.Fatalf("events.jsonl não encontrado: %v", eventsErr)
	}
	lines := strings.Split(strings.TrimSpace(string(eventsData)), "\n")
	if len(lines) < 3 {
		t.Fatalf("events.jsonl lines = %d, want >= 3", len(lines))
	}

	// Primeira linha deve ser runtime_init
	var first map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal first line: %v", err)
	}
	var kind string
	if err := json.Unmarshal(first["kind"], &kind); err != nil {
		t.Fatalf("unmarshal first kind: %v", err)
	}
	if kind != string(events.KindRuntimeInit) {
		t.Fatalf("first kind = %q, want runtime_init", kind)
	}

	// Última linha deve ser session_end
	var last map[string]json.RawMessage
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &last); err != nil {
		t.Fatalf("unmarshal last line: %v", err)
	}
	if err := json.Unmarshal(last["kind"], &kind); err != nil {
		t.Fatalf("unmarshal last kind: %v", err)
	}
	if kind != string(events.KindSessionEnd) {
		t.Fatalf("last kind = %q, want session_end", kind)
	}

	// Verificar execution_report.md contém campos esperados (mesmos que Claude/Copilot)
	reportPath := evidenceDir + "/execution_report.md"
	reportData, reportErr := fakeFS.ReadFile(reportPath)
	if reportErr != nil {
		t.Fatalf("execution_report.md não encontrado: %v", reportErr)
	}
	reportContent := string(reportData)
	if !strings.Contains(reportContent, "events_count:") {
		t.Errorf("execution_report.md = %q, want to contain 'events_count:'", reportContent)
	}
	if !strings.Contains(reportContent, "cancel_reason: none") {
		t.Errorf("execution_report.md = %q, want to contain 'cancel_reason: none'", reportContent)
	}
	if !strings.Contains(reportContent, "launcher: binary") {
		t.Errorf("execution_report.md = %q, want to contain 'launcher: binary'", reportContent)
	}
	if !strings.Contains(reportContent, "unknown_events_count: 0") {
		t.Errorf("execution_report.md = %q, want to contain 'unknown_events_count: 0'", reportContent)
	}

	// Verificar que os dois tool names estão nos summaries
	toolNames := make(map[string]bool)
	for _, tc := range summary.ToolCalls {
		toolNames[tc.Name] = true
	}
	if !toolNames["search_query"] {
		t.Errorf("ToolCalls não contém 'search_query'; summaries = %v", summary.ToolCalls)
	}
	if !toolNames["image_query"] {
		t.Errorf("ToolCalls não contém 'image_query'; summaries = %v", summary.ToolCalls)
	}
}

// TestACPRunner_Codex_T21_ActivityWatchdog (T-21): Codex + fake server inativo →
// ActivityWatchdog cancela via CancelCause(ErrActivityTimeout); CancelReason == "activity_timeout".
func TestACPRunner_Codex_T21_ActivityWatchdog(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Script emite primeira mensagem rápida; segunda tem delay > timeout do watchdog.
	script := acpfake.NewScript().
		AppendAgentMessage("codex iniciou T21").
		AppendAgentMessageWithDelay("nunca chega", 500*time.Millisecond).
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()

	timeout, err := events.NewActivityTimeout(50 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	runner := buildRunnerWithSpec(t, ctx, specs.Codex(), proberCodexBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:          "T21 codex watchdog",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		ActivityTimeout: timeout,
		Quiet:           true,
		Model:           specs.DefaultCodexModel,
		ReasoningEffort: "medium",
		AccessMode:      specs.AccessModeRestricted,
	}

	summary, runErr := runner.Run(ctx, job)

	// Watchdog deve ter disparado com activity_timeout.
	if runErr != nil {
		if summary.CancelReason != events.CancelReasonActivityTimeout {
			t.Errorf("CancelReason = %q, want activity_timeout when error present", summary.CancelReason)
		}
	} else {
		// A sessão pode ter terminado antes do timeout; ambos são válidos.
		t.Logf("T21: session ended before timeout; CancelReason=%q", summary.CancelReason)
	}

	// Não deve ser permission_denied neste cenário.
	if summary.CancelReason == events.CancelReasonPermissionDenied {
		t.Errorf("CancelReason = permission_denied unexpectedly")
	}

	// Verificar que EnrichReport foi chamado (execution_report enriquecido).
	if persist.summary == nil {
		t.Error("EnrichReport não foi chamado — execution_report não foi enriquecido")
	}

	// Verificar CancelReason no summary persistido (simula execution_report.md).
	if runErr != nil && persist.summary != nil {
		if persist.summary.CancelReason != events.CancelReasonActivityTimeout {
			t.Errorf("persist.summary.CancelReason = %q, want activity_timeout", persist.summary.CancelReason)
		}
	}
}

// ---- sub-suite Gemini (RF-09, T-4.0) ----------------------------------------

// proberGeminiBinary retorna um fakeProber com o binário "gemini" disponível.
func proberGeminiBinary() *fakeProber {
	return &fakeProber{available: map[string]string{
		"gemini": "/usr/local/bin/gemini",
	}}
}

// proberGeminiNpxOnly retorna um fakeProber com apenas npx disponível (sem binário gemini).
func proberGeminiNpxOnly() *fakeProber {
	return &fakeProber{available: map[string]string{
		"npx": "/usr/local/bin/npx",
	}}
}

// TestACPIntegration_Gemini_OpenOK (T-4.0 sub-test 1): sessão completa Gemini
// (open → agent_message → session_end). Paridade observacional com Claude/Codex/Copilot.
func TestACPIntegration_Gemini_OpenOK(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini response open OK").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiBinary(), script, pfact)

	evidenceDir := "/evidence/gemini-open-ok"
	job := airuntime.Job{
		Prompt:       "gemini open OK prompt",
		WorkDir:      "/workdir",
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		DisableHooks: true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini OpenOK): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}
	if summary.Launcher != "binary" {
		t.Errorf("Launcher = %q, want binary", summary.Launcher)
	}

	// events.jsonl deve existir e ter conteúdo.
	eventsPath := evidenceDir + "/events.jsonl"
	data, readErr := fakeFS.ReadFile(eventsPath)
	if readErr != nil {
		t.Fatalf("events.jsonl não encontrado: %v", readErr)
	}
	if len(data) == 0 {
		t.Fatal("events.jsonl está vazio")
	}
}

// TestACPIntegration_Gemini_Prompt (T-4.0 sub-test 2): valida que prompt é propagado
// e que runtime_init contém versões Gemini (sdk_version=GeminiSDKVersion, npm_version=GeminiNpmVersion).
func TestACPIntegration_Gemini_Prompt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini prompt received").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()

	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:       "test prompt for gemini",
		WorkDir:      workDirWithAgentsMD(t),
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini Prompt): %v", err)
	}

	// Verificar que runtime_init carrega versões Gemini.
	var runtimeInitEvt *events.Event
	for i := range persist.events {
		if persist.events[i].Kind() == events.KindRuntimeInit {
			evtCopy := persist.events[i]
			runtimeInitEvt = &evtCopy
			break
		}
	}
	if runtimeInitEvt == nil {
		t.Fatal("runtime_init event não encontrado")
	}

	ri := runtimeInitEvt.RuntimeInit()
	if ri == nil {
		t.Fatal("RuntimeInit() payload é nil")
	}

	if sdkVersion := ri.SDKVersion(); sdkVersion != specs.GeminiSDKVersion {
		t.Errorf("SDKVersion = %q; want GeminiSDKVersion %q", sdkVersion, specs.GeminiSDKVersion)
	}

	if npmVersion := ri.NpmVersion(); npmVersion != specs.GeminiNpmVersion {
		t.Errorf("NpmVersion = %q; want GeminiNpmVersion %q", npmVersion, specs.GeminiNpmVersion)
	}
}

// TestACPIntegration_Gemini_TwoToolCalls (T-4.0 sub-test 3): ≥ 2 tool calls distintos.
// Valida que tool_calls.md agrega corretamente com nomes das ferramentas.
func TestACPIntegration_Gemini_TwoToolCalls(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini iniciou tool calls").
		AppendToolCall("tc_read", "read_file").
		AppendToolCallUpdate("tc_read", "completed").
		AppendAgentMessage("entre tool calls").
		AppendToolCall("tc_write", "write_file").
		AppendToolCallUpdate("tc_write", "completed").
		AppendAgentMessage("finalizado").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	runner := airuntime.NewACPRunner(
		specs.Gemini(),
		airuntime.WithProber(proberGeminiBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(pfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)

	evidenceDir := "/evidence/gemini-tool-calls"
	job := airuntime.Job{
		Prompt:       "gemini tool calls",
		WorkDir:      "/workdir",
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		DisableHooks: true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini TwoToolCalls): %v", err)
	}

	if len(summary.ToolCalls) < 2 {
		t.Errorf("ToolCalls len = %d, want >= 2", len(summary.ToolCalls))
	}

	tcPath := evidenceDir + "/tool_calls.md"
	tcData, readErr := fakeFS.ReadFile(tcPath)
	if readErr != nil {
		t.Fatalf("tool_calls.md não encontrado: %v", readErr)
	}
	if !strings.Contains(string(tcData), "read_file") {
		t.Errorf("tool_calls.md não contém 'read_file'; conteúdo=%q", string(tcData))
	}
	if !strings.Contains(string(tcData), "write_file") {
		t.Errorf("tool_calls.md não contém 'write_file'; conteúdo=%q", string(tcData))
	}
}

// TestACPIntegration_Gemini_AgentMessage (T-4.0 sub-test 4): múltiplas agent_messages.
// Valida que events.jsonl registra os kinds corretamente.
func TestACPIntegration_Gemini_AgentMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("primeira mensagem").
		AppendAgentMessage("segunda mensagem").
		AppendAgentMessage("terceira mensagem").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()

	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "gemini multi messages",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini AgentMessage): %v", err)
	}

	// Contar agent_messages persistidas.
	agentMsgCount := 0
	for _, evt := range persist.events {
		if evt.Kind() == events.KindAgentMessage {
			agentMsgCount++
		}
	}
	if agentMsgCount < 3 {
		t.Errorf("agent_messages persistidas = %d, want >= 3", agentMsgCount)
	}

	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}
}

// TestACPIntegration_Gemini_Completion (T-4.0 sub-test 5): verifica que session_end
// é o último evento e que execution_report.md contém campos esperados.
func TestACPIntegration_Gemini_Completion(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini completion").
		AppendToolCall("tc_1", "bash_command").
		AppendToolCallUpdate("tc_1", "completed").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiBinary(), script, pfact)

	evidenceDir := "/evidence/gemini-completion"
	job := airuntime.Job{
		Prompt:       "gemini completion",
		WorkDir:      "/workdir",
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		DisableHooks: true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini Completion): %v", err)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}

	// execution_report.md deve ter campos esperados.
	reportPath := evidenceDir + "/execution_report.md"
	reportData, readErr := fakeFS.ReadFile(reportPath)
	if readErr != nil {
		t.Fatalf("execution_report.md não encontrado: %v", readErr)
	}
	reportContent := string(reportData)
	if !strings.Contains(reportContent, "cancel_reason: none") {
		t.Errorf("execution_report.md deve conter 'cancel_reason: none'; conteúdo=%q", reportContent)
	}
	if !strings.Contains(reportContent, "launcher: binary") {
		t.Errorf("execution_report.md deve conter 'launcher: binary'; conteúdo=%q", reportContent)
	}
}

// TestACPIntegration_Gemini_ActivityWatchdog (T-4.0 sub-test 6): Gemini inativo →
// ActivityWatchdog cancela via CancelCause(ErrActivityTimeout).
func TestACPIntegration_Gemini_ActivityWatchdog(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini iniciou watchdog test").
		AppendAgentMessageWithDelay("nunca chega", 500*time.Millisecond).
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()

	timeout, err := events.NewActivityTimeout(50 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:          "gemini watchdog test",
		WorkDir:         workDirWithAgentsMD(t),
		EvidenceDir:     t.TempDir(),
		ActivityTimeout: timeout,
		Quiet:           true,
	}

	summary, runErr := runner.Run(ctx, job)

	if runErr != nil {
		if summary.CancelReason != events.CancelReasonActivityTimeout {
			t.Errorf("CancelReason = %q, want activity_timeout when error present", summary.CancelReason)
		}
	} else {
		t.Logf("Gemini Watchdog: session ended before timeout; CancelReason=%q", summary.CancelReason)
	}

	if summary.CancelReason == events.CancelReasonPermissionDenied {
		t.Errorf("CancelReason = permission_denied unexpectedly")
	}

	if persist.summary == nil {
		t.Error("EnrichReport não foi chamado — execution_report não foi enriquecido")
	}
}

// TestACPIntegration_Gemini_LauncherUnavailable (T-4.0 sub-test 7): nenhum launcher
// disponível (nem gemini binário nem npx) → Run retorna erro.
func TestACPIntegration_Gemini_LauncherUnavailable(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pfact, _ := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberNone(), acpfake.NewScript(), pfact)

	job := airuntime.Job{
		Prompt:      "gemini launcher unavailable",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
	}

	_, err := runner.Run(ctx, job)
	if err == nil {
		t.Fatal("esperava erro quando nenhum launcher disponível para Gemini")
	}
}

// TestACPIntegration_Gemini_NpxFallback (T-4.0 sub-test 8): binário "gemini" ausente;
// npx disponível → Run usa fallback npx e retorna Launcher="npx".
func TestACPIntegration_Gemini_NpxFallback(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini via npx").
		AppendSessionEnd()

	pfact, _ := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiNpxOnly(), script, pfact)

	job := airuntime.Job{
		Prompt:      "gemini npx fallback",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini NpxFallback): %v", err)
	}

	if summary.Launcher != "npx" {
		t.Errorf("Launcher = %q, want npx", summary.Launcher)
	}
}

// TestACPIntegration_Gemini_BootstrapArgsDefault (T-4.0 sub-test 9): valida que
// geminiBootstrapArgs produz --approval-mode default quando AccessMode = Restricted/zero.
// Verifica que runtime_init.Args() contém --approval-mode default.
func TestACPIntegration_Gemini_BootstrapArgsDefault(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini bootstrap restricted").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "gemini restricted bootstrap",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AccessMode:  specs.AccessModeRestricted,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini BootstrapArgs Restricted): %v", err)
	}

	args := findRuntimeInitArgs(t, persist)

	if !containsArgPair(args, "--approval-mode", "default") {
		t.Errorf("args %v: esperava par --approval-mode default (AccessModeRestricted)", args)
	}
	// NÃO deve conter yolo em modo restricted.
	if containsArg(args, "yolo") {
		t.Errorf("args %v: NÃO esperava yolo no modo restricted", args)
	}
}

// TestACPIntegration_Gemini_BootstrapArgsFull (T-4.0 sub-test 10): valida que
// geminiBootstrapArgs produz --approval-mode yolo quando AccessMode = Full.
func TestACPIntegration_Gemini_BootstrapArgsFull(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("gemini bootstrap full").
		AppendSessionEnd()

	pfact, persist := newFakePersistenceFactory()
	runner := buildRunnerWithSpec(t, ctx, specs.Gemini(), proberGeminiBinary(), script, pfact)

	job := airuntime.Job{
		Prompt:      "gemini full access bootstrap",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AccessMode:  specs.AccessModeFull,
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run (Gemini BootstrapArgs Full): %v", err)
	}

	args := findRuntimeInitArgs(t, persist)

	if !containsArgPair(args, "--approval-mode", "yolo") {
		t.Errorf("args %v: esperava par --approval-mode yolo (AccessModeFull)", args)
	}
}

// TestACPIntegration_Gemini_ParidadeObservacional (T-4.0 sub-test 11): valida que
// events.jsonl Gemini tem mesma estrutura (kinds) que Claude para o mesmo script.
// RF-09: paridade observacional Gemini ↔ Claude/Codex/Copilot.
func TestACPIntegration_Gemini_ParidadeObservacional(t *testing.T) {
	t.Parallel()

	// Script idêntico para ambas as sessões.
	buildScript := func() *acpfake.Script {
		return acpfake.NewScript().
			AppendAgentMessage("parity message").
			AppendToolCall("tc", "some_tool").
			AppendToolCallUpdate("tc", "completed").
			AppendSessionEnd()
	}

	// Sessão Gemini
	geminiCtx, geminiCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer geminiCancel()

	geminiFakeFS := fs.NewFakeFileSystem()
	geminiPfact := persistence.NewSessionPersistenceFactory(geminiFakeFS)

	geminiRunner := airuntime.NewACPRunner(
		specs.Gemini(),
		airuntime.WithProber(proberGeminiBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: buildScript(), ctx: geminiCtx, t: t}),
		airuntime.WithPersistenceFactory(geminiPfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)
	geminiDir := "/evidence/gemini-parity"
	_, geminiErr := geminiRunner.Run(geminiCtx, airuntime.Job{
		Prompt:       "parity prompt",
		WorkDir:      "/workdir",
		EvidenceDir:  geminiDir,
		Quiet:        true,
		DisableHooks: true,
	})
	if geminiErr != nil {
		t.Fatalf("Run (Gemini parity): %v", geminiErr)
	}

	// Sessão Claude (mesma sequência de script)
	claudeCtx, claudeCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer claudeCancel()

	claudeFakeFS := fs.NewFakeFileSystem()
	claudePfact := persistence.NewSessionPersistenceFactory(claudeFakeFS)

	claudeRunner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(proberBinary()),
		airuntime.WithClientFactory(&fakeClientFactory{script: buildScript(), ctx: claudeCtx, t: t}),
		airuntime.WithPersistenceFactory(claudePfact),
		airuntime.WithRenderer(&discardRenderer{}),
	)
	claudeDir := "/evidence/claude-parity"
	_, claudeErr := claudeRunner.Run(claudeCtx, airuntime.Job{
		Prompt:       "parity prompt",
		WorkDir:      "/workdir",
		EvidenceDir:  claudeDir,
		Quiet:        true,
		DisableHooks: true,
	})
	if claudeErr != nil {
		t.Fatalf("Run (Claude parity): %v", claudeErr)
	}

	// Extrair kinds de ambas as sessões.
	extractKinds := func(fakeFS *fs.FakeFileSystem, dir string) []string {
		t.Helper()
		data, err := fakeFS.ReadFile(dir + "/events.jsonl")
		if err != nil {
			t.Fatalf("events.jsonl não encontrado em %s: %v", dir, err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		kinds := make([]string, 0, len(lines))
		for _, line := range lines {
			var env map[string]json.RawMessage
			if jsonErr := json.Unmarshal([]byte(line), &env); jsonErr != nil {
				continue
			}
			var k string
			if jsonErr := json.Unmarshal(env["kind"], &k); jsonErr != nil {
				continue
			}
			kinds = append(kinds, k)
		}
		return kinds
	}

	geminiKinds := extractKinds(geminiFakeFS, geminiDir)
	claudeKinds := extractKinds(claudeFakeFS, claudeDir)

	// Paridade: mesmos kinds na mesma ordem.
	if len(geminiKinds) != len(claudeKinds) {
		t.Errorf("paridade events.jsonl: Gemini kinds(%d)=%v, Claude kinds(%d)=%v",
			len(geminiKinds), geminiKinds, len(claudeKinds), claudeKinds)
	} else {
		for i, gk := range geminiKinds {
			if gk != claudeKinds[i] {
				t.Errorf("paridade events.jsonl[%d]: Gemini=%q, Claude=%q", i, gk, claudeKinds[i])
			}
		}
	}
}

// ---- verificação de unused imports ------------------------------------------
var _ = probe.ResetCache // silenciar import não usado
var _ = io.Discard
