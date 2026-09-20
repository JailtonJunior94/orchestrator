package qualitygate

import (
	"context"
	"testing"
)

func TestShellExecutor_CapturesExitCodeAndOutput(t *testing.T) {
	executor := NewShellExecutor()

	ok, err := executor.Execute(context.Background(), t.TempDir(), "echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", ok.ExitCode)
	}

	failed, err := executor.Execute(context.Background(), t.TempDir(), "exit 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if failed.ExitCode != 3 {
		t.Fatalf("ExitCode = %d, want 3", failed.ExitCode)
	}
}
