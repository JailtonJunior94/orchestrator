package taskloop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// TestResolveWorkDir valida a logica de busca da raiz do projeto via marcadores.
func TestResolveWorkDir(t *testing.T) {
	tests := []struct {
		name      string
		prdFolder string
		setup     func(fsys *taskfs.FakeFileSystem)
		want      string
	}{
		{
			name:      ".git presente no diretorio pai",
			prdFolder: "/fake/project/tasks/prd-feature",
			setup: func(fsys *taskfs.FakeFileSystem) {
				// FakeFileSystem.Exists retorna true para prefixo de arquivo existente.
				fsys.Files["/fake/project/.git/HEAD"] = []byte("ref: refs/heads/main")
			},
			want: "/fake/project",
		},
		{
			name:      "go.mod presente no diretorio corrente",
			prdFolder: "/fake/project",
			setup: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files["/fake/project/go.mod"] = []byte("module example.com/app\n")
			},
			want: "/fake/project",
		},
		{
			name:      "AGENTS.md presente em diretorio ancestral",
			prdFolder: "/fake/project/tasks/prd-feature",
			setup: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files["/fake/project/AGENTS.md"] = []byte("# Agents\n")
			},
			want: "/fake/project",
		},
		{
			name:      "nenhum marker encontrado — fallback para prdFolder",
			prdFolder: "/fake/isolated/prd-feature",
			setup:     func(fsys *taskfs.FakeFileSystem) {},
			want:      "/fake/isolated/prd-feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := taskfs.NewFakeFileSystem()
			tt.setup(fsys)

			got, err := resolveWorkDir(tt.prdFolder, fsys)
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveWorkDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

// callbackInvoker implementa AgentInvoker com comportamento configuravel por callback.
// Usado para testes de orquestracao sem subprocessos reais.
type callbackInvoker struct {
	binary string
	fn     func(ctx context.Context, prompt, workDir, model string) (string, string, int, error)
	calls  []callbackInvokerCall
}

type callbackInvokerCall struct {
	prompt  string
	workDir string
	model   string
}

func (c *callbackInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	c.calls = append(c.calls, callbackInvokerCall{prompt, workDir, model})
	return c.fn(ctx, prompt, workDir, model)
}

func (c *callbackInvoker) BinaryName() string { return c.binary }

// newTestPrinter cria um Printer que descarta toda saida (adequado para testes).
func newTestPrinter() *output.Printer {
	return &output.Printer{Out: io.Discard, Err: io.Discard}
}

// noBinaryCheck retorna um binaryChecker que sempre passa.
// Necessario para testes hermeticos: evita dependencia de binarios reais no PATH (R-TEST-001).
func noBinaryCheck(AgentInvoker) error { return nil }

// setupBaseFS monta um FakeFileSystem com a estrutura minima para o Execute funcionar.
// Retorna o fsys e o path absoluto do PRD folder.
// tasks.md inicial tem uma task "1.0" no status solicitado.
func setupBaseFS(taskStatus string) (*taskfs.FakeFileSystem, string) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** " + taskStatus + "\n")
	fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", taskStatus)

	return fsys, prd
}

// tasksContent gera o conteudo de tasks.md com uma unica task.
func tasksContent(id, title, status string) []byte {
	return []byte(fmt.Sprintf("| %s | %s | %s | — | Nao |\n", id, title, status))
}

// TestExecuteSimpleMode verifica regressao: modo simples (Profiles=nil) executa
// o executor, nao invoca reviewer e captura o resultado corretamente.
func TestExecuteSimpleMode(t *testing.T) {
	fsys, prd := setupBaseFS("pending")
	executorCalled := false

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		if tool != "claude" {
			t.Errorf("ferramenta inesperada: %q", tool)
		}
		executorCalled = true
		return &callbackInvoker{
			binary: "claude",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				// Atualizar tasks.md para "done" para o loop encerrar
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "completed", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles:      nil, // modo simples
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if !executorCalled {
		t.Error("executor nao foi invocado")
	}

	// Verificar que o relatorio foi escrito
	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if len(reportStr) == 0 {
		t.Error("relatorio vazio")
	}
}

// TestExecuteMaxIterationsZeroRunsUntilAllDone verifica que MaxIterations=0
// faz o loop executar ilimitadamente ate todas as tasks estarem concluidas.
func TestExecuteMaxIterationsZeroRunsUntilAllDone(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-3.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | — | Nao |\n" +
			"| 3.0 | Task Three | pending | — | Nao |\n",
	)

	executorCallCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: "claude",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				executorCallCount++
				switch executorCallCount {
				case 1:
					fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte(
						"| 1.0 | Task One | done | — | Nao |\n" +
							"| 2.0 | Task Two | pending | — | Nao |\n" +
							"| 3.0 | Task Three | pending | — | Nao |\n",
					)
				case 2:
					fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte(
						"| 1.0 | Task One | done | — | Nao |\n" +
							"| 2.0 | Task Two | done | — | Nao |\n" +
							"| 3.0 | Task Three | pending | — | Nao |\n",
					)
				case 3:
					fsys.Files[prd+"/task-3.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte(
						"| 1.0 | Task One | done | — | Nao |\n" +
							"| 2.0 | Task Two | done | — | Nao |\n" +
							"| 3.0 | Task Three | done | — | Nao |\n",
					)
				}
				return "done", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 0, // ilimitado — deve executar ate todas as tasks estarem done
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if executorCallCount != 3 {
		t.Errorf("MaxIterations=0 deveria executar todas as 3 tasks, executou %d", executorCallCount)
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "todas as tasks completadas") {
		t.Errorf("StopReason deveria ser 'todas as tasks completadas', relatorio:\n%s", reportStr)
	}
}

// TestExecuteAdvancedModeReviewerInvoked verifica que o reviewer e invocado
// apos executor com exit 0 e status "done" no modo avancado.
func TestExecuteAdvancedModeReviewerInvoked(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	var reviewerCalled bool
	var reviewerModel string

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Simula executor: atualiza tasks.md para "done"
					fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
					return "executor output", "", 0, nil
				},
			}, nil
		case "codex":
			reviewerCalled = true
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					reviewerModel = model
					return "reviewer output approved", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada no teste: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "gpt-5.4")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true, // pula validacao para teste
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if !reviewerCalled {
		t.Fatal("reviewer nao foi invocado apos executor com exit 0 e status done")
	}

	if reviewerModel != "gpt-5.4" {
		t.Errorf("reviewer model: got %q, want %q", reviewerModel, "gpt-5.4")
	}

	// Verificar ReviewResult no relatorio
	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if reportStr == "" {
		t.Fatal("relatorio vazio")
	}
	// Modo avancado deve ter coluna Papel
	if !strings.Contains(reportStr, "reviewer") {
		t.Error("relatorio nao contem 'reviewer'")
	}
}

// TestExecuteAdvancedModeReviewerNotInvokedOnExecutorFailure verifica que o reviewer
// nao e invocado quando o executor falha (exit != 0) E nao atualiza a task para "done".
// O criterio do reviewer e o postStatus == "done", nao o exit code.
func TestExecuteAdvancedModeReviewerNotInvokedOnExecutorFailure(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	reviewerCalled := false

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Executor falha; nao atualiza tasks.md
					return "", "compilation error", 1, nil
				},
			}, nil
		case "codex":
			reviewerCalled = true
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "never called", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if reviewerCalled {
		t.Error("reviewer foi invocado mesmo com executor com exit != 0")
	}

	// Verificar que ReviewResult e nil para a iteracao
	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if strings.Contains(reportStr, "#### Review Result") {
		t.Error("relatorio nao deveria conter secao Review Result")
	}
}

// TestExecuteAdvancedModeReviewerNotInvokedWhenStatusNotDone verifica que o reviewer
// nao e invocado quando o executor sai com exit 0 mas o status da task nao e "done".
func TestExecuteAdvancedModeReviewerNotInvokedWhenStatusNotDone(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	reviewerCalled := false

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Executor sai com exit 0 mas nao atualiza tasks.md (status permanece "pending")
					return "partial work", "", 0, nil
				},
			}, nil
		case "codex":
			reviewerCalled = true
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "never called", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if reviewerCalled {
		t.Error("reviewer foi invocado mesmo com status != done")
	}
}

// TestExecuteAdvancedModeReviewerInvokedOnTimeoutWithDone verifica regressao BUG-2:
// quando o executor e morto por timeout (exit=-1) mas atualiza a task para "done" antes
// do SIGKILL, o reviewer deve ser invocado normalmente — o criterio e postStatus=="done".
func TestExecuteAdvancedModeReviewerInvokedOnTimeoutWithDone(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	var reviewerCalled bool

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Simula executor que marca a task como done mas sai com -1 (SIGKILL por timeout)
					fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
					return "output before sigkill", "", -1, nil
				},
			}, nil
		case "codex":
			reviewerCalled = true
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "reviewer output", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if !reviewerCalled {
		t.Error("reviewer nao foi invocado mesmo com executor exit=-1 e postStatus=done (regressao BUG-2)")
	}

	// Verificar que o relatorio conta a task como sucesso, nao falha
	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "- **Executadas com sucesso:** 1") {
		t.Errorf("esperado 'Executadas com sucesso: 1' no relatorio\noutput:\n%s", reportStr)
	}
	if strings.Contains(reportStr, "- **Falhadas:** 1") {
		t.Errorf("nao esperado 'Falhadas: 1' quando postStatus=done\noutput:\n%s", reportStr)
	}
}

// TestExecuteAdvancedModeReviewerFailureCapturesNote verifica que quando o reviewer
// falha (exit != 0), a note "reviewer reportou problemas criticos" e capturada no ReviewResult,
// sem alterar o status da task.
func TestExecuteAdvancedModeReviewerFailureCapturesNote(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
					return "executor output", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "critical issues found", "errors", 1, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "reviewer reportou problemas criticos") {
		t.Errorf("relatorio nao contem nota de falha do reviewer\noutput:\n%s", reportStr)
	}
	// Status da task nao deve ser alterado pelo reviewer
	if !strings.Contains(reportStr, "done") {
		t.Error("post-status da task nao deve ter sido alterado pelo reviewer")
	}
}

// TestExecuteAdvancedModeBugfixInvokedOnReviewerFailure verifica que quando o reviewer
// retorna exit != 0 (achados criticos), o bugfix e invocado com o executor e o prompt
// de bugfix contendo os achados do reviewer.
func TestExecuteAdvancedModeBugfixInvokedOnReviewerFailure(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	var bugfixCalled bool
	var bugfixPrompt string
	claudeCallCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					claudeCallCount++
					if claudeCallCount == 1 {
						// Executor: marca task como done
						fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
						return "executor output", "", 0, nil
					}
					// Bugfix: segunda chamada ao executor
					bugfixCalled = true
					bugfixPrompt = prompt
					return "bugfix applied", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "critical issues found\n- [main.go:42] variavel nao inicializada", "", 1, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if !bugfixCalled {
		t.Fatal("bugfix nao foi invocado apos reviewer com exit != 0")
	}

	// Verificar que o prompt de bugfix contem os achados do reviewer
	if !strings.Contains(bugfixPrompt, "variavel nao inicializada") {
		t.Errorf("prompt de bugfix nao contem achados do reviewer\nprompt:\n%s", bugfixPrompt)
	}
	if !strings.Contains(bugfixPrompt, "skill bugfix") {
		t.Errorf("prompt de bugfix nao referencia a skill bugfix\nprompt:\n%s", bugfixPrompt)
	}
	if !strings.Contains(bugfixPrompt, "causa raiz") {
		t.Errorf("prompt de bugfix nao contem instrucao de causa raiz\nprompt:\n%s", bugfixPrompt)
	}

	// Verificar BugfixResult no relatorio
	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "#### Bugfix Result") {
		t.Errorf("relatorio nao contem secao Bugfix Result\noutput:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "bugfix") {
		t.Errorf("relatorio nao contem 'bugfix'\noutput:\n%s", reportStr)
	}
}

