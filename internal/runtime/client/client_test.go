package client_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// buildClientWithFake constrói um acpClient real conectado ao acpfake via pipes in-process.
func buildClientWithFake(t *testing.T, ctx context.Context, script *acpfake.Script) client.Client {
	t.Helper()

	srv := acpfake.NewServer(script)
	pc, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("acpfake.Start: %v", err)
	}

	return client.NewTestClient(t.TempDir(), pc.ClientWriter, pc.ClientReader)
}

// collectEvents drena o canal de eventos até fechar, com timeout.
func collectEvents(t *testing.T, ch <-chan events.Event, timeout time.Duration) []events.Event {
	t.Helper()
	var evts []events.Event
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return evts
			}
			evts = append(evts, evt)
		case <-timer.C:
			t.Log("collectEvents: timeout atingido")
			return evts
		}
	}
}

// TestAcpClient_HappyPath: round-trip com script de 5 updates; canal recebe eventos + fecha.
func TestAcpClient_HappyPath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("msg1").
		AppendAgentThought("pensamento").
		AppendToolCall("tc_1", "read_file").
		AppendToolCallUpdate("tc_1", "completed").
		AppendAgentMessage("msg2").
		AppendSessionEnd()

	launcher := specs.NewBinaryLauncher("echo") // dummy; não usado com IOProvider
	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	_ = launcher // injetado via pipe, não via subprocess
	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "faça algo"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 8*time.Second)

	// Esperamos ao menos 4 eventos mapeados (msg1, pensamento, tc start, tc update, msg2).
	if len(evts) < 4 {
		t.Errorf("esperava ao menos 4 eventos, got %d: %v", len(evts), kindsOf(evts))
	}
	if evts[len(evts)-1].Kind() != events.KindSessionEnd {
		t.Fatalf("último evento = %q, want session_end", evts[len(evts)-1].Kind())
	}
}

// TestAcpClient_FakeMute: fake vazio; cancelar contexto fecha canal.
func TestAcpClient_FakeMute(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Script vazio sem AppendSessionEnd → Prompt retorna imediatamente (script empty).
	script := acpfake.NewScript()
	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Cancelar o contexto deve encerrar a sessão.
	cancel()

	select {
	case <-c.Updates():
		// Canal fechou — OK.
	case <-time.After(3 * time.Second):
		t.Error("canal não fechou após cancelamento do contexto")
	}
}

// TestAcpClient_UnknownDrift: fake emite kind desconhecido; canal recebe KindUnknown.
func TestAcpClient_UnknownDrift(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendUnknown("weird_future_kind", nil).
		AppendSessionEnd()

	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 8*time.Second)

	if len(evts) == 0 {
		t.Fatal("esperava ao menos 1 evento (unknown), got 0")
	}

	found := false
	for _, e := range evts {
		if e.Kind() == events.KindUnknown {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava KindUnknown, kinds recebidos: %v", kindsOf(evts))
	}
}

// TestAcpClient_AbruptClose: fake encerra logo; canal fecha; Err pode ser nil ou erro.
func TestAcpClient_AbruptClose(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("msg").
		AppendSessionEnd()

	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 5*time.Second)
	if len(evts) == 0 {
		t.Fatal("esperava ao menos 1 evento, got 0")
	}

	// Após o canal fechar, Err() pode ser nil (sessão encerrou normalmente) ou erro.
	// Ambos são válidos.
	_ = c.Err()
}

func TestAcpClient_RequestPermissionCancelsPrompt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("antes da permissão").
		AppendRequestPermission("edit_file").
		AppendAgentMessage("depois da permissão").
		AppendSessionEnd()

	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Drena ate o canal fechar. A entrega da mensagem pre-permissao NAO e
	// deterministica: o acp-go-sdk despacha RequestPermission (request JSON-RPC)
	// em goroutine propria, concorrente com a fila sequencial de notificacoes
	// (SessionUpdate). O cancelamento disparado pela permissao pode fechar o
	// eventCh antes da notificacao anterior ser entregue (trySend ve closed e
	// descarta). Afirmar len(evts) > 0 era a origem do flake (intermitente sob
	// -race/carga). Afirmamos apenas os invariantes garantidos:
	//   1. Err() == ErrPermissionDenied (permissionRequested e setado antes do
	//      cancel; deterministico).
	//   2. nenhum session_end (cancel impede a resposta normal de Prompt).
	evts := collectEvents(t, c.Updates(), 8*time.Second)
	if !errors.Is(c.Err(), client.ErrPermissionDenied) {
		t.Fatalf("Err() = %v, want ErrPermissionDenied", c.Err())
	}
	for _, evt := range evts {
		if evt.Kind() == events.KindSessionEnd {
			t.Fatalf("não esperava session_end após requestPermission cancelado")
		}
	}
}

