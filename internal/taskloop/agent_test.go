package taskloop

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// claudeBinary retorna o binario efetivo para claude: "claudiney" se disponivel, senao "claude".
func claudeBinary() string {
	if _, err := exec.LookPath("claudiney"); err == nil {
		return "claudiney"
	}
	return "claude"
}

// TestNewAgentInvoker valida que a factory retorna o invoker correto para cada ferramenta
// e erro tipado para ferramenta invalida.
func TestNewAgentInvoker(t *testing.T) {
	tests := []struct {
		name       string
		tool       string
		wantBinary string
		wantErr    bool
	}{
		{name: "claude", tool: "claude", wantBinary: claudeBinary(), wantErr: false},
		{name: "codex", tool: "codex", wantBinary: "codex", wantErr: false},
		{name: "gemini", tool: "gemini", wantBinary: "gemini", wantErr: false},
		{name: "copilot", tool: "copilot", wantBinary: "copilot", wantErr: false},
		{name: "ferramenta invalida", tool: "invalid-tool", wantBinary: "", wantErr: true},
		{name: "string vazia", tool: "", wantBinary: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker, err := NewAgentInvoker(tt.tool)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("esperado erro para tool=%q, mas nenhum erro retornado", tt.tool)
				}
				return
			}
			if err != nil {
				t.Fatalf("nao esperado erro para tool=%q: %v", tt.tool, err)
			}
			if invoker.BinaryName() != tt.wantBinary {
				t.Errorf("BinaryName() = %q, want %q", invoker.BinaryName(), tt.wantBinary)
			}
		})
	}
}

// writeFakeBinary cria um script executavel no dir informado que imprime cada argumento
// em uma linha separada para stdout. Retorna o path do script.
func writeFakeBinary(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	// printf '%s\n' "$@" imprime cada arg em linha separada, sem interpolacao de shell.
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("nao foi possivel criar fake binary %q: %v", path, err)
	}
}

// TestBuildPromptContainsAgentsMd verifica que o prompt gerado instrui leitura de AGENTS.md
// (contrato de carga base exigido por RF-04 quando --bare esta ativo).
func TestBuildPromptContainsAgentsMd(t *testing.T) {
	prompt := BuildPrompt("tasks/prd-feat/01_task.md", "tasks/prd-feat")

	required := []string{
		"AGENTS.md",
		".agents/skills/execute-task/SKILL.md",
		"tasks/prd-feat/01_task.md",
		"tasks/prd-feat",
		"Do NOT modify any other task file.",
		"Do NOT modify any row in tasks.md except the current task row.",
		"Do NOT start the next task or mark any other row in tasks.md as in_progress.",
		"Leave follow-up tasks unchanged for a future isolated session.",
	}
	for _, r := range required {
		if !strings.Contains(prompt, r) {
			t.Errorf("BuildPrompt() nao contem %q\nprompt:\n%s", r, prompt)
		}
	}

	// AGENTS.md deve aparecer ANTES da referencia a SKILL.md
	agentsIdx := strings.Index(prompt, "AGENTS.md")
	skillIdx := strings.Index(prompt, ".agents/skills/execute-task/SKILL.md")
	if agentsIdx >= skillIdx {
		t.Errorf("AGENTS.md (pos=%d) deve aparecer antes de SKILL.md (pos=%d) no prompt", agentsIdx, skillIdx)
	}
}

