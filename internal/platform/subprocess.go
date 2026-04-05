package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// CommandResult contains the subprocess execution details.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// StreamResult provides incremental access to subprocess output.
type StreamResult struct {
	Stdout    io.Reader
	Stderr    io.Reader
	Wait      func() error
	ExitCode  func() int
	StartedAt time.Time
}

// CommandRunner executes subprocesses without relying on a shell.
type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, stdin string) (CommandResult, error)
	RunStreaming(ctx context.Context, name string, args []string, stdin string) (*StreamResult, error)
}

// ExecCommandRunner is the production subprocess implementation.
type ExecCommandRunner struct{}

// NewCommandRunner creates a production subprocess runner.
func NewCommandRunner() CommandRunner {
	return ExecCommandRunner{}
}

// Run executes the given binary with explicit arguments.
func (ExecCommandRunner) Run(ctx context.Context, name string, args []string, stdin string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = bytes.NewBufferString(stdin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startedAt := time.Now()
	err := cmd.Run()
	duration := time.Since(startedAt)

	result := CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCodeFromError(err),
		Duration: duration,
	}

	if err != nil {
		return result, err
	}

	return result, nil
}

// RunStreaming starts the process and returns immediately with readers for
// stdout and stderr. The caller must consume both readers and call Wait()
// to collect the exit code and release resources.
func (ExecCommandRunner) RunStreaming(ctx context.Context, name string, args []string, stdin string) (*StreamResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe for %q: %w", name, err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stderr pipe for %q: %w", name, err)
	}

	startedAt := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting %q: %w", name, err)
	}

	exitCode := 0
	result := &StreamResult{
		Stdout:    stdoutPipe,
		Stderr:    stderrPipe,
		StartedAt: startedAt,
		Wait: func() error {
			waitErr := cmd.Wait()
			if waitErr != nil {
				exitCode = exitCodeFromError(waitErr)
				return waitErr
			}
			return nil
		},
		ExitCode: func() int { return exitCode },
	}

	return result, nil
}

func exitCodeFromError(err error) int {
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return -1
}
