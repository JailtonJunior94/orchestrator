//go:build integration

package taskloop

// TestParidadeIntegration — testes de integracao com mock binaries (build tag: integration).
//
// Cada subteste cria um PRD folder real em t.TempDir(), instala um mock binary no PATH
// e executa Service.Execute com NewAgentInvoker (subprocesso real, nao callbackInvoker).
//
// O objetivo e verificar que o estado final do filesystem (task files, tasks.md, report)
// e identico para todas as 4 ferramentas no mesmo cenario — paridade semantica observavel.
//
// # Limitacoes conhecidas dos mock binaries vs CLIs reais (RF-10)
//
// 1. Flags ignoradas: o mock recebe --dangerously-skip-permissions, --bare, --yolo, etc.
//    e as ignora. As CLIs reais podem falhar com flags invalidas ou ter comportamento
//    diferente baseado nelas.
//
// 2. Output de stdout/stderr: o mock nao produz output conversacional. Testes de
//    integracao com CLIs reais exibiriam progresso, raciocinio e confirmacoes.
//
// 3. Escrita de tasks.md: o mock apenas atualiza o task file. CLIs reais geralmente
//    tambem atualizam tasks.md e geram execution reports — o loop lida com ambos os casos.
//
// 4. Autenticacao: o mock simula erros de auth via stderr/exit code. CLIs reais
//    verificam tokens, sessoes e permissoes antes de processar o prompt.
//
// 5. Subagentes e recursao: CLIs reais podem invocar subagentes ou ferramentas internas.
//    O mock executa atomicamente sem recursao.
//
// 6. Timeout vs flush: o mock de timeout dorme e e morto por SIGKILL. CLIs reais
//    podem fazer flush parcial antes de serem mortas, resultando em estado parcialmente
//    atualizado. Os testes de timeout verificam apenas que o loop nao trava.
//
// 7. claudiney wrapper: o mock claudiney recebe args sem --bare e sem
//    --dangerously-skip-permissions (diferente do mock claude). O comportamento
//    do mock e identico, mas o real claudiney tem semantica propria de wrapper.

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	osfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// successMockScript e um script Python3 portavel que:
// 1. Le o caminho do task file do prompt passado como argumento
// 2. Substitui "**Status:** <status>" por "**Status:** done" no task file
// 3. Retorna exit 0
//
// Funciona com todos os adapters: claude (-p <prompt>), codex (posicional), gemini (-p), copilot (-p).
const successMockScript = `#!/usr/bin/env python3
import sys, re, os

text = ' '.join(sys.argv[1:])
m = re.search(r'implementar a task (\S+\.md)', text)
if m:
    path = m.group(1)
    if os.path.exists(path):
        with open(path, 'r') as f:
            content = f.read()
        content = re.sub(r'\*\*Status:\*\* \w+', '**Status:** done', content, count=1)
        with open(path, 'w') as f:
            f.write(content)
sys.exit(0)
`

// failureMockScript simula falha de invocacao: exit 1 sem alterar nenhum arquivo.
// Cenario: agente falhou ao processar o prompt (ex: erro de contexto, limite de tokens).
const failureMockScript = `#!/usr/bin/env python3
import sys
sys.exit(1)
`

// authErrorMockScript simula erro de autenticacao: imprime "not authenticated" no stderr e exit 1.
// O loop deve detectar isAuthError e abortar com StopReason contendo "autenticado".
const authErrorMockScript = `#!/usr/bin/env python3
import sys
print("not authenticated", file=sys.stderr)
sys.exit(1)
`

// timeoutMockScript simula processo que demora e excede o timeout.
// O loop mata o processo via SIGKILL (process group) e registra exit code -1.
const timeoutMockScript = `#!/usr/bin/env python3
import sys, time
time.sleep(60)
sys.exit(0)
`

// integrationPrinter descarta toda saida durante os testes de integracao.
func integrationPrinter() *output.Printer {
	return &output.Printer{Out: io.Discard, Err: io.Discard}
}