// TestInvokerArgs valida os argumentos exatos passados por cada invoker ao subprocesso.
// Usa fake binaries em TempDir para interceptar a invocacao sem depender de CLIs reais.
// Cobre dois cenarios por invoker: model vazio (regressao) e model preenchido (--model na posicao correta).
func TestInvokerArgs(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		prompt   string
		model    string
		wantArgs []string // argumentos esperados apos o binario, em ordem
	}{
		// --- model vazio (regressao — comportamento anterior preservado, exceto gemini) ---
		{
			name:   "claudeInvoker sem model — flags sem --bare quando claudiney disponivel",
			tool:   "claude",
			prompt: "execute task",
			model:  "",
			// claudiney fake criado no TempDir: o wrapper ja inclui --dangerously-skip-permissions
			// internamente, entao o invoker passa apenas --print -p <prompt>
			wantArgs: []string{"--print", "-p", "execute task"},
		},
		{
			name:   "codexInvoker sem model — prompt como argumento posicional",
			tool:   "codex",
			prompt: "execute task",
			model:  "",
			wantArgs: []string{
				"exec", "--yolo", "execute task",
			},
		},
		{
			name:   "geminiInvoker sem model — usa --approval-mode=yolo (correcao de --yolo deprecated)",
			tool:   "gemini",
			prompt: "execute task",
			model:  "",
			wantArgs: []string{
				"--approval-mode=yolo", "-p", "execute task",
			},
		},
		{
			name:   "copilotInvoker sem model — flags identicas ao anterior",
			tool:   "copilot",
			prompt: "execute task",
			model:  "",
			wantArgs: []string{
				"--autopilot", "--yolo", "-p", "execute task",
			},
		},
		// --- model preenchido (--model na posicao correta por ferramenta) ---
		{
			name:   "claudeInvoker com model — --model antes de --dangerously-skip-permissions",
			tool:   "claude",
			prompt: "execute task",
			model:  "claude-sonnet-4-6",
			// claudiney fake no TempDir: invoker passa --model antes de --print
			wantArgs: []string{
				"--model", "claude-sonnet-4-6", "--print", "-p", "execute task",
			},
		},
		{
			name:   "codexInvoker com model — exec + --model antes de --yolo",
			tool:   "codex",
			prompt: "execute task",
			model:  "gpt-5.4",
			wantArgs: []string{
				"exec", "--model", "gpt-5.4", "--yolo", "execute task",
			},
		},
		{
			name:   "geminiInvoker com model — --model antes de --approval-mode=yolo",
			tool:   "gemini",
			prompt: "execute task",
			model:  "gemini-2.5-pro",
			wantArgs: []string{
				"--model", "gemini-2.5-pro",
				"--approval-mode=yolo", "-p", "execute task",
			},
		},
		{
			name:   "copilotInvoker com model — --model antes de --autopilot",
			tool:   "copilot",
			prompt: "execute task",
			model:  "claude-sonnet-4.5",
			wantArgs: []string{
				"--model", "claude-sonnet-4.5",
				"--autopilot", "--yolo", "-p", "execute task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cria diretorio temporario com fake binary para a ferramenta.
			dir := t.TempDir()
			writeFakeBinary(t, dir, tt.tool)
			// Para claude, cria tambem fake claudiney no mesmo dir para que
			// exec.LookPath("claudiney") o encontre antes do real.
			if tt.tool == "claude" {
				writeFakeBinary(t, dir, "claudiney")
			}

			// Coloca o fake binary na frente do PATH original.
			origPath := os.Getenv("PATH")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)

			invoker, err := NewAgentInvoker(tt.tool)
			if err != nil {
				t.Fatalf("NewAgentInvoker(%q): %v", tt.tool, err)
			}

			stdout, _, exitCode, err := invoker.Invoke(context.Background(), tt.prompt, dir, tt.model)
			if err != nil {
				t.Fatalf("Invoke retornou erro inesperado: %v", err)
			}
			if exitCode != 0 {
				t.Fatalf("exit code inesperado: %d", exitCode)
			}

			// Cada argumento e impresso em uma linha pelo fake binary.
			got := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("numero de args: got %d, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.wantArgs), got, tt.wantArgs)
			}
			for i, want := range tt.wantArgs {
				if got[i] != want {
					t.Errorf("arg[%d]: got %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

// TestIsAuthError valida deteccao de padroes de erro de autenticacao conhecidos.
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{name: "claude not logged in", output: "Not logged in · Please run /login", want: true},
		{name: "not authenticated", output: "Error: not authenticated", want: true},
		{name: "authentication required", output: "authentication required: please login", want: true},
		{name: "unauthorized", output: "401 Unauthorized", want: true},
		{name: "login required", output: "Login required to continue", want: true},
		{name: "api key missing", output: "API key not configured", want: true},
		{name: "normal error — not auth", output: "compilation failed: syntax error", want: false},
		{name: "empty output", output: "", want: false},
		{name: "task output — not auth", output: "Task completed successfully", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAuthError(tt.output)
			if got != tt.want {
				t.Errorf("isAuthError(%q) = %v, want %v", tt.output, got, tt.want)
			}
		})
	}
}

// TestAuthGuidance valida que authGuidance retorna orientacao especifica por ferramenta.
func TestAuthGuidance(t *testing.T) {
	for _, tool := range []string{"claude", "copilot", "gemini", "codex"} {
		guidance := authGuidance(tool)
		if guidance == "" {
			t.Errorf("authGuidance(%q) retornou string vazia", tool)
		}
	}
	// Ferramenta desconhecida deve retornar orientacao generica
	guidance := authGuidance("desconhecido")
	if guidance == "" {
		t.Error("authGuidance para ferramenta desconhecida retornou string vazia")
	}
}

// TestAuthGuidanceClaudeAnthropicKey verifica que a orientacao do claude menciona
// ANTHROPIC_API_KEY como alternativa para autenticacao nao-interativa (BUG-001).
// Claude Code em modo subprocesso nao herda sessao do processo pai — ANTHROPIC_API_KEY
// e a alternativa suportada para uso programatico.
func TestAuthGuidanceClaudeAnthropicKey(t *testing.T) {
	guidance := authGuidance("claude")
	if !strings.Contains(guidance, "ANTHROPIC_API_KEY") {
		t.Errorf("authGuidance(\"claude\") deve mencionar ANTHROPIC_API_KEY como alternativa, obteve: %q", guidance)
	}
}

// TestWarnClaudeAuthWithAPIKey verifica que warnClaudeAuth retorna string vazia
// quando ANTHROPIC_API_KEY esta definido (autenticacao nao-interativa disponivel).
func TestWarnClaudeAuthWithAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-test-key")
	got := warnClaudeAuth()
	if got != "" {
		t.Errorf("warnClaudeAuth() com ANTHROPIC_API_KEY definido deve retornar \"\", obteve: %q", got)
	}
}

// TestLiveOutputSetterInterface valida que todos os invokers concretos implementam LiveOutputSetter.
func TestLiveOutputSetterInterface(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}
	for _, tool := range tools {
		invoker, err := NewAgentInvoker(tool)
		if err != nil {
			t.Fatalf("NewAgentInvoker(%q): %v", tool, err)
		}
		if _, ok := invoker.(LiveOutputSetter); !ok {
			t.Errorf("invoker %q nao implementa LiveOutputSetter", tool)
		}
	}
}

