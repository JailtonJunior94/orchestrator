package fs_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

var errInjectedWriteFailure = errors.New("injected write failure")

type failingFileSystem struct {
	fs.FileSystem
	failAtWrite int
	failPath    string
	writeCount  int
}

func newFailingFileSystemAtCount(inner fs.FileSystem, failAtWrite int) *failingFileSystem {
	return &failingFileSystem{FileSystem: inner, failAtWrite: failAtWrite}
}

func newFailingFileSystemAtPath(inner fs.FileSystem, failPath string) *failingFileSystem {
	return &failingFileSystem{FileSystem: inner, failPath: failPath}
}

func (f *failingFileSystem) shouldFail(path string) bool {
	if f.failPath != "" {
		return path == f.failPath
	}
	return f.failAtWrite > 0 && f.writeCount == f.failAtWrite
}

func (f *failingFileSystem) WriteFile(path string, data []byte) error {
	f.writeCount++
	if f.shouldFail(path) {
		return errInjectedWriteFailure
	}
	return f.FileSystem.WriteFile(path, data)
}

func (f *failingFileSystem) WriteFileAtomic(path string, data []byte) error {
	f.writeCount++
	if f.shouldFail(path) {
		return errInjectedWriteFailure
	}
	return f.FileSystem.WriteFileAtomic(path, data)
}

func TestFailingFileSystem_PropagatesErrorMidSequence(t *testing.T) {
	inner := fs.NewFakeFileSystem()
	decorated := newFailingFileSystemAtCount(inner, 2)

	paths := []string{"/a.txt", "/b.txt", "/c.txt"}
	var firstErr error
	successCount := 0
	for _, p := range paths {
		if err := decorated.WriteFile(p, []byte("x")); err != nil {
			firstErr = err
			break
		}
		successCount++
	}

	if firstErr == nil {
		t.Fatal("expected the injected failure at the second write of the sequence to be observed")
	}
	if !errors.Is(firstErr, errInjectedWriteFailure) {
		t.Fatalf("expected errInjectedWriteFailure, got %v", firstErr)
	}
	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful write before the injected failure, got %d", successCount)
	}
	if !inner.Exists("/a.txt") {
		t.Error("the write before the injected failure must have reached the underlying filesystem")
	}
	if inner.Exists("/b.txt") || inner.Exists("/c.txt") {
		t.Error("writes at or after the injected failure must never reach the underlying filesystem")
	}
}

func TestFailingFileSystem_FailsAtSpecificPath(t *testing.T) {
	inner := fs.NewFakeFileSystem()
	decorated := newFailingFileSystemAtPath(inner, "/manifest.json")

	if err := decorated.WriteFile("/skill.md", []byte("x")); err != nil {
		t.Fatalf("unrelated path must not fail: %v", err)
	}
	err := decorated.WriteFile("/manifest.json", []byte("x"))
	if !errors.Is(err, errInjectedWriteFailure) {
		t.Fatalf("expected errInjectedWriteFailure writing the targeted path, got %v", err)
	}
	if inner.Exists("/manifest.json") {
		t.Error("the targeted write must never reach the underlying filesystem")
	}
}

func TestFailingFileSystem_DefaultFakeStillNeverFails(t *testing.T) {
	inner := fs.NewFakeFileSystem()
	if err := inner.WriteFile("/x.txt", []byte("x")); err != nil {
		t.Fatalf("FakeFileSystem default write behavior must remain untouched (V-32): %v", err)
	}
	if err := inner.WriteFileAtomic("/y.txt", []byte("y")); err != nil {
		t.Fatalf("FakeFileSystem default atomic write behavior must remain untouched (V-32): %v", err)
	}
}
