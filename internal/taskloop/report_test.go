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
			// Regressao BUG-2: exit=-1 (SIGKILL por timeout) com task done deve contar como sucesso.
			// O agente completou o trabalho antes de ser morto; o criterio primario e o estado da task.
			name: "exit code -1 e post status done — conta como sucesso (regressao BUG-2)",
			iterations: []IterationResult{
				{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "done", ExitCode: -1},
			},
			wantIn: []string{
				"- **Executadas com sucesso:** 1",
				"- **Puladas:** 0",
				"- **Falhadas:** 0",
			},
		},
		{
			// Regressao BUG-2: exit!=0 (qualquer valor) com task done deve contar como sucesso.
			name: "exit code nao zero e post status done — conta como sucesso",
			iterations: []IterationResult{
				{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "done", ExitCode: 1},
			},
			wantIn: []string{
				"- **Executadas com sucesso:** 1",
				"- **Falhadas:** 0",
			},
		},
		{
			// exit!=0 com status nao-done e nao-failed (ex: in_progress) deve contar como falhada.
			name: "exit code nao zero e post status in_progress — conta como falhada",
			iterations: []IterationResult{
				{Sequence: 1, TaskID: "1.0", Title: "A", PreStatus: "pending", PostStatus: "in_progress", ExitCode: 1},
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

// TestReportRenderSimples verifica regressao: modo simples com Mode="simples"
// deve conter "Modo: simples", "Tool:", tabela sem coluna Papel, e toda estrutura atual.
func TestReportRenderSimples(t *testing.T) {
	tests := []struct {
		name      string
		report    *Report
		wantIn    []string
		wantNotIn []string
	}{
		{
			name: "modo simples — cabecalho com Modo e Tool",
			report: &Report{
				Mode:       "simples",
				PRDFolder:  "tasks/prd-minha-feature",
				Tool:       "claude",
				StartTime:  time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC),
				EndTime:    time.Date(2026, 4, 19, 10, 30, 0, 0, time.UTC),
				StopReason: "concluido",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "Setup",
						PreStatus:  "pending",
						PostStatus: "done",
						Duration:   5 * time.Minute,
						ExitCode:   0,
					},
				},
			},
			wantIn: []string{
				"**Modo:** simples",
				"**Tool:** claude",
				"tasks/prd-minha-feature",
				"| # | Task ID | Title | Pre-Status | Post-Status | Duration | Exit Code |",
				"| 1 | 1.0 | Setup | pending | done |",
			},
			wantNotIn: []string{
				"**Executor:**",
				"**Reviewer:**",
				"| Papel |",
				"avancado",
			},
		},
		{
			name: "modo simples — tabela nao tem coluna Papel",
			report: &Report{
				Mode:       "simples",
				PRDFolder:  "tasks/prd-test",
				Tool:       "codex",
				StartTime:  time.Now(),
				EndTime:    time.Now(),
				StopReason: "done",
				Iterations: []IterationResult{
					{Sequence: 1, TaskID: "2.0", Title: "Impl", PreStatus: "pending", PostStatus: "done", ExitCode: 0},
				},
			},
			wantIn: []string{
				"**Modo:** simples",
				"**Tool:** codex",
				"| # | Task ID | Title | Pre-Status | Post-Status | Duration | Exit Code |",
			},
			wantNotIn: []string{
				"| Papel |",
				"reviewer",
			},
		},
		{
			name: "modo vazio (backward compat) — sem linha Modo no cabecalho",
			report: &Report{
				PRDFolder:  "tasks/prd-test",
				Tool:       "claude",
				StartTime:  time.Now(),
				EndTime:    time.Now(),
				StopReason: "done",
			},
			wantIn: []string{
				"**Tool:** claude",
			},
			wantNotIn: []string{
				"**Modo:**",
				"**Executor:**",
				"**Reviewer:**",
			},
		},
		{
			name: "modo simples — ReviewResult em IterationResult nao gera sub-linha na tabela",
			report: &Report{
				Mode:       "simples",
				PRDFolder:  "tasks/prd-test",
				Tool:       "claude",
				StartTime:  time.Now(),
				EndTime:    time.Now(),
				StopReason: "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "A",
						PreStatus:  "pending",
						PostStatus: "done",
						ExitCode:   0,
						ReviewResult: &ReviewResult{
							Duration: 2 * time.Second,
							ExitCode: 0,
							Output:   "aprovado",
						},
					},
				},
			},
			wantIn: []string{
				"| 1 | 1.0 | A | pending | done |",
			},
			wantNotIn: []string{
				"reviewer",
				"| Papel |",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rendered := string(tc.report.Render())
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

// TestReportRenderAvancado verifica que o modo avancado inclui perfis no cabecalho,
// coluna Papel na tabela, e sub-linhas de reviewer quando ReviewResult != nil.
func TestReportRenderAvancado(t *testing.T) {
	execProfile, _ := NewExecutionProfile("executor", "claude", "claude-sonnet-4-6")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "gpt-5.4")

	tests := []struct {
		name      string
		report    *Report
		wantIn    []string
		wantNotIn []string
	}{
		{
			name: "modo avancado — cabecalho com perfis executor e reviewer",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-minha-feature",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC),
				EndTime:         time.Date(2026, 4, 19, 10, 30, 0, 0, time.UTC),
				StopReason:      "concluido",
			},
			wantIn: []string{
				"**Modo:** avancado",
				"**Executor:** claude / anthropic / claude-sonnet-4-6",
				"**Reviewer:** codex / openai / gpt-5.4",
				"tasks/prd-minha-feature",
			},
			wantNotIn: []string{
				"**Tool:**",
				"**Modo:** simples",
			},
		},
		{
			name: "modo avancado — tabela com coluna Papel",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "Setup",
						PreStatus:  "pending",
						PostStatus: "done",
						Duration:   10 * time.Second,
						ExitCode:   0,
						Role:       "executor",
					},
				},
			},
			wantIn: []string{
				"| # | Task ID | Title | Papel | Pre-Status | Post-Status | Duration | Exit Code |",
				"| 1 | 1.0 | Setup | executor | pending | done |",
			},
		},
		{
			name: "modo avancado — Role vazio assume executor",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "A",
						PreStatus:  "pending",
						PostStatus: "done",
						ExitCode:   0,
					},
				},
			},
			wantIn: []string{
				"| 1 | 1.0 | A | executor | pending | done |",
			},
		},
		{
			name: "modo avancado — reviewer nil mostra 'nao configurado'",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: nil,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
			},
			wantIn: []string{
				"**Reviewer:** nao configurado",
			},
		},
		{
			name: "modo avancado — modelo vazio exibe 'default'",
			report: func() *Report {
				ep, _ := NewExecutionProfile("executor", "claude", "")
				return &Report{
					Mode:            "avancado",
					PRDFolder:       "tasks/prd-test",
					ExecutorProfile: &ep,
					StartTime:       time.Now(),
					EndTime:         time.Now(),
					StopReason:      "done",
				}
			}(),
			wantIn: []string{
				"**Executor:** claude / anthropic / default",
			},
		},
		{
			name: "modo avancado — sub-linha reviewer presente quando ReviewResult != nil",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "A",
						PreStatus:  "pending",
						PostStatus: "done",
						Duration:   5 * time.Second,
						ExitCode:   0,
						Role:       "executor",
						ReviewResult: &ReviewResult{
							Duration: 3 * time.Second,
							ExitCode: 0,
							Output:   "revisao aprovada",
							Note:     "",
						},
					},
				},
			},
			wantIn: []string{
				"| 1 | 1.0 | A | executor | pending | done |",
				"reviewer",
				"3s",
				"#### Review Result",
				"revisao aprovada",
			},
		},
		{
			name: "modo avancado — sub-linha reviewer ausente quando ReviewResult == nil",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:     1,
						TaskID:       "1.0",
						Title:        "A",
						PreStatus:    "pending",
						PostStatus:   "done",
						Duration:     5 * time.Second,
						ExitCode:     0,
						Role:         "executor",
						ReviewResult: nil,
					},
				},
			},
			wantIn: []string{
				"| 1 | 1.0 | A | executor | pending | done |",
			},
			wantNotIn: []string{
				"#### Review Result",
			},
		},
		{
			name: "modo avancado — ReviewResult com exit code != 0 e note",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "A",
						PreStatus:  "pending",
						PostStatus: "done",
						Duration:   5 * time.Second,
						ExitCode:   0,
						Role:       "executor",
						ReviewResult: &ReviewResult{
							Duration: 4 * time.Second,
							ExitCode: 1,
							Output:   "problemas encontrados",
							Note:     "reviewer reportou problemas criticos",
						},
					},
				},
			},
			wantIn: []string{
				"#### Review Result",
				"reviewer reportou problemas criticos",
				"problemas encontrados",
			},
		},
		{
			name: "modo avancado — multiplas iteracoes com RF-13: iteracao conta apenas executor",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence: 1, TaskID: "1.0", Title: "A",
						PreStatus: "pending", PostStatus: "done", ExitCode: 0,
						ReviewResult: &ReviewResult{Duration: 2 * time.Second, ExitCode: 0},
					},
					{
						Sequence: 2, TaskID: "2.0", Title: "B",
						PreStatus: "pending", PostStatus: "done", ExitCode: 0,
						ReviewResult: &ReviewResult{Duration: 3 * time.Second, ExitCode: 0},
					},
				},
			},
			wantIn: []string{
				"- **Iterations:** 2",
				"| 1 | 1.0 | A | executor |",
				"| 2 | 2.0 | B | executor |",
			},
		},
		{
			name: "modo avancado — secao Results nao aparece sem iteracoes",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "nenhuma task elegivel",
			},
			wantIn: []string{
				"**Modo:** avancado",
				"nenhuma task elegivel",
			},
			wantNotIn: []string{
				"## Results",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rendered := string(tc.report.Render())
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

// TestReportRenderBugfixOutputTruncation verifica que o output do bugfix
// e truncado quando excede maxOutputLen (2000 chars).
func TestReportRenderBugfixOutputTruncation(t *testing.T) {
	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")
	longOutput := strings.Repeat("x", 3000)

	report := &Report{
		Mode:            "avancado",
		PRDFolder:       "tasks/prd-test",
		ExecutorProfile: &execProfile,
		ReviewerProfile: &revProfile,
		StartTime:       time.Now(),
		EndTime:         time.Now(),
		StopReason:      "done",
		Iterations: []IterationResult{
			{
				Sequence:   1,
				TaskID:     "1.0",
				Title:      "A",
				PreStatus:  "pending",
				PostStatus: "done",
				Duration:   5 * time.Second,
				ExitCode:   0,
				ReviewResult: &ReviewResult{
					Duration: 3 * time.Second,
					ExitCode: 1,
					Output:   "issues",
				},
				BugfixResult: &BugfixResult{
					Duration: 8 * time.Second,
					ExitCode: 0,
					Output:   longOutput,
				},
			},
		},
	}

	rendered := string(report.Render())
	if strings.Contains(rendered, longOutput) {
		t.Error("bugfix output nao foi truncado")
	}
	if !strings.Contains(rendered, "... (truncated)") {
		t.Error("marca de truncamento nao encontrada no bugfix output")
	}
}

// TestReportRenderSimplesModeIgnoresBugfixResult verifica que o modo simples
// nao gera sub-linhas ou secoes de bugfix mesmo quando BugfixResult esta presente.
func TestReportRenderSimplesModeIgnoresBugfixResult(t *testing.T) {
	report := &Report{
		Mode:       "simples",
		PRDFolder:  "tasks/prd-test",
		Tool:       "claude",
		StartTime:  time.Now(),
		EndTime:    time.Now(),
		StopReason: "done",
		Iterations: []IterationResult{
			{
				Sequence:   1,
				TaskID:     "1.0",
				Title:      "A",
				PreStatus:  "pending",
				PostStatus: "done",
				ExitCode:   0,
				ReviewResult: &ReviewResult{
					Duration: 3 * time.Second,
					ExitCode: 1,
					Output:   "issues",
				},
				BugfixResult: &BugfixResult{
					Duration: 5 * time.Second,
					ExitCode: 0,
					Output:   "fixed",
				},
			},
		},
	}

	rendered := string(report.Render())
	if strings.Contains(rendered, "#### Bugfix Result") {
		t.Error("modo simples nao deveria conter secao Bugfix Result")
	}
	if strings.Contains(rendered, "| bugfix |") {
		t.Error("modo simples nao deveria conter sub-linha bugfix na tabela")
	}
	if strings.Contains(rendered, "#### Review Result") {
		t.Error("modo simples nao deveria conter secao Review Result")
	}
}

// TestReportRenderAvancadoBugfix verifica que o modo avancado inclui sub-linhas
// e detalhes de bugfix quando BugfixResult != nil.
func TestReportRenderAvancadoBugfix(t *testing.T) {
	execProfile, _ := NewExecutionProfile("executor", "claude", "claude-sonnet-4-6")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "gpt-5.4")

	tests := []struct {
		name      string
		report    *Report
		wantIn    []string
		wantNotIn []string
	}{
		{
			name: "bugfix presente apos reviewer com exit != 0",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "A",
						PreStatus:  "pending",
						PostStatus: "done",
						Duration:   5 * time.Second,
						ExitCode:   0,
						Role:       "executor",
						ReviewResult: &ReviewResult{
							Duration: 3 * time.Second,
							ExitCode: 1,
							Output:   "critical issues found",
							Note:     "reviewer reportou problemas criticos",
						},
						BugfixResult: &BugfixResult{
							Duration: 8 * time.Second,
							ExitCode: 0,
							Output:   "bugfix applied successfully",
						},
					},
				},
			},
			wantIn: []string{
				"| bugfix |",
				"8s",
				"#### Bugfix Result",
				"bugfix applied successfully",
				"#### Review Result",
				"reviewer reportou problemas criticos",
			},
		},
		{
			name: "bugfix com exit != 0 captura note",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "A",
						PreStatus:  "pending",
						PostStatus: "done",
						Duration:   5 * time.Second,
						ExitCode:   0,
						ReviewResult: &ReviewResult{
							Duration: 3 * time.Second,
							ExitCode: 1,
							Output:   "issues",
							Note:     "reviewer reportou problemas criticos",
						},
						BugfixResult: &BugfixResult{
							Duration: 4 * time.Second,
							ExitCode: 1,
							Output:   "could not fix",
							Note:     "bugfix nao conseguiu corrigir todos os achados",
						},
					},
				},
			},
			wantIn: []string{
				"#### Bugfix Result",
				"bugfix nao conseguiu corrigir todos os achados",
				"could not fix",
			},
		},
		{
			name: "sem bugfix — secao Bugfix Result ausente",
			report: &Report{
				Mode:            "avancado",
				PRDFolder:       "tasks/prd-test",
				ExecutorProfile: &execProfile,
				ReviewerProfile: &revProfile,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				StopReason:      "done",
				Iterations: []IterationResult{
					{
						Sequence:   1,
						TaskID:     "1.0",
						Title:      "A",
						PreStatus:  "pending",
						PostStatus: "done",
						Duration:   5 * time.Second,
						ExitCode:   0,
						ReviewResult: &ReviewResult{
							Duration: 2 * time.Second,
							ExitCode: 0,
							Output:   "approved",
						},
					},
				},
			},
			wantIn: []string{
				"#### Review Result",
			},
			wantNotIn: []string{
				"#### Bugfix Result",
				"| bugfix |",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rendered := string(tc.report.Render())
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
