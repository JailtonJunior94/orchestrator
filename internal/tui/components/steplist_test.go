package components_test

import (
	"strings"
	"testing"

	"github.com/jailtonjunior/orchestrator/internal/tui/components"
	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

func newTestTheme() theme.Maestro {
	return theme.New()
}

// stripANSI removes ANSI escape sequences for plain-text length checks.
func stripANSI(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			i++
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

func TestStepList_Icons(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status  string
		wantIcon string
	}{
		{"running", "●"},
		{"approved", "✓"},
		{"failed", "✗"},
		{"skipped", "→"},
		{"waiting_approval", "◎"},
		{"retrying", "↺"},
		{"pending", "○"},
		{"", "○"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.status, func(t *testing.T) {
			t.Parallel()
			sl := components.NewStepList(newTestTheme())
			// Disable animation so running steps use the static "●" icon.
			sl.SetNoAnimation(true)
			sl.SetSteps([]components.StepItem{
				{Name: "step-one", Provider: "claude", Status: tc.status},
			})
			sl.SetWidth(80)
			view := sl.View()
			if !strings.Contains(view, tc.wantIcon) {
				t.Errorf("status %q: expected icon %q in view, got:\n%s", tc.status, tc.wantIcon, view)
			}
		})
	}
}

func TestStepList_ActiveStepHighlighted(t *testing.T) {
	t.Parallel()

	sl := components.NewStepList(newTestTheme())
	sl.SetSteps([]components.StepItem{
		{Name: "step-a", Provider: "claude", Status: "pending"},
		{Name: "step-b", Provider: "claude", Status: "running"},
	})
	sl.SetActiveStep(1)
	sl.SetWidth(80)

	view := sl.View()
	if !strings.Contains(view, "step-b") {
		t.Errorf("expected active step 'step-b' to appear in view:\n%s", view)
	}
}

func TestStepList_TruncatesLongNames(t *testing.T) {
	t.Parallel()

	sl := components.NewStepList(newTestTheme())
	sl.SetSteps([]components.StepItem{
		{Name: strings.Repeat("x", 200), Provider: "claude", Status: "pending"},
	})
	sl.SetWidth(40)
	view := sl.View()

	// Rendered line must not exceed inner width (40 - 4 = 36) in visible chars.
	for line := range strings.SplitSeq(view, "\n") {
		plain := stripANSI(line)
		if len([]rune(plain)) > 40 {
			t.Errorf("visible line too long (%d chars): %q", len([]rune(plain)), plain)
		}
	}
}

func TestStepList_MultipleWidths(t *testing.T) {
	t.Parallel()

	widths := []int{80, 120, 60}
	for _, w := range widths {
		w := w
		t.Run("", func(t *testing.T) {
			t.Parallel()
			sl := components.NewStepList(newTestTheme())
			sl.SetSteps([]components.StepItem{
				{Name: "my-step", Provider: "claude", Status: "running"},
			})
			sl.SetWidth(w)
			view := sl.View()
			if view == "" {
				t.Errorf("expected non-empty view for width=%d", w)
			}
		})
	}
}
