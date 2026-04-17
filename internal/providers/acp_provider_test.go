package providers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/jailtonjunior/orchestrator/internal/acp"
	"github.com/jailtonjunior/orchestrator/internal/providers"
)

// mockConnection is a level-3 mock of acp.ACPConnection for provider tests.
type mockConnection struct {
	initResp          *acpsdk.InitializeResponse
	initErr           error
	newSessID         string
	newSessErr        error
	loadSessErr       error
	loadSessSupported bool
	loadSessCalls     int
	loadSessCWDs      []string
	promptErr         error
	promptStopReason  string
	promptDeadlineSet bool
	updates           chan acp.TypedUpdate
	done              chan struct{}
	closeErr          error
}

func newMockConn() *mockConnection {
	return &mockConnection{
		initResp: &acpsdk.InitializeResponse{
			ProtocolVersion: acpsdk.ProtocolVersionNumber,
		},
		loadSessSupported: true,
		promptStopReason:  "end_turn",
		updates: make(chan acp.TypedUpdate, 16),
		done:    make(chan struct{}),
	}
}

func (m *mockConnection) Initialize(_ context.Context) (*acpsdk.InitializeResponse, error) {
	return m.initResp, m.initErr
}
func (m *mockConnection) NewSession(_ context.Context, _ string) (string, error) {
	return m.newSessID, m.newSessErr
}
func (m *mockConnection) LoadSession(_ context.Context, _ string, cwd string) (bool, error) {
	m.loadSessCalls++
	m.loadSessCWDs = append(m.loadSessCWDs, cwd)
	return m.loadSessSupported, m.loadSessErr
}
func (m *mockConnection) Prompt(ctx context.Context, _ string, _ string) (string, error) {
	_, m.promptDeadlineSet = ctx.Deadline()
	return m.promptStopReason, m.promptErr
}
func (m *mockConnection) Updates() <-chan acp.TypedUpdate { return m.updates }
func (m *mockConnection) Done() <-chan struct{}           { return m.done }
func (m *mockConnection) Close(_ context.Context) error  { return m.closeErr }

// providerWithMock creates an acpProvider wired to a provided mockConnection
// via the exported test helper.
func providerWithMock(conn acp.ACPConnection, spec acp.AgentSpec) providers.ACPProvider {
	return providers.NewACPProviderWithConn(spec, conn)
}

// ---- Tests ----------------------------------------------------------------

