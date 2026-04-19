package git_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/git"
)

// These tests rely on the project itself being a real git repository.
// They are integration tests that exercise the CLIRepository against the OS git binary.

func TestCLIRepository_IsRepo_true(t *testing.T) {
	// The orchestrator project root is a git repo.
	root := projectRoot(t)
	r := git.NewCLIRepository()
	if !r.IsRepo(root) {
		t.Errorf("IsRepo(%q) = false, expected true for a known git repo", root)
	}
}

func TestCLIRepository_IsRepo_false(t *testing.T) {
	tmp := t.TempDir() // a plain temp dir is never a git repo
	r := git.NewCLIRepository()
	if r.IsRepo(tmp) {
		t.Errorf("IsRepo(%q) = true, expected false for plain temp dir", tmp)
	}
}

func TestCLIRepository_Root(t *testing.T) {
	root := projectRoot(t)
	r := git.NewCLIRepository()
	got, err := r.Root(root)
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	if got == "" {
		t.Error("Root() should return a non-empty path")
	}
	// The returned root should be an existing directory.
	if _, err := os.Stat(got); err != nil {
		t.Errorf("Root() = %q is not a valid directory: %v", got, err)
	}
}

func TestCLIRepository_Root_notRepo(t *testing.T) {
	tmp := t.TempDir()
	r := git.NewCLIRepository()
	_, err := r.Root(tmp)
	if err == nil {
		t.Error("Root() should return error for non-git directory")
	}
}

// projectRoot walks up from the test file's directory until it finds a .git dir.
func projectRoot(t *testing.T) string {
	t.Helper()
	// Start from the current working directory (go test sets it to the package dir).
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find .git root")
		}
		dir = parent
	}
}