// TestAcpClient_RequestPermissionBypassAutoApproves é a regressão do fix de permissão (Copilot):
// com SetBypassPermissions(true) (AccessMode==full), RequestPermission auto-aprova (seleciona allow)
// em vez de cancelar — a sessão prossegue até session_end sem ErrPermissionDenied.
func TestAcpClient_RequestPermissionBypassAutoApproves(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("antes da permissão").
		AppendRequestPermission("edit_file").
		AppendAgentMessage("depois da permissão").
		AppendSessionEnd()

	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	// AccessMode==full: habilitar bypass antes de abrir a sessão.
	bp, ok := c.(interface{ SetBypassPermissions(bool) })
	if !ok {
		t.Fatal("client não expõe SetBypassPermissions")
	}
	bp.SetBypassPermissions(true)

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 8*time.Second)
	if errors.Is(c.Err(), client.ErrPermissionDenied) {
		t.Fatalf("com bypass não deveria haver ErrPermissionDenied; Err=%v", c.Err())
	}
	var sawSessionEnd bool
	for _, evt := range evts {
		if evt.Kind() == events.KindSessionEnd {
			sawSessionEnd = true
		}
	}
	if !sawSessionEnd {
		t.Fatal("esperava session_end (sessão prossegue após permissão auto-aprovada)")
	}
}

// TestAcpClient_CloseIdempotent: Close pode ser chamado múltiplas vezes sem pânico.
func TestAcpClient_CloseIdempotent(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendSessionEnd()
	c := buildClientWithFake(t, ctx, script)

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	collectEvents(t, c.Updates(), 3*time.Second)

	// Múltiplos Close devem ser idempotentes.
	for i := range 3 {
		if err := c.Close(); err != nil {
			t.Errorf("Close #%d: %v", i, err)
		}
	}
}

// TestAcpClient_Updates_BeforeOpen: Updates() antes de Open retorna canal válido.
func TestAcpClient_Updates_BeforeOpen(t *testing.T) {
	t.Parallel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(t.TempDir())
	ch := c.Updates()
	if ch == nil {
		t.Fatal("Updates() retornou nil antes de Open")
	}
}

// TestAcpClient_Err_BeforeOpen: Err() antes de Open retorna nil.
func TestAcpClient_Err_BeforeOpen(t *testing.T) {
	t.Parallel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(t.TempDir())
	if err := c.Err(); err != nil {
		t.Errorf("Err() antes de Open = %v; want nil", err)
	}
}

// TestClientFactory_New verifica que a factory cria clientes não-nil.
func TestClientFactory_New(t *testing.T) {
	t.Parallel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(t.TempDir())
	if c == nil {
		t.Fatal("factory.New retornou nil")
	}
	_ = c.Close()
}

// TestNoGoroutineLeak: Close() repetido sem Open não causa deadlock.
func TestNoGoroutineLeak(t *testing.T) {
	t.Parallel()

	factory := client.NewDefaultClientFactory()
	for i := range 5 {
		c := factory.New(t.TempDir())
		for range 2 {
			if err := c.Close(); err != nil {
				t.Errorf("Close #%d: %v", i, err)
			}
		}
	}
}

// TestLauncher_CommandNotFound: Open com launcher inexistente retorna erro.
func TestLauncher_CommandNotFound(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(t.TempDir())
	defer func() { _ = c.Close() }()

	launcher := specs.NewBinaryLauncher("__nao_existe_neste_sistema__")
	err := c.Open(ctx, launcher, "prompt")
	if err == nil {
		t.Error("esperava erro ao abrir com launcher inexistente")
	}
}

type fakeHandshakeWaiter struct {
	err error
}

func (w fakeHandshakeWaiter) Wait(_ context.Context) error { return w.err }

func TestAcpClient_HandshakeAbortsSessionBeforeFirstPrompt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("nunca deveria chegar ao modelo").
		AppendSessionEnd()

	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	hw, ok := c.(interface {
		SetHandshakeWaiter(client.HandshakeWaiter)
	})
	if !ok {
		t.Fatal("client não expõe SetHandshakeWaiter")
	}
	wantErr := errors.New("sentinel not received")
	hw.SetHandshakeWaiter(fakeHandshakeWaiter{err: wantErr})

	err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt")
	if err == nil {
		t.Fatal("Open() com handshake falho deveria retornar erro; sessão não pode prosseguir")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Open() erro = %v; want wrap de %v", err, wantErr)
	}

	evts := collectEvents(t, c.Updates(), 2*time.Second)
	for _, evt := range evts {
		t.Errorf("nenhum evento deveria ser emitido — handshake abortou antes do primeiro prompt; evt=%+v", evt)
	}
}

