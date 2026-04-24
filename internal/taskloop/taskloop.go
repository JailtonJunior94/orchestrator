package taskloop

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	// UI e capacidades do terminal, resolvidas na CLI.
	RequestedUIMode      UIMode
	EffectiveUIMode      UIMode
	TerminalCapabilities TerminalCapabilities
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
	observer        LoopObserver
	events          *eventPublisher
	invokerFactory  func(tool string) (AgentInvoker, error)
	binaryChecker   func(AgentInvoker) error // nil = usar CheckAgentBinary
	liveOutOverride io.Writer                // nil = usar os.Stderr; permite injecao em testes
	heartbeatTicker heartbeatTickerFactory
	heartbeatEvery  time.Duration
}

// NewService cria um novo Service de task-loop.
func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return NewServiceWithObserver(fsys, printer, nil)
}

// NewServiceWithObserver cria um Service com observer injetado.
func NewServiceWithObserver(fsys fs.FileSystem, printer *output.Printer, observer LoopObserver) *Service {
	return &Service{
		fsys:            fsys,
		printer:         printer,
		observer:        observer,
		invokerFactory:  NewAgentInvoker,
		heartbeatTicker: newRealtimeHeartbeatTicker,
		heartbeatEvery:  defaultHeartbeatInterval,
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

// printDryRunAdvancedHeaderWithPrinter imprime o cabecalho do dry-run para modo avancado (RF-09, RF-12).
// Exibe: modo, perfis resolvidos com status de compatibilidade, template de revisao,
// tasks elegiveis e preview do template resolvido para a primeira task elegivel.
// Deve ser chamado uma unica vez antes do loop principal, apenas quando DryRun=true e Profiles!=nil.
func (s *Service) printDryRunAdvancedHeaderWithPrinter(printer *output.Printer, opts Options, absFolder, workDir string) {
	s.printDryRun(printer, "modo: avancado")

	table := NewCompatibilityTable()

	// Executor — tool / provider / model + status de compatibilidade
	exec := opts.Profiles.Executor
	execModelDisplay := exec.Model()
	if execModelDisplay == "" {
		execModelDisplay = "default"
	}
	execStatus := compatibilityStatusLabel(table, exec.Tool(), exec.Model())
	s.printDryRun(printer, "executor: %s / %s / %s %s", exec.Tool(), exec.Provider(), execModelDisplay, execStatus)

	// Reviewer — tool / provider / model + status de compatibilidade (quando configurado)
	if opts.Profiles.Reviewer != nil {
		rev := *opts.Profiles.Reviewer
		revModelDisplay := rev.Model()
		if revModelDisplay == "" {
			revModelDisplay = "default"
		}
		revStatus := compatibilityStatusLabel(table, rev.Tool(), rev.Model())
		s.printDryRun(printer, "reviewer: %s / %s / %s %s", rev.Tool(), rev.Provider(), revModelDisplay, revStatus)
	}

	// Template de revisao
	if opts.ReviewerPromptTemplate != "" {
		s.printDryRun(printer, "template de revisao: %s", opts.ReviewerPromptTemplate)
	} else {
		s.printDryRun(printer, "template de revisao: default (embutido)")
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
		s.printDryRun(printer, "tasks elegiveis: nenhuma")
		return
	}

	ids := make([]string, 0, len(eligible))
	for _, t := range eligible {
		ids = append(ids, t.ID)
	}
	s.printDryRun(printer, "tasks elegiveis: %s", strings.Join(ids, ", "))

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

	s.printDryRun(printer, "--- preview do template (task %s) ---", firstTask.ID)
	for _, line := range strings.Split(preview, "\n") {
		s.printDryRun(printer, "%s", line)
	}
	s.printDryRun(printer, "--- fim do preview ---")
}

// Execute roda o loop principal de execucao de tasks.
func (s *Service) Execute(opts Options) error {
	requestedUIMode, err := normalizeRequestedUIMode(opts.RequestedUIMode)
	if err != nil {
		return err
	}
	effectiveUIMode, err := normalizeEffectiveUIMode(opts.EffectiveUIMode, requestedUIMode, opts.TerminalCapabilities)
	if err != nil {
		return err
	}
	directPrinter := s.directPrinter(effectiveUIMode)

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
			directPrinter.Warn("pre-flight: executor incompativel (%v) — usando fallback-tool %q", execErr, opts.FallbackTool)
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
				directPrinter.Warn("pre-flight: reviewer incompativel (%v) — usando fallback-tool %q", revErr, opts.FallbackTool)
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
				directPrinter.Warn("claude auth: %s", warn)
			}
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

	session, err := NewLoopSession(report.Mode, opts.MaxIterations, effectiveUIMode == UIModeTUI, 0, report.StartTime)
	if err != nil {
		return fmt.Errorf("inicializar sessao do task-loop: %w", err)
	}
	events := newEventPublisher(session, s.resolveObserver(effectiveUIMode, session, opts.TerminalCapabilities), s.printer)
	s.events = events
	defer func() {
		s.events = nil
	}()
	if err := events.start(); err != nil {
		return fmt.Errorf("inicializar observer do task-loop: %w", err)
	}
	if err := events.consume(LoopEvent{
		Time:    report.StartTime,
		Kind:    EventSessionStarted,
		Message: "sessao iniciada",
	}); err != nil {
		return fmt.Errorf("publicar evento de inicio da sessao: %w", err)
	}
	if err := events.consume(LoopEvent{
		Time:    report.StartTime,
		Kind:    EventProgressUpdated,
		Message: "pre-flight concluido",
	}); err != nil {
		return fmt.Errorf("publicar evento de pre-flight: %w", err)
	}

	skipped := make(map[string]bool)
	iteration := 0

	directPrinter.Info("task-loop iniciado: folder=%s tool=%s max=%d timeout=%s ui=%s",
		opts.PRDFolder, opts.Tool, opts.MaxIterations, opts.Timeout, effectiveUIMode)

	// Dry-run modo avancado: imprimir cabecalho com perfis, compatibilidade e preview (RF-09, RF-12)
	if opts.DryRun && opts.Profiles != nil {
		s.printDryRunAdvancedHeaderWithPrinter(directPrinter, opts, absFolder, workDir)
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
		progressEvent, err := newProgressEvent(time.Now(), tasks)
		if err != nil {
			return fmt.Errorf("calcular progresso da sessao: %w", err)
		}
		if err := events.consume(progressEvent); err != nil {
			return fmt.Errorf("publicar progresso da sessao: %w", err)
		}

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
			directPrinter.Warn("iteracao %d: %v — pulando task %s", iteration, err, task.ID)
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
		taskRef, err := NewTaskRef(task.ID, task.Title)
		if err != nil {
			return fmt.Errorf("criar referencia da task %s: %w", task.ID, err)
		}
		iterationTool, err := resolveToolName(executorTool)
		if err != nil {
			return fmt.Errorf("resolver ferramenta do executor: %w", err)
		}

		if err := events.consume(LoopEvent{
			Time:       time.Now(),
			Kind:       EventIterationSelected,
			Iteration:  iteration,
			Role:       RoleExecutor,
			Task:       taskRef,
			Tool:       iterationTool,
			Phase:      PhasePreparing,
			Message:    fmt.Sprintf("iteracao %d selecionada", iteration),
			PreStatus:  preStatus,
			PostStatus: preStatus,
		}); err != nil {
			return fmt.Errorf("publicar selecao da iteracao %d: %w", iteration, err)
		}

		directPrinter.Step("iteracao %d: executando task %s (%s)", iteration, task.ID, task.Title)

		if opts.DryRun {
			if opts.Profiles != nil {
				// Modo avancado: exibe plano de iteracao com executor e reviewer (RF-09)
				if opts.Profiles.Reviewer != nil {
					directPrinter.DryRun("iteracao %d: executaria task %s com executor, depois reviewer", iteration, task.ID)
				} else {
					directPrinter.DryRun("iteracao %d: executaria task %s com executor", iteration, task.ID)
				}
			} else {
				// Modo simples: comportamento atual preservado (regressao zero)
				tool := opts.Tool
				directPrinter.DryRun("invocaria %s com prompt para task %s (%s)", tool, task.ID, task.Title)
				directPrinter.DryRun("task file: %s", relTaskFile)
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
			if err := events.consume(LoopEvent{
				Time:       time.Now(),
				Kind:       EventPhaseChanged,
				Iteration:  iteration,
				Role:       RoleExecutor,
				Task:       taskRef,
				Tool:       iterationTool,
				Phase:      PhaseFailed,
				Message:    "dry-run: iteracao encerrada sem invocar agente",
				PreStatus:  preStatus,
				PostStatus: "dry-run",
			}); err != nil {
				return fmt.Errorf("publicar encerramento do dry-run da iteracao %d: %w", iteration, err)
			}
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
		gitSnapshotBefore, _ := captureGitStatusSnapshot(context.Background(), workDir)
		// Invocar agente com timeout
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		startTime := time.Now()
		invokeResult := s.invokeObservedAgent(ctx, invoker, prompt, workDir, executorModel, invocationMetadata{
			iteration:  iteration,
			task:       taskRef,
			tool:       iterationTool,
			role:       RoleExecutor,
			phase:      PhaseRunning,
			preStatus:  preStatus,
			postStatus: preStatus,
			uiMode:     effectiveUIMode,
		})
		elapsed := time.Since(startTime)
		cancel()
		if invokeResult.eventErr != nil {
			return fmt.Errorf("publicar lifecycle do executor na iteracao %d: %w", iteration, invokeResult.eventErr)
		}

		stdout, stderr, exitCode, invokeErr := invokeResult.stdout, invokeResult.stderr, invokeResult.exitCode, invokeResult.err

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

		// Tambem verificar no tasks.md atualizado
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
		executorFailure := mapLoopFailure(iterationTool, exitCode, stdout, stderr, invokeErr, postStatus)

		if isolationErr != nil {
			failure := NewLoopFailure(ErrorTaskIsolationViolation, "violacao de isolamento detectada", isolationErr)
			failure.Tool = iterationTool
			failure.TaskID = task.ID
			failure.Iteration = iteration
			if err := events.consume(LoopEvent{
				Time:       time.Now(),
				Kind:       EventFailureObserved,
				Iteration:  iteration,
				Role:       RoleExecutor,
				Task:       taskRef,
				Tool:       iterationTool,
				Phase:      PhaseFailed,
				Message:    failure.Message,
				ErrorCode:  failure.Code,
				PreStatus:  preStatus,
				PostStatus: postStatus,
				Failure:    failure,
			}); err != nil {
				return fmt.Errorf("publicar falha de isolamento da iteracao %d: %w", iteration, err)
			}
			iterResult.Note = fmt.Sprintf("violacao de isolamento detectada: %v", isolationErr)
			directPrinter.Error("iteracao %d: %s", iteration, iterResult.Note)
			report.Iterations = append(report.Iterations, iterResult)
			report.StopReason = fmt.Sprintf("abortado: agente violou isolamento da task %s", task.ID)
			if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
				if finalTasks, err := ParseTasksFile(content); err == nil {
					report.FinalTasks = finalTasks
				}
			}
			break
		}

		if invokeErr != nil {
			iterResult.Note = fmt.Sprintf("erro de invocacao: %v", invokeErr)
			directPrinter.Error("iteracao %d: %v", iteration, invokeErr)
			skipped[task.ID] = true
		} else if exitCode != 0 {
			combined := stdout + stderr
			if isAuthError(combined) {
				guidance := authGuidance(executorTool)
				iterResult.Note = fmt.Sprintf("erro de autenticacao: %s nao esta autenticado — %s", executorTool, guidance)
				directPrinter.Error("  erro de autenticacao detectado para %s — %s", executorTool, guidance)
				report.Iterations = append(report.Iterations, iterResult)
				report.StopReason = fmt.Sprintf("abortado: %s nao esta autenticado", executorTool)
				report.FinalTasks = tasks
				failure := NewLoopFailure(ErrorToolAuthRequired, fmt.Sprintf("%s nao esta autenticado", executorTool), nil)
				failure.Tool = iterationTool
				failure.TaskID = task.ID
				failure.Iteration = iteration
				if err := events.consume(LoopEvent{
					Time:       time.Now(),
					Kind:       EventFailureObserved,
					Iteration:  iteration,
					Role:       RoleExecutor,
					Task:       taskRef,
					Tool:       iterationTool,
					Phase:      PhaseAuthRequired,
					Message:    failure.Message,
					ErrorCode:  failure.Code,
					PreStatus:  preStatus,
					PostStatus: postStatus,
					Failure:    failure,
				}); err != nil {
					return fmt.Errorf("publicar falha de autenticacao da iteracao %d: %w", iteration, err)
				}
				break
			}
			iterResult.Note = fmt.Sprintf("agente saiu com codigo %d", exitCode)
			// Detectar output vazio em execucao terminada forcadamente (exit -1 = SIGKILL por timeout).
			// Indica que a CLI pode nao suportar escrita em pipe sem TTY (ex: codex).
			if exitCode == -1 && stdout == "" && stderr == "" {
				iterResult.Note = appendNote(iterResult.Note,
					fmt.Sprintf("saida vazia — %s pode requerer TTY ou nao suportar output em pipe", executorTool))
			}
			if stderr != "" {
				directPrinter.Debug("stderr: %s", truncate(stderr, 500))
			}
		}

		// === REVIEWER (RF-05, RF-06, RF-07) ===
		// Invocado quando: modo avancado com reviewer configurado, sem erro de invocacao
		// e status da task e "done". O exit code do executor nao e verificado: o agente
		// pode ter sido morto por timeout (exit -1) depois de marcar a task como done,
		// e o reviewer deve operar sobre o estado observavel da task.
		// RF-13: reviewer e sub-etapa e nao incrementa o contador de iteracoes.
		if opts.Profiles != nil && opts.Profiles.Reviewer != nil &&
			invokeErr == nil && postStatus == "done" {
			gitSnapshotAfter, _ := captureGitStatusSnapshot(context.Background(), workDir)
			reviewSnapshot, err := captureTaskIsolationSnapshotWithMode(absFolder, taskIsolationModeReviewer, s.fsys)
			if err != nil {
				return fmt.Errorf("erro ao capturar snapshot de isolamento do reviewer na task %s: %w", task.ID, err)
			}
			reviewerTool, err := resolveToolName(opts.Profiles.Reviewer.Tool())
			if err != nil {
				return fmt.Errorf("resolver ferramenta do reviewer na iteracao %d: %w", iteration, err)
			}
			reviewResult, reviewErr := s.invokeReviewer(
				iteration,
				taskRef,
				preStatus,
				postStatus,
				opts,
				relTaskFile,
				relPRD,
				workDir,
				changedGitPathsSince(gitSnapshotBefore, gitSnapshotAfter),
			)
			if reviewErr != nil {
				return fmt.Errorf("executar reviewer na iteracao %d: %w", iteration, reviewErr)
			}
			iterResult.ReviewResult = reviewResult
			reviewIsolationErr := validateReviewerIsolation(reviewSnapshot, absFolder, task.ID, taskFile, s.fsys)
			if reviewIsolationErr != nil {
				if restoreErr := restoreTaskIsolationSnapshotAt(reviewSnapshot, absFolder, s.fsys); restoreErr != nil {
					return fmt.Errorf("violacao de isolamento do reviewer na task %s: %v; falha ao restaurar snapshot: %w", task.ID, reviewIsolationErr, restoreErr)
				}
				failure := NewLoopFailure(ErrorTaskIsolationViolation, "violacao de isolamento detectada", reviewIsolationErr)
				failure.Tool = reviewerTool
				failure.TaskID = task.ID
				failure.Iteration = iteration
				if err := events.consume(LoopEvent{
					Time:       time.Now(),
					Kind:       EventFailureObserved,
					Iteration:  iteration,
					Role:       RoleReviewer,
					Task:       taskRef,
					Tool:       reviewerTool,
					Phase:      PhaseFailed,
					Message:    failure.Message,
					ErrorCode:  failure.Code,
					PreStatus:  preStatus,
					PostStatus: postStatus,
					Failure:    failure,
				}); err != nil {
					return fmt.Errorf("publicar falha de isolamento do reviewer na iteracao %d: %w", iteration, err)
				}
				if iterResult.ReviewResult == nil {
					iterResult.ReviewResult = &ReviewResult{}
				}
				iterResult.ReviewResult.Note = appendNote(iterResult.ReviewResult.Note,
					fmt.Sprintf("violacao de isolamento detectada: %v", reviewIsolationErr))
				directPrinter.Error("iteracao %d: reviewer violou isolamento da task %s: %v", iteration, task.ID, reviewIsolationErr)
				report.Iterations = append(report.Iterations, iterResult)
				report.StopReason = fmt.Sprintf("abortado: reviewer violou isolamento da task %s", task.ID)
				if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
					if finalTasks, err := ParseTasksFile(content); err == nil {
						report.FinalTasks = finalTasks
					}
				}
				break
			}
			if iterResult.ReviewResult != nil && iterResult.ReviewResult.Succeeded {
				reviewerMessage := "reviewer finalizado"
				if strings.TrimSpace(iterResult.ReviewResult.Note) != "" {
					reviewerMessage = iterResult.ReviewResult.Note
				}
				if err := s.publishRoleFinished(
					iteration,
					taskRef,
					reviewerTool,
					RoleReviewer,
					PhaseDone,
					preStatus,
					postStatus,
					iterResult.ReviewResult.ExitCode,
					reviewerMessage,
				); err != nil {
					return fmt.Errorf("publicar encerramento do reviewer na iteracao %d: %w", iteration, err)
				}
			}
		} else if executorFailure != nil {
			executorFailure.Tool = iterationTool
			executorFailure.TaskID = task.ID
			executorFailure.Iteration = iteration
			if err := events.consume(LoopEvent{
				Time:       time.Now(),
				Kind:       EventFailureObserved,
				Iteration:  iteration,
				Role:       RoleExecutor,
				Task:       taskRef,
				Tool:       iterationTool,
				Phase:      executorFailure.Code.defaultPhase(),
				Message:    executorFailure.Message,
				ErrorCode:  executorFailure.Code,
				ExitCode:   exitCode,
				PreStatus:  preStatus,
				PostStatus: postStatus,
				Failure:    executorFailure,
			}); err != nil {
				return fmt.Errorf("publicar encerramento do executor na iteracao %d: %w", iteration, err)
			}
		} else if err := events.consume(LoopEvent{
			Time:       time.Now(),
			Kind:       EventPhaseChanged,
			Iteration:  iteration,
			Role:       RoleExecutor,
			Task:       taskRef,
			Tool:       iterationTool,
			Phase:      PhaseDone,
			Message:    buildRoleCompletedMessage("executor", invokeErr, exitCode, preStatus, postStatus),
			ExitCode:   exitCode,
			PreStatus:  preStatus,
			PostStatus: postStatus,
		}); err != nil {
			return fmt.Errorf("publicar encerramento do executor na iteracao %d: %w", iteration, err)
		}

		// Se status nao mudou, skip para evitar loop infinito
		if postStatus == preStatus {
			iterResult.Note = appendNote(iterResult.Note, "status inalterado apos execucao; pulando")
			skipped[task.ID] = true
		}

		// Se status terminal nao-done, skip
		if postStatus == "failed" || postStatus == "blocked" || postStatus == "needs_input" {
			skipped[task.ID] = true
		}

		directPrinter.Info("  resultado: %s -> %s (exit=%d, duracao=%s)",
			preStatus, postStatus, exitCode, elapsed.Truncate(time.Second))

		report.Iterations = append(report.Iterations, iterResult)
		if updatedContent, readErr := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); readErr == nil {
			if updatedTasks, parseErr := ParseTasksFile(updatedContent); parseErr == nil {
				progressEvent, eventErr := newProgressEvent(time.Now(), reconcileTaskStatuses(updatedTasks, absFolder, s.fsys))
				if eventErr != nil {
					return fmt.Errorf("calcular progresso apos iteracao %d: %w", iteration, eventErr)
				}
				if err := events.consume(progressEvent); err != nil {
					return fmt.Errorf("publicar progresso apos iteracao %d: %w", iteration, err)
				}
			}
		}
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
	if len(report.FinalTasks) == 0 {
		if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
			if tasks, err := ParseTasksFile(content); err == nil {
				report.FinalTasks = tasks
			}
		}
	}
	if len(report.FinalTasks) > 0 {
		progressEvent, err := newProgressEvent(time.Now(), report.FinalTasks)
		if err != nil {
			return fmt.Errorf("calcular progresso final da sessao: %w", err)
		}
		if err := events.consume(progressEvent); err != nil {
			return fmt.Errorf("publicar progresso final da sessao: %w", err)
		}
	}
	report.Summary = session.FinalSummary(report.StopReason, opts.ReportPath)

	// Escrever relatorio
	reportContent := report.Render()
	if err := s.fsys.WriteFile(opts.ReportPath, reportContent); err != nil {
		return fmt.Errorf("erro ao escrever relatorio: %w", err)
	}
	if err := events.consume(LoopEvent{
		Time:    time.Now(),
		Kind:    EventSessionFinished,
		Message: report.StopReason,
	}); err != nil {
		return fmt.Errorf("publicar encerramento da sessao: %w", err)
	}
	events.finish(report.StopReason, opts.ReportPath)
	s.printPersistentFinalSummary(effectiveUIMode, report.Summary)

	directPrinter.Info("task-loop finalizado: %s", report.StopReason)
	directPrinter.Info("relatorio salvo em: %s", opts.ReportPath)

	return nil
}

