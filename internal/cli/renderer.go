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
