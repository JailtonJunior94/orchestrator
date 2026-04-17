package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	acpsdk "github.com/coder/acp-go-sdk"
)

// acpClient implements acp.Client from the SDK.
// It receives callbacks from the agent and routes them to typed channels.
type acpClient struct {
	logger *slog.Logger

	updates chan TypedUpdate
	done    chan struct{}

	mu                sync.RWMutex
	fileHandler       fileRequestHandler
	permissionHandler permissionRequestHandler
}

type fileRequestHandler interface {
	ReadTextFile(ctx context.Context, params acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error)
	WriteTextFile(ctx context.Context, params acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error)
}

type permissionRequestHandler interface {
	RequestPermission(ctx context.Context, params acpsdk.RequestPermissionRequest) (acpsdk.RequestPermissionResponse, error)
}

// compile-time interface check.
var _ acpsdk.Client = (*acpClient)(nil)

// newACPClient constructs an acpClient with buffered update delivery and
// optional filesystem/permission handlers for ACP callbacks.
func newACPClient(logger *slog.Logger, fileHandler fileRequestHandler, permissionHandler permissionRequestHandler) *acpClient {
	client := &acpClient{
		logger:  logger,
		updates: make(chan TypedUpdate, updateBufferSize),
		done:    make(chan struct{}),
	}
	client.fileHandler = fileHandler
	client.permissionHandler = permissionHandler
	return client
}

// SetFileHandler swaps the handler used for filesystem callbacks.
func (c *acpClient) SetFileHandler(handler fileRequestHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fileHandler = handler
}

// SetPermissionHandler swaps the handler used for permission callbacks.
func (c *acpClient) SetPermissionHandler(handler permissionRequestHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.permissionHandler = handler
}

func (c *acpClient) getFileHandler() fileRequestHandler {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fileHandler
}

func (c *acpClient) getPermissionHandler() permissionRequestHandler {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.permissionHandler
}

// SessionUpdate receives streaming notifications from the agent and routes them
// to the typed update channel.
func (c *acpClient) SessionUpdate(_ context.Context, params acpsdk.SessionNotification) error {
	u := params.Update
	var update TypedUpdate

	switch {
	case u.AgentMessageChunk != nil:
		text := extractText(u.AgentMessageChunk.Content)
		update = TypedUpdate{Kind: UpdateMessage, Text: text}

	case u.AgentThoughtChunk != nil:
		text := extractText(u.AgentThoughtChunk.Content)
		update = TypedUpdate{Kind: UpdateThought, Text: text}

	case u.ToolCall != nil:
		tc := u.ToolCall
		update = TypedUpdate{
			Kind: UpdateToolCall,
			ToolCall: &ToolCallInfo{
				ID:        string(tc.ToolCallId),
				Title:     tc.Title,
				Kind:      string(tc.Kind),
				Status:    string(tc.Status),
				Input:     marshalACPValue(tc.RawInput),
				Output:    marshalACPValue(tc.RawOutput),
				Locations: flattenLocations(tc.Locations),
			},
		}

	case u.ToolCallUpdate != nil:
		tcu := u.ToolCallUpdate
		info := &ToolCallInfo{
			ID: string(tcu.ToolCallId),
		}
		if tcu.Title != nil {
			info.Title = *tcu.Title
		}
		if tcu.Kind != nil {
			info.Kind = string(*tcu.Kind)
		}
		if tcu.Status != nil {
			info.Status = string(*tcu.Status)
		}
		if tcu.RawInput != nil {
			info.Input = marshalACPValue(tcu.RawInput)
		}
		if tcu.RawOutput != nil {
			info.Output = marshalACPValue(tcu.RawOutput)
		}
		if len(tcu.Locations) > 0 {
			info.Locations = flattenLocations(tcu.Locations)
		}
		update = TypedUpdate{Kind: UpdateToolUpdate, ToolCall: info}

	default:
		// Ignore unknown or unsupported update kinds.
		return nil
	}

	c.logger.Debug("acp_update_received",
		slog.String("update_kind", string(update.Kind)),
	)

	select {
	case c.updates <- update:
	case <-c.done:
	}
	return nil
}

