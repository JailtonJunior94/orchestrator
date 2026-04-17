package providers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/acp"
)

// ACPProvider executes prompts through the ACP protocol.
// It is the single provider implementation for all agents in V1.
type ACPProvider interface {
	Name() string
	Execute(ctx context.Context, input acp.ACPInput) (acp.ACPOutput, error)
	ExecuteStream(ctx context.Context, input acp.ACPInput, onUpdate func(acp.TypedUpdate)) (acp.ACPOutput, error)
	ResumeSession(ctx context.Context, input acp.ACPInput) (string, error)
	Available() error
	Close() error
}

// acpProvider is the concrete implementation of ACPProvider.
type acpProvider struct {
	spec        acp.AgentSpec
	registry    *acp.Registry
	logger      *slog.Logger
	connOpts    []acp.ConnectionOption
	fixedConnFn func() (acp.ACPConnection, error) // non-nil only in tests
	storeRoot   string
}

// NewACPProvider creates an ACPProvider for the given spec and registry.
func NewACPProvider(spec acp.AgentSpec, registry *acp.Registry, logger *slog.Logger, connOpts ...acp.ConnectionOption) ACPProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &acpProvider{
		spec:      spec,
		registry:  registry,
		logger:    logger,
		connOpts:  append([]acp.ConnectionOption{}, connOpts...),
		storeRoot: ".",
	}
}

// NewACPProviderWithConn creates an ACPProvider that always uses the provided
// connection instead of dialling a new one. Intended for tests only.
func NewACPProviderWithConn(spec acp.AgentSpec, conn acp.ACPConnection) ACPProvider {
	return &acpProvider{
		spec:        spec,
		logger:      slog.Default(),
		fixedConnFn: func() (acp.ACPConnection, error) { return conn, nil },
	}
}

// Name returns the provider name.
func (p *acpProvider) Name() string {
	return p.spec.Name
}

// Available reports whether the agent binary is resolvable in PATH.
func (p *acpProvider) Available() error {
	if !p.registry.Available(p.spec.Name) {
		return fmt.Errorf("%w: provider %q", acp.ErrAgentNotAvailable, p.spec.Name)
	}
	return nil
}

// Close is a no-op at the provider level; connections are closed per-execution.
func (p *acpProvider) Close() error {
	return nil
}

// Execute sends a prompt to the agent and returns the accumulated output.
func (p *acpProvider) Execute(ctx context.Context, input acp.ACPInput) (acp.ACPOutput, error) {
	return p.ExecuteStream(ctx, input, nil)
}

// ResumeSession verifies or materializes the ACP session that should be reused
// by a continued run. It returns the resolved session ID, which may change when
// the provider falls back to creating a new session.
func (p *acpProvider) ResumeSession(ctx context.Context, input acp.ACPInput) (string, error) {
	if input.SessionID == "" {
		return "", errors.New("acp resume session: session id must not be empty")
	}

	conn, err := p.dial(input)
	if err != nil {
		return "", err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = conn.Close(shutdownCtx)
	}()

	initCtx, cancelInit := withOptionalTimeout(ctx, input.Timeout)
	defer cancelInit()
	if _, err := conn.Initialize(initCtx); err != nil {
		return "", err
	}

	return p.setupSession(ctx, conn, input)
}

// ExecuteStream sends a prompt to the agent, calling onUpdate for each TypedUpdate.
// When onUpdate is nil, updates are still accumulated but not forwarded.
func (p *acpProvider) ExecuteStream(ctx context.Context, input acp.ACPInput, onUpdate func(acp.TypedUpdate)) (acp.ACPOutput, error) {
	if err := input.Validate(); err != nil {
		return acp.ACPOutput{}, err
	}

	conn, err := p.dial(input)
	if err != nil {
		return acp.ACPOutput{}, err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = conn.Close(shutdownCtx)
	}()

	initCtx, cancelInit := withOptionalTimeout(ctx, input.Timeout)
	defer cancelInit()
	if _, err := conn.Initialize(initCtx); err != nil {
		return acp.ACPOutput{}, err
	}

	sessionID, err := p.setupSession(ctx, conn, input)
	if err != nil {
		return acp.ACPOutput{}, err
	}

	promptCtx, cancelPrompt := withOptionalTimeout(ctx, input.Timeout)
	defer cancelPrompt()
	stopReason, err := conn.Prompt(promptCtx, sessionID, input.Prompt)
	if err != nil {
		return acp.ACPOutput{}, err
	}

	return p.drain(conn, sessionID, stopReason, onUpdate)
}

