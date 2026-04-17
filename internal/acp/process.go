package acp

import "os/exec"

// waitAsync starts a goroutine that calls cmd.Wait and closes the returned
// channel when the process exits. It is safe to call even if Wait was already
// called elsewhere; the extra call will simply return an error that is ignored.
func waitAsync(cmd *exec.Cmd) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	return done
}
