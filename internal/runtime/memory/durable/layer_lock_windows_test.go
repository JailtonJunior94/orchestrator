//go:build windows

package durable

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCreateLayerLockFile_PublishesContentAtomicallyWithVisibility(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "layer.lock")

	if err := createLayerLockFile(path); err != nil {
		t.Fatalf("createLayerLockFile: %v", err)
	}

	_, ref, ok := readLayerLockRaw(path)
	if !ok {
		t.Fatal("lock file must be fully readable the instant it becomes visible under its final name — no empty/incomplete window")
	}
	if ref.PID != os.Getpid() {
		t.Errorf("ref.PID = %d, want %d", ref.PID, os.Getpid())
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			t.Errorf("temp file leaked after successful createLayerLockFile: %s", e.Name())
		}
	}
}

func TestCreateLayerLockFile_SecondCallReportsAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "layer.lock")

	if err := createLayerLockFile(path); err != nil {
		t.Fatalf("first createLayerLockFile: %v", err)
	}
	err := createLayerLockFile(path)
	if err == nil {
		t.Fatal("second createLayerLockFile must fail: lock file already exists")
	}
	if !os.IsExist(err) {
		t.Errorf("second createLayerLockFile error = %v, want os.ErrExist-compatible error", err)
	}
}

func TestReadLayerLockRawWithRetry_RetriesBeforeDecidingOrphan(t *testing.T) {
	originalAttempts := layerLockReadRetryAttempts
	originalDelay := layerLockReadRetryDelay
	originalHook := readLayerLockRawHook
	t.Cleanup(func() {
		layerLockReadRetryAttempts = originalAttempts
		layerLockReadRetryDelay = originalDelay
		readLayerLockRawHook = originalHook
	})
	layerLockReadRetryAttempts = 3
	layerLockReadRetryDelay = 0

	callCount := 0
	wantRef := ProcessRef{PID: 4242, Hostname: "host", StartedAt: time.Now().UTC()}
	wantRaw := []byte("complete-lock-content")
	readLayerLockRawHook = func(string) ([]byte, ProcessRef, bool) {
		callCount++
		if callCount < 3 {
			return []byte("partial"), ProcessRef{}, false
		}
		return wantRaw, wantRef, true
	}

	gotRaw, gotRef, ok := readLayerLockRawWithRetry("irrelevant-path")

	if !ok {
		t.Fatal("readLayerLockRawWithRetry must retry and eventually observe a valid lock file instead of declaring orphan on the first incomplete read")
	}
	if callCount != 3 {
		t.Errorf("readLayerLockRawHook call count = %d, want 3 (in-progress-write window simulated for 2 reads before success)", callCount)
	}
	if gotRef != wantRef {
		t.Errorf("gotRef = %+v, want %+v", gotRef, wantRef)
	}
	if string(gotRaw) != string(wantRaw) {
		t.Errorf("gotRaw = %q, want %q", gotRaw, wantRaw)
	}
}

func TestReadLayerLockRawWithRetry_DeclaresOrphanAfterExhaustingRetries(t *testing.T) {
	originalAttempts := layerLockReadRetryAttempts
	originalDelay := layerLockReadRetryDelay
	originalHook := readLayerLockRawHook
	t.Cleanup(func() {
		layerLockReadRetryAttempts = originalAttempts
		layerLockReadRetryDelay = originalDelay
		readLayerLockRawHook = originalHook
	})
	layerLockReadRetryAttempts = 3
	layerLockReadRetryDelay = 0

	callCount := 0
	readLayerLockRawHook = func(string) ([]byte, ProcessRef, bool) {
		callCount++
		return []byte("partial"), ProcessRef{}, false
	}

	_, _, ok := readLayerLockRawWithRetry("irrelevant-path")

	if ok {
		t.Fatal("readLayerLockRawWithRetry must not fabricate a ref when the file never becomes readable")
	}
	if callCount != layerLockReadRetryAttempts {
		t.Errorf("readLayerLockRawHook call count = %d, want %d (bounded retry, no infinite loop)", callCount, layerLockReadRetryAttempts)
	}
}

func TestTakeOverStaleLayerLock_DoesNotDeleteLockRecreatedByAnotherProcessDuringDecision(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "layer.lock")

	staleRef := ProcessRef{PID: 99999, Hostname: "dead-host", StartedAt: time.Now().Add(-time.Hour).UTC()}
	staleContent := fmt.Appendf(nil, "%d\n%s\n%s\n", staleRef.PID, staleRef.Hostname, staleRef.StartedAt.Format(time.RFC3339Nano))
	if err := os.WriteFile(path, staleContent, 0o644); err != nil {
		t.Fatalf("seed stale lock file: %v", err)
	}

	if err := removeLayerLockFileIfUnchanged(path, staleContent); err != nil {
		t.Fatalf("simulate winning takeover by a concurrent process: %v", err)
	}
	if err := createLayerLockFile(path); err != nil {
		t.Fatalf("simulate concurrent process publishing its own lock: %v", err)
	}
	recreated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read recreated lock: %v", err)
	}

	err = removeLayerLockFileIfUnchanged(path, staleContent)
	if !errors.Is(err, ErrLayerLocked) {
		t.Fatalf("removeLayerLockFileIfUnchanged against a stale snapshot must refuse with ErrLayerLocked instead of deleting a lock recreated in the meantime, got: %v", err)
	}
	survivingContent, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("lock recreated by the concurrent process must survive the losing takeover attempt: %v", readErr)
	}
	if string(survivingContent) != string(recreated) {
		t.Errorf("surviving lock content = %q, want unchanged %q", survivingContent, recreated)
	}
}
