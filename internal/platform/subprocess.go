package platform

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

// CommandResult contains the subprocess execution details.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// CommandRunner executes subprocesses without relying on a shell.
type CommandRunner interface {
	Run(ctx context.Context, name string, args []string, stdin string) (CommandResult, error)
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