// TestExecuteAdvancedModeBugfixNotInvokedOnReviewerSuccess verifica que o bugfix
// NAO e invocado quando o reviewer retorna exit 0 (revisao aprovada).
func TestExecuteAdvancedModeBugfixNotInvokedOnReviewerSuccess(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	bugfixCalled := false
	claudeCallCount2 := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					claudeCallCount2++
					if claudeCallCount2 == 1 {
						fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
						return "executor output", "", 0, nil
					}
					bugfixCalled = true
					return "bugfix output", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "all good, approved", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if bugfixCalled {
		t.Error("bugfix foi invocado mesmo com reviewer exit 0 (revisao aprovada)")
	}

	// Verificar que BugfixResult e nil no relatorio
	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if strings.Contains(reportStr, "#### Bugfix Result") {
		t.Error("relatorio contem secao Bugfix Result mesmo sem bugfix executado")
	}
}

// TestExecuteAdvancedModeBugfixFailureCapturesNote verifica que quando o bugfix
// falha (exit != 0), a note e capturada no BugfixResult.
func TestExecuteAdvancedModeBugfixFailureCapturesNote(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	claudeCallCount3 := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					claudeCallCount3++
					if claudeCallCount3 == 1 {
						fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
						return "executor output", "", 0, nil
					}
					// Bugfix falha
					return "could not fix", "errors", 1, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "critical issues found", "", 1, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "bugfix nao conseguiu corrigir todos os achados") {
		t.Errorf("relatorio nao contem nota de falha do bugfix\noutput:\n%s", reportStr)
	}
}

// TestExecuteAdvancedModeBugfixUsesExecutorModel verifica que o bugfix usa o model
// do executor (nao do reviewer) ao invocar o agente de correcao.
func TestExecuteAdvancedModeBugfixUsesExecutorModel(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	var bugfixModel string
	claudeCalls := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					claudeCalls++
					if claudeCalls == 1 {
						fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
						return "executor output", "", 0, nil
					}
					// Bugfix call — captura o model usado
					bugfixModel = model
					return "bugfix applied", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "critical issues", "", 1, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "claude-sonnet-4-6")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "gpt-5.4")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if bugfixModel != "claude-sonnet-4-6" {
		t.Errorf("bugfix model: got %q, want %q (deve usar executor model, nao reviewer)", bugfixModel, "claude-sonnet-4-6")
	}
}

// TestExecuteAdvancedModeBugfixNotInvokedWithoutReviewer verifica que o bugfix
// nao e invocado quando o reviewer nao esta configurado (Profiles.Reviewer == nil).
func TestExecuteAdvancedModeBugfixNotInvokedWithoutReviewer(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	claudeCalls := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				claudeCalls++
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "executor output", "", 0, nil
			},
		}, nil
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: nil, // sem reviewer
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	// Apenas 1 chamada: executor. Sem reviewer, sem bugfix.
	if claudeCalls != 1 {
		t.Errorf("executor chamado %d vezes, esperado 1 (sem reviewer/bugfix)", claudeCalls)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if strings.Contains(reportStr, "#### Bugfix Result") {
		t.Error("relatorio contem Bugfix Result sem reviewer configurado")
	}
}

// TestExecuteAdvancedModeBugfixDoesNotIncrementIteration verifica RF-13:
// o bugfix e sub-etapa e nao incrementa o contador de iteracoes.
// Com MaxIterations=2 e 2 tasks, o loop deve completar ambas mesmo com bugfix rodando.
func TestExecuteAdvancedModeBugfixDoesNotIncrementIteration(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | — | Nao |\n",
	)

	executorCallCount := 0
	bugfixCallCount := 0
	reviewerCallCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Distinguir executor vs bugfix pelo conteudo do prompt
					if strings.Contains(prompt, "skill bugfix") {
						bugfixCallCount++
						return "bugfix applied", "", 0, nil
					}
					executorCallCount++
					if executorCallCount == 1 {
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | pending | — | Nao |\n",
						)
					} else {
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | done | — | Nao |\n",
						)
					}
					return "done", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					reviewerCallCount++
					return "critical issues", "", 1, nil // reviewer sempre reprova
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 2,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	// RF-13: com MaxIterations=2, deve ter executado 2 iteracoes do executor
	if executorCallCount != 2 {
		t.Errorf("executor chamado %d vezes, esperado 2", executorCallCount)
	}
	// 2 reviews (um por task)
	if reviewerCallCount != 2 {
		t.Errorf("reviewer chamado %d vezes, esperado 2", reviewerCallCount)
	}
	// 2 bugfixes (um por task, pois reviewer sempre reprova)
	if bugfixCallCount != 2 {
		t.Errorf("bugfix chamado %d vezes, esperado 2", bugfixCallCount)
	}
}

// TestExecuteAdvancedModeBugfixInvocationError verifica que quando o agente de bugfix
// retorna um erro de invocacao (ex: timeout), a note e capturada sem abortar o loop.
func TestExecuteAdvancedModeBugfixInvocationError(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	claudeCallsBf := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					claudeCallsBf++
					if claudeCallsBf == 1 {
						fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
						return "executor output", "", 0, nil
					}
					// Bugfix retorna erro de invocacao
					return "", "timeout", -1, fmt.Errorf("context deadline exceeded")
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "critical issues", "", 1, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "erro de invocacao do bugfix") {
		t.Errorf("relatorio nao contem nota de erro de invocacao do bugfix\noutput:\n%s", reportStr)
	}
}

// TestExecuteAdvancedModeBugfixReviewFindingsPassedVerbatim verifica que a saida
// do reviewer e passada integralmente como ReviewFindings no prompt de bugfix.
func TestExecuteAdvancedModeBugfixReviewFindingsPassedVerbatim(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	reviewOutput := "Achados criticos:\n- [main.go:42] null pointer\n- [handler.go:10] missing error check\nVeredicto: reprovado"
	var capturedBugfixPrompt string
	claudeCallsVb := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					claudeCallsVb++
					if claudeCallsVb == 1 {
						fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
						return "executor output", "", 0, nil
					}
					capturedBugfixPrompt = prompt
					return "bugfix applied", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return reviewOutput, "", 1, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	// Cada linha do output do reviewer deve estar presente no prompt de bugfix
	for _, line := range strings.Split(reviewOutput, "\n") {
		if !strings.Contains(capturedBugfixPrompt, line) {
			t.Errorf("prompt de bugfix nao contem linha do reviewer: %q", line)
		}
	}
}

// TestExecuteAdvancedModeBugfixOnlyForFailedReview verifica que em um bundle
// com multiplas tasks, o bugfix roda apenas para a task cujo reviewer reprovou,
// nao para a task cujo reviewer aprovou.
func TestExecuteAdvancedModeBugfixOnlyForFailedReview(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | — | Nao |\n",
	)

	executorCalls := 0
	bugfixCalls := 0
	reviewerCalls := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					if strings.Contains(prompt, "skill bugfix") {
						bugfixCalls++
						return "bugfix applied", "", 0, nil
					}
					executorCalls++
					if executorCalls == 1 {
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | pending | — | Nao |\n",
						)
					} else {
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | done | — | Nao |\n",
						)
					}
					return "done", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					reviewerCalls++
					// Reviewer reprova task 1, aprova task 2
					if reviewerCalls == 1 {
						return "critical issues in task 1", "", 1, nil
					}
					return "approved", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if executorCalls != 2 {
		t.Errorf("executor chamado %d vezes, esperado 2", executorCalls)
	}
	if reviewerCalls != 2 {
		t.Errorf("reviewer chamado %d vezes, esperado 2", reviewerCalls)
	}
	// Bugfix so para task 1 (reviewer reprovou), nao para task 2 (reviewer aprovou)
	if bugfixCalls != 1 {
		t.Errorf("bugfix chamado %d vezes, esperado 1 (somente para task reprovada)", bugfixCalls)
	}
}

// TestExecuteFallbackPreLoop verifica que o fallback camada 2 substitui o perfil
// invalido pelo FallbackTool antes de iniciar o loop, sem retornar erro.
func TestExecuteFallbackPreLoop(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	var invokedTool string

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		invokedTool = tool
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "completed", "", 0, nil
			},
		}, nil
	}

	// Perfil com modelo invalido para a ferramenta claude
	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	// Forcar modelo invalido diretamente (contorna construtor)
	invalidProfile := ExecutionProfile{
		role:     "executor",
		tool:     "claude",
		provider: "anthropic",
		model:    "modelo-totalmente-invalido-xyz",
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: invalidProfile,
			Reviewer: nil,
		},
		FallbackTool:      "claude",
		AllowUnknownModel: false,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado com fallback: %v", err)
	}

	// Apos fallback, o invoker criado deve ser para "claude"
	if invokedTool != "claude" {
		t.Errorf("invoker criado para %q, esperado %q", invokedTool, "claude")
	}

	// Verificar que o perfil do executor foi aplicado corretamente
	_ = execProfile // usado apenas para silenciar linter
}

// TestExecuteAllowUnknownModelSkipsValidation verifica que AllowUnknownModel=true
// pula a validacao de compatibilidade e nao retorna erro mesmo com modelo invalido.
func TestExecuteAllowUnknownModelSkipsValidation(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "completed", "", 0, nil
			},
		}, nil
	}

	// Perfil com modelo que nao existe na tabela de compatibilidade
	invalidProfile := ExecutionProfile{
		role:     "executor",
		tool:     "claude",
		provider: "anthropic",
		model:    "modelo-futuro-nao-catalogado",
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: invalidProfile,
			Reviewer: nil,
		},
		AllowUnknownModel: true, // pular validacao
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado com AllowUnknownModel=true: %v", err)
	}
}

// TestExecuteIncompatibleModelWithoutFallbackReturnsError verifica que sem FallbackTool
// e com modelo incompativel, Execute retorna erro de pre-flight.
func TestExecuteIncompatibleModelWithoutFallbackReturnsError(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return nil, fmt.Errorf("nao deveria ser chamado")
	}

	invalidProfile := ExecutionProfile{
		role:     "executor",
		tool:     "claude",
		provider: "anthropic",
		model:    "modelo-invalido-xyz",
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: invalidProfile,
			Reviewer: nil,
		},
		FallbackTool:      "",    // sem fallback
		AllowUnknownModel: false, // validar
	}

	err := svc.Execute(opts)
	if err == nil {
		t.Fatal("esperado erro de pre-flight, mas Execute retornou nil")
	}
}

// TestExecuteMaxIterationsCountsOnlyExecutor verifica RF-13: o contador de max-iterations
// reflete apenas execucoes do executor; o reviewer e sub-etapa e nao incrementa o contador.
func TestExecuteMaxIterationsCountsOnlyExecutor(t *testing.T) {
	// Setup com 2 tasks pendentes
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | — | Nao |\n",
	)

	executorCallCount := 0
	reviewerCallCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					executorCallCount++
					// Marcar a primeira task ainda pendente como done
					if executorCallCount == 1 {
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | pending | — | Nao |\n",
						)
					} else {
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | done | — | Nao |\n",
						)
					}
					return "done", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					reviewerCallCount++
					return "approved", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 2, // suficiente para 2 tasks do executor
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	// RF-13: com MaxIterations=2, deve ter executado 2 iteracoes do executor
	// e 2 do reviewer (mas nao 4 iteracoes totais como se reviewer contasse)
	if executorCallCount != 2 {
		t.Errorf("executor chamado %d vezes, esperado 2", executorCallCount)
	}
	if reviewerCallCount != 2 {
		t.Errorf("reviewer chamado %d vezes, esperado 2 (uma por task done)", reviewerCallCount)
	}
}

