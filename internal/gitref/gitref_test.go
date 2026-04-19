package gitref_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/gitref"
)

// setupRepo creates a temporary git repo with a file, a commit, and a tag.
// Returns the repo path and a cleanup function.
func setupRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "test")
	run("config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(repoDir, "hello.txt"), []byte("hello world\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "initial commit")
	run("tag", "v1.0.0")

	if err := os.WriteFile(filepath.Join(repoDir, "world.txt"), []byte("world\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "second commit")

	return repoDir
}

func TestResolveTag(t *testing.T) {
	repo := setupRepo(t)

	ref, err := gitref.Resolve(repo, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer ref.Cleanup()

	if ref.Label != "v1.0.0" {
		t.Errorf("Label = %q, want %q", ref.Label, "v1.0.0")
	}
	if ref.Commit == "" {
		t.Error("Commit should not be empty")
	}
	if ref.Dir == "" {
		t.Error("Dir should not be empty")
	}

	content, err := os.ReadFile(filepath.Join(ref.Dir, "hello.txt"))
	if err != nil {
		t.Fatalf("reading hello.txt: %v", err)
	}
	if string(content) != "hello world\n" {
		t.Errorf("hello.txt content = %q, want %q", content, "hello world\n")
	}

	// world.txt should NOT be present (tagged before that commit)
	if _, err := os.Stat(filepath.Join(ref.Dir, "world.txt")); !os.IsNotExist(err) {
		t.Error("world.txt should not exist at v1.0.0")
	}
}

func TestResolveBranch(t *testing.T) {
	repo := setupRepo(t)

	ref, err := gitref.Resolve(repo, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer ref.Cleanup()

	if ref.Label != "main" {
		t.Errorf("Label = %q, want %q", ref.Label, "main")
	}

	// Both files should exist on main (HEAD)
	for _, f := range []string{"hello.txt", "world.txt"} {
		if _, err := os.Stat(filepath.Join(ref.Dir, f)); err != nil {
			t.Errorf("expected %s to exist: %v", f, err)
		}
	}
}

func TestResolveShortSHA(t *testing.T) {
	repo := setupRepo(t)

	// Get the short SHA of the first commit (tagged v1.0.0)
	out, err := exec.Command("git", "-C", repo, "rev-parse", "--short", "v1.0.0").Output()
	if err != nil {
		t.Fatalf("getting short SHA: %v", err)
	}
	shortSHA := string(out[:len(out)-1]) // trim newline

	ref, err := gitref.Resolve(repo, shortSHA)
	if err != nil {
		t.Fatalf("unexpected error resolving short SHA %q: %v", shortSHA, err)
	}
	defer ref.Cleanup()

	if ref.Commit == "" {
		t.Error("Commit should not be empty")
	}
}

func TestResolveInvalidRef(t *testing.T) {
	repo := setupRepo(t)

	_, err := gitref.Resolve(repo, "nonexistent-ref-xyz")
	if err == nil {
		t.Fatal("expected error for invalid ref, got nil")
	}
}

func TestResolveNonGitDir(t *testing.T) {
	notRepo := t.TempDir()

	_, err := gitref.Resolve(notRepo, "main")
	if err == nil {
		t.Fatal("expected error for non-git directory, got nil")
	}
}

func TestCleanupRemovesTempDir(t *testing.T) {
	repo := setupRepo(t)

	ref, err := gitref.Resolve(repo, "v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dir := ref.Dir
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("temp dir should exist before cleanup: %v", err)
	}

	ref.Cleanup()

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("temp dir should be removed after Cleanup()")
	}
}
