//go:build windows

package taskloop

import (
	"errors"
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/procref"
)

func acquireOrchestratorLock(path string) (func() error, error) {
	f, err := tryCreateOrchestratorLockFile(path)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(currentOrchestratorLockIdentity().marshal()); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("taskloop: gravar identidade do lock: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("taskloop: sincronizar lock do escritor: %w", err)
	}
	return func() error {
		if err := f.Close(); err != nil {
			return fmt.Errorf("taskloop: fechar lock do escritor: %w", err)
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("taskloop: remover lock do escritor: %w", err)
		}
		return nil
	}, nil
}

func tryCreateOrchestratorLockFile(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err == nil {
		return f, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("taskloop: obter lock do escritor: %w", err)
	}
	if breakErr := breakOrphanedOrchestratorLock(path); breakErr != nil {
		return nil, breakErr
	}
	f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			owner, _ := os.ReadFile(path)
			return nil, fmt.Errorf("%w: %s", ErrWriterLocked, describeOrchestratorLockOwner(owner))
		}
		return nil, fmt.Errorf("taskloop: obter lock do escritor: %w", err)
	}
	return f, nil
}

func breakOrphanedOrchestratorLock(path string) error {
	owner, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil
	}
	id, ok := parseOrchestratorLockIdentity(owner)
	if !ok {
		return nil
	}
	result := procref.DefaultLivenessProbe.Probe(procref.ProcessRef{PID: id.PID, Hostname: id.Hostname})
	if !result.Reliable || result.Alive {
		return nil
	}
	if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("taskloop: remover lock orfao do escritor: %w", removeErr)
	}
	return nil
}