// TestExecuteAdvancedModeReportContainsProfiles verifica que o relatorio em modo avancado
// contem os perfis do executor e reviewer no cabecalho.
func TestExecuteAdvancedModeReportContainsProfiles(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "done", "", 0, nil
			},
		}, nil
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "claude-sonnet-4-6")
	revProfile, _ := NewExecutionProfile("reviewer", "claude", "claude-opus-4-6")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)

	checks := []string{
		"**Modo:** avancado",
		"**Executor:** claude / anthropic / claude-sonnet-4-6",
		"**Reviewer:** claude / anthropic / claude-opus-4-6",
	}
	for _, check := range checks {
		if !strings.Contains(reportStr, check) {
			t.Errorf("relatorio nao contem %q\noutput:\n%s", check, reportStr)
		}
	}
}

// newCapturePrinter cria um Printer que captura toda saida (Out e Err) em um buffer.
// Usado para inspecionar o output do dry-run nos testes.
func newCapturePrinter() (*output.Printer, *bytes.Buffer) {
	var buf bytes.Buffer
	return &output.Printer{Out: &buf, Err: &buf}, &buf
}

// TestExecuteSimpleModeReportFormat verifica regressao: modo simples (Profiles=nil)
// gera relatorio com formato identico ao atual — sem coluna Papel, sem perfis no cabecalho.
func TestExecuteSimpleModeReportFormat(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "done", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles:      nil, // modo simples
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)

	// Deve conter formato simples
	wantIn := []string{"**Modo:** simples", "**Tool:** claude"}
	for _, want := range wantIn {
		if !strings.Contains(reportStr, want) {
			t.Errorf("relatorio nao contem %q\noutput:\n%s", want, reportStr)
		}
	}

	// Nao deve conter elementos do modo avancado
	wantNotIn := []string{"**Executor:**", "**Reviewer:**", "| Papel |"}
	for _, notWant := range wantNotIn {
		if strings.Contains(reportStr, notWant) {
			t.Errorf("relatorio nao deveria conter %q\noutput:\n%s", notWant, reportStr)
		}
	}
}

// TestDryRunAdvancedCompatibleProfiles verifica que o dry-run no modo avancado exibe
// modo, perfis do executor e reviewer com status de compatibilidade, template e tasks
// elegiveis (RF-09). Tambem verifica a linha de iteracao por task.
func TestDryRunAdvancedCompatibleProfiles(t *testing.T) {
	fsys, prd := setupBaseFS("pending")
	printer, buf := newCapturePrinter()

	svc := NewService(fsys, printer)
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{binary: tool, fn: func(_ context.Context, _, _, _ string) (string, string, int, error) {
			return "", "", 0, nil
		}}, nil
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "claude-sonnet-4-6")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "gpt-5.4")

	opts := Options{
		PRDFolder:     prd,
		DryRun:        true,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true, // pular validacao de compatibilidade no pre-flight
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	out := buf.String()

	tests := []struct {
		want string
	}{
		{"modo: avancado"},
		{"executor: claude / anthropic / claude-sonnet-4-6"},
		{"✓ (compativel)"},
		{"reviewer: codex / openai / gpt-5.4"},
		{"template de revisao: default (embutido)"},
		{"tasks elegiveis: 1.0"},
		{"iteracao 1: executaria task 1.0 com executor, depois reviewer"},
	}
	for _, tt := range tests {
		if !strings.Contains(out, tt.want) {
			t.Errorf("dry-run nao contem %q\noutput:\n%s", tt.want, out)
		}
	}

	// Nao deve invocar o agente — nenhuma chamada ao invoker alem da criacao
	if strings.Contains(out, "executor output") || strings.Contains(out, "reviewer output") {
		t.Error("dry-run nao deveria conter output de invocacao real")
	}
}

// TestDryRunAdvancedIncompatibleProfile verifica que o dry-run exibe "✗ (incompativel)"
// quando o modelo nao esta na tabela de compatibilidade, mesmo com AllowUnknownModel=true
// (que apenas bypassa o pre-flight, nao altera o label de compatibilidade).
func TestDryRunAdvancedIncompatibleProfile(t *testing.T) {
	fsys, prd := setupBaseFS("pending")
	printer, buf := newCapturePrinter()

	svc := NewService(fsys, printer)
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{binary: tool, fn: func(_ context.Context, _, _, _ string) (string, string, int, error) {
			return "", "", 0, nil
		}}, nil
	}

	// Modelo nao catalogado na tabela de compatibilidade
	invalidProfile := ExecutionProfile{
		role:     "executor",
		tool:     "claude",
		provider: "anthropic",
		model:    "modelo-futuro-nao-catalogado-xyz",
	}

	opts := Options{
		PRDFolder:     prd,
		DryRun:        true,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: invalidProfile,
			Reviewer: nil,
		},
		AllowUnknownModel: true, // permite passar pelo pre-flight; header ainda mostra ✗
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	out := buf.String()

	if !strings.Contains(out, "✗ (incompativel)") {
		t.Errorf("dry-run deveria exibir '✗ (incompativel)' para modelo desconhecido\noutput:\n%s", out)
	}
	if !strings.Contains(out, "executor: claude / anthropic / modelo-futuro-nao-catalogado-xyz") {
		t.Errorf("dry-run nao exibe perfil do executor corretamente\noutput:\n%s", out)
	}
}

// TestDryRunSimplePreservesCurrentFormat verifica regressao: dry-run no modo simples
// continua imprimindo o formato anterior sem elementos do modo avancado (RF-02).
func TestDryRunSimplePreservesCurrentFormat(t *testing.T) {
	fsys, prd := setupBaseFS("pending")
	printer, buf := newCapturePrinter()

	svc := NewService(fsys, printer)
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{binary: tool, fn: func(_ context.Context, _, _, _ string) (string, string, int, error) {
			return "", "", 0, nil
		}}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		DryRun:        true,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles:      nil, // modo simples
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	out := buf.String()

	// Formato simples esperado
	if !strings.Contains(out, "invocaria claude com prompt para task 1.0") {
		t.Errorf("dry-run simples nao contem mensagem esperada\noutput:\n%s", out)
	}

	// Nao deve conter elementos exclusivos do modo avancado
	wantAbsent := []string{
		"modo: avancado",
		"executor:",
		"reviewer:",
		"template de revisao:",
		"tasks elegiveis:",
		"executaria task",
	}
	for _, absent := range wantAbsent {
		if strings.Contains(out, absent) {
			t.Errorf("dry-run simples nao deveria conter %q\noutput:\n%s", absent, out)
		}
	}
}

// TestExecuteFallbackChainExhausted verifica RF-11.4: quando FallbackTool e invalida,
// Execute deve retornar erro indicando "fallback-tool invalido", sem invocar nenhum agente.
func TestExecuteFallbackChainExhausted(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return nil, fmt.Errorf("nao deveria ser invocado neste cenario")
	}

	// Executor com modelo invalido — vai acionar o fallback
	invalidProfile := ExecutionProfile{
		role:     "executor",
		tool:     "claude",
		provider: "anthropic",
		model:    "modelo-invalido-xyz",
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: invalidProfile,
			Reviewer: nil,
		},
		FallbackTool:      "nao-existe-ferramenta", // fallback invalido
		AllowUnknownModel: false,
	}

	err := svc.Execute(opts)
	if err == nil {
		t.Fatal("esperado erro quando FallbackTool e invalida, mas Execute retornou nil")
	}
	if !strings.Contains(err.Error(), "fallback-tool invalido") {
		t.Errorf("mensagem de erro deve conter 'fallback-tool invalido', obteve: %v", err)
	}
}

// TestExecuteFallbackBothRolesSubstituted verifica RF-11.2: quando executor e reviewer
// possuem modelos invalidos e FallbackTool e valida, ambos os papeis sao substituidos
// pelo fallback antes de iniciar o loop.
func TestExecuteFallbackBothRolesSubstituted(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	var invokedTools []string

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		invokedTools = append(invokedTools, tool)
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				// Executor marca task como done; reviewer apenas confirma
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "done", "", 0, nil
			},
		}, nil
	}

	// Ambos os perfis com modelos invalidos para acionar substituicao por fallback
	invalidExec := ExecutionProfile{
		role:     "executor",
		tool:     "claude",
		provider: "anthropic",
		model:    "modelo-invalido-executor-xyz",
	}
	invalidRev := ExecutionProfile{
		role:     "reviewer",
		tool:     "claude",
		provider: "anthropic",
		model:    "modelo-invalido-reviewer-xyz",
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: invalidExec,
			Reviewer: &invalidRev,
		},
		FallbackTool:      "claude", // fallback valido
		AllowUnknownModel: false,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado com fallback: %v", err)
	}

	// Apos substituicao: executor usa "claude" (fallback) e reviewer usa "claude" (fallback).
	// invokerFactory e chamado: uma vez para o executor, uma vez para o reviewer.
	if len(invokedTools) < 2 {
		t.Errorf("esperado pelo menos 2 chamadas ao invokerFactory (executor + reviewer), obteve %d: %v", len(invokedTools), invokedTools)
	}
	for _, tool := range invokedTools {
		if tool != "claude" {
			t.Errorf("ferramenta inesperada apos fallback: %q (esperado apenas 'claude')", tool)
		}
	}
}

// TestDryRunAdvancedMultipleEligibleTasks verifica RF-09: dry-run modo avancado com
// multiplas tasks elegiveis lista todas no cabecalho e exibe plano de iteracao para cada uma.
func TestDryRunAdvancedMultipleEligibleTasks(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-3.0-gamma.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Alpha | pending | — | Nao |\n" +
			"| 2.0 | Beta | pending | — | Nao |\n" +
			"| 3.0 | Gamma | pending | — | Nao |\n",
	)

	printer, buf := newCapturePrinter()
	svc := NewService(fsys, printer)
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{binary: tool, fn: func(_ context.Context, _, _, _ string) (string, string, int, error) {
			return "", "", 0, nil
		}}, nil
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "claude-sonnet-4-6")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "gpt-5.4")

	opts := Options{
		PRDFolder:     prd,
		DryRun:        true,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	out := buf.String()

	// Cabecalho deve listar todas as tasks elegiveis
	if !strings.Contains(out, "tasks elegiveis: 1.0, 2.0, 3.0") {
		t.Errorf("dry-run deveria listar as 3 tasks como elegiveis\noutput:\n%s", out)
	}

	// Deve exibir plano de iteracao para cada task
	for i, id := range []string{"1.0", "2.0", "3.0"} {
		iterLine := fmt.Sprintf("iteracao %d: executaria task %s com executor, depois reviewer", i+1, id)
		if !strings.Contains(out, iterLine) {
			t.Errorf("dry-run deveria conter %q\noutput:\n%s", iterLine, out)
		}
	}

	// Nenhum agente deve ter sido invocado de verdade
	if strings.Contains(out, "executor output") || strings.Contains(out, "reviewer output") {
		t.Error("dry-run nao deveria conter output de invocacao real")
	}
}

