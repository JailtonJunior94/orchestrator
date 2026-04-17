package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jailtonjunior/orchestrator/internal/acp"
	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
	"github.com/jailtonjunior/orchestrator/internal/tui/components"
	"github.com/jailtonjunior/orchestrator/internal/tui/theme"
)

const (
	defaultInitialWidth  = 120
	defaultInitialHeight = 30
)

// progressEvent is the interface that all progress channel events implement.
// Each concrete event type also implements tea.Msg.
type progressEvent interface {
	progressEventTag()
}

// stepStartedMsg is emitted when a workflow step begins execution.
type stepStartedMsg struct{ step stepItem }

func (stepStartedMsg) progressEventTag() {}

// stepFinishedMsg is emitted when a workflow step completes.
type stepFinishedMsg struct{ step stepItem }

func (stepFinishedMsg) progressEventTag() {}

// typedUpdateMsg carries a typed streaming update from the ACP agent.
type typedUpdateMsg struct {
	stepName string
	kind     string
	text     string
	toolCall *acp.ToolCallInfo
}

func (typedUpdateMsg) progressEventTag() {}

// waitApprovalMsg signals that a step is waiting for HITL approval.
type waitApprovalMsg struct {
	stepName string
	output   string
}

func (waitApprovalMsg) progressEventTag() {}

// runCompletedMsg signals that the run finished successfully.
type runCompletedMsg struct{}

func (runCompletedMsg) progressEventTag() {}

// runFailedMsg signals that the run failed.
type runFailedMsg struct{}

func (runFailedMsg) progressEventTag() {}

// model is the root Bubbletea model for the orq TUI.
type model struct {
	width, height int
	mode          mode

	// Workflow state
	steps      []stepItem
	activeStep int

	// Sub-components
	stepList   components.StepList
	outputView components.OutputView
	hitlBar    components.HITLBar
	statusBar  components.StatusBar
	palette    components.PaletteModel

	// Flash / success animation state
	flashStep    string // step name currently flashing (empty = none)
	flashTicks   int    // remaining frames for flash
	successTicks int    // remaining frames for success colour

	// Communication
	prompter   *TUIPrompter
	progressCh <-chan progressEvent

	// Config
	noAnimation  bool
	theme        theme.Maestro
	branch       string
	runStartedAt time.Time
	summaries    []runtimeapp.WorkflowSummary

	// Focus: 0 = left panel (steps), 1 = right panel (output)
	focusedPanel    int
	requestedAction *paletteAction
}

// defaultPaletteItems returns the fixed command entries shown in the palette.
func defaultPaletteItems(summaries []runtimeapp.WorkflowSummary) []components.PaletteItem {
	items := []components.PaletteItem{
		{Key: "command:run", Title: "run", Description: "Choose and run a workflow"},
		{Key: "command:list", Title: "list", Description: "Open the workflow selector"},
		{Key: "command:continue", Title: "continue", Description: "Unavailable during an active run"},
	}

	for _, summary := range summaries {
		description := strings.TrimSpace(summary.Summary)
		if description == "" {
			description = strings.TrimSpace(summary.Description)
		}
		items = append(items, components.PaletteItem{
			Key:         "workflow:" + summary.Name,
			Title:       summary.Name,
			Description: description,
		})
	}

	return items
}

// initialModel constructs the initial model with all sub-components wired up.
func initialModel(
	steps []stepItem,
	prompter *TUIPrompter,
	progressCh <-chan progressEvent,
	noAnimation bool,
) model {
	t := theme.New()
	sl := components.NewStepList(t)
	sl.SetNoAnimation(noAnimation)
	initialSteps, activeStep := prepareInitialSteps(steps)
	m := model{
		mode:        modeRunning,
		width:       defaultInitialWidth,
		height:      defaultInitialHeight,
		steps:       initialSteps,
		activeStep:  activeStep,
		stepList:    sl,
		outputView:  components.NewOutputView(80, 20),
		hitlBar:     components.NewHITLBar(t),
		statusBar:   components.NewStatusBar(t),
		palette:     components.NewPaletteModel(defaultPaletteItems(nil), t),
		prompter:    prompter,
		progressCh:  progressCh,
		noAnimation: noAnimation,
		theme:       t,
	}

	m.syncStepList()
	m.outputView.SetContent(m.initialOutputPlaceholder())
	return m
}

func prepareInitialSteps(steps []stepItem) ([]stepItem, int) {
	if len(steps) == 0 {
		return nil, 0
	}

	cloned := append([]stepItem(nil), steps...)
	activeStep := 0
	hasActive := false

	for i := range cloned {
		if cloned[i].status == "running" || cloned[i].status == "retrying" || cloned[i].status == "waiting_approval" {
			activeStep = i
			hasActive = true
			break
		}
	}

	if !hasActive && cloned[0].status == "pending" {
		cloned[0].status = "running"
	}

	return cloned, activeStep
}

func (m model) initialOutputPlaceholder() string {
	if len(m.steps) == 0 || m.activeStep < 0 || m.activeStep >= len(m.steps) {
		return "Waiting for provider output..."
	}

	step := m.steps[m.activeStep]
	if step.provider != "" {
		return fmt.Sprintf("Starting step %q with %s...\nWaiting for provider output...", step.name, step.provider)
	}

	return fmt.Sprintf("Starting step %q...\nWaiting for provider output...", step.name)
}

func (m *model) setWorkflowSummaries(summaries []runtimeapp.WorkflowSummary) {
	m.summaries = append([]runtimeapp.WorkflowSummary(nil), summaries...)
	m.palette.SetItems(defaultPaletteItems(m.summaries))
}

func (m *model) syncStatusBar() {
	m.statusBar.Branch = m.branch
	m.statusBar.TotalSteps = len(m.steps)
	if len(m.steps) == 0 || m.activeStep < 0 || m.activeStep >= len(m.steps) {
		m.statusBar.CurrentStep = 0
		m.statusBar.Provider = ""
		return
	}

	m.statusBar.CurrentStep = m.activeStep + 1
	m.statusBar.Provider = m.steps[m.activeStep].provider
	if !m.runStartedAt.IsZero() {
		m.statusBar.Duration = time.Since(m.runStartedAt)
	}
}

// Init returns the commands to start listening for progress events and HITL
// requests as soon as the program launches.
func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		listenForProgress(m.progressCh),
	}
	if m.prompter != nil {
		cmds = append(cmds, m.prompter.WaitForRequest())
	}
	return tea.Batch(cmds...)
}

// listenForProgress returns a tea.Cmd that reads one event from progressCh
// and converts it to a tea.Msg. The Cmd is re-scheduled after each event.
func listenForProgress(ch <-chan progressEvent) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		event, ok := <-ch
		if !ok {
			return nil
		}
		return event
	}
}
