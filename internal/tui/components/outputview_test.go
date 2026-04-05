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