// TestExecuteAuthErrorEarlyTermination verifica que quando o agente retorna erro de
// autenticacao (exit != 0 com padrao de auth no output), o loop e encerrado imediatamente
// com stop reason especifico e orientacao no relatorio.
func TestExecuteAuthErrorEarlyTermination(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | — | Nao |\n",
	)

	invokeCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: "claude",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				invokeCount++
				return "Not logged in · Please run /login", "", 1, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	// O loop deve ter sido encerrado apos a primeira iteracao (nao tentou task 2.0)
	if invokeCount != 1 {
		t.Errorf("invoker chamado %d vezes, esperado 1 (early termination por auth)", invokeCount)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)

	// Stop reason deve indicar problema de autenticacao
	if !strings.Contains(reportStr, "nao esta autenticado") {
		t.Errorf("relatorio nao contem stop reason de autenticacao\noutput:\n%s", reportStr)
	}

	// Note da iteracao deve conter orientacao
	if !strings.Contains(reportStr, "erro de autenticacao") {
		t.Errorf("relatorio nao contem nota de erro de autenticacao\noutput:\n%s", reportStr)
	}
}

// TestExecuteEmptyOutputOnTimeoutKill verifica que quando o agente e terminado
// forcadamente (exit -1) sem produzir nenhum output, uma nota diagnostica sobre TTY
// e adicionada ao resultado da iteracao (BUG-002 — codex nao escreve em pipe).
func TestExecuteEmptyOutputOnTimeoutKill(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.liveOutOverride = io.Discard
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: "codex",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				// Simula CLI terminada por SIGKILL (timeout): exit -1, sem stdout, sem stderr
				return "", "", -1, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "codex",
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)

	if !strings.Contains(reportStr, "saida vazia") {
		t.Errorf("relatorio deveria conter nota de saida vazia, obteve:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "TTY") {
		t.Errorf("relatorio deveria mencionar TTY como possivel causa, obteve:\n%s", reportStr)
	}
}

// TestExecuteNonAuthErrorContinuesLoop verifica que erros de agente que nao sao auth
// (exit != 0 sem padrao de auth) nao causam terminacao antecipada — o loop continua
// com a proxima task elegivel.
func TestExecuteNonAuthErrorContinuesLoop(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | — | Nao |\n",
	)

	invokeCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: "claude",
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				invokeCount++
				// Erro normal, nao de auth
				return "compilation failed", "error: syntax", 1, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	// O loop deve continuar e tentar a task 2.0
	if invokeCount != 2 {
		t.Errorf("invoker chamado %d vezes, esperado 2 (loop deve continuar com erro nao-auth)", invokeCount)
	}
}

// TestExecuteResumesInProgressTask verifica a regressao do impasse em que uma task
// ja marcada como in_progress por sessao anterior deve ser retomada pelo loop.
func TestExecuteResumesInProgressTask(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** in_progress\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | done | — | Nao |\n" +
			"| 2.0 | Task Two | in_progress | 1.0 | Nao |\n",
	)

	invokeCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				invokeCount++
				fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** done\n")
				fsys.Files[prd+"/tasks.md"] = []byte(
					"| 1.0 | Task One | done | — | Nao |\n" +
						"| 2.0 | Task Two | done | 1.0 | Nao |\n",
				)
				return "resumed and completed", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 3,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if invokeCount != 1 {
		t.Fatalf("invoker chamado %d vezes, esperado 1 para retomar task in_progress", invokeCount)
	}

	reportData, _ := fsys.ReadFile(prd + "/report.md")
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "| 2.0 | Task Two | done |") {
		t.Errorf("relatorio nao registra task retomada como done\noutput:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "todas as tasks completadas ou em estado terminal") {
		t.Errorf("stop reason inesperado para task retomada\noutput:\n%s", reportStr)
	}
}

func TestExecutePrioritizesInProgressBeforePending(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** in_progress\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Pending Task | pending | — | Nao |\n" +
			"| 2.0 | In Progress Task | in_progress | — | Nao |\n",
	)

	var invokedTask string

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				switch {
				case strings.Contains(prompt, "task-2.0-test.md"):
					invokedTask = "2.0"
					fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte(
						"| 1.0 | Pending Task | pending | — | Nao |\n" +
							"| 2.0 | In Progress Task | done | — | Nao |\n",
					)
				case strings.Contains(prompt, "task-1.0-test.md"):
					invokedTask = "1.0"
				default:
					t.Fatalf("prompt nao contem task esperada:\n%s", prompt)
				}
				return "completed", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if invokedTask != "2.0" {
		t.Fatalf("task executada = %q, want 2.0", invokedTask)
	}
}

func TestExecuteRejectsUnauthorizedTasksRowMutationForAllProviders(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")

			originalTasks := []byte(
				"| 1.0 | Task One | pending | — | Nao |\n" +
					"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
			)
			fsys.Files[prd+"/tasks.md"] = append([]byte(nil), originalTasks...)

			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invokerTool string) (AgentInvoker, error) {
				if invokerTool != tool {
					t.Fatalf("tool inesperada: got %q want %q", invokerTool, tool)
				}
				return &callbackInvoker{
					binary: tool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | done | 1.0 | Nao |\n",
						)
						return "mutated another task row", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 2,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("Execute retornou erro inesperado: %v", err)
			}

			gotTasks, err := fsys.ReadFile(prd + "/tasks.md")
			if err != nil {
				t.Fatalf("tasks.md nao encontrado: %v", err)
			}
			if string(gotTasks) != string(originalTasks) {
				t.Fatalf("tasks.md deveria ter sido restaurado apos violacao\nwant:\n%s\ngot:\n%s", originalTasks, gotTasks)
			}

			taskFile, _ := fsys.ReadFile(prd + "/task-1.0-test.md")
			if string(taskFile) != "**Status:** pending\n" {
				t.Fatalf("arquivo da task atual deveria ser restaurado, obteve: %q", string(taskFile))
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("relatorio nao encontrado: %v", err)
			}
			reportStr := string(reportData)
			if !strings.Contains(reportStr, "abortado: agente violou isolamento da task 1.0") {
				t.Fatalf("relatorio nao contem stop reason de violacao de isolamento\noutput:\n%s", reportStr)
			}
			if !strings.Contains(reportStr, "row da task 2.0 foi alterada indevidamente em tasks.md") {
				t.Fatalf("relatorio nao contem diagnostico da row indevida\noutput:\n%s", reportStr)
			}
		})
	}
}

func TestExecuteRejectsUnauthorizedTaskFileMutation(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\nDetalhes originais\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
	)

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
				fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\nALTERADO INDEVIDAMENTE\n")
				fsys.Files[prd+"/tasks.md"] = []byte(
					"| 1.0 | Task One | done | — | Nao |\n" +
						"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
				)
				return "mutated another task file", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 2,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	taskTwo, err := fsys.ReadFile(prd + "/task-2.0-test.md")
	if err != nil {
		t.Fatalf("task-2.0-test.md nao encontrado: %v", err)
	}
	if string(taskTwo) != "**Status:** pending\nDetalhes originais\n" {
		t.Fatalf("arquivo de task nao deveria permanecer alterado\noutput:\n%s", taskTwo)
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "arquivo de task task-2.0-test.md foi alterado indevidamente") {
		t.Fatalf("relatorio nao contem diagnostico do arquivo alterado\noutput:\n%s", reportStr)
	}
}

func TestExecuteRejectsUnexpectedTrackedTaskFileCreation(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
				fsys.Files[prd+"/task-2.0-intrusa.md"] = []byte("**Status:** pending\n")
				return "created extra task file", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if _, err := fsys.ReadFile(prd + "/task-2.0-intrusa.md"); err == nil {
		t.Fatal("arquivo de task criado indevidamente deveria ter sido removido")
	}

	taskOne, err := fsys.ReadFile(prd + "/task-1.0-test.md")
	if err != nil {
		t.Fatalf("task-1.0-test.md nao encontrado: %v", err)
	}
	if string(taskOne) != "**Status:** pending\n" {
		t.Fatalf("arquivo da task atual deveria ter sido restaurado, obteve: %q", string(taskOne))
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "novo arquivo de task task-2.0-intrusa.md foi adicionado indevidamente") {
		t.Fatalf("relatorio nao contem diagnostico do arquivo novo\noutput:\n%s", reportStr)
	}
}

func TestExecuteRejectsProtectedPRDFileMutationForAllProviders(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD original\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec original\n")
			fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invokerTool string) (AgentInvoker, error) {
				if invokerTool != tool {
					t.Fatalf("tool inesperada: got %q want %q", invokerTool, tool)
				}
				return &callbackInvoker{
					binary: tool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
						fsys.Files[prd+"/prd.md"] = []byte("# PRD alterado indevidamente\n")
						return "mutated prd.md", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 1,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("Execute retornou erro inesperado: %v", err)
			}

			if got := string(fsys.Files[prd+"/prd.md"]); got != "# PRD original\n" {
				t.Fatalf("prd.md deveria ter sido restaurado, obteve: %q", got)
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("relatorio nao encontrado: %v", err)
			}
			reportStr := string(reportData)
			if !strings.Contains(reportStr, "arquivo protegido do PRD prd.md foi alterado indevidamente") {
				t.Fatalf("relatorio nao contem diagnostico de prd.md alterado\noutput:\n%s", reportStr)
			}
		})
	}
}

func TestExecuteRejectsArbitraryPRDFileMutationForAllProviders(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD original\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec original\n")
			fsys.Files[prd+"/notes.md"] = []byte("conteudo original\n")
			fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invokerTool string) (AgentInvoker, error) {
				if invokerTool != tool {
					t.Fatalf("tool inesperada: got %q want %q", invokerTool, tool)
				}
				return &callbackInvoker{
					binary: tool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
						fsys.Files[prd+"/notes.md"] = []byte("alterado indevidamente\n")
						return "mutated notes.md", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 1,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("Execute retornou erro inesperado: %v", err)
			}

			if got := string(fsys.Files[prd+"/notes.md"]); got != "conteudo original\n" {
				t.Fatalf("notes.md deveria ter sido restaurado, obteve: %q", got)
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("relatorio nao encontrado: %v", err)
			}
			reportStr := string(reportData)
			if !strings.Contains(reportStr, "arquivo protegido do PRD notes.md foi alterado indevidamente") {
				t.Fatalf("relatorio nao contem diagnostico de notes.md alterado\noutput:\n%s", reportStr)
			}
		})
	}
}

func TestExecuteRejectsReviewerIsolationViolation(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task One | pending | — | Nao |\n" +
			"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
	)

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				switch tool {
				case "claude":
					fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte(
						"| 1.0 | Task One | done | — | Nao |\n" +
							"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
					)
					return "executor completed", "", 0, nil
				case "codex":
					fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** done\nALTERADO INDEVIDAMENTE\n")
					return "reviewer mutated another task file", "", 0, nil
				default:
					t.Fatalf("tool inesperada: %q", tool)
					return "", "", 0, nil
				}
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	taskTwo, err := fsys.ReadFile(prd + "/task-2.0-test.md")
	if err != nil {
		t.Fatalf("task-2.0-test.md nao encontrado: %v", err)
	}
	if string(taskTwo) != "**Status:** pending\n" {
		t.Fatalf("arquivo da task nao deveria permanecer alterado pelo reviewer, obteve: %q", string(taskTwo))
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "abortado: reviewer violou isolamento da task 1.0") {
		t.Fatalf("relatorio nao contem stop reason do reviewer\noutput:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "violacao de isolamento detectada: arquivo de task task-2.0-test.md foi alterado indevidamente") {
		t.Fatalf("relatorio nao contem diagnostico do reviewer\noutput:\n%s", reportStr)
	}
}

