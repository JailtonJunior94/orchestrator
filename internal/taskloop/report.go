package taskloop

import (
	"fmt"
	"strings"
	"time"
)

// ReviewResult armazena o resultado da revisao de uma task.
type ReviewResult struct {
	Duration  time.Duration
	ExitCode  int
	Output    string
	Note      string
	Succeeded bool
}

// Report armazena os resultados consolidados da execucao do task-loop.
type Report struct {
	PRDFolder       string
	Tool            string            // preservado para modo simples
	Mode            string            // "simples" ou "avancado"
	ExecutorProfile *ExecutionProfile // nil em modo simples
	ReviewerProfile *ExecutionProfile // nil quando reviewer nao configurado
	StartTime       time.Time
	EndTime         time.Time
	StopReason      string
	Summary         FinalSummary
	Iterations      []IterationResult
	FinalTasks      []TaskEntry
}

// IterationResult armazena o resultado de uma unica iteracao do loop.
type IterationResult struct {
	Sequence     int
	TaskID       string
	Title        string
	PreStatus    string
	PostStatus   string
	Duration     time.Duration
	ExitCode     int
	AgentOutput  string
	Note         string
	Role         string        // "executor" (default) — para compatibilidade
	ReviewResult *ReviewResult // nil quando review nao executado
}

const maxOutputLen = 2000

// Render gera o relatorio consolidado em formato Markdown.
// Quando Mode == "avancado", inclui perfis de execucao, coluna Papel e sub-linhas de reviewer.
// Caso contrario, produz o formato simples backward-compatible.
func (r *Report) Render() []byte {
	if r.Mode == "avancado" {
		return r.renderAvancado()
	}
	return r.renderSimples()
}

