package aispecharness

import (
	"fmt"
	"os"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
	"github.com/spf13/cobra"
)

var taskLoopCmd = &cobra.Command{
	Use:   "task-loop <prd-folder>",
	Short: "Executa tasks de um PRD folder sequencialmente via agente de IA",
	Long: `Executa em loop todas as tasks elegiveis de uma pasta PRD, invocando
um agente de IA (Claude Code, Codex, Gemini ou Copilot) para cada task.

Cada iteracao:
  1. Parseia tasks.md para identificar a proxima task elegivel
  2. Invoca o agente selecionado com a skill execute-task
  3. Verifica o status resultante da task
  4. Continua ate nao haver mais tasks elegiveis

Gera um relatorio consolidado em Markdown ao final da execucao.

Exit codes:
  0 — loop completado (com ou sem tasks restantes)
  1 — erro de pre-flight ou execucao
  2 — uso incorreto

Exemplos:
  ai-spec task-loop --tool claude tasks/prd-minha-feature
  ai-spec task-loop --tool codex --dry-run tasks/prd-minha-feature
  ai-spec task-loop --tool gemini --max-iterations 5 --timeout 1h tasks/prd-minha-feature`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prdFolder := args[0]

		tool, _ := cmd.Flags().GetString("tool")
		if !taskloop.ValidTools[tool] {
			fmt.Fprintf(os.Stderr, "Erro: ferramenta invalida %q — opcoes: claude, codex, gemini, copilot\n", tool)
			os.Exit(2)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		maxIter, _ := cmd.Flags().GetInt("max-iterations")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		reportPath, _ := cmd.Flags().GetString("report-path")

		if reportPath == "" {
			reportPath = fmt.Sprintf("task-loop-report-%s.md", time.Now().Format("20060102-150405"))
		}

		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()

		svc := taskloop.NewService(fsys, printer)
		return svc.Execute(taskloop.Options{
			PRDFolder:     prdFolder,
			Tool:          tool,
			DryRun:        dryRun,
			MaxIterations: maxIter,
			Timeout:       timeout,
			ReportPath:    reportPath,
		})
	},
}

func init() {
	taskLoopCmd.Flags().String("tool", "", "Agente de IA: claude, codex, gemini, copilot (obrigatorio)")
	taskLoopCmd.Flags().Bool("dry-run", false, "Mostra o que seria executado sem invocar o agente")
	taskLoopCmd.Flags().Int("max-iterations", 20, "Limite maximo de iteracoes do loop")
	taskLoopCmd.Flags().Duration("timeout", 30*time.Minute, "Timeout por task")
	taskLoopCmd.Flags().String("report-path", "", "Caminho do relatorio final (default: task-loop-report-<timestamp>.md)")
	_ = taskLoopCmd.MarkFlagRequired("tool")

	rootCmd.AddCommand(taskLoopCmd)
}
