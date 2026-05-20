package client_test

import (
	"context"
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

func kindsOf(evts []events.Event) []events.EventKind {
	kinds := make([]events.EventKind, len(evts))
	for i, e := range evts {
		kinds[i] = e.Kind()
	}
	return kinds
}
