package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/runtime"
)

// Renderer prints high-level progress to the terminal.
type Renderer struct {
	writer io.Writer
}

// NewRenderer creates a CLI renderer.
func NewRenderer(writer io.Writer) *Renderer {
	return &Renderer{writer: writer}
}

// StepStarted prints the start of a workflow step.
func (r *Renderer) StepStarted(step runtime.ProgressStep) {
	_, _ = fmt.Fprintf(r.writer, "[%d/%d] %s (%s)\n  ⏳ Gerando...\n", step.Index, step.Total, step.Name, step.Provider)
}

// StepFinished prints the completion of a workflow step.
func (r *Renderer) StepFinished(step runtime.ProgressStep) {
	_, _ = fmt.Fprintf(r.writer, "  ✅ %s (%s)\n", step.Status, step.Duration.Round(10*time.Millisecond))
}

// OutputChunk is a no-op in plain text mode; output is streamed directly by the provider.
func (r *Renderer) OutputChunk(_ string, _ []byte) {}

// WaitingApproval notifies the renderer that a step is awaiting HITL approval.
func (r *Renderer) WaitingApproval(stepName string, _ string) {
	_, _ = fmt.Fprintf(r.writer, "  ⏸  %s waiting for approval\n", stepName)
}

// RunCompleted notifies the renderer that the run finished successfully.
func (r *Renderer) RunCompleted(_ string, status string) {
	_, _ = fmt.Fprintf(r.writer, "✔ Run completed: %s\n", status)
}

// RunFailed notifies the renderer that the run finished with an error.
func (r *Renderer) RunFailed(_ string, err error) {
	_, _ = fmt.Fprintf(r.writer, "✖ Run failed: %v\n", err)
}

// Verify Renderer implements ProgressReporter at compile time.
var _ runtime.ProgressReporter = (*Renderer)(nil)