// setupIntegrationPRD cria um PRD folder real em tmpBase com a estrutura minima:
// - AGENTS.md no pai (para resolveWorkDir encontrar a raiz)
// - tasks.md, prd.md, techspec.md e task-1.0-feature.md no prdDir
//
// Retorna o caminho absoluto do prdDir criado.
func setupIntegrationPRD(t *testing.T, tmpBase string) string {
	t.Helper()

	mustWriteIntegrationFile(t, filepath.Join(tmpBase, "AGENTS.md"), "# Agents\n")

	prdDir := filepath.Join(tmpBase, "tasks", "prd-integration")
	if err := os.MkdirAll(prdDir, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", prdDir, err)
	}

	mustWriteIntegrationFile(t, filepath.Join(prdDir, "prd.md"), "# PRD\n")
	mustWriteIntegrationFile(t, filepath.Join(prdDir, "techspec.md"), "# TechSpec\n")
	mustWriteIntegrationFile(t, filepath.Join(prdDir, "task-1.0-feature.md"), "**Status:** pending\n")
	mustWriteIntegrationFile(t, filepath.Join(prdDir, "tasks.md"),
		"| 1.0 | Feature Implementation | pending | — | Nao |\n")

	return prdDir
}

func mustWriteIntegrationFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// installMockBinaries cria os mock binaries em binDir e adiciona binDir ao PATH do teste.
// Restaura o PATH original ao encerrar o subteste via t.Cleanup (via t.Setenv).
func installMockBinaries(t *testing.T, binDir string, names []string, script string) {
	t.Helper()
	for _, name := range names {
		binPath := filepath.Join(binDir, name)
		if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
			t.Fatalf("criar mock binary %s: %v", name, err)
		}
		if err := os.Chmod(binPath, 0o755); err != nil {
			t.Fatalf("chmod mock binary %s: %v", name, err)
		}
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// runIntegrationExecute cria um Service com OSFileSystem e NewAgentInvoker real,
// executa Service.Execute e retorna o erro resultante.
// Nota: binaryChecker = nil para usar CheckAgentBinary real (mock esta no PATH).
func runIntegrationExecute(t *testing.T, prdDir, tool, reportPath string, timeout time.Duration) error {
	t.Helper()
	fsys := osfs.NewOSFileSystem()
	svc := NewService(fsys, integrationPrinter())
	svc.invokerFactory = NewAgentInvoker
	// binaryChecker = nil → usa CheckAgentBinary real; mock esta no PATH via t.Setenv

	opts := Options{
		PRDFolder:     prdDir,
		Tool:          tool,
		MaxIterations: 3,
		Timeout:       timeout,
		ReportPath:    reportPath,
	}
	return svc.Execute(opts)
}

// TestParidadeIntegration verifica paridade semantica entre ferramentas com mock binaries reais.
//
// Cenarios:
//   - P-INT-1 (sucesso): mock atualiza task file para "done" e retorna exit 0.
//     Verificacoes: task file="done", report contem "pending -> done", stop reason correto.
//   - P-INT-2 (falha): mock retorna exit 1 sem alterar arquivos.
//     Verificacoes: task file permanece "pending", report registra skip.
//   - P-INT-3 (auth error): mock imprime "not authenticated" + exit 1.
//     Verificacoes: loop abortado, stop reason contem "autenticado".
//
// Para cada cenario, o mesmo comportamento e esperado para todas as ferramentas:
// claude (binary claude), claude (binary claudiney), codex, gemini, copilot.
func TestParidadeIntegration(t *testing.T) {
	type toolCase struct {
		tool     string   // ferramenta para Options.Tool
		binaries []string // binaries a criar no mock dir
		label    string   // label para o nome do subteste
	}

	toolCases := []toolCase{
		{tool: "claude", binaries: []string{"claude"}, label: "claude"},
		{tool: "claude", binaries: []string{"claudiney"}, label: "claude-claudiney"},
		{tool: "codex", binaries: []string{"codex"}, label: "codex"},
		{tool: "gemini", binaries: []string{"gemini"}, label: "gemini"},
		{tool: "copilot", binaries: []string{"copilot"}, label: "copilot"},
	}

	type scenarioCase struct {
		name   string
		script string
		// verify recebe: prdDir, erro de Execute, label da ferramenta
		verify func(t *testing.T, prdDir string, execErr error, toolLabel string)
	}

	scenarios := []scenarioCase{
		{
			name:   "P-INT-1-sucesso",
			script: successMockScript,
			verify: func(t *testing.T, prdDir string, execErr error, toolLabel string) {
				t.Helper()
				if execErr != nil {
					t.Fatalf("[%s] Execute retornou erro inesperado: %v", toolLabel, execErr)
				}

				// Verificar task file atualizado para "done"
				taskFile := filepath.Join(prdDir, "task-1.0-feature.md")
				data, err := os.ReadFile(taskFile)
				if err != nil {
					t.Fatalf("[%s] task file nao encontrado: %v", toolLabel, err)
				}
				if !strings.Contains(string(data), "**Status:** done") {
					t.Errorf("[%s] task file deveria conter '**Status:** done', conteudo atual:\n%s",
						toolLabel, data)
				}

				// Verificar report gerado
				reportPath := filepath.Join(prdDir, "report.md")
				reportData, err := os.ReadFile(reportPath)
				if err != nil {
					t.Fatalf("[%s] report nao encontrado: %v", toolLabel, err)
				}
				reportStr := string(reportData)

				if !strings.Contains(reportStr, "pending -> done") {
					t.Errorf("[%s] report deveria conter 'pending -> done':\n%s", toolLabel, reportStr)
				}

				if !strings.Contains(reportStr, "todas as tasks") {
					t.Errorf("[%s] report deveria conter stop reason de conclusao, got:\n%s",
						toolLabel, reportStr)
				}
			},
		},
		{
			name:   "P-INT-2-falha-exit1",
			script: failureMockScript,
			verify: func(t *testing.T, prdDir string, execErr error, toolLabel string) {
				t.Helper()
				if execErr != nil {
					t.Fatalf("[%s] Execute retornou erro inesperado: %v", toolLabel, execErr)
				}

				// Task file deve permanecer "pending" (mock nao atualizou)
				taskFile := filepath.Join(prdDir, "task-1.0-feature.md")
				data, err := os.ReadFile(taskFile)
				if err != nil {
					t.Fatalf("[%s] task file nao encontrado: %v", toolLabel, err)
				}
				if strings.Contains(string(data), "**Status:** done") {
					t.Errorf("[%s] task file nao deveria ter sido atualizado para 'done' apos falha, got:\n%s",
						toolLabel, data)
				}

				// Report deve registrar a iteracao com exit code != 0
				reportPath := filepath.Join(prdDir, "report.md")
				reportData, err := os.ReadFile(reportPath)
				if err != nil {
					t.Fatalf("[%s] report nao encontrado: %v", toolLabel, err)
				}
				reportStr := string(reportData)

				if !strings.Contains(reportStr, "exit") && !strings.Contains(reportStr, "1") {
					t.Errorf("[%s] report deveria registrar exit code de falha:\n%s", toolLabel, reportStr)
				}
			},
		},
		{
			name:   "P-INT-3-auth-error",
			script: authErrorMockScript,
			verify: func(t *testing.T, prdDir string, execErr error, toolLabel string) {
				t.Helper()
				// Execute pode retornar nil (loop abortado internamente) ou err
				// O criterio e o stop reason no report

				reportPath := filepath.Join(prdDir, "report.md")
				reportData, err := os.ReadFile(reportPath)
				if err != nil {
					t.Fatalf("[%s] report nao encontrado apos auth error: %v", toolLabel, err)
				}
				reportStr := string(reportData)

				if !strings.Contains(reportStr, "autenticad") {
					t.Errorf("[%s] report deveria mencionar erro de autenticacao, got:\n%s",
						toolLabel, reportStr)
				}
			},
		},
	}

	for _, sc := range scenarios {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			// Registrar resultados por ferramenta para comparar paridade
			type toolResult struct {
				taskFileStatus string
				reportContains []string
			}
			results := make(map[string]toolResult, len(toolCases))

			for _, tc := range toolCases {
				tc := tc
				t.Run(tc.label, func(t *testing.T) {
					// Cada subteste recebe filesystem completamente isolado
					tmpBase := t.TempDir()
					prdDir := setupIntegrationPRD(t, tmpBase)
					binDir := t.TempDir()

					installMockBinaries(t, binDir, tc.binaries, sc.script)

					reportPath := filepath.Join(prdDir, "report.md")
					execErr := runIntegrationExecute(t, prdDir, tc.tool, reportPath, 10*time.Second)

					sc.verify(t, prdDir, execErr, tc.label)

					// Coletar resultado para comparacao de paridade
					taskData, _ := os.ReadFile(filepath.Join(prdDir, "task-1.0-feature.md"))
					reportData, _ := os.ReadFile(reportPath)
					results[tc.label] = toolResult{
						taskFileStatus: string(taskData),
						reportContains: []string{string(reportData)},
					}
				})
			}

			// Verificar paridade: todos os resultados devem ser semanticamente equivalentes
			if len(results) >= 2 {
				var referenceLabel string
				var referenceStatus string
				for label, res := range results {
					if referenceLabel == "" {
						referenceLabel = label
						referenceStatus = res.taskFileStatus
						continue
					}
					if res.taskFileStatus != referenceStatus {
						t.Errorf(
							"paridade violada no cenario %s: ferramenta %q produziu task file %q, "+
								"mas ferramenta %q produziu %q",
							sc.name, label, res.taskFileStatus,
							referenceLabel, referenceStatus,
						)
					}
				}
			}
		})
	}
}

