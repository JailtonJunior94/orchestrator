package aispecharness

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
	"github.com/spf13/cobra"
)

// accessModeFullWarnOnce garante que o warning de --access-mode=full seja emitido
// apenas uma vez por execução do processo (ADR-013 D-08, R-03 alto).
var accessModeFullWarnOnce sync.Once

// runtimeACPCatalog é a fonte de verdade para quais tools podem usar --runtime=acp nesta versão.
// Responsabilidade do CLI (ADR-012 D-04 / ADR-013 D-04): specs são unidades atômicas; a tabela de roteamento fica no CLI.
// Runtimes suportados nesta versão: claude (ADR-009), codex (ADR-013), copilot (ADR-012), gemini (ADR-015).
// Para adicionar suporte a um novo tool: registrar entrada e atualizar testes T-13/T-14/T-15/T-16.
var runtimeACPCatalog = map[string]func() specs.Spec{
	"claude":  specs.Claude,
	"codex":   specs.Codex,
	"copilot": specs.Copilot,
	"gemini":  specs.Gemini,
}

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
		agentName, _ := cmd.Flags().GetString("agent")
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
		runtime, _ := cmd.Flags().GetString("runtime")
		activityTimeout, _ := cmd.Flags().GetDuration("activity-timeout")
		activityTimeoutSet := cmd.Flags().Changed("activity-timeout")
		quiet, _ := cmd.Flags().GetBool("quiet")
		reasoningEffort, _ := cmd.Flags().GetString("reasoning-effort")
		accessMode, _ := cmd.Flags().GetString("access-mode")
		mcpNested, _ := cmd.Flags().GetBool("mcp-nested")
		noNormalize, _ := cmd.Flags().GetBool("no-normalize")
		memWorkflowLimitLines, _ := cmd.Flags().GetInt("memory-workflow-limit-lines")
		memWorkflowLimitBytes, _ := cmd.Flags().GetInt("memory-workflow-limit-bytes")
		memTaskLimitLines, _ := cmd.Flags().GetInt("memory-task-limit-lines")
		memTaskLimitBytes, _ := cmd.Flags().GetInt("memory-task-limit-bytes")
		memoryLimitsSet := cmd.Flags().Changed("memory-workflow-limit-lines") ||
			cmd.Flags().Changed("memory-workflow-limit-bytes") ||
			cmd.Flags().Changed("memory-task-limit-lines") ||
			cmd.Flags().Changed("memory-task-limit-bytes")
		disableHooks, _ := cmd.Flags().GetBool("disable-hooks")
		skipDriftGuard, _ := cmd.Flags().GetBool("skip-drift-guard")
		autoReview, _ := cmd.Flags().GetBool("auto-review")

		// RF-16: switch tool-aware para defaults de memory (F3-Gemini).
		// Aplica defaults Gemini-generosos (250 linhas/20 KiB workflow; 400 linhas/32 KiB task)
		// somente quando --tool gemini e a flag correspondente NÃO foi setada explicitamente.
		// Override explícito via flag prevalece (ADR-015 TD-04).
		// Defaults Claude/Codex/Copilot (150/200 linhas, 12/16 KiB) preservados (RF-30).
		if tool == "gemini" {
			if !cmd.Flags().Changed("memory-workflow-limit-lines") {
				memWorkflowLimitLines = 250
			}
			if !cmd.Flags().Changed("memory-task-limit-lines") {
				memTaskLimitLines = 400
			}
			if !cmd.Flags().Changed("memory-workflow-limit-bytes") {
				memWorkflowLimitBytes = 20 * 1024
			}
			if !cmd.Flags().Changed("memory-task-limit-bytes") {
				memTaskLimitBytes = 32 * 1024
			}
		}

		// Validação enum --reasoning-effort (RF-09, RF-10 — ADR-013 D-08)
		validReasoning := map[string]bool{"low": true, "medium": true, "high": true}
		if !validReasoning[reasoningEffort] {
			_, _ = fmt.Fprintf(os.Stderr,
				"--reasoning-effort inválido: %q — valores aceitos: low|medium|high\n", reasoningEffort)
			return fmt.Errorf("exit2")
		}

		// Validação enum --access-mode (RF-11, RF-13 — ADR-013 D-08)
		validAccess := map[string]bool{"restricted": true, "full": true}
		if !validAccess[accessMode] {
			_, _ = fmt.Fprintf(os.Stderr,
				"--access-mode inválido: %q — valores aceitos: restricted|full\n", accessMode)
			return fmt.Errorf("exit2")
		}

		// Warning único para --access-mode=full via sync.Once (R-03 alto, ADR-013 D-08, PRD HU-03/Q1)
		if accessMode == "full" {
			accessModeFullWarnOnce.Do(func() {
				effectiveTool := tool
				if effectiveTool == "" {
					effectiveTool = execTool
				}
				switch effectiveTool {
				case "gemini":
					// RF-33 (ADR-015): warning específico para Gemini --approval-mode=yolo.
					_, _ = fmt.Fprintln(os.Stderr,
						"WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. "+
							"Pré-condição: consentimento operacional. Ver GEMINI.md.")
				default:
					_, _ = fmt.Fprintln(os.Stderr,
						"WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. "+
							"Pré-condição: consentimento operacional. Codex terá acesso pleno ao filesystem e à rede. "+
							"Use somente em ambientes isolados. Ver CODEX.md.")
				}
			})
		}

		// Validacao de --runtime (RF-01, RF-02, RF-07)
		if runtime != "legacy" && runtime != "acp" {
			_, _ = fmt.Fprintf(os.Stderr, "runtime inválido: %q — valores aceitos: legacy, acp\n", runtime)
			return fmt.Errorf("exit2")
		}
		if runtime == "acp" {
			effectiveTool := tool
			if effectiveTool == "" {
				effectiveTool = execTool
			}
			if _, ok := runtimeACPCatalog[effectiveTool]; !ok {
				supported := make([]string, 0, len(runtimeACPCatalog))
				for k := range runtimeACPCatalog {
					supported = append(supported, k)
				}
				sort.Strings(supported)
				_, _ = fmt.Fprintf(os.Stderr,
					"runtime acp suporta apenas --tool em %v nesta versão\n", supported)
				return fmt.Errorf("exit2")
			}
		}
		if activityTimeout < 0 {
			_, _ = fmt.Fprintf(os.Stderr, "--activity-timeout não pode ser negativo\n")
			return fmt.Errorf("exit2")
		}

		// Validacao mutua exclusiva de --agent com --tool e modo avancado (D-06)
		if agentName != "" && (tool != "" || execTool != "" || revTool != "") {
			_, _ = fmt.Fprintf(os.Stderr, "--agent e mutuamente exclusivo com --tool, --executor-tool e --reviewer-tool\n")
			return fmt.Errorf("%w", taskloop.ErrFlagsConflitantes)
		}

		// Validacao mutua exclusiva entre modo simples e avancado
		if tool != "" && (execTool != "" || revTool != "") {
			return fmt.Errorf("--tool e --executor-tool/--reviewer-tool sao mutuamente exclusivas")
		}
		if tool == "" && execTool == "" && agentName == "" {
			return fmt.Errorf("informe --tool (modo simples), --executor-tool (modo avancado) ou --agent (agente declarativo)")
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
		err = svc.Execute(taskloop.Options{
			PRDFolder:                prdFolder,
			Tool:                     tool,
			DryRun:                   dryRun,
			MaxIterations:            maxIter,
			Timeout:                  timeout,
			ReportPath:               reportPath,
			Profiles:                 profiles,
			FallbackTool:             fallbackTool,
			AllowUnknownModel:        allowUnknown,
			ReviewerPromptTemplate:   reviewerTmpl,
			ExecutorFallbackModel:    execFallbackModel,
			ReviewerFallbackModel:    revFallbackModel,
			Runtime:                  runtime,
			ActivityTimeout:          activityTimeout,
			ActivityTimeoutSet:       activityTimeoutSet,
			Quiet:                    quiet,
			AgentName:                agentName,
			ReasoningEffort:          reasoningEffort,
			AccessMode:               accessMode,
			MCPNested:                mcpNested,
			NoNormalize:              noNormalize,
			MemoryWorkflowLimitLines: memWorkflowLimitLines,
			MemoryWorkflowLimitBytes: memWorkflowLimitBytes,
			MemoryTaskLimitLines:     memTaskLimitLines,
			MemoryTaskLimitBytes:     memTaskLimitBytes,
			MemoryLimitsSet:          memoryLimitsSet,
			DisableHooks:             disableHooks,
			SkipDriftGuard:           skipDriftGuard,
			AutoReview:               autoReview,
		})
		if errors.Is(err, airuntime.ErrLauncherUnavailable) {
			_, _ = fmt.Fprintln(os.Stderr, err)
			return fmt.Errorf("exit2")
		}
		return err
	},
}

