package taskloop

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// Options agrupa as opcoes do comando task-loop.
type Options struct {
	PRDFolder     string
	Tool          string
	DryRun        bool
	MaxIterations int
	Timeout       time.Duration
	ReportPath    string
	// Modo avancado: perfis por papel (nil = modo simples via Tool)
	Profiles               *ProfileConfig
	FallbackTool           string // ferramenta de fallback para validacao pre-loop (camada 2)
	AllowUnknownModel      bool   // pular validacao de compatibilidade ferramenta-modelo
	ReviewerPromptTemplate string // path do template de prompt de revisao customizado
	ExecutorFallbackModel  string // --fallback-model nativo do executor (Claude only, camada 1)
	ReviewerFallbackModel  string // --fallback-model nativo do reviewer (Claude only, camada 1)
}

// Service orquestra a execucao sequencial de tasks de um PRD folder.
type Service struct {
	fsys            fs.FileSystem
	printer         *output.Printer
	invokerFactory  func(tool string) (AgentInvoker, error)
	binaryChecker   func(AgentInvoker) error // nil = usar CheckAgentBinary
	liveOutOverride io.Writer                // nil = usar os.Stderr; permite injecao em testes
}

// NewService cria um novo Service de task-loop.
func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{
		fsys:           fsys,
		printer:        printer,
		invokerFactory: NewAgentInvoker,
	}
}

// createInvokerWithFallback cria um invoker para a ferramenta com suporte a fallback nativo.
// O fallback nativo (--fallback-model) so e suportado pelo claudeInvoker.
// Outros invokers ignoram silenciosamente o fallbackModel.
func (s *Service) createInvokerWithFallback(tool, fallbackModel string) (AgentInvoker, error) {
	inv, err := s.invokerFactory(tool)
	if err != nil {
		return nil, err
	}
	if fallbackModel != "" {
		if ci, ok := inv.(*claudeInvoker); ok {
			ci.fallbackModel = fallbackModel
		}
	}
	return inv, nil
}

// compatibilityStatusLabel retorna o label de compatibilidade para exibicao no dry-run.
// Verifica a tabela interna independentemente de AllowUnknownModel: o status reflete
// o que a tabela conhece, nao se o usuario optou por ignorar a validacao.
func compatibilityStatusLabel(table *CompatibilityTable, tool, model string) string {
	if table.IsSupported(tool, model) {
		return "✓ (compativel)"
	}
	return "✗ (incompativel)"
}

