package qualitygate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

type ExecutionResult struct {
	Command  string
	ExitCode int
	Output   string
}

type Executor interface {
	Execute(ctx context.Context, dir, command string) (ExecutionResult, error)
}

type shellExecutor struct{}

var _ Executor = shellExecutor{}

func NewShellExecutor() Executor {
	return shellExecutor{}
}

func (shellExecutor) Execute(ctx context.Context, dir, command string) (ExecutionResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	if err == nil {
		return ExecutionResult{Command: command, ExitCode: 0, Output: buf.String()}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return ExecutionResult{Command: command, ExitCode: exitErr.ExitCode(), Output: buf.String()}, nil
	}

	return ExecutionResult{Command: command, ExitCode: -1, Output: buf.String()}, fmt.Errorf("qualitygate: execute command %q: %w", command, err)
}
