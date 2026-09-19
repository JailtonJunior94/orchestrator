package batchreport_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/batchreport"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/tracking"
	"github.com/JailtonJunior94/ai-spec-harness/internal/txn"
)

func TestBuild_ConflictOutranksEveryOtherCategory(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Files["/project/managed.md"] = []byte("tampered")

	tracker := tracking.NewTransactional(ffs, "/project", ".ai_spec_harness.json", map[string]string{
		"managed.md": "expected-hash-that-never-matches",
	})
	tracker.AllowOverwrite()

	if err := tracker.WriteFile("/project/managed.md", []byte("new content")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tracker.MarkMerged("/project/managed.md"); err != nil {
		t.Fatalf("MarkMerged: %v", err)
	}
	if err := tracker.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	report := batchreport.Build(tracker)
	if report.Outcomes["managed.md"] != batchreport.OutcomeConflict {
		t.Fatalf("outcome = %v, want conflict — a file that is both merged AND conflicting must surface as conflict, the highest-precedence category", report.Outcomes["managed.md"])
	}
}

func TestBuild_MergedOutranksCreated(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	tracker := tracking.New(ffs, "/project", ".ai_spec_harness.json")
	if err := tracker.WriteFile("/project/settings.json", []byte("{}")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tracker.MarkMerged("/project/settings.json"); err != nil {
		t.Fatalf("MarkMerged: %v", err)
	}
	if err := tracker.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	report := batchreport.Build(tracker)
	if report.Outcomes["settings.json"] != batchreport.OutcomeMerged {
		t.Fatalf("outcome = %v, want merged — merged must outrank created per the explicit precedence rule", report.Outcomes["settings.json"])
	}
}

func TestBuild_CountsByOutcomeMatchesFiveCategories(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Files["/project/preserved.md"] = []byte("same")

	tracker := tracking.New(ffs, "/project", ".ai_spec_harness.json")
	if err := tracker.WriteFile("/project/created.md", []byte("brand new")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tracker.WriteFile("/project/preserved.md", []byte("same")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tracker.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	report := batchreport.Build(tracker)
	counts := report.CountsByOutcome()
	if counts[batchreport.OutcomeCreated] != 1 {
		t.Errorf("created count = %d, want 1", counts[batchreport.OutcomeCreated])
	}
	if counts[batchreport.OutcomePreserved] != 1 {
		t.Errorf("preserved count = %d, want 1", counts[batchreport.OutcomePreserved])
	}
	if counts[batchreport.OutcomeConflict] != 0 {
		t.Errorf("conflict count = %d, want 0", counts[batchreport.OutcomeConflict])
	}
}

func TestConflictError_NamesEachFileAndCanonicalOrigin(t *testing.T) {
	err := batchreport.ConflictError([]txn.Conflict{
		{Path: "AGENTS.md", Expected: "aaa", Actual: "bbb"},
	}, "/source/of/truth")

	if err == nil {
		t.Fatal("ConflictError must never return nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "AGENTS.md") {
		t.Errorf("error must name the conflicting file: %s", msg)
	}
	if !strings.Contains(msg, "/source/of/truth") {
		t.Errorf("error must point to the canonical origin: %s", msg)
	}
}

func TestPrint_ReportsCountsAndNamesConflicts(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Files["/project/managed.md"] = []byte("tampered")

	tracker := tracking.NewTransactional(ffs, "/project", ".ai_spec_harness.json", map[string]string{
		"managed.md": "expected-hash-that-never-matches",
	})
	tracker.AllowOverwrite()
	if err := tracker.WriteFile("/project/managed.md", []byte("new content")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := tracker.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	out := &bytes.Buffer{}
	printer := output.New(false)
	printer.Out = out
	printer.Err = out

	batchreport.Print(printer, "install", batchreport.Build(tracker))

	rendered := out.String()
	if !strings.Contains(rendered, "conflict=1") {
		t.Errorf("report should count 1 conflict: %s", rendered)
	}
	if !strings.Contains(rendered, "managed.md") {
		t.Errorf("report should name the conflicting file: %s", rendered)
	}
}
