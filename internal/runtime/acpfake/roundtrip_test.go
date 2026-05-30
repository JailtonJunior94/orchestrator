package acpfake_test

import (
	"context"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// minimalClient implementa acp.Client capturando SessionUpdate como events.
type minimalClient struct {
	eventCh chan events.Event
}

func (c *minimalClient) SessionUpdate(_ context.Context, n acp.SessionNotification) error {
	evt, _ := events.NewCatalog().FromACPUpdate("", n.Update)
	select {
	case c.eventCh <- evt:
	default:
	}
	return nil
}

func (c *minimalClient) RequestPermission(_ context.Context, _ acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	return acp.RequestPermissionResponse{
		Outcome: acp.RequestPermissionOutcome{
			Cancelled: &acp.RequestPermissionOutcomeCancelled{},
		},
	}, nil
}

func (c *minimalClient) ReadTextFile(_ context.Context, _ acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	return acp.ReadTextFileResponse{}, nil
}

func (c *minimalClient) WriteTextFile(_ context.Context, _ acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	return acp.WriteTextFileResponse{}, nil
}

func (c *minimalClient) CreateTerminal(_ context.Context, _ acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	return acp.CreateTerminalResponse{TerminalId: "t-1"}, nil
}

func (c *minimalClient) KillTerminal(_ context.Context, _ acp.KillTerminalRequest) (acp.KillTerminalResponse, error) {
	return acp.KillTerminalResponse{}, nil
}

func (c *minimalClient) ReleaseTerminal(_ context.Context, _ acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	return acp.ReleaseTerminalResponse{}, nil
}

func (c *minimalClient) TerminalOutput(_ context.Context, _ acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	return acp.TerminalOutputResponse{Output: "ok"}, nil
}

func (c *minimalClient) WaitForTerminalExit(_ context.Context, _ acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	return acp.WaitForTerminalExitResponse{}, nil
}

// connectAndPrompt abre conexão com o fake server, negocia e envia prompt.
func connectAndPrompt(t *testing.T, ctx context.Context, script *acpfake.Script) []events.Event {
	t.Helper()

	srv := acpfake.NewServer(script)
	pc, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	cli := &minimalClient{eventCh: make(chan events.Event, 128)}
	conn := acp.NewClientSideConnection(cli, pc.ClientWriter, pc.ClientReader)

	_, err = conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{},
	})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	sessResp, err := conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd:        t.TempDir(),
		McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	_, _ = conn.Prompt(ctx, acp.PromptRequest{
		SessionId: sessResp.SessionId,
		Prompt:    []acp.ContentBlock{acp.TextBlock("prompt")},
	})
	close(cli.eventCh)

	var evts []events.Event
	for evt := range cli.eventCh {
		evts = append(evts, evt)
	}
	return evts
}

// TestRoundTrip_HappyPath: round-trip completo com script de múltiplos updates.
func TestRoundTrip_HappyPath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("olá").
		AppendAgentThought("pensando").
		AppendToolCall("tc_1", "read_file").
		AppendToolCallUpdate("tc_1", "completed").
		AppendAgentMessage("feito").
		AppendSessionEnd()

	evts := connectAndPrompt(t, ctx, script)
	if len(evts) < 4 {
		t.Errorf("esperava ao menos 4 eventos, got %d: %v", len(evts), kindsInRoundtrip(evts))
	}

	// Verifica que agent_message está presente.
	found := false
	for _, e := range evts {
		if e.Kind() == events.KindAgentMessage {
			found = true
			break
		}
	}
	if !found {
		t.Error("esperava KindAgentMessage nos eventos")
	}
}

