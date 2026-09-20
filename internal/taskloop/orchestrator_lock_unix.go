//go:build !windows

package taskloop

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

func acquireOrchestratorLock(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("taskloop: abrir lock do escritor: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		owner, _ := io.ReadAll(f)
		_ = f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, fmt.Errorf("%w: %s", ErrWriterLocked, describeOrchestratorLockOwner(owner))
		}
		return nil, fmt.Errorf("taskloop: obter lock do escritor: %w", err)
	}
	if err := writeOrchestratorLockIdentity(f); err != nil {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
		return nil, err
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

func writeOrchestratorLockIdentity(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("taskloop: truncar lock do escritor: %w", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return fmt.Errorf("taskloop: posicionar lock do escritor: %w", err)
	}
	if _, err := f.Write(currentOrchestratorLockIdentity().marshal()); err != nil {
		return fmt.Errorf("taskloop: gravar identidade do lock: %w", err)
	}
	return f.Sync()
}
