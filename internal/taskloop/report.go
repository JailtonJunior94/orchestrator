package taskloop

import (
	"fmt"
	"strings"
	"time"
)

// Report armazena os resultados consolidados da execucao do task-loop.
type Report struct {
	PRDFolder  string
	Tool       string
	StartTime  time.Time
	EndTime    time.Time
	StopReason string
	Iterations []IterationResult
	FinalTasks []TaskEntry
}

// IterationResult armazena o resultado de uma unica iteracao do loop.
type IterationResult struct {
	Sequence    int
	TaskID      string
	Title       string
	PreStatus   string
	PostStatus  string
	Duration    time.Duration
	ExitCode    int
	AgentOutput string
	Note        string
}

const maxOutputLen = 2000

// Render gera o relatorio consolidado em formato Markdown.
func (r *Report) Render() []byte {
	var b strings.Builder

	b.WriteString("# Task Loop Execution Report\n\n")

	// Summary
	b.WriteString("## Summary\n")
	fmt.Fprintf(&b, "- **PRD Folder:** `%s`\n", r.PRDFolder)
	fmt.Fprintf(&b, "- **Tool:** %s\n", r.Tool)
	fmt.Fprintf(&b, "- **Start Time:** %s\n", r.StartTime.Format(time.RFC3339))
	fmt.Fprintf(&b, "- **End Time:** %s\n", r.EndTime.Format(time.RFC3339))
	fmt.Fprintf(&b, "- **Total Duration:** %s\n", r.EndTime.Sub(r.StartTime).Truncate(time.Second))
	fmt.Fprintf(&b, "- **Iterations:** %d\n", len(r.Iterations))
	fmt.Fprintf(&b, "- **Stop Reason:** %s\n\n", r.StopReason)

	// Results table
	if len(r.Iterations) > 0 {
		b.WriteString("## Results\n\n")
		b.WriteString("| # | Task ID | Title | Pre-Status | Post-Status | Duration | Exit Code |\n")
		b.WriteString("|---|---------|-------|------------|-------------|----------|-----------|\n")
		for _, it := range r.Iterations {
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %d |\n",
				it.Sequence, it.TaskID, it.Title, it.PreStatus, it.PostStatus,
				it.Duration.Truncate(time.Second), it.ExitCode)
		}
		b.WriteString("\n")
	}

	// Final task status
	if len(r.FinalTasks) > 0 {
		b.WriteString("## Final Task Status\n\n")
		b.WriteString("| Task ID | Title | Final Status |\n")
		b.WriteString("|---------|-------|--------------|\n")
		for _, t := range r.FinalTasks {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", t.ID, t.Title, t.Status)
		}
		b.WriteString("\n")
	}

	// Skipped tasks
	var skipped []IterationResult
	for _, it := range r.Iterations {
		if it.Note != "" {
			skipped = append(skipped, it)
		}
	}
	if len(skipped) > 0 {
		b.WriteString("## Skipped Tasks\n\n")
		for _, it := range skipped {
			fmt.Fprintf(&b, "- **%s** (%s): %s\n", it.TaskID, it.Title, it.Note)
		}
		b.WriteString("\n")
	}

	// Iteration details
	if len(r.Iterations) > 0 {
		b.WriteString("## Iteration Details\n\n")
		for _, it := range r.Iterations {
			fmt.Fprintf(&b, "### Iteration %d: Task %s — %s\n", it.Sequence, it.TaskID, it.Title)
			fmt.Fprintf(&b, "- **Duration:** %s\n", it.Duration.Truncate(time.Second))
			fmt.Fprintf(&b, "- **Exit Code:** %d\n", it.ExitCode)
			fmt.Fprintf(&b, "- **Status Change:** %s -> %s\n", it.PreStatus, it.PostStatus)
			if it.Note != "" {
				fmt.Fprintf(&b, "- **Note:** %s\n", it.Note)
			}
			if it.AgentOutput != "" {
				output := it.AgentOutput
				if len(output) > maxOutputLen {
					output = output[:maxOutputLen] + "\n... (truncated)"
				}
				b.WriteString("- **Agent Output:**\n")
				b.WriteString("  ```\n")
				for _, line := range strings.Split(output, "\n") {
					b.WriteString("  " + line + "\n")
				}
				b.WriteString("  ```\n")
			}
			b.WriteString("\n")
		}
	}

	return []byte(b.String())
}
