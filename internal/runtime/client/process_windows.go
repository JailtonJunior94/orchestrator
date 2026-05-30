//go:build windows

package client

import (
	"os"
	"os/exec"
	"time"
)

// configureProcessGroup (Windows): sem grupos de processos POSIX. Apenas instala WaitDelay para
// limitar a drenagem dos pipes; o kill do CommandContext recai sobre o processo direto.
func (c *Catalog) configureProcessGroup(cmd *exec.Cmd) {
	cmd.WaitDelay = 10 * time.Second
}

// interruptProcess (Windows): sinaliza o processo direto (sem grupos POSIX).
func (c *Catalog) interruptProcess(cmd *exec.Cmd) error {
	return cmd.Process.Signal(os.Interrupt)
}

// killProcessHard (Windows): mata o processo direto.
func (c *Catalog) killProcessHard(cmd *exec.Cmd) error {
	return cmd.Process.Kill()
}
