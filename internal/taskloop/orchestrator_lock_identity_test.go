package taskloop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAcquireOrchestratorLock_WritesIdentity(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".sdd-orchestrate.lock")

	release, err := acquireOrchestratorLock(lockPath)
	if err != nil {
		t.Fatalf("acquireOrchestratorLock: %v", err)
	}
	defer func() { _ = release() }()

	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("read lock file: %v", err)
	}
	id, ok := parseOrchestratorLockIdentity(data)
	if !ok {
		t.Fatalf("lock file does not carry a parseable identity: %s", data)
	}
	if id.PID <= 0 {
		t.Fatalf("expected a positive pid, got %d", id.PID)
	}
	if id.CLI != "ai-spec" {
		t.Fatalf("expected cli=ai-spec, got %q", id.CLI)
	}
}

func TestDescribeOrchestratorLockOwner_UnidentifiedContent(t *testing.T) {
	desc := describeOrchestratorLockOwner([]byte("not json"))
	if !strings.Contains(desc, "unknown owner") {
		t.Fatalf("expected unknown owner description, got %q", desc)
	}
}

func TestDescribeOrchestratorLockOwner_ParsesIdentity(t *testing.T) {
	id := currentOrchestratorLockIdentity()
	desc := describeOrchestratorLockOwner(id.marshal())
	if !strings.Contains(desc, "cli=ai-spec") {
		t.Fatalf("expected owner description to name the cli, got %q", desc)
	}
}
