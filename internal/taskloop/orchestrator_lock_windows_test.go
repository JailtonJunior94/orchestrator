//go:build windows

package taskloop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireOrchestratorLock_BreaksOrphanedLockFromDeadProcess(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".sdd-orchestrate.lock")

	staleID := orchestratorLockIdentity{PID: 999999, Hostname: "this-host-does-not-matter", CLI: "ai-spec"}
	hostname, _ := os.Hostname()
	staleID.Hostname = hostname
	if err := os.WriteFile(lockPath, staleID.marshal(), 0o600); err != nil {
		t.Fatalf("seed stale lock: %v", err)
	}

	release, err := acquireOrchestratorLock(lockPath)
	if err != nil {
		t.Fatalf("acquireOrchestratorLock should break the orphaned lock and succeed: %v", err)
	}
	defer func() { _ = release() }()
}

func TestAcquireOrchestratorLock_KeepsFailClosedForAliveOwner(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".sdd-orchestrate.lock")

	aliveID := currentOrchestratorLockIdentity()
	if err := os.WriteFile(lockPath, aliveID.marshal(), 0o600); err != nil {
		t.Fatalf("seed alive lock: %v", err)
	}

	_, err := acquireOrchestratorLock(lockPath)
	if !errors.Is(err, ErrWriterLocked) {
		t.Fatalf("expected ErrWriterLocked for a lock owned by a live process, got %v", err)
	}
}
