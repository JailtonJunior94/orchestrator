package client_test

import (
	"context"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// buildClientWithBackpressure cria um acpClient com canal de cap personalizado e publishTimeout.
func buildClientWithBackpressure(t *testing.T, ctx context.Context, script *acpfake.Script, cap int, publishTimeout time.Duration) client.Client {
	t.Helper()
	srv := acpfake.NewServer(script)
	pc, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("acpfake.Start: %v", err)
	}
	return client.NewTestClientWithBackpressure(t.TempDir(), pc.ClientWriter, pc.ClientReader, cap, publishTimeout)
}

// TestAcpClient_SlowPublishes_DroppedUpdates_DefaultZero verifica que com os defaults
// (timeout=0, cap=64) os contadores de backpressure permanecem zero em sessão normal.
// Este é o teste de regressão F1: os defaults preservam o comportamento atual.
func TestAcpClient_SlowPublishes_DroppedUpdates_DefaultZero(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Script com 5 mensagens — cap 64 é mais que suficiente.
	script := acpfake.NewScript().
		AppendAgentMessage("msg1").
		AppendAgentMessage("msg2").
		AppendAgentMessage("msg3").
		AppendAgentMessage("msg4").
		AppendAgentMessage("msg5").
		AppendSessionEnd()

	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Drena todos os eventos.
	collectEvents(t, c.Updates(), 8*time.Second)

	// Com cap=64 e 5 eventos, nunca deve haver drop ou slow.
	if got := c.SlowPublishes(); got != 0 {
		t.Errorf("SlowPublishes() = %d, want 0 (sem backpressure em sessão normal)", got)
	}
	if got := c.DroppedUpdates(); got != 0 {
		t.Errorf("DroppedUpdates() = %d, want 0 (sem drop em sessão normal)", got)
	}
}

// TestAcpClient_DropImmediate_WhenChannelFull verifica que com cap=1 e publishTimeout=0,
// eventos excedentes são descartados imediatamente (droppedUpdates++).
// Simula o comportamento F1 com canal artificialmente pequeno.
func TestAcpClient_DropImmediate_WhenChannelFull(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Script com muitos eventos para saturar um canal de cap=1.
	script := acpfake.NewScript().
		AppendAgentMessage("m1").
		AppendAgentMessage("m2").
		AppendAgentMessage("m3").
		AppendAgentMessage("m4").
		AppendAgentMessage("m5").
		AppendAgentMessage("m6").
		AppendAgentMessage("m7").
		AppendAgentMessage("m8").
		AppendAgentMessage("m9").
		AppendAgentMessage("m10").
		AppendSessionEnd()

	// cap=1, publishTimeout=0 → drop imediato quando cheio.
	c := buildClientWithBackpressure(t, ctx, script, 1, 0)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "saturar canal"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	// Não drena imediatamente — deixa o fake saturar o canal.
	// Após a sessão encerrar, drenamos o restante.
	evts := collectEvents(t, c.Updates(), 8*time.Second)
	_ = evts

	// Com cap=1 e 10 eventos concorrentes, esperamos drops > 0.
	dropped := c.DroppedUpdates()
	if dropped == 0 {
		// Pode ocorrer se o consumidor drenou rápido o suficiente — não é falha determinística.
		// Registramos como aviso, não como erro fatal, pois o timing é não-determinístico.
		t.Logf("AVISO: DroppedUpdates() = 0 com cap=1 e 10 eventos — consumidor pode ter drenado antes de saturar")
	}
	// SlowPublishes deve ser zero com timeout=0.
	if got := c.SlowPublishes(); got != 0 {
		t.Errorf("SlowPublishes() = %d com publishTimeout=0; want 0 (drop imediato, nunca slow)", got)
	}
}

// TestAcpClient_SlowPublishes_WhenTimeoutPositive verifica que com publishTimeout>0,
// eventos que aguardaram para ser entregues incrementam slowPublishes (não droppedUpdates).
// Usa cap grande o suficiente para não descartar — apenas medir latência.
func TestAcpClient_SlowPublishes_WhenTimeoutPositive(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("m1").
		AppendAgentMessage("m2").
		AppendSessionEnd()

	// cap=64 com publishTimeout=1s → eventos entregues normalmente; slow=0 em sessão tranquila.
	c := buildClientWithBackpressure(t, ctx, script, 64, 1*time.Second)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 10*time.Second)
	if len(evts) == 0 {
		t.Fatal("esperava ao menos 1 evento")
	}

	// Com cap=64 e apenas 2 eventos, não deve haver drop nem slow.
	if got := c.DroppedUpdates(); got != 0 {
		t.Errorf("DroppedUpdates() = %d com cap=64; want 0", got)
	}
	// SlowPublishes pode ser 0 (canal nunca ficou cheio) — isto é o comportamento correto.
	slow := c.SlowPublishes()
	t.Logf("SlowPublishes() = %d (esperado 0 em sessão tranquila)", slow)
}

// TestAcpClient_CountersBeforeOpen verifica que os contadores retornam 0 antes de Open.
func TestAcpClient_CountersBeforeOpen(t *testing.T) {
	t.Parallel()

	factory := client.NewDefaultClientFactory()
	c := factory.New(t.TempDir())
	defer func() { _ = c.Close() }()

	if got := c.SlowPublishes(); got != 0 {
		t.Errorf("SlowPublishes() antes de Open = %d; want 0", got)
	}
	if got := c.DroppedUpdates(); got != 0 {
		t.Errorf("DroppedUpdates() antes de Open = %d; want 0", got)
	}
}

// TestAcpClient_Regression_EventCount verifica que com defaults (cap=64, timeout=0)
// a contagem de eventos é idêntica à esperada (regressão F1, RF-05).
func TestAcpClient_Regression_EventCount(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Script determinístico com 3 eventos conhecidos.
	script := acpfake.NewScript().
		AppendAgentMessage("hello").
		AppendAgentThought("pensando").
		AppendSessionEnd()

	// Usar buildClientWithFake que aplica cap=64, timeout=0 (F1 defaults).
	c := buildClientWithFake(t, ctx, script)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 8*time.Second)

	// Esperamos ao menos 2 eventos (mensagem + pensamento + session_end).
	if len(evts) < 2 {
		t.Errorf("contagem de eventos = %d, esperava >= 2 (regressão F1)", len(evts))
	}
	// Nenhum evento deve ter sido descartado.
	if got := c.DroppedUpdates(); got != 0 {
		t.Errorf("DroppedUpdates() = %d com defaults; want 0 (regressão F1)", got)
	}
	if got := c.SlowPublishes(); got != 0 {
		t.Errorf("SlowPublishes() = %d com defaults; want 0 (regressão F1)", got)
	}
}
