package taskloop

import (
	"context"
	"fmt"
	"path/filepath"
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
}

// Service orquestra a execucao sequencial de tasks de um PRD folder.
type Service struct {
	fsys    fs.FileSystem
	printer *output.Printer
}

// NewService cria um novo Service de task-loop.
func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fsys: fsys, printer: printer}
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

	// Criar invoker e verificar binario
	invoker, err := NewAgentInvoker(opts.Tool)
	if err != nil {
		return err
	}

	if !opts.DryRun {
		if err := CheckAgentBinary(invoker); err != nil {
			return err
		}
	}

	// Resolver diretorio de trabalho (raiz do projeto — pai do prd folder ou cwd)
	workDir, err := resolveWorkDir(absFolder, s.fsys)
	if err != nil {
		return fmt.Errorf("erro ao resolver diretorio de trabalho: %w", err)
	}

	report := &Report{
		PRDFolder: opts.PRDFolder,
		Tool:      opts.Tool,
		StartTime: time.Now(),
	}

	skipped := make(map[string]bool)
	iteration := 0

	s.printer.Info("task-loop iniciado: folder=%s tool=%s max=%d timeout=%s",
		opts.PRDFolder, opts.Tool, opts.MaxIterations, opts.Timeout)

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
		iteration++

		// Resolver arquivo da task
		taskFile, err := ResolveTaskFile(absFolder, task, s.fsys)
		if err != nil {
			s.printer.Warn("iteracao %d: %v — pulando task %s", iteration, err, task.ID)
			skipped[task.ID] = true
			report.Iterations = append(report.Iterations, IterationResult{
				Sequence:  iteration,
				TaskID:    task.ID,
				Title:     task.Title,
				PreStatus: task.Status,
				PostStatus: task.Status,
				Note:      fmt.Sprintf("arquivo nao encontrado: %v", err),
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
			s.printer.DryRun("invocaria %s com prompt para task %s (%s)", opts.Tool, task.ID, task.Title)
			s.printer.DryRun("task file: %s", relTaskFile)
			report.Iterations = append(report.Iterations, IterationResult{
				Sequence:   iteration,
				TaskID:     task.ID,
				Title:      task.Title,
				PreStatus:  preStatus,
				PostStatus: "dry-run",
				Note:       "dry-run: agente nao invocado",
			})
			skipped[task.ID] = true
			continue
		}

		// Invocar agente com timeout
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		startTime := time.Now()
		stdout, stderr, exitCode, invokeErr := invoker.Invoke(ctx, prompt, workDir)
		elapsed := time.Since(startTime)
		cancel()

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
		}

		if invokeErr != nil {
			iterResult.Note = fmt.Sprintf("erro de invocacao: %v", invokeErr)
			s.printer.Error("iteracao %d: %v", iteration, invokeErr)
			skipped[task.ID] = true
		} else if exitCode != 0 {
			iterResult.Note = fmt.Sprintf("agente saiu com codigo %d", exitCode)
			if stderr != "" {
				s.printer.Debug("stderr: %s", truncate(stderr, 500))
			}
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

		s.printer.Info("  resultado: %s -> %s (exit=%d, duracao=%s)",
			preStatus, postStatus, exitCode, elapsed.Truncate(time.Second))

		report.Iterations = append(report.Iterations, iterResult)
	}

	if iteration >= opts.MaxIterations {
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
