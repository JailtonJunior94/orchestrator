package components_test

import (
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/tui/components"
)

func TestOutputView_AppendContentAccumulates(t *testing.T) {
	t.Parallel()

	ov := components.NewOutputView(80, 24)
	ov.AppendContent([]byte("line one\n"))
	ov.AppendContent([]byte("line two\n"))

	view := ov.View()
	if !strings.Contains(view, "line one") {
		t.Errorf("expected 'line one' in view, got: %s", view)
	}
	if !strings.Contains(view, "line two") {
		t.Errorf("expected 'line two' in view, got: %s", view)
	}
}

func TestOutputView_EmptyChunkIgnored(t *testing.T) {
	t.Parallel()

	ov := components.NewOutputView(80, 24)
	ov.AppendContent([]byte("hello"))
	ov.AppendContent([]byte{})

	view := ov.View()
	if !strings.Contains(view, "hello") {
		t.Errorf("expected 'hello' in view, got: %s", view)
	}
}

func TestOutputView_SetContent_Replaces(t *testing.T) {
	t.Parallel()

	ov := components.NewOutputView(80, 24)
	ov.AppendContent([]byte("old content"))
	ov.SetContent("new content")

	view := ov.View()
	if strings.Contains(view, "old content") {
		t.Errorf("old content should be gone after SetContent, got: %s", view)
	}
	if !strings.Contains(view, "new content") {
		t.Errorf("expected 'new content' in view, got: %s", view)
	}
}

func TestOutputView_SetSizeDoesNotPanic(t *testing.T) {
	t.Parallel()

	ov := components.NewOutputView(80, 24)
	ov.AppendContent([]byte("content line"))
	ov.SetSize(120, 40)
	ov.SetSize(60, 20)
	_ = ov.View()
}

func TestOutputView_ReformatsMarkdownTablesForReadableViewport(t *testing.T) {
	t.Parallel()

	ov := components.NewOutputView(48, 24)
	ov.SetContent(strings.Join([]string{
		"# Tasks",
		"",
		"| Task | Owner | Status |",
		"| --- | --- | --- |",
		"| Magic link | auth-team | In progress |",
		"| Email template | platform | Pending review |",
	}, "\n"))

	view := ov.View()
	if strings.Contains(view, "| Task | Owner | Status |") {
		t.Fatalf("expected markdown table to be reformatted, got: %s", view)
	}
	if !strings.Contains(view, "Table view:") {
		t.Fatalf("expected reformatted table heading, got: %s", view)
	}
	if !strings.Contains(view, "[1]") || !strings.Contains(view, "Task: Magic link") {
		t.Fatalf("expected first row to be rendered as stacked fields, got: %s", view)
	}
	if !strings.Contains(view, "Owner: auth-team") || !strings.Contains(view, "Status: In progress") {
		t.Fatalf("expected row fields to remain visible, got: %s", view)
	}
}

func TestOutputView_PreservesPlainTextOutput(t *testing.T) {
	t.Parallel()

	ov := components.NewOutputView(48, 24)
	ov.SetContent("plain text\nsecond line")

	view := ov.View()
	if !strings.Contains(view, "plain text") || !strings.Contains(view, "second line") {
		t.Fatalf("expected plain text output to remain unchanged, got: %s", view)
	}
}
