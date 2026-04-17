package acp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
)

const (
	// defaultHandshakeTimeout is the maximum time allowed for the ACP handshake.
	defaultHandshakeTimeout = 10 * time.Second

	// updateBufferSize is the capacity of the TypedUpdate channel.
	updateBufferSize = 256
)

// ACPConnection manages the lifecycle of a single ACP agent connection.
// It wraps the SDK's ClientSideConnection and owns the subprocess.
type ACPConnection interface {
	// Initialize performs the ACP handshake with the agent.
	// Returns the agent's capabilities or an error if the handshake fails.
	Initialize(ctx context.Context) (*acpsdk.InitializeResponse, error)

	// NewSession creates a fresh session with the given working directory.
	// Returns the opaque session ID assigned by the agent.
	NewSession(ctx context.Context, cwd string) (string, error)

	// LoadSession attempts to resume an existing session by ID.
	// Falls back to creating a new session when the agent does not support LoadSession.
	LoadSession(ctx context.Context, sessionID string, cwd string) (bool, error)

	// Prompt sends a text prompt to the active session.
	Prompt(ctx context.Context, sessionID string, content string) (string, error)

	// Updates returns the read-only channel of typed streaming updates.
	Updates() <-chan TypedUpdate

	// Done returns a channel that closes when the agent peer disconnects.
	Done() <-chan struct{}

	// Close shuts down the connection, draining resources.
	Close(ctx context.Context) error
}

// connection is the concrete implementation of ACPConnection.
type connection struct {
	logger           *slog.Logger
	handshakeTimeout time.Duration
	permission       permissionRequestHandler
	permissionMeta   PermissionMetadata
	permissionPolicy PermissionPolicy
	fsHandlerFactory func(cwd string) *FSHandler

	cmd         *exec.Cmd
	stderr      *bytes.Buffer
	stdinCloser interface{ Close() error } // write end of stdin pipe

	conn              *acpsdk.ClientSideConnection
	client            *acpClient
	agentCapabilities acpsdk.AgentCapabilities
}

// ConnectionOption configures a connection before it is used.
type ConnectionOption func(*connection)

// WithHandshakeTimeout overrides the default handshake timeout.
func WithHandshakeTimeout(d time.Duration) ConnectionOption {
	return func(c *connection) { c.handshakeTimeout = d }
}

// WithLogger injects a structured logger into the connection.
func WithLogger(l *slog.Logger) ConnectionOption {
	return func(c *connection) { c.logger = l }
}

// WithPermissionHandler injects the handler used for ACP permission requests.
func WithPermissionHandler(handler permissionRequestHandler) ConnectionOption {
	return func(c *connection) { c.permission = handler }
}

// WithPermissionMetadata binds workflow/provider metadata to the permission
// handler used by this connection.
func WithPermissionMetadata(metadata PermissionMetadata) ConnectionOption {
	return func(c *connection) { c.permissionMeta = metadata }
}

// WithPermissionPolicy overrides the permission policy for one ACP execution.
func WithPermissionPolicy(policy PermissionPolicy) ConnectionOption {
	return func(c *connection) { c.permissionPolicy = policy }
}

// WithFSHandlerFactory overrides how filesystem callbacks are materialized for
// each session working directory.
func WithFSHandlerFactory(factory func(cwd string) *FSHandler) ConnectionOption {
	return func(c *connection) { c.fsHandlerFactory = factory }
}

// NewConnection creates an ACPConnection for the given command.
// The command must not have been started yet; NewConnection will start it.
func NewConnection(cmd *exec.Cmd, opts ...ConnectionOption) (ACPConnection, error) {
	c := &connection{
		logger:           slog.Default(),
		handshakeTimeout: defaultHandshakeTimeout,
		cmd:              cmd,
		stderr:           new(bytes.Buffer),
	}
	for _, o := range opts {
		o(c)
	}
	if c.permission == nil {
		c.permission = NewPermissionHandler(nil, false, PermissionPolicy{}, c.logger)
	}
	if handler, ok := c.permission.(*PermissionHandler); ok {
		c.permission = handler.WithExecution(c.permissionMeta, c.permissionPolicy)
	}
	if c.fsHandlerFactory == nil {
		c.fsHandlerFactory = func(cwd string) *FSHandler {
			return NewFSHandler(cwd, c.logger)
		}
	}

	cmd.Stderr = c.stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("acp: open stdin pipe: %w", err)
	}
	c.stdinCloser = stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("acp: open stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("acp: start agent process: %w", err)
	}

	c.client = newACPClient(c.logger, nil, c.permission)
	c.conn = acpsdk.NewClientSideConnection(c.client, stdin, stdout)
	c.conn.SetLogger(c.logger)

	return c, nil
}

