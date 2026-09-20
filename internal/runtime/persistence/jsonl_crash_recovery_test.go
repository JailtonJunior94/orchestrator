package persistence_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
)

func TestNewJSONLWriter_RejectsTruncatedMidLine(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	path := "/evidence/task-crash/events.jsonl"

	w, err := persistence.NewJSONLWriter(path, fsys)
	if err != nil {
		t.Fatalf("NewJSONLWriter: %v", err)
	}
	if err := w.Append(makeAgentMessageEvent(t, "primeiro")); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if err := w.Append(makeAgentMessageEvent(t, "segundo")); err != nil {
		t.Fatalf("Append: %v", err)
	}

	full, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	truncated := full[:len(full)-5]
	if err := fsys.WriteFile(path, truncated); err != nil {
		t.Fatalf("WriteFile(truncated): %v", err)
	}

	_, err = persistence.NewJSONLWriter(path, fsys)
	if err == nil {
		t.Fatal("expected error reopening a jsonl file truncated mid-line, got nil")
	}
	if !errors.Is(err, persistence.ErrCorruptedJSONL) {
		t.Fatalf("expected ErrCorruptedJSONL, got %v", err)
	}
}

func TestNewJSONLWriter_RejectsInvalidJSONLine(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	path := "/evidence/task-crash-2/events.jsonl"

	if err := fsys.WriteFile(path, []byte("{\"a\":1}\nnot json\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := persistence.NewJSONLWriter(path, fsys)
	if err == nil {
		t.Fatal("expected error opening a jsonl file with an invalid line, got nil")
	}
	if !errors.Is(err, persistence.ErrCorruptedJSONL) {
		t.Fatalf("expected ErrCorruptedJSONL, got %v", err)
	}
}

func TestNewJSONLWriter_OnlyLastRecordIsLostOnCrash(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	path := "/evidence/task-crash-3/events.jsonl"

	w, err := persistence.NewJSONLWriter(path, fsys)
	if err != nil {
		t.Fatalf("NewJSONLWriter: %v", err)
	}
	for _, msg := range []string{"um", "dois", "tres"} {
		if err := w.Append(makeAgentMessageEvent(t, msg)); err != nil {
			t.Fatalf("Append(%q): %v", msg, err)
		}
	}

	beforeCrash, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	w2, err := persistence.NewJSONLWriter(path, fsys)
	if err != nil {
		t.Fatalf("NewJSONLWriter (reopen): %v", err)
	}
	if err := w2.Append(makeAgentMessageEvent(t, "quatro-parcial")); err != nil {
		t.Fatalf("Append: %v", err)
	}

	afterCrash, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	partial := afterCrash[:len(beforeCrash)+3]
	if err := fsys.WriteFile(path, partial); err != nil {
		t.Fatalf("WriteFile(partial): %v", err)
	}

	_, err = persistence.NewJSONLWriter(path, fsys)
	if err == nil {
		t.Fatal("expected error reopening after simulated crash mid-append")
	}
	if !errors.Is(err, persistence.ErrCorruptedJSONL) {
		t.Fatalf("expected ErrCorruptedJSONL, got %v", err)
	}

	if err := fsys.WriteFile(path, beforeCrash); err != nil {
		t.Fatalf("WriteFile(beforeCrash): %v", err)
	}
	w3, err := persistence.NewJSONLWriter(path, fsys)
	if err != nil {
		t.Fatalf("session data before the crash must remain readable: %v", err)
	}
	if w3 == nil {
		t.Fatal("expected a usable writer after recovering the pre-crash state")
	}
}
