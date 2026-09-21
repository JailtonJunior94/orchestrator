package qualitygate

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_CONFIG_COUNT=1", "GIT_CONFIG_KEY_0=commit.gpgsign", "GIT_CONFIG_VALUE_0=false",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestComputeStateDigest_ChangesWhenWorkingTreeChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available in PATH")
	}

	dir := t.TempDir()
	runGitCommand(t, dir, "init")
	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGitCommand(t, dir, "add", "file.txt")
	runGitCommand(t, dir, "commit", "-m", "initial")

	digestBefore := ComputeStateDigest(dir)
	if digestBefore == "" {
		t.Fatalf("ComputeStateDigest() = empty, want non-empty for a git repo")
	}

	digestUnchanged := ComputeStateDigest(dir)
	if digestUnchanged != digestBefore {
		t.Fatalf("ComputeStateDigest() changed with no working tree modification: %q != %q", digestUnchanged, digestBefore)
	}

	if err := os.WriteFile(filePath, []byte("v2\n"), 0o644); err != nil {
		t.Fatalf("modify file: %v", err)
	}

	digestAfter := ComputeStateDigest(dir)
	if digestAfter == digestBefore {
		t.Fatalf("ComputeStateDigest() did not change after a real working tree modification")
	}
}

func TestComputeStateDigest_NonGitDirReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	if digest := ComputeStateDigest(dir); digest != "" {
		t.Fatalf("ComputeStateDigest() = %q, want empty for non-git dir", digest)
	}
}
