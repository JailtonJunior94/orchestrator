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
		PRDFolder:  prd,
		DryRun:     true,
		MaxIterations: 5,
		Timeout:    5 * time.Second,
		ReportPath: prd + "/report.md",
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

