package taskloop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestWriteTasksTableStatus_BreaksHardLinkOnTasksFile(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "tasks.md")
	content := "| # | Título | Status | Dependências |\n|---|---|---|---|\n| 1.0 | Uma | pending | — |\n"
	if err := os.WriteFile(original, []byte(content), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	hardLinked := filepath.Join(dir, "tasks-hardlink.md")
	if err := os.Link(original, hardLinked); err != nil {
		t.Skipf("hard links unsupported on this filesystem: %v", err)
	}

	fsys := fs.NewOSFileSystem()
	if err := NewCatalog().writeTasksTableStatus(hardLinked, "1.0", "done", fsys); err != nil {
		t.Fatalf("writeTasksTableStatus: %v", err)
	}

	infoA, err := os.Stat(original)
	if err != nil {
		t.Fatalf("stat original: %v", err)
	}
	infoB, err := os.Stat(hardLinked)
	if err != nil {
		t.Fatalf("stat linked: %v", err)
	}
	if os.SameFile(infoA, infoB) {
		t.Fatalf("expected the hard link to be broken by the atomic rename, but the inode is still shared")
	}
}

func TestWriteTaskFileStatus_BreaksSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "task-1.0-real.md")
	link := filepath.Join(dir, "task-1.0.md")
	content := "# Task\n\n**Status:** pending\n"
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	fsys := fs.NewOSFileSystem()
	if err := NewCatalog().writeTaskFileStatus(link, "done", fsys); err != nil {
		t.Fatalf("writeTaskFileStatus: %v", err)
	}

	if fsys.IsSymlink(link) {
		t.Fatalf("expected the symlink to be replaced by the atomic write, but it is still a symlink")
	}
	targetContent, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(targetContent) != content {
		t.Fatalf("expected the symlink target to remain untouched, got %q", targetContent)
	}
}
