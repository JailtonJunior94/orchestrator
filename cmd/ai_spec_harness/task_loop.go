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
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
	"github.com/spf13/cobra"
)

var _accessModeFullWarnOnce sync.Once

var runtimeACPCatalog = specs.NewCatalog().ACPSpecCatalog()

type taskLoopCommand struct{}

func newTaskLoopCmd() *cobra.Command {
	handler := &taskLoopCommand{}
	cmd := &cobra.Command{
		Use:   "task-loop <prd-folder>",
		Short: "Executa tasks de um PRD folder sequencialmente via agente de IA",
		Long: `Executa em loop todas as tasks elegiveis de uma pasta PRD, invocando
um agente de IA (Claude Code, Codex, Copilot ou OpenCode) para cada task.

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
  ai-spec task-loop --tool claude .specs/prd-minha-feature
  ai-spec task-loop --tool codex --dry-run .specs/prd-minha-feature
  ai-spec task-loop --executor-tool claude --executor-model claude-sonnet-4-6 \
    --reviewer-tool claude --reviewer-model claude-opus-4-6 .specs/prd-minha-feature`,
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
			maxBugfixIterations, _ := cmd.Flags().GetInt("max-bugfix-iterations")
			maxBugfixIterationsSet := cmd.Flags().Changed("max-bugfix-iterations")
			durableMemory, _ := cmd.Flags().GetBool("durable-memory")
			durableMemorySet := cmd.Flags().Changed("durable-memory")

			validReasoning := map[string]bool{"low": true, "medium": true, "high": true}
			if !validReasoning[reasoningEffort] {
				_, _ = fmt.Fprintf(os.Stderr,
					"--reasoning-effort inválido: %q — valores aceitos: low|medium|high\n", reasoningEffort)
				return newExitError(2)
			}

			validAccess := map[string]bool{"restricted": true, "full": true}
			if !validAccess[accessMode] {
				_, _ = fmt.Fprintf(os.Stderr,
					"--access-mode inválido: %q — valores aceitos: restricted|full\n", accessMode)
				return newExitError(2)
			}

			if accessMode == "full" {
				_accessModeFullWarnOnce.Do(func() {
					_, _ = fmt.Fprintln(os.Stderr,
						"WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. "+
							"Pré-condição: consentimento operacional. Codex terá acesso pleno ao filesystem e à rede. "+
							"Use somente em ambientes isolados. Ver CODEX.md.")
				})
			}

			if runtime != "legacy" && runtime != "acp" {
				_, _ = fmt.Fprintf(os.Stderr, "runtime inválido: %q — valores aceitos: legacy, acp\n", runtime)
				return newExitError(2)
			}
			if runtime == "acp" {
				effectiveTool := tool
				if effectiveTool == "" {
					effectiveTool = execTool
				}
				if _, ok := runtimeACPCatalog[effectiveTool]; !ok {
					if _, resolveErr := skills.NewCatalog().ResolveTool(effectiveTool); resolveErr != nil {
						var removedErr *skills.RemovedAgentError
						if errors.As(resolveErr, &removedErr) {
							return removedErr
						}
					}
					supported := make([]string, 0, len(runtimeACPCatalog))
					for k := range runtimeACPCatalog {
						supported = append(supported, k)
					}
					sort.Strings(supported)
					_, _ = fmt.Fprintf(os.Stderr,
						"runtime acp suporta apenas --tool em %v nesta versão\n", supported)
					return newExitError(2)
				}
			}
			if activityTimeout < 0 {
				_, _ = fmt.Fprintf(os.Stderr, "--activity-timeout não pode ser negativo\n")
				return newExitError(2)
			}

			if maxBugfixIterationsSet && maxBugfixIterations < 1 {
				_, _ = fmt.Fprintf(os.Stderr,
					"--max-bugfix-iterations inválido: %d — minimo aceito: 1\n", maxBugfixIterations)
				return newExitError(2)
			}
			if !maxBugfixIterationsSet {
				maxBugfixIterations = 0
			}

			if agentName != "" && (tool != "" || execTool != "" || revTool != "") {
				_, _ = fmt.Fprintf(os.Stderr, "--agent e mutuamente exclusivo com --tool, --executor-tool e --reviewer-tool\n")
				return fmt.Errorf("%w", taskloop.ErrFlagsConflitantes)
			}

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

			if tool != "" && runtime == "legacy" && !taskloop.ValidTools[tool] {
				if _, err := skills.NewCatalog().ResolveTool(tool); err != nil {
					return err
				}
				return fmt.Errorf("ferramenta invalida %q — opcoes: claude, codex, copilot", tool)
			}

			profiles, err := taskloop.NewCatalog().ResolveProfiles(tool, execTool, execModel, revTool, revModel)
			if err != nil {
				return err
			}

			if reportPath == "" {
				reportPath = fmt.Sprintf("task-loop-report-%s.md", time.Now().Format("20060102-150405"))
			}

			printer := output.New(newCommandEnv().verbose(cmd))
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
				MaxBugfixIterations:      maxBugfixIterations,
				MaxBugfixIterationsSet:   maxBugfixIterationsSet,
				DurableMemoryEnabled:     durableMemory,
				DurableMemoryEnabledSet:  durableMemorySet,
			})
			if errors.Is(err, airuntime.ErrLauncherUnavailable) {
				_, _ = fmt.Fprintln(os.Stderr, err)
				return newExitError(2)
			}
			return err
		},
	}

	handler.registerFlags(cmd)
	return cmd
}

