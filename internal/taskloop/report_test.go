package taskloop

import (
	"strings"
	"testing"
	"time"
)

func TestReportRender(t *testing.T) {
	report := &Report{
		PRDFolder: "tasks/prd-test",
		Tool:      "claude",
		StartTime: time.Date(2026, 4, 19, 14, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 4, 19, 15, 0, 0, 0, time.UTC),
		StopReason: "todas as tasks completadas ou em estado terminal",
		Iterations: []IterationResult{
			{
				Sequence:    1,
				TaskID:      "1.0",
				Title:       "Setup domain",
				PreStatus:   "pending",
				PostStatus:  "done",
				Duration:    10 * time.Minute,
				ExitCode:    0,
				AgentOutput: "task completed successfully",
			},
			{
				Sequence:    2,
				TaskID:      "2.0",
				Title:       "Implement ports",
				PreStatus:   "pending",
				PostStatus:  "failed",
				Duration:    5 * time.Minute,
				ExitCode:    1,
				AgentOutput: "compilation error",
				Note:        "agente saiu com codigo 1",
			},
		},
		FinalTasks: []TaskEntry{
			{ID: "1.0", Title: "Setup domain", Status: "done"},
			{ID: "2.0", Title: "Implement ports", Status: "failed"},
		},
	}

	rendered := string(report.Render())

	checks := []string{
		"# Task Loop Execution Report",
		"tasks/prd-test",
		"claude",
		"2 ",
		"todas as tasks completadas",
		"| 1 | 1.0 | Setup domain | pending | done |",
		"| 2 | 2.0 | Implement ports | pending | failed |",
		"## Final Task Status",
		"| 1.0 | Setup domain | done |",
		"## Skipped Tasks",
		"agente saiu com codigo 1",
		"## Iteration Details",
		"### Iteration 1: Task 1.0",
		"task completed successfully",
	}

	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Errorf("relatorio nao contem: %q", check)
		}
	}
}

func TestReportRenderEmpty(t *testing.T) {
	report := &Report{
		PRDFolder:  "tasks/prd-empty",
		Tool:       "codex",
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		StopReason: "nenhuma task elegivel",
	}

	rendered := string(report.Render())
	if !strings.Contains(rendered, "nenhuma task elegivel") {
		t.Error("relatorio vazio nao contem stop reason")
	}
	if strings.Contains(rendered, "## Results") {
		t.Error("relatorio vazio nao deveria ter secao Results")
	}
}

func TestReportTruncatesOutput(t *testing.T) {
	longOutput := strings.Repeat("x", 3000)
	report := &Report{
		PRDFolder:  "tasks/prd-test",
		Tool:       "claude",
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		StopReason: "done",
		Iterations: []IterationResult{
			{
				Sequence:    1,
				TaskID:      "1.0",
				Title:       "Test",
				PreStatus:   "pending",
				PostStatus:  "done",
				AgentOutput: longOutput,
			},
		},
	}

	rendered := string(report.Render())
	if strings.Contains(rendered, longOutput) {
		t.Error("output nao foi truncado")
	}
	if !strings.Contains(rendered, "... (truncated)") {
		t.Error("marca de truncamento nao encontrada")
	}
}
