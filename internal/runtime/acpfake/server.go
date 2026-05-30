package acpfake

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
)

// Server é um servidor ACP fake que usa o AgentSideConnection do coder/acp-go-sdk.
// Emite updates ao cliente seguindo o script fornecido.
// Uso exclusivo em testes (RF-13).
type Server struct {
	script *Script
	mu     sync.Mutex
	conn   *acp.AgentSideConnection
}

// NewServer cria um Server com o script fornecido.
func NewServer(script *Script) *Server {
	return &Server{script: script}
}

// PipeConn representa as extremidades do pipe para cliente e servidor.
type PipeConn struct {
	// ClientReader é lido pelo cliente (saída do agente fake).
	ClientReader io.Reader
	// ClientWriter é escrito pelo cliente (entrada do agente fake).
	ClientWriter io.Writer
}

// Start inicializa a conexão servidor-cliente via os.Pipe() e inicia o servidor em background.
// Retorna o PipeConn com as extremidades que o cliente deve usar para stdinPipe/stdoutPipe.
func (s *Server) Start(ctx context.Context) (*PipeConn, error) {
	// Pipe do agente para o cliente (stdout do agente → stdin do cliente).
	agentToClientR, agentToClientW := io.Pipe()

	// Pipe do cliente para o agente (stdout do cliente → stdin do agente).
	clientToAgentR, clientToAgentW := io.Pipe()

	agent := &fakeAgent{
		server: s,
	}

	// AgentSideConnection: peerInput = onde o agente escreve (agentToClientW),
	// peerOutput = onde o agente lê (clientToAgentR).
	conn := acp.NewAgentSideConnection(agent, agentToClientW, clientToAgentR)
	agent.conn = conn
	s.setConn(conn)

	// O servidor roda até que a conexão feche.
	go func() {
		select {
		case <-conn.Done():
		case <-ctx.Done():
			_ = agentToClientW.Close()
			_ = clientToAgentR.Close()
		}
	}()

	return &PipeConn{
		// O cliente lê de agentToClientR (o que o agente escreve).
		ClientReader: agentToClientR,
		// O cliente escreve para clientToAgentW (o agente lê de clientToAgentR).
		ClientWriter: clientToAgentW,
	}, nil
}

func (s *Server) setConn(conn *acp.AgentSideConnection) {
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
}

func (s *Server) getConn() *acp.AgentSideConnection {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn
}

// fakeAgent implementa acp.Agent e emite os updates do script.
type fakeAgent struct {
	conn   *acp.AgentSideConnection
	server *Server
}

func (a *fakeAgent) Initialize(_ context.Context, _ acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion:   acp.ProtocolVersionNumber,
		AgentCapabilities: acp.AgentCapabilities{LoadSession: false},
	}, nil
}

func (a *fakeAgent) Authenticate(_ context.Context, _ acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *fakeAgent) Cancel(_ context.Context, _ acp.CancelNotification) error {
	return nil
}

func (a *fakeAgent) NewSession(_ context.Context, _ acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	return acp.NewSessionResponse{SessionId: acp.SessionId("fake-session-001")}, nil
}

func (a *fakeAgent) ListSessions(_ context.Context, _ acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, nil
}

func (a *fakeAgent) ResumeSession(_ context.Context, _ acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, nil
}

func (a *fakeAgent) CloseSession(_ context.Context, _ acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	return acp.CloseSessionResponse{}, nil
}

func (a *fakeAgent) SetSessionConfigOption(_ context.Context, _ acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, nil
}

func (a *fakeAgent) SetSessionMode(_ context.Context, _ acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}

// Prompt emite os updates do script e encerra a sessão.
func (a *fakeAgent) Prompt(ctx context.Context, p acp.PromptRequest) (acp.PromptResponse, error) {
	conn := a.server.getConn()
	if conn == nil {
		return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
	}

	for _, su := range a.server.script.Updates() {
		if ctx.Err() != nil {
			return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, ctx.Err()
		}

		if su.Delay > 0 {
			select {
			case <-ctx.Done():
				return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, ctx.Err()
			case <-time.After(su.Delay):
			}
		}

		// Update vazio = marcador de sessão finalizada (AppendSessionEnd).
		if NewCatalog().isSessionEnd(su.Update) {
			return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
		}

		// Plan com entrada "requestPermission:*" = solicitar permissão (RF-16).
		if reqPerm, toolName := NewCatalog().isRequestPermission(su.Update); reqPerm {
			_, _ = conn.RequestPermission(ctx, acp.RequestPermissionRequest{
				SessionId: p.SessionId,
				ToolCall: acp.ToolCallUpdate{
					ToolCallId: acp.ToolCallId("call_perm"),
					Title:      acp.Ptr(toolName),
					Status:     acp.Ptr(acp.ToolCallStatusPending),
				},
				Options: []acp.PermissionOption{
					{
						Kind:     acp.PermissionOptionKindAllowOnce,
						Name:     "Permitir",
						OptionId: acp.PermissionOptionId("allow"),
					},
					{
						Kind:     acp.PermissionOptionKindRejectOnce,
						Name:     "Rejeitar",
						OptionId: acp.PermissionOptionId("reject"),
					},
				},
			})
			continue
		}

		_ = conn.SessionUpdate(ctx, acp.SessionNotification{
			SessionId: p.SessionId,
			Update:    su.Update,
		})
	}

	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, nil
}

// isSessionEnd retorna true para o marcador de fim de sessão (update vazio).
func (c *Catalog) isSessionEnd(u acp.SessionUpdate) bool {
	return u.AgentMessageChunk == nil &&
		u.AgentThoughtChunk == nil &&
		u.ToolCall == nil &&
		u.ToolCallUpdate == nil &&
		u.UserMessageChunk == nil &&
		u.Plan == nil
}

// isRequestPermission detecta o marcador de requestPermission.
func (c *Catalog) isRequestPermission(u acp.SessionUpdate) (bool, string) {
	if u.Plan == nil || len(u.Plan.Entries) == 0 {
		return false, ""
	}
	content := u.Plan.Entries[0].Content
	if strings.HasPrefix(content, "requestPermission:") {
		return true, strings.TrimPrefix(content, "requestPermission:")
	}
	return false, ""
}
