package taskloop

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// AgentInvoker abstrai a invocacao de um agente de IA via CLI.
// Quando model == "", o invoker nao passa --model ao subprocesso (usa default da ferramenta).
// Quando model != "", o invoker insere --model <model> na posicao correta conforme cada ferramenta.
type AgentInvoker interface {
	Invoke(ctx context.Context, prompt, workDir, model string) (stdout, stderr string, exitCode int, err error)
	BinaryName() string
}

// NewAgentInvoker cria o invoker adequado para a ferramenta especificada.
func NewAgentInvoker(tool string) (AgentInvoker, error) {
	switch tool {
	case "claude":
		return &claudeInvoker{}, nil
	case "codex":
		return &codexInvoker{}, nil
	case "gemini":
		return &geminiInvoker{}, nil
	case "copilot":
		return &copilotInvoker{}, nil
	default:
		return nil, fmt.Errorf("ferramenta nao suportada: %q — opcoes: claude, codex, gemini, copilot", tool)
	}
}

// ValidTools eh o conjunto de ferramentas aceitas pelo task-loop.
var ValidTools = map[string]bool{
	"claude":  true,
	"codex":   true,
	"gemini":  true,
	"copilot": true,
}

// CheckAgentBinary verifica se o binario do agente esta disponivel no PATH.
func CheckAgentBinary(invoker AgentInvoker) error {
	bin := invoker.BinaryName()
	_, err := exec.LookPath(bin)
	if err != nil {
		return fmt.Errorf("binario %q nao encontrado no PATH — instale antes de usar --tool", bin)
	}
	return nil
}

// BuildPrompt constroi o prompt para o agente executar uma task especifica.
// Inclui instrucao explicita de leitura de AGENTS.md porque --bare pula o
// carregamento automatico de CLAUDE.md (RF-04, contrato de carga base).
func BuildPrompt(taskFilePath, prdFolder string) string {
	return fmt.Sprintf(`You are executing the "execute-task" skill.

First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/execute-task/SKILL.md

Target task file: %s
PRD folder: %s

Execute ONLY this task. Follow all skill steps:
1. Validate eligibility
2. Load context (prd.md, techspec.md)
3. Implement
4. Validate (tests, lint)
5. Review
6. Update task status in task file and tasks.md
7. Generate execution report

Do NOT modify any other task file.
Do NOT modify any row in tasks.md except the current task row.
Do NOT start the next task or mark any other row in tasks.md as in_progress.
Leave follow-up tasks unchanged for a future isolated session.

Update **Status:** in %s and the corresponding row in %s/tasks.md to reflect the final state.`, taskFilePath, prdFolder, taskFilePath, prdFolder)
}

// LiveOutputSetter permite configurar um writer para streaming de output do agente.
// Invokers que implementam esta interface recebem live output via SetLiveOutput
// antes do loop principal. O writer e passado internamente a runCmd para tee do stdout.
type LiveOutputSetter interface {
	SetLiveOutput(w io.Writer)
}

// isAuthError detecta erros de autenticacao conhecidos no output do agente.
// Retorna true quando o output contem padroes que indicam falha de login/token.
func isAuthError(output string) bool {
	patterns := []string{
		"not logged in",
		"please run /login",
		"not authenticated",
		"authentication required",
		"unauthorized",
		"login required",
		"auth token",
		"api key",
	}
	lower := strings.ToLower(output)
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

// authGuidance retorna instrucoes de autenticacao especificas por ferramenta.
func authGuidance(tool string) string {
	switch tool {
	case "claude":
		return "execute 'claude' em um terminal separado e faca login com '/login', ou defina ANTHROPIC_API_KEY no ambiente para uso nao-interativo"
	case "copilot":
		return "execute 'gh auth login' para autenticar o GitHub Copilot"
	case "gemini":
		return "execute 'gemini' em um terminal separado e siga o fluxo de autenticacao"
	case "codex":
		return "configure OPENAI_API_KEY ou execute 'codex auth' para autenticar"
	default:
		return "verifique a autenticacao da ferramenta antes de executar task-loop"
	}
}

// warnClaudeAuth retorna aviso quando autenticacao do claude parece indisponivel.
// Verificacao heuristica: ANTHROPIC_API_KEY ausente E ~/.claude/ ausente ou vazio.
// Nunca bloqueia a execucao — e um aviso antecipado, nao uma prova de autenticacao valida.
// Util para detectar falha de auth ANTES de iniciar o loop, evitando que a primeira
// iteracao falhe silenciosamente so na hora de invocar o agente.
func warnClaudeAuth() string {
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return "" // API key presente — autenticacao nao-interativa disponivel
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "" // nao conseguiu verificar; assume ok
	}
	entries, err := os.ReadDir(filepath.Join(home, ".claude"))
	if err != nil || len(entries) == 0 {
		return "ANTHROPIC_API_KEY nao definido e ~/.claude/ vazio ou ausente — " +
			"autenticacao de subprocesso pode falhar; " +
			authGuidance("claude")
	}
	return "" // diretorio existe com arquivos; sessao provavelmente disponivel
}