// TestParidadeIntegrationTimeout verifica que o loop nao trava quando o mock binary
// demora mais que o timeout configurado. O processo e morto via SIGKILL e o loop
// registra exit code -1 sem alterar o task file.
//
// Nota: este cenario usa timeout curto (2s) para nao atrasar o suite de testes.
func TestParidadeIntegrationTimeout(t *testing.T) {
	tools := []struct {
		tool   string
		binary string
		label  string
	}{
		{"claude", "claude", "claude"},
		{"codex", "codex", "codex"},
		{"gemini", "gemini", "gemini"},
		{"copilot", "copilot", "copilot"},
	}

	for _, tc := range tools {
		tc := tc
		t.Run(tc.label, func(t *testing.T) {
			tmpBase := t.TempDir()
			prdDir := setupIntegrationPRD(t, tmpBase)
			binDir := t.TempDir()

			installMockBinaries(t, binDir, []string{tc.binary}, timeoutMockScript)

			reportPath := filepath.Join(prdDir, "report.md")
			execErr := runIntegrationExecute(t, prdDir, tc.tool, reportPath, 2*time.Second)

			if execErr != nil {
				t.Fatalf("[%s] Execute nao deveria retornar erro por timeout: %v", tc.label, execErr)
			}

			// Task file deve permanecer inalterado (processo foi morto antes de escrever)
			taskFile := filepath.Join(prdDir, "task-1.0-feature.md")
			data, err := os.ReadFile(taskFile)
			if err != nil {
				t.Fatalf("[%s] task file nao encontrado: %v", tc.label, err)
			}
			if strings.Contains(string(data), "**Status:** done") {
				t.Errorf("[%s] task file nao deveria ter sido atualizado apos timeout, got:\n%s",
					tc.label, data)
			}

			// Report deve existir e registrar a iteracao com exit code -1
			reportData, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatalf("[%s] report nao encontrado: %v", tc.label, err)
			}
			reportStr := string(reportData)

			if !strings.Contains(reportStr, "-1") && !strings.Contains(reportStr, "timeout") &&
				!strings.Contains(reportStr, "inalterado") {
				t.Errorf("[%s] report deveria registrar timeout ou exit -1 ou status inalterado:\n%s",
					tc.label, reportStr)
			}

		})
	}
}

