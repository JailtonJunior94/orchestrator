package platform

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"
)

func TestExecCommandRunner(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		wantCode int
		wantErr  bool
	}{
		{name: "success", mode: "success", wantCode: 0},
		{name: "failure", mode: "failure", wantCode: 3, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GO_WANT_HELPER_PROCESS", "1")

			runner := NewCommandRunner()
			result, err := runner.Run(context.Background(), os.Args[0], []string{"-test.run=TestHelperProcess", "--", tt.mode}, "stdin-value")
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ExitCode != tt.wantCode {
				t.Fatalf("exit code = %d, want %d", result.ExitCode, tt.wantCode)
			}
			if tt.mode == "success" && result.Stdout != "stdout:stdin-value" {
				t.Fatalf("stdout = %q", result.Stdout)
			}
			if tt.mode == "failure" && result.Stderr != "stderr:boom" {
				t.Fatalf("stderr = %q", result.Stderr)
			}
		})
	}
}

func TestExecCommandRunnerTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	runner := NewCommandRunner()
	result, err := runner.Run(ctx, os.Args[0], []string{"-test.run=TestHelperProcess", "--", "sleep"}, "")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if result.ExitCode != -1 {
		t.Fatalf("exit code = %d, want -1", result.ExitCode)
	}
}

func TestFakeClockAdvance(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	clock := NewFakeClock(start)
	clock.Advance(2 * time.Minute)
	if got := clock.Now(); !got.Equal(start.Add(2 * time.Minute)) {
		t.Fatalf("now = %v", got)
	}
}

func TestFakeFileSystem(t *testing.T) {
	t.Parallel()

	fileSystem, err := NewFakeFileSystem()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = fileSystem.Close()
	}()

	if err := fileSystem.WriteFile("nested/file.txt", []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := fileSystem.ReadFile("nested/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q", data)
	}
}

func TestHelperProcess(t *testing.T) {
	t.Helper()
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for idx, arg := range args {
		if arg == "--" {
			args = args[idx+1:]
			break
		}
	}

	mode := args[0]
	switch mode {
	case "success":
		input, _ := io.ReadAll(os.Stdin)
		_, _ = os.Stdout.WriteString("stdout:" + string(input))
		os.Exit(0)
	case "failure":
		_, _ = os.Stderr.WriteString("stderr:boom")
		os.Exit(3)
	case "sleep":
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	default:
		panic(errors.New("unknown helper mode"))
	}
}

// --- Task 3.0 tests: RunStreaming ---

func TestRunStreamingReadsOutputIncrementally(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	runner := NewCommandRunner()
	result, err := runner.RunStreaming(context.Background(), os.Args[0], []string{"-test.run=TestHelperProcess", "--", "success"}, "world")
	if err != nil {
		t.Fatalf("RunStreaming error: %v", err)
	}

	data, err := io.ReadAll(result.Stdout)
	if err != nil {
		t.Fatalf("reading stdout: %v", err)
	}

	if err := result.Wait(); err != nil {
		t.Fatalf("Wait error: %v", err)
	}

	if string(data) != "stdout:world" {
		t.Fatalf("stdout = %q; want %q", string(data), "stdout:world")
	}
	if result.ExitCode() != 0 {
		t.Fatalf("exit code = %d; want 0", result.ExitCode())
	}
}

func TestRunStreamingWaitReturnsExitCode(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	runner := NewCommandRunner()
	result, err := runner.RunStreaming(context.Background(), os.Args[0], []string{"-test.run=TestHelperProcess", "--", "failure"}, "")
	if err != nil {
		t.Fatalf("RunStreaming error: %v", err)
	}

	// Drain readers before Wait to avoid broken pipe.
	_, _ = io.ReadAll(result.Stdout)
	stderrData, _ := io.ReadAll(result.Stderr)

	waitErr := result.Wait()
	if waitErr == nil {
		t.Fatal("expected non-nil error from Wait for failing process")
	}

	if string(stderrData) != "stderr:boom" {
		t.Fatalf("stderr = %q; want %q", string(stderrData), "stderr:boom")
	}
}

func TestRunStreamingContextCancellation(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	runner := NewCommandRunner()
	result, err := runner.RunStreaming(ctx, os.Args[0], []string{"-test.run=TestHelperProcess", "--", "sleep"}, "")
	if err != nil {
		t.Fatalf("RunStreaming error: %v", err)
	}

	_, _ = io.ReadAll(result.Stdout)
	waitErr := result.Wait()
	if waitErr == nil {
		t.Fatal("expected error after context cancellation")
	}
}

func TestFakeRunStreamingReturnsControlledData(t *testing.T) {
	t.Parallel()

	fake := FakeCommandRunner{
		RunStreamingFunc: func(_ context.Context, _ string, _ []string, _ string) (*StreamResult, error) {
			return NewFakeStreamResult("hello streaming", ""), nil
		},
	}

	result, err := fake.RunStreaming(context.Background(), "echo", []string{"hello"}, "")
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	data, err := io.ReadAll(result.Stdout)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if string(data) != "hello streaming" {
		t.Fatalf("stdout = %q; want %q", string(data), "hello streaming")
	}

	if err := result.Wait(); err != nil {
		t.Fatalf("wait error: %v", err)
	}
}