func TestACPProvider_Execute_AccumulatesChunks(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	conn.newSessID = "sess-1"

	// Enqueue updates then close done.
	conn.updates <- acp.TypedUpdate{Kind: acp.UpdateMessage, Text: "hello "}
	conn.updates <- acp.TypedUpdate{Kind: acp.UpdateMessage, Text: "world"}
	conn.updates <- acp.TypedUpdate{Kind: acp.UpdateThought, Text: "thinking"}
	close(conn.done)

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	out, err := p.Execute(context.Background(), acp.ACPInput{
		Prompt:  "prompt",
		WorkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Content != "hello world" {
		t.Fatalf("Content = %q, want %q", out.Content, "hello world")
	}
	if out.Thoughts != "thinking" {
		t.Fatalf("Thoughts = %q, want %q", out.Thoughts, "thinking")
	}
	if out.SessionID != "sess-1" {
		t.Fatalf("SessionID = %q, want %q", out.SessionID, "sess-1")
	}
	if out.StopReason != "end_turn" {
		t.Fatalf("StopReason = %q, want %q", out.StopReason, "end_turn")
	}
}

func TestACPProvider_ExecuteStream_AllUpdateTypesReceived(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	conn.newSessID = "sess-2"

	toolInfo := &acp.ToolCallInfo{ID: "tc-1", Title: "edit", Kind: "file", Status: "running"}
	conn.updates <- acp.TypedUpdate{Kind: acp.UpdateMessage, Text: "msg"}
	conn.updates <- acp.TypedUpdate{Kind: acp.UpdateThought, Text: "thought"}
	conn.updates <- acp.TypedUpdate{Kind: acp.UpdateToolCall, ToolCall: toolInfo}
	conn.updates <- acp.TypedUpdate{Kind: acp.UpdateToolUpdate, ToolCall: &acp.ToolCallInfo{ID: "tc-1", Status: "done"}}
	close(conn.done)

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	var received []acp.TypedUpdate
	out, err := p.ExecuteStream(context.Background(), acp.ACPInput{
		Prompt:  "prompt",
		WorkDir: t.TempDir(),
	}, func(u acp.TypedUpdate) {
		received = append(received, u)
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(received) != 4 {
		t.Fatalf("received %d updates, want 4", len(received))
	}
	if out.Content != "msg" {
		t.Fatalf("Content = %q", out.Content)
	}
	if len(out.ToolCalls) != 2 {
		t.Fatalf("ToolCalls len = %d, want 2", len(out.ToolCalls))
	}
}

func TestACPProvider_SessionResume_UsesLoadSession(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	close(conn.done)

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	_, err := p.Execute(context.Background(), acp.ACPInput{
		Prompt:    "prompt",
		SessionID: "existing-session",
		WorkDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if conn.loadSessCalls != 1 {
		t.Fatal("expected LoadSession to be called for non-empty SessionID")
	}
}

func TestACPProvider_SessionResume_FallsBackToNewSessionWhenUnsupported(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	conn.loadSessSupported = false
	conn.newSessID = "fresh-session"
	close(conn.done)

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	out, err := p.Execute(context.Background(), acp.ACPInput{
		Prompt:    "prompt",
		SessionID: "stale-session",
		WorkDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if conn.loadSessCalls != 1 {
		t.Fatalf("LoadSession calls = %d, want 1", conn.loadSessCalls)
	}
	if out.SessionID != "fresh-session" {
		t.Fatalf("SessionID = %q, want %q", out.SessionID, "fresh-session")
	}
}

func TestACPProvider_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	_, err := p.Execute(context.Background(), acp.ACPInput{Prompt: ""})
	if err == nil {
		t.Fatal("expected error for empty prompt")
	}
}

func TestACPProvider_Execute_InitError(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	conn.initErr = errors.New("handshake failed")

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	_, err := p.Execute(context.Background(), acp.ACPInput{
		Prompt:  "prompt",
		WorkDir: t.TempDir(),
	})
	if err == nil || err.Error() != "handshake failed" {
		t.Fatalf("expected handshake error, got %v", err)
	}
}

func TestACPProvider_ExecuteStream_AppliesOperationTimeout(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	conn.newSessID = "sess-timeout"
	close(conn.done)

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	_, err := p.ExecuteStream(context.Background(), acp.ACPInput{
		Prompt:  "prompt",
		WorkDir: t.TempDir(),
		Timeout: 2 * time.Second,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !conn.promptDeadlineSet {
		t.Fatal("expected provider to apply timeout deadline to prompt context")
	}
}

func TestACPProvider_ResumeSession_UsesLoadSession(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	close(conn.done)

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	sessionID, err := p.ResumeSession(context.Background(), acp.ACPInput{
		SessionID: "existing-session",
		WorkDir:   t.TempDir(),
		Timeout:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sessionID != "existing-session" {
		t.Fatalf("sessionID = %q, want %q", sessionID, "existing-session")
	}
	if conn.loadSessCalls != 1 {
		t.Fatalf("LoadSession calls = %d, want 1", conn.loadSessCalls)
	}
}

func TestACPProvider_ResumeSession_FallsBackToNewSession(t *testing.T) {
	t.Parallel()

	conn := newMockConn()
	conn.loadSessSupported = false
	conn.newSessID = "fresh-session"
	close(conn.done)

	spec := acp.AgentSpec{Name: "test-agent", Binary: "test-bin"}
	p := providerWithMock(conn, spec)

	sessionID, err := p.ResumeSession(context.Background(), acp.ACPInput{
		SessionID: "stale-session",
		WorkDir:   t.TempDir(),
		Timeout:   time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if sessionID != "fresh-session" {
		t.Fatalf("sessionID = %q, want %q", sessionID, "fresh-session")
	}
}