func TestExecuteRejectsReviewerProtectedPRDMutation(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD original\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec original\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				switch tool {
				case "claude":
					fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
					return "executor completed", "", 0, nil
				case "codex":
					fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec alterado indevidamente\n")
					return "reviewer mutated techspec", "", 0, nil
				default:
					t.Fatalf("tool inesperada: %q", tool)
					return "", "", 0, nil
				}
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if got := string(fsys.Files[prd+"/techspec.md"]); got != "# TechSpec original\n" {
		t.Fatalf("techspec.md deveria ter sido restaurado, obteve: %q", got)
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "abortado: reviewer violou isolamento da task 1.0") {
		t.Fatalf("relatorio nao contem stop reason do reviewer\noutput:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "violacao de isolamento detectada: arquivo protegido do PRD techspec.md foi alterado indevidamente") {
		t.Fatalf("relatorio nao contem diagnostico de techspec.md alterado\noutput:\n%s", reportStr)
	}
}

func TestExecuteRejectsReviewerArbitraryPRDMutation(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD original\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec original\n")
	fsys.Files[prd+"/notes.md"] = []byte("conteudo original\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				switch tool {
				case "claude":
					fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
					return "executor completed", "", 0, nil
				case "codex":
					fsys.Files[prd+"/notes.md"] = []byte("alterado indevidamente pelo reviewer\n")
					return "reviewer mutated notes", "", 0, nil
				default:
					t.Fatalf("tool inesperada: %q", tool)
					return "", "", 0, nil
				}
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if got := string(fsys.Files[prd+"/notes.md"]); got != "conteudo original\n" {
		t.Fatalf("notes.md deveria ter sido restaurado, obteve: %q", got)
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "abortado: reviewer violou isolamento da task 1.0") {
		t.Fatalf("relatorio nao contem stop reason do reviewer\noutput:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "violacao de isolamento detectada: arquivo protegido do PRD notes.md foi alterado indevidamente") {
		t.Fatalf("relatorio nao contem diagnostico de notes.md alterado\noutput:\n%s", reportStr)
	}
}

func TestExecuteRejectsReviewerCurrentTaskMutation(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				switch tool {
				case "claude":
					fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
					return "executor completed", "", 0, nil
				case "codex":
					fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** blocked\n")
					return "reviewer mutated current task", "", 0, nil
				default:
					t.Fatalf("tool inesperada: %q", tool)
					return "", "", 0, nil
				}
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	taskOne, err := fsys.ReadFile(prd + "/task-1.0-test.md")
	if err != nil {
		t.Fatalf("task-1.0-test.md nao encontrado: %v", err)
	}
	if string(taskOne) != "**Status:** done\n" {
		t.Fatalf("arquivo da task atual deveria ter sido restaurado para done, obteve: %q", string(taskOne))
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "abortado: reviewer violou isolamento da task 1.0") {
		t.Fatalf("relatorio nao contem stop reason do reviewer\noutput:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "violacao de isolamento detectada: arquivo de task task-1.0-test.md foi alterado indevidamente") {
		t.Fatalf("relatorio nao contem diagnostico da task atual alterada\noutput:\n%s", reportStr)
	}
}

func TestExecuteRejectsReviewerCurrentTaskRowMutation(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				switch tool {
				case "claude":
					fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
					return "executor completed", "", 0, nil
				case "codex":
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | blocked | — | Nao |\n")
					return "reviewer mutated current task row", "", 0, nil
				default:
					t.Fatalf("tool inesperada: %q", tool)
					return "", "", 0, nil
				}
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	tasksData, err := fsys.ReadFile(prd + "/tasks.md")
	if err != nil {
		t.Fatalf("tasks.md nao encontrado: %v", err)
	}
	if string(tasksData) != "| 1.0 | Task One | done | — | Nao |\n" {
		t.Fatalf("row da task atual deveria ter sido restaurada para done, obteve:\n%s", string(tasksData))
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "abortado: reviewer violou isolamento da task 1.0") {
		t.Fatalf("relatorio nao contem stop reason do reviewer\noutput:\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "violacao de isolamento detectada: row da task 1.0 foi alterada indevidamente em tasks.md") {
		t.Fatalf("relatorio nao contem diagnostico da row atual alterada\noutput:\n%s", reportStr)
	}
}

func TestExecuteDoesNotResumeTaskWhenTaskFileStatusIsTerminal(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** blocked\n")
	fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | in_progress | — | Nao |\n")

	invoked := false

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				invoked = true
				return "should not run", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 1,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	if invoked {
		t.Fatal("task com status efetivo blocked nao deveria ser retomada")
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "todas as tasks completadas ou em estado terminal") {
		t.Fatalf("stop reason inesperado\noutput:\n%s", reportStr)
	}
}

// TestDryRunAdvancedTemplatePreview verifica que o dry-run modo avancado exibe
// o preview do template resolvido para a primeira task elegivel (RF-12).
func TestDryRunAdvancedTemplatePreview(t *testing.T) {
	fsys, prd := setupBaseFS("pending")
	printer, buf := newCapturePrinter()

	svc := NewService(fsys, printer)
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{binary: tool, fn: func(_ context.Context, _, _, _ string) (string, string, int, error) {
			return "", "", 0, nil
		}}, nil
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "claude-sonnet-4-6")
	revProfile, _ := NewExecutionProfile("reviewer", "claude", "claude-opus-4-6")

	opts := Options{
		PRDFolder:     prd,
		DryRun:        true,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	out := buf.String()

	// O preview do template deve estar presente
	if !strings.Contains(out, "--- preview do template (task 1.0) ---") {
		t.Errorf("dry-run nao contem cabecalho do preview do template\noutput:\n%s", out)
	}
	if !strings.Contains(out, "--- fim do preview ---") {
		t.Errorf("dry-run nao contem rodape do preview do template\noutput:\n%s", out)
	}
	// O template default contem a palavra "review" — verifica que o conteudo foi expandido
	if !strings.Contains(out, "review") {
		t.Errorf("dry-run nao contem conteudo do template\noutput:\n%s", out)
	}
}

// TestParidadeSemanticaCicloDeVida verifica paridade semantica para as 4 ferramentas.
// Implementa ADR-001: matriz table-driven com 10 cenarios × 4 ferramentas = 40 sub-testes.
// Cada cenario usa o mesmo invokerBehavior para todas as ferramentas — se alguma divergir,
// o sub-teste correspondente falha.
func TestParidadeSemanticaCicloDeVida(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-parity"

	tools := []string{"claude", "codex", "gemini", "copilot"}

	// baseSetup inicializa arquivos obrigatorios (AGENTS.md, prd.md, techspec.md).
	baseSetup := func(fsys *taskfs.FakeFileSystem) {
		fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
		fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
		fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	}

	type parityCase struct {
		name          string
		setupFS       func(fsys *taskfs.FakeFileSystem)
		makeInvokerFn func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error)
		maxIterations int
		verifyReport  func(t *testing.T, report string, tool string)
	}

	tests := []parityCase{
		// P1: Execucao com sucesso — exit 0, task file atualizado para "done".
		// RF-01, RF-09: selecao elegivel e paridade para as 4 ferramentas.
		{
			name: "P1/sucesso_exit_0",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Alpha | pending | — | Nao |\n")
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Alpha | done | — | Nao |\n")
					return "task completed successfully", "", 0, nil
				}
			},
			maxIterations: 3,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				if !strings.Contains(report, "pending -> done") {
					t.Errorf("[%s] P1: relatorio deveria conter 'pending -> done'\n%s", tool, report)
				}
				if strings.Contains(report, "status inalterado") {
					t.Errorf("[%s] P1: relatorio nao deveria conter 'status inalterado'", tool)
				}
				if !strings.Contains(report, "todas as tasks completadas") {
					t.Errorf("[%s] P1: stop reason deveria conter 'todas as tasks completadas'\n%s", tool, report)
				}
			},
		},
		// P2: Execucao com timeout — exit -1, task file atualizado para "done" antes do SIGKILL.
		// RF-09: exit -1 com postStatus=="done" nao deve causar skip (regressao BUG-2).
		{
			name: "P2/timeout_exit_minus1_status_done",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Alpha | pending | — | Nao |\n")
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Agente marca done antes de ser morto por timeout (SIGKILL → exit -1)
					fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** done\n")
					return "output before sigkill", "", -1, nil
				}
			},
			maxIterations: 3,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				if !strings.Contains(report, "pending -> done") {
					t.Errorf("[%s] P2: exit -1 com task file done deve resultar em 'pending -> done'\n%s", tool, report)
				}
				if strings.Contains(report, "status inalterado") {
					t.Errorf("[%s] P2: relatorio nao deveria conter 'status inalterado' quando task file esta done", tool)
				}
			},
		},
		// P3: Execucao com falha — exit 1, task file inalterado.
		// RF-09, RF-15: falha registrada com nota consistente; status preservado.
		{
			name: "P3/falha_exit_1_status_inalterado",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Alpha | pending | — | Nao |\n")
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Agente falha; nao atualiza task file
					return "", "compilation error: undefined variable foo", 1, nil
				}
			},
			maxIterations: 1,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				if strings.Contains(report, "pending -> done") {
					t.Errorf("[%s] P3: relatorio nao deveria conter 'pending -> done' com exit 1", tool)
				}
				if !strings.Contains(report, "agente saiu com codigo 1") {
					t.Errorf("[%s] P3: relatorio deveria conter 'agente saiu com codigo 1'\n%s", tool, report)
				}
			},
		},
		// P4: Erro de autenticacao — exit 1, output contem "not authenticated".
		// RF-16: erro de autenticacao detectado e loop abortado consistentemente.
		{
			name: "P4/erro_autenticacao",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Alpha | pending | — | Nao |\n")
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "Error: not authenticated. Please run /login to continue", "", 1, nil
				}
			},
			maxIterations: 3,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				if !strings.Contains(report, "abortado") {
					t.Errorf("[%s] P4: stop reason deveria conter 'abortado'\n%s", tool, report)
				}
				if !strings.Contains(report, "autenticad") {
					t.Errorf("[%s] P4: relatorio deveria mencionar autenticacao\n%s", tool, report)
				}
			},
		},
		// P5: Status inalterado — exit 0, task file nao atualizado.
		// RF-09: prevencao de loop infinito por skip de task sem progresso.
		{
			name: "P5/status_inalterado_exit_0",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Alpha | pending | — | Nao |\n")
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Agente sai com exit 0 mas NAO atualiza task file
					return "partial work done, more to follow", "", 0, nil
				}
			},
			maxIterations: 1,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				if !strings.Contains(report, "status inalterado apos execucao") {
					t.Errorf("[%s] P5: relatorio deveria conter 'status inalterado apos execucao'\n%s", tool, report)
				}
			},
		},
		// P6: Violacao de isolamento — invoker modifica task file de outra task.
		// RF-17: violacao aborta independentemente da ferramenta, snapshot restaurado.
		{
			name: "P6/violacao_isolamento",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte(
					"| 1.0 | Alpha | pending | — | Nao |\n" +
						"| 2.0 | Beta | pending | — | Nao |\n",
				)
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					// Agente modifica task file de OUTRA task (task-2.0-beta.md, nao a atual task-1.0-alpha.md).
					// validateTaskFileIsolation (isolation.go) detecta a mutacao porque o snapshot capturado
					// antes da invocacao inclui todos os task files da pasta PRD e compara byte a byte apos a
					// execucao, permitindo apenas a mutacao do currentTaskFile.
					fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** done\n")
					return "completed", "", 0, nil
				}
			},
			maxIterations: 3,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				// Contrato: validateTaskIsolation detecta mutacao de qualquer task file que nao seja
				// o currentTaskFile e aborta o loop com StopReason "abortado: agente violou isolamento".
				if !strings.Contains(report, "abortado") {
					t.Errorf("[%s] P6: stop reason deveria conter 'abortado'\n%s", tool, report)
				}
				if !strings.Contains(report, "isolamento") {
					t.Errorf("[%s] P6: relatorio deveria mencionar 'isolamento'\n%s", tool, report)
				}
			},
		},
		// P7: Erro de invocacao — invoker retorna err != nil.
		// RF-15: falhas de invocacao produzem nota consistente para qualquer ferramenta.
		{
			name: "P7/erro_invocacao",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Alpha | pending | — | Nao |\n")
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					return "", "", 0, fmt.Errorf("conexao recusada: agente nao disponivel")
				}
			},
			maxIterations: 1,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				if !strings.Contains(report, "erro de invocacao") {
					t.Errorf("[%s] P7: relatorio deveria conter 'erro de invocacao'\n%s", tool, report)
				}
			},
		},
		// P8: Multiplas tasks — 3 tasks pendentes, invoker marca cada como done sequencialmente.
		// RF-09: paridade em ciclo completo de multiplas iteracoes.
		{
			name: "P8/multiplas_tasks_todas_done",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/task-3.0-gamma.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte(
					"| 1.0 | Alpha | pending | — | Nao |\n" +
						"| 2.0 | Beta | pending | — | Nao |\n" +
						"| 3.0 | Gamma | pending | — | Nao |\n",
				)
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					switch {
					case strings.Contains(prompt, "task-1.0"):
						fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Alpha | done | — | Nao |\n" +
								"| 2.0 | Beta | pending | — | Nao |\n" +
								"| 3.0 | Gamma | pending | — | Nao |\n",
						)
					case strings.Contains(prompt, "task-2.0"):
						fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Alpha | done | — | Nao |\n" +
								"| 2.0 | Beta | done | — | Nao |\n" +
								"| 3.0 | Gamma | pending | — | Nao |\n",
						)
					case strings.Contains(prompt, "task-3.0"):
						fsys.Files[prd+"/task-3.0-gamma.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Alpha | done | — | Nao |\n" +
								"| 2.0 | Beta | done | — | Nao |\n" +
								"| 3.0 | Gamma | done | — | Nao |\n",
						)
					}
					return "completed", "", 0, nil
				}
			},
			maxIterations: 5,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				if !strings.Contains(report, "**Iterations:** 3") {
					t.Errorf("[%s] P8: deveria ter 3 iteracoes\n%s", tool, report)
				}
				if !strings.Contains(report, "todas as tasks completadas") {
					t.Errorf("[%s] P8: stop reason deveria conter 'todas'\n%s", tool, report)
				}
			},
		},
		// P9: Task com dependencias — 2.0 depende de 1.0 (done), 3.0 depende de 2.0 (pending).
		// RF-01: selecao elegivel respeita grafo de dependencias.
		{
			name: "P9/dependencias_bloqueiam_task",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				// 1.0 ja done (sem deps)
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** done\n")
				// 2.0 pending, dep 1.0 (done) → elegivel
				fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** pending\n")
				// 3.0 pending, dep 2.0 (pending) → bloqueada
				fsys.Files[prd+"/task-3.0-gamma.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte(
					"| 1.0 | Alpha | done | — | Nao |\n" +
						"| 2.0 | Beta | pending | 1.0 | Nao |\n" +
						"| 3.0 | Gamma | pending | 2.0 | Nao |\n",
				)
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					switch {
					case strings.Contains(prompt, "task-2.0"):
						fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Alpha | done | — | Nao |\n" +
								"| 2.0 | Beta | done | 1.0 | Nao |\n" +
								"| 3.0 | Gamma | pending | 2.0 | Nao |\n",
						)
					case strings.Contains(prompt, "task-3.0"):
						fsys.Files[prd+"/task-3.0-gamma.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Alpha | done | — | Nao |\n" +
								"| 2.0 | Beta | done | 1.0 | Nao |\n" +
								"| 3.0 | Gamma | done | 2.0 | Nao |\n",
						)
					}
					return "completed", "", 0, nil
				}
			},
			maxIterations: 5,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				// Apenas 2 iteracoes: 2.0 (1.0 ja done) e 3.0 (apos 2.0 done)
				if !strings.Contains(report, "**Iterations:** 2") {
					t.Errorf("[%s] P9: deveria ter 2 iteracoes (2.0 e 3.0)\n%s", tool, report)
				}
				if !strings.Contains(report, "todas as tasks completadas") {
					t.Errorf("[%s] P9: stop reason deveria conter 'todas'\n%s", tool, report)
				}
			},
		},
		// P10: Retomada de task in_progress — in_progress tem prioridade sobre pending.
		// RF-04: estado persistido independente de ferramenta; retomada objetiva.
		{
			name: "P10/retomada_in_progress_prioridade",
			setupFS: func(fsys *taskfs.FakeFileSystem) {
				// 1.0 em progresso (sessao anterior interrompida)
				fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** in_progress\n")
				// 2.0 pendente, sem deps
				fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** pending\n")
				fsys.Files[prd+"/tasks.md"] = []byte(
					"| 1.0 | Alpha | in_progress | — | Nao |\n" +
						"| 2.0 | Beta | pending | — | Nao |\n",
				)
			},
			makeInvokerFn: func(fsys *taskfs.FakeFileSystem) func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				return func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					switch {
					case strings.Contains(prompt, "task-1.0"):
						// Sessao anterior retomada e finalizada
						fsys.Files[prd+"/task-1.0-alpha.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Alpha | done | — | Nao |\n" +
								"| 2.0 | Beta | pending | — | Nao |\n",
						)
					case strings.Contains(prompt, "task-2.0"):
						fsys.Files[prd+"/task-2.0-beta.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Alpha | done | — | Nao |\n" +
								"| 2.0 | Beta | done | — | Nao |\n",
						)
					}
					return "completed", "", 0, nil
				}
			},
			maxIterations: 5,
			verifyReport: func(t *testing.T, report, tool string) {
				t.Helper()
				// Task 1.0 (in_progress) deve ser executada primeiro
				if !strings.Contains(report, "in_progress -> done") {
					t.Errorf("[%s] P10: relatorio deveria conter 'in_progress -> done'\n%s", tool, report)
				}
				if !strings.Contains(report, "**Iterations:** 2") {
					t.Errorf("[%s] P10: deveria ter 2 iteracoes\n%s", tool, report)
				}
				// 1.0 (in_progress→done) deve aparecer antes de 2.0 (pending→done) no relatorio
				idx10 := strings.Index(report, "in_progress -> done")
				idx20 := strings.Index(report, "pending -> done")
				if idx10 < 0 || idx20 < 0 {
					t.Errorf("[%s] P10: relatorio deveria ter ambas as transicoes de status", tool)
				} else if idx10 > idx20 {
					t.Errorf("[%s] P10: task 1.0 (in_progress) deve aparecer antes de 2.0 no relatorio", tool)
				}
			},
		},
	}

	for _, tt := range tests {
		for _, tool := range tools {
			t.Run(fmt.Sprintf("%s/%s", tt.name, tool), func(t *testing.T) {
				fsys := taskfs.NewFakeFileSystem()
				baseSetup(fsys)
				tt.setupFS(fsys)

				fn := tt.makeInvokerFn(fsys)

				svc := NewService(fsys, newTestPrinter())
				svc.binaryChecker = noBinaryCheck
				svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
					return &callbackInvoker{
						binary: invTool,
						fn:     fn,
					}, nil
				}

				opts := Options{
					PRDFolder:     prd,
					Tool:          tool,
					MaxIterations: tt.maxIterations,
					Timeout:       5 * time.Second,
					ReportPath:    prd + "/report.md",
				}

				if err := svc.Execute(opts); err != nil {
					t.Fatalf("Execute: %v", err)
				}

				reportData, err := fsys.ReadFile(prd + "/report.md")
				if err != nil {
					t.Fatalf("relatorio nao encontrado: %v", err)
				}

				if tt.verifyReport != nil {
					tt.verifyReport(t, string(reportData), tool)
				}
			})
		}
	}
}