// printDryRunAdvancedHeader imprime o cabecalho do dry-run para modo avancado (RF-09, RF-12).
// Exibe: modo, perfis resolvidos com status de compatibilidade, template de revisao,
// tasks elegiveis e preview do template resolvido para a primeira task elegivel.
// Deve ser chamado uma unica vez antes do loop principal, apenas quando DryRun=true e Profiles!=nil.
func (s *Service) printDryRunAdvancedHeader(opts Options, absFolder, workDir string) {
	s.printer.DryRun("modo: avancado")

	table := NewCompatibilityTable()

	// Executor — tool / provider / model + status de compatibilidade
	exec := opts.Profiles.Executor
	execModelDisplay := exec.Model()
	if execModelDisplay == "" {
		execModelDisplay = "default"
	}
	execStatus := compatibilityStatusLabel(table, exec.Tool(), exec.Model())
	s.printer.DryRun("executor: %s / %s / %s %s", exec.Tool(), exec.Provider(), execModelDisplay, execStatus)

	// Reviewer — tool / provider / model + status de compatibilidade (quando configurado)
	if opts.Profiles.Reviewer != nil {
		rev := *opts.Profiles.Reviewer
		revModelDisplay := rev.Model()
		if revModelDisplay == "" {
			revModelDisplay = "default"
		}
		revStatus := compatibilityStatusLabel(table, rev.Tool(), rev.Model())
		s.printer.DryRun("reviewer: %s / %s / %s %s", rev.Tool(), rev.Provider(), revModelDisplay, revStatus)
	}

	// Template de revisao
	if opts.ReviewerPromptTemplate != "" {
		s.printer.DryRun("template de revisao: %s", opts.ReviewerPromptTemplate)
	} else {
		s.printer.DryRun("template de revisao: default (embutido)")
	}

	// Ler tasks.md para identificar tasks elegiveis
	tasksContent, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md"))
	if err != nil {
		return
	}
	tasks, err := ParseTasksFile(tasksContent)
	if err != nil {
		return
	}
	tasks = reconcileTaskStatuses(tasks, absFolder, s.fsys)
	eligible := FindEligible(tasks, nil)

	if len(eligible) == 0 {
		s.printer.DryRun("tasks elegiveis: nenhuma")
		return
	}

	ids := make([]string, 0, len(eligible))
	for _, t := range eligible {
		ids = append(ids, t.ID)
	}
	s.printer.DryRun("tasks elegiveis: %s", strings.Join(ids, ", "))

	// RF-12: preview do template resolvido para a primeira task elegivel
	firstTask := eligible[0]
	taskFile, err := ResolveTaskFile(absFolder, firstTask, s.fsys)
	if err != nil {
		return
	}
	relTaskFile, _ := filepath.Rel(workDir, taskFile)
	if relTaskFile == "" {
		relTaskFile = taskFile
	}
	relPRD, _ := filepath.Rel(workDir, absFolder)
	if relPRD == "" {
		relPRD = absFolder
	}

	preview, err := BuildReviewPrompt(opts.ReviewerPromptTemplate, ReviewTemplateData{
		TaskFile:  relTaskFile,
		PRDFolder: relPRD,
		TechSpec:  filepath.Join(relPRD, "techspec.md"),
		TasksFile: filepath.Join(relPRD, "tasks.md"),
		Diff:      "(dry-run: diff nao disponivel)",
	}, s.fsys)
	if err != nil {
		return
	}

	s.printer.DryRun("--- preview do template (task %s) ---", firstTask.ID)
	for _, line := range strings.Split(preview, "\n") {
		s.printer.DryRun("%s", line)
	}
	s.printer.DryRun("--- fim do preview ---")
}

