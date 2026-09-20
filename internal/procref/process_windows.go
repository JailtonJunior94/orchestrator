//go:build windows

package procref

import (
	"os"
	"syscall"
)

const (
	processQueryLimitedInformation = 0x1000
	errnoAccessDenied              = syscall.Errno(5)
	errnoInvalidParameter          = syscall.Errno(87)
)

func checkProcessLiveness(ref ProcessRef) LivenessResult {
	if ref.PID <= 0 {
		return LivenessResult{Reliable: true, Alive: false}
	}

	hostname, err := os.Hostname()
	if err != nil || (ref.Hostname != "" && ref.Hostname != hostname) {
		return LivenessResult{Reliable: false, Alive: false}
	}

	handle, openErr := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(ref.PID))
	if openErr != nil {
		if errno, ok := openErr.(syscall.Errno); ok {
			switch errno {
			case errnoInvalidParameter:
				return LivenessResult{Reliable: true, Alive: false}
			case errnoAccessDenied:
				return LivenessResult{Reliable: true, Alive: true}
			}
		}
		return LivenessResult{Reliable: false, Alive: false}
	}
	defer syscall.CloseHandle(handle)

	var exitCode uint32
	if exitErr := syscall.GetExitCodeProcess(handle, &exitCode); exitErr != nil {
		return LivenessResult{Reliable: false, Alive: false}
	}
	if exitCode == 259 {
		return LivenessResult{Reliable: true, Alive: true}
	}
	return LivenessResult{Reliable: true, Alive: false}
}