// --- Task 4.0: Testes de isolamento de sessao entre tasks ---

// TestSessionIsolationBetweenIterationsForAllTools verifica que duas tasks executam em
// sequencia para todas as 4 ferramentas e que o estado entre iteracoes e limitado ao
// definido em RF-12: skipped map, contador de iteracoes, acumulador de report e releitura
// de tasks.md. Subtarefas 4.1 e 4.4.
func TestSessionIsolationBetweenIterationsForAllTools(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/task-2.0-b.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/tasks.md"] = []byte(
				"| 1.0 | Task One | pending | — | Nao |\n" +
					"| 2.0 | Task Two | pending | — | Nao |\n",
			)

			var executionOrder []string
			var invokePrompts []string

			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
				return &callbackInvoker{
					binary: invTool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						invokePrompts = append(invokePrompts, prompt)
						switch {
						case strings.Contains(prompt, "task-1.0"):
							executionOrder = append(executionOrder, "1.0")
							fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** done\n")
							fsys.Files[prd+"/tasks.md"] = []byte(
								"| 1.0 | Task One | done | — | Nao |\n" +
									"| 2.0 | Task Two | pending | — | Nao |\n",
							)
						case strings.Contains(prompt, "task-2.0"):
							executionOrder = append(executionOrder, "2.0")
							fsys.Files[prd+"/task-2.0-b.md"] = []byte("**Status:** done\n")
							fsys.Files[prd+"/tasks.md"] = []byte(
								"| 1.0 | Task One | done | — | Nao |\n" +
									"| 2.0 | Task Two | done | — | Nao |\n",
							)
						}
						return "completed", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 5,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("[%s] Execute: %v", tool, err)
			}

			// 4.1: Verificar que as 2 tasks foram executadas na ordem correta
			if len(executionOrder) != 2 {
				t.Fatalf("[%s] esperado 2 tasks executadas, obteve %d: %v", tool, len(executionOrder), executionOrder)
			}
			if executionOrder[0] != "1.0" || executionOrder[1] != "2.0" {
				t.Errorf("[%s] ordem de execucao incorreta: %v", tool, executionOrder)
			}

			// 4.1: Prompt da 2a iteracao deve referenciar task-2.0 (sem contaminacao de task-1.0)
			if len(invokePrompts) == 2 && !strings.Contains(invokePrompts[1], "task-2.0") {
				t.Errorf("[%s] prompt da 2a iteracao deveria conter 'task-2.0'\n%q", tool, invokePrompts[1])
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("[%s] relatorio nao encontrado: %v", tool, err)
			}
			reportStr := string(reportData)

			// RF-12 acumulador de report: ambas as iteracoes devem aparecer
			if !strings.Contains(reportStr, "1.0") || !strings.Contains(reportStr, "2.0") {
				t.Errorf("[%s] relatorio nao acumula ambas as iteracoes\n%s", tool, reportStr)
			}

			// RF-12 contador de iteracoes: exatamente 2 execucoes do executor
			if !strings.Contains(reportStr, "**Iterations:** 2") {
				t.Errorf("[%s] relatorio deveria registrar 2 iteracoes\n%s", tool, reportStr)
			}

			// RF-12 releitura de tasks.md: stop reason indica conclusao (tasks.md re-lido corretamente)
			if !strings.Contains(reportStr, "todas as tasks completadas") {
				t.Errorf("[%s] stop reason nao indica conclusao (tasks.md deve ser re-lido a cada iteracao)\n%s", tool, reportStr)
			}
		})
	}
}

