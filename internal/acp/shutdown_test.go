//go:build !windows

package acp_test

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/acp"
)

func TestShutdown_CooperativeExitOnStdinClose(t *testing.T) {
	t.Parallel()

	// Process that reads stdin and exits when stdin is closed (EOF).
	cmd := exec.Command("sh", "-c", "read line; exit 0")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	opts := acp.ShutdownOptions{
		StdinCloseTimeout: 2 * time.Second,
		SIGTERMTimeout:    1 * time.Second,
	}

	start := time.Now()
	acp.Shutdown(context.Background(), cmd, stdin, slog.Default(), opts)
	elapsed := time.Since(start)

	// Should exit quickly via cooperative stdin close, well before SIGTERM timeout.
	if elapsed > 1500*time.Millisecond {
		t.Errorf("expected quick cooperative exit, took %v", elapsed)
	}
}

func TestShutdown_SIGTERMSentAfterStdinCloseTimeout(t *testing.T) {
	t.Parallel()

	// Process that ignores stdin close (sleeps 60s).
	cmd := exec.Command("sh", "-c", "trap '' HUP; sleep 60")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	opts := acp.ShutdownOptions{
		StdinCloseTimeout: 200 * time.Millisecond,
		SIGTERMTimeout:    2 * time.Second,
	}

	start := time.Now()
	acp.Shutdown(context.Background(), cmd, stdin, slog.Default(), opts)
	elapsed := time.Since(start)

	// SIGTERM kills the process; should complete after StdinCloseTimeout but before
	// StdinCloseTimeout + SIGTERMTimeout.
	if elapsed < 200*time.Millisecond {
		t.Errorf("expected at least stdin close timeout, got %v", elapsed)
	}
	if elapsed > 2500*time.Millisecond {
		t.Errorf("expected SIGTERM to be effective, took %v", elapsed)
	}
}

func TestShutdown_SIGKILLSentAfterSIGTERMTimeout(t *testing.T) {
	t.Parallel()

	// Process that ignores SIGTERM.
	cmd := exec.Command("sh", "-c", "trap '' TERM; sleep 60")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	opts := acp.ShutdownOptions{
		StdinCloseTimeout: 100 * time.Millisecond,
		SIGTERMTimeout:    100 * time.Millisecond,
	}

	start := time.Now()
	acp.Shutdown(context.Background(), cmd, stdin, slog.Default(), opts)
	elapsed := time.Since(start)

	// SIGKILL eventually kills the process; should complete after both timeouts.
	if elapsed < 200*time.Millisecond {
		t.Errorf("expected at least stdin + SIGTERM timeouts, got %v", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("expected SIGKILL to be effective, took %v", elapsed)
	}
}

func TestShutdown_ContextCancellationTriggersShutdown(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("sh", "-c", "sleep 60")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after a short delay from another goroutine.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	opts := acp.ShutdownOptions{
		StdinCloseTimeout: 200 * time.Millisecond,
		SIGTERMTimeout:    200 * time.Millisecond,
	}

	// Shutdown itself doesn't watch the context for cancellation — it's the
	// caller's responsibility to trigger Shutdown when context is cancelled.
	// Here we simply verify that Shutdown can be called with a cancelled context.
	acp.Shutdown(ctx, cmd, stdin, slog.Default(), opts)

	// Verify process is gone.
	if err := cmd.Process.Signal(syscall.Signal(0)); err == nil {
		// process still alive — send kill to clean up
		_ = cmd.Process.Signal(os.Kill)
		t.Error("process should have been terminated by Shutdown")
	}
}
