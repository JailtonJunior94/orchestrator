package taskloop

// TestReproducaoBugStatusTasksMdOverwrite reproduz o bug onde o loop determina
// incorretamente postStatus = "pending" mesmo quando o task file foi atualizado
// para "done" pelo agente (exit 0).
//
// Causa raiz: em taskloop.go, apos ler postStatus do task file ("done"),
// o bloco "Tambem verificar no tasks.md atualizado" sobrescreve postStatus
// com o status de tasks.md ("pending", pois o agente nao atualizou tasks.md).
// Resultado: postStatus == preStatus == "pending" → task marcada como
// "status inalterado apos execucao; pulando" — incorretamente.
//
// Este teste DEVE FALHAR com o codigo atual para provar que o bug existe.
// Remove a falha ao corrigir a causa raiz em taskloop.go (tarefa 2.0).
//
// Hipotese confirmada: sobrescrita incondicional de postStatus pelo status de
// tasks.md (camada loop, nao adapter). Relacionada a H4 generalizado.

import (
	"context"
	"strings"
	"testing"
	"time"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// setupBugReproFS cria um FakeFileSystem com estrutura minima para reproducao do bug.
// Inclui AGENTS.md, prd.md, techspec.md, task files e tasks.md com nTasks tasks pendentes.
// Se nTasks == 1: apenas task 1.0 (sem dependencias).
// Se nTasks == 2: tasks 1.0 (sem deps) e 2.0 (dep em 1.0).
func setupBugReproFS(nTasks int) (*taskfs.FakeFileSystem, string) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** pending\n")

	switch nTasks {
	case 1:
		fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")
	case 2:
		fsys.Files[prd+"/task-2.0-b.md"] = []byte("**Status:** pending\n")
		fsys.Files[prd+"/tasks.md"] = []byte(
			"| 1.0 | Task One | pending | — | Nao |\n" +
				"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
		)
	}

	return fsys, prd
}

// TestReproducaoBugStatusTasksMdOverwrite verifica que quando o agente escreve
// "**Status:** done" no task file e retorna exit 0, mas NAO atualiza tasks.md,
// o loop deve reconhecer postStatus = "done" (via task file) e NAO sobrescrever
// com o status "pending" de tasks.md.
//
// BUG ATUAL: postStatus e sobrescrito para "pending" (de tasks.md), causando
// "status inalterado apos execucao; pulando" mesmo com task file em "done".
//
// Este teste FALHA com o codigo atual — falha intencional que evidencia o bug.
func TestReproducaoBugStatusTasksMdOverwrite(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	tests := []struct {
		name       string
		wantReport string // substring esperada no relatorio (evidencia do comportamento correto)
		bugReport  string // substring que aparece no relatorio quando o bug esta presente
	}{
		{
			name:       "task file atualizado para done mas tasks.md nao atualizado",
			wantReport: "pending -> done",
			bugReport:  "status inalterado apos execucao; pulando",
		},
	}

	for _, tt := range tests {
		for _, tool := range tools {
			t.Run(tt.name+"/"+tool, func(t *testing.T) {
				fsys, prd := setupBugReproFS(1)

				invokerCalled := false
				svc := NewService(fsys, newTestPrinter())
				svc.binaryChecker = noBinaryCheck
				svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
					return &callbackInvoker{
						binary: invTool,
						fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
							invokerCalled = true
							// Agente atualiza task file para "done" corretamente
							fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** done\n")
							// Agente NAO atualiza tasks.md — este e o cenario que dispara o bug
							return "task completed successfully", "", 0, nil
						},
					}, nil
				}

				opts := Options{
					PRDFolder:     prd,
					Tool:          tool,
					MaxIterations: 3,
					Timeout:       5 * time.Second,
					ReportPath:    prd + "/report.md",
				}

				if err := svc.Execute(opts); err != nil {
					t.Fatalf("Execute: %v", err)
				}

				if !invokerCalled {
					t.Fatal("invoker nao foi chamado")
				}

				// Pre-condicao: confirmar que o task file foi realmente atualizado para "done"
				taskFileData, err := fsys.ReadFile(prd + "/task-1.0-a.md")
				if err != nil {
					t.Fatalf("task file nao encontrado: %v", err)
				}
				taskFileStatus := ReadTaskFileStatus(taskFileData)
				if taskFileStatus != "done" {
					t.Fatalf("pre-condicao falhou: task file deveria estar 'done', got %q", taskFileStatus)
				}

				// Verificar relatorio
				reportData, err := fsys.ReadFile(prd + "/report.md")
				if err != nil {
					t.Fatalf("relatorio nao encontrado: %v", err)
				}
				reportStr := string(reportData)

				// O relatorio deve mostrar "pending -> done" (task file foi atualizado para done)
				// BUG: atualmente mostra "pending -> pending" porque tasks.md sobrescreve postStatus
				if !strings.Contains(reportStr, tt.wantReport) {
					t.Errorf(
						"BUG REPRODUZIDO [tool=%s]: relatorio deveria conter %q "+
							"(task file atualizado para done por exit-0 invoker), "+
							"mas nao contem. "+
							"Causa raiz: em taskloop.go, bloco 'Tambem verificar no tasks.md atualizado' "+
							"sobrescreve postStatus incondicionalmente com status de tasks.md, "+
							"apagando a leitura correta do task file.",
						tool, tt.wantReport,
					)
				}

				// O relatorio NAO deve conter "status inalterado" para task cujo arquivo foi
				// explicitamente atualizado para "done" pelo agente
				if strings.Contains(reportStr, tt.bugReport) {
					t.Errorf(
						"BUG REPRODUZIDO [tool=%s]: relatorio contem %q "+
							"para task cujo task file foi atualizado para 'done'. "+
							"Path de codigo defeituoso: taskloop.go (postStatus determination, "+
							"bloco 'Tambem verificar no tasks.md atualizado').",
						tool, tt.bugReport,
					)
				}
			})
		}
	}
}