// invokeReviewer invoca o reviewer apos execucao bem-sucedida do executor.
// Cria contexto proprio com o mesmo timeout do executor.
// Retorna ReviewResult com o resultado da revisao ou nota de erro.
func (s *Service) invokeReviewer(
	iteration int,
	task TaskRef,
	preStatus string,
	postStatus string,
	opts Options,
	relTaskFile string,
	relPRD string,
	workDir string,
	reviewPaths []string,
) (*ReviewResult, error) {
	reviewerTool, err := resolveToolName(opts.Profiles.Reviewer.Tool())
	if err != nil {
		return nil, fmt.Errorf("resolver ferramenta do reviewer: %w", err)
	}
	if err := s.publishRoleStarted(iteration, task, reviewerTool, RoleReviewer, PhaseReviewing, preStatus, postStatus); err != nil {
		return nil, fmt.Errorf("publicar inicio do reviewer: %w", err)
	}

	reviewerInvoker, err := s.createInvokerWithFallback(
		opts.Profiles.Reviewer.Tool(),
		opts.ReviewerFallbackModel,
	)
	if err != nil {
		return s.reviewerFailureResult(
			iteration,
			task,
			reviewerTool,
			preStatus,
			postStatus,
			fmt.Sprintf("erro ao criar invoker do reviewer: %v", err),
			NewLoopFailure(ErrorToolExecutionFailed, "falha ao preparar reviewer", err),
		)
	}

	diff := captureGitDiff(context.Background(), workDir, reviewPaths)
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
		return s.reviewerFailureResult(
			iteration,
			task,
			reviewerTool,
			preStatus,
			postStatus,
			fmt.Sprintf("erro ao construir prompt de revisao: %v", promptErr),
			NewLoopFailure(ErrorToolExecutionFailed, "falha ao preparar prompt de revisao", promptErr),
		)
	}

	rctx, rcancel := context.WithTimeout(context.Background(), opts.Timeout)
	rStart := time.Now()
	invokeResult := s.invokeObservedAgent(rctx, reviewerInvoker, reviewPrompt, workDir, opts.Profiles.Reviewer.Model(), invocationMetadata{
		iteration:  iteration,
		task:       task,
		tool:       reviewerTool,
		role:       RoleReviewer,
		phase:      PhaseReviewing,
		preStatus:  preStatus,
		postStatus: postStatus,
		uiMode:     opts.EffectiveUIMode,
	})
	rElapsed := time.Since(rStart)
	rcancel()
	if invokeResult.eventErr != nil {
		return nil, fmt.Errorf("publicar lifecycle do reviewer: %w", invokeResult.eventErr)
	}

	rStdout, rStderr, rExitCode, rErr := invokeResult.stdout, invokeResult.stderr, invokeResult.exitCode, invokeResult.err

	reviewResult := &ReviewResult{
		Duration:  rElapsed,
		ExitCode:  rExitCode,
		Output:    rStdout,
		Succeeded: true,
	}
	if failure := mapLoopFailure(reviewerTool, rExitCode, rStdout, rStderr, rErr, postStatus); failure != nil {
		reviewResult.Succeeded = false
		failure.Tool = reviewerTool
		failure.TaskID = task.ID
		failure.Iteration = iteration
		if rErr == nil && rExitCode != 0 && failure.Code == ErrorToolExecutionFailed {
			reviewResult.Note = "reviewer reportou problemas criticos"
		} else {
			reviewResult.Note = failure.Message
		}
		if err := s.publishEvent(LoopEvent{
			Time:       time.Now(),
			Kind:       EventFailureObserved,
			Iteration:  iteration,
			Role:       RoleReviewer,
			Task:       task,
			Tool:       reviewerTool,
			Phase:      failure.Code.defaultPhase(),
			Message:    failure.Message,
			ErrorCode:  failure.Code,
			ExitCode:   rExitCode,
			PreStatus:  preStatus,
			PostStatus: postStatus,
			Failure:    failure,
		}); err != nil {
			return nil, fmt.Errorf("publicar encerramento do reviewer: %w", err)
		}
	} else if rErr != nil {
		reviewResult.Succeeded = false
		reviewResult.Note = fmt.Sprintf("erro de invocacao do reviewer: %v", rErr)
	} else if rExitCode != 0 {
		reviewResult.Succeeded = false
		reviewResult.Note = "reviewer reportou problemas criticos"
	}
	return reviewResult, nil
}