// TestCleanEnvResetsAIInvocationDepth verifica que cleanEnv() remove qualquer valor de
// AI_INVOCATION_DEPTH herdado do processo pai e sempre define como 0 (RF-04.4 do PRD sequencial).
func TestCleanEnvResetsAIInvocationDepth(t *testing.T) {
	// Simula processo pai com profundidade diferente de 0
	t.Setenv("AI_INVOCATION_DEPTH", "99")

	env := cleanEnv()

	var depthValues []string
	for _, e := range env {
		if strings.HasPrefix(e, "AI_INVOCATION_DEPTH=") {
			depthValues = append(depthValues, e)
		}
	}

	if len(depthValues) != 1 {
		t.Errorf("AI_INVOCATION_DEPTH deve aparecer exatamente uma vez, encontrado %d vezes: %v", len(depthValues), depthValues)
	}
	if len(depthValues) > 0 && depthValues[0] != "AI_INVOCATION_DEPTH=0" {
		t.Errorf("AI_INVOCATION_DEPTH deve ser 0, obteve %q", depthValues[0])
	}
}

// writeEnvPrinterBinary cria um script executavel que imprime o valor da variavel de ambiente
// informada no formato "VAR=VALUE". Ignora todos os argumentos recebidos.
func writeEnvPrinterBinary(t *testing.T, dir, name, envVar string) {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\nprintf '%s=%s\\n' " + envVar + " \"$" + envVar + "\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("nao foi possivel criar env printer binary %q: %v", path, err)
	}
}