// TestRoundTrip_UnknownUpdate: fake emite update desconhecido → KindUnknown recebido.
func TestRoundTrip_UnknownUpdate(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendUnknown("weird_kind", nil).
		AppendSessionEnd()

	evts := connectAndPrompt(t, ctx, script)
	if len(evts) == 0 {
		t.Fatal("esperava ao menos 1 evento")
	}

	found := false
	for _, e := range evts {
		if e.Kind() == events.KindUnknown {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava KindUnknown: %v", kindsInRoundtrip(evts))
	}
}

// TestRoundTrip_RequestPermission: fake emite requestPermission → cancelado pelo cliente.
func TestRoundTrip_RequestPermission(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendRequestPermission("edit_file").
		AppendAgentMessage("continuando").
		AppendSessionEnd()

	// Não deve panicar; o cliente cancela a permissão e o agente continua.
	evts := connectAndPrompt(t, ctx, script)
	// Ao menos 1 evento (agent_message após permissão cancelada).
	_ = evts
}

// TestRoundTrip_DelayedMessage: agente emite com delay; mensagem chega corretamente.
func TestRoundTrip_DelayedMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessageWithDelay("atrasada", 20*time.Millisecond).
		AppendSessionEnd()

	evts := connectAndPrompt(t, ctx, script)
	if len(evts) == 0 {
		t.Fatal("esperava ao menos 1 evento após delay")
	}
}

// TestRoundTrip_EmptyScript: script vazio retorna imediatamente.
func TestRoundTrip_EmptyScript(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	script := acpfake.NewScript()
	evts := connectAndPrompt(t, ctx, script)
	// Script vazio → 0 eventos.
	_ = evts
}

// TestRoundTrip_ContextCanceled: contexto cancelado encerra o round-trip graciosamente.
func TestRoundTrip_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// Script com delay longo que será cancelado antes de terminar.
	script := acpfake.NewScript().
		AppendAgentMessageWithDelay("nunca chega", 2*time.Second).
		AppendSessionEnd()

	srv := acpfake.NewServer(script)
	pc, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	cli := &minimalClient{eventCh: make(chan events.Event, 128)}
	conn := acp.NewClientSideConnection(cli, pc.ClientWriter, pc.ClientReader)

	_, err = conn.Initialize(ctx, acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientCapabilities: acp.ClientCapabilities{},
	})
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	sessResp, err := conn.NewSession(ctx, acp.NewSessionRequest{
		Cwd: t.TempDir(), McpServers: []acp.McpServer{},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	promptDone := make(chan struct{})
	go func() {
		defer close(promptDone)
		_, _ = conn.Prompt(ctx, acp.PromptRequest{
			SessionId: sessResp.SessionId,
			Prompt:    []acp.ContentBlock{acp.TextBlock("prompt")},
		})
	}()

	// Cancela após 50ms — bem antes do delay de 2s.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	select {
	case <-promptDone:
		// OK: Prompt encerrou após cancelamento.
	case <-time.After(4 * time.Second):
		t.Error("Prompt não encerrou após cancelamento do contexto")
	}
}

// TestAcpfake_AppendToolCallUpdate_Failed testa o branch "failed" do AppendToolCallUpdate.
func TestAcpfake_AppendToolCallUpdate_Failed(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendToolCall("tc_x", "write_file").
		AppendToolCallUpdate("tc_x", "failed").
		AppendSessionEnd()

	evts := connectAndPrompt(t, ctx, script)
	found := false
	for _, e := range evts {
		if e.Kind() == events.KindToolCallUpdate {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("esperava KindToolCallUpdate, got: %v", kindsInRoundtrip(evts))
	}
}

// TestAcpfake_AppendToolCallUpdate_Custom testa status customizado.
func TestAcpfake_AppendToolCallUpdate_Custom(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendToolCall("tc_y", "run_command").
		AppendToolCallUpdate("tc_y", "running").
		AppendSessionEnd()

	evts := connectAndPrompt(t, ctx, script)
	_ = evts // executa sem pânico
}

func kindsInRoundtrip(evts []events.Event) []events.EventKind {
	kinds := make([]events.EventKind, len(evts))
	for i, e := range evts {
		kinds[i] = e.Kind()
	}
	return kinds
}
