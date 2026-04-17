package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jailtonjunior/orchestrator/internal/acp"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
	"github.com/jailtonjunior/orchestrator/internal/tui/components"
)

// buildModel returns an initialised model with the given terminal dimensions
// and optional steps. progressCh is a closed channel so that listenForProgress
// returns immediately (nil msg) without blocking.
func buildModel(width, height int, steps []stepItem) model {
	ch := make(chan progressEvent)
	close(ch)
	m := initialModel(steps, nil, ch, false)
	m.width = width
	m.height = height
	return m
}

// ───────────────────────────────────────────────────────────────────────────
// Layout dimension tests (table-driven)
// ───────────────────────────────────────────────────────────────────────────

func TestLayoutDimensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		termW       int
		termH       int
		hitlVisible bool
		wantLeftW   int
		wantRightW  int
		wantH       int
	}{
		{
			name:        "120x40 horizontal split",
			termW:       120,
			termH:       40,
			hitlVisible: false,
			wantLeftW:   40, // 120/3
			wantRightW:  80, // 120 - 40
			wantH:       37, // 40 - 1(statusbar) - 2(borders)
		},
		{
			name:        "80x24 – still returns values",
			termW:       80,
			termH:       24,
			hitlVisible: false,
			wantLeftW:   26,
			wantRightW:  54,
			wantH:       21,
		},
		{
			name:        "200x60 large terminal",
			termW:       200,
			termH:       60,
			hitlVisible: false,
			wantLeftW:   66,
			wantRightW:  134,
			wantH:       57,
		},
		{
			name:        "120x40 HITL visible subtracts 2",
			termW:       120,
			termH:       40,
			hitlVisible: true,
			wantLeftW:   40,
			wantRightW:  80,
			wantH:       35, // 37 - 2(hitl)
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			leftW, rightW, h := layoutDimensions(tc.termW, tc.termH, tc.hitlVisible)
			if leftW != tc.wantLeftW {
				t.Errorf("leftW = %d, want %d", leftW, tc.wantLeftW)
			}
			if rightW != tc.wantRightW {
				t.Errorf("rightW = %d, want %d", rightW, tc.wantRightW)
			}
			if h != tc.wantH {
				t.Errorf("contentH = %d, want %d", h, tc.wantH)
			}
		})
	}
}

// ───────────────────────────────────────────────────────────────────────────
// Layout direction: horizontal vs vertical stack
// ───────────────────────────────────────────────────────────────────────────

func TestView_HorizontalVsVertical(t *testing.T) {
	t.Parallel()

	steps := []stepItem{
		{name: "compile", provider: "claude", status: "pending"},
	}

	t.Run("width 120 -> horizontal (join horizontal)", func(t *testing.T) {
		t.Parallel()
		m := buildModel(120, 40, steps)
		view := m.View()
		if strings.TrimSpace(view.Content) == "" {
			t.Error("expected non-empty view for 120-col terminal")
		}
	})

	t.Run("width 80 -> vertical stack", func(t *testing.T) {
		t.Parallel()
		m := buildModel(80, 24, steps)
		view := m.View()
		if strings.TrimSpace(view.Content) == "" {
			t.Error("expected non-empty view for 80-col terminal")
		}
	})
}

func TestInitialModel_PopulatesStepListBeforeProgress(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, []stepItem{
		{name: "plan", provider: "claude", status: "pending"},
		{name: "implement", provider: "codex", status: "pending"},
	})

	view := m.View().Content
	if !strings.Contains(view, "plan") {
		t.Fatalf("expected initial view to include first workflow step, got: %q", view)
	}
	if !strings.Contains(view, "implement") {
		t.Fatalf("expected initial view to include second workflow step, got: %q", view)
	}
	if !strings.Contains(view, `Starting step "plan" with claude`) {
		t.Fatalf("expected initial view to include contextual output placeholder, got: %q", view)
	}
	if !strings.Contains(view, "Waiting for provider output") {
		t.Fatalf("expected initial view to include output placeholder, got: %q", view)
	}
}

func TestInitialModel_RendersUsefulContentBeforeWindowResize(t *testing.T) {
	t.Parallel()

	m := initialModel([]stepItem{
		{name: "plan", provider: "claude", status: "pending"},
		{name: "implement", provider: "codex", status: "pending"},
	}, nil, nil, true)

	view := m.View().Content
	if strings.TrimSpace(view) == "" {
		t.Fatal("expected initial view to render before first WindowSizeMsg")
	}
	if !strings.Contains(view, "plan") {
		t.Fatalf("expected initial view to include workflow step before resize, got: %q", view)
	}
	if !strings.Contains(view, "Waiting for provider output") {
		t.Fatalf("expected initial view to include placeholder output before resize, got: %q", view)
	}
}

