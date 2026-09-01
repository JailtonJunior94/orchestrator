//go:build !windows

package taskloop

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func acquireOrchestratorLock(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("taskloop: abrir lock do escritor: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrWriterLocked
		}
		return nil, fmt.Errorf("taskloop: obter lock do escritor: %w", err)
	}
	return func() error {
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN); err != nil {
			_ = f.Close()
			return fmt.Errorf("taskloop: liberar lock do escritor: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("taskloop: fechar lock do escritor: %w", err)
		}
		return nil
	}, nil
}
