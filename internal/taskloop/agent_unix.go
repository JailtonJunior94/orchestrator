//go:build !windows

package taskloop

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"syscall"
	"time"
)

func runCmd(ctx context.Context, workDir string, liveOut io.Writer, name string, args ...string) (string, string, int, error) {
	return runCmdMonitored(ctx, workDir, liveOut, nil, name, args...)
}

func runCmdMonitored(
	ctx context.Context,
	workDir string,
	liveOut io.Writer,
	onStart func(),
	name string,
	args ...string,
) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	cmd.Env = cleanEnv()

	// Cria novo grupo de processos para que filhos do agente sejam incluidos no kill.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Ao cancelar o contexto, mata o grupo inteiro em vez de apenas o processo pai.
	// Isso fecha os pipes e desbloqueia cmd.Wait() mesmo com filhos orfaos.
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	// Fallback: forca drenagem dos pipes se ainda abertos 10s apos o SIGKILL.
	cmd.WaitDelay = 10 * time.Second

	var stdoutBuf, stderrBuf bytes.Buffer
	if liveOut != nil {
		liveOut = newSynchronizedWriter(liveOut)
		// Transmite stdout E stderr ao vivo para liveOut (ex: os.Stderr do processo pai).
		// Permite ver progresso de ferramentas que escrevem em stderr (ex: avisos, status).
		// O buffer captura independentemente para inclusao no relatorio final.
		cmd.Stdout = io.MultiWriter(&stdoutBuf, liveOut)
		cmd.Stderr = io.MultiWriter(&stderrBuf, liveOut)
	} else {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	}

	if err := cmd.Start(); err != nil {
		return stdoutBuf.String(), stderrBuf.String(), -1, err
	}
	if onStart != nil {
		onStart()
	}

	err := cmd.Wait()
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
