package persistence_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
)

func TestWriteToolCalls_Empty(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	err := persistence.NewCatalog().WriteToolCalls("/evidence/task/tool_calls.md", nil, fsys)
	if err != nil {
		t.Fatalf("WriteToolCalls(nil): %v", err)
	}

	data, _ := fsys.ReadFile("/evidence/task/tool_calls.md")
	got := string(data)

	golden := loadGolden(t, "tool_calls_empty.md")
	if got != golden {
		t.Errorf("conteúdo diverge do golden:\ngot:\n%s\nwant:\n%s", got, golden)
	}
}

func TestWriteToolCalls_WithEvents(t *testing.T) {
	fsys := fs.NewFakeFileSystem()

	ts := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	summaries := []events.ToolCallSummary{
		{
			ID:        events.NewToolCallID("tc_001"),
			Name:      "read_file",
			StartedAt: ts,
			Updates:   1,
			Final:     true,
		},
		{
			ID:        events.NewToolCallID("tc_002"),
			Name:      "write_file",
			StartedAt: ts.Add(time.Second),
			Updates:   0,
			Final:     false,
		},
	}

	err := persistence.NewCatalog().WriteToolCalls("/evidence/task/tool_calls.md", summaries, fsys)
	if err != nil {
		t.Fatalf("WriteToolCalls: %v", err)
	}

	data, _ := fsys.ReadFile("/evidence/task/tool_calls.md")
	got := string(data)

	golden := loadGolden(t, "tool_calls_with_events.md")
	if got != golden {
		t.Errorf("conteúdo diverge do golden:\ngot:\n%s\nwant:\n%s", got, golden)
	}
}

func TestWriteToolCalls_MkdirError(t *testing.T) {
	efs := &errFS{
		FakeFileSystem: fs.NewFakeFileSystem(),
		mkdirErr:       errors.New("sem espaço"),
	}
	err := persistence.NewCatalog().WriteToolCalls("/dir/tool_calls.md", nil, efs)
	if err == nil {
		t.Fatal("esperava erro de MkdirAll")
	}
}

func TestWriteToolCalls_WriteError(t *testing.T) {
	efs := &errFS{
		FakeFileSystem: fs.NewFakeFileSystem(),
		writeErr:       errors.New("disco cheio"),
	}
	err := persistence.NewCatalog().WriteToolCalls("/dir/tool_calls.md", nil, efs)
	if err == nil {
		t.Fatal("esperava erro de WriteFile")
	}
}

// loadGolden lê um arquivo golden de testdata/.
func loadGolden(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("loadGolden(%q): %v", name, err)
	}
	return string(data)
}