// ReadTextFile is called by the agent to request file content from the host.
// The real implementation is delegated to FSHandler; this stub returns an error
// so that the engine can inject the actual handler via the fs_handler.go adapter.
func (c *acpClient) ReadTextFile(ctx context.Context, params acpsdk.ReadTextFileRequest) (acpsdk.ReadTextFileResponse, error) {
	handler := c.getFileHandler()
	if handler != nil {
		return handler.ReadTextFile(ctx, params)
	}
	return acpsdk.ReadTextFileResponse{}, fmt.Errorf("ReadTextFile not configured for path %q", params.Path)
}

// WriteTextFile is called by the agent to write content to a file on the host.
// The real implementation is delegated to FSHandler.
func (c *acpClient) WriteTextFile(ctx context.Context, params acpsdk.WriteTextFileRequest) (acpsdk.WriteTextFileResponse, error) {
	handler := c.getFileHandler()
	if handler != nil {
		return handler.WriteTextFile(ctx, params)
	}
	return acpsdk.WriteTextFileResponse{}, fmt.Errorf("WriteTextFile not configured for path %q", params.Path)
}

// RequestPermission is called by the agent before performing a sensitive
// operation. When a permission handler is configured, it owns the decision.
// Otherwise the client falls back to auto-approving the first available option
// while emitting a TypedUpdate for the TUI.
func (c *acpClient) RequestPermission(ctx context.Context, params acpsdk.RequestPermissionRequest) (acpsdk.RequestPermissionResponse, error) {
	handler := c.getPermissionHandler()
	if handler != nil {
		return handler.RequestPermission(ctx, params)
	}

	title := ""
	if params.ToolCall.Title != nil {
		title = *params.ToolCall.Title
	}

	text := title
	if text == "" {
		text = "unknown operation"
	}

	update := TypedUpdate{Kind: UpdatePermission, Text: text}
	select {
	case c.updates <- update:
	case <-c.done:
	}

	if len(params.Options) == 0 {
		return acpsdk.RequestPermissionResponse{
			Outcome: acpsdk.RequestPermissionOutcome{
				Cancelled: &acpsdk.RequestPermissionOutcomeCancelled{},
			},
		}, nil
	}

	return acpsdk.RequestPermissionResponse{
		Outcome: acpsdk.RequestPermissionOutcome{
			Selected: &acpsdk.RequestPermissionOutcomeSelected{
				OptionId: params.Options[0].OptionId,
			},
		},
	}, nil
}

// CreateTerminal is required by the Client interface; not used in V1.
func (c *acpClient) CreateTerminal(_ context.Context, params acpsdk.CreateTerminalRequest) (acpsdk.CreateTerminalResponse, error) {
	return acpsdk.CreateTerminalResponse{}, fmt.Errorf("CreateTerminal not supported")
}

// KillTerminal is required by the Client interface; not used in V1.
func (c *acpClient) KillTerminal(_ context.Context, _ acpsdk.KillTerminalRequest) (acpsdk.KillTerminalResponse, error) {
	return acpsdk.KillTerminalResponse{}, fmt.Errorf("KillTerminal not supported")
}

// TerminalOutput is required by the Client interface; not used in V1.
func (c *acpClient) TerminalOutput(_ context.Context, _ acpsdk.TerminalOutputRequest) (acpsdk.TerminalOutputResponse, error) {
	return acpsdk.TerminalOutputResponse{}, fmt.Errorf("TerminalOutput not supported")
}

// ReleaseTerminal is required by the Client interface; not used in V1.
func (c *acpClient) ReleaseTerminal(_ context.Context, _ acpsdk.ReleaseTerminalRequest) (acpsdk.ReleaseTerminalResponse, error) {
	return acpsdk.ReleaseTerminalResponse{}, fmt.Errorf("ReleaseTerminal not supported")
}

// WaitForTerminalExit is required by the Client interface; not used in V1.
func (c *acpClient) WaitForTerminalExit(_ context.Context, _ acpsdk.WaitForTerminalExitRequest) (acpsdk.WaitForTerminalExitResponse, error) {
	return acpsdk.WaitForTerminalExitResponse{}, fmt.Errorf("WaitForTerminalExit not supported")
}

// extractText returns the text content from a ContentBlock, or empty string if not a text block.
func extractText(block acpsdk.ContentBlock) string {
	if block.Text != nil {
		return block.Text.Text
	}
	return ""
}

func marshalACPValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case nil:
		return ""
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}

func flattenLocations(locations []acpsdk.ToolCallLocation) []string {
	paths := make([]string, 0, len(locations))
	for _, location := range locations {
		if location.Path == "" {
			continue
		}
		paths = append(paths, location.Path)
	}
	return paths
}
