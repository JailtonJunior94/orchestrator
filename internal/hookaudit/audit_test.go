package hookaudit

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAuditLogRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hook-decisions.jsonl")

	writer, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter unexpected error: %v", err)
	}

	first, err := NewEntry(time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC), "before_tool", "provider-x", "git-policy", "BLOCK", 12)
	if err != nil {
		t.Fatalf("NewEntry unexpected error: %v", err)
	}
	first = first.WithReason("git push requires explicit approval", "P-GIT-001", "G-GIT-PUSH").WithEvidence("log-line-1")

	second, err := NewEntry(time.Date(2026, 9, 18, 12, 1, 0, 0, time.UTC), "after_tool", "provider-x", "quality-gate", "ALLOW", 5)
	if err != nil {
		t.Fatalf("NewEntry unexpected error: %v", err)
	}

	if err := writer.Append(first); err != nil {
		t.Fatalf("Append first unexpected error: %v", err)
	}
	if err := writer.Append(second); err != nil {
		t.Fatalf("Append second unexpected error: %v", err)
	}

	reader := NewReader(path)
	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if entries[0].Decision != "BLOCK" || entries[0].PolicyID != "P-GIT-001" || entries[0].GateID != "G-GIT-PUSH" {
		t.Fatalf("first entry mismatch: %+v", entries[0])
	}
	if len(entries[0].Evidence) != 1 || entries[0].Evidence[0] != "log-line-1" {
		t.Fatalf("first entry evidence mismatch: %+v", entries[0])
	}
	if entries[1].Decision != "ALLOW" || entries[1].Hook != "quality-gate" {
		t.Fatalf("second entry mismatch: %+v", entries[1])
	}
}

func TestAuditLogAppend_DoesNotTruncatePriorEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hook-decisions.jsonl")

	writer, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter unexpected error: %v", err)
	}

	for i := 0; i < 5; i++ {
		entry, err := NewEntry(time.Now().UTC(), "before_tool", "provider-x", "hook", "ALLOW", int64(i))
		if err != nil {
			t.Fatalf("NewEntry unexpected error: %v", err)
		}
		if err := writer.Append(entry); err != nil {
			t.Fatalf("Append unexpected error: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile unexpected error: %v", err)
	}
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 5 {
		t.Fatalf("lines = %d, want 5", lines)
	}
}

func TestAuditLogReader_MissingFileReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	reader := NewReader(filepath.Join(dir, "does-not-exist.jsonl"))

	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll unexpected error: %v", err)
	}
	if entries != nil {
		t.Fatalf("entries = %v, want nil", entries)
	}
}

func TestAuditLogReader_CorruptLineIsReported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hook-decisions.jsonl")
	if err := os.WriteFile(path, []byte("{\"ts\":\"not-json\n"), 0o644); err != nil {
		t.Fatalf("WriteFile unexpected error: %v", err)
	}

	reader := NewReader(path)
	_, err := reader.ReadAll()
	if !errors.Is(err, ErrCorruptEntry) {
		t.Fatalf("error = %v, want ErrCorruptEntry", err)
	}
}

func TestNewEntry_RejectsEmptyRequiredFields(t *testing.T) {
	if _, err := NewEntry(time.Now(), "", "p", "hook", "ALLOW", 0); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("empty event error = %v, want ErrInvalidEntry", err)
	}
	if _, err := NewEntry(time.Now(), "before_tool", "p", "", "ALLOW", 0); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("empty hook error = %v, want ErrInvalidEntry", err)
	}
	if _, err := NewEntry(time.Now(), "before_tool", "p", "hook", "", 0); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("empty decision error = %v, want ErrInvalidEntry", err)
	}
	if _, err := NewEntry(time.Now(), "before_tool", "p", "hook", "ALLOW", -1); !errors.Is(err, ErrInvalidEntry) {
		t.Fatalf("negative duration error = %v, want ErrInvalidEntry", err)
	}
}
