package taskloop

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

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

// BuildPrompt constroi o prompt para o agente executar uma task especifica.
// Resolve referencias dinamicamente a partir da stack do projeto e do conteudo
// de prd.md e techspec.md (RF-04, contrato de carga base).
func BuildPrompt(taskFilePath, prdFolder, workDir string, fsys fs.FileSystem) string {
	refs := resolveReferences(prdFolder, workDir, fsys)
	return fmt.Sprintf(`Use a skill execute-task para implementar a task %s.

Contexto obrigatorio:
- Leia o arquivo de task antes de iniciar qualquer alteracao
- Arquitetura: descreva a camada e os contratos relevantes apos ler o arquivo de task
- Referencias a carregar: %s

Criterios de execucao nao negociaveis:
- preservar contratos publicos existentes (nenhuma assinatura publica muda sem ADR)
- nenhuma interface nova sem fronteira real justificada
- context.Context em todas as operacoes de IO
- testes table-driven para todos os cenarios do criterio de pronto
- registrar evidencia de conclusao no arquivo de task (output do teste, lint)
- nao fechar a task sem evidencia de validacao e verificar se continua o mesmo comportamento

PRD folder: %s`, taskFilePath, strings.Join(refs, ", "), prdFolder)
}

// resolveReferences determina as referencias a carregar com base na stack do projeto
// (go.mod, package.json, pyproject.toml) e no conteudo de prd.md + techspec.md.
func resolveReferences(prdFolder, workDir string, fsys fs.FileSystem) []string {
	var refs []string

	// Stack detection: linguagem principal determina a skill de implementacao
	switch {
	case fsys.Exists(filepath.Join(workDir, "go.mod")):
		refs = append(refs, "go-implementation")
	case fsys.Exists(filepath.Join(workDir, "package.json")):
		refs = append(refs, "node-implementation")
	case fsys.Exists(filepath.Join(workDir, "pyproject.toml")),
		fsys.Exists(filepath.Join(workDir, "requirements.txt")):
		refs = append(refs, "python-implementation")
	}

	// Carregar conteudo de prd.md e techspec.md para deteccao semantica
	content := readProjectContext(prdFolder, fsys)

	// DDD: padroes de dominio mencionados
	if containsAny(content, "aggregate", "entity", "value object", "bounded context",
		"domain event", "repository pattern", "ubiquitous language", "ddd") {
		refs = append(refs, "ddd")
	}

	// tests e sempre necessario (criterio de pronto exige testes table-driven)
	refs = append(refs, "tests")

	// Observabilidade: traces, metrics, logs, OpenTelemetry
	if containsAny(content, "telemetry", "opentelemetry", "otel", "tracing", "metrics",
		"prometheus", "grafana", "observabilidade", "instrumentacao") {
		refs = append(refs, "observability")
	}

	// Seguranca: autenticacao, autorizacao, criptografia
	if containsAny(content, "security", "authentication", "authorization", "jwt",
		"oauth", "rbac", "encryption", "seguranca", "autenticacao", "autorizacao") {
		refs = append(refs, "security")
	}

	// Persistencia: banco de dados, migrations, queries
	if containsAny(content, "database", "sql", "migration", "persistence", "repository",
		"postgres", "mysql", "sqlite", "mongodb", "banco de dados", "persistencia") {
		refs = append(refs, "persistence")
	}

	// Concorrencia: goroutines, channels, workers
	if containsAny(content, "goroutine", "channel", "mutex", "concurrent", "worker pool",
		"concorrencia", "paralelismo", "async") {
		refs = append(refs, "concurrency")
	}

	// API: endpoints HTTP, gRPC, REST
	if containsAny(content, "http", "rest", "grpc", "endpoint", "handler", "middleware",
		"api gateway", "rota", "router") {
		refs = append(refs, "api")
	}

	return refs
}

// readProjectContext le prd.md e techspec.md e retorna o conteudo combinado em lowercase.
func readProjectContext(prdFolder string, fsys fs.FileSystem) string {
	var sb strings.Builder
	for _, name := range []string{"prd.md", "techspec.md"} {
		if data, err := fsys.ReadFile(filepath.Join(prdFolder, name)); err == nil {
			sb.Write(data)
			sb.WriteByte('\n')
		}
	}
	return strings.ToLower(sb.String())
}

