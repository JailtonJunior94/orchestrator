package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

func newTestBar() HITLBar {
	return NewHITLBar(theme.New())
}

func TestHITLBar_ViewIsEmptyWhenHidden(t *testing.T) {
	bar := newTestBar()
	if v := bar.View(); v != "" {
		t.Errorf("expected empty view when hidden, got %q", v)
	}
}

func TestHITLBar_ViewShowsStepNameAndActions(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(120)
	bar.Show("build", "some output")

	v := bar.View()
	if !strings.Contains(v, "build") {
		t.Errorf("expected step name 'build' in view, got: %q", v)
	}
	for _, key := range []string{"a", "e", "r", "q"} {
		if !strings.Contains(v, key) {
			t.Errorf("expected key %q in HITL bar view", key)
		}
	}
}

func TestHITLBar_KeyA_EmitsApprove(t *testing.T) {
	bar := newTestBar()
	bar.Show("step1", "output")

	updated, cmd, action := bar.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	if action != int(hitl.ActionApprove) {
		t.Errorf("expected ActionApprove (%d), got %d", hitl.ActionApprove, action)
	}
	if updated.Visible() {
		t.Error("bar should be hidden after action")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	actionMsg, ok := msg.(HITLActionMsg)
	if !ok {
		t.Fatalf("expected HITLActionMsg, got %T", msg)
	}
	if actionMsg.Action != hitl.ActionApprove {
		t.Errorf("expected ActionApprove in message")
	}
}

func TestHITLBar_KeyE_EmitsEdit(t *testing.T) {
	bar := newTestBar()
	bar.Show("step1", "output")
	_, _, action := bar.Update(tea.KeyPressMsg{Text: "e", Code: 'e'})
	if action != int(hitl.ActionEdit) {
		t.Errorf("expected ActionEdit, got %d", action)
	}
}

func TestHITLBar_KeyR_EmitsRedo(t *testing.T) {
	bar := newTestBar()
	bar.Show("step1", "output")
	_, _, action := bar.Update(tea.KeyPressMsg{Text: "r", Code: 'r'})
	if action != int(hitl.ActionRedo) {
		t.Errorf("expected ActionRedo, got %d", action)
	}
}

func TestHITLBar_KeyQ_EmitsExit(t *testing.T) {
	bar := newTestBar()
	bar.Show("step1", "output")
	_, _, action := bar.Update(tea.KeyPressMsg{Text: "q", Code: 'q'})
	if action != int(hitl.ActionExit) {
		t.Errorf("expected ActionExit, got %d", action)
	}
}

func TestHITLBar_NoActionWhenHidden(t *testing.T) {
	bar := newTestBar()
	// bar is not shown — any key should return -1.
	_, _, action := bar.Update(tea.KeyPressMsg{Text: "a", Code: 'a'})
	if action != -1 {
		t.Errorf("expected -1 when bar is hidden, got %d", action)
	}
}
