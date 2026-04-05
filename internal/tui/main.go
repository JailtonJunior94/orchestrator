package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/term"

	runtimeapp "github.com/jailtonjunior/orchestrator/internal/runtime/application"
	"github.com/jailtonjunior/orchestrator/internal/tui/components"
)

// Wiring holds the TUI-specific communication objects that must be created
// before bootstrap so that the engine receives the correct prompter and
// progress reporter at construction time.
type Wiring struct {
	Prompter   *TUIPrompter
	Reporter   *tuiProgressReporter
	progressCh chan progressEvent
}

// NewWiring constructs TUI-specific channels and wires them into a Wiring
// value. The caller must pass Wiring.Reporter and Wiring.Prompter to
// bootstrap.New so the engine uses the TUI communication objects.
func NewWiring() *Wiring {
	ch := make(chan progressEvent, 64)
	return &Wiring{
		Prompter:   NewTUIPrompter(),
		Reporter:   newProgressReporter(ch).(*tuiProgressReporter),
		progressCh: ch,
	}
}

// ShouldUseTUI returns true when interactive TUI mode should be used.
// Returns false if noTUI is set or stdout is not a terminal.
func ShouldUseTUI(noTUI bool) bool {
	if noTUI {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// RunTUI runs the Bubbletea program and starts the engine in a separate
// goroutine. wiring must be the same object passed to bootstrap.New so that
// the engine and the TUI share the same channels.
//
// If Bubbletea fails to initialise, the function logs a warning and returns an
// actionable error directing the user to --no-tui (ADR-004). No automatic
// fallback to plain text is performed to avoid double-run and state corruption.
// noAnimation must already account for the ORQ_NO_ANIMATION env var; callers
// are responsible for merging flag and env-var values before calling RunTUI.
func RunTUI(ctx context.Context, svc runtimeapp.Service, wiring *Wiring, workflowName, input string, noAnimation bool) error {
	currentWorkflow := workflowName
	currentInput := input

	for {
		summaries, err := svc.ListWorkflowDetails(ctx)
		if err != nil {
			return err
		}
		summary := findSummaryByName(summaries, currentWorkflow)

		wiring.Prompter.Reset()
		wiring.progressCh = make(chan progressEvent, 64)
		wiring.Reporter.Reset(wiring.progressCh)

		engineCtx, cancelEngine := context.WithCancel(ctx)
		doneCh := make(chan error, 1)
		go func(workflowName, input string) {
			_, runErr := svc.Run(engineCtx, workflowName, input)
			wiring.Prompter.Close()
			close(wiring.progressCh)
			doneCh <- runErr
		}(currentWorkflow, currentInput)

		m := initialModel(workflowSteps(summary), wiring.Prompter, wiring.progressCh, noAnimation)
		m.branch = currentGitBranch(ctx)
		m.runStartedAt = time.Now()
		m.setWorkflowSummaries(summaries)
		m.syncStatusBar()

		p := tea.NewProgram(m)
		final, tuiErr := p.Run()

		cancelEngine()
		engineErr := <-doneCh

		if tuiErr != nil {
			slog.Warn("TUI initialization failed; re-run with --no-tui for plain text mode", "error", tuiErr)
			return fmt.Errorf("TUI initialization failed: %w (use --no-tui to bypass)", tuiErr)
		}

		action := requestedAction(final)
		if action == nil {
			return engineErr
		}

		nextWorkflow, err := resolvePaletteAction(summaries, action)
		if err != nil {
			return err
		}
		if nextWorkflow == "" {
			return nil
		}

		currentWorkflow = nextWorkflow
		currentInput, err = collectWorkflowInput(summaryForWorkflow(summaries, nextWorkflow))
		if err != nil {
			return err
		}
	}
}

func findSummaryByName(summaries []runtimeapp.WorkflowSummary, name string) *runtimeapp.WorkflowSummary {
	for _, summary := range summaries {
		if summary.Name == name {
			match := summary
			return &match
		}
	}

	return nil
}

func workflowSteps(summary *runtimeapp.WorkflowSummary) []stepItem {
	if summary == nil {
		return nil
	}

	steps := make([]stepItem, 0, len(summary.StepNames))
	for i, stepName := range summary.StepNames {
		provider := ""
		if i < len(summary.Providers) {
			provider = summary.Providers[i]
		}
		steps = append(steps, stepItem{
			name:     stepName,
			provider: provider,
			status:   "pending",
		})
	}

	return steps
}

func summaryForWorkflow(summaries []runtimeapp.WorkflowSummary, workflowName string) *runtimeapp.WorkflowSummary {
	return findSummaryByName(summaries, workflowName)
}

func requestedAction(final tea.Model) *paletteAction {
	m, ok := final.(model)
	if !ok {
		return nil
	}

	return m.requestedAction
}

func resolvePaletteAction(summaries []runtimeapp.WorkflowSummary, action *paletteAction) (string, error) {
	switch action.kind {
	case paletteActionOpenList:
		return RunListTUI(summaries)
	case paletteActionRunWorkflow:
		return action.workflowName, nil
	default:
		return "", nil
	}
}

func currentGitBranch(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func collectWorkflowInput(summary *runtimeapp.WorkflowSummary) (string, error) {
	if summary == nil || (!summary.RequiresInput && len(summary.Inputs) == 0) {
		return "", nil
	}

	form, result := components.BuildInputForm(summaryInputFields(summary))
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("input form cancelled: %w", err)
	}

	values := make(map[string]any)
	for _, field := range summaryInputFields(summary) {
		if value := result.StringValues[field.Name]; value != nil && strings.TrimSpace(*value) != "" {
			values[field.Name] = *value
		}
		if value := result.BoolValues[field.Name]; value != nil {
			values[field.Name] = *value
		}
	}

	rendered := renderSummaryInput(summary, values)
	if strings.TrimSpace(rendered) == "" {
		return "", fmt.Errorf("missing required inputs: %v", summaryRequiredInputNames(summary))
	}

	return rendered, nil
}

func summaryInputFields(summary *runtimeapp.WorkflowSummary) []components.InputField {
	if summary == nil || len(summary.Inputs) == 0 {
		return []components.InputField{{
			Name:        "input",
			Type:        "text",
			Label:       "Workflow Input",
			Placeholder: "Describe what you want to build...",
		}}
	}

	fields := make([]components.InputField, 0, len(summary.Inputs))
	for _, input := range summary.Inputs {
		fieldType := input.Type
		if fieldType == "" {
			fieldType = "text"
		}
		label := input.Label
		if label == "" {
			label = input.Name
		}
		fields = append(fields, components.InputField{
			Name:        input.Name,
			Type:        fieldType,
			Label:       label,
			Placeholder: input.Placeholder,
			Options:     append([]string(nil), input.Options...),
		})
	}

	return fields
}

func renderSummaryInput(summary *runtimeapp.WorkflowSummary, values map[string]any) string {
	if summary != nil && len(summary.Inputs) == 1 {
		if value, ok := values[summary.Inputs[0].Name]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case bool:
				if typed {
					return "true"
				}
				return "false"
			}
		}
	}

	if len(values) == 1 {
		if value, ok := values["input"]; ok {
			if typed, ok := value.(string); ok {
				return typed
			}
		}
	}

	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return ""
	}

	return string(data)
}

func summaryRequiredInputNames(summary *runtimeapp.WorkflowSummary) []string {
	if summary == nil {
		return []string{"input"}
	}

	names := make([]string, 0, len(summary.Inputs))
	for _, input := range summary.Inputs {
		if input.Required || len(summary.Inputs) == 1 {
			names = append(names, input.Name)
		}
	}
	if len(names) > 0 {
		return names
	}
	if summary.RequiresInput {
		return []string{"input"}
	}

	return nil
}
