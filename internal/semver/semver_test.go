package semver_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/semver"
)

// initRepo creates a temporary git repository, configures identity, and returns its path.
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")
	return dir
}

// addCommit creates a file and commits it with the given message.
func addCommit(t *testing.T, dir, msg string) {
	t.Helper()
	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte(msg), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	cmd := exec.Command("git", "-C", dir, "add", "file.txt")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", dir, "commit", "-m", msg)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit %q: %v\n%s", msg, err, out)
	}
}

// addTag creates a lightweight tag.
func addTag(t *testing.T, dir, tag string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "tag", tag)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git tag %q: %v\n%s", tag, err, out)
	}
}

// ---- Tests ----

func TestEvaluate_Bootstrap(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "chore: initial")

	d, err := semver.Evaluate(dir)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Action != "bootstrap" {
		t.Errorf("action = %q, want bootstrap", d.Action)
	}
	if !d.BootstrapRequired {
		t.Error("bootstrap_required should be true")
	}
	if d.BaseVersion == "" {
		t.Error("base_version should not be empty")
	}
}

func TestEvaluate_NoRelease(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "chore: setup")
	addTag(t, dir, "v1.0.0")
	addCommit(t, dir, "docs: update readme")
	addCommit(t, dir, "chore: bump ci")

	d, err := semver.Evaluate(dir)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Action != "no_release" {
		t.Errorf("action = %q, want no_release", d.Action)
	}
	if d.Bump != semver.BumpNone {
		t.Errorf("bump = %q, want none", d.Bump)
	}
	if d.LastTag != "v1.0.0" {
		t.Errorf("last_tag = %q, want v1.0.0", d.LastTag)
	}
}

func TestEvaluate_PatchBump(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "chore: setup")
	addTag(t, dir, "v1.2.3")
	addCommit(t, dir, "fix: correct nil pointer")

	d, err := semver.Evaluate(dir)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Action != "release" {
		t.Errorf("action = %q, want release", d.Action)
	}
	if d.Bump != semver.BumpPatch {
		t.Errorf("bump = %q, want patch", d.Bump)
	}
	if d.TargetVersion != "1.2.4" {
		t.Errorf("target_version = %q, want 1.2.4", d.TargetVersion)
	}
	if d.CommitCount != 1 {
		t.Errorf("commit_count = %d, want 1", d.CommitCount)
	}
}

func TestEvaluate_MinorBump(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "chore: setup")
	addTag(t, dir, "v1.2.0")
	addCommit(t, dir, "feat: add X")

	d, err := semver.Evaluate(dir)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Action != "release" {
		t.Errorf("action = %q, want release", d.Action)
	}
	if d.Bump != semver.BumpMinor {
		t.Errorf("bump = %q, want minor", d.Bump)
	}
	if d.TargetVersion != "1.3.0" {
		t.Errorf("target_version = %q, want 1.3.0", d.TargetVersion)
	}
	if d.LastTag != "v1.2.0" {
		t.Errorf("last_tag = %q, want v1.2.0", d.LastTag)
	}
	if d.CommitCount != 1 {
		t.Errorf("commit_count = %d, want 1", d.CommitCount)
	}
}

func TestEvaluate_MajorBump_ExclamationMark(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "chore: setup")
	addTag(t, dir, "v1.0.0")
	addCommit(t, dir, "feat!: remove legacy API")

	d, err := semver.Evaluate(dir)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Bump != semver.BumpMajor {
		t.Errorf("bump = %q, want major", d.Bump)
	}
	if d.TargetVersion != "2.0.0" {
		t.Errorf("target_version = %q, want 2.0.0", d.TargetVersion)
	}
}

func TestEvaluate_MajorBump_BreakingChangeFooter(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "chore: setup")
	addTag(t, dir, "v1.0.0")

	// Commit with BREAKING CHANGE in body
	f := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(f, []byte("breaking"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cmd := exec.Command("git", "-C", dir, "add", "file.txt")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-C", dir, "-c", "commit.gpgsign=false", "commit", "-m", "feat: add new thing\n\nBREAKING CHANGE: removes old API")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	d, err := semver.Evaluate(dir)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Bump != semver.BumpMajor {
		t.Errorf("bump = %q, want major", d.Bump)
	}
}

func TestEvaluate_HighestBumpWins(t *testing.T) {
	dir := initRepo(t)
	addCommit(t, dir, "chore: setup")
	addTag(t, dir, "v2.0.0")
	addCommit(t, dir, "fix: typo")
	addCommit(t, dir, "feat: add Y")
	addCommit(t, dir, "docs: update")

	d, err := semver.Evaluate(dir)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if d.Bump != semver.BumpMinor {
		t.Errorf("bump = %q, want minor (feat beats fix)", d.Bump)
	}
	if d.TargetVersion != "2.1.0" {
		t.Errorf("target_version = %q, want 2.1.0", d.TargetVersion)
	}
	if d.CommitCount != 3 {
		t.Errorf("commit_count = %d, want 3", d.CommitCount)
	}
}

func TestDetermineBump(t *testing.T) {
	cases := []struct {
		commits []semver.Commit
		want    semver.BumpKind
	}{
		{nil, semver.BumpNone},
		{[]semver.Commit{{Type: "docs"}}, semver.BumpNone},
		{[]semver.Commit{{Type: "fix"}}, semver.BumpPatch},
		{[]semver.Commit{{Type: "perf"}}, semver.BumpPatch},
		{[]semver.Commit{{Type: "refactor"}}, semver.BumpPatch},
		{[]semver.Commit{{Type: "feat"}}, semver.BumpMinor},
		{[]semver.Commit{{Type: "feat", Breaking: true}}, semver.BumpMajor},
		{[]semver.Commit{{Breaking: true}}, semver.BumpMajor},
	}
	for _, tc := range cases {
		got := semver.DetermineBump(tc.commits)
		if got != tc.want {
			t.Errorf("DetermineBump(%v) = %q, want %q", tc.commits, got, tc.want)
		}
	}
}

func TestComputeNext(t *testing.T) {
	cases := []struct {
		current string
		bump    semver.BumpKind
		want    string
	}{
		{"v1.2.3", semver.BumpPatch, "1.2.4"},
		{"v1.2.3", semver.BumpMinor, "1.3.0"},
		{"v1.2.3", semver.BumpMajor, "2.0.0"},
		{"1.2.3", semver.BumpPatch, "1.2.4"},
		{"v1.2.0", semver.BumpMinor, "1.3.0"},
	}
	for _, tc := range cases {
		got := semver.ComputeNext(tc.current, tc.bump)
		if got != tc.want {
			t.Errorf("ComputeNext(%q, %q) = %q, want %q", tc.current, tc.bump, got, tc.want)
		}
	}
}