func TestAcpClient_HandshakeSuccessAllowsSessionToProceed(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("olá após handshake").
		AppendSessionEnd()

	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	hw, ok := c.(interface {
		SetHandshakeWaiter(client.HandshakeWaiter)
	})
	if !ok {
		t.Fatal("client não expõe SetHandshakeWaiter")
	}
	hw.SetHandshakeWaiter(fakeHandshakeWaiter{err: nil})

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open() com handshake bem-sucedido: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 5*time.Second)
	var sawSessionEnd bool
	for _, evt := range evts {
		if evt.Kind() == events.KindSessionEnd {
			sawSessionEnd = true
		}
	}
	if !sawSessionEnd {
		t.Fatal("esperava session_end após handshake bem-sucedido")
	}
}

func TestAcpClient_ChildEnvSanitizationAppliesToSpawnedProcessOnly(t *testing.T) {
	dir := t.TempDir()
	envDumpPath := filepath.Join(dir, "envdump.txt")
	scriptPath := filepath.Join(dir, "dump-env.sh")
	scriptBody := "#!/bin/sh\n" +
		"{ echo \"AISPEC_TEST_MARKER=$AISPEC_TEST_MARKER\"; echo \"OPENCODE_PURE=$OPENCODE_PURE\"; } > " + envDumpPath + "\n" +
		"exit 1\n"
	if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	t.Setenv("OPENCODE_PURE", "1")

	sanitized := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "OPENCODE_PURE=") {
			continue
		}
		sanitized = append(sanitized, kv)
	}
	sanitized = append(sanitized, "AISPEC_TEST_MARKER=present")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(dir)
	defer func() { _ = c.Close() }()

	ce, ok := c.(interface{ SetChildEnv([]string) })
	if !ok {
		t.Fatal("client não expõe SetChildEnv")
	}
	ce.SetChildEnv(sanitized)

	_ = c.Open(ctx, specs.NewBinaryLauncher(scriptPath), "prompt")

	deadline := time.Now().Add(10 * time.Second)
	var data []byte
	for time.Now().Before(deadline) {
		var readErr error
		data, readErr = os.ReadFile(envDumpPath)
		if readErr == nil && len(data) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if os.Getenv("OPENCODE_PURE") != "1" {
		t.Fatal("sanitização vazou para o ambiente do processo pai (teste) — RF-21 violado")
	}
	if !strings.Contains(string(data), "AISPEC_TEST_MARKER=present") {
		t.Fatalf("processo filho não recebeu childEnv sanitizado; dump=%q", data)
	}
	if strings.Contains(string(data), "OPENCODE_PURE=1") {
		t.Fatalf("processo filho ainda contém o interruptor OPENCODE_PURE; dump=%q", data)
	}
}

func parseEnvDump(t *testing.T, data []byte) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for _, line := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		if line == "" {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("linha de dump sem '=': %q", line)
		}
		out[name] = value
	}
	return out
}

func TestAcpClient_NilChildEnvInheritsFullEnvironByteIdentical(t *testing.T) {
	dir := t.TempDir()
	envDumpPath := filepath.Join(dir, "envdump.txt")
	scriptPath := filepath.Join(dir, "dump-full-env.sh")
	scriptBody := "#!/bin/sh\nenv > " + envDumpPath + "\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(scriptBody), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}

	t.Setenv("AISPEC_VETOR1_MARKER", "present-in-parent")

	shellBootstrapArtifacts := map[string]bool{"SHLVL": true, "OLDPWD": true, "PWD": true, "_": true}

	want := make(map[string]string, len(os.Environ()))
	for _, kv := range os.Environ() {
		name, value, ok := strings.Cut(kv, "=")
		if !ok || shellBootstrapArtifacts[name] {
			continue
		}
		want[name] = value
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(dir)
	defer func() { _ = c.Close() }()

	_ = c.Open(ctx, specs.NewBinaryLauncher(scriptPath), "prompt")

	deadline := time.Now().Add(10 * time.Second)
	var data []byte
	for time.Now().Before(deadline) {
		var readErr error
		data, readErr = os.ReadFile(envDumpPath)
		if readErr == nil && len(data) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(data) == 0 {
		t.Fatal("processo filho não produziu dump de ambiente")
	}

	got := parseEnvDump(t, data)
	for name := range shellBootstrapArtifacts {
		delete(got, name)
	}

	if len(got) != len(want) {
		t.Fatalf("ambiente do filho tem %d variáveis; pai tem %d (não é byte-idêntico)", len(got), len(want))
	}
	for name, wantValue := range want {
		gotValue, ok := got[name]
		if !ok {
			t.Errorf("variável %q ausente no ambiente do filho — herança não é byte-idêntica", name)
			continue
		}
		if gotValue != wantValue {
			t.Errorf("variável %q = %q no filho; pai tinha %q", name, gotValue, wantValue)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("variável %q presente no filho mas ausente no pai — ambiente foi alterado", name)
		}
	}
}

func kindsOf(evts []events.Event) []events.EventKind {
	kinds := make([]events.EventKind, len(evts))
	for i, e := range evts {
		kinds[i] = e.Kind()
	}
	return kinds
}