func (s *Service) reviewerFailureResult(
	iteration int,
	task TaskRef,
	tool ToolName,
	preStatus string,
	postStatus string,
	note string,
	failure *LoopFailure,
) (*ReviewResult, error) {
	if failure == nil {
		failure = NewLoopFailure(ErrorToolExecutionFailed, "falha ao preparar reviewer", nil)
	}
	failure.Tool = tool
	failure.TaskID = task.ID
	failure.Iteration = iteration
	if err := s.publishEvent(LoopEvent{
		Time:       time.Now(),
		Kind:       EventFailureObserved,
		Iteration:  iteration,
		Role:       RoleReviewer,
		Task:       task,
		Tool:       tool,
		Phase:      failure.Code.defaultPhase(),
		Message:    failure.Message,
		ErrorCode:  failure.Code,
		PreStatus:  preStatus,
		PostStatus: postStatus,
		Failure:    failure,
	}); err != nil {
		return nil, fmt.Errorf("publicar falha do reviewer: %w", err)
	}
	return &ReviewResult{
		Note:      note,
		Succeeded: false,
	}, nil
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

type observedInvocation struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	eventErr error
}

type invocationMetadata struct {
	iteration  int
	task       TaskRef
	tool       ToolName
	role       AgentRole
	phase      AgentPhase
	preStatus  string
	postStatus string
	uiMode     UIMode
}

