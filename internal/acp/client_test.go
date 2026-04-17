package acp

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
)

type stubFileHandler struct {
	readPath  string
	writePath string
}

func (s *stubFileHandler) ReadTextFile(_ context.Context, params acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error) {
	s.readPath = params.Path
	return acpsdk.ReadTextFileResponse{Content: "file-body"}, nil
}

func (s *stubFileHandler) WriteTextFile(_ context.Context, params acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error) {
	s.writePath = params.Path
	return acpsdk.WriteTextFileResponse{}, nil
}

type stubPermissionHandler struct {
	selected acpsdk.PermissionOptionId
}

func (s *stubPermissionHandler) RequestPermission(_ context.Context, params acpsdk.RequestPermissionRequest) (acpsdk.RequestPermissionResponse, error) {
	if len(params.Options) == 0 {
		return acpsdk.RequestPermissionResponse{}, errors.New("no options")
	}
	s.selected = params.Options[0].OptionId
	return acpsdk.RequestPermissionResponse{
		Outcome: acpsdk.NewRequestPermissionOutcomeSelected(params.Options[0].OptionId),
	}, nil
}

// --- SessionUpdate routing tests ---

func TestACPClient_SessionUpdate_MessageChunk(t *testing.T) {
	c := newACPClient(slog.Default(), nil, nil)
	notif := acpsdk.SessionNotification{
		Update: acpsdk.UpdateAgentMessageText("hello world"),
	}

	if err := c.SessionUpdate(context.Background(), notif); err != nil {
		t.Fatalf("SessionUpdate: %v", err)
	}

	select {
	case u := <-c.updates:
		if u.Kind != UpdateMessage {
			t.Errorf("Kind = %q, want %q", u.Kind, UpdateMessage)
		}
		if u.Text != "hello world" {
			t.Errorf("Text = %q, want %q", u.Text, "hello world")
		}
	default:
		t.Fatal("no update received")
	}
}

func TestACPClient_SessionUpdate_ThoughtChunk(t *testing.T) {
	c := newACPClient(slog.Default(), nil, nil)
	notif := acpsdk.SessionNotification{
		Update: acpsdk.UpdateAgentThoughtText("thinking..."),
	}

	if err := c.SessionUpdate(context.Background(), notif); err != nil {
		t.Fatalf("SessionUpdate: %v", err)
	}

	select {
	case u := <-c.updates:
		if u.Kind != UpdateThought {
			t.Errorf("Kind = %q, want %q", u.Kind, UpdateThought)
		}
		if u.Text != "thinking..." {
			t.Errorf("Text = %q, want %q", u.Text, "thinking...")
		}
	default:
		t.Fatal("no update received")
	}
}

func TestACPClient_SessionUpdate_ToolCall(t *testing.T) {
	c := newACPClient(slog.Default(), nil, nil)
	kind := acpsdk.ToolKind("shell")
	status := acpsdk.ToolCallStatus("running")
	notif := acpsdk.SessionNotification{
		Update: acpsdk.SessionUpdate{
			ToolCall: &acpsdk.SessionUpdateToolCall{
				ToolCallId:    "tc-1",
				Title:         "run shell",
				Kind:          kind,
				Status:        status,
				RawInput:      map[string]any{"command": "go test ./..."},
				Locations:     []acpsdk.ToolCallLocation{{Path: "internal/runtime/engine.go"}},
				SessionUpdate: "tool_call",
			},
		},
	}

	if err := c.SessionUpdate(context.Background(), notif); err != nil {
		t.Fatalf("SessionUpdate: %v", err)
	}

	select {
	case u := <-c.updates:
		if u.Kind != UpdateToolCall {
			t.Errorf("Kind = %q, want %q", u.Kind, UpdateToolCall)
		}
		if u.ToolCall == nil {
			t.Fatal("ToolCall is nil")
		}
		if u.ToolCall.ID != "tc-1" {
			t.Errorf("ID = %q, want %q", u.ToolCall.ID, "tc-1")
		}
		if u.ToolCall.Input != `{"command":"go test ./..."}` {
			t.Errorf("Input = %q", u.ToolCall.Input)
		}
		if len(u.ToolCall.Locations) != 1 || u.ToolCall.Locations[0] != "internal/runtime/engine.go" {
			t.Errorf("Locations = %#v", u.ToolCall.Locations)
		}
	default:
		t.Fatal("no update received")
	}
}