// containsAny retorna true se content contiver ao menos um dos termos (case-insensitive).
// Espera content ja em lowercase para evitar conversao repetida.
func containsAny(content string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(content, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// LiveOutputSetter permite configurar um writer para streaming de output do agente.
// Invokers que implementam esta interface recebem live output via SetLiveOutput
// antes do loop principal. O writer e passado internamente a runCmd para tee do stdout.
type LiveOutputSetter interface {
	SetLiveOutput(w io.Writer)
}

type processStartHookSetter interface {
	SetProcessStartHook(fn func())
}

type invocationErrorRecorder struct {
	mu  sync.Mutex
	err error
}

func (r *invocationErrorRecorder) Record(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err == nil {
		r.err = err
	}
}

func (r *invocationErrorRecorder) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

type lifecycleOutputWriter struct {
	target   io.Writer
	onOutput func(string)
	mu       sync.Mutex
}

type synchronizedWriter struct {
	target io.Writer
	mu     sync.Mutex
}

func newSynchronizedWriter(target io.Writer) io.Writer {
	if target == nil {
		return nil
	}
	return &synchronizedWriter{target: target}
}

func (w *synchronizedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.target.Write(p)
}

func newLifecycleOutputWriter(target io.Writer, onOutput func(string)) *lifecycleOutputWriter {
	return &lifecycleOutputWriter{
		target:   target,
		onOutput: onOutput,
	}
}

func (w *lifecycleOutputWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(strings.TrimSpace(string(p))) > 0 && w.onOutput != nil {
		w.onOutput(observedOutputMessage(p))
	}
	if w.target == nil {
		return len(p), nil
	}
	return w.target.Write(p)
}

func observedOutputMessage(chunk []byte) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(string(chunk))), " ")
	if normalized == "" {
		return "saida observada do agente"
	}
	return fmt.Sprintf("saida observada: %s", truncate(normalized, 120))
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

func mapLoopFailure(tool ToolName, exitCode int, stdout, stderr string, invokeErr error, postStatus string) *LoopFailure {
	combined := strings.TrimSpace(stdout + "\n" + stderr)
	switch {
	case isAuthError(combined):
		return NewLoopFailure(
			ErrorToolAuthRequired,
			fmt.Sprintf("%s requer autenticacao: %s", tool, authGuidance(string(tool))),
			invokeErr,
		)
	case errors.Is(invokeErr, exec.ErrNotFound), isMissingBinaryError(invokeErr):
		return NewLoopFailure(
			ErrorToolBinaryMissing,
			fmt.Sprintf("binario de %s nao encontrado no PATH", tool),
			invokeErr,
		)
	case errors.Is(invokeErr, context.DeadlineExceeded), errors.Is(invokeErr, context.Canceled), exitCode == -1:
		return NewLoopFailure(
			ErrorToolTimeout,
			fmt.Sprintf("tempo limite da iteracao excedido para %s", tool),
			invokeErr,
		)
	case invokeErr != nil:
		return NewLoopFailure(
			ErrorToolExecutionFailed,
			fmt.Sprintf("falha ao executar %s", tool),
			invokeErr,
		)
	case exitCode != 0:
		return NewLoopFailure(
			ErrorToolExecutionFailed,
			fmt.Sprintf("%s encerrou com falha (exit=%d)", tool, exitCode),
			nil,
		)
	case normalizeStatus(postStatus) != "done":
		return NewLoopFailure(
			ErrorToolExecutionFailed,
			fmt.Sprintf("%s encerrou sem concluir a task", tool),
			nil,
		)
	default:
		return nil
	}
}

func isMissingBinaryError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "executable file not found") ||
		strings.Contains(lower, "file not found") ||
		strings.Contains(lower, "no such file or directory")
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
	startHook     func()
}

func (c *claudeInvoker) BinaryName() string {
	if _, err := exec.LookPath("claudiney"); err == nil {
		return "claudiney"
	}
	return "claude"
}

func (c *claudeInvoker) SetLiveOutput(w io.Writer) { c.liveOut = w }

func (c *claudeInvoker) SetProcessStartHook(fn func()) { c.startHook = fn }

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
	return runCmdMonitored(ctx, workDir, c.liveOut, c.startHook, bin, args...)
}

// --- Codex ---

type codexInvoker struct {
	liveOut   io.Writer
	startHook func()
}

func (c *codexInvoker) BinaryName() string { return "codex" }

func (c *codexInvoker) SetLiveOutput(w io.Writer) { c.liveOut = w }

func (c *codexInvoker) SetProcessStartHook(fn func()) { c.startHook = fn }

func (c *codexInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	args := make([]string, 0, 5)
	args = append(args, "exec")
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "--yolo", prompt)
	return runCmdMonitored(ctx, workDir, c.liveOut, c.startHook, "codex", args...)
}

// --- Gemini ---

type geminiInvoker struct {
	liveOut   io.Writer
	startHook func()
}

func (g *geminiInvoker) BinaryName() string { return "gemini" }

func (g *geminiInvoker) SetLiveOutput(w io.Writer) { g.liveOut = w }

func (g *geminiInvoker) SetProcessStartHook(fn func()) { g.startHook = fn }

func (g *geminiInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	args := make([]string, 0, 5)
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "--approval-mode=yolo", "-p", prompt)
	return runCmdMonitored(ctx, workDir, g.liveOut, g.startHook, "gemini", args...)
}

// --- Copilot ---

type copilotInvoker struct {
	liveOut   io.Writer
	startHook func()
}

func (c *copilotInvoker) BinaryName() string { return "copilot" }

func (c *copilotInvoker) SetLiveOutput(w io.Writer) { c.liveOut = w }

func (c *copilotInvoker) SetProcessStartHook(fn func()) { c.startHook = fn }

func (c *copilotInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
	args := make([]string, 0, 6)
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "--autopilot", "--yolo", "-p", prompt)
	return runCmdMonitored(ctx, workDir, c.liveOut, c.startHook, "copilot", args...)
}
