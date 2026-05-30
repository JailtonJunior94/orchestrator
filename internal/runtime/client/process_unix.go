//go:build !windows

package client

import (
	"os/exec"
	"syscall"
	"time"
)

// configureProcessGroup coloca o subprocesso ACP em um novo grupo de processos e instala um Cancel
// que mata o grupo inteiro (SIGKILL) quando o ctx do CommandContext é cancelado. Mesma estratégia
// de internal/taskloop/agent_unix.go: ao cancelar (watchdog/cap absoluto), o subtree do agente é
// terminado — fecha os pipes, desbloqueia a leitura bloqueante do SDK ACP e evita processos órfãos
// (ex.: codex-acp via npx spawna um neto que, sem isto, sobrevive ao kill do processo direto).
// WaitDelay limita a drenagem dos pipes após o SIGKILL para garantir que cmd.Wait() retorne.
func (c *Catalog) configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = 10 * time.Second
}

// interruptProcess envia SIGINT ao GRUPO do subprocesso (PID negativo) — mata também netos
// (ex.: codex-acp via npx), evitando órfãos no teardown explícito (Close/conclusão natural).
func (c *Catalog) interruptProcess(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
}

// killProcessHard envia SIGKILL ao GRUPO do subprocesso (fallback após o período de graça).
func (c *Catalog) killProcessHard(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
