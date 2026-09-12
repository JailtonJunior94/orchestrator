//go:build integration

package durable_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

const (
	envOrphanHelperFlag = "AISPEC_ORPHAN_LOCK_HELPER"
	envOrphanLockPath   = "AISPEC_ORPHAN_LOCK_PATH"
	orphanHelperSelect  = "-test.run=^TestHelperProcessHoldsLayerLockUntilKilled$"
)

func TestHelperProcessHoldsLayerLockUntilKilled(t *testing.T) {
	if os.Getenv(envOrphanHelperFlag) != "1" {
		t.Skip("helper process only, invoked via subprocess reinvocation")
	}

	lockPath := os.Getenv(envOrphanLockPath)
	if lockPath == "" {
		t.Fatal("orphan lock helper: missing AISPEC_ORPHAN_LOCK_PATH")
	}

	release, err := durable.DefaultLayerLocker.Lock(lockPath)
	if err != nil {
		fmt.Printf("lock-error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = release() }()

	fmt.Println("locked")
	time.Sleep(2 * time.Minute)
}

func TestOrphanLayerLockReleasedWhenOwningProcessDies(t *testing.T) {
	tasksDir := t.TempDir()
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: tasksDir}

	activePath, err := scope.ActivePath()
	if err != nil {
		t.Fatalf("resolve PRD scope active path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(activePath), 0o755); err != nil {
		t.Fatalf("pre-create memory directory: %v", err)
	}

	lockPath, err := scope.SidecarPath(".lock")
	if err != nil {
		t.Fatalf("resolve lock path: %v", err)
	}

	cmd := exec.Command(os.Args[0], orphanHelperSelect)
	cmd.Env = append(os.Environ(),
		envOrphanHelperFlag+"=1",
		envOrphanLockPath+"="+lockPath,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start lock-holder subprocess: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	holderReady := make(chan string, 1)
	go func() {
		if scanner.Scan() {
			holderReady <- scanner.Text()
			return
		}
		holderReady <- ""
	}()

	select {
	case line := <-holderReady:
		if strings.TrimSpace(line) != "locked" {
			_ = cmd.Process.Kill()
			t.Fatalf("lock-holder subprocess did not report it holds the lock, got: %q", line)
		}
	case <-time.After(10 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("lock-holder subprocess never reported it holds the lock")
	}

	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("kill lock-holder subprocess without releasing the lock: %v", err)
	}
	_ = cmd.Wait()

	fsys := fs.NewOSFileSystem()
	layer := durable.NewLayer(fsys)
	catalog := durable.NewCatalog()

	key, err := catalog.DeriveSemanticKey(durable.StructuredSignal{Kind: "orphan-lock", Subject: "after-dead-owner"})
	if err != nil {
		t.Fatalf("derive semantic key: %v", err)
	}
	content := "durable fact written after the previous layer lock owner died without releasing"
	fact := durable.Fact{
		Identity:   durable.Identity{Key: key, Hash: catalog.HashContent(content)},
		Content:    content,
		Durability: durable.DurabilityPRD,
		Origin: durable.FactOrigin{
			Session: "successor",
			CLI:     "integration-test",
			Date:    time.Now().UTC().Format(time.RFC3339),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deadline := time.Now().Add(5 * time.Second)
	var consolidateErr error
	for time.Now().Before(deadline) {
		_, consolidateErr = layer.Consolidate(ctx, scope, []durable.Fact{fact})
		if consolidateErr == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if consolidateErr != nil {
		t.Fatalf("layer lock was never released after the owning process died (RF-16): %v", consolidateErr)
	}

	facts, _, err := layer.Read(context.Background(), scope)
	if err != nil {
		t.Fatalf("read consolidated layer page: %v", err)
	}

	found := false
	for _, f := range facts {
		if f.Identity.Key == key {
			found = true
			if f.Content != content {
				t.Errorf("fact content = %q, want %q (must not be silently overwritten)", f.Content, content)
			}
		}
	}
	if !found {
		t.Fatalf("fact written by the successor after taking over the layer was not persisted, got: %+v", facts)
	}
}
