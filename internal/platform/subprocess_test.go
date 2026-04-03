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
