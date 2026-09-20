//go:build !windows

package taskloop

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

func acquireTasksFileLock(tasksFile string) (func() error, error) {
	lockPath := tasksFile + ".lock"
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("taskloop: open tasks.md lock: %w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		flockErr := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if flockErr == nil {
			break
		}
		if !errors.Is(flockErr, syscall.EWOULDBLOCK) && !errors.Is(flockErr, syscall.EAGAIN) {
			_ = f.Close()
			return nil, fmt.Errorf("taskloop: acquire tasks.md lock: %w", flockErr)
		}
		if time.Now().After(deadline) {
			_ = f.Close()
			return nil, ErrTasksFileLockTimeout
		}
		time.Sleep(50 * time.Millisecond)
	}
	return func() error {
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
			_ = f.Close()
			return fmt.Errorf("taskloop: release tasks.md lock: %w", err)
		}
		return f.Close()
	}, nil
}