func (c *taskLoopCommand) registerFlags(cmd *cobra.Command) {

	cmd.Flags().String("tool", "", "Agente de IA: claude, codex, copilot, opencode (modo simples)")
	cmd.Flags().String("agent", "", "Nome do agente declarativo (AGENT.md); mutuamente exclusivo com --tool e --executor-tool")
	cmd.Flags().Bool("dry-run", false, "Mostra o que seria executado sem invocar o agente")
	cmd.Flags().Int("max-iterations", 20, "Limite maximo de iteracoes do loop")
	cmd.Flags().Duration("timeout", 30*time.Minute, "Timeout por task")
	cmd.Flags().String("report-path", "", "Caminho do relatorio final (default: task-loop-report-<timestamp>.md)")

	cmd.Flags().String("executor-tool", "", "Ferramenta do executor (modo avancado): claude, codex, copilot, opencode")
	cmd.Flags().String("executor-model", "", "Modelo do executor (ex: claude-sonnet-4-6)")
	cmd.Flags().String("reviewer-tool", "", "Ferramenta do reviewer (modo avancado): claude, codex, copilot, opencode")
	cmd.Flags().String("reviewer-model", "", "Modelo do reviewer (ex: claude-opus-4-6)")
	cmd.Flags().String("fallback-tool", "", "Ferramenta de fallback para validacao pre-loop")
	cmd.Flags().Bool("allow-unknown-model", false, "Aceitar combinacoes ferramenta-modelo nao catalogadas")
	cmd.Flags().String("reviewer-prompt-template", "", "Caminho do template de prompt de revisao customizado")
	cmd.Flags().String("executor-fallback-model", "", "Modelo de fallback nativo do executor (Claude only)")
	cmd.Flags().String("reviewer-fallback-model", "", "Modelo de fallback nativo do reviewer (Claude only)")

	cmd.Flags().String("runtime", "legacy", "Runtime de invocacao: legacy (default) ou acp (tools suportados: claude, codex, copilot, opencode)")
	cmd.Flags().Duration("activity-timeout", 120*time.Second, "Timeout de inatividade do agente ACP (0 = desabilitado); aceita time.Duration: 90s, 2m")
	cmd.Flags().Bool("quiet", false, "Suprime stream humano (stdout); jsonl e warnings continuam")

	cmd.Flags().String("reasoning-effort", "medium",
		"Esforço de raciocínio do Codex: low|medium|high (default: medium). Apenas Codex consome este parâmetro; ignorado por Claude/Copilot.")
	cmd.Flags().String("access-mode", "restricted",
		"Modo de acesso do Codex: restricted|full (default: restricted). AVISO: full ativa sandbox_mode=danger-full-access — use somente em ambientes isolados. Apenas Codex consome este parâmetro.")

	cmd.Flags().Bool("mcp-nested", false,
		"Habilita servidor MCP interno que expõe tool run_agent (F2-Claude). Quando true, spawna mcpserver.Server antes de c.Open. Default false preserva comportamento F1-Claude.")
	cmd.Flags().Bool("no-normalize", false,
		"Desabilita normalização de tool-calls driver-aware (F2-Claude, debug). Default false = normalização ativa (raw_name e normalized_name gravados lado a lado).")

	cmd.Flags().Int("memory-workflow-limit-lines", 150,
		"Limite de linhas do arquivo de workflow memory antes de solicitar compactação (F3-Claude). Default 150.")
	cmd.Flags().Int("memory-workflow-limit-bytes", 12288,
		"Limite de bytes do arquivo de workflow memory antes de solicitar compactação (F3-Claude). Default 12288 (12KB).")
	cmd.Flags().Int("memory-task-limit-lines", 200,
		"Limite de linhas do arquivo de task memory antes de solicitar compactação (F3-Claude). Default 200.")
	cmd.Flags().Int("memory-task-limit-bytes", 16384,
		"Limite de bytes do arquivo de task memory antes de solicitar compactação (F3-Claude). Default 16384 (16KB).")
	cmd.Flags().Bool("disable-hooks", false,
		"Desabilita TODOS os hooks Go in-process: governance, token_budget e memory_persist (F3-Claude, debug). "+
			"AVISO: --disable-hooks desliga inclusive o hook de governance (validação AGENTS.md). "+
			"Shell hooks em .claude/hooks/*.sh continuam ativos no modo interativo. Default false.")

	cmd.Flags().Bool("skip-drift-guard", false,
		"Desabilita SOMENTE o hook spec_drift (spec-hash/PRD-first, ADR-022), mantendo governance e "+
			"token_budget ativos. Use em CI sem PRD rastreável ou durante desenvolvimento inicial. "+
			"Diferente de --disable-hooks (que desliga todos os hooks). Default false.")

	cmd.Flags().Bool("auto-review", false,
		"Habilita auto-review opt-in (F5-Claude): após session end, spawna nova sessão com skill review "+
			"e git diff acumulado. Parseia [HARD]/BLOQUEADO/CRÍTICO → Summary.ReviewStatus=blocked. "+
			"HARD: default false; sessões filho têm auto-review=false forçado (anti-recursão). "+
			"Dobra custo de tokens — usar somente quando necessário. Ver ADR-014 §D-07.")

	cmd.Flags().Int("max-bugfix-iterations", 5,
		"Teto de rodadas do Ciclo de Aprovacao (RF-35); minimo aceito 1; default 5. "+
			"Configuravel tambem via arquivo de configuracao (workspace/global), respeitando a precedencia "+
			"flags > workspace > global > defaults (ADR-016). Valor 0 ou negativo falha explicitamente "+
			"em vez de ser normalizado em silencio.")

	cmd.Flags().Bool("durable-memory", false,
		"Ativa o subsistema de memoria duravel (fachada + camadas project/prd/task) em vez do "+
			"memory store legado (RF-28). Default false preserva o prompt byte a byte identico. "+
			"Configuravel tambem via chave durable_memory_enabled na cascata de configuracao, "+
			"respeitando a precedencia flags > workspace > global > defaults (ADR-016).")
}
