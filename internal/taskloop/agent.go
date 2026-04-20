package taskloop

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AgentInvoker abstrai a invocacao de um agente de IA via CLI.
type AgentInvoker interface {
	Invoke(ctx context.Context, prompt, workDir string) (stdout, stderr string, exitCode int, err error)
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
func BuildPrompt(taskFilePath, prdFolder string) string {
	return fmt.Sprintf(`You are executing the "execute-task" skill.

Read and follow the instructions in: .agents/skills/execute-task/SKILL.md

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

Update **Status:** in %s and the corresponding row in %s/tasks.md to reflect the final state.`, taskFilePath, prdFolder, taskFilePath, prdFolder)
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

func runCmd(ctx context.Context, workDir string, name string, args ...string) (string, string, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	cmd.Env = cleanEnv()

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return stdoutBuf.String(), stderrBuf.String(), -1, err
		}
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode, nil
}

// --- Claude ---

type claudeInvoker struct{}

func (c *claudeInvoker) BinaryName() string { return "claude" }

func (c *claudeInvoker) Invoke(ctx context.Context, prompt, workDir string) (string, string, int, error) {
	return runCmd(ctx, workDir, "claude", "--dangerously-skip-permissions", "--print", "-p", prompt)
}

// --- Codex ---

type codexInvoker struct{}

func (c *codexInvoker) BinaryName() string { return "codex" }

func (c *codexInvoker) Invoke(ctx context.Context, prompt, workDir string) (string, string, int, error) {
	return runCmd(ctx, workDir, "codex", "exec", "--dangerously-bypass-approvals-and-sandbox", prompt)
}

// --- Gemini ---

type geminiInvoker struct{}

func (g *geminiInvoker) BinaryName() string { return "gemini" }

func (g *geminiInvoker) Invoke(ctx context.Context, prompt, workDir string) (string, string, int, error) {
	return runCmd(ctx, workDir, "gemini", "--yolo", "-p", prompt)
}

// --- Copilot ---

type copilotInvoker struct{}

func (c *copilotInvoker) BinaryName() string { return "copilot" }

func (c *copilotInvoker) Invoke(ctx context.Context, prompt, workDir string) (string, string, int, error) {
	return runCmd(ctx, workDir, "copilot", "-p", prompt, "--yolo")
}
