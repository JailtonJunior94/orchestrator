package taskloop

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
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

// defaultExecutorTemplate e o template embutido via go:embed para o prompt do executor.
//
//go:embed executor_template.tmpl
var defaultExecutorTemplate string

// PromptContext agrupa o contexto dinamico extraido de prd.md e techspec.md
// para enriquecer o prompt do executor com arquitetura e referencias relevantes.
type PromptContext struct {
	Architecture string
	References   string
}

// ExecutorTemplateData agrupa os placeholders do template do executor.
type ExecutorTemplateData struct {
	TaskFile     string
	PRDFolder    string
	Architecture string
	References   string
}

// BuildPromptContext le prd.md e techspec.md do prdFolder e extrai
// contexto de arquitetura e referencias relevantes para o prompt.
func BuildPromptContext(prdFolder, workDir string, fsys fs.FileSystem) PromptContext {
	ctx := PromptContext{}

	techspecPath := filepath.Join(workDir, prdFolder, "techspec.md")
	prdPath := filepath.Join(workDir, prdFolder, "prd.md")

	techspec, _ := fsys.ReadFile(techspecPath)
	prd, _ := fsys.ReadFile(prdPath)

	combined := string(techspec) + "\n" + string(prd)

	ctx.Architecture = extractArchitecture(string(techspec))
	ctx.References = detectReferences(combined)

	return ctx
}

// extractArchitecture extrai o resumo de arquitetura da techspec.
// Procura secoes "Arquitetura" ou "Resumo Executivo" e retorna
// um trecho conciso (max 1500 chars) para compor o prompt.
func extractArchitecture(techspec string) string {
	for _, heading := range []string{
		"## Arquitetura do Sistema",
		"## Arquitetura",
		"## Architecture",
		"## Resumo Executivo",
	} {
		idx := strings.Index(techspec, heading)
		if idx < 0 {
			continue
		}
		rest := techspec[idx:]
		nlIdx := strings.Index(rest, "\n")
		if nlIdx < 0 {
			continue
		}
		afterHeading := rest[nlIdx+1:]
		nextSection := strings.Index(afterHeading, "\n## ")
		var section string
		if nextSection >= 0 {
			section = strings.TrimSpace(afterHeading[:nextSection])
		} else {
			section = strings.TrimSpace(afterHeading)
		}
		if len(section) > 1500 {
			section = section[:1500] + "\n(...)"
		}
		if section != "" {
			return section
		}
	}
	return "ler techspec.md para contexto de arquitetura"
}

// detectReferences analisa o conteudo combinado de prd+techspec e retorna
// a lista de referencias relevantes para carregar no prompt.
func detectReferences(content string) string {
	lower := strings.ToLower(content)
	var refs []string

	if containsAnyPattern(lower, ".go", "go.mod", "golang", "internal/", "func ", "package ") {
		refs = append(refs, "go-implementation")
	}
	if containsAnyPattern(lower, ".ts", ".tsx", "node", "npm", "typescript") {
		refs = append(refs, "node-implementation")
	}
	if containsAnyPattern(lower, ".py", "python", "pip", "django", "flask") {
		refs = append(refs, "python-implementation")
	}

	if containsAnyPattern(lower, "domain", "aggregate", "entity", "value object", "bounded context", "ddd") {
		refs = append(refs, "ddd")
	}
	if containsAnyPattern(lower, "seguranca", "security", "auth", "credential", "vulnerab") {
		refs = append(refs, "security")
	}

	refs = append(refs, "tests")
	return strings.Join(refs, ", ")
}

// containsAnyPattern retorna true se s contem qualquer um dos patterns.
func containsAnyPattern(s string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// BuildPrompt constroi o prompt para o agente executar uma task especifica.
// Inclui instrucao explicita de leitura de AGENTS.md porque --bare pula o
// carregamento automatico de CLAUDE.md (RF-04, contrato de carga base).
// Arquitetura e referencias sao preenchidos dinamicamente a partir do PromptContext.
func BuildPrompt(taskFilePath, prdFolder string, ctx PromptContext) string {
	data := ExecutorTemplateData{
		TaskFile:     taskFilePath,
		PRDFolder:    prdFolder,
		Architecture: ctx.Architecture,
		References:   ctx.References,
	}

	tmpl, err := template.New("executor").Parse(defaultExecutorTemplate)
	if err != nil {
		return fmt.Sprintf("Use a skill execute-task para implementar a task %s. PRD folder: %s", taskFilePath, prdFolder)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Use a skill execute-task para implementar a task %s. PRD folder: %s", taskFilePath, prdFolder)
	}

	return buf.String()
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