// Execute roda o loop principal de execucao de tasks.
func (s *Service) Execute(opts Options) error {
	absFolder, err := filepath.Abs(opts.PRDFolder)
	if err != nil {
		return fmt.Errorf("caminho invalido %q: %w", opts.PRDFolder, err)
	}

	// Pre-flight: validar arquivos obrigatorios
	for _, required := range []string{"tasks.md", "prd.md", "techspec.md"} {
		path := filepath.Join(absFolder, required)
		if !s.fsys.Exists(path) {
			return fmt.Errorf("arquivo obrigatorio nao encontrado: %s", path)
		}
	}

	// Pre-flight: copiar perfis localmente para nao modificar o caller (fallback camada 2 pode substituir)
	if opts.Profiles != nil {
		profilesCopy := *opts.Profiles
		if opts.Profiles.Reviewer != nil {
			revCopy := *opts.Profiles.Reviewer
			profilesCopy.Reviewer = &revCopy
		}
		opts.Profiles = &profilesCopy
	}

	// Pre-flight: validar compatibilidade de perfis contra a tabela (camada 2 de fallback)
	if opts.Profiles != nil && !opts.AllowUnknownModel {
		table := NewCompatibilityTable()

		execErr := table.ValidateCombination(opts.Profiles.Executor.Tool(), opts.Profiles.Executor.Model())
		if execErr != nil {
			if opts.FallbackTool == "" {
				return fmt.Errorf("pre-flight: %w", execErr)
			}
			fallbackExec, fbErr := NewExecutionProfile("executor", opts.FallbackTool, "")
			if fbErr != nil {
				return fmt.Errorf("pre-flight: fallback-tool invalido: %w", fbErr)
			}
			s.printer.Warn("pre-flight: executor incompativel (%v) — usando fallback-tool %q", execErr, opts.FallbackTool)
			opts.Profiles.Executor = fallbackExec
		}

		if opts.Profiles.Reviewer != nil {
			revErr := table.ValidateCombination(opts.Profiles.Reviewer.Tool(), opts.Profiles.Reviewer.Model())
			if revErr != nil {
				if opts.FallbackTool == "" {
					return fmt.Errorf("pre-flight: %w", revErr)
				}
				fallbackRev, fbErr := NewExecutionProfile("reviewer", opts.FallbackTool, "")
				if fbErr != nil {
					return fmt.Errorf("pre-flight: fallback-tool invalido: %w", fbErr)
				}
				s.printer.Warn("pre-flight: reviewer incompativel (%v) — usando fallback-tool %q", revErr, opts.FallbackTool)
				opts.Profiles.Reviewer = &fallbackRev
			}
		}
	}

	// Determinar ferramenta do executor
	executorTool := opts.Tool
	if opts.Profiles != nil {
		executorTool = opts.Profiles.Executor.Tool()
	}

	// Criar invoker do executor e verificar binario
	invoker, err := s.createInvokerWithFallback(executorTool, opts.ExecutorFallbackModel)
	if err != nil {
		return err
	}

	if !opts.DryRun {
		checker := s.binaryChecker
		if checker == nil {
			checker = CheckAgentBinary
		}
		if err := checker(invoker); err != nil {
			return err
		}

		// Pre-flight: aviso antecipado de autenticacao para claude.
		// Detecta ausencia de ANTHROPIC_API_KEY e de sessao local antes de iniciar
		// o loop, evitando que a falha de auth so apareca na primeira iteracao.
		if executorTool == "claude" {
			if warn := warnClaudeAuth(); warn != "" {
				s.printer.Warn("claude auth: %s", warn)
			}
		}

		// Configurar streaming de output do agente para o terminal.
		// Permite ao usuario acompanhar progresso de agentes lentos (ex: Gemini).
		if lo, ok := invoker.(LiveOutputSetter); ok {
			liveOut := s.liveOutOverride
			if liveOut == nil {
				liveOut = os.Stderr
			}
			lo.SetLiveOutput(liveOut)
		}
	}

	// Resolver diretorio de trabalho (raiz do projeto — pai do prd folder ou cwd)
	workDir, err := resolveWorkDir(absFolder, s.fsys)
	if err != nil {
		return fmt.Errorf("erro ao resolver diretorio de trabalho: %w", err)
	}

	// Inicializar relatorio
	report := &Report{
		PRDFolder: opts.PRDFolder,
		Tool:      opts.Tool,
		StartTime: time.Now(),
	}
	if opts.Profiles != nil {
		report.Mode = opts.Profiles.Mode
		ep := opts.Profiles.Executor
		report.ExecutorProfile = &ep
		if opts.Profiles.Reviewer != nil {
			rp := *opts.Profiles.Reviewer
			report.ReviewerProfile = &rp
		}
	} else {
		report.Mode = "simples"
	}

	skipped := make(map[string]bool)
	iteration := 0

	s.printer.Info("task-loop iniciado: folder=%s tool=%s max=%d timeout=%s",
		opts.PRDFolder, opts.Tool, opts.MaxIterations, opts.Timeout)

	// Dry-run modo avancado: imprimir cabecalho com perfis, compatibilidade e preview (RF-09, RF-12)
	if opts.DryRun && opts.Profiles != nil {
		s.printDryRunAdvancedHeader(opts, absFolder, workDir)
	}

	for iteration < opts.MaxIterations {
		// Re-ler tasks.md a cada iteracao (agente pode ter atualizado)
		tasksContent, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md"))
		if err != nil {
			return fmt.Errorf("erro ao ler tasks.md: %w", err)
		}

		tasks, err := ParseTasksFile(tasksContent)
		if err != nil {
			return fmt.Errorf("erro ao parsear tasks.md: %w", err)
		}
		tasks = reconcileTaskStatuses(tasks, absFolder, s.fsys)

		eligible := FindEligible(tasks, skipped)
		if len(eligible) == 0 {
			if AllTerminal(tasks) {
				report.StopReason = "todas as tasks completadas ou em estado terminal"
			} else {
				report.StopReason = "nenhuma task elegivel (restantes estao bloqueadas, falharam ou aguardam input)"
			}
			report.FinalTasks = tasks
			break
		}

		task := eligible[0]
		iteration++ // RF-13: conta apenas iteracoes de executor; reviewer e sub-etapa

		// Resolver arquivo da task
		taskFile, err := ResolveTaskFile(absFolder, task, s.fsys)
		if err != nil {
			s.printer.Warn("iteracao %d: %v — pulando task %s", iteration, err, task.ID)
			skipped[task.ID] = true
			report.Iterations = append(report.Iterations, IterationResult{
				Sequence:   iteration,
				TaskID:     task.ID,
				Title:      task.Title,
				PreStatus:  task.Status,
				PostStatus: task.Status,
				Role:       "executor",
				Note:       fmt.Sprintf("arquivo nao encontrado: %v", err),
			})
			continue
		}

		// Ler status pre-execucao do arquivo individual
		preStatus := task.Status
		if fileStatus := readTaskStatus(taskFile, s.fsys); fileStatus != "" {
			preStatus = fileStatus
		}

		// Path relativo para o prompt
		relTaskFile, _ := filepath.Rel(workDir, taskFile)
		if relTaskFile == "" {
			relTaskFile = taskFile
		}
		relPRD, _ := filepath.Rel(workDir, absFolder)
		if relPRD == "" {
			relPRD = absFolder
		}

		prompt := BuildPrompt(relTaskFile, relPRD)

		s.printer.Step("iteracao %d: executando task %s (%s)", iteration, task.ID, task.Title)

		if opts.DryRun {
			if opts.Profiles != nil {
				// Modo avancado: exibe plano de iteracao com executor e reviewer (RF-09)
				if opts.Profiles.Reviewer != nil {
					s.printer.DryRun("iteracao %d: executaria task %s com executor, depois reviewer", iteration, task.ID)
				} else {
					s.printer.DryRun("iteracao %d: executaria task %s com executor", iteration, task.ID)
				}
			} else {
				// Modo simples: comportamento atual preservado (regressao zero)
				tool := opts.Tool
				s.printer.DryRun("invocaria %s com prompt para task %s (%s)", tool, task.ID, task.Title)
				s.printer.DryRun("task file: %s", relTaskFile)
			}
			report.Iterations = append(report.Iterations, IterationResult{
				Sequence:   iteration,
				TaskID:     task.ID,
				Title:      task.Title,
				PreStatus:  preStatus,
				PostStatus: "dry-run",
				Role:       "executor",
				Note:       "dry-run: agente nao invocado",
			})
			skipped[task.ID] = true
			continue
		}

		// Determinar model do executor
		executorModel := ""
		if opts.Profiles != nil {
			executorModel = opts.Profiles.Executor.Model()
		}

		snapshot, err := captureTaskIsolationSnapshotWithMode(absFolder, taskIsolationModeExecutor, s.fsys)
		if err != nil {
			return fmt.Errorf("erro ao capturar snapshot de isolamento da task %s: %w", task.ID, err)
		}

		// Invocar agente com timeout
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		startTime := time.Now()
		stdout, stderr, exitCode, invokeErr := invoker.Invoke(ctx, prompt, workDir, executorModel)
		elapsed := time.Since(startTime)
		cancel()

		isolationErr := validateTaskIsolation(snapshot, absFolder, task.ID, taskFile, s.fsys)
		if isolationErr != nil {
			if restoreErr := restoreTaskIsolationSnapshotAt(snapshot, absFolder, s.fsys); restoreErr != nil {
				return fmt.Errorf("violacao de isolamento na task %s: %v; falha ao restaurar snapshot: %w", task.ID, isolationErr, restoreErr)
			}
		}

		// Ler status pos-execucao
		postStatus := preStatus
		if fileStatus := readTaskStatus(taskFile, s.fsys); fileStatus != "" {
			postStatus = fileStatus
		}

		// Verificar no tasks.md atualizado apenas como fallback: quando o task file
		// nao atualizou o status (postStatus == preStatus), o agente pode ter escrito
		// diretamente em tasks.md. Quando o task file ja tem status diferente de
		// preStatus, ele e a fonte prioritaria — tasks.md nao deve sobrescreve-lo.
		if postStatus == preStatus {
			if updatedContent, readErr := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); readErr == nil {
				if updatedTasks, parseErr := ParseTasksFile(updatedContent); parseErr == nil {
					for _, ut := range updatedTasks {
						if ut.ID == task.ID && ut.Status != "" {
							postStatus = ut.Status
							break
						}
					}
				}
			}
		}

		iterResult := IterationResult{
			Sequence:    iteration,
			TaskID:      task.ID,
			Title:       task.Title,
			PreStatus:   preStatus,
			PostStatus:  postStatus,
			Duration:    elapsed,
			ExitCode:    exitCode,
			AgentOutput: stdout,
			Role:        "executor",
		}

		if isolationErr != nil {
			iterResult.Note = fmt.Sprintf("violacao de isolamento detectada: %v", isolationErr)
			s.printer.Error("iteracao %d: %s", iteration, iterResult.Note)
			report.Iterations = append(report.Iterations, iterResult)
			report.StopReason = fmt.Sprintf("abortado: agente violou isolamento da task %s", task.ID)
			if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
				if finalTasks, err := ParseTasksFile(content); err == nil {
					report.FinalTasks = finalTasks
				}
			}
			break
		}

		outcome := classifyIterationOutcome(preStatus, postStatus, exitCode, invokeErr, stdout, stderr)

		if outcome.Abort {
			guidance := authGuidance(executorTool)
			iterResult.Note = fmt.Sprintf("erro de autenticacao: %s nao esta autenticado — %s", executorTool, guidance)
			s.printer.Error("  erro de autenticacao detectado para %s — %s", executorTool, guidance)
			report.Iterations = append(report.Iterations, iterResult)
			report.StopReason = fmt.Sprintf("abortado: %s nao esta autenticado", executorTool)
			report.FinalTasks = tasks
			break
		}

		if outcome.Note != "" {
			iterResult.Note = appendNote(iterResult.Note, outcome.Note)
		}

		if invokeErr != nil {
			s.printer.Error("iteracao %d: %v", iteration, invokeErr)
		} else if exitCode != 0 {
			// Nota especifica por ferramenta para output vazio em SIGKILL (exit -1).
			// Nao entra em classifyIterationOutcome porque requer o nome da ferramenta.
			if exitCode == -1 && stdout == "" && stderr == "" {
				iterResult.Note = appendNote(iterResult.Note,
					fmt.Sprintf("saida vazia — %s pode requerer TTY ou nao suportar output em pipe", executorTool))
			}
			if stderr != "" {
				s.printer.Debug("stderr: %s", truncate(stderr, 500))
			}
		}

		if outcome.Skip {
			skipped[task.ID] = true
		}

		// === REVIEWER (RF-05, RF-06, RF-07) ===
		// Invocado quando: modo avancado com reviewer configurado, sem erro de invocacao
		// e status da task e "done". O exit code do executor nao e verificado: o agente
		// pode ter sido morto por timeout (exit -1) depois de marcar a task como done,
		// e o reviewer deve operar sobre o estado observavel da task.
		// RF-13: reviewer e sub-etapa e nao incrementa o contador de iteracoes.
		if opts.Profiles != nil && opts.Profiles.Reviewer != nil && outcome.RunReviewer {
			reviewSnapshot, err := captureTaskIsolationSnapshotWithMode(absFolder, taskIsolationModeReviewer, s.fsys)
			if err != nil {
				return fmt.Errorf("erro ao capturar snapshot de isolamento do reviewer na task %s: %w", task.ID, err)
			}
			iterResult.ReviewResult = s.invokeReviewer(opts, relTaskFile, relPRD, workDir)
			reviewIsolationErr := validateReviewerIsolation(reviewSnapshot, absFolder, task.ID, taskFile, s.fsys)
			if reviewIsolationErr != nil {
				if restoreErr := restoreTaskIsolationSnapshotAt(reviewSnapshot, absFolder, s.fsys); restoreErr != nil {
					return fmt.Errorf("violacao de isolamento do reviewer na task %s: %v; falha ao restaurar snapshot: %w", task.ID, reviewIsolationErr, restoreErr)
				}
				if iterResult.ReviewResult == nil {
					iterResult.ReviewResult = &ReviewResult{}
				}
				iterResult.ReviewResult.Note = appendNote(iterResult.ReviewResult.Note,
					fmt.Sprintf("violacao de isolamento detectada: %v", reviewIsolationErr))
				s.printer.Error("iteracao %d: reviewer violou isolamento da task %s: %v", iteration, task.ID, reviewIsolationErr)
				report.Iterations = append(report.Iterations, iterResult)
				report.StopReason = fmt.Sprintf("abortado: reviewer violou isolamento da task %s", task.ID)
				if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
					if finalTasks, err := ParseTasksFile(content); err == nil {
						report.FinalTasks = finalTasks
					}
				}
				break
			}
		}

		s.printer.Info("  resultado: %s -> %s (exit=%d, duracao=%s)",
			preStatus, postStatus, exitCode, elapsed.Truncate(time.Second))

		report.Iterations = append(report.Iterations, iterResult)
	}

	if iteration >= opts.MaxIterations && report.StopReason == "" {
		report.StopReason = fmt.Sprintf("limite de iteracoes atingido (%d)", opts.MaxIterations)
		// Ler estado final das tasks
		if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
			if tasks, err := ParseTasksFile(content); err == nil {
				report.FinalTasks = tasks
			}
		}
	}

	report.EndTime = time.Now()

	// Escrever relatorio
	reportContent := report.Render()
	if err := s.fsys.WriteFile(opts.ReportPath, reportContent); err != nil {
		return fmt.Errorf("erro ao escrever relatorio: %w", err)
	}

	s.printer.Info("task-loop finalizado: %s", report.StopReason)
	s.printer.Info("relatorio salvo em: %s", opts.ReportPath)

	return nil
}

