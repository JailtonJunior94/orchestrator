package gitref

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ResolvedRef holds the result of resolving a git ref to a temporary directory.
type ResolvedRef struct {
	Dir     string
	Label   string
	Commit  string
	Cleanup func()
}

// Resolve resolves a git ref (tag, branch, or SHA) in repoPath to a temporary
// directory containing the file tree at that ref.
func Resolve(repoPath, ref string) (*ResolvedRef, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, errors.New("git not found in PATH: please install git")
	}

	toplevelCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--show-toplevel")
	if out, err := toplevelCmd.Output(); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", repoPath)
	} else {
		_ = out
	}

	commitCmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", ref+"^{commit}")
	commitOut, err := commitCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("invalid ref %q: use a name resolvable by git rev-parse (tag, branch, or SHA)", ref)
	}
	commit := strings.TrimSpace(string(commitOut))

	tmpDir, err := os.MkdirTemp("", "gitref-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}

	archiveCmd := exec.Command("git", "-C", repoPath, "archive", commit)
	tarCmd := exec.Command("tar", "-x", "-C", tmpDir)

	pipe, err := archiveCmd.StdoutPipe()
	if err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("creating archive pipe: %w", err)
	}
	tarCmd.Stdin = pipe

	if err := archiveCmd.Start(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("starting git archive: %w", err)
	}
	if err := tarCmd.Start(); err != nil {
		archiveCmd.Wait()
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("starting tar: %w", err)
	}
	if err := archiveCmd.Wait(); err != nil {
		tarCmd.Wait()
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("git archive failed: %w", err)
	}
	if err := tarCmd.Wait(); err != nil {
		os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("tar extraction failed: %w", err)
	}

	return &ResolvedRef{
		Dir:    tmpDir,
		Label:  ref,
		Commit: commit,
		Cleanup: func() {
			os.RemoveAll(tmpDir)
		},
	}, nil
}