func (s *Service) invokeObservedAgent(
	ctx context.Context,
	invoker AgentInvoker,
	prompt string,
	workDir string,
	model string,
	meta invocationMetadata,
) observedInvocation {
	recorder := &invocationErrorRecorder{}
	baseLiveOut := s.liveOutOverride
	if meta.uiMode == UIModeTUI {
		baseLiveOut = nil
	} else if baseLiveOut == nil {
		baseLiveOut = os.Stderr
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	defer stopHeartbeat()

	var heartbeatDone <-chan error
	var heartbeatMu sync.Mutex
	started := false
	startLifecycle := func() {
		heartbeatMu.Lock()
		defer heartbeatMu.Unlock()
		if started {
			return
		}
		started = true
		recorder.Record(s.publishEvent(LoopEvent{
			Time:       time.Now(),
			Kind:       EventPhaseChanged,
			Iteration:  meta.iteration,
			Role:       meta.role,
			Task:       meta.task,
			Tool:       meta.tool,
			Phase:      meta.phase,
			Message:    "subprocesso iniciado",
			PreStatus:  meta.preStatus,
			PostStatus: meta.postStatus,
		}))
		heartbeatDone = startHeartbeat(
			heartbeatCtx,
			s.heartbeatTicker,
			s.heartbeatEvery,
			func(at time.Time) error {
				return s.publishEvent(LoopEvent{
					Time:       at,
					Kind:       EventHeartbeatObserved,
					Iteration:  meta.iteration,
					Role:       meta.role,
					Task:       meta.task,
					Tool:       meta.tool,
					Phase:      meta.phase,
					Message:    "atividade operacional em andamento",
					PreStatus:  meta.preStatus,
					PostStatus: meta.postStatus,
				})
			},
		)
	}

	if setter, ok := invoker.(processStartHookSetter); ok {
		setter.SetProcessStartHook(startLifecycle)
	} else {
		startLifecycle()
	}

	if setter, ok := invoker.(LiveOutputSetter); ok {
		setter.SetLiveOutput(newLifecycleOutputWriter(baseLiveOut, func(message string) {
			recorder.Record(s.publishEvent(LoopEvent{
				Time:       time.Now(),
				Kind:       EventOutputObserved,
				Iteration:  meta.iteration,
				Role:       meta.role,
				Task:       meta.task,
				Tool:       meta.tool,
				Message:    message,
				PreStatus:  meta.preStatus,
				PostStatus: meta.postStatus,
			}))
		}))
	}

	stdout, stderr, exitCode, err := invoker.Invoke(ctx, prompt, workDir, model)
	stopHeartbeat()

	heartbeatMu.Lock()
	done := heartbeatDone
	heartbeatMu.Unlock()
	if done != nil {
		if heartbeatErr := <-done; heartbeatErr != nil {
			recorder.Record(heartbeatErr)
		}
	}

	return observedInvocation{
		stdout:   stdout,
		stderr:   stderr,
		exitCode: exitCode,
		err:      err,
		eventErr: recorder.Err(),
	}
}

func (s *Service) directPrinter(mode UIMode) *output.Printer {
	if mode != UIModeTUI {
		return s.printer
	}
	return &output.Printer{
		Out:     io.Discard,
		Err:     io.Discard,
		Verbose: s.printer.Verbose,
	}
}

func (s *Service) printDryRun(printer *output.Printer, format string, args ...any) {
	if printer == nil {
		return
	}
	printer.DryRun(format, args...)
}

func (s *Service) publishEvent(event LoopEvent) error {
	if s.events == nil {
		return nil
	}
	return s.events.consume(event)
}

func (s *Service) resolveObserver(mode UIMode, session *LoopSession, capabilities TerminalCapabilities) LoopObserver {
	if s.observer != nil {
		return s.observer
	}
	if mode == UIModePlain {
		return NewTextPresenter(s.printer, func() SessionSnapshot {
			return session.SnapshotAt(time.Now())
		})
	}
	if mode == UIModeTUI {
		return NewBubbleTeaPresenter(capabilities, func() SessionSnapshot {
			return session.SnapshotAt(time.Now())
		})
	}
	return nil
}

func (s *Service) printPersistentFinalSummary(mode UIMode, summary FinalSummary) {
	if mode != UIModeTUI || s.printer == nil {
		return
	}
	s.printer.Info(
		"resumo final: stop=%s iteracoes=%d lote=%s report=%s",
		firstNonEmpty(summary.StopReason, "execucao encerrada"),
		summary.IterationsRun,
		formatBatchProgress(summary.Progress),
		firstNonEmpty(summary.ReportPath, "n/a"),
	)
	if summary.LastFailure != nil {
		s.printer.Error("falha final: %s", renderFailureMessage(summary.LastFailure))
	}
	s.printer.Info("task-loop finalizado: %s", firstNonEmpty(summary.StopReason, "execucao encerrada"))
	s.printer.Info("relatorio salvo em: %s", firstNonEmpty(summary.ReportPath, "n/a"))
}

type eventPublisher struct {
	session  *LoopSession
	observer LoopObserver
	fallback *TextPresenter
	printer  *output.Printer
	mu       sync.Mutex
	degraded bool
}

func newEventPublisher(session *LoopSession, observer LoopObserver, printer *output.Printer) *eventPublisher {
	fallback := NewTextPresenter(printer, func() SessionSnapshot {
		return session.SnapshotAt(time.Now())
	})
	if observer == nil {
		observer = fallback
	}
	return &eventPublisher{
		session:  session,
		observer: observer,
		fallback: fallback,
		printer:  printer,
	}
}

func (p *eventPublisher) start() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	snapshot := p.session.SnapshotAt(p.session.startedAt)
	if err := p.observer.Start(snapshot); err != nil {
		p.printer.Warn("observer start falhou; seguindo com fallback seguro: %v", err)
		p.observer = p.fallback
		p.degraded = true
		if err := p.fallback.Start(snapshot); err != nil {
			p.printer.Warn("fallback textual falhou ao iniciar: %v", err)
		}
	}
	return nil
}

