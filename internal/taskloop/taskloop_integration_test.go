//go:build integration

package taskloop

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type integrationScriptInvoker struct {
	scriptPath string
	scenario   string
	taskFile   string
	tasksFile  string
	liveOut    io.Writer
	startHook  func()
}

func (i *integrationScriptInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	return runCmdMonitored(ctx, workDir, i.liveOut, i.startHook, "/bin/sh", i.scriptPath, i.scenario, i.taskFile, i.tasksFile)
}

func (i *integrationScriptInvoker) BinaryName() string { return "/bin/sh" }

func (i *integrationScriptInvoker) SetLiveOutput(w io.Writer) { i.liveOut = w }

func (i *integrationScriptInvoker) SetProcessStartHook(fn func()) { i.startHook = fn }

func TestServiceExecuteIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ui         UIMode
		scenario   string
		wantOut    []string
		wantReport []string
		wantErr    []string
	}{
		{
			name:     "plain done",
			ui:       UIModePlain,
			scenario: "done",
			wantOut: []string{
				"resumo final: stop=limite de iteracoes atingido (1) iteracoes=1 lote=done=1 failed=0 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
				"report=",
			},
			wantReport: []string{
				"- **Batch Progress:** done=1 failed=0 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
				"| 1.0 | Test Task | done |",
			},
		},
		{
			name:     "plain failed",
			ui:       UIModePlain,
			scenario: "failed",
			wantOut: []string{
				"resumo final: stop=limite de iteracoes atingido (1) iteracoes=1 lote=done=0 failed=1 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
			},
			wantReport: []string{
				"- **Batch Progress:** done=0 failed=1 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
				"- **Last Failure:** claude encerrou com falha (exit=1); hint: inspecione stderr, diff local e report da iteracao",
			},
			wantErr: []string{
				"falha final: claude encerrou com falha (exit=1); hint: inspecione stderr, diff local e report da iteracao",
			},
		},
		{
			name:     "plain timeout",
			ui:       UIModePlain,
			scenario: "timeout",
			wantOut: []string{
				"resumo final: stop=limite de iteracoes atingido (1) iteracoes=1 lote=done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
			},
			wantReport: []string{
				"- **Batch Progress:** done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
				"- **Last Failure:** tempo limite da iteracao excedido para claude; hint: revise possivel travamento da CLI ou aumente o timeout configurado",
			},
			wantErr: []string{
				"falha final: tempo limite da iteracao excedido para claude; hint: revise possivel travamento da CLI ou aumente o timeout configurado",
			},
		},
		{
			name:     "plain auth required",
			ui:       UIModePlain,
			scenario: "auth_required",
			wantOut: []string{
				"resumo final: stop=abortado: claude nao esta autenticado iteracoes=1 lote=done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
			},
			wantReport: []string{
				"- **Batch Progress:** done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
				"- **Last Failure:** claude nao esta autenticado; hint: verifique login, sessao local ou credenciais da ferramenta ativa",
			},
			wantErr: []string{
				"falha final: claude nao esta autenticado; hint: verifique login, sessao local ou credenciais da ferramenta ativa",
			},
		},
		{
			name:     "tui done",
			ui:       UIModeTUI,
			scenario: "done",
			wantOut: []string{
				"resumo final: stop=limite de iteracoes atingido (1) iteracoes=1 lote=done=1 failed=0 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
				"relatorio salvo em:",
			},
			wantReport: []string{
				"- **Batch Progress:** done=1 failed=0 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
			},
		},
		{
			name:     "tui failed",
			ui:       UIModeTUI,
			scenario: "failed",
			wantOut: []string{
				"resumo final: stop=limite de iteracoes atingido (1) iteracoes=1 lote=done=0 failed=1 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
				"relatorio salvo em:",
			},
			wantReport: []string{
				"- **Batch Progress:** done=0 failed=1 blocked=0 needs_input=0 pending=0 in_progress=0 total=1",
				"- **Last Failure:** claude encerrou com falha (exit=1); hint: inspecione stderr, diff local e report da iteracao",
			},
			wantErr: []string{
				"falha final: claude encerrou com falha (exit=1); hint: inspecione stderr, diff local e report da iteracao",
			},
		},
		{
			name:     "tui timeout",
			ui:       UIModeTUI,
			scenario: "timeout",
			wantOut: []string{
				"resumo final: stop=limite de iteracoes atingido (1) iteracoes=1 lote=done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
				"relatorio salvo em:",
			},
			wantReport: []string{
				"- **Batch Progress:** done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
				"- **Last Failure:** tempo limite da iteracao excedido para claude; hint: revise possivel travamento da CLI ou aumente o timeout configurado",
			},
			wantErr: []string{
				"falha final: tempo limite da iteracao excedido para claude; hint: revise possivel travamento da CLI ou aumente o timeout configurado",
			},
		},
		{
			name:     "tui auth required",
			ui:       UIModeTUI,
			scenario: "auth_required",
			wantOut: []string{
				"resumo final: stop=abortado: claude nao esta autenticado iteracoes=1 lote=done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
				"relatorio salvo em:",
			},
			wantReport: []string{
				"- **Batch Progress:** done=0 failed=0 blocked=0 needs_input=0 pending=1 in_progress=0 total=1",
				"- **Last Failure:** claude nao esta autenticado; hint: verifique login, sessao local ou credenciais da ferramenta ativa",
			},
			wantErr: []string{
				"falha final: claude nao esta autenticado; hint: verifique login, sessao local ou credenciais da ferramenta ativa",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prd, reportPath, taskFile, tasksFile, scriptPath := setupIntegrationTaskLoopFixture(t)
			printer, outputBuf := newCapturePrinter()
			var tuiProgram *fakeBubbleTeaProgram
			svc := NewService(taskfs.NewOSFileSystem(), printer)
			if tt.ui == UIModeTUI {
				presenter := NewBubbleTeaPresenter(
					TerminalCapabilities{Interactive: true, Width: 96, Height: 24, SupportsAltScreen: true},
					func() SessionSnapshot { return SessionSnapshot{} },
				)
				tuiProgram = newFakeBubbleTeaProgram()
				presenter.programFactory = func(model tea.Model, options ...tea.ProgramOption) bubbleTeaProgram {
					return tuiProgram
				}
				svc = NewServiceWithObserver(taskfs.NewOSFileSystem(), printer, presenter)
			}
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(tool string) (AgentInvoker, error) {
				return &integrationScriptInvoker{
					scriptPath: scriptPath,
					scenario:   tt.scenario,
					taskFile:   taskFile,
					tasksFile:  tasksFile,
				}, nil
			}

			opts := Options{
				PRDFolder:       prd,
				Tool:            "claude",
				MaxIterations:   1,
				Timeout:         150 * time.Millisecond,
				ReportPath:      reportPath,
				RequestedUIMode: tt.ui,
				EffectiveUIMode: tt.ui,
				TerminalCapabilities: TerminalCapabilities{
					Interactive:       tt.ui == UIModeTUI,
					Width:             96,
					Height:            24,
					SupportsAltScreen: tt.ui == UIModeTUI,
				},
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("Execute() erro inesperado: %v", err)
			}

			reportData, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("os.ReadFile(%s): %v", reportPath, err)
			}
			report := string(reportData)
			for _, want := range tt.wantReport {
				if !strings.Contains(report, want) {
					t.Fatalf("relatorio nao contem %q\noutput:\n%s", want, report)
				}
			}

			out := outputBuf.String()
			for _, want := range tt.wantOut {
				if !strings.Contains(out, want) {
					t.Fatalf("saida nao contem %q\noutput:\n%s", want, out)
				}
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(out, want) {
					t.Fatalf("saida nao contem %q\noutput:\n%s", want, out)
				}
			}
			if tuiProgram != nil {
				msgs := tuiProgram.Messages()
				if len(msgs) == 0 {
					t.Fatal("tui nao recebeu mensagens")
				}
				summaryMsg, ok := msgs[len(msgs)-1].(bubbleTeaSummaryMsg)
				if !ok {
					t.Fatalf("ultima mensagem da tui = %T, esperado bubbleTeaSummaryMsg", msgs[len(msgs)-1])
				}
				if summaryMsg.summary.Progress != reportSummaryProgress(t, report) {
					t.Fatalf("summary da tui diverge do report: got=%+v want=%+v", summaryMsg.summary.Progress, reportSummaryProgress(t, report))
				}
			}
		})
	}
}

