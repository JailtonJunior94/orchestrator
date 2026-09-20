package taskloop

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestWithTasksFileLock_SkipsRealLockForFakeFileSystem(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	calls := 0
	err := withTasksFileLock(fsys, "/fake/tasks.md", func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("withTasksFileLock: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected fn called once, got %d", calls)
	}
	if _, err := os.Stat("/fake/tasks.md.lock"); err == nil {
		t.Fatal("withTasksFileLock must not create a real lock file for FakeFileSystem")
	}
}

func TestWithTasksFileLock_SerializesConcurrentWritersOnRealFilesystem(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.md")
	if err := os.WriteFile(tasksPath, []byte("start\n"), 0o644); err != nil {
		t.Fatalf("seed tasks.md: %v", err)
	}

	fsys := fs.NewOSFileSystem()
	const writers = 20
	var wg sync.WaitGroup
	wg.Add(writers)
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		go func(i int) {
			defer wg.Done()
			err := withTasksFileLock(fsys, tasksPath, func() error {
				content, readErr := fsys.ReadFile(tasksPath)
				if readErr != nil {
					return readErr
				}
				updated := append(content, []byte(fmt.Sprintf("writer-%d\n", i))...)
				return fsys.WriteFileAtomic(tasksPath, updated)
			})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent withTasksFileLock: %v", err)
		}
	}

	final, err := os.ReadFile(tasksPath)
	if err != nil {
		t.Fatalf("read final tasks.md: %v", err)
	}
	lineCount := 0
	for _, b := range final {
		if b == '\n' {
			lineCount++
		}
	}
	if lineCount != writers+1 {
		t.Fatalf("expected %d lines (lost update under concurrency), got %d:\n%s", writers+1, lineCount, final)
	}
}