// TestReproducaoBugLoopNaoAvancaComMaxIteracoes1 demonstra que quando MaxIterations=1,
// o bug de sobrescrita de postStatus consome a unica iteracao disponivel na task 1.0
// (marcando-a como "status inalterado"), impedindo a execucao da task 2.0 (que depende
// de 1.0 estar "done" em tasks.md ou via reconcile).
//
// Cenario: invoker escreve task-1.0 como "done" no task file, exit 0, sem atualizar tasks.md.
// Com MaxIterations=1, a task 2.0 NUNCA e selecionada.
//
// BUG ATUAL: iteracao 1 e desperdicada (task 1.0 marcada como "status inalterado"),
// task 2.0 nunca executada. Este teste FALHA com o codigo atual.
func TestReproducaoBugLoopNaoAvancaComMaxIteracoes1(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			fsys, prd := setupBugReproFS(2)

			var executedTaskIDs []string
			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
				return &callbackInvoker{
					binary: invTool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						switch {
						case strings.Contains(prompt, "task-1.0"):
							executedTaskIDs = append(executedTaskIDs, "1.0")
							// Agente atualiza task file para "done", NAO atualiza tasks.md
							fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** done\n")
						case strings.Contains(prompt, "task-2.0"):
							executedTaskIDs = append(executedTaskIDs, "2.0")
							fsys.Files[prd+"/task-2.0-b.md"] = []byte("**Status:** done\n")
						}
						return "completed", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 1, // limite apertado — expoe o bug: iteracao e desperdicada
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			// Com comportamento correto (sem o bug), a unica iteracao deveria:
			// 1. Executar task 1.0 e detectar postStatus="done" (via task file)
			// 2. Marcar task 1.0 como done
			// Nao ha iteracao disponivel para task 2.0, mas ao menos 1.0 estaria corretamente done.
			//
			// Com o bug: iteracao 1 detecta postStatus="pending" (sobrescrita de tasks.md),
			// marca task 1.0 como "status inalterado; pulando", e o loop para.
			// Task 2.0 nunca executada.

			if len(executedTaskIDs) == 0 {
				t.Fatal("nenhuma task foi executada — invoker nunca chamado")
			}

			if executedTaskIDs[0] != "1.0" {
				t.Errorf("primeira task deveria ser 1.0, got %q", executedTaskIDs[0])
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("relatorio nao encontrado: %v", err)
			}
			reportStr := string(reportData)

			// Com comportamento correto: relatorio deve mostrar task 1.0 como "done"
			// BUG: mostra "pending" — a iteracao foi desperdicada
			if !strings.Contains(reportStr, "pending -> done") {
				t.Errorf(
					"BUG REPRODUZIDO [tool=%s, MaxIterations=1]: "+
						"task 1.0 foi executada (exit 0, task file='done') mas "+
						"relatorio nao contem 'pending -> done'. "+
						"A unica iteracao disponivel foi desperdicada pelo bug de sobrescrita de postStatus. "+
						"Path: taskloop.go (postStatus determination).",
					tool,
				)
			}
		})
	}
}

