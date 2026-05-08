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
		// Transmite stdout E stderr ao vivo para liveOut (ex: os.Stderr do processo pai).
		// Permite ver progresso de ferramentas que escrevem em stderr (ex: avisos, status).
		// O buffer captura independentemente para inclusao no relatorio final.
		// syncWriter serializa as goroutines de stdout/stderr ao escrever no mesmo destino
		// (sem isto, writers nao thread-safe como bytes.Buffer racem ao receber ambos).
		shared := &syncWriter{w: liveOut}
		cmd.Stdout = io.MultiWriter(&stdoutBuf, shared)
		cmd.Stderr = io.MultiWriter(&stderrBuf, shared)
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