// dial creates and connects a new ACPConnection for this provider.
func (p *acpProvider) dial(input acp.ACPInput) (acp.ACPConnection, error) {
	if p.fixedConnFn != nil {
		return p.fixedConnFn()
	}

	binary, args, err := p.registry.Resolve(p.spec.Name)
	if err != nil {
		return nil, err
	}

	//nolint:gosec // Args are validated through the registry spec.
	cmd := exec.Command(binary, args...)
	opts := append([]acp.ConnectionOption{acp.WithLogger(p.logger)}, p.connOpts...)
	opts = append(opts,
		acp.WithPermissionMetadata(acp.PermissionMetadata{
			Provider: input.ProviderName,
			Workflow: input.WorkflowName,
		}),
		acp.WithPermissionPolicy(input.PermissionPolicy),
		acp.WithFSHandlerFactory(func(cwd string) *acp.FSHandler {
			return acp.NewFSHandler(cwd, p.logger, acp.FSAuditMetadata{
				RunID:     input.RunID,
				Workflow:  input.WorkflowName,
				Step:      input.StepName,
				Provider:  input.ProviderName,
				StoreRoot: p.storeRoot,
			})
		}),
	)
	return acp.NewConnection(cmd, opts...)
}

// setupSession creates a new session or resumes an existing one.
func (p *acpProvider) setupSession(ctx context.Context, conn acp.ACPConnection, input acp.ACPInput) (string, error) {
	workDir := input.WorkDir
	if workDir == "" {
		workDir = "."
	}

	if input.SessionID != "" {
		loadCtx, cancel := withOptionalTimeout(ctx, input.Timeout)
		defer cancel()
		loaded, err := conn.LoadSession(loadCtx, input.SessionID, workDir)
		if err != nil {
			if errors.Is(err, acp.ErrSessionNotFound) || errors.Is(err, acp.ErrSessionExpired) {
				return "", err
			}
			return "", err
		}
		if loaded {
			return input.SessionID, nil
		}
	}

	newSessionCtx, cancel := withOptionalTimeout(ctx, input.Timeout)
	defer cancel()
	return conn.NewSession(newSessionCtx, workDir)
}

// drain reads all updates from the connection until Done, accumulating output.
func (p *acpProvider) drain(conn acp.ACPConnection, sessionID string, stopReason string, onUpdate func(acp.TypedUpdate)) (acp.ACPOutput, error) {
	start := time.Now()
	out := acp.ACPOutput{
		SessionID:  sessionID,
		StopReason: stopReason,
	}

	updates := conn.Updates()
	done := conn.Done()

	for {
		select {
		case update, ok := <-updates:
			if !ok {
				out.Duration = time.Since(start)
				return out, nil
			}
			p.accumulateUpdate(&out, update)
			if onUpdate != nil {
				onUpdate(update)
			}
		case <-done:
			// Drain remaining buffered updates.
			for {
				select {
				case update := <-updates:
					p.accumulateUpdate(&out, update)
					if onUpdate != nil {
						onUpdate(update)
					}
				default:
					out.Duration = time.Since(start)
					return out, nil
				}
			}
		}
	}
}

func withOptionalTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// accumulateUpdate merges a single TypedUpdate into the ACPOutput.
func (p *acpProvider) accumulateUpdate(out *acp.ACPOutput, u acp.TypedUpdate) {
	switch u.Kind {
	case acp.UpdateMessage:
		out.Content += u.Text
	case acp.UpdateThought:
		out.Thoughts += u.Text
	case acp.UpdateToolCall, acp.UpdateToolUpdate:
		if u.ToolCall != nil {
			out.ToolCalls = append(out.ToolCalls, *u.ToolCall)
		}
	}
}
