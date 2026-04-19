package specdrift_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/specdrift"
)

// --- CheckCoverage ---

func TestCheckCoverage_AllPresent(t *testing.T) {
	prd := []byte("Requirements: RF-01, RF-02, RF-03")
	tasks := []byte("This task covers RF-01, RF-02, and RF-03.")

	result := specdrift.CheckCoverage(prd, tasks)

	if !result.Pass {
		t.Errorf("expected Pass=true, got false; missing=%v", result.MissingIDs)
	}
	if len(result.MissingIDs) != 0 {
		t.Errorf("expected no missing IDs, got %v", result.MissingIDs)
	}
	if len(result.FoundIDs) != 3 {
		t.Errorf("expected 3 found IDs, got %d: %v", len(result.FoundIDs), result.FoundIDs)
	}
}

func TestCheckCoverage_MissingID(t *testing.T) {
	prd := []byte("Requirements: RF-01, RF-02")
	tasks := []byte("This task covers RF-01 only.")

	result := specdrift.CheckCoverage(prd, tasks)

	if result.Pass {
		t.Error("expected Pass=false, got true")
	}
	if len(result.MissingIDs) != 1 || result.MissingIDs[0] != "RF-02" {
		t.Errorf("expected MissingIDs=[RF-02], got %v", result.MissingIDs)
	}
}

func TestCheckCoverage_CaseInsensitive(t *testing.T) {
	prd := []byte("rf-01 and req-02 must be covered")
	tasks := []byte("RF-01 and REQ-02 are implemented here")

	result := specdrift.CheckCoverage(prd, tasks)

	if !result.Pass {
		t.Errorf("expected Pass=true (case-insensitive), missing=%v", result.MissingIDs)
	}
}

func TestCheckCoverage_REQIDs(t *testing.T) {
	prd := []byte("REQ-01 and REQ-02 are required")
	tasks := []byte("REQ-01 done. REQ-02 done.")

	result := specdrift.CheckCoverage(prd, tasks)

	if !result.Pass {
		t.Errorf("expected Pass=true, missing=%v", result.MissingIDs)
	}
}

func TestCheckCoverage_EmptySource(t *testing.T) {
	prd := []byte("No requirements here")
	tasks := []byte("Some tasks")

	result := specdrift.CheckCoverage(prd, tasks)

	if !result.Pass {
		t.Error("expected Pass=true when no IDs in source")
	}
	if len(result.FoundIDs) != 0 {
		t.Errorf("expected no found IDs, got %v", result.FoundIDs)
	}
}

func TestCheckCoverage_DuplicateIDsInSource(t *testing.T) {
	prd := []byte("RF-01 is mentioned. RF-01 is mentioned again. RF-02 too.")
	tasks := []byte("RF-01 and RF-02 covered.")

	result := specdrift.CheckCoverage(prd, tasks)

	if !result.Pass {
		t.Errorf("expected Pass=true, missing=%v", result.MissingIDs)
	}
	if len(result.FoundIDs) != 2 {
		t.Errorf("expected 2 unique found IDs, got %d: %v", len(result.FoundIDs), result.FoundIDs)
	}
}

// --- CheckHash ---

func hashOf(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}

func TestCheckHash_Match(t *testing.T) {
	spec := []byte("spec content v1")
	h := hashOf(spec)
	tasks := []byte(fmt.Sprintf("<!-- spec-hash-prd: %s -->", h))

	result := specdrift.CheckHash(spec, tasks, "prd")

	if !result.Match {
		t.Errorf("expected Match=true, got false; actual=%s expected=%s", result.ActualHash, result.ExpectedHash)
	}
	if result.NoHashFound {
		t.Error("expected NoHashFound=false")
	}
}

func TestCheckHash_Divergent(t *testing.T) {
	spec := []byte("spec content v2")
	tasks := []byte("<!-- spec-hash-prd: 0000000000000000000000000000000000000000000000000000000000000000 -->")

	result := specdrift.CheckHash(spec, tasks, "prd")

	if result.Match {
		t.Error("expected Match=false for divergent hash")
	}
	if result.NoHashFound {
		t.Error("expected NoHashFound=false")
	}
}

func TestCheckHash_NoComment(t *testing.T) {
	spec := []byte("spec content")
	tasks := []byte("No hash comment here")

	result := specdrift.CheckHash(spec, tasks, "prd")

	if !result.NoHashFound {
		t.Error("expected NoHashFound=true")
	}
	if result.Match {
		t.Error("expected Match=false when no hash comment")
	}
}

