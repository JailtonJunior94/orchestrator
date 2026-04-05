package components

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

func testPaletteItems() []PaletteItem {
	return []PaletteItem{
		{Key: "run", Title: "run", Description: "Run a workflow"},
		{Key: "list", Title: "list", Description: "List workflows"},
		{Key: "continue", Title: "continue", Description: "Continue a run"},
		{Key: "dev-workflow", Title: "dev-workflow", Description: "Dev workflow"},
		{Key: "prod-workflow", Title: "prod-workflow", Description: "Prod workflow"},
	}
}

func TestFuzzyFilter_matchesContains(t *testing.T) {
	items := testPaletteItems()

	got := FuzzyFilter(items, "dev")
	if len(got) != 1 || got[0].Key != "dev-workflow" {
		t.Errorf("expected [dev-workflow], got %v", got)
	}
}

func TestFuzzyFilter_noMatchForOther(t *testing.T) {
	items := testPaletteItems()

	got := FuzzyFilter(items, "dev")
	for _, item := range got {
		if item.Key == "prod-workflow" {
			t.Errorf("prod-workflow should not match 'dev' query")
		}
	}
}

func TestFuzzyFilter_emptyQueryReturnsAll(t *testing.T) {
	items := testPaletteItems()
	got := FuzzyFilter(items, "")
	if len(got) != len(items) {
		t.Errorf("expected all items for empty query, got %d", len(got))
	}
}

func TestFuzzyFilter_caseInsensitive(t *testing.T) {
	items := testPaletteItems()
	got := FuzzyFilter(items, "DEV")
	if len(got) != 1 || got[0].Key != "dev-workflow" {
		t.Errorf("expected case-insensitive match, got %v", got)
	}
}

func TestPaletteModel_toggleVisible(t *testing.T) {
	p := NewPaletteModel(testPaletteItems(), theme.New())

	if p.Visible() {
		t.Fatal("palette should start hidden")
	}

	p.Toggle()
	if !p.Visible() {
		t.Error("palette should be visible after first toggle")
	}

	p.Toggle()
	if p.Visible() {
		t.Error("palette should be hidden after second toggle")
	}
}

func TestPaletteModel_escClosePalette(t *testing.T) {
	p := NewPaletteModel(testPaletteItems(), theme.New())
	p.Toggle()

	p, _, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEscape, Text: ""})
	if p.Visible() {
		t.Error("palette should be hidden after Esc")
	}
}

func TestPaletteModel_enterSelectsItem(t *testing.T) {
	items := testPaletteItems()
	p := NewPaletteModel(items, theme.New())
	p.Toggle()

	// cursor is at 0 → selects first item
	p, _, selected := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if selected == nil {
		t.Fatal("expected a selected item")
	}
	if selected.Key != items[0].Key {
		t.Errorf("expected %q, got %q", items[0].Key, selected.Key)
	}
	if p.Visible() {
		t.Error("palette should close after selection")
	}
}

func TestPaletteModel_downUpCursor(t *testing.T) {
	p := NewPaletteModel(testPaletteItems(), theme.New())
	p.Toggle()

	p, _, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	p, _, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if p.cursor != 2 {
		t.Errorf("expected cursor=2 after two downs, got %d", p.cursor)
	}

	p, _, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if p.cursor != 1 {
		t.Errorf("expected cursor=1 after up, got %d", p.cursor)
	}
}
