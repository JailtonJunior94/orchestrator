package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/jailtonjunior/orchestrator/internal/hitl"
	"github.com/jailtonjunior/orchestrator/internal/tui/components"
)

const (
	flashDuration   = 6  // frames at ~10 fps ≈ 0.6 s
	successDuration = 20 // frames at ~10 fps ≈ 2 s
	animationFPS    = time.Second / 10
)

// flashTickMsg drives the flash/success animation timer.
type flashTickMsg struct{}

// Update dispatches incoming messages to the correct handler and returns the
// updated model plus any follow-up commands.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// When the palette is open, delegate most messages to it first.
	if m.mode == modePalette {
		return m.handlePaletteMsg(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	// Progress events from the engine goroutine.
	case stepStartedMsg:
		return m.handleStepStarted(msg)

	case stepFinishedMsg:
		return m.handleStepFinished(msg)

	case outputChunkMsg:
		return m.handleOutputChunk(msg)

	case waitApprovalMsg:
		return m.handleWaitApproval(msg)

	case runCompletedMsg:
		m.mode = modeCompleted
		if !m.noAnimation {
			m.successTicks = successDuration
			return m, tea.Batch(listenForProgress(m.progressCh), animationTick())
		}
		return m, tea.Quit

	case runFailedMsg:
		m.mode = modeCompleted
		return m, tea.Quit

	// Flash / success animation ticks.
	case flashTickMsg:
		return m.handleFlashTick()

	// HITL request from the engine via TUIPrompter.
	case promptResponseMsg:
		m.mode = modeWaitingApproval
		m.hitlBar.Show(msg.req.output, msg.req.output)
		return m, nil

	// HITL action chosen by the user in the HITL bar.
	case components.HITLActionMsg:
		return m.handleHITLAction(msg)
	}

	// Forward unhandled messages to focused sub-components.
	var cmd tea.Cmd
	if m.focusedPanel == 1 {
		cmd = m.outputView.Update(msg)
	}
	// Forward to stepList so spinner ticks are processed.
	m.stepList.UpdateSpinner(msg)
	return m, cmd
}

// handleWindowSize updates all component dimensions when the terminal resizes.
func (m model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	leftW, rightW, contentH := layoutDimensions(m.width, m.height, m.mode == modeWaitingApproval)

	m.stepList.SetWidth(leftW)
	m.outputView.SetSize(rightW-4, contentH)
	m.hitlBar.SetWidth(m.width)
	m.syncStatusBar()

	return m, nil
}

// handleKey processes keyboard input.
func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Ctrl+P opens the command palette from any non-approval mode.
	if msg.String() == "ctrl+p" && m.mode != modeWaitingApproval {
		m.mode = modePalette
		cmd := m.palette.Toggle()
		return m, cmd
	}

	// HITL mode: delegate to bar first.
	if m.mode == modeWaitingApproval {
		var barCmd tea.Cmd
		var action int
		m.hitlBar, barCmd, action = m.hitlBar.Update(msg)
		if action >= 0 {
			return m, barCmd
		}
	}

	switch msg.String() {
	case "tab":
		m.focusedPanel = (m.focusedPanel + 1) % 2
	case "q", "ctrl+c":
		if m.mode != modeWaitingApproval {
			return m, tea.Quit
		}
	}

	return m, nil
}

// handlePaletteMsg delegates messages to the palette while it is open.
func (m model) handlePaletteMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var selected *components.PaletteItem
	m.palette, cmd, selected = m.palette.Update(msg)

	if selected != nil {
		switch {
		case selected.Key == "command:run", selected.Key == "command:list":
			m.mode = modeRunning
			m.outputView.SetContent("Command palette cannot switch workflows while a run is active.")
			return m, nil
		case selected.Key == "command:continue":
			m.mode = modeRunning
			m.outputView.SetContent("`continue` is unavailable during an active workflow run.")
			return m, nil
		case strings.HasPrefix(selected.Key, "workflow:"):
			m.mode = modeRunning
			m.outputView.SetContent("Command palette cannot switch workflows while a run is active.")
			return m, nil
		default:
			m.mode = modeRunning
			return m, cmd
		}
	}

	if !m.palette.Visible() {
		// Esc closed the palette.
		m.mode = modeRunning
	}
	return m, cmd
}

