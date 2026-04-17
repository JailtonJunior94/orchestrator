package acp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
)

// mockAgent is a minimal in-process ACP agent for testing.
type mockAgent struct {
	initErr      error
	protocolVer  acpsdk.ProtocolVersion
	supportsLoad bool
	sessionID    string
}

func (m *mockAgent) Authenticate(_ context.Context, _ acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (m *mockAgent) Initialize(_ context.Context, req acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	if m.initErr != nil {
		return acpsdk.InitializeResponse{}, m.initErr
	}
	ver := req.ProtocolVersion
	if m.protocolVer != 0 {
		ver = m.protocolVer
	}
	return acpsdk.InitializeResponse{
		ProtocolVersion: ver,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: m.supportsLoad,
		},
	}, nil
}

func (m *mockAgent) Cancel(_ context.Context, _ acpsdk.CancelNotification) error { return nil }

func (m *mockAgent) ListSessions(_ context.Context, _ acpsdk.ListSessionsRequest) (acpsdk.ListSessionsResponse, error) {
	return acpsdk.ListSessionsResponse{}, acpsdk.NewMethodNotFound(acpsdk.AgentMethodSessionList)
}

func (m *mockAgent) NewSession(_ context.Context, _ acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{SessionId: acpsdk.SessionId(m.sessionID)}, nil
}

func (m *mockAgent) Prompt(ctx context.Context, req acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
}

func (m *mockAgent) SetSessionConfigOption(_ context.Context, _ acpsdk.SetSessionConfigOptionRequest) (acpsdk.SetSessionConfigOptionResponse, error) {
	return acpsdk.SetSessionConfigOptionResponse{}, nil
}

func (m *mockAgent) SetSessionMode(_ context.Context, _ acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

// mockAgentWithLoad also implements AgentLoader.
type mockAgentWithLoad struct {
	mockAgent
	loadErr error
}

func (m *mockAgentWithLoad) LoadSession(_ context.Context, req acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	if m.loadErr != nil {
		return acpsdk.LoadSessionResponse{}, m.loadErr
	}
	return acpsdk.LoadSessionResponse{}, nil
}

// pipeConn wires a mock agent to a real connection using in-memory pipes.
func pipeConn(t *testing.T, agent acpsdk.Agent, opts ...ConnectionOption) *connection {
	t.Helper()

	agentR, clientW := io.Pipe()
	clientR, agentW := io.Pipe()

	_ = acpsdk.NewAgentSideConnection(agent, agentW, agentR)

	t.Cleanup(func() {
		_ = clientW.Close()
		_ = agentW.Close()
	})

	opts = append([]ConnectionOption{WithLogger(slog.Default())}, opts...)
	c := &connection{
		logger:           slog.Default(),
		handshakeTimeout: defaultHandshakeTimeout,
		stderr:           nil,
	}
	for _, o := range opts {
		o(c)
	}
	c.client = newACPClient(c.logger, nil, nil)
	c.conn = acpsdk.NewClientSideConnection(c.client, clientW, clientR)
	c.conn.SetLogger(c.logger)
	return c
}

// --- Handshake tests ---

func TestConnection_Initialize_Success(t *testing.T) {
	agent := &mockAgent{sessionID: "s-1"}
	c := pipeConn(t, agent)

	resp, err := c.Initialize(context.Background())
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	if resp.ProtocolVersion != acpsdk.ProtocolVersionNumber {
		t.Errorf("ProtocolVersion = %d, want %d", resp.ProtocolVersion, acpsdk.ProtocolVersionNumber)
	}
}

func TestConnection_Initialize_VersionMismatch(t *testing.T) {
	agent := &mockAgent{protocolVer: 9999}
	c := pipeConn(t, agent)

	_, err := c.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected error for version mismatch, got nil")
	}
	if !isErrorContaining(err, ErrProtocolVersionMismatch) {
		t.Errorf("error = %v, want to wrap ErrProtocolVersionMismatch", err)
	}
}

func TestConnection_Initialize_Timeout(t *testing.T) {
	// Agent that reads requests but never responds, so the client times out.
	agentR, clientW := io.Pipe()
	clientR, agentW := io.Pipe()

	// Drain client writes so they don't block; never write back.
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := agentR.Read(buf); err != nil {
				return
			}
		}
	}()

	t.Cleanup(func() {
		_ = clientW.Close()
		_ = agentW.Close()
		_ = agentR.Close()
		_ = clientR.Close()
	})

	c := &connection{
		logger:           slog.Default(),
		handshakeTimeout: 50 * time.Millisecond,
		client:           newACPClient(slog.Default(), nil, nil),
	}
	c.conn = acpsdk.NewClientSideConnection(c.client, clientW, clientR)
	c.conn.SetLogger(c.logger)

	_, err := c.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !isErrorContaining(err, ErrHandshakeTimeout) {
		t.Errorf("error = %v, want ErrHandshakeTimeout", err)
	}
}

