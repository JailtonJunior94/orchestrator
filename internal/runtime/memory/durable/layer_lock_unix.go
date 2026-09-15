//go:build !windows

package durable

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func acquireLayerLock(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("durable: open layer lock: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrLayerLocked
		}
		return nil, fmt.Errorf("durable: acquire layer lock: %w", err)
	}
	return func() error {
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
			_ = f.Close()
			return fmt.Errorf("durable: release layer lock: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("durable: close layer lock: %w", err)
		}
		return nil
	}, nil
}