func cleanEnv() []string {
	env := os.Environ()
	var cleaned []string
	for _, e := range env {
		if strings.HasPrefix(e, "AI_INVOCATION_DEPTH=") {
			continue
		}
		cleaned = append(cleaned, e)
	}
	cleaned = append(cleaned, "AI_INVOCATION_DEPTH=0")
	return cleaned
}

// --- Claude ---

// claudeInvoker invoca o CLI do Claude Code.
// fallbackModel, quando nao vazio, propaga --fallback-model ao subprocesso (camada 1 de fallback).
type claudeInvoker struct {
	fallbackModel string
	liveOut       io.Writer
}

func (c *claudeInvoker) BinaryName() string {
	if _, err := exec.LookPath("claudiney"); err == nil {
		return "claudiney"
	}
	return "claude"
}

func (c *claudeInvoker) SetLiveOutput(w io.Writer) { c.liveOut = w }

func (c *claudeInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	bin := c.BinaryName()
	args := make([]string, 0, 9)
	if model != "" {
		args = append(args, "--model", model)
	}
	if c.fallbackModel != "" {
		args = append(args, "--fallback-model", c.fallbackModel)
	}
	if bin == "claudiney" {
		// claudiney ja inclui --dangerously-skip-permissions e usa CLAUDE_CONFIG_DIR
		// sem --bare para permitir autenticacao via keychain/config local
		args = append(args, "--print", "-p", prompt)
	} else {
		args = append(args, "--dangerously-skip-permissions", "--print", "--bare", "-p", prompt)
	}
	return runCmd(ctx, workDir, c.liveOut, bin, args...)
}

// --- Codex ---

type codexInvoker struct {
	liveOut io.Writer
}

func (c *codexInvoker) BinaryName() string { return "codex" }

func (c *codexInvoker) SetLiveOutput(w io.Writer) { c.liveOut = w }

func (c *codexInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	args := make([]string, 0, 5)
	args = append(args, "exec")
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "--yolo", prompt)
	return runCmd(ctx, workDir, c.liveOut, "codex", args...)
}

// --- Gemini ---

type geminiInvoker struct {
	liveOut io.Writer
}

func (g *geminiInvoker) BinaryName() string { return "gemini" }

func (g *geminiInvoker) SetLiveOutput(w io.Writer) { g.liveOut = w }

func (g *geminiInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	args := make([]string, 0, 5)
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "--approval-mode=yolo", "-p", prompt)
	return runCmd(ctx, workDir, g.liveOut, "gemini", args...)
}

// --- Copilot ---

type copilotInvoker struct {
	liveOut io.Writer
}

func (c *copilotInvoker) BinaryName() string { return "copilot" }

func (c *copilotInvoker) SetLiveOutput(w io.Writer) { c.liveOut = w }

func (c *copilotInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	args := make([]string, 0, 6)
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "--autopilot", "--yolo", "-p", prompt)
	return runCmd(ctx, workDir, c.liveOut, "copilot", args...)
}
