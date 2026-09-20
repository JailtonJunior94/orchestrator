//go:build windows

package taskloop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireTasksFileLock_BreaksOrphanedLockFromDeadProcess(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.md")
	lockPath := tasksPath + ".lock"

	staleID := orchestratorLockIdentity{PID: 999999, CLI: "ai-spec"}
	hostname, _ := os.Hostname()
	staleID.Hostname = hostname
	if err := os.WriteFile(lockPath, staleID.marshal(), 0o600); err != nil {
		t.Fatalf("seed stale lock: %v", err)
	}

	release, err := acquireTasksFileLock(tasksPath)
	if err != nil {
		t.Fatalf("acquireTasksFileLock should break the orphaned lock and succeed: %v", err)
	}
	defer func() { _ = release() }()
}

func TestAcquireTasksFileLock_KeepsFailClosedForAliveOwner(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.md")
	lockPath := tasksPath + ".lock"

	aliveID := currentOrchestratorLockIdentity()
	if err := os.WriteFile(lockPath, aliveID.marshal(), 0o600); err != nil {
		t.Fatalf("seed alive lock: %v", err)
	}

	_, err := tryCreateTasksFileLock(lockPath)
	if !errors.Is(err, ErrWriterLocked) {
		t.Fatalf("expected ErrWriterLocked for a lock owned by a live process, got %v", err)
	}
}
