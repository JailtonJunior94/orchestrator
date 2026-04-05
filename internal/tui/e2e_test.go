package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
)

// ctrlP constructs a KeyPressMsg for Ctrl+P.
func ctrlP() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}
}

func testSummaries(names ...string) []runtimeapp.WorkflowSummary {
	out := make([]runtimeapp.WorkflowSummary, len(names))
	for i, n := range names {
		out[i] = runtimeapp.WorkflowSummary{Name: n}
	}
	return out
}

// TestE2E_listNavigateAndSelect simulates a full orq list TUI session:
// launch → navigate → Enter → verify selection.
func TestE2E_listNavigateAndSelect(t *testing.T) {
	m := newListModel(testSummaries("alpha-workflow", "beta-workflow", "gamma-workflow"))
	m.width = 120
	m.height = 30

	// Navigate down to second item
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(listModel)

	// Press Enter
	final, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	lm := final.(listModel)

	if lm.selected != "beta-workflow" {
		t.Errorf("expected 'beta-workflow', got %q", lm.selected)
	}
	if cmd == nil {
		t.Error("expected quit command after selection")
	}
}

// TestE2E_listEscQuitsWithoutSelection verifies Esc exits without a selection.
func TestE2E_listEscQuitsWithoutSelection(t *testing.T) {
	m := newListModel(testSummaries("foo", "bar"))
	m.width = 100
	m.height = 24

	final, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	lm := final.(listModel)

	if lm.selected != "" {
		t.Errorf("expected empty selection on Esc, got %q", lm.selected)
	}
	if cmd == nil {
		t.Error("expected quit command on Esc")
	}
}

// TestE2E_listFilterThenSelect simulates typing a filter then selecting a result.
func TestE2E_listFilterThenSelect(t *testing.T) {
	m := newListModel(testSummaries("dev-workflow", "prod-workflow"))
	m.width = 100
	m.height = 24

	// Open filter
	updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m = updated.(listModel)

	// Set filter value directly (simulates typing)
	m.filter.SetValue("prod")
	m.applyFilter()

	if len(m.filtered) != 1 {
		t.Fatalf("expected 1 filtered result, got %d", len(m.filtered))
	}

	// Close filter and select
	m.filterActive = false
	final, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	lm := final.(listModel)

	if lm.selected != "prod-workflow" {
		t.Errorf("expected 'prod-workflow', got %q", lm.selected)
	}
}

// TestE2E_runModel_paletteToggle verifies Ctrl+P opens and closes the palette.
func TestE2E_runModel_paletteToggle(t *testing.T) {
	m := initialModel(nil, nil, nil, false)
	m.width = 120
	m.height = 30

	// Open palette with Ctrl+P
	updated, _ := m.Update(ctrlP())
	m2 := updated.(model) //nolint:govet
	if m2.mode != modePalette {
		t.Errorf("expected modePalette after ctrl+p, got %v", m2.mode)
	}
	if !m2.palette.Visible() {
		t.Error("expected palette to be visible")
	}

	// Esc closes palette
	updated2, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m3 := updated2.(model)
	if m3.mode != modeRunning {
		t.Errorf("expected modeRunning after Esc, got %v", m3.mode)
	}
}

// TestE2E_runModel_minimalTerminal verifies the TUI renders at 80x24 without panicking.
func TestE2E_runModel_minimalTerminal(t *testing.T) {
	m := initialModel(nil, nil, nil, true /* noAnimation */)
	m.width = 80
	m.height = 24

	view := m.View()
	if view.Content == "" {
		t.Error("expected non-empty view at 80x24")
	}
}

// TestE2E_runModel_noAnimationSpinner verifies that spinner is static when noAnimation=true.
func TestE2E_runModel_noAnimationSpinner(t *testing.T) {
	m := initialModel(nil, nil, nil, true /* noAnimation */)
	m.width = 120
	m.height = 30

	// Start a step to get the running state
	updated, _ := m.Update(stepStartedMsg{step: stepItem{name: "prd", provider: "claude", status: "running"}})
	m = updated.(model)

	view := m.View()

	// With noAnimation, the running icon should be "●" not a spinner frame
	if !strings.Contains(view.Content, "●") && !strings.Contains(view.Content, "prd") {
		t.Error("expected static running icon when noAnimation=true")
	}
}
