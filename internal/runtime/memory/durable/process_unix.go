//go:build !windows

package durable

import (
	"errors"
	"os"
	"syscall"
)

func checkProcessLiveness(ref ProcessRef) LivenessResult {
	if ref.PID <= 0 {
		return LivenessResult{Reliable: true, Alive: false}
	}

	hostname, err := os.Hostname()
	if err != nil || (ref.Hostname != "" && ref.Hostname != hostname) {
		return LivenessResult{Reliable: false, Alive: false}
	}

	killErr := syscall.Kill(ref.PID, 0)
	switch {
	case killErr == nil:
		return LivenessResult{Reliable: true, Alive: true}
	case errors.Is(killErr, syscall.EPERM):
		return LivenessResult{Reliable: true, Alive: true}
	case errors.Is(killErr, syscall.ESRCH):
		return LivenessResult{Reliable: true, Alive: false}
	default:
		return LivenessResult{Reliable: false, Alive: false}
	}
}