func (r *Report) renderSimples() []byte {
	var b strings.Builder
	summary := r.effectiveSummary()

	b.WriteString("# Task Loop Execution Report\n\n")

	// Summary
	b.WriteString("## Summary\n")
	fmt.Fprintf(&b, "- **PRD Folder:** `%s`\n", r.PRDFolder)
	if r.Mode == "simples" {
		b.WriteString("- **Modo:** simples\n")
	}
	fmt.Fprintf(&b, "- **Tool:** %s\n", r.Tool)
	fmt.Fprintf(&b, "- **Start Time:** %s\n", r.StartTime.Format(time.RFC3339))
	fmt.Fprintf(&b, "- **End Time:** %s\n", r.EndTime.Format(time.RFC3339))
	fmt.Fprintf(&b, "- **Total Duration:** %s\n", r.EndTime.Sub(r.StartTime).Truncate(time.Second))
	fmt.Fprintf(&b, "- **Iterations:** %d\n", summary.IterationsRun)
	fmt.Fprintf(&b, "- **Stop Reason:** %s\n", firstNonEmpty(summary.StopReason, "execucao encerrada"))
	fmt.Fprintf(&b, "- **Batch Progress:** %s\n", formatBatchProgress(summary.Progress))
	if summary.ReportPath != "" {
		fmt.Fprintf(&b, "- **Report Path:** `%s`\n", summary.ReportPath)
	}
	if summary.LastFailure != nil {
		fmt.Fprintf(&b, "- **Last Failure:** %s\n", renderFailureMessage(summary.LastFailure))
	}
	b.WriteString("\n")

	// Results table — sem coluna Papel
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

	// Resumo por status
	if len(r.Iterations) > 0 {
		var success, skippedCount, failedCount int
		for _, it := range r.Iterations {
			switch {
			// postStatus == "done" e o criterio primario de sucesso.
			// exit code nao-zero (ex: -1 por timeout) e ignorado quando a task
			// foi concluida: o agente pode ter sido morto por SIGKILL apos marcar
			// a task como done, e o resultado observavel e o estado da task.
			case it.PostStatus == "done":
				success++
			case it.PostStatus == "failed" || it.ExitCode != 0:
				failedCount++
			default:
				skippedCount++
			}
		}
		b.WriteString("## Resumo\n\n")
		fmt.Fprintf(&b, "- **Executadas com sucesso:** %d\n", success)
		fmt.Fprintf(&b, "- **Puladas:** %d\n", skippedCount)
		fmt.Fprintf(&b, "- **Falhadas:** %d\n\n", failedCount)
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

func (r *Report) renderAvancado() []byte {
	var b strings.Builder
	summary := r.effectiveSummary()

	b.WriteString("# Task Loop Execution Report\n\n")

	// Summary — modo avancado
	b.WriteString("## Summary\n")
	fmt.Fprintf(&b, "- **PRD Folder:** `%s`\n", r.PRDFolder)
	b.WriteString("- **Modo:** avancado\n")
	if r.ExecutorProfile != nil {
		ep := r.ExecutorProfile
		fmt.Fprintf(&b, "- **Executor:** %s / %s / %s\n", ep.Tool(), ep.Provider(), modelOrDefault(ep.Model()))
	}
	if r.ReviewerProfile != nil {
		rp := r.ReviewerProfile
		fmt.Fprintf(&b, "- **Reviewer:** %s / %s / %s\n", rp.Tool(), rp.Provider(), modelOrDefault(rp.Model()))
	} else {
		b.WriteString("- **Reviewer:** nao configurado\n")
	}
	fmt.Fprintf(&b, "- **Start Time:** %s\n", r.StartTime.Format(time.RFC3339))
	fmt.Fprintf(&b, "- **End Time:** %s\n", r.EndTime.Format(time.RFC3339))
	fmt.Fprintf(&b, "- **Total Duration:** %s\n", r.EndTime.Sub(r.StartTime).Truncate(time.Second))
	fmt.Fprintf(&b, "- **Iterations:** %d\n", summary.IterationsRun)
	fmt.Fprintf(&b, "- **Stop Reason:** %s\n", firstNonEmpty(summary.StopReason, "execucao encerrada"))
	fmt.Fprintf(&b, "- **Batch Progress:** %s\n", formatBatchProgress(summary.Progress))
	if summary.ReportPath != "" {
		fmt.Fprintf(&b, "- **Report Path:** `%s`\n", summary.ReportPath)
	}
	if summary.LastFailure != nil {
		fmt.Fprintf(&b, "- **Last Failure:** %s\n", renderFailureMessage(summary.LastFailure))
	}
	b.WriteString("\n")

	// Results table — com coluna Papel e sub-linhas de reviewer
	if len(r.Iterations) > 0 {
		b.WriteString("## Results\n\n")
		b.WriteString("| # | Task ID | Title | Papel | Pre-Status | Post-Status | Duration | Exit Code |\n")
		b.WriteString("|---|---------|-------|-------|------------|-------------|----------|-----------|\n")
		for _, it := range r.Iterations {
			role := it.Role
			if role == "" {
				role = "executor"
			}
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %s | %d |\n",
				it.Sequence, it.TaskID, it.Title, role,
				it.PreStatus, it.PostStatus,
				it.Duration.Truncate(time.Second), it.ExitCode)
			// Sub-linha do reviewer logo apos o executor correspondente
			if it.ReviewResult != nil {
				rr := it.ReviewResult
				fmt.Fprintf(&b, "|  | %s |  | reviewer |  |  | %s | %d |\n",
					it.TaskID, rr.Duration.Truncate(time.Second), rr.ExitCode)
			}
		}
		b.WriteString("\n")
	}

	// Resumo por status
	if len(r.Iterations) > 0 {
		var success, skippedCount, failedCount int
		for _, it := range r.Iterations {
			switch {
			// postStatus == "done" e o criterio primario de sucesso.
			// exit code nao-zero (ex: -1 por timeout) e ignorado quando a task
			// foi concluida: o agente pode ter sido morto por SIGKILL apos marcar
			// a task como done, e o resultado observavel e o estado da task.
			case it.PostStatus == "done":
				success++
			case it.PostStatus == "failed" || it.ExitCode != 0:
				failedCount++
			default:
				skippedCount++
			}
		}
		b.WriteString("## Resumo\n\n")
		fmt.Fprintf(&b, "- **Executadas com sucesso:** %d\n", success)
		fmt.Fprintf(&b, "- **Puladas:** %d\n", skippedCount)
		fmt.Fprintf(&b, "- **Falhadas:** %d\n\n", failedCount)
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

	// Iteration details — inclui detalhes de review quando presente
	if len(r.Iterations) > 0 {
		b.WriteString("## Iteration Details\n\n")
		for _, it := range r.Iterations {
			role := it.Role
			if role == "" {
				role = "executor"
			}
			fmt.Fprintf(&b, "### Iteration %d: Task %s — %s\n", it.Sequence, it.TaskID, it.Title)
			fmt.Fprintf(&b, "- **Papel:** %s\n", role)
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
			if it.ReviewResult != nil {
				rr := it.ReviewResult
				b.WriteString("#### Review Result\n")
				fmt.Fprintf(&b, "- **Duration:** %s\n", rr.Duration.Truncate(time.Second))
				fmt.Fprintf(&b, "- **Exit Code:** %d\n", rr.ExitCode)
				if rr.Note != "" {
					fmt.Fprintf(&b, "- **Note:** %s\n", rr.Note)
				}
				if rr.Output != "" {
					output := rr.Output
					if len(output) > maxOutputLen {
						output = output[:maxOutputLen] + "\n... (truncated)"
					}
					b.WriteString("- **Review Output:**\n")
					b.WriteString("  ```\n")
					for _, line := range strings.Split(output, "\n") {
						b.WriteString("  " + line + "\n")
					}
					b.WriteString("  ```\n")
				}
			}
			b.WriteString("\n")
		}
	}

	return []byte(b.String())
}

// modelOrDefault retorna o modelo ou "default" quando o modelo nao esta configurado.
func modelOrDefault(model string) string {
	if model == "" {
		return "default"
	}
	return model
}

func (r *Report) effectiveSummary() FinalSummary {
	summary := r.Summary
	if summary.StopReason == "" {
		summary.StopReason = strings.TrimSpace(r.StopReason)
	}
	if summary.IterationsRun == 0 {
		summary.IterationsRun = len(r.Iterations)
	}
	if summary.Progress.Total == 0 {
		if progress, err := computeBatchProgress(r.FinalTasks); err == nil && progress.Total > 0 {
			summary.Progress = progress
		}
	}
	return summary
}
