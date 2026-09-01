//go:build windows

package taskloop

import (
	"errors"
	"fmt"
	"os"
)

// Windows nao oferece flock no pacote syscall. O Create exclusivo mantém o
// mesmo fail-closed: uma retomada requer remover explicitamente um lock órfão
// após confirmação operacional, nunca sobrescrevê-lo silenciosamente.
func acquireOrchestratorLock(path string) (func() error, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrWriterLocked
		}
		return nil, fmt.Errorf("taskloop: obter lock do escritor: %w", err)
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