func setupIntegrationTaskLoopFixture(t *testing.T) (string, string, string, string, string) {
	t.Helper()

	baseDir := t.TempDir()
	prdDir := filepath.Join(baseDir, "tasks", "prd-integration")
	if err := os.MkdirAll(prdDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%s): %v", prdDir, err)
	}

	writeIntegrationFile(t, filepath.Join(baseDir, "AGENTS.md"), "# Agents\n")
	writeIntegrationFile(t, filepath.Join(baseDir, "go.mod"), "module example.com/integration\n\ngo 1.26.2\n")
	writeIntegrationFile(t, filepath.Join(prdDir, "prd.md"), "# PRD\n")
	writeIntegrationFile(t, filepath.Join(prdDir, "techspec.md"), "# TechSpec\n")

	taskFile := filepath.Join(prdDir, "task-1.0-test.md")
	tasksFile := filepath.Join(prdDir, "tasks.md")
	writeIntegrationFile(t, taskFile, "**Status:** pending\n")
	writeIntegrationFile(t, tasksFile, "| 1.0 | Test Task | pending | - | Nao |\n")

	scriptPath := filepath.Join(baseDir, "agent-stub.sh")
	writeIntegrationFile(t, scriptPath, integrationAgentStubScript())
	if err := os.Chmod(scriptPath, 0o755); err != nil {
		t.Fatalf("os.Chmod(%s): %v", scriptPath, err)
	}

	return prdDir, filepath.Join(prdDir, "report.md"), taskFile, tasksFile, scriptPath
}

func writeIntegrationFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%s): %v", path, err)
	}
}

func integrationAgentStubScript() string {
	return strings.TrimSpace(`
#!/bin/sh
scenario="$1"
task_file="$2"
tasks_file="$3"

case "$scenario" in
  done)
    printf '%s\n' '**Status:** done' > "$task_file"
    printf '%s\n' '| 1.0 | Test Task | done | - | Nao |' > "$tasks_file"
    printf '%s\n' 'task concluida'
    exit 0
    ;;
  failed)
    printf '%s\n' '**Status:** failed' > "$task_file"
    printf '%s\n' '| 1.0 | Test Task | failed | - | Nao |' > "$tasks_file"
    printf '%s\n' 'compilation failed' >&2
    exit 1
    ;;
  auth_required)
    printf '%s\n' '**Status:** pending' > "$task_file"
    printf '%s\n' '| 1.0 | Test Task | pending | - | Nao |' > "$tasks_file"
    printf '%s\n' 'Not logged in. Please run /login' >&2
    exit 1
    ;;
  timeout)
    sleep 1
    exit 0
    ;;
  *)
    printf '%s\n' "cenario desconhecido: $scenario" >&2
    exit 2
    ;;
esac
`) + "\n"
}

func reportSummaryProgress(t *testing.T, report string) BatchProgress {
	t.Helper()

	progress := BatchProgress{}
	_, err := fmt.Sscanf(
		extractReportBatchProgressLine(t, report),
		"- **Batch Progress:** done=%d failed=%d blocked=%d needs_input=%d pending=%d in_progress=%d total=%d",
		&progress.Done,
		&progress.Failed,
		&progress.Blocked,
		&progress.NeedsInput,
		&progress.Pending,
		&progress.InProgress,
		&progress.Total,
	)
	if err != nil {
		t.Fatalf("nao foi possivel ler progresso do report: %v", err)
	}
	return progress
}

func extractReportBatchProgressLine(t *testing.T, report string) string {
	t.Helper()

	for _, line := range strings.Split(report, "\n") {
		if strings.HasPrefix(line, "- **Batch Progress:** ") {
			return line
		}
	}
	t.Fatalf("linha de progresso agregado nao encontrada no report\noutput:\n%s", report)
	return ""
}