func (p *eventPublisher) consume(event LoopEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	snapshot, err := p.session.Apply(event)
	if err != nil {
		return fmt.Errorf("reduzir evento %s: %w", event.Kind, err)
	}
	if err := p.observer.Consume(event); err != nil {
		p.printer.Warn("observer consume falhou; seguindo com fallback seguro: %v", err)
		p.observer = p.fallback
		p.degraded = true
		if err := p.fallback.Start(snapshot); err != nil {
			p.printer.Warn("fallback textual falhou ao sincronizar snapshot: %v", err)
		}
		if err := p.fallback.Consume(event); err != nil {
			p.printer.Warn("fallback textual falhou ao consumir evento: %v", err)
		}
	}
	return nil
}

func (p *eventPublisher) finish(stopReason, reportPath string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	summary := p.session.FinalSummary(stopReason, reportPath)
	if err := p.observer.Finish(summary); err != nil {
		p.printer.Warn("observer finish falhou; seguindo com fallback seguro: %v", err)
		p.observer = p.fallback
		p.degraded = true
		if err := p.fallback.Finish(summary); err != nil {
			p.printer.Warn("fallback textual falhou ao finalizar: %v", err)
		}
	}
}

func computeBatchProgress(tasks []TaskEntry) (BatchProgress, error) {
	counts := map[string]int{
		"done":        0,
		"failed":      0,
		"blocked":     0,
		"needs_input": 0,
		"pending":     0,
		"in_progress": 0,
	}
	for _, task := range tasks {
		status := normalizeStatus(task.Status)
		if _, ok := counts[status]; !ok {
			return BatchProgress{}, fmt.Errorf("status de task invalido para progresso: %s", task.Status)
		}
		counts[status]++
	}
	progress, err := NewBatchProgress(
		len(tasks),
		counts["done"],
		counts["failed"],
		counts["blocked"],
		counts["needs_input"],
		counts["pending"],
		counts["in_progress"],
	)
	if err != nil {
		return BatchProgress{}, fmt.Errorf("construir progresso do lote: %w", err)
	}
	return progress, nil
}

