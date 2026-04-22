//go:build windows

package taskloop

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"time"
)

func runCmd(ctx context.Context, workDir string, liveOut io.Writer, name string, args ...string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	cmd.Env = cleanEnv()

	// No Windows nao ha suporte a grupos de processos Unix; apenas mata o processo pai.
	cmd.Cancel = func() error {
		return cmd.Process.Kill()
	}

	// Fallback: forca drenagem dos pipes se ainda abertos 10s apos o Kill.
	cmd.WaitDelay = 10 * time.Second

	var stdoutBuf, stderrBuf bytes.Buffer
	if liveOut != nil {
		cmd.Stdout = io.MultiWriter(&stdoutBuf, liveOut)
		cmd.Stderr = io.MultiWriter(&stderrBuf, liveOut)
	} else {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	}

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return stdoutBuf.String(), stderrBuf.String(), -1, err
		}
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode, nil
}
