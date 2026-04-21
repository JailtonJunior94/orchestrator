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

func TestReportRenderResumoSection(t *testing.T) {
	tests := []struct {
		name       string
		iterations []IterationResult
		wantIn     []string
		wantNotIn  []string
	}{
		{
			name: "mix de sucesso, falha e skip",
			iterations: []IterationResult{
				{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "done", ExitCode: 0},
				{Sequence: 2, TaskID: "2.0", Title: "B", PreStatus: "pending", PostStatus: "failed", ExitCode: 1},
				{Sequence: 3, TaskID: "3.0", Title: "C", PreStatus: "pending", PostStatus: "in_progress", ExitCode: 0},
			},
			wantIn: []string{
				"## Resumo",
				"- **Executadas com sucesso:** 1",
				"- **Puladas:** 1",
				"- **Falhadas:** 1",
			},
		},
		{
			name:      "sem iteracoes — secao Resumo nao aparece",
			iterations: nil,
			wantNotIn: []string{"## Resumo"},
		},
		{
			name: "todas com sucesso — falhadas e puladas zero",
			iterations: []IterationResult{
				{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "done", ExitCode: 0},
				{Sequence: 2, TaskID: "2.0", Title: "B", PreStatus: "pending", PostStatus: "done", ExitCode: 0},
			},
			wantIn: []string{
				"## Resumo",
				"- **Executadas com sucesso:** 2",
				"- **Puladas:** 0",
				"- **Falhadas:** 0",
			},
		},
		{
			name: "exit code nao zero mas post status nao failed — conta como falhada",
			iterations: []IterationResult{
				{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "done", ExitCode: 1},
			},
			wantIn: []string{
				"- **Executadas com sucesso:** 0",
				"- **Falhadas:** 1",
			},
		},
		{
			name: "exit code zero mas post status failed — conta como falhada",
			iterations: []IterationResult{
				{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "failed", ExitCode: 0},
			},
			wantIn: []string{
				"- **Executadas com sucesso:** 0",
				"- **Falhadas:** 1",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report := &Report{
				PRDFolder:  "tasks/prd-test",
				Tool:       "claude",
				StartTime:  time.Now(),
				EndTime:    time.Now(),
				StopReason: "done",
				Iterations: tc.iterations,
			}
			rendered := string(report.Render())
			for _, want := range tc.wantIn {
				if !strings.Contains(rendered, want) {
					t.Errorf("esperado %q no relatorio, nao encontrado\noutput:\n%s", want, rendered)
				}
			}
			for _, notWant := range tc.wantNotIn {
				if strings.Contains(rendered, notWant) {
					t.Errorf("nao esperado %q no relatorio, mas foi encontrado\noutput:\n%s", notWant, rendered)
				}
			}
		})
	}
}

func TestReportRenderResumoOrdem(t *testing.T) {
	report := &Report{
		PRDFolder:  "tasks/prd-test",
		Tool:       "claude",
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		StopReason: "done",
		Iterations: []IterationResult{
			{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "done", ExitCode: 0},
		},
		FinalTasks: []TaskEntry{
			{ID: "1.0", Title: "A", Status: "done"},
		},
	}
	rendered := string(report.Render())

	posResults := strings.Index(rendered, "## Results")
	posResumo := strings.Index(rendered, "## Resumo")
	posFinal := strings.Index(rendered, "## Final Task Status")

	if posResults == -1 || posResumo == -1 || posFinal == -1 {
		t.Fatalf("uma ou mais secoes nao encontradas: Results=%d Resumo=%d FinalTask=%d", posResults, posResumo, posFinal)
	}
	if posResults >= posResumo || posResumo >= posFinal {
		t.Errorf("ordem incorreta: Results(%d) < Resumo(%d) < FinalTaskStatus(%d)", posResults, posResumo, posFinal)
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
