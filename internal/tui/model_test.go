package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
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

func TestUpdate_OutputChunkMsg(t *testing.T) {
	t.Parallel()

	m := buildModel(120, 40, nil)
	msg := outputChunkMsg{stepName: "build", chunk: []byte("hello world")}
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
