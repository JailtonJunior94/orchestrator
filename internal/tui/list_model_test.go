package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
)

func testWorkflows() []runtimeapp.WorkflowSummary {
	return []runtimeapp.WorkflowSummary{
		{Name: "dev-workflow", Summary: "Development workflow", Description: "Creates planning artefacts", Inputs: []runtimeapp.WorkflowInput{{Name: "input", Label: "Feature Request", Type: "multiline"}}, RequiresInput: true, StepNames: []string{"prd", "code"}, Providers: []string{"claude", "claude"}},
		{Name: "prod-workflow", Summary: "Production workflow", StepNames: []string{"build"}, Providers: []string{"claude"}},
		{Name: "staging-workflow", Summary: "Staging workflow", StepNames: []string{"test"}, Providers: []string{"claude"}},
	}
}

func TestListModel_initialState(t *testing.T) {
	m := newListModel(testWorkflows())
	if m.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", m.cursor)
	}
	if len(m.filtered) != len(testWorkflows()) {
		t.Errorf("expected all workflows visible, got %d", len(m.filtered))
	}
	if m.selected != "" {
		t.Error("expected no selection at start")
	}
}

func TestListModel_navigateDown(t *testing.T) {
	m := newListModel(testWorkflows())
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	lm := updated.(listModel)
	if lm.cursor != 1 {
		t.Errorf("expected cursor=1, got %d", lm.cursor)
	}
}

func TestListModel_navigateUp(t *testing.T) {
	m := newListModel(testWorkflows())
	m.cursor = 2
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	lm := updated.(listModel)
	if lm.cursor != 1 {
		t.Errorf("expected cursor=1, got %d", lm.cursor)
	}
}

func TestListModel_enterSelectsWorkflow(t *testing.T) {
	m := newListModel(testWorkflows())
	m.cursor = 1
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	lm := updated.(listModel)
	if lm.selected != "prod-workflow" {
		t.Errorf("expected 'prod-workflow', got %q", lm.selected)
	}
	if cmd == nil {
		t.Error("expected quit command after selection")
	}
}

func TestListModel_escQuits(t *testing.T) {
	m := newListModel(testWorkflows())
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Error("expected quit command on Esc")
	}
}

func TestListModel_filterReducesList(t *testing.T) {
	m := newListModel(testWorkflows())

	// Activate filter mode
	updated, _ := m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	lm := updated.(listModel)
	if !lm.filterActive {
		t.Fatal("expected filterActive after /")
	}

	// Type "dev"
	lm.filter.SetValue("dev")
	lm.applyFilter()

	if len(lm.filtered) != 1 || lm.filtered[0].Name != "dev-workflow" {
		t.Errorf("expected only dev-workflow, got %v", lm.filtered)
	}
}

func TestListModel_renderAt80x24(t *testing.T) {
	m := newListModel(testWorkflows())
	m.width = 80
	m.height = 24

	view := m.View()
	if view.Content == "" {
		t.Error("expected non-empty view at 80x24")
	}
	if !strings.Contains(view.Content, "Development") {
		t.Fatalf("expected summary in view, got %q", view.Content)
	}
	if !strings.Contains(view.Content, "Feature Request") {
		t.Fatalf("expected input preview in view, got %q", view.Content)
	}
}

func TestListModel_cursorDoesNotExceedBounds(t *testing.T) {
	m := newListModel(testWorkflows())
	m.cursor = len(testWorkflows()) - 1

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	lm := updated.(listModel)
	if lm.cursor != len(testWorkflows())-1 {
		t.Errorf("cursor should not exceed list length, got %d", lm.cursor)
	}

	m2 := newListModel(testWorkflows())
	updated2, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	lm2 := updated2.(listModel)
	if lm2.cursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", lm2.cursor)
	}
}