func init() {
	// Flags existentes (preservadas)
	taskLoopCmd.Flags().String("tool", "", "Agente de IA: claude, codex, gemini, copilot (modo simples)")
	taskLoopCmd.Flags().String("agent", "", "Nome do agente declarativo (AGENT.md); mutuamente exclusivo com --tool e --executor-tool")
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

	// Flags ACP runtime (RF-01, RF-02, RF-07, RF-11)
	taskLoopCmd.Flags().String("runtime", "legacy", "Runtime de invocacao: legacy (default) ou acp (tools suportados: claude, codex, copilot)")
	taskLoopCmd.Flags().Duration("activity-timeout", 120*time.Second, "Timeout de inatividade do agente ACP (0 = desabilitado); aceita time.Duration: 90s, 2m")
	taskLoopCmd.Flags().Bool("quiet", false, "Suprime stream humano (stdout); jsonl e warnings continuam")

	// Flags Codex-específicas (RF-09, RF-10, RF-11, RF-13 — ADR-013 D-08).
	// Para Claude/Copilot são aceitas mas sem efeito (BootstrapArgs no-op).
	taskLoopCmd.Flags().String("reasoning-effort", "medium",
		"Esforço de raciocínio do Codex: low|medium|high (default: medium). Apenas Codex consome este parâmetro; ignorado por Claude/Copilot.")
	taskLoopCmd.Flags().String("access-mode", "restricted",
		"Modo de acesso do Codex: restricted|full (default: restricted). AVISO: full ativa sandbox_mode=danger-full-access — use somente em ambientes isolados. Apenas Codex consome este parâmetro.")

	// Flags F2-Claude (RF-01, RF-02 — ADR-014).
	taskLoopCmd.Flags().Bool("mcp-nested", false,
		"Habilita servidor MCP interno que expõe tool run_agent (F2-Claude). Quando true, spawna mcpserver.Server antes de c.Open. Default false preserva comportamento F1-Claude.")
	taskLoopCmd.Flags().Bool("no-normalize", false,
		"Desabilita normalização de tool-calls driver-aware (F2-Claude, debug). Default false = normalização ativa (raw_name e normalized_name gravados lado a lado).")

	// Flags F3-Claude: limites de memória e controle de hooks (RF-01, RF-02 — F3-Claude).
	// Defaults espelham Compozy: 150 linhas/12KB workflow; 200 linhas/16KB task.
	// Zero-value em Job.MemoryLimits aplica os defaults de memory.DefaultLimits() (fallback no runner).
	taskLoopCmd.Flags().Int("memory-workflow-limit-lines", 150,
		"Limite de linhas do arquivo de workflow memory antes de solicitar compactação (F3-Claude). Default 150.")
	taskLoopCmd.Flags().Int("memory-workflow-limit-bytes", 12288,
		"Limite de bytes do arquivo de workflow memory antes de solicitar compactação (F3-Claude). Default 12288 (12KB).")
	taskLoopCmd.Flags().Int("memory-task-limit-lines", 200,
		"Limite de linhas do arquivo de task memory antes de solicitar compactação (F3-Claude). Default 200.")
	taskLoopCmd.Flags().Int("memory-task-limit-bytes", 16384,
		"Limite de bytes do arquivo de task memory antes de solicitar compactação (F3-Claude). Default 16384 (16KB).")
	taskLoopCmd.Flags().Bool("disable-hooks", false,
		"Desabilita TODOS os hooks Go in-process: governance, token_budget e memory_persist (F3-Claude, debug). "+
			"AVISO: --disable-hooks desliga inclusive o hook de governance (validação AGENTS.md). "+
			"Shell hooks em .claude/hooks/*.sh continuam ativos no modo interativo. Default false.")

	// Flag de bypass do guard de governança em runtime (ADR-022, RG-01/RG-02).
	// Desabilita SOMENTE o spec_drift; governance/token_budget permanecem ativos.
	taskLoopCmd.Flags().Bool("skip-drift-guard", false,
		"Desabilita SOMENTE o hook spec_drift (spec-hash/PRD-first, ADR-022), mantendo governance e "+
			"token_budget ativos. Use em CI sem PRD rastreável ou durante desenvolvimento inicial. "+
			"Diferente de --disable-hooks (que desliga todos os hooks). Default false.")

	// Flag F5-Claude: auto-review opt-in (RF-06 — ADR-014 §D-07).
	// HARD: default false; child session de review tem AutoReview=false forçado (anti-recursão).
	// HARD: Claude NÃO modifica internal/wrapper/ValidTools (ADR-014 §D-07).
	taskLoopCmd.Flags().Bool("auto-review", false,
		"Habilita auto-review opt-in (F5-Claude): após session end, spawna nova sessão com skill review "+
			"e git diff acumulado. Parseia [HARD]/BLOQUEADO/CRÍTICO → Summary.ReviewStatus=blocked. "+
			"HARD: default false; sessões filho têm auto-review=false forçado (anti-recursão). "+
			"Dobra custo de tokens — usar somente quando necessário. Ver ADR-014 §D-07.")

	rootCmd.AddCommand(taskLoopCmd)
}
