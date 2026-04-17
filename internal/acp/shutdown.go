//go:build !windows

package acp

import (
	"context"
	"log/slog"
	"os/exec"
	"syscall"
	"time"
)

const (
	// defaultStdinCloseTimeout is the time to wait for the agent to exit after stdin is closed.
	defaultStdinCloseTimeout = 5 * time.Second

	// defaultSIGTERMTimeout is the time to wait for the agent to exit after SIGTERM.
	defaultSIGTERMTimeout = 3 * time.Second
)

// ShutdownOptions configures timeouts for the graceful shutdown sequence.
type ShutdownOptions struct {
	StdinCloseTimeout time.Duration
	SIGTERMTimeout    time.Duration
}

// defaultShutdownOptions returns the default shutdown timeouts.
func defaultShutdownOptions() ShutdownOptions {
	return ShutdownOptions{
		StdinCloseTimeout: defaultStdinCloseTimeout,
		SIGTERMTimeout:    defaultSIGTERMTimeout,
	}
}

// Shutdown performs a graceful shutdown of an ACP agent subprocess on Unix systems.
// The sequence is:
//  1. Close the stdin pipe to signal cooperative termination.
//  2. Wait up to StdinCloseTimeout for the process to exit.
//  3. Send SIGTERM if still running, wait up to SIGTERMTimeout.
//  4. Send SIGKILL as a last resort.
//
// stdinCloser should be the WriteCloser end of the stdin pipe opened before
// the process was started. It may be nil if stdin was not piped.
func Shutdown(ctx context.Context, cmd *exec.Cmd, stdinCloser interface{ Close() error }, logger *slog.Logger, opts ...ShutdownOptions) {
	o := defaultShutdownOptions()
	if len(opts) > 0 {
		o = opts[0]
	}

	binary := cmd.Path
	logger.InfoContext(ctx, "acp_shutdown_started",
		slog.String("agent_binary", binary),
	)

	// Step 1: close stdin cooperatively.
	if stdinCloser != nil {
		_ = stdinCloser.Close()
	}

	// waitDone closes when the process exits (or was already exited).
	done := waitAsync(cmd)

	// Step 2: wait for cooperative exit.
	select {
	case <-done:
		return
	case <-time.After(o.StdinCloseTimeout):
	}

	// Step 3: escalate to SIGTERM.
	logger.WarnContext(ctx, "acp_shutdown_escalated",
		slog.String("signal", "SIGTERM"),
		slog.String("agent_binary", binary),
	)
	_ = cmd.Process.Signal(syscall.SIGTERM)

	select {
	case <-done:
		return
	case <-time.After(o.SIGTERMTimeout):
	}

	// Step 4: escalate to SIGKILL.
	logger.WarnContext(ctx, "acp_shutdown_escalated",
		slog.String("signal", "SIGKILL"),
		slog.String("agent_binary", binary),
	)
	_ = cmd.Process.Kill()
	<-done
}