func TestInitialModel_MarksFirstStepRunningBeforeProgress(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, []stepItem{
		{name: "plan", provider: "claude", status: "pending"},
		{name: "implement", provider: "codex", status: "pending"},
	})

	if got := m.steps[0].status; got != "running" {
		t.Fatalf("first step status = %q, want running", got)
	}
	if got := m.activeStep; got != 0 {
		t.Fatalf("activeStep = %d, want 0", got)
	}
	if got := m.statusBar.CurrentStep; got != 0 {
		t.Fatalf("statusBar.CurrentStep = %d, want 0 before sync", got)
	}

	m.syncStatusBar()
	if got := m.statusBar.CurrentStep; got != 1 {
		t.Fatalf("statusBar.CurrentStep after sync = %d, want 1", got)
	}
	if got := m.statusBar.Provider; got != "claude" {
		t.Fatalf("statusBar.Provider after sync = %q, want claude", got)
	}
}

func TestInitialModel_PreservesExistingActiveStepStatus(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, []stepItem{
		{name: "plan", provider: "claude", status: "approved"},
		{name: "implement", provider: "codex", status: "running"},
	})

	if got := m.steps[0].status; got != "approved" {
		t.Fatalf("first step status = %q, want approved", got)
	}
	if got := m.steps[1].status; got != "running" {
		t.Fatalf("second step status = %q, want running", got)
	}
	if got := m.activeStep; got != 1 {
		t.Fatalf("activeStep = %d, want 1", got)
	}
}

// ───────────────────────────────────────────────────────────────────────────
// Message handling tests
// ───────────────────────────────────────────────────────────────────────────

func TestUpdate_StepStartedMsg(t *testing.T) {
	t.Parallel()

	steps := []stepItem{
		{name: "build", provider: "claude", status: "pending"},
		{name: "test", provider: "claude", status: "pending"},
	}
	m := buildModel(120, 40, steps)

	msg := stepStartedMsg{step: stepItem{name: "test", provider: "claude", status: "running"}}
	updated, _ := m.Update(msg)
	um := updated.(model)

	if um.steps[1].status != "running" {
		t.Errorf("expected steps[1].status = running, got %s", um.steps[1].status)
	}
	if um.activeStep != 1 {
		t.Errorf("expected activeStep = 1, got %d", um.activeStep)
	}
}

func TestUpdate_TypedUpdateMsg(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	msg := typedUpdateMsg{stepName: "build", kind: "message", text: "hello world"}
	updated, _ := m.Update(msg)
	um := updated.(model)

	view := um.outputView.View()
	if !strings.Contains(view, "hello world") {
		t.Errorf("expected outputView to contain 'hello world', got: %q", view)
	}
}

func TestUpdate_WaitApprovalMsg(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	if m.mode != modeRunning {
		t.Fatalf("initial mode should be modeRunning, got %v", m.mode)
	}

	msg := waitApprovalMsg{stepName: "deploy", output: "ready to deploy?"}
	updated, _ := m.Update(msg)
	um := updated.(model)

	if um.mode != modeWaitingApproval {
		t.Errorf("mode = %v, want modeWaitingApproval", um.mode)
	}
	if !um.hitlBar.Visible() {
		t.Error("hitlBar should be visible after waitApprovalMsg")
	}
}