// TestRunLoopIntegrationCaminhoFeliz — RunLoop end-to-end com FakeFileSystem,
// stubs deterministicos e telemetria opt-in. Cobre RNF-01 (contrato),
// RNF-02 (rastreabilidade via LoopReport) e RNF-03 (telemetria opt-in).
func TestRunLoopIntegrationCaminhoFeliz(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	fsys, prd := setupRunLoopFS([]string{"1.0", "2.0"})
	svc := NewService(fsys, &output.Printer{Out: io.Discard, Err: io.Discard})

	deps := RunLoopDeps{
		Selector: &stubSelector{queue: []TaskEntry{
			{ID: "1.0", Title: "T 1.0", Status: "pending"},
			{ID: "2.0", Title: "T 2.0", Status: "pending"},
		}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: &stubReviewer{results: []FinalReviewResult{{Verdict: VerdictApproved}}},
	}

	var report *LoopReport
	var execErr error
	stderr := captureStderr(t, func() {
		report, execErr = svc.RunLoop(context.Background(), Options{
			PRDFolder:  prd,
			ReportPath: prd + "/loop-report.json",
			Timeout:    5 * time.Second,
		}, deps)
	})

	if execErr != nil {
		t.Fatalf("RunLoop: %v", execErr)
	}
	if len(report.TasksCompleted) != 2 {
		t.Fatalf("TasksCompleted=%d, want 2", len(report.TasksCompleted))
	}
	if report.FinalReview == nil || report.FinalReview.Verdict != VerdictApproved {
		t.Fatalf("verdict=%+v, want Approved", report.FinalReview)
	}

	wantedEvents := []string{
		"event=task_completed value=1.0",
		"event=task_completed value=2.0",
		"event=final_review_verdict value=APPROVED",
	}
	for _, ev := range wantedEvents {
		if !strings.Contains(stderr, ev) {
			t.Errorf("telemetria ausente: %q\nstderr:\n%s", ev, stderr)
		}
	}
}

// TestRunLoopIntegrationEscalonamento — bugfix exausto apos MaxBugfixIterations
// emite escalated=bugfix_exhausted e LoopReport.Escalated=true.
func TestRunLoopIntegrationEscalonamento(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, &output.Printer{Out: io.Discard, Err: io.Discard})

	critical := []Finding{{Severity: SeverityCritical, File: "x.go", Line: 1, Message: "bug"}}
	reviewer := &stubReviewer{results: []FinalReviewResult{
		{Verdict: VerdictRejected, Findings: critical},
		{Verdict: VerdictRejected, Findings: critical},
		{Verdict: VerdictRejected, Findings: critical},
		{Verdict: VerdictRejected, Findings: critical},
	}}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: reviewer,
		BugfixInvoker: &runloopBugfixInvoker{},
		DiffCapturer:  &runloopDiffCapturer{},
	}

	var report *LoopReport
	var execErr error
	stderr := captureStderr(t, func() {
		report, execErr = svc.RunLoop(context.Background(), Options{
			PRDFolder:           prd,
			MaxBugfixIterations: 3,
			ReportPath:          prd + "/loop-report.json",
		}, deps)
	})

	if !errors.Is(execErr, ErrBugfixExhausted) {
		t.Fatalf("err=%v, want ErrBugfixExhausted", execErr)
	}
	if !report.Escalated || report.BugfixCycles != 3 {
		t.Fatalf("Escalated=%v BugfixCycles=%d", report.Escalated, report.BugfixCycles)
	}
	if !strings.Contains(stderr, "event=escalated value=bugfix_exhausted") {
		t.Errorf("telemetria de escalonamento ausente:\n%s", stderr)
	}
}