// invokeReviewer invoca o reviewer apos execucao bem-sucedida do executor.
// Cria contexto proprio com o mesmo timeout do executor.
// Retorna ReviewResult com o resultado da revisao ou nota de erro.
func (s *Service) invokeReviewer(opts Options, relTaskFile, relPRD, workDir string) *ReviewResult {
	reviewerInvoker, err := s.createInvokerWithFallback(
		opts.Profiles.Reviewer.Tool(),
		opts.ReviewerFallbackModel,
	)
	if err != nil {
		return &ReviewResult{
			Note: fmt.Sprintf("erro ao criar invoker do reviewer: %v", err),
		}
	}

	diff := captureGitDiff(context.Background(), workDir)
	reviewPrompt, promptErr := BuildReviewPrompt(
		opts.ReviewerPromptTemplate,
		ReviewTemplateData{
			TaskFile:  relTaskFile,
			PRDFolder: relPRD,
			TechSpec:  filepath.Join(relPRD, "techspec.md"),
			TasksFile: filepath.Join(relPRD, "tasks.md"),
			Diff:      diff,
		},
		s.fsys,
	)
	if promptErr != nil {
		return &ReviewResult{
			Note: fmt.Sprintf("erro ao construir prompt de revisao: %v", promptErr),
		}
	}

	rctx, rcancel := context.WithTimeout(context.Background(), opts.Timeout)
	rStart := time.Now()
	rStdout, _, rExitCode, rErr := reviewerInvoker.Invoke(
		rctx,
		reviewPrompt,
		workDir,
		opts.Profiles.Reviewer.Model(),
	)
	rElapsed := time.Since(rStart)
	rcancel()

	reviewResult := &ReviewResult{
		Duration: rElapsed,
		ExitCode: rExitCode,
		Output:   rStdout,
	}
	if rErr != nil {
		reviewResult.Note = fmt.Sprintf("erro de invocacao do reviewer: %v", rErr)
	} else if rExitCode != 0 {
		reviewResult.Note = "reviewer reportou problemas criticos"
	}

	return reviewResult
}