func TestCheckHash_TechspecLabel(t *testing.T) {
	spec := []byte("techspec content")
	h := hashOf(spec)
	tasks := []byte(fmt.Sprintf("<!-- spec-hash-techspec: %s -->", h))

	result := specdrift.CheckHash(spec, tasks, "techspec")

	if !result.Match {
		t.Errorf("expected Match=true for techspec label")
	}
}

func TestCheckHash_WrongLabel(t *testing.T) {
	spec := []byte("spec content")
	h := hashOf(spec)
	// Hash is for "prd" but we query "techspec"
	tasks := []byte(fmt.Sprintf("<!-- spec-hash-prd: %s -->", h))

	result := specdrift.CheckHash(spec, tasks, "techspec")

	if !result.NoHashFound {
		t.Error("expected NoHashFound=true when querying wrong label")
	}
}

// --- CheckDrift ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func TestCheckDrift_FullPass(t *testing.T) {
	dir := t.TempDir()

	prdContent := "RF-01 and RF-02 must be satisfied"
	techspecContent := "REQ-10 must be implemented"

	prdHash := hashOf([]byte(prdContent))
	techHash := hashOf([]byte(techspecContent))

	tasksContent := fmt.Sprintf(
		"RF-01 done. RF-02 done. REQ-10 done.\n<!-- spec-hash-prd: %s -->\n<!-- spec-hash-techspec: %s -->",
		prdHash, techHash,
	)

	writeFile(t, dir, "prd.md", prdContent)
	writeFile(t, dir, "techspec.md", techspecContent)
	writeFile(t, dir, "tasks.md", tasksContent)

	report, err := specdrift.CheckDrift(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Pass {
		t.Errorf("expected Pass=true, got false")
		for _, c := range report.Coverage {
			if !c.Pass {
				t.Logf("coverage fail: %s missing %v", c.SourceFile, c.MissingIDs)
			}
		}
		for _, h := range report.Hashes {
			if !h.Match {
				t.Logf("hash fail: %s expected=%s actual=%s noHash=%v", h.File, h.ExpectedHash, h.ActualHash, h.NoHashFound)
			}
		}
	}
}

func TestCheckDrift_MissingTasksFile(t *testing.T) {
	dir := t.TempDir()

	_, err := specdrift.CheckDrift(dir)
	if err == nil {
		t.Error("expected error when tasks.md is absent")
	}
}

func TestCheckDrift_OnlyPRD(t *testing.T) {
	dir := t.TempDir()

	prdContent := "RF-01 required"
	prdHash := hashOf([]byte(prdContent))
	tasksContent := fmt.Sprintf("RF-01 done.\n<!-- spec-hash-prd: %s -->", prdHash)

	writeFile(t, dir, "prd.md", prdContent)
	writeFile(t, dir, "tasks.md", tasksContent)
	// no techspec.md

	report, err := specdrift.CheckDrift(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Pass {
		t.Error("expected Pass=true")
	}
	if len(report.Coverage) != 1 {
		t.Errorf("expected 1 coverage entry, got %d", len(report.Coverage))
	}
}

func TestCheckDrift_CoverageFail(t *testing.T) {
	dir := t.TempDir()

	prdContent := "RF-01 and RF-02 required"
	prdHash := hashOf([]byte(prdContent))
	tasksContent := fmt.Sprintf("RF-01 done.\n<!-- spec-hash-prd: %s -->", prdHash)

	writeFile(t, dir, "prd.md", prdContent)
	writeFile(t, dir, "tasks.md", tasksContent)

	report, err := specdrift.CheckDrift(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Pass {
		t.Error("expected Pass=false due to missing RF-02")
	}
}

func TestCheckDrift_HashFail(t *testing.T) {
	dir := t.TempDir()

	prdContent := "RF-01 required"
	tasksContent := "RF-01 done.\n<!-- spec-hash-prd: 0000000000000000000000000000000000000000000000000000000000000000 -->"

	writeFile(t, dir, "prd.md", prdContent)
	writeFile(t, dir, "tasks.md", tasksContent)

	report, err := specdrift.CheckDrift(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.Pass {
		t.Error("expected Pass=false due to hash mismatch")
	}
}