func TestACPClient_SessionUpdate_ToolCallUpdate(t *testing.T) {
	c := newACPClient(slog.Default(), nil, nil)
	title := "updated title"
	kind := acpsdk.ToolKind("shell")
	status := acpsdk.ToolCallStatus("done")
	rawOutput := map[string]any{"exit_code": 0}
	notif := acpsdk.SessionNotification{
		Update: acpsdk.SessionUpdate{
			ToolCallUpdate: &acpsdk.SessionToolCallUpdate{
				ToolCallId:    "tc-2",
				Title:         &title,
				Kind:          &kind,
				Status:        &status,
				RawOutput:     rawOutput,
				Locations:     []acpsdk.ToolCallLocation{{Path: "internal/acp/client.go"}},
				SessionUpdate: "tool_call_update",
			},
		},
	}

	if err := c.SessionUpdate(context.Background(), notif); err != nil {
		t.Fatalf("SessionUpdate: %v", err)
	}

	select {
	case u := <-c.updates:
		if u.Kind != UpdateToolUpdate {
			t.Errorf("Kind = %q, want %q", u.Kind, UpdateToolUpdate)
		}
		if u.ToolCall == nil {
			t.Fatal("ToolCall is nil")
		}
		if u.ToolCall.ID != "tc-2" {
			t.Errorf("ID = %q, want %q", u.ToolCall.ID, "tc-2")
		}
		if u.ToolCall.Title != "updated title" {
			t.Errorf("Title = %q, want %q", u.ToolCall.Title, "updated title")
		}
		if u.ToolCall.Output != `{"exit_code":0}` {
			t.Errorf("Output = %q", u.ToolCall.Output)
		}
		if len(u.ToolCall.Locations) != 1 || u.ToolCall.Locations[0] != "internal/acp/client.go" {
			t.Errorf("Locations = %#v", u.ToolCall.Locations)
		}
	default:
		t.Fatal("no update received")
	}
}

func TestACPClient_SessionUpdate_UnknownKind_Ignored(t *testing.T) {
	c := newACPClient(slog.Default(), nil, nil)
	// Empty update (unknown kind) should be silently dropped.
	notif := acpsdk.SessionNotification{Update: acpsdk.SessionUpdate{}}
	if err := c.SessionUpdate(context.Background(), notif); err != nil {
		t.Fatalf("SessionUpdate: %v", err)
	}
	select {
	case u := <-c.updates:
		t.Errorf("unexpected update: %+v", u)
	case <-time.After(10 * time.Millisecond):
		// expected: no update
	}
}

func TestACPClient_Done_ClosesChannel(t *testing.T) {
	c := newACPClient(slog.Default(), nil, nil)
	close(c.done)

	// Sending to a closed client should not block (update is dropped).
	notif := acpsdk.SessionNotification{
		Update: acpsdk.UpdateAgentMessageText("dropped"),
	}
	done := make(chan struct{})
	go func() {
		_ = c.SessionUpdate(context.Background(), notif)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("SessionUpdate blocked after done closed")
	}
}

func TestACPClient_ReadTextFile_UsesInjectedHandler(t *testing.T) {
	t.Parallel()

	handler := &stubFileHandler{}
	c := newACPClient(slog.Default(), handler, nil)

	resp, err := c.ReadTextFile(context.Background(), acpsdk.ReadTextFileRequest{Path: "notes.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "file-body" {
		t.Fatalf("content = %q, want %q", resp.Content, "file-body")
	}
	if handler.readPath != "notes.txt" {
		t.Fatalf("readPath = %q, want %q", handler.readPath, "notes.txt")
	}
}

func TestACPClient_WriteTextFile_UsesInjectedHandler(t *testing.T) {
	t.Parallel()

	handler := &stubFileHandler{}
	c := newACPClient(slog.Default(), handler, nil)

	if _, err := c.WriteTextFile(context.Background(), acpsdk.WriteTextFileRequest{Path: "result.txt", Content: "ok"}); err != nil {
		t.Fatal(err)
	}
	if handler.writePath != "result.txt" {
		t.Fatalf("writePath = %q, want %q", handler.writePath, "result.txt")
	}
}

func TestACPClient_RequestPermission_UsesInjectedHandler(t *testing.T) {
	t.Parallel()

	handler := &stubPermissionHandler{}
	c := newACPClient(slog.Default(), nil, handler)

	resp, err := c.RequestPermission(context.Background(), acpsdk.RequestPermissionRequest{
		Options: []acpsdk.PermissionOption{
			{OptionId: "allow", Kind: acpsdk.PermissionOptionKindAllowOnce, Name: "Allow"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Outcome.Selected == nil {
		t.Fatal("expected selected outcome")
	}
	if handler.selected != "allow" {
		t.Fatalf("selected = %q, want %q", handler.selected, "allow")
	}
}