// iterationOutcome representa a decisao tomada apos uma invocacao do agente.
// Produzida por classifyIterationOutcome — sem side effects.
type iterationOutcome struct {
	Skip        bool   // task deve ser ignorada nesta execucao
	Abort       bool   // loop deve ser abortado (ex: erro de autenticacao)
	Note        string // descricao do motivo (acumulavel via appendNote)
	RunReviewer bool   // reviewer deve ser invocado
}

// classifyIterationOutcome determina o estado final de uma iteracao a partir
// de dados observaveis, sem depender da ferramenta nem produzir side effects.
//
// Regras (em ordem de precedencia):
//   - invokeErr != nil                       → Skip=true, Note="erro de invocacao: ..."
//   - exitCode != 0 && isAuthError(combined) → Abort=true (retorno antecipado)
//   - exitCode != 0 (sem auth error)         → Note="agente saiu com codigo N"
//   - invokeErr == nil && postStatus=="done" → RunReviewer=true
//   - postStatus == preStatus                → Skip=true, Note=appended "status inalterado..."
//   - postStatus em {failed,blocked,needs_input} → Skip=true
func classifyIterationOutcome(
	preStatus, postStatus string,
	exitCode int,
	invokeErr error,
	stdout, stderr string,
) iterationOutcome {
	outcome := iterationOutcome{}

	if invokeErr != nil {
		outcome.Skip = true
		outcome.Note = fmt.Sprintf("erro de invocacao: %v", invokeErr)
	} else if exitCode != 0 {
		combined := stdout + stderr
		if isAuthError(combined) {
			return iterationOutcome{Abort: true, Note: "erro de autenticacao"}
		}
		outcome.Note = fmt.Sprintf("agente saiu com codigo %d", exitCode)
	}

	// RunReviewer: apenas quando invokeErr == nil e task concluida
	if invokeErr == nil && postStatus == "done" {
		outcome.RunReviewer = true
	}

	// Status inalterado → skip para prevenir loop infinito
	if postStatus == preStatus {
		outcome.Note = appendNote(outcome.Note, "status inalterado apos execucao; pulando")
		outcome.Skip = true
	}

	// Status terminal nao-done → skip
	if postStatus == "failed" || postStatus == "blocked" || postStatus == "needs_input" {
		outcome.Skip = true
	}

	return outcome
}

// resolveWorkDir tenta encontrar a raiz do projeto (diretorio que contem go.mod, .git, ou AGENTS.md).
// Recebe fsys para manter testabilidade com FakeFileSystem.
func resolveWorkDir(prdFolder string, fsys fs.FileSystem) (string, error) {
	dir, err := filepath.Abs(prdFolder)
	if err != nil {
		return "", err
	}
	for {
		for _, marker := range []string{".git", "go.mod", "AGENTS.md"} {
			markerPath := filepath.Join(dir, marker)
			if fsys.Exists(markerPath) {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Abs(prdFolder)
		}
		dir = parent
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func appendNote(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + "; " + addition
}