func newProgressEvent(at time.Time, tasks []TaskEntry) (LoopEvent, error) {
	progress, err := computeBatchProgress(tasks)
	if err != nil {
		return LoopEvent{}, err
	}
	return LoopEvent{
		Time:     at,
		Kind:     EventProgressUpdated,
		Message:  "progresso do lote atualizado",
		Progress: &progress,
	}, nil
}

func resolveToolName(raw string) (ToolName, error) {
	tool, err := NewToolName(raw)
	if err != nil {
		return "", err
	}
	return tool, nil
}

func buildRoleCompletedMessage(role string, invokeErr error, exitCode int, preStatus, postStatus string) string {
	switch {
	case invokeErr != nil:
		return fmt.Sprintf("%s finalizado com erro de invocacao: %v", role, invokeErr)
	case exitCode != 0:
		return fmt.Sprintf("%s finalizado com exit=%d", role, exitCode)
	case normalizeStatus(postStatus) == normalizeStatus(preStatus):
		return fmt.Sprintf("%s finalizado sem alterar status da task", role)
	default:
		return fmt.Sprintf("%s finalizado: %s -> %s", role, preStatus, postStatus)
	}
}

func (s *Service) publishRoleStarted(
	iteration int,
	task TaskRef,
	tool ToolName,
	role AgentRole,
	phase AgentPhase,
	preStatus string,
	postStatus string,
) error {
	return s.publishEvent(LoopEvent{
		Time:       time.Now(),
		Kind:       EventPhaseChanged,
		Iteration:  iteration,
		Role:       role,
		Task:       task,
		Tool:       tool,
		Phase:      phase,
		Message:    fmt.Sprintf("%s iniciado", role),
		PreStatus:  preStatus,
		PostStatus: postStatus,
	})
}

func (s *Service) publishRoleFinished(
	iteration int,
	task TaskRef,
	tool ToolName,
	role AgentRole,
	phase AgentPhase,
	preStatus string,
	postStatus string,
	exitCode int,
	message string,
) error {
	return s.publishEvent(LoopEvent{
		Time:       time.Now(),
		Kind:       EventPhaseChanged,
		Iteration:  iteration,
		Role:       role,
		Task:       task,
		Tool:       tool,
		Phase:      phase,
		Message:    strings.TrimSpace(message),
		ExitCode:   exitCode,
		PreStatus:  preStatus,
		PostStatus: postStatus,
	})
}
