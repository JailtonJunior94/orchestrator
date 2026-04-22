package aispecharness

import (
	"fmt"
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

Modo simples:  use --tool para invocar um unico agente (comportamento atual).
Modo avancado: use --executor-tool e opcionalmente --reviewer-tool para
               configurar executor e reviewer independentes com modelos distintos.

Gera um relatorio consolidado em Markdown ao final da execucao.

Exit codes:
  0 — loop completado (com ou sem tasks restantes)
  1 — erro de pre-flight ou execucao
  2 — uso incorreto

Exemplos:
  ai-spec task-loop --tool claude tasks/prd-minha-feature
  ai-spec task-loop --tool codex --dry-run tasks/prd-minha-feature
  ai-spec task-loop --executor-tool claude --executor-model claude-sonnet-4-6 \
    --reviewer-tool claude --reviewer-model claude-opus-4-6 tasks/prd-minha-feature`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prdFolder := args[0]

		tool, _ := cmd.Flags().GetString("tool")
		execTool, _ := cmd.Flags().GetString("executor-tool")
		execModel, _ := cmd.Flags().GetString("executor-model")
		revTool, _ := cmd.Flags().GetString("reviewer-tool")
		revModel, _ := cmd.Flags().GetString("reviewer-model")
		fallbackTool, _ := cmd.Flags().GetString("fallback-tool")
		allowUnknown, _ := cmd.Flags().GetBool("allow-unknown-model")
		reviewerTmpl, _ := cmd.Flags().GetString("reviewer-prompt-template")
		execFallbackModel, _ := cmd.Flags().GetString("executor-fallback-model")
		revFallbackModel, _ := cmd.Flags().GetString("reviewer-fallback-model")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		maxIter, _ := cmd.Flags().GetInt("max-iterations")
		timeout, _ := cmd.Flags().GetDuration("timeout")
		reportPath, _ := cmd.Flags().GetString("report-path")

		// Validacao mutua exclusiva entre modo simples e avancado
		if tool != "" && (execTool != "" || revTool != "") {
			return fmt.Errorf("--tool e --executor-tool/--reviewer-tool sao mutuamente exclusivas")
		}
		if tool == "" && execTool == "" {
			return fmt.Errorf("informe --tool (modo simples) ou --executor-tool (modo avancado)")
		}
		if execModel != "" && execTool == "" {
			return fmt.Errorf("--executor-model requer --executor-tool")
		}
		if revModel != "" && revTool == "" {
			return fmt.Errorf("--reviewer-model requer --reviewer-tool")
		}

		// Validacao de ferramenta no modo simples
		if tool != "" && !taskloop.ValidTools[tool] {
			return fmt.Errorf("ferramenta invalida %q — opcoes: claude, codex, gemini, copilot", tool)
		}

		// Resolver perfis: converte flags em ProfileConfig (nil = modo simples)
		profiles, err := taskloop.ResolveProfiles(tool, execTool, execModel, revTool, revModel)
		if err != nil {
			return err
		}

		if reportPath == "" {
			reportPath = fmt.Sprintf("task-loop-report-%s.md", time.Now().Format("20060102-150405"))
		}

		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()

		svc := taskloop.NewService(fsys, printer)
		return svc.Execute(taskloop.Options{
			PRDFolder:              prdFolder,
			Tool:                   tool,
			DryRun:                 dryRun,
			MaxIterations:          maxIter,
			Timeout:                timeout,
			ReportPath:             reportPath,
			Profiles:               profiles,
			FallbackTool:           fallbackTool,
			AllowUnknownModel:      allowUnknown,
			ReviewerPromptTemplate: reviewerTmpl,
			ExecutorFallbackModel:  execFallbackModel,
			ReviewerFallbackModel:  revFallbackModel,
		})
	},
}

func init() {
	// Flags existentes (preservadas)
	taskLoopCmd.Flags().String("tool", "", "Agente de IA: claude, codex, gemini, copilot (modo simples)")
	taskLoopCmd.Flags().Bool("dry-run", false, "Mostra o que seria executado sem invocar o agente")
	taskLoopCmd.Flags().Int("max-iterations", 20, "Limite maximo de iteracoes do loop")
	taskLoopCmd.Flags().Duration("timeout", 30*time.Minute, "Timeout por task")
	taskLoopCmd.Flags().String("report-path", "", "Caminho do relatorio final (default: task-loop-report-<timestamp>.md)")

	// Flags novas — modo avancado por papel
	taskLoopCmd.Flags().String("executor-tool", "", "Ferramenta do executor (modo avancado): claude, codex, gemini, copilot")
	taskLoopCmd.Flags().String("executor-model", "", "Modelo do executor (ex: claude-sonnet-4-6)")
	taskLoopCmd.Flags().String("reviewer-tool", "", "Ferramenta do reviewer (modo avancado): claude, codex, gemini, copilot")
	taskLoopCmd.Flags().String("reviewer-model", "", "Modelo do reviewer (ex: claude-opus-4-6)")
	taskLoopCmd.Flags().String("fallback-tool", "", "Ferramenta de fallback para validacao pre-loop")
	taskLoopCmd.Flags().Bool("allow-unknown-model", false, "Aceitar combinacoes ferramenta-modelo nao catalogadas")
	taskLoopCmd.Flags().String("reviewer-prompt-template", "", "Caminho do template de prompt de revisao customizado")
	taskLoopCmd.Flags().String("executor-fallback-model", "", "Modelo de fallback nativo do executor (Claude only)")
	taskLoopCmd.Flags().String("reviewer-fallback-model", "", "Modelo de fallback nativo do reviewer (Claude only)")

	rootCmd.AddCommand(taskLoopCmd)
}
