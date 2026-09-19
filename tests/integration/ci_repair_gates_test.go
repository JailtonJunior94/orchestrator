//go:build integration

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	ciRepairBaseOnce sync.Once
	ciRepairBaseDir  string
	ciRepairBaseErr  error
)

func ciRepairBaseSnapshot(t *testing.T) string {
	t.Helper()
	ciRepairBaseOnce.Do(func() {
		source := repoRootForDispatch(t)
		entries, err := os.ReadDir(source)
		if err != nil {
			ciRepairBaseErr = err
			return
		}
		base, err := os.MkdirTemp("", "ci-repair-base-")
		if err != nil {
			ciRepairBaseErr = err
			return
		}
		for _, entry := range entries {
			if entry.Name() == ".git" {
				continue
			}
			if cpErr := copyTree(filepath.Join(source, entry.Name()), base+string(filepath.Separator), false); cpErr != nil {
				ciRepairBaseErr = cpErr
				return
			}
		}
		ciRepairBaseDir = base
	})
	if ciRepairBaseErr != nil {
		t.Fatalf("build ci repair base snapshot: %v", ciRepairBaseErr)
	}
	return ciRepairBaseDir
}

func cloneFullRepoForCIRepairGate(t *testing.T) string {
	t.Helper()
	base := ciRepairBaseSnapshot(t)
	clone := t.TempDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatalf("read ci repair base: %v", err)
	}
	for _, entry := range entries {
		if err := copyTree(filepath.Join(base, entry.Name()), clone+string(filepath.Separator), true); err != nil {
			t.Fatalf("clone %s: %v", entry.Name(), err)
		}
	}
	return clone
}

func runGoInClone(t *testing.T, clone string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = clone
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run go %v: %v (%s)", args, err, out)
		}
		exitCode = ee.ExitCode()
	}
	return string(out), exitCode
}

func TestCIWorkflowRegistersRepairedGates(t *testing.T) {
	t.Parallel()

	repo := repoRootForDispatch(t)
	workflow, err := os.ReadFile(filepath.Join(repo, ".github", "workflows", "test.yml"))
	if err != nil {
		t.Fatalf("read test.yml: %v", err)
	}
	content := string(workflow)

	required := []string{
		"go test -v -tags=integration ./internal/integration/... ./internal/skills/... ./tests/integration/... ./internal/parity/...",
		"go test -v ./internal/contextgen/... -run TestContextgen_Snapshots",
		"make check-skills-lock",
	}
	for _, want := range required {
		if !strings.Contains(content, want) {
			t.Errorf("test.yml must contain the exact command %q (RF-20.1/RF-59/RF-60); a repaired gate not registered in the pipeline protects nothing", want)
		}
	}

	stale := "-run TestSnapshot "
	if strings.Contains(content, stale) {
		t.Errorf("test.yml still contains the broken filter %q (V-33): it matches no test name and passes with zero tests executed", stale)
	}
}

func TestSnapshotGateFailsOnDeliberateDrift(t *testing.T) {
	t.Parallel()
	clone := cloneFullRepoForCIRepairGate(t)

	snapshot := filepath.Join("testdata", "snapshots", "go-monolith.agents.md")
	original, err := os.ReadFile(filepath.Join(clone, snapshot))
	if err != nil {
		t.Fatalf("read %s: %v", snapshot, err)
	}
	replaceInClone(t, clone, snapshot, append(append([]byte(nil), original...), []byte("\n<!-- drift -->\n")...))

	out, exitCode := runGoInClone(t, clone, "test", "-v", "./internal/contextgen/...", "-run", "TestContextgen_Snapshots")
	if exitCode == 0 {
		t.Fatalf("snapshot gate stayed green after deliberate drift in %s; output=%s", snapshot, out)
	}
	if !strings.Contains(out, "TestContextgen_Snapshots") {
		t.Fatalf("test name must appear nominally in the log even on failure; output=%s", out)
	}
}

func TestSnapshotGateIsGreenOnUntouchedClone(t *testing.T) {
	t.Parallel()
	clone := cloneFullRepoForCIRepairGate(t)

	out, exitCode := runGoInClone(t, clone, "test", "-v", "./internal/contextgen/...", "-run", "TestContextgen_Snapshots")
	if exitCode != 0 {
		t.Fatalf("snapshot gate must be green on an untouched clone; exit=%d output=%s", exitCode, out)
	}
	if !strings.Contains(out, "TestContextgen_Snapshots/go-monolith") {
		t.Fatalf("subtest names must appear nominally in the log; output=%s", out)
	}
}

func TestSkillsLockGateFailsWhenLockedSkillDrifts(t *testing.T) {
	t.Parallel()
	clone := cloneFullRepoForCIRepairGate(t)

	const trackedSkill = "semantic-commit"
	skillMD := filepath.Join(".agents", "skills", trackedSkill, "SKILL.md")
	original, err := os.ReadFile(filepath.Join(clone, skillMD))
	if err != nil {
		t.Fatalf("read %s: %v", skillMD, err)
	}
	replaceInClone(t, clone, skillMD, append(append([]byte(nil), original...), []byte("\n<!-- drift -->\n")...))

	out, exitCode := runGoInClone(t, clone, "run", ".", "skills", "--verify", ".")
	if exitCode == 0 {
		t.Fatalf("skills lock gate stayed green after editing %s without updating skills-lock.json; output=%s", skillMD, out)
	}
	if !strings.Contains(out, trackedSkill) {
		t.Fatalf("failure must name the drifted skill; output=%s", out)
	}
}

func TestSkillsLockGateIsGreenOnUntouchedClone(t *testing.T) {
	t.Parallel()
	clone := cloneFullRepoForCIRepairGate(t)

	out, exitCode := runGoInClone(t, clone, "run", ".", "skills", "--verify", ".")
	if exitCode != 0 {
		t.Fatalf("skills lock gate must be green on an untouched clone; exit=%d output=%s", exitCode, out)
	}
}
