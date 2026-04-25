package specdrift_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/specdrift"
)

// writeFileSync is a test helper that creates a file in dir with the given content.
func writeFileSync(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

func hashOfSync(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}

func readFileSync(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(b)
}

// TestSyncSpecHash_UpdatesExistingHashes verifies that stale hash comments are replaced.
func TestSyncSpecHash_UpdatesExistingHashes(t *testing.T) {
	dir := t.TempDir()

	prdContent := "- RF-01: feature one"
	techspecContent := "architecture details"

	tasksContent := fmt.Sprintf(
		"<!-- spec-hash-prd: 0000000000000000000000000000000000000000000000000000000000000000 -->\n"+
			"<!-- spec-hash-techspec: 0000000000000000000000000000000000000000000000000000000000000000 -->\n"+
			"# Tasks\n",
	)

	writeFileSync(t, dir, "prd.md", prdContent)
	writeFileSync(t, dir, "techspec.md", techspecContent)
	tasksPath := writeFileSync(t, dir, "tasks.md", tasksContent)

	if err := specdrift.SyncSpecHash(tasksPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readFileSync(t, tasksPath)

	expectedPRD := hashOfSync([]byte(prdContent))
	expectedTech := hashOfSync([]byte(techspecContent))

	if !strings.Contains(got, fmt.Sprintf("<!-- spec-hash-prd: %s -->", expectedPRD)) {
		t.Errorf("updated tasks.md missing correct prd hash\ngot:\n%s", got)
	}
	if !strings.Contains(got, fmt.Sprintf("<!-- spec-hash-techspec: %s -->", expectedTech)) {
		t.Errorf("updated tasks.md missing correct techspec hash\ngot:\n%s", got)
	}
	if strings.Contains(got, "0000000000000000000000000000000000000000000000000000000000000000") {
		t.Errorf("stale placeholder hash still present after sync\ngot:\n%s", got)
	}
}

// TestSyncSpecHash_InsertsWhenMissing verifies that missing hash comments are prepended.
func TestSyncSpecHash_InsertsWhenMissing(t *testing.T) {
	dir := t.TempDir()

	prdContent := "- RF-01: something"
	writeFileSync(t, dir, "prd.md", prdContent)
	tasksPath := writeFileSync(t, dir, "tasks.md", "# Tasks\n")

	if err := specdrift.SyncSpecHash(tasksPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := readFileSync(t, tasksPath)
	expectedPRD := hashOfSync([]byte(prdContent))

	if !strings.Contains(got, fmt.Sprintf("<!-- spec-hash-prd: %s -->", expectedPRD)) {
		t.Errorf("inserted prd hash not found\ngot:\n%s", got)
	}
	// comment should appear before the heading
	hashIdx := strings.Index(got, "spec-hash-prd")
	headingIdx := strings.Index(got, "# Tasks")
	if hashIdx > headingIdx {
		t.Errorf("hash comment should appear before heading, but hash at %d, heading at %d", hashIdx, headingIdx)
	}
}

// TestSyncSpecHash_SkipsMissingSpecFiles verifies that absent spec files are silently skipped.
func TestSyncSpecHash_SkipsMissingSpecFiles(t *testing.T) {
	dir := t.TempDir()
	tasksPath := writeFileSync(t, dir, "tasks.md", "# Tasks\n")
	// no prd.md, no techspec.md

	if err := specdrift.SyncSpecHash(tasksPath); err != nil {
		t.Fatalf("should not error when spec files are absent: %v", err)
	}

	got := readFileSync(t, tasksPath)
	if got != "# Tasks\n" {
		t.Errorf("tasks.md should be unchanged when no spec files exist\ngot:\n%s", got)
	}
}

// TestSyncSpecHash_NoOpWhenHashesMatch verifies that tasks.md is not rewritten when up to date.
func TestSyncSpecHash_NoOpWhenHashesMatch(t *testing.T) {
	dir := t.TempDir()

	prdContent := "- RF-01: something"
	prdHash := hashOfSync([]byte(prdContent))

	tasksContent := fmt.Sprintf("<!-- spec-hash-prd: %s -->\n# Tasks\n", prdHash)

	writeFileSync(t, dir, "prd.md", prdContent)
	tasksPath := writeFileSync(t, dir, "tasks.md", tasksContent)

	// capture mtime before sync
	infoBefore, _ := os.Stat(tasksPath)

	if err := specdrift.SyncSpecHash(tasksPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	infoAfter, _ := os.Stat(tasksPath)
	if !infoBefore.ModTime().Equal(infoAfter.ModTime()) {
		t.Errorf("tasks.md was rewritten even though hashes were already correct")
	}
}

// TestSyncSpecHash_MissingTasksFile verifies that a missing tasks.md returns an error.
func TestSyncSpecHash_MissingTasksFile(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.md")

	err := specdrift.SyncSpecHash(tasksPath)
	if err == nil {
		t.Error("expected error when tasks.md does not exist")
	}
}

// TestSyncSpecHash_RoundTrip verifies that sync + CheckDrift passes after syncing.
func TestSyncSpecHash_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	prdContent := "- RF-01: feature\n- RF-02: other"
	techspecContent := "tech details"

	writeFileSync(t, dir, "prd.md", prdContent)
	writeFileSync(t, dir, "techspec.md", techspecContent)

	// tasks.md with stale hashes but correct RF coverage
	tasksContent := fmt.Sprintf(
		"<!-- spec-hash-prd: 0000000000000000000000000000000000000000000000000000000000000000 -->\n"+
			"<!-- spec-hash-techspec: 0000000000000000000000000000000000000000000000000000000000000000 -->\n"+
			"RF-01 done. RF-02 done.\n",
	)

	tasksPath := writeFileSync(t, dir, "tasks.md", tasksContent)

	if err := specdrift.SyncSpecHash(tasksPath); err != nil {
		t.Fatalf("sync error: %v", err)
	}

	report, err := specdrift.CheckDrift(dir)
	if err != nil {
		t.Fatalf("CheckDrift error after sync: %v", err)
	}
	if !report.Pass {
		for _, c := range report.Coverage {
			if !c.Pass {
				t.Logf("coverage fail: %v", c.MissingIDs)
			}
		}
		for _, h := range report.Hashes {
			if !h.Match {
				t.Logf("hash fail: %s expected=%s actual=%s", h.File, h.ExpectedHash, h.ActualHash)
			}
		}
		t.Error("expected CheckDrift to pass after SyncSpecHash")
	}
}