func TestConnection_Initialize_IncludesStderrInError(t *testing.T) {
	agent := &mockAgent{initErr: errors.New("agent boom")}
	c := pipeConn(t, agent)
	c.stderr = bytes.NewBufferString("adapter failed to boot")

	_, err := c.Initialize(context.Background())
	if err == nil {
		t.Fatal("expected initialize error")
	}
	if got := err.Error(); got == "" || !containsSubstring(got, "agent boom") {
		t.Fatalf("expected original error details, got %q", got)
	}
	if got := err.Error(); got == "" || !containsSubstring(got, "stderr: adapter failed to boot") {
		t.Fatalf("expected stderr in error, got %q", got)
	}
}

// --- NewSession test ---

func TestConnection_NewSession_ReturnsSessionID(t *testing.T) {
	agent := &mockAgent{sessionID: "sess-abc"}
	c := pipeConn(t, agent)

	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	id, err := c.NewSession(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if id != "sess-abc" {
		t.Errorf("sessionID = %q, want %q", id, "sess-abc")
	}
	if c.client.getFileHandler() == nil {
		t.Fatal("expected NewSession to install a filesystem handler")
	}
}

// --- LoadSession tests ---

func TestConnection_LoadSession_Success(t *testing.T) {
	agent := &mockAgentWithLoad{
		mockAgent: mockAgent{sessionID: "s-1", supportsLoad: true},
	}
	c := pipeConn(t, agent)

	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// Establish cwd via NewSession before loading.
	if _, err := c.NewSession(context.Background(), t.TempDir()); err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	loaded, err := c.LoadSession(context.Background(), "s-1", t.TempDir())
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}
	if !loaded {
		t.Fatal("expected LoadSession to report success")
	}
	if c.client.getFileHandler() == nil {
		t.Fatal("expected LoadSession to install a filesystem handler")
	}
}

func TestConnection_LoadSession_Fallback_WhenNotSupported(t *testing.T) {
	agent := &mockAgent{sessionID: "s-1", supportsLoad: false}
	c := pipeConn(t, agent)

	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	// Should not return an error even when agent doesn't support LoadSession.
	loaded, err := c.LoadSession(context.Background(), "s-1", t.TempDir())
	if err != nil {
		t.Fatalf("LoadSession fallback: %v", err)
	}
	if loaded {
		t.Fatal("expected unsupported load session to report fallback")
	}
}

func TestConnection_LoadSession_MapsNotFoundError(t *testing.T) {
	agent := &mockAgentWithLoad{
		mockAgent: mockAgent{sessionID: "s-1", supportsLoad: true},
		loadErr:   errors.New("session not found on agent"),
	}
	c := pipeConn(t, agent)

	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if _, err := c.LoadSession(context.Background(), "missing", t.TempDir()); !isErrorContaining(err, ErrSessionNotFound) {
		t.Fatalf("LoadSession error = %v, want ErrSessionNotFound", err)
	}
}

func TestConnection_LoadSession_MapsExpiredError(t *testing.T) {
	agent := &mockAgentWithLoad{
		mockAgent: mockAgent{sessionID: "s-1", supportsLoad: true},
		loadErr:   errors.New("session expired after idle timeout"),
	}
	c := pipeConn(t, agent)

	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	if _, err := c.LoadSession(context.Background(), "expired", t.TempDir()); !isErrorContaining(err, ErrSessionExpired) {
		t.Fatalf("LoadSession error = %v, want ErrSessionExpired", err)
	}
}

func containsSubstring(s, sub string) bool {
	return strings.Contains(s, sub)
}

// --- Prompt test ---

func TestConnection_Prompt_SendsAndReceives(t *testing.T) {
	agent := &mockAgent{sessionID: "s-1"}
	c := pipeConn(t, agent)

	if _, err := c.Initialize(context.Background()); err != nil {
		t.Fatalf("Initialize: %v", err)
	}

	stopReason, err := c.Prompt(context.Background(), "s-1", "Hello!")
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	if stopReason != string(acpsdk.StopReasonEndTurn) {
		t.Fatalf("StopReason = %q, want %q", stopReason, acpsdk.StopReasonEndTurn)
	}
}

// isErrorContaining checks whether target appears in the error chain.
func isErrorContaining(err, target error) bool {
	if err == nil {
		return false
	}
	type unwrapper interface{ Unwrap() error }
	for {
		if err.Error() == target.Error() {
			return true
		}
		uw, ok := err.(unwrapper)
		if !ok {
			break
		}
		err = uw.Unwrap()
		if err == nil {
			break
		}
	}
	return false
}