// handleFlashTick decrements animation counters and emits another tick if needed.
func (m model) handleFlashTick() (tea.Model, tea.Cmd) {
	active := false

	if m.flashTicks > 0 {
		m.flashTicks--
		active = true
		if m.flashTicks == 0 {
			m.flashStep = ""
		}
	}

	if m.successTicks > 0 {
		m.successTicks--
		active = true
		if m.successTicks == 0 {
			// Animation finished; quit.
			return m, tea.Quit
		}
	}

	if active {
		return m, animationTick()
	}
	return m, nil
}

// animationTick schedules the next animation frame.
func animationTick() tea.Cmd {
	return tea.Tick(animationFPS, func(_ time.Time) tea.Msg {
		return flashTickMsg{}
	})
}

// handleStepStarted marks a step as running and updates both panels.
// Appends the step dynamically if it is not already in the list (e.g. when
// the model is initialised without pre-populated steps).
func (m model) handleStepStarted(msg stepStartedMsg) (tea.Model, tea.Cmd) {
	found := false
	for i, s := range m.steps {
		if s.name == msg.step.name {
			m.steps[i].status = "running"
			m.activeStep = i
			found = true
			break
		}
	}
	if !found {
		m.steps = append(m.steps, stepItem{
			name:     msg.step.name,
			provider: msg.step.provider,
			status:   "running",
		})
		m.activeStep = len(m.steps) - 1
	}
	m.syncStepList()
	m.syncStatusBar()
	cmds := []tea.Cmd{listenForProgress(m.progressCh)}
	if !m.noAnimation {
		cmds = append(cmds, m.stepList.TickSpinner())
	}
	return m, tea.Batch(cmds...)
}

// handleStepFinished marks a step as finished and triggers flash on failure.
func (m model) handleStepFinished(msg stepFinishedMsg) (tea.Model, tea.Cmd) {
	for i, s := range m.steps {
		if s.name == msg.step.name {
			m.steps[i].status = msg.step.status
			m.steps[i].duration = msg.step.duration
			break
		}
	}
	m.syncStepList()
	m.syncStatusBar()

	cmd := listenForProgress(m.progressCh)
	if msg.step.status == "failed" && !m.noAnimation {
		m.flashStep = msg.step.name
		m.flashTicks = flashDuration
		cmd = tea.Batch(cmd, animationTick())
	}
	return m, cmd
}

// handleOutputChunk appends a provider output chunk to the OutputView.
func (m model) handleOutputChunk(msg outputChunkMsg) (tea.Model, tea.Cmd) {
	m.outputView.AppendContent(msg.chunk)
	m.syncStatusBar()
	return m, listenForProgress(m.progressCh)
}

// handleWaitApproval transitions to waiting-approval mode.
func (m model) handleWaitApproval(msg waitApprovalMsg) (tea.Model, tea.Cmd) {
	m.mode = modeWaitingApproval
	m.hitlBar.Show(msg.stepName, msg.output)
	m.syncStatusBar()
	return m, listenForProgress(m.progressCh)
}

// handleHITLAction processes the user action from the HITL bar and responds
// to the engine goroutine via the TUIPrompter.
func (m model) handleHITLAction(msg components.HITLActionMsg) (tea.Model, tea.Cmd) {
	m.mode = modeRunning
	m.hitlBar.Hide()
	m.syncStatusBar()

	if m.prompter != nil {
		result := hitl.PromptResult{Action: msg.Action, Output: msg.Output}
		m.prompter.Respond(result)
	}

	// Re-queue prompter listener for the next HITL cycle.
	var promptCmd tea.Cmd
	if m.prompter != nil {
		promptCmd = m.prompter.WaitForRequest()
	}

	if msg.Action == hitl.ActionExit {
		return m, tea.Quit
	}

	return m, promptCmd
}

// syncStepList pushes the current steps slice into the StepList component.
func (m *model) syncStepList() {
	items := make([]components.StepItem, len(m.steps))
	for i, s := range m.steps {
		items[i] = components.StepItem{
			Name:     s.name,
			Provider: s.provider,
			Status:   s.status,
		}
	}
	m.stepList.SetSteps(items)
	m.stepList.SetActiveStep(m.activeStep)
}

// layoutDimensions computes left panel width, right panel width and content
// height following the 4 Golden Rules.
func layoutDimensions(termWidth, termHeight int, hitlVisible bool) (leftW, rightW, contentH int) {
	// Rule #1: subtract status bar (1 line) and borders (2 lines).
	contentH = termHeight - 1 - 2
	if hitlVisible {
		contentH -= 2 // HITL bar height
	}
	if contentH < 1 {
		contentH = 1
	}

	// Rule #4: weight-based 1:2 split.
	leftW = termWidth / 3
	rightW = termWidth - leftW
	return leftW, rightW, contentH
}