// Initialize performs the ACP handshake with the agent.
func (c *connection) Initialize(ctx context.Context) (*acpsdk.InitializeResponse, error) {
	tctx, cancel := context.WithTimeout(ctx, c.handshakeTimeout)
	defer cancel()

	c.logger.InfoContext(tctx, "acp_handshake_started",
		slog.Int("protocol_version", int(acpsdk.ProtocolVersionNumber)),
	)

	start := time.Now()
	resp, err := c.conn.Initialize(tctx, acpsdk.InitializeRequest{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		ClientCapabilities: acpsdk.ClientCapabilities{
			Fs: acpsdk.FileSystemCapabilities{
				ReadTextFile:  true,
				WriteTextFile: true,
			},
		},
	})
	if err != nil {
		if errors.Is(tctx.Err(), context.DeadlineExceeded) {
			c.logger.ErrorContext(ctx, "acp_handshake_failed",
				slog.String("error", "timeout"),
			)
			return nil, ErrHandshakeTimeout
		}
		c.logger.ErrorContext(ctx, "acp_handshake_failed",
			slog.String("error", err.Error()),
		)
		return nil, c.wrapError("acp handshake", err)
	}

	if resp.ProtocolVersion != acpsdk.ProtocolVersionNumber {
		c.logger.ErrorContext(ctx, "acp_handshake_failed",
			slog.String("error", "protocol version mismatch"),
			slog.Int("expected", int(acpsdk.ProtocolVersionNumber)),
			slog.Int("received", int(resp.ProtocolVersion)),
		)
		return nil, fmt.Errorf("%w: expected %d, got %d",
			ErrProtocolVersionMismatch,
			acpsdk.ProtocolVersionNumber,
			resp.ProtocolVersion,
		)
	}

	c.agentCapabilities = resp.AgentCapabilities
	c.logger.InfoContext(ctx, "acp_handshake_completed",
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	)

	return &resp, nil
}

// NewSession creates a fresh session with the given working directory.
func (c *connection) NewSession(ctx context.Context, cwd string) (string, error) {
	if c.fsHandlerFactory == nil {
		c.fsHandlerFactory = func(dir string) *FSHandler {
			return NewFSHandler(dir, c.logger)
		}
	}

	resp, err := c.conn.NewSession(ctx, acpsdk.NewSessionRequest{
		Cwd:        cwd,
		McpServers: []acpsdk.McpServer{},
	})
	if err != nil {
		return "", c.wrapError("acp new session", err)
	}

	sessionID := string(resp.SessionId)
	c.client.SetFileHandler(c.fsHandlerFactory(cwd))
	c.logger.InfoContext(ctx, "acp_session_created",
		slog.String("session_id", sessionID),
		slog.String("cwd", cwd),
	)
	return sessionID, nil
}

// LoadSession attempts to resume an existing session in the given working directory.
// It returns loaded=false when the agent does not support session resume so the
// caller can fall back explicitly to NewSession.
func (c *connection) LoadSession(ctx context.Context, sessionID string, cwd string) (bool, error) {
	if c.fsHandlerFactory == nil {
		c.fsHandlerFactory = func(dir string) *FSHandler {
			return NewFSHandler(dir, c.logger)
		}
	}

	if !c.agentCapabilities.LoadSession {
		c.logger.WarnContext(ctx, "acp_session_load_fallback",
			slog.String("session_id", sessionID),
			slog.String("reason", "agent does not support loadSession"),
		)
		return false, nil
	}

	_, err := c.conn.LoadSession(ctx, acpsdk.LoadSessionRequest{
		SessionId:  acpsdk.SessionId(sessionID),
		Cwd:        cwd,
		McpServers: []acpsdk.McpServer{},
	})
	if err != nil {
		return false, c.wrapError("acp load session", mapLoadSessionError(err))
	}

	c.client.SetFileHandler(c.fsHandlerFactory(cwd))
	c.logger.InfoContext(ctx, "acp_session_loaded",
		slog.String("session_id", sessionID),
		slog.String("cwd", cwd),
	)
	return true, nil
}

// Prompt sends a text prompt to the agent session.
func (c *connection) Prompt(ctx context.Context, sessionID string, content string) (string, error) {
	c.logger.InfoContext(ctx, "acp_prompt_sent",
		slog.String("session_id", sessionID),
	)

	resp, err := c.conn.Prompt(ctx, acpsdk.PromptRequest{
		SessionId: acpsdk.SessionId(sessionID),
		Prompt:    []acpsdk.ContentBlock{acpsdk.TextBlock(content)},
	})
	if err != nil {
		return "", c.wrapError("acp prompt", err)
	}
	return string(resp.StopReason), nil
}

// Updates returns the channel of typed streaming updates from the agent.
func (c *connection) Updates() <-chan TypedUpdate {
	return c.client.updates
}

// Done returns a channel that closes when the agent peer disconnects.
func (c *connection) Done() <-chan struct{} {
	return c.conn.Done()
}

// Close shuts down the connection using the graceful shutdown sequence:
// close stdin → SIGTERM (Unix) → SIGKILL.
func (c *connection) Close(ctx context.Context) error {
	close(c.client.done)
	Shutdown(ctx, c.cmd, c.stdinCloser, c.logger)
	return nil
}

func mapLoadSessionError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "not found"), strings.Contains(message, "unknown session"):
		return fmt.Errorf("%w: %v", ErrSessionNotFound, err)
	case strings.Contains(message, "expired"), strings.Contains(message, "evicted"):
		return fmt.Errorf("%w: %v", ErrSessionExpired, err)
	default:
		return err
	}
}

func (c *connection) wrapError(operation string, err error) error {
	if err == nil {
		return nil
	}

	stderr := strings.TrimSpace(c.stderrText())
	if stderr == "" {
		return fmt.Errorf("%s: %w", operation, err)
	}

	return fmt.Errorf("%s: %w; stderr: %s", operation, err, stderr)
}

func (c *connection) stderrText() string {
	if c == nil || c.stderr == nil {
		return ""
	}
	return c.stderr.String()
}
