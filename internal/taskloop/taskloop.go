package taskloop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type Options struct {
	PRDFolder     string
	Tool          string
	DryRun        bool
	MaxIterations int
	Timeout       time.Duration
	ReportPath    string

	Profiles               *ProfileConfig
	FallbackTool           string
	AllowUnknownModel      bool
	ReviewerPromptTemplate string
	ExecutorFallbackModel  string
	ReviewerFallbackModel  string
	MaxBugfixIterations    int
	MaxBugfixIterationsSet bool
	HandoffLeaseTTL        time.Duration
	HandoffLeaseTTLSet     bool
	NonInteractive         bool

	DurableMemoryEnabled bool

	DurableMemoryEnabledSet bool

	Runtime         string
	ActivityTimeout time.Duration

	ActivityTimeoutSet bool
	Quiet              bool

	AgentName string

	ReasoningEffort string
	AccessMode      string
	AddDirs         []string

	MCPNested bool

	NoNormalize bool

	MemoryWorkflowLimitLines int
	MemoryWorkflowLimitBytes int
	MemoryTaskLimitLines     int
	MemoryTaskLimitBytes     int

	MemoryLimitsSet bool

	DisableHooks bool

	SkipDriftGuard bool

	AutoReview bool

	Concurrent int

	BatchSize int
}

type Service struct {
	fsys              fs.FileSystem
	printer           *output.Printer
	invokerFactory    func(tool string) (AgentInvoker, error)
	acpInvokerFactory func(opts Options) AgentInvoker
	binaryChecker     func(AgentInvoker) error
	liveOutOverride   io.Writer
}

func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{
		fsys:           fsys,
		printer:        printer,
		invokerFactory: NewAgentInvoker,
	}
}

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

func (c *Catalog) compatibilityStatusLabel(table *CompatibilityTable, tool, model string) string {
	if table.IsSupported(tool, model) {
		return "✓ (compativel)"
	}
	return "✗ (incompativel)"
}

func (s *Service) printDryRunAdvancedHeader(opts Options, absFolder, workDir string) {
	s.printer.DryRun("modo: avancado")

	table := NewCompatibilityTable()

	exec := opts.Profiles.Executor
	execModelDisplay := exec.Model()
	if execModelDisplay == "" {
		execModelDisplay = "default"
	}
	execStatus := NewCatalog().compatibilityStatusLabel(table, exec.Tool(), exec.Model())
	s.printer.DryRun("executor: %s / %s / %s %s", exec.Tool(), exec.Provider(), execModelDisplay, execStatus)

	if opts.Profiles.Reviewer != nil {
		rev := *opts.Profiles.Reviewer
		revModelDisplay := rev.Model()
		if revModelDisplay == "" {
			revModelDisplay = "default"
		}
		revStatus := NewCatalog().compatibilityStatusLabel(table, rev.Tool(), rev.Model())
		s.printer.DryRun("reviewer: %s / %s / %s %s", rev.Tool(), rev.Provider(), revModelDisplay, revStatus)
	}

	if opts.ReviewerPromptTemplate != "" {
		s.printer.DryRun("template de revisao: %s", opts.ReviewerPromptTemplate)
	} else {
		s.printer.DryRun("template de revisao: default (embutido)")
	}

	tasksContent, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md"))
	if err != nil {
		return
	}
	tasks, err := NewCatalog().ParseTasksFile(tasksContent)
	if err != nil {
		return
	}
	tasks = NewCatalog().reconcileTaskStatuses(tasks, absFolder, s.fsys)
	eligible := NewCatalog().FindEligible(tasks, nil)

	if len(eligible) == 0 {
		s.printer.DryRun("tasks elegiveis: nenhuma")
		return
	}

	ids := make([]string, 0, len(eligible))
	for _, t := range eligible {
		ids = append(ids, t.ID)
	}
	s.printer.DryRun("tasks elegiveis: %s", strings.Join(ids, ", "))

	firstTask := eligible[0]
	taskFile, err := NewCatalog().ResolveTaskFile(absFolder, firstTask, s.fsys)
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

	preview, err := NewCatalog().BuildReviewPrompt(opts.ReviewerPromptTemplate, ReviewTemplateData{
		TaskFile:       relTaskFile,
		PRDFolder:      relPRD,
		TechSpec:       filepath.Join(relPRD, "techspec.md"),
		TasksFile:      filepath.Join(relPRD, "tasks.md"),
		Diff:           "(dry-run: diff nao disponivel)",
		CompletedTasks: "(dry-run: nenhuma task executada)",
		RiskAreas:      NewCatalog().detectRiskAreas(relPRD, workDir, "", s.fsys),
	}, s.fsys)
	if err != nil {
		return
	}

	s.printer.DryRun("--- preview do template (task %s) ---", firstTask.ID)
	for line := range strings.SplitSeq(preview, "\n") {
		s.printer.DryRun("%s", line)
	}
	s.printer.DryRun("--- fim do preview ---")
}

