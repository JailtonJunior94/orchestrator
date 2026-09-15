//go:build !windows

package taskloop

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunCmdReportsTimeoutInsteadOfSilentExitCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	start := time.Now()
	stdout, stderr, exitCode, err := NewCatalog().runCmd(ctx, ".", nil, "sh", "-c", "sleep 30")
	elapsed := time.Since(start)

	if elapsed > 10*time.Second {
		t.Fatalf("runCmd did not interrupt the child process: elapsed %s", elapsed)
	}
	if !errors.Is(err, ErrAgentTimeout) {
		t.Errorf("err = %v, want ErrAgentTimeout so the caller can tell a timeout from an empty output", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err = %v, want the wrapped context.DeadlineExceeded", err)
	}
	if exitCode != -1 {
		t.Errorf("exitCode = %d, want -1", exitCode)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}
}

func TestRunCmdKeepsExitCodeWhenThereIsNoTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stdout, _, exitCode, err := NewCatalog().runCmd(ctx, ".", nil, "sh", "-c", "printf done; exit 3")
	if err != nil {
		t.Fatalf("runCmd: %v", err)
	}
	if exitCode != 3 {
		t.Errorf("exitCode = %d, want 3", exitCode)
	}
	if stdout != "done" {
		t.Errorf("stdout = %q, want %q", stdout, "done")
	}
}