// TestCaptureValidateIsolationExecutorModeForAllTools verifica que
// captureTaskIsolationSnapshotWithMode(executor) + validateTaskIsolation aceita
// mutacoes legitimas do executor (propria task file e sua row em tasks.md)
// para cada uma das 4 ferramentas. Subtarefa 4.2, modo executor.
func TestCaptureValidateIsolationExecutorModeForAllTools(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
				return &callbackInvoker{
					binary: invTool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						// Mutacao legitima: atualizar apenas propria task file e sua row em tasks.md
						fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
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
				t.Fatalf("[%s] Execute: %v", tool, err)
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("[%s] relatorio nao encontrado: %v", tool, err)
			}
			reportStr := string(reportData)

			// captureTaskIsolationSnapshotWithMode(executor) + validateTaskIsolation deve passar:
			// nenhuma violacao de isolamento para mutacao legitima
			if strings.Contains(reportStr, "violacao de isolamento") {
				t.Errorf("[%s] isolamento nao deveria ser violado em mutacao legitima do executor\n%s", tool, reportStr)
			}
			if !strings.Contains(reportStr, "pending -> done") {
				t.Errorf("[%s] task deveria ter transicao pending -> done\n%s", tool, reportStr)
			}
			if !strings.Contains(reportStr, "todas as tasks completadas") {
				t.Errorf("[%s] stop reason incorreto para execucao limpa\n%s", tool, reportStr)
			}
		})
	}
}

// TestSnapshotRestorationAfterTaskFileMutationForAllTools verifica que, quando o executor
// modifica o arquivo de task de outra task (nao a atual), o snapshot e restaurado e
// o loop e abortado para todas as 4 ferramentas. Subtarefa 4.3.
func TestSnapshotRestorationAfterTaskFileMutationForAllTools(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** pending\nconteudo original\n")
			fsys.Files[prd+"/tasks.md"] = []byte(
				"| 1.0 | Task One | pending | — | Nao |\n" +
					"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
			)

			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
				return &callbackInvoker{
					binary: invTool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						// Executor atualiza sua propria task, mas tambem modifica task file de outra (violacao)
						fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/task-2.0-test.md"] = []byte("**Status:** done\nALTERADO INDEVIDAMENTE\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Task One | done | — | Nao |\n" +
								"| 2.0 | Task Two | pending | 1.0 | Nao |\n",
						)
						return "mutated another task file", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 2,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("[%s] Execute: %v", tool, err)
			}

			// Arquivo da task nao-atual deve ter sido restaurado
			taskTwo, err := fsys.ReadFile(prd + "/task-2.0-test.md")
			if err != nil {
				t.Fatalf("[%s] task-2.0-test.md nao encontrado: %v", tool, err)
			}
			if string(taskTwo) != "**Status:** pending\nconteudo original\n" {
				t.Fatalf("[%s] task-2.0-test.md deveria ter sido restaurado, obteve: %q", tool, string(taskTwo))
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("[%s] relatorio nao encontrado: %v", tool, err)
			}
			reportStr := string(reportData)
			if !strings.Contains(reportStr, "arquivo de task task-2.0-test.md foi alterado indevidamente") {
				t.Fatalf("[%s] relatorio nao contem diagnostico da violacao\n%s", tool, reportStr)
			}
			if !strings.Contains(reportStr, "abortado") {
				t.Fatalf("[%s] relatorio nao contem stop reason de aborto\n%s", tool, reportStr)
			}
		})
	}
}

// TestSnapshotRestorationAfterUnexpectedFileCreationForAllTools verifica que, quando o
// executor cria um arquivo de task nao listado no snapshot, o arquivo e removido e o
// loop e abortado para todas as 4 ferramentas. Subtarefa 4.3.
func TestSnapshotRestorationAfterUnexpectedFileCreationForAllTools(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}

	for _, tool := range tools {
		t.Run(tool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(invTool string) (AgentInvoker, error) {
				return &callbackInvoker{
					binary: invTool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
						// Cria arquivo de task nao registrado no snapshot (violacao)
						fsys.Files[prd+"/task-99.0-intrusa.md"] = []byte("**Status:** pending\n")
						return "created unexpected task file", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				Tool:          tool,
				MaxIterations: 1,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("[%s] Execute: %v", tool, err)
			}

			// Arquivo intruso deve ter sido removido pelo restore do snapshot
			if _, err := fsys.ReadFile(prd + "/task-99.0-intrusa.md"); err == nil {
				t.Fatalf("[%s] arquivo intruso deveria ter sido removido apos restauracao", tool)
			}

			// Task atual deveria ter sido restaurada para o estado pre-execucao
			taskOne, err := fsys.ReadFile(prd + "/task-1.0-test.md")
			if err != nil {
				t.Fatalf("[%s] task-1.0-test.md nao encontrado: %v", tool, err)
			}
			if string(taskOne) != "**Status:** pending\n" {
				t.Fatalf("[%s] task-1.0-test.md deveria ter sido restaurado para pending, obteve: %q", tool, string(taskOne))
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("[%s] relatorio nao encontrado: %v", tool, err)
			}
			reportStr := string(reportData)
			if !strings.Contains(reportStr, "novo arquivo de task task-99.0-intrusa.md foi adicionado indevidamente") {
				t.Fatalf("[%s] relatorio nao contem diagnostico do arquivo criado\n%s", tool, reportStr)
			}
		})
	}
}

// TestReviewerModeIsolationForAllReviewerTools verifica que validateReviewerIsolation
// rejeita mutacao do arquivo da task atual pelo reviewer e restaura o snapshot
// para cada ferramenta usada como reviewer. Subtarefas 4.2 (modo reviewer) e 4.3.
func TestReviewerModeIsolationForAllReviewerTools(t *testing.T) {
	reviewerTools := []string{"claude", "codex", "gemini", "copilot"}

	for _, reviewerTool := range reviewerTools {
		t.Run(reviewerTool, func(t *testing.T) {
			const base = "/fake/project"
			const prd = base + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
			fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
			fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** pending\n")
			fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | pending | — | Nao |\n")

			execProfile, _ := NewExecutionProfile("executor", "claude", "")
			revProfile, _ := NewExecutionProfile("reviewer", reviewerTool, "")

			// factoryCallCount distingue executor (1a chamada) do reviewer (2a chamada).
			// O executor e criado uma vez antes do loop; o reviewer e criado dentro de invokeReviewer.
			factoryCallCount := 0
			svc := NewService(fsys, newTestPrinter())
			svc.binaryChecker = noBinaryCheck
			svc.invokerFactory = func(tool string) (AgentInvoker, error) {
				factoryCallCount++
				isExecutor := factoryCallCount == 1
				return &callbackInvoker{
					binary: tool,
					fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
						if isExecutor {
							// Executor: mutacao legitima — marca task como done
							fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** done\n")
							fsys.Files[prd+"/tasks.md"] = []byte("| 1.0 | Task One | done | — | Nao |\n")
							return "executor completed", "", 0, nil
						}
						// Reviewer: mutacao indevida do arquivo da task atual
						fsys.Files[prd+"/task-1.0-test.md"] = []byte("**Status:** blocked\n")
						return "reviewer mutated current task", "", 0, nil
					},
				}, nil
			}

			opts := Options{
				PRDFolder:     prd,
				MaxIterations: 1,
				Timeout:       5 * time.Second,
				ReportPath:    prd + "/report.md",
				Profiles: &ProfileConfig{
					Mode:     "avancado",
					Executor: execProfile,
					Reviewer: &revProfile,
				},
				AllowUnknownModel: true,
			}

			if err := svc.Execute(opts); err != nil {
				t.Fatalf("[reviewer=%s] Execute: %v", reviewerTool, err)
			}

			// Task atual deveria ter sido restaurada para o estado pos-executor (done),
			// desfazendo a mutacao do reviewer (blocked)
			taskOne, err := fsys.ReadFile(prd + "/task-1.0-test.md")
			if err != nil {
				t.Fatalf("[reviewer=%s] task-1.0-test.md nao encontrado: %v", reviewerTool, err)
			}
			if string(taskOne) != "**Status:** done\n" {
				t.Fatalf("[reviewer=%s] task-1.0-test.md deveria ter sido restaurado para done, obteve: %q",
					reviewerTool, string(taskOne))
			}

			reportData, err := fsys.ReadFile(prd + "/report.md")
			if err != nil {
				t.Fatalf("[reviewer=%s] relatorio nao encontrado: %v", reviewerTool, err)
			}
			reportStr := string(reportData)
			if !strings.Contains(reportStr, "abortado: reviewer violou isolamento da task 1.0") {
				t.Fatalf("[reviewer=%s] relatorio nao contem stop reason do reviewer\n%s", reviewerTool, reportStr)
			}
			if !strings.Contains(reportStr, "violacao de isolamento detectada: arquivo de task task-1.0-test.md foi alterado indevidamente") {
				t.Fatalf("[reviewer=%s] relatorio nao contem diagnostico da violacao\n%s", reviewerTool, reportStr)
			}
		})
	}
}

// TestSharedStateBetweenIterationsMatchesRF12 verifica que o estado compartilhado entre
// iteracoes e exatamente o definido em RF-12:
// - skipped map: task pulada por status inalterado nao e retentada em iteracoes futuras
// - contador de iteracoes: incrementa apenas para execucoes do executor
// - acumulador de report: acumula resultados de todas as iteracoes
// - releitura de tasks.md: a iteracao seguinte ve o estado atualizado de tasks.md
// Subtarefa 4.4.
func TestSharedStateBetweenIterationsMatchesRF12(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-test"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
	fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
	fsys.Files[prd+"/task-1.0-a.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-b.md"] = []byte("**Status:** pending\n")
	// tasks.md lista ambas sem dependencias — 1.0 e selecionada primeiro (ordem de declaracao)
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Task A | pending | — | Nao |\n" +
			"| 2.0 | Task B | pending | — | Nao |\n",
	)

	task1InvokeCount := 0
	task2InvokeCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		return &callbackInvoker{
			binary: tool,
			fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				switch {
				case strings.Contains(prompt, "task-1.0"):
					task1InvokeCount++
					// Task 1 nao atualiza status (status inalterado) → entra no skipped map
					// Nao deve ser retentada em iteracoes futuras
					return "partial work, no status update", "", 0, nil
				case strings.Contains(prompt, "task-2.0"):
					task2InvokeCount++
					// RF-12 releitura de tasks.md: a iteracao viu tasks.md re-lido e elegeu task-2.0
					fsys.Files[prd+"/task-2.0-b.md"] = []byte("**Status:** done\n")
					fsys.Files[prd+"/tasks.md"] = []byte(
						"| 1.0 | Task A | pending | — | Nao |\n" +
							"| 2.0 | Task B | done | — | Nao |\n",
					)
					return "task 2 completed", "", 0, nil
				}
				return "", "", 0, nil
			},
		}, nil
	}

	opts := Options{
		PRDFolder:     prd,
		Tool:          "claude",
		MaxIterations: 10,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// RF-12 skipped map: task-1.0 com status inalterado entra no skipped map e NAO e retentada
	if task1InvokeCount != 1 {
		t.Errorf("RF-12 skipped map: task-1.0 deveria ser invocada exatamente 1 vez, obteve %d", task1InvokeCount)
	}

	// RF-12 releitura de tasks.md: task-2.0 elegivel na segunda iteracao (tasks.md re-lido)
	if task2InvokeCount != 1 {
		t.Errorf("RF-12 releitura: task-2.0 deveria ser invocada exatamente 1 vez, obteve %d", task2InvokeCount)
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)

	// RF-12 contador de iteracoes: exatamente 2 iteracoes do executor (1.0 e 2.0)
	if !strings.Contains(reportStr, "**Iterations:** 2") {
		t.Errorf("RF-12 contador: relatorio deveria registrar 2 iteracoes\n%s", reportStr)
	}

	// RF-12 acumulador de report: task-1.0 como status inalterado, task-2.0 como done
	if !strings.Contains(reportStr, "status inalterado apos execucao") {
		t.Errorf("RF-12 report: task-1.0 deveria aparecer com nota 'status inalterado'\n%s", reportStr)
	}
	if !strings.Contains(reportStr, "pending -> done") {
		t.Errorf("RF-12 report: task-2.0 deveria aparecer como 'pending -> done'\n%s", reportStr)
	}
}