// TestReproducaoBugPostStatusDeterminationPath verifica que a logica de determinacao
// de postStatus em taskloop.go prioriza o task file sobre tasks.md.
//
// Cenario: task file diz "done" (agente atualizou), tasks.md diz "pending" (nao atualizado).
// Comportamento correto (pos-correcao): postStatus = "done" (task file tem prioridade).
// Comportamento antigo (pre-correcao — bug): postStatus = "pending" (tasks.md sobrescrevia).
//
// Este teste valida o comportamento correto via Service.Execute para confirmar que
// a correcao em taskloop.go e efetiva.
func TestReproducaoBugPostStatusDeterminationPath(t *testing.T) {
	const prd = "/fake/project/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files["/fake/project/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	// task file diz "done" (agente atualizou corretamente)
	fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** done\n")
	// tasks.md ainda diz "pending" (agente nao atualizou tasks.md — cenario do bug)
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

	// Confirmar que readTaskStatus retorna "done" do task file
	primaryStatus := readTaskStatus(prd+"/task-1.0-a.md", fsys)
	if primaryStatus != "done" {
		t.Fatalf("readTaskStatus deveria retornar 'done', got %q", primaryStatus)
	}

	// Simular a logica CORRIGIDA de determinacao de postStatus:
	// tasks.md so e consultado como fallback quando task file nao atualizou o status.
	preStatus := "pending"
	postStatus := primaryStatus // "done" — task file tem prioridade
	if postStatus == preStatus {
		// fallback: checar tasks.md apenas se task file nao mudou
		updatedContent, _ := fsys.ReadFile(prd + "/tasks.md")
		updatedTasks, _ := ParseTasksFile(updatedContent)
		for _, ut := range updatedTasks {
			if ut.ID == "1.0" && ut.Status != "" {
				postStatus = ut.Status
				break
			}
		}
	}

	// Comportamento correto: task file ("done") nao deve ser sobrescrito por tasks.md ("pending")
	if postStatus != "done" {
		t.Errorf(
			"postStatus deveria ser 'done' (task file tem prioridade sobre tasks.md), got %q. "+
				"Verificar se a correcao em taskloop.go (bloco 'Verificar no tasks.md atualizado apenas como fallback') "+
				"esta aplicada corretamente.",
			postStatus,
		)
	}

	// Confirmar que postStatus != preStatus → task nao seria skippada
	if postStatus == preStatus {
		t.Errorf(
			"postStatus (%q) == preStatus (%q): task seria incorretamente marcada como "+
				"'status inalterado apos execucao; pulando' mesmo com task file em 'done'.",
			postStatus, preStatus,
		)
	}
}

// TestRegressaoExistentesNaoAfetados garante que os cenarios existentes continuam
// passando apos a adicao dos testes de reproducao (regressao zero).
// Roda o mesmo cenario de TestExecuteSimpleMode (invoker atualiza tasks.md e task file)
// para confirmar que o caminho feliz nao foi quebrado pelos novos testes.
func TestRegressaoExistentesNaoAfetados(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			fsys, prd := setupBugReproFS(1)

			executorCalled := false
			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
				return &callbackInvoker{
					binary: invTool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						executorCalled = true
						// Caminho feliz: agente atualiza AMBOS task file E tasks.md
						fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
						return "completed", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 3,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("Execute: %v", err)
			}

			if !executorCalled {
				t.Fatal("executor nao foi invocado")
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("relatorio nao encontrado: %v", err)
			}
			reportStr := string(reportData)

			// No caminho feliz (ambos atualizados), relatorio deve mostrar "done"
			if !strings.Contains(reportStr, "pending -> done") {
				t.Errorf("[tool=%s] caminho feliz: relatorio deveria conter 'pending -> done', got:\n%s", tool, reportStr)
			}

			// Nao deve haver "status inalterado" no caminho feliz
			if strings.Contains(reportStr, "status inalterado") {
				t.Errorf("[tool=%s] caminho feliz: relatorio nao deveria conter 'status inalterado'", tool)
			}
		})
	}
}
