package semver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// BumpKind represents the type of version bump.
type BumpKind string

const (
	BumpMajor BumpKind = "major"
	BumpMinor BumpKind = "minor"
	BumpPatch BumpKind = "patch"
	BumpNone  BumpKind = "none"
)

// Commit represents a parsed conventional commit.
type Commit struct {
	Hash     string
	Type     string
	Breaking bool
	Subject  string
	Raw      string
}

// Decision holds all output fields for the semver-next command.
type Decision struct {
	Action            string   `json:"action"`
	BootstrapRequired bool     `json:"bootstrap_required"`
	ReleaseRequired   bool     `json:"release_required"`
	LastTag           string   `json:"last_tag"`
	BaseVersion       string   `json:"base_version"`
	Bump              BumpKind `json:"bump"`
	TargetVersion     string   `json:"target_version"`
	CommitRange       string   `json:"commit_range"`
	CommitCount       int      `json:"commit_count"`
}

// FindLastTag finds the most recent reachable v* tag.
func FindLastTag(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "describe", "--tags", "--abbrev=0", "--match", "v*")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseConventionalCommits parses commits in the given range.
func ParseConventionalCommits(repoPath, commitRange string) ([]Commit, error) {
	cmd := exec.Command("git", "-C", repoPath, "log", "--format=%H %s", commitRange)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		c := parseCommit(parts[0], parts[1])
		commits = append(commits, c)
	}

	// Also check commit bodies for BREAKING CHANGE footer.
	for i, c := range commits {
		body, err := getCommitBody(repoPath, c.Hash)
		if err == nil && strings.Contains(body, "BREAKING CHANGE") {
			commits[i].Breaking = true
		}
	}

	return commits, nil
}

func getCommitBody(repoPath, hash string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "show", "--format=%b", "-s", hash)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func parseCommit(hash, subject string) Commit {
	c := Commit{Hash: hash, Raw: subject}

	if strings.Contains(subject, "BREAKING CHANGE") {
		c.Breaking = true
	}

	idx := strings.Index(subject, ":")
	if idx < 0 {
		return c
	}

	prefix := subject[:idx]
	c.Subject = strings.TrimSpace(subject[idx+1:])

	if strings.HasSuffix(prefix, "!") {
		c.Breaking = true
		prefix = prefix[:len(prefix)-1]
	}

	// Strip optional scope: feat(scope) → feat
	if parenIdx := strings.Index(prefix, "("); parenIdx >= 0 {
		prefix = prefix[:parenIdx]
	}

	c.Type = strings.TrimSpace(prefix)
	return c
}

// DetermineBump returns the highest bump kind from a set of commits.
func DetermineBump(commits []Commit) BumpKind {
	best := BumpNone
	for _, c := range commits {
		b := bumpForCommit(c)
		if bumpRank(b) > bumpRank(best) {
			best = b
		}
	}
	return best
}

func bumpForCommit(c Commit) BumpKind {
	if c.Breaking {
		return BumpMajor
	}
	switch c.Type {
	case "feat":
		return BumpMinor
	case "fix", "perf", "refactor":
		return BumpPatch
	}
	return BumpNone
}

func bumpRank(b BumpKind) int {
	switch b {
	case BumpMajor:
		return 3
	case BumpMinor:
		return 2
	case BumpPatch:
		return 1
	}
	return 0
}

// ComputeNext increments the semver string based on bump kind.
// current may be "v1.2.3" or "1.2.3"; returns bare "1.2.3" without prefix.
func ComputeNext(current string, bump BumpKind) string {
	v := strings.TrimPrefix(current, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return v
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	switch bump {
	case BumpMajor:
		major++
		minor = 0
		patch = 0
	case BumpMinor:
		minor++
		patch = 0
	case BumpPatch:
		patch++
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

func readVersionFile(repoPath string) string {
	data, err := os.ReadFile(filepath.Join(repoPath, "VERSION"))
	if err != nil {
		return "0.0.0"
	}
	v := strings.TrimSpace(string(data))
	if v == "" {
		return "0.0.0"
	}
	return strings.TrimPrefix(v, "v")
}

// Evaluate orchestrates all semver logic and returns a Decision.
func Evaluate(repoPath string) (*Decision, error) {
	d := &Decision{}

	lastTag, err := FindLastTag(repoPath)
	if err != nil || lastTag == "" {
		d.Action = "bootstrap"
		d.BootstrapRequired = true
		d.BaseVersion = readVersionFile(repoPath)
		d.Bump = BumpNone
		d.TargetVersion = d.BaseVersion
		d.CommitRange = "HEAD"
		return d, nil
	}

	d.LastTag = lastTag
	d.BaseVersion = strings.TrimPrefix(lastTag, "v")
	d.CommitRange = lastTag + "..HEAD"

	commits, err := ParseConventionalCommits(repoPath, d.CommitRange)
	if err != nil {
		commits = []Commit{}
	}
	d.CommitCount = len(commits)

	bump := DetermineBump(commits)
	d.Bump = bump

	if bump == BumpNone {
		d.Action = "no_release"
		d.TargetVersion = d.BaseVersion
	} else {
		d.Action = "release"
		d.ReleaseRequired = true
		d.TargetVersion = ComputeNext(lastTag, bump)
	}

	return d, nil
}
