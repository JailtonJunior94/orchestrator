//go:build windows

package acp

import (
	"context"
	"log/slog"
	"os/exec"
	"time"
)

const (
	// defaultStdinCloseTimeout is the time to wait for the agent to exit after stdin is closed.
	defaultStdinCloseTimeout = 5 * time.Second

	// defaultSIGTERMTimeout is unused on Windows but kept for API consistency.
	defaultSIGTERMTimeout = 3 * time.Second
)

// ShutdownOptions configures timeouts for the graceful shutdown sequence.
type ShutdownOptions struct {
	StdinCloseTimeout time.Duration
	// SIGTERMTimeout is ignored on Windows (no SIGTERM support).
	SIGTERMTimeout time.Duration
}

// defaultShutdownOptions returns the default shutdown timeouts.
func defaultShutdownOptions() ShutdownOptions {
	return ShutdownOptions{
		StdinCloseTimeout: defaultStdinCloseTimeout,
		SIGTERMTimeout:    defaultSIGTERMTimeout,
	}
}

// Shutdown performs a graceful shutdown of an ACP agent subprocess on Windows.
// Windows does not support SIGTERM/SIGKILL. The sequence is:
//  1. Close the stdin pipe to signal cooperative termination.
//  2. Wait up to StdinCloseTimeout for the process to exit.
//  3. Call Process.Kill() (maps to TerminateProcess on Windows).
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

	done := waitAsync(cmd)

	// Step 2: wait for cooperative exit.
	select {
	case <-done:
		return
	case <-time.After(o.StdinCloseTimeout):
	}

	// Step 3: force kill (Windows has no SIGTERM).
	logger.WarnContext(ctx, "acp_shutdown_escalated",
		slog.String("signal", "Kill"),
		slog.String("agent_binary", binary),
	)
	_ = cmd.Process.Kill()
	<-done
}
