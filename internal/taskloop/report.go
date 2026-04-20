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
	b.WriteString(fmt.Sprintf("- **PRD Folder:** `%s`\n", r.PRDFolder))
	b.WriteString(fmt.Sprintf("- **Tool:** %s\n", r.Tool))
	b.WriteString(fmt.Sprintf("- **Start Time:** %s\n", r.StartTime.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- **End Time:** %s\n", r.EndTime.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- **Total Duration:** %s\n", r.EndTime.Sub(r.StartTime).Truncate(time.Second)))
	b.WriteString(fmt.Sprintf("- **Iterations:** %d\n", len(r.Iterations)))
	b.WriteString(fmt.Sprintf("- **Stop Reason:** %s\n\n", r.StopReason))

	// Results table
	if len(r.Iterations) > 0 {
		b.WriteString("## Results\n\n")
		b.WriteString("| # | Task ID | Title | Pre-Status | Post-Status | Duration | Exit Code |\n")
		b.WriteString("|---|---------|-------|------------|-------------|----------|-----------|\n")
		for _, it := range r.Iterations {
			b.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s | %d |\n",
				it.Sequence, it.TaskID, it.Title, it.PreStatus, it.PostStatus,
				it.Duration.Truncate(time.Second), it.ExitCode))
		}
		b.WriteString("\n")
	}

	// Final task status
	if len(r.FinalTasks) > 0 {
		b.WriteString("## Final Task Status\n\n")
		b.WriteString("| Task ID | Title | Final Status |\n")
		b.WriteString("|---------|-------|--------------|\n")
		for _, t := range r.FinalTasks {
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", t.ID, t.Title, t.Status))
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
			b.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", it.TaskID, it.Title, it.Note))
		}
		b.WriteString("\n")
	}

	// Iteration details
	if len(r.Iterations) > 0 {
		b.WriteString("## Iteration Details\n\n")
		for _, it := range r.Iterations {
			b.WriteString(fmt.Sprintf("### Iteration %d: Task %s — %s\n", it.Sequence, it.TaskID, it.Title))
			b.WriteString(fmt.Sprintf("- **Duration:** %s\n", it.Duration.Truncate(time.Second)))
			b.WriteString(fmt.Sprintf("- **Exit Code:** %d\n", it.ExitCode))
			b.WriteString(fmt.Sprintf("- **Status Change:** %s -> %s\n", it.PreStatus, it.PostStatus))
			if it.Note != "" {
				b.WriteString(fmt.Sprintf("- **Note:** %s\n", it.Note))
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
