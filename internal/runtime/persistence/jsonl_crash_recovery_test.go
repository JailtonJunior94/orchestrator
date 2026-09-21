package persistence_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
)

func TestNewJSONLWriter_RepairsTruncatedMidLine(t *testing.T) {
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

	w2, err := persistence.NewJSONLWriter(path, fsys)
	if err != nil {
		t.Fatalf("reopening a jsonl file truncated mid-line should self-heal, got error: %v", err)
	}
	if w2 == nil {
		t.Fatal("expected a usable writer after repairing a mid-line truncation")
	}

	repaired, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(repaired): %v", err)
	}
	if err := persistence.VerifyJSONLIntegrity(repaired); err != nil {
		t.Fatalf("repaired content must be valid jsonl: %v", err)
	}
	if lineCount(repaired) != 1 {
		t.Fatalf("expected only the first pre-crash line to survive the repair, got %d lines", lineCount(repaired))
	}
	firstOriginalLine := splitLines(full)[0]
	if !bytes.Equal(splitLines(repaired)[0], firstOriginalLine) {
		t.Fatal("expected the surviving line to be byte-identical to the original pre-crash line")
	}
}

func TestNewJSONLWriter_RejectsCorruptionInTheMiddleOfTheFile(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	path := "/evidence/task-crash-2/events.jsonl"

	if err := fsys.WriteFile(path, []byte("{\"a\":1}\nnot json\n{\"c\":3}\n")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := persistence.NewJSONLWriter(path, fsys)
	if err == nil {
		t.Fatal("expected error opening a jsonl file with corruption in the middle of the file")
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

	w3, err := persistence.NewJSONLWriter(path, fsys)
	if err != nil {
		t.Fatalf("session must remain usable after a crash mid-append, only the last record should be lost: %v", err)
	}
	if w3 == nil {
		t.Fatal("expected a usable writer after recovering the pre-crash state")
	}

	repaired, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(repaired): %v", err)
	}
	if err := persistence.VerifyJSONLIntegrity(repaired); err != nil {
		t.Fatalf("repaired content must be valid jsonl: %v", err)
	}
	if !bytes.Equal(repaired, beforeCrash) {
		t.Fatal("expected the repaired content to be byte-identical to the pre-crash state, only the partial record should be dropped")
	}
	if lineCount(repaired) != 3 {
		t.Fatalf("expected exactly the 3 pre-crash records to survive the repair, got %d lines", lineCount(repaired))
	}

	if err := w3.Append(makeAgentMessageEvent(t, "cinco")); err != nil {
		t.Fatalf("session must be continuable after the repair: %v", err)
	}
}

func lineCount(content []byte) int {
	n := 0
	for _, line := range splitLines(content) {
		if len(line) > 0 {
			n++
		}
	}
	return n
}

func splitLines(content []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range content {
		if b == '\n' {
			lines = append(lines, content[start:i])
			start = i + 1
		}
	}
	if start < len(content) {
		lines = append(lines, content[start:])
	}
	return lines
}
