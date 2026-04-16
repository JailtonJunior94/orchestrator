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

func TestHITLBar_ViewShowsQuestionChoicesAndCustomAnswer(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(120)
	bar.Show("clarify", "Which authentication flow should be used?\n- Magic link\n- OTP via email")

	v := bar.View()
	for _, expected := range []string{"Magic link", "OTP via email", "Custom answer"} {
		if !strings.Contains(v, expected) {
			t.Fatalf("expected %q in HITL bar view, got: %q", expected, v)
		}
	}
}

func TestHITLBar_ViewShowsCanonicalAnswersForOpenQuestions(t *testing.T) {
	bar := newTestBar()
	bar.SetWidth(120)
	bar.Show("clarify", "Antes de gerar o PRD, responda com o que souber:\n- Taxa de conversão de login?\n- Tempo máximo de entrega do e-mail?")

	v := bar.View()
	for _, expected := range []string{
		"I don't know yet",
		"Use reasonable assumptions",
		"Mark as TBD and continue",
		"Custom answer",
	} {
		if !strings.Contains(v, expected) {
			t.Fatalf("expected %q in HITL bar view, got: %q", expected, v)
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

func TestHITLBar_EnterOnChoice_EmitsEditWithSelectedOutput(t *testing.T) {
	bar := newTestBar()
	bar.Show("clarify", "Which authentication flow should be used?\n- Magic link\n- OTP via email")

	updated, cmd, action := bar.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if action != int(hitl.ActionEdit) {
		t.Fatalf("expected ActionEdit, got %d", action)
	}
	if updated.Visible() {
		t.Fatal("bar should be hidden after selecting an answer")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	actionMsg, ok := msg.(HITLActionMsg)
	if !ok {
		t.Fatalf("expected HITLActionMsg, got %T", msg)
	}
	if actionMsg.Action != hitl.ActionEdit {
		t.Fatalf("expected ActionEdit, got %v", actionMsg.Action)
	}
	if actionMsg.Output != "Magic link" {
		t.Fatalf("expected selected output %q, got %q", "Magic link", actionMsg.Output)
	}
}

func TestHITLBar_CustomAnswer_EmitsEditWithTypedOutput(t *testing.T) {
	bar := newTestBar()
	bar.Show("clarify", "Which authentication flow should be used?\n- Magic link\n- OTP via email")

	bar, _, action := bar.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if action != -1 {
		t.Fatalf("expected no action while moving cursor, got %d", action)
	}
	bar, _, action = bar.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if action != -1 {
		t.Fatalf("expected no action while moving cursor, got %d", action)
	}
	bar, cmd, action := bar.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if action != -1 {
		t.Fatalf("expected no action when opening custom answer mode, got %d", action)
	}
	if cmd == nil {
		t.Fatal("expected focus command when entering custom answer mode")
	}

	for _, key := range []tea.KeyPressMsg{
		{Code: 'C', Text: "C"},
		{Code: 'u', Text: "u"},
		{Code: 's', Text: "s"},
		{Code: 't', Text: "t"},
		{Code: 'o', Text: "o"},
		{Code: 'm', Text: "m"},
	} {
		bar, _, action = bar.Update(key)
		if action != -1 {
			t.Fatalf("expected no action while typing custom answer, got %d", action)
		}
	}

	updated, submitCmd, action := bar.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if action != int(hitl.ActionEdit) {
		t.Fatalf("expected ActionEdit after submitting custom answer, got %d", action)
	}
	if submitCmd == nil {
		t.Fatal("expected non-nil submit command")
	}

	msg := submitCmd()
	actionMsg, ok := msg.(HITLActionMsg)
	if !ok {
		t.Fatalf("expected HITLActionMsg, got %T", msg)
	}
	if actionMsg.Output != "Custom" {
		t.Fatalf("expected custom output %q, got %q", "Custom", actionMsg.Output)
	}
	if updated.Visible() {
		t.Fatal("bar should be hidden after submitting custom answer")
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

func TestExtractQuestionChoices_ReturnsNilWithoutQuestionContext(t *testing.T) {
	choices := extractQuestionChoices("Implementation plan:\n- Create PRD\n- Write tests")
	if choices != nil {
		t.Fatalf("expected nil choices, got %#v", choices)
	}
}

func TestBuildAnswerOptions_UsesCanonicalAnswersForEscapedProviderQuestionnaire(t *testing.T) {
	options := buildAnswerOptions("{\"type\":\"result\",\"subtype\":\"success\",\"result\":\"Antes de gerar o PRD, responda com o que souber:\\n- Taxa de conversão de login?\\n- Tempo máximo de entrega do e-mail?\"}")
	if len(options) != 3 {
		t.Fatalf("expected 3 canonical options, got %d", len(options))
	}
	if options[0].Label != "I don't know yet" {
		t.Fatalf("first label = %q", options[0].Label)
	}
}