func TestHandleHITLAction_RespondsWithSelectedOutput(t *testing.T) {
	t.Parallel()

	prompter := NewTUIPrompter()
	m := initialModel(nil, prompter, nil, true)
	m.mode = modeWaitingApproval
	m.hitlBar.Show("clarify", "Which authentication flow should be used?\n- Magic link\n- OTP via email")

	go func() {
		updated, _ := m.Update(components.HITLActionMsg{
			Action: hitl.ActionEdit,
			Output: "Magic link",
		})
		_ = updated
	}()

	select {
	case result := <-prompter.responseCh:
		if result.Action != hitl.ActionEdit {
			t.Fatalf("expected ActionEdit, got %v", result.Action)
		}
		if result.Output != "Magic link" {
			t.Fatalf("expected output %q, got %q", "Magic link", result.Output)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected response to be sent to TUIPrompter")
	}
}

func TestUpdate_TabAlternatesFocus(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	if m.focusedPanel != 0 {
		t.Fatalf("initial focus should be 0 (left), got %d", m.focusedPanel)
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	um := updated.(model)
	if um.focusedPanel != 1 {
		t.Errorf("focusedPanel = %d, want 1 after Tab", um.focusedPanel)
	}

	updated2, _ := um.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	um2 := updated2.(model)
	if um2.focusedPanel != 0 {
		t.Errorf("focusedPanel = %d, want 0 after second Tab", um2.focusedPanel)
	}
}

func TestUpdate_PaletteWorkflowSelectionDoesNotInterruptActiveRun(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	m.setWorkflowSummaries([]runtimeapp.WorkflowSummary{{Name: "dev-workflow", Summary: "Dev"}})

	opened, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	withPalette := opened.(model)
	selected, _ := withPalette.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	selectedModel := selected.(model)
	selectedAgain, _ := selectedModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	thirdModel := selectedAgain.(model)
	fourth, _ := thirdModel.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	fourthModel := fourth.(model)
	final, _ := fourthModel.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	result := final.(model)

	if result.requestedAction != nil {
		t.Fatalf("requestedAction = %#v, want nil", result.requestedAction)
	}
	if result.mode != modeRunning {
		t.Fatalf("mode = %v, want modeRunning", result.mode)
	}
	if !strings.Contains(result.outputView.View(), "cannot switch workflows while a run is active") {
		t.Fatalf("output = %q", result.outputView.View())
	}
}

func TestUpdate_StepStartedSyncsStatusBar(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, []stepItem{
		{name: "prd", provider: "gemini", status: "pending"},
		{name: "tasks", provider: "codex", status: "pending"},
	})
	m.branch = "main"
	m.runStartedAt = time.Now().Add(-2 * time.Second)

	updated, _ := m.Update(stepStartedMsg{step: stepItem{name: "tasks", provider: "codex", status: "running"}})
	um := updated.(model)

	if um.statusBar.Branch != "main" {
		t.Fatalf("branch = %q", um.statusBar.Branch)
	}
	if um.statusBar.Provider != "codex" {
		t.Fatalf("provider = %q", um.statusBar.Provider)
	}
	if um.statusBar.CurrentStep != 2 || um.statusBar.TotalSteps != 2 {
		t.Fatalf("step = %d/%d", um.statusBar.CurrentStep, um.statusBar.TotalSteps)
	}
}

// ───────────────────────────────────────────────────────────────────────────
// Height accounting: borders, status bar and HITL bar
// ───────────────────────────────────────────────────────────────────────────

func TestLayoutDimensions_HeightAccountsForDeductions(t *testing.T) {
	t.Parallel()

	termH := 50
	_, _, noHITL := layoutDimensions(120, termH, false)
	_, _, withHITL := layoutDimensions(120, termH, true)

	// Without HITL: height = termH - 1(statusbar) - 2(borders) = 47
	if noHITL != termH-3 {
		t.Errorf("noHITL contentH = %d, want %d", noHITL, termH-3)
	}
	// With HITL: subtract 2 more = 45
	if withHITL != termH-5 {
		t.Errorf("withHITL contentH = %d, want %d", withHITL, termH-5)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Typed update rendering — task 7.0
// ─────────────────────────────────────────────────────────────────────────────

func TestUpdate_TypedUpdate_Thought(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	msg := typedUpdateMsg{stepName: "plan", kind: "thought", text: "analyzing the problem"}
	updated, _ := m.Update(msg)
	um := updated.(model)

	view := um.outputView.View()
	if !strings.Contains(view, "[thinking]") {
		t.Errorf("expected '[thinking]' prefix for thought update, got: %q", view)
	}
	if !strings.Contains(view, "analyzing the problem") {
		t.Errorf("expected thought text in view, got: %q", view)
	}
}

func TestUpdate_TypedUpdate_ToolCall(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	msg := typedUpdateMsg{
		stepName: "plan",
		kind:     "tool_call",
		toolCall: &acp.ToolCallInfo{ID: "tc-1", Title: "Edit main.go", Status: "running", Input: `{"path":"main.go"}`},
	}
	updated, _ := m.Update(msg)
	um := updated.(model)

	view := um.outputView.View()
	if !strings.Contains(view, "Edit main.go") {
		t.Errorf("expected tool call title in view, got: %q", view)
	}
	if !strings.Contains(view, "running") {
		t.Errorf("expected tool call status in view, got: %q", view)
	}
	if !strings.Contains(view, `{"path":"main.go"}`) {
		t.Errorf("expected tool call input in view, got: %q", view)
	}
}

func TestUpdate_TypedUpdate_ToolUpdate_UpdatesInPlace(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)

	// Initial tool_call.
	m2, _ := m.Update(typedUpdateMsg{
		stepName: "plan",
		kind:     "tool_call",
		toolCall: &acp.ToolCallInfo{ID: "tc-2", Title: "Run tests", Status: "running"},
	})
	m3, _ := m2.Update(typedUpdateMsg{
		stepName: "plan",
		kind:     "tool_update",
		toolCall: &acp.ToolCallInfo{ID: "tc-2", Title: "Run tests", Status: "completed"},
	})
	um := m3.(model)

	view := um.outputView.View()
	if !strings.Contains(view, "Run tests") {
		t.Errorf("expected tool name in view, got: %q", view)
	}
	if !strings.Contains(view, "completed") {
		t.Errorf("expected updated status 'completed', got: %q", view)
	}
	// Exactly one occurrence of the tool — not duplicated.
	if count := strings.Count(view, "Run tests"); count != 1 {
		t.Errorf("expected tool line once, found %d times in: %q", count, view)
	}
}

func TestUpdate_TypedUpdate_Permission(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	msg := typedUpdateMsg{stepName: "plan", kind: "permission", text: "write to /tmp/output.txt"}
	updated, _ := m.Update(msg)
	um := updated.(model)

	view := um.outputView.View()
	if !strings.Contains(view, "[permission]") {
		t.Errorf("expected '[permission]' prefix for permission update, got: %q", view)
	}
	if !strings.Contains(view, "write to /tmp/output.txt") {
		t.Errorf("expected permission description in view, got: %q", view)
	}
}