func (s *Service) Execute(opts Options) error {
	absFolder, err := filepath.Abs(opts.PRDFolder)
	if err != nil {
		return fmt.Errorf("caminho invalido %q: %w", opts.PRDFolder, err)
	}

	for _, required := range []string{"tasks.md", "prd.md", "techspec.md"} {
		path := filepath.Join(absFolder, required)
		if !s.fsys.Exists(path) {
			return fmt.Errorf("arquivo obrigatorio nao encontrado: %s", path)
		}
	}

	var resolvedAgent *agents.ResolvedAgent
	var agentCatalog []agents.ResolvedAgent
	if opts.AgentName != "" {
		workDirForAgent, wdErr := NewCatalog().resolveWorkDir(absFolder, s.fsys)
		if wdErr != nil {
			workDirForAgent = absFolder
		}
		home, _ := os.UserHomeDir()
		reg := agents.NewDefaultRegistry(s.fsys, workDirForAgent, home)
		catalog, _ := reg.Discover(context.Background())
		agentCatalog = catalog
		agent, agentErr := reg.Resolve(opts.AgentName)
		if agentErr != nil {
			return fmt.Errorf("agente %q nao encontrado: %w", opts.AgentName, agentErr)
		}
		resolvedAgent = &agent

		if opts.Profiles == nil {
			override := agents.RuntimeOverride{}
			agentProfile, profileErr := NewCatalog().ResolveProfileFromAgent(agent, override, opts.AllowUnknownModel)
			if profileErr != nil {
				return fmt.Errorf("erro ao derivar perfil do agente %q: %w", opts.AgentName, profileErr)
			}
			opts.Profiles = agentProfile
		}

		if opts.Runtime == "" || opts.Runtime == "legacy" {
			opts.Runtime = "acp"
		}
	}

	if opts.Profiles != nil {
		profilesCopy := *opts.Profiles
		if opts.Profiles.Reviewer != nil {
			revCopy := *opts.Profiles.Reviewer
			profilesCopy.Reviewer = &revCopy
		}
		opts.Profiles = &profilesCopy
	}

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

	executorTool := opts.Tool
	if opts.Profiles != nil {
		executorTool = opts.Profiles.Executor.Tool()
	}

	var invoker AgentInvoker
	if opts.Runtime == "acp" {
		if s.acpInvokerFactory != nil {
			invoker = s.acpInvokerFactory(opts)
		} else {

			resolvedRC, rcErr := NewCatalog().resolveRuntimeConfig(absFolder, NewCatalog().optionsToConfigOverrides(opts))
			if rcErr != nil {
				return fmt.Errorf("taskloop: wiring RuntimeConfig: %w", rcErr)
			}
			opts = NewCatalog().applyResolvedMaxBugfixIterations(opts, resolvedRC)

			spec, specErr := NewCatalog().resolveACPSpec(executorTool)
			if specErr != nil {
				return fmt.Errorf("taskloop: resolver spec ACP: %w", specErr)
			}
			factory := persistence.NewSessionPersistenceFactory(fs.NewOSFileSystem())
			runner := airuntime.NewACPRunner(
				spec, airuntime.NewCatalog().
					WithPersistenceFactory(factory),
			)

			invoker = NewACPInvoker(runner, opts.Quiet, resolvedRC.Timeout.Duration(), NewCatalog().WithACPInvokerReasoningEffort(opts.ReasoningEffort), NewCatalog().WithACPInvokerAccessMode(specs.AccessMode(opts.AccessMode)), NewCatalog().WithACPInvokerAddDirs(opts.AddDirs), NewCatalog().WithACPInvokerMCPNested(opts.MCPNested), NewCatalog().WithACPInvokerNoNormalize(opts.NoNormalize), NewCatalog().WithACPInvokerMemoryLimitLines(opts.MemoryWorkflowLimitLines, opts.MemoryTaskLimitLines), NewCatalog().WithACPInvokerMemoryLimitBytes(opts.MemoryWorkflowLimitBytes, opts.MemoryTaskLimitBytes), NewCatalog().WithACPInvokerMemoryLimitsExplicit(opts.MemoryLimitsSet), NewCatalog().WithACPInvokerDisableHooks(opts.DisableHooks), NewCatalog().WithACPInvokerSkipDriftGuard(opts.SkipDriftGuard), NewCatalog().WithACPInvokerTasksDir(opts.PRDFolder), NewCatalog().WithACPInvokerAutoReview(opts.AutoReview), NewCatalog().WithACPInvokerMaxRetries(resolvedRC.MaxRetries), NewCatalog().WithACPInvokerRetryBackoffMultiplier(resolvedRC.RetryBackoffMultiplier), NewCatalog().WithACPInvokerMaxBugfixIterations(resolvedRC.MaxBugfixIterations), NewCatalog().WithACPInvokerHandoffLeaseTTL(resolvedRC.HandoffLeaseTTL), NewCatalog().WithACPInvokerDurableMemory(resolvedRC.DurableMemoryEnabled))
		}
	} else {
		var invokerErr error
		invoker, invokerErr = s.createInvokerWithFallback(executorTool, opts.ExecutorFallbackModel)
		if invokerErr != nil {
			return invokerErr
		}
	}

	if !opts.DryRun {
		checker := s.binaryChecker
		if checker == nil {
			checker = NewCatalog().CheckAgentBinary
		}
		if opts.Runtime != "acp" {
			if err := checker(invoker); err != nil {
				return err
			}
		}

		if executorTool == "claude" {
			if warn := NewCatalog().warnClaudeAuth(); warn != "" {
				s.printer.Warn("claude auth: %s", warn)
			}
		}

		if lo, ok := invoker.(LiveOutputSetter); ok {
			liveOut := s.liveOutOverride
			if liveOut == nil {
				liveOut = os.Stderr
			}
			lo.SetLiveOutput(liveOut)
		}
	}

	workDir, err := NewCatalog().resolveWorkDir(absFolder, s.fsys)
	if err != nil {
		return fmt.Errorf("erro ao resolver diretorio de trabalho: %w", err)
	}

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

	maxLabel := strconv.Itoa(opts.MaxIterations)
	if opts.MaxIterations == 0 {
		maxLabel = "ilimitado"
	}
	s.printer.Info("task-loop iniciado: folder=%s tool=%s max=%s timeout=%s",
		opts.PRDFolder, opts.Tool, maxLabel, opts.Timeout)

	if opts.DryRun && opts.Profiles != nil {
		s.printDryRunAdvancedHeader(opts, absFolder, workDir)
	}

	for opts.MaxIterations == 0 || iteration < opts.MaxIterations {

		tasksContent, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md"))
		if err != nil {
			return fmt.Errorf("erro ao ler tasks.md: %w", err)
		}

		tasks, err := NewCatalog().ParseTasksFile(tasksContent)
		if err != nil {
			return fmt.Errorf("erro ao parsear tasks.md: %w", err)
		}
		tasks = NewCatalog().reconcileTaskStatuses(tasks, absFolder, s.fsys)

		eligible := NewCatalog().FindEligible(tasks, skipped)
		if len(eligible) == 0 {
			if NewCatalog().AllTerminal(tasks) {
				report.StopReason = "todas as tasks completadas ou em estado terminal"
			} else {
				report.StopReason = "nenhuma task elegivel (restantes estao bloqueadas, falharam ou aguardam input)"
			}
			report.FinalTasks = tasks
			break
		}

		task := eligible[0]
		iteration++

		taskFile, err := NewCatalog().ResolveTaskFile(absFolder, task, s.fsys)
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

		preStatus := task.Status
		if fileStatus := NewCatalog().readTaskStatus(taskFile, s.fsys); fileStatus != "" {
			preStatus = fileStatus
		}

		relTaskFile, _ := filepath.Rel(workDir, taskFile)
		if relTaskFile == "" {
			relTaskFile = taskFile
		}
		relPRD, _ := filepath.Rel(workDir, absFolder)
		if relPRD == "" {
			relPRD = absFolder
		}

		promptCtx := NewCatalog().BuildPromptContext(relPRD, workDir, s.fsys, resolvedAgent, agentCatalog)
		prompt := NewCatalog().BuildPrompt(relTaskFile, relPRD, promptCtx)

		s.printer.Step("iteracao %d: executando task %s (%s)", iteration, task.ID, task.Title)

		if opts.DryRun {
			if opts.Profiles != nil {

				if opts.Profiles.Reviewer != nil {
					s.printer.DryRun("iteracao %d: executaria task %s com executor, depois reviewer", iteration, task.ID)
				} else {
					s.printer.DryRun("iteracao %d: executaria task %s com executor", iteration, task.ID)
				}
			} else {

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

		executorModel := ""
		if opts.Profiles != nil {
			executorModel = opts.Profiles.Executor.Model()
		}

		snapshot, err := NewCatalog().captureTaskIsolationSnapshotWithMode(absFolder, _taskIsolationModeExecutor, s.fsys)
		if err != nil {
			return fmt.Errorf("erro ao capturar snapshot de isolamento da task %s: %w", task.ID, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		startTime := time.Now()
		stdout, stderr, exitCode, invokeErr := invoker.Invoke(ctx, prompt, workDir, executorModel)
		elapsed := time.Since(startTime)
		cancel()

		isolationErr := NewCatalog().validateTaskIsolation(snapshot, absFolder, task.ID, taskFile, s.fsys)
		if isolationErr != nil {
			if restoreErr := NewCatalog().restoreTaskIsolationSnapshotAt(snapshot, absFolder, s.fsys); restoreErr != nil {
				return fmt.Errorf("violacao de isolamento na task %s: %v; falha ao restaurar snapshot: %w", task.ID, isolationErr, restoreErr)
			}
		}

		postStatus := preStatus
		if fileStatus := NewCatalog().readTaskStatus(taskFile, s.fsys); fileStatus != "" {
			postStatus = fileStatus
		}

		if postStatus == preStatus {
			if updatedContent, readErr := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); readErr == nil {
				if updatedTasks, parseErr := NewCatalog().ParseTasksFile(updatedContent); parseErr == nil {
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
				if finalTasks, err := NewCatalog().ParseTasksFile(content); err == nil {
					report.FinalTasks = finalTasks
				}
			}
			break
		}

		outcome := NewCatalog().classifyIterationOutcome(preStatus, postStatus, exitCode, invokeErr, stdout, stderr)

		if outcome.Abort {
			guidance := NewCatalog().authGuidance(executorTool)
			iterResult.Note = fmt.Sprintf("erro de autenticacao: %s nao esta autenticado — %s", executorTool, guidance)
			s.printer.Error("  erro de autenticacao detectado para %s — %s", executorTool, guidance)
			report.Iterations = append(report.Iterations, iterResult)
			report.StopReason = fmt.Sprintf("abortado: %s nao esta autenticado", executorTool)
			report.FinalTasks = tasks
			break
		}

		if outcome.Note != "" {
			iterResult.Note = NewCatalog().appendNote(iterResult.Note, outcome.Note)
		}

		if invokeErr != nil {
			if opts.Runtime == "acp" && errors.Is(invokeErr, airuntime.ErrLauncherUnavailable) {
				return invokeErr
			}
			s.printer.Error("iteracao %d: %v", iteration, invokeErr)
		} else if exitCode != 0 {

			if exitCode == -1 && stdout == "" && stderr == "" {
				iterResult.Note = NewCatalog().appendNote(iterResult.Note,
					fmt.Sprintf("saida vazia — %s pode requerer TTY ou nao suportar output em pipe", executorTool))
			}
			if stderr != "" {
				s.printer.Debug("stderr: %s", NewCatalog().truncate(stderr, 500))
			}
		}

		if outcome.Skip {
			skipped[task.ID] = true
		}

		cycleConducted := false
		if opts.Profiles != nil && opts.Profiles.Reviewer != nil && outcome.RunReviewer {
			reviewSnapshot, err := NewCatalog().captureTaskIsolationSnapshotWithMode(absFolder, _taskIsolationModeReviewer, s.fsys)
			if err != nil {
				return fmt.Errorf("erro ao capturar snapshot de isolamento do reviewer na task %s: %w", task.ID, err)
			}

			var cycleCriteria []approval.AcceptanceCriterion
			if taskFileContent, readErr := s.fsys.ReadFile(taskFile); readErr == nil {
				cycleCriteria, _ = acceptanceCriteriaFromTaskFile(taskFileContent)
			}
			if len(cycleCriteria) > 0 {
				cycleCtx, cycleCancel := context.WithTimeout(context.Background(), opts.Timeout)
				review, bugfix := s.conductApprovalCycle(cycleCtx, opts, task, cycleCriteria, relTaskFile, relPRD, workDir)
				cycleCancel()
				cycleConducted = true
				iterResult.ReviewResult = review
				iterResult.BugfixResult = bugfix
			}
			if !cycleConducted {
				iterResult.ReviewResult = s.invokeReviewer(opts, relTaskFile, relPRD, workDir, task.ID, report.Iterations)
			}
			reviewIsolationErr := NewCatalog().validateReviewerIsolation(reviewSnapshot, absFolder, task.ID, taskFile, s.fsys)
			if reviewIsolationErr != nil {
				if restoreErr := NewCatalog().restoreTaskIsolationSnapshotAt(reviewSnapshot, absFolder, s.fsys); restoreErr != nil {
					return fmt.Errorf("violacao de isolamento do reviewer na task %s: %v; falha ao restaurar snapshot: %w", task.ID, reviewIsolationErr, restoreErr)
				}
				if iterResult.ReviewResult == nil {
					iterResult.ReviewResult = &ReviewResult{}
				}
				iterResult.ReviewResult.Note = NewCatalog().appendNote(iterResult.ReviewResult.Note,
					fmt.Sprintf("violacao de isolamento detectada: %v", reviewIsolationErr))
				s.printer.Error("iteracao %d: reviewer violou isolamento da task %s: %v", iteration, task.ID, reviewIsolationErr)
				report.Iterations = append(report.Iterations, iterResult)
				report.StopReason = fmt.Sprintf("abortado: reviewer violou isolamento da task %s", task.ID)
				if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
					if finalTasks, err := NewCatalog().ParseTasksFile(content); err == nil {
						report.FinalTasks = finalTasks
					}
				}
				break
			}
		}

		if !cycleConducted && iterResult.ReviewResult != nil && iterResult.ReviewResult.ExitCode != 0 {
			bugfixSnapshot, bfSnapErr := NewCatalog().captureTaskIsolationSnapshotWithMode(absFolder, _taskIsolationModeExecutor, s.fsys)
			if bfSnapErr != nil {
				return fmt.Errorf("erro ao capturar snapshot de isolamento do bugfix na task %s: %w", task.ID, bfSnapErr)
			}

			diff := NewCatalog().captureGitDiff(context.Background(), workDir)
			iterResult.BugfixResult = s.invokeBugfix(opts, relTaskFile, relPRD, workDir, iterResult.ReviewResult.Output, diff)

			bugfixIsolationErr := NewCatalog().validateTaskIsolation(bugfixSnapshot, absFolder, task.ID, taskFile, s.fsys)
			if bugfixIsolationErr != nil {
				if restoreErr := NewCatalog().restoreTaskIsolationSnapshotAt(bugfixSnapshot, absFolder, s.fsys); restoreErr != nil {
					return fmt.Errorf("violacao de isolamento do bugfix na task %s: %v; falha ao restaurar snapshot: %w", task.ID, bugfixIsolationErr, restoreErr)
				}
				if iterResult.BugfixResult == nil {
					iterResult.BugfixResult = &BugfixResult{}
				}
				iterResult.BugfixResult.Note = NewCatalog().appendNote(iterResult.BugfixResult.Note,
					fmt.Sprintf("violacao de isolamento detectada: %v", bugfixIsolationErr))
				s.printer.Error("iteracao %d: bugfix violou isolamento da task %s: %v", iteration, task.ID, bugfixIsolationErr)
				report.Iterations = append(report.Iterations, iterResult)
				report.StopReason = fmt.Sprintf("abortado: bugfix violou isolamento da task %s", task.ID)
				if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
					if finalTasks, err := NewCatalog().ParseTasksFile(content); err == nil {
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

	if opts.MaxIterations > 0 && iteration >= opts.MaxIterations && report.StopReason == "" {
		report.StopReason = fmt.Sprintf("limite de iteracoes atingido (%d)", opts.MaxIterations)

		if content, err := s.fsys.ReadFile(filepath.Join(absFolder, "tasks.md")); err == nil {
			if tasks, err := NewCatalog().ParseTasksFile(content); err == nil {
				report.FinalTasks = tasks
			}
		}
	}

	report.EndTime = time.Now()

	reportContent := report.Render()
	if err := s.fsys.WriteFile(opts.ReportPath, reportContent); err != nil {
		return fmt.Errorf("erro ao escrever relatorio: %w", err)
	}

	s.printer.Info("task-loop finalizado: %s", report.StopReason)
	s.printer.Info("relatorio salvo em: %s", opts.ReportPath)

	return nil
}

func (s *Service) invokeReviewer(opts Options, relTaskFile, relPRD, workDir, taskID string, iterations []IterationResult) *ReviewResult {
	reviewerInvoker, err := s.createInvokerWithFallback(
		opts.Profiles.Reviewer.Tool(),
		opts.ReviewerFallbackModel,
	)
	if err != nil {
		return &ReviewResult{
			Note: fmt.Sprintf("erro ao criar invoker do reviewer: %v", err),
		}
	}

	diff := NewCatalog().captureGitDiff(context.Background(), workDir)
	reviewPrompt, promptErr := NewCatalog().BuildReviewPrompt(
		opts.ReviewerPromptTemplate,
		ReviewTemplateData{
			TaskFile:       relTaskFile,
			PRDFolder:      relPRD,
			TechSpec:       filepath.Join(relPRD, "techspec.md"),
			TasksFile:      filepath.Join(relPRD, "tasks.md"),
			Diff:           diff,
			CompletedTasks: NewCatalog().formatCompletedTasks(iterations, taskID),
			RiskAreas:      NewCatalog().detectRiskAreas(relPRD, workDir, diff, s.fsys),
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

func (s *Service) invokeBugfix(opts Options, relTaskFile, relPRD, workDir, reviewFindings, diff string) *BugfixResult {
	executorTool := opts.Tool
	if opts.Profiles != nil {
		executorTool = opts.Profiles.Executor.Tool()
	}

	bugfixInvoker, err := s.createInvokerWithFallback(executorTool, opts.ExecutorFallbackModel)
	if err != nil {
		return &BugfixResult{
			Note: fmt.Sprintf("erro ao criar invoker do bugfix: %v", err),
		}
	}

	bugfixPrompt, promptErr := NewCatalog().BuildBugfixPrompt(BugfixTemplateData{
		TaskFile:       relTaskFile,
		PRDFolder:      relPRD,
		TechSpec:       filepath.Join(relPRD, "techspec.md"),
		TasksFile:      filepath.Join(relPRD, "tasks.md"),
		ReviewFindings: reviewFindings,
		Diff:           diff,
	})
	if promptErr != nil {
		return &BugfixResult{
			Note: fmt.Sprintf("erro ao construir prompt de bugfix: %v", promptErr),
		}
	}

	executorModel := ""
	if opts.Profiles != nil {
		executorModel = opts.Profiles.Executor.Model()
	}

	s.printer.Step("  bugfix: corrigindo achados criticos da revisao")

	bctx, bcancel := context.WithTimeout(context.Background(), opts.Timeout)
	bStart := time.Now()
	bStdout, _, bExitCode, bErr := bugfixInvoker.Invoke(bctx, bugfixPrompt, workDir, executorModel)
	bElapsed := time.Since(bStart)
	bcancel()

	bugfixResult := &BugfixResult{
		Duration: bElapsed,
		ExitCode: bExitCode,
		Output:   bStdout,
	}
	if bErr != nil {
		bugfixResult.Note = fmt.Sprintf("erro de invocacao do bugfix: %v", bErr)
	} else if bExitCode != 0 {
		bugfixResult.Note = "bugfix nao conseguiu corrigir todos os achados"
	}

	return bugfixResult
}

type iterationOutcome struct {
	Skip        bool
	Abort       bool
	Note        string
	RunReviewer bool
}

func (c *Catalog) classifyIterationOutcome(
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
		if NewCatalog().isAuthError(combined) {
			return iterationOutcome{Abort: true, Note: "erro de autenticacao"}
		}
		outcome.Note = fmt.Sprintf("agente saiu com codigo %d", exitCode)
	}

	if invokeErr == nil && postStatus == "done" {
		outcome.RunReviewer = true
	}

	if postStatus == preStatus {
		outcome.Note = NewCatalog().appendNote(outcome.Note, "status inalterado apos execucao; pulando")
		outcome.Skip = true
	}

	if postStatus == "failed" || postStatus == "blocked" || postStatus == "needs_input" {
		outcome.Skip = true
	}

	return outcome
}

var acpSpecCatalog = specs.NewCatalog().ACPSpecCatalog()

func (c *Catalog) resolveACPSpec(tool string) (specs.Spec, error) {
	return specs.NewCatalog().ResolveACPSpec(tool)
}

func (c *Catalog) resolveWorkDir(prdFolder string, fsys fs.FileSystem) (string, error) {
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

func (c *Catalog) truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (c *Catalog) appendNote(existing, addition string) string {
	if existing == "" {
		return addition
	}
	return existing + "; " + addition
}
