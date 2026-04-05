// Package tui contains the Bubbletea TUI for the orq CLI.
package tui

import "time"

// mode represents the current interaction mode of the TUI.
// Forward declaration — consumed by model.go in later tasks.
//
//nolint:unused
type mode int

//nolint:unused
const (
	modeRunning mode = iota
	modeWaitingApproval
	modePalette
	modeForm
	modeCompleted
)

// stepItem is the view model for a single workflow step in the step list panel.
// Forward declaration — consumed by components in later tasks.
//
//nolint:unused
type stepItem struct {
	name     string
	provider string
	status   string // maps to domain.StepStatus
	duration time.Duration
}

type paletteActionKind int

const (
	paletteActionNone paletteActionKind = iota
	paletteActionOpenList
	paletteActionRunWorkflow
)

type paletteAction struct {
	kind         paletteActionKind
	workflowName string
}

// Status icon constants used in the step list panel.
const (
	iconRunning         = "●"
	iconApproved        = "✓"
	iconFailed          = "✗"
	iconPending         = "○"
	iconSkipped         = "→"
	iconWaitingApproval = "◎"
)