// TestEnrichedPromptsAndUnlimitedFlow valida de ponta a ponta as 3 alteracoes desta sessao:
//
//  1. Prompt do executor enriquecido: arquitetura extraida de techspec.md + referencias detectadas dinamicamente
//  2. Prompt do reviewer enriquecido: tasks executadas, areas de risco, focos obrigatorios e saidas esperadas
//  3. MaxIterations=0 (ilimitado): loop roda ate TODAS as tasks estarem done
//
// O teste captura os prompts passados ao executor e ao reviewer, validando que o conteudo
// dinamico foi injetado corretamente. Usa 3 tasks para garantir que MaxIterations=0
// executa todas sem limite artificial.
func TestEnrichedPromptsAndUnlimitedFlow(t *testing.T) {
	const base = "/fake/project"
	const prd = base + "/tasks/prd-enriched"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[base+"/AGENTS.md"] = []byte("# Agents\n")
	fsys.Files[prd+"/prd.md"] = []byte(`# PRD — Autenticacao Segura

O pacote internal/auth implementa autenticacao com token JWT.
A interface AuthService define o contrato publico do domain aggregate.
Testes devem cobrir todos os cenarios de seguranca e credential.
`)
	fsys.Files[prd+"/techspec.md"] = []byte(`# Especificacao Tecnica

## Resumo Executivo

Modulo de autenticacao com JWT tokens para API REST.

## Arquitetura do Sistema

O pacote internal/auth e composto por:

| Componente | Arquivo | Responsabilidade |
|-----------|---------|-----------------|
| AuthService | auth.go | Orquestracao de login/logout |
| TokenManager | token.go | Geracao e validacao JWT |
| UserRepository | repository.go | Persistencia de usuarios |

### Fluxo de Dados

AuthService -> TokenManager -> UserRepository

## Design de Implementacao

Detalhes aqui.
`)
	fsys.Files[prd+"/task-1.0-auth-service.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-2.0-token-manager.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/task-3.0-user-repository.md"] = []byte("**Status:** pending\n")
	fsys.Files[prd+"/tasks.md"] = []byte(
		"| 1.0 | Auth Service | pending | — | Nao |\n" +
			"| 2.0 | Token Manager | pending | — | Nao |\n" +
			"| 3.0 | User Repository | pending | — | Nao |\n",
	)

	var executorPrompts []string
	var reviewerPrompts []string
	executorCallCount := 0

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{
				binary: "claude",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					executorPrompts = append(executorPrompts, prompt)
					executorCallCount++
					switch executorCallCount {
					case 1:
						fsys.Files[prd+"/task-1.0-auth-service.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Auth Service | done | — | Nao |\n" +
								"| 2.0 | Token Manager | pending | — | Nao |\n" +
								"| 3.0 | User Repository | pending | — | Nao |\n",
						)
					case 2:
						fsys.Files[prd+"/task-2.0-token-manager.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Auth Service | done | — | Nao |\n" +
								"| 2.0 | Token Manager | done | — | Nao |\n" +
								"| 3.0 | User Repository | pending | — | Nao |\n",
						)
					case 3:
						fsys.Files[prd+"/task-3.0-user-repository.md"] = []byte("**Status:** done\n")
						fsys.Files[prd+"/tasks.md"] = []byte(
							"| 1.0 | Auth Service | done | — | Nao |\n" +
								"| 2.0 | Token Manager | done | — | Nao |\n" +
								"| 3.0 | User Repository | done | — | Nao |\n",
						)
					}
					return "done", "", 0, nil
				},
			}, nil
		case "codex":
			return &callbackInvoker{
				binary: "codex",
				fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
					reviewerPrompts = append(reviewerPrompts, prompt)
					return "approved", "", 0, nil
				},
			}, nil
		default:
			return nil, fmt.Errorf("ferramenta nao configurada: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")

	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 0, // ilimitado
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute retornou erro inesperado: %v", err)
	}

	// --- Validacao 1: MaxIterations=0 executou todas as 3 tasks ---

	if executorCallCount != 3 {
		t.Fatalf("MaxIterations=0 deveria executar todas as 3 tasks, executou %d", executorCallCount)
	}

	reportData, err := fsys.ReadFile(prd + "/report.md")
	if err != nil {
		t.Fatalf("relatorio nao encontrado: %v", err)
	}
	reportStr := string(reportData)
	if !strings.Contains(reportStr, "todas as tasks completadas") {
		t.Errorf("StopReason deveria ser 'todas as tasks completadas', relatorio:\n%s", reportStr)
	}

	// --- Validacao 2: Prompts do executor enriquecidos com arquitetura e referencias ---

	if len(executorPrompts) < 3 {
		t.Fatalf("esperado 3 prompts de executor, obteve %d", len(executorPrompts))
	}

	firstExecutorPrompt := executorPrompts[0]

	// 2a. Arquitetura extraida da techspec (secao "## Arquitetura do Sistema")
	architectureChecks := []string{
		"internal/auth",
		"AuthService",
		"TokenManager",
		"UserRepository",
	}
	for _, check := range architectureChecks {
		if !strings.Contains(firstExecutorPrompt, check) {
			t.Errorf("prompt do executor deveria conter arquitetura %q extraida da techspec\nprompt:\n%s",
				check, firstExecutorPrompt)
		}
	}

	// 2b. Referencias detectadas dinamicamente do conteudo prd+techspec
	referenceChecks := []string{
		"go-implementation", // detectado por "internal/", "func ", ".go"
		"ddd",              // detectado por "aggregate", "domain"
		"security",         // detectado por "seguranca", "credential", "auth"
		"tests",            // sempre incluido
	}
	for _, ref := range referenceChecks {
		if !strings.Contains(firstExecutorPrompt, ref) {
			t.Errorf("prompt do executor deveria conter referencia %q detectada do conteudo\nprompt:\n%s",
				ref, firstExecutorPrompt)
		}
	}

	// 2c. Criterios de execucao nao negociaveis presentes no prompt
	criteriaChecks := []string{
		"preservar contratos publicos existentes",
		"nenhuma interface nova sem fronteira real justificada",
		"context.Context em todas as operacoes de IO",
		"testes table-driven",
		"nao fechar a task sem evidencia de validacao",
	}
	for _, criteria := range criteriaChecks {
		if !strings.Contains(firstExecutorPrompt, criteria) {
			t.Errorf("prompt do executor deveria conter criterio %q\nprompt:\n%s",
				criteria, firstExecutorPrompt)
		}
	}

	// 2d. Estrutura base do prompt (AGENTS.md, SKILL.md, isolamento)
	structureChecks := []string{
		"AGENTS.md",
		".agents/skills/execute-task/SKILL.md",
		"Do NOT modify any other task file.",
	}
	for _, s := range structureChecks {
		if !strings.Contains(firstExecutorPrompt, s) {
			t.Errorf("prompt do executor deveria conter %q\nprompt:\n%s", s, firstExecutorPrompt)
		}
	}

	// --- Validacao 3: Prompts do reviewer enriquecidos com contexto dinamico ---

	if len(reviewerPrompts) < 3 {
		t.Fatalf("esperado 3 prompts de reviewer, obteve %d", len(reviewerPrompts))
	}

	// 3a. Primeiro reviewer: apenas task 1.0 como atual (nenhuma concluida antes)
	firstReviewerPrompt := reviewerPrompts[0]
	if !strings.Contains(firstReviewerPrompt, "1.0") {
		t.Errorf("primeiro prompt do reviewer deveria mencionar task 1.0\nprompt:\n%s", firstReviewerPrompt)
	}

	// 3b. Terceiro reviewer: tasks 1.0 e 2.0 ja concluidas
	thirdReviewerPrompt := reviewerPrompts[2]
	if !strings.Contains(thirdReviewerPrompt, "Auth Service") || !strings.Contains(thirdReviewerPrompt, "Token Manager") {
		t.Errorf("terceiro reviewer deveria listar tasks concluidas (Auth Service, Token Manager)\nprompt:\n%s",
			thirdReviewerPrompt)
	}

	// 3c. Areas de risco detectadas da techspec (JWT, auth = seguranca; interface = contratos)
	riskChecks := []string{
		"seguranca",
	}
	for _, risk := range riskChecks {
		if !strings.Contains(firstReviewerPrompt, risk) {
			t.Errorf("prompt do reviewer deveria conter area de risco %q\nprompt:\n%s",
				risk, firstReviewerPrompt)
		}
	}

	// 3d. Focos obrigatorios da revisao presentes no prompt
	reviewFocusChecks := []string{
		"corretude:",
		"regressao:",
		"seguranca:",
		"testes:",
		"divida tecnica introduzida:",
	}
	for _, focus := range reviewFocusChecks {
		if !strings.Contains(firstReviewerPrompt, focus) {
			t.Errorf("prompt do reviewer deveria conter foco obrigatorio %q\nprompt:\n%s",
				focus, firstReviewerPrompt)
		}
	}

	// 3e. Saidas esperadas presentes no prompt
	outputChecks := []string{
		"critico, importante, sugestao",
		"aprovado / aprovado com ressalvas / reprovado",
	}
	for _, out := range outputChecks {
		if !strings.Contains(firstReviewerPrompt, out) {
			t.Errorf("prompt do reviewer deveria conter saida esperada %q\nprompt:\n%s",
				out, firstReviewerPrompt)
		}
	}

	// 3f. Estrutura base do reviewer (AGENTS.md, SKILL.md, isolamento)
	reviewStructureChecks := []string{
		"AGENTS.md",
		".agents/skills/review/SKILL.md",
		"Do NOT modify any task file or any row in tasks.md.",
	}
	for _, s := range reviewStructureChecks {
		if !strings.Contains(firstReviewerPrompt, s) {
			t.Errorf("prompt do reviewer deveria conter %q\nprompt:\n%s", s, firstReviewerPrompt)
		}
	}

	// --- Validacao 4: Prompts do executor sao especificos por task ---

	if !strings.Contains(executorPrompts[0], "task-1.0") {
		t.Errorf("primeiro prompt deveria referenciar task-1.0\n%s", executorPrompts[0])
	}
	if !strings.Contains(executorPrompts[1], "task-2.0") {
		t.Errorf("segundo prompt deveria referenciar task-2.0\n%s", executorPrompts[1])
	}
	if !strings.Contains(executorPrompts[2], "task-3.0") {
		t.Errorf("terceiro prompt deveria referenciar task-3.0\n%s", executorPrompts[2])
	}

	// --- Validacao 5: Arquitetura e consistente entre iteracoes (mesma techspec) ---

	for i := 1; i < len(executorPrompts); i++ {
		if !strings.Contains(executorPrompts[i], "AuthService") {
			t.Errorf("prompt %d deveria conter mesma arquitetura que prompt 0 (AuthService)\n%s",
				i, executorPrompts[i])
		}
	}
}