// TestRunLoopIntegrationApprovedWithRemarksNonInteractive — em modo nao-interativo,
// findings sob APPROVED_WITH_REMARKS recebem decisao automatica (DefaultAction)
// sem precisar de prompter humano.
func TestRunLoopIntegrationApprovedWithRemarksNonInteractive(t *testing.T) {
	fsys, prd := setupRunLoopFS([]string{"1.0"})
	svc := NewService(fsys, &output.Printer{Out: io.Discard, Err: io.Discard})

	findings := []Finding{
		{Severity: SeverityImportant, File: "a.go", Line: 1, Message: "x"},
		{Severity: SeveritySuggestion, File: "b.go", Line: 2, Message: "y"},
	}

	deps := RunLoopDeps{
		Selector:      &stubSelector{queue: []TaskEntry{{ID: "1.0", Title: "T 1.0"}}},
		Executor:      &stubExecutor{},
		Gate:          &stubGate{},
		Recorder:      &stubRecorder{},
		FinalReviewer: &stubReviewer{results: []FinalReviewResult{{Verdict: VerdictApprovedWithRemarks, Findings: findings}}},
	}

	report, err := svc.RunLoop(context.Background(), Options{
		PRDFolder:      prd,
		NonInteractive: true,
		ReportPath:     prd + "/loop-report.json",
	}, deps)
	if err != nil {
		t.Fatalf("RunLoop: %v", err)
	}
	if report.ActionPlan == nil {
		t.Fatalf("ActionPlan ausente em modo nao-interativo")
	}
	if got := len(report.ActionPlan.Decisions); got != len(findings) {
		t.Fatalf("decisoes=%d, want %d", got, len(findings))
	}
	for _, d := range report.ActionPlan.Decisions {
		if d.Action == "" {
			t.Errorf("decisao sem acao em modo nao-interativo: %+v", d)
		}
	}
	taskContent, err := fsys.ReadFile(prd + "/task-1.0-t.md")
	if err != nil {
		t.Fatalf("task file: %v", err)
	}
	if !strings.Contains(string(taskContent), "## Plano de Ação") {
		t.Fatalf("plano de acao nao persistido no arquivo da task:\n%s", taskContent)
	}
	tasksContent, err := fsys.ReadFile(prd + "/tasks.md")
	if err != nil {
		t.Fatalf("tasks.md: %v", err)
	}
	if !strings.Contains(string(tasksContent), "Follow-up:") {
		t.Fatalf("follow-up nao anexado ao tasks.md:\n%s", tasksContent)
	}
}
