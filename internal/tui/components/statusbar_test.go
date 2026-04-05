package components_test

import (
	"strings"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/tui/components"
)

func TestStatusBar_SingleLine(t *testing.T) {
	t.Parallel()

	sb := components.NewStatusBar(newTestTheme())
	sb.Branch = "main"
	sb.Provider = "claude"
	sb.CurrentStep = 1
	sb.TotalSteps = 3
	sb.Duration = 5 * time.Second

	view := sb.View(120)
	lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 line, got %d: %q", len(lines), view)
	}
}

func TestStatusBar_WideTerminalShowsBranch(t *testing.T) {
	t.Parallel()

	sb := components.NewStatusBar(newTestTheme())
	sb.Branch = "feature/my-branch"
	sb.Provider = "claude"
	sb.CurrentStep = 2
	sb.TotalSteps = 4
	sb.Duration = 10 * time.Second

	view := sb.View(120)
	if !strings.Contains(view, "feature/my-branch") {
		t.Errorf("expected branch in wide terminal view, got: %s", view)
	}
}

func TestStatusBar_NarrowTerminalOmitsBranch(t *testing.T) {
	t.Parallel()

	sb := components.NewStatusBar(newTestTheme())
	sb.Branch = "feature/my-branch"
	sb.Provider = "claude"
	sb.CurrentStep = 1
	sb.TotalSteps = 2
	sb.Duration = 3 * time.Second

	view := sb.View(80)
	if strings.Contains(view, "feature/my-branch") {
		t.Errorf("branch should be omitted in narrow terminal, got: %s", view)
	}
}

func TestStatusBar_ContainsProviderAndStep(t *testing.T) {
	t.Parallel()

	sb := components.NewStatusBar(newTestTheme())
	sb.Provider = "copilot"
	sb.CurrentStep = 3
	sb.TotalSteps = 5

	view := sb.View(120)
	if !strings.Contains(view, "copilot") {
		t.Errorf("expected provider 'copilot' in view, got: %s", view)
	}
	if !strings.Contains(view, "3/5") {
		t.Errorf("expected step '3/5' in view, got: %s", view)
	}
}

func TestStatusBar_MultipleWidths(t *testing.T) {
	t.Parallel()

	cases := []struct {
		width int
		name  string
	}{
		{80, "80x24"},
		{120, "120x40"},
		{60, "60x20"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sb := components.NewStatusBar(newTestTheme())
			sb.Provider = "claude"
			sb.CurrentStep = 1
			sb.TotalSteps = 2
			view := sb.View(tc.width)
			lines := strings.Split(strings.TrimRight(view, "\n"), "\n")
			if len(lines) != 1 {
				t.Errorf("[%s] expected 1 line, got %d", tc.name, len(lines))
			}
		})
	}
}
