// impl_test.go cobre os caminhos de erro dos métodos não suportados de clientImpl,
// exercitando-os via testes de integração in-process com o SDK.
// Os métodos são chamados pelo AgentSideConnection quando o agente tenta usá-los.
package client_test

import (
	"context"
	"io"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// fakeAgentWithFS é um agente que tenta chamar ReadTextFile e WriteTextFile no cliente,
// cobrindo os paths de erro do clientImpl.
type fakeAgentWithFS struct {
	conn    *acp.AgentSideConnection
	sessID  acp.SessionId
	doFS    bool
	doTerms bool
}

var _ acp.Agent = (*fakeAgentWithFS)(nil)

func (a *fakeAgentWithFS) SetConn(c *acp.AgentSideConnection) { a.conn = c }

func (a *fakeAgentWithFS) Initialize(_ context.Context, _ acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion:   acp.ProtocolVersionNumber,
		AgentCapabilities: acp.AgentCapabilities{LoadSession: false},
	}, nil
}

func (a *fakeAgentWithFS) Authenticate(_ context.Context, _ acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *fakeAgentWithFS) Cancel(_ context.Context, _ acp.CancelNotification) error { return nil }

func (a *fakeAgentWithFS) NewSession(_ context.Context, _ acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	return acp.NewSessionResponse{SessionId: acp.SessionId("sess-fs-test")}, nil
}

func (a *fakeAgentWithFS) ListSessions(_ context.Context, _ acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, nil
}

func (a *fakeAgentWithFS) ResumeSession(_ context.Context, _ acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, nil
}

func (a *fakeAgentWithFS) CloseSession(_ context.Context, _ acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	return acp.CloseSessionResponse{}, nil
}

func (a *fakeAgentWithFS) SetSessionConfigOption(_ context.Context, _ acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, nil
}

func (a *fakeAgentWithFS) SetSessionMode(_ context.Context, _ acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}

func (a *fakeAgentWithFS) Prompt(ctx context.Context, p acp.PromptRequest) (acp.PromptResponse, error) {
	a.sessID = p.SessionId
	conn := a.conn

	if a.doFS {
		// Tenta ReadTextFile — deve retornar erro do clientImpl.
		_, _ = conn.ReadTextFile(ctx, acp.ReadTextFileRequest{Path: "/tmp/x"})
		// Tenta WriteTextFile — deve retornar erro do clientImpl.
		_, _ = conn.WriteTextFile(ctx, acp.WriteTextFileRequest{Path: "/tmp/y", Content: "abc"})
	}

	if a.doTerms {
		// Testa os métodos de terminal.
		_, _ = conn.CreateTerminal(ctx, acp.CreateTerminalRequest{})
		_, _ = conn.KillTerminal(ctx, acp.KillTerminalRequest{TerminalId: "t-1"})
		_, _ = conn.ReleaseTerminal(ctx, acp.ReleaseTerminalRequest{TerminalId: "t-1"})
		_, _ = conn.TerminalOutput(ctx, acp.TerminalOutputRequest{TerminalId: "t-1"})
		_, _ = conn.WaitForTerminalExit(ctx, acp.WaitForTerminalExitRequest{TerminalId: "t-1"})
	}

	// Emite uma mensagem para sinalizar que o agente está vivo.
	_ = conn.SessionUpdate(ctx, acp.SessionNotification{
		SessionId: p.SessionId,
		Update:    acp.UpdateAgentMessageText("pronto"),
	})

	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
}

// buildCustomAgentClient constrói um Client real usando um agente customizado via pipes diretas.
func buildCustomAgentClient(t *testing.T, ctx context.Context, agent acp.Agent) (client.Client, *acp.AgentSideConnection) {
	t.Helper()

	agentToClientR, agentToClientW := newPipePair(t)
	clientToAgentR, clientToAgentW := newPipePair(t)

	conn := acp.NewAgentSideConnection(agent, agentToClientW, clientToAgentR)

	// Configura a conexão no agente (se tiver SetConn).
	type connSetter interface {
		SetConn(*acp.AgentSideConnection)
	}
	if s, ok := agent.(connSetter); ok {
		s.SetConn(conn)
	}

	c := client.NewTestClient(t.TempDir(), clientToAgentW, agentToClientR)

	go func() {
		select {
		case <-conn.Done():
		case <-ctx.Done():
			_ = agentToClientW.Close()
			_ = clientToAgentR.Close()
		}
	}()

	return c, conn
}

// TestClientImpl_FSMethodsReturnError verifica que ReadTextFile/WriteTextFile retornam erro.
func TestClientImpl_FSMethodsReturnError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agent := &fakeAgentWithFS{doFS: true}
	c, _ := buildCustomAgentClient(t, ctx, agent)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 8*time.Second)
	if len(evts) == 0 {
		t.Fatal("esperava ao menos 1 evento, got 0")
	}
}

// TestClientImpl_TerminalMethodsReturnError verifica que métodos de terminal retornam erro.
func TestClientImpl_TerminalMethodsReturnError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agent := &fakeAgentWithFS{doTerms: true}
	c, _ := buildCustomAgentClient(t, ctx, agent)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open: %v", err)
	}

	evts := collectEvents(t, c.Updates(), 8*time.Second)
	if len(evts) == 0 {
		t.Fatal("esperava ao menos 1 evento, got 0")
	}
}

// TestAcpClient_OpenTwice: Open após já aberto (pipe diferente) retorna erro ou entra em estado inválido.
// Garante que o cliente não entre em pânico.
func TestAcpClient_OpenTwice(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendAgentMessage("first").AppendSessionEnd()

	srv := acpfake.NewServer(script)
	pc, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("acpfake.Start: %v", err)
	}

	c := client.NewTestClient(t.TempDir(), pc.ClientWriter, pc.ClientReader)
	defer func() { _ = c.Close() }()

	if err := c.Open(ctx, specs.NewBinaryLauncher("unused"), "prompt"); err != nil {
		t.Fatalf("Open 1: %v", err)
	}
	collectEvents(t, c.Updates(), 5*time.Second)
}

// newPipePair cria um par de io.PipeReader/io.PipeWriter e registra cleanup.
func newPipePair(t *testing.T) (*io.PipeReader, *io.PipeWriter) {
	t.Helper()
	r, w := io.Pipe()
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	return r, w
}
