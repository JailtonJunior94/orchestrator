//go:build windows

package taskloop

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/procref"
)

func acquireTasksFileLock(tasksFile string) (func() error, error) {
	lockPath := tasksFile + ".lock"
	deadline := time.Now().Add(30 * time.Second)
	for {
		f, err := tryCreateTasksFileLock(lockPath)
		if err == nil {
			return func() error {
				if closeErr := f.Close(); closeErr != nil {
					return fmt.Errorf("taskloop: close tasks.md lock: %w", closeErr)
				}
				return os.Remove(lockPath)
			}, nil
		}
		if !errors.Is(err, ErrWriterLocked) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, ErrTasksFileLockTimeout
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func tryCreateTasksFileLock(lockPath string) (*os.File, error) {
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err == nil {
		if _, writeErr := f.Write(currentOrchestratorLockIdentity().marshal()); writeErr != nil {
			_ = f.Close()
			return nil, fmt.Errorf("taskloop: gravar identidade do lock de tasks.md: %w", writeErr)
		}
		if syncErr := f.Sync(); syncErr != nil {
			_ = f.Close()
			return nil, fmt.Errorf("taskloop: sincronizar lock de tasks.md: %w", syncErr)
		}
		return f, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("taskloop: acquire tasks.md lock: %w", err)
	}
	breakOrphanedTasksFileLock(lockPath)
	f, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			owner, _ := os.ReadFile(lockPath)
			return nil, fmt.Errorf("%w: %s", ErrWriterLocked, describeOrchestratorLockOwner(owner))
		}
		return nil, fmt.Errorf("taskloop: acquire tasks.md lock: %w", err)
	}
	if _, writeErr := f.Write(currentOrchestratorLockIdentity().marshal()); writeErr != nil {
		_ = f.Close()
		return nil, fmt.Errorf("taskloop: gravar identidade do lock de tasks.md: %w", writeErr)
	}
	return f, nil
}

func breakOrphanedTasksFileLock(lockPath string) {
	owner, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		return
	}
	id, ok := parseOrchestratorLockIdentity(owner)
	if !ok {
		return
	}
	result := procref.DefaultLivenessProbe.Probe(procref.ProcessRef{PID: id.PID, Hostname: id.Hostname})
	if !result.Reliable || result.Alive {
		return
	}
	_ = os.Remove(lockPath)
}