// TestAgentEnvironmentIsolation verifica que AI_INVOCATION_DEPTH e resetado para 0
// no ambiente do subprocesso, mesmo que o processo pai tenha valor diferente (RF-04.4).
func TestAgentEnvironmentIsolation(t *testing.T) {
	// Definir valor diferente no processo pai
	t.Setenv("AI_INVOCATION_DEPTH", "5")

	dir := t.TempDir()
	writeEnvPrinterBinary(t, dir, "claude", "AI_INVOCATION_DEPTH")

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)

	inv := &claudeInvoker{}
	stdout, _, exitCode, err := inv.Invoke(context.Background(), "test-prompt", dir, "")
	if err != nil {
		t.Fatalf("Invoke retornou erro inesperado: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("exit code inesperado: %d", exitCode)
	}

	got := strings.TrimSpace(stdout)
	if got != "AI_INVOCATION_DEPTH=0" {
		t.Errorf("subprocesso deveria ver AI_INVOCATION_DEPTH=0, obteve: %q", got)
	}
}

// TestClaudeInvokerFallbackModel verifica que --fallback-model e propagado ao subprocesso
// quando fallbackModel esta configurado no claudeInvoker (camada 1 de fallback nativo).
func TestClaudeInvokerFallbackModel(t *testing.T) {
	tests := []struct {
		name          string
		model         string
		fallbackModel string
		wantArgs      []string
	}{
		{
			name:          "sem model sem fallback — sem --bare quando claudiney disponivel",
			model:         "",
			fallbackModel: "",
			wantArgs: []string{
				"--print", "-p", "execute task",
			},
		},
		{
			name:          "com model e com fallback — --model antes de --fallback-model",
			model:         "claude-sonnet-4-6",
			fallbackModel: "claude-haiku-4-5",
			wantArgs: []string{
				"--model", "claude-sonnet-4-6",
				"--fallback-model", "claude-haiku-4-5",
				"--print", "-p", "execute task",
			},
		},
		{
			name:          "sem model mas com fallback — apenas --fallback-model",
			model:         "",
			fallbackModel: "claude-haiku-4-5",
			wantArgs: []string{
				"--fallback-model", "claude-haiku-4-5",
				"--print", "-p", "execute task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFakeBinary(t, dir, "claude")
			writeFakeBinary(t, dir, "claudiney")

			origPath := os.Getenv("PATH")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)

			inv := &claudeInvoker{fallbackModel: tt.fallbackModel}
			stdout, _, exitCode, err := inv.Invoke(context.Background(), "execute task", dir, tt.model)
			if err != nil {
				t.Fatalf("Invoke retornou erro inesperado: %v", err)
			}
			if exitCode != 0 {
				t.Fatalf("exit code inesperado: %d", exitCode)
			}

			got := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
			if len(got) != len(tt.wantArgs) {
				t.Fatalf("numero de args: got %d, want %d\ngot:  %v\nwant: %v",
					len(got), len(tt.wantArgs), got, tt.wantArgs)
			}
			for i, want := range tt.wantArgs {
				if got[i] != want {
					t.Errorf("arg[%d]: got %q, want %q", i, got[i], want)
				}
			}
		})
	}
}
