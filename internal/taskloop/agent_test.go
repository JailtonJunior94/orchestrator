package taskloop

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
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

// TestBuildPromptContainsRequiredElements verifica que o prompt gerado contem os elementos
// obrigatorios: path da task, prd folder, skill execute-task e criterios de execucao.
func TestBuildPromptContainsRequiredElements(t *testing.T) {
	const workDir = "/fake/project"
	const prdFolder = "tasks/prd-feat"

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[workDir+"/go.mod"] = []byte("module example.com/app\n")
	fsys.Files[workDir+"/"+prdFolder+"/prd.md"] = []byte("# PRD\nImplementar feature de dominio.\n")
	fsys.Files[workDir+"/"+prdFolder+"/techspec.md"] = []byte("# TechSpec\nArquitetura em camadas.\n")

	prompt := BuildPrompt("tasks/prd-feat/01_task.md", prdFolder, workDir, fsys)

	required := []string{
		"execute-task",
		"tasks/prd-feat/01_task.md",
		prdFolder,
		"Leia o arquivo de task antes de iniciar qualquer alteracao",
		"go-implementation",
		"tests",
		"preservar contratos publicos existentes",
		"context.Context em todas as operacoes de IO",
		"testes table-driven",
		"registrar evidencia de conclusao",
		"nao fechar a task sem evidencia de validacao",
	}
	for _, r := range required {
		if !strings.Contains(prompt, r) {
			t.Errorf("BuildPrompt() nao contem %q\nprompt:\n%s", r, prompt)
		}
	}
}

// TestResolveReferences verifica a deteccao dinamica de referencias por stack e conteudo.
func TestResolveReferences(t *testing.T) {
	tests := []struct {
		name      string
		setupFsys func(*taskfs.FakeFileSystem, string, string)
		wantRefs  []string
		noRefs    []string
	}{
		{
			name: "projeto Go sem keywords adicionais inclui go-implementation e tests",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/go.mod"] = []byte("module example.com\n")
				fsys.Files[prd+"/prd.md"] = []byte("# PRD simples\n")
				fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec simples\n")
			},
			wantRefs: []string{"go-implementation", "tests"},
			noRefs:   []string{"ddd", "security", "persistence", "concurrency", "api", "observability"},
		},
		{
			name: "projeto Node inclui node-implementation",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/package.json"] = []byte(`{"name":"app"}`+"\n")
				fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
				fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			},
			wantRefs: []string{"node-implementation", "tests"},
			noRefs:   []string{"go-implementation"},
		},
		{
			name: "techspec menciona DDD adiciona ddd",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/go.mod"] = []byte("module example.com\n")
				fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
				fsys.Files[prd+"/techspec.md"] = []byte("Usar aggregate root e value object para modelar o dominio.\n")
			},
			wantRefs: []string{"go-implementation", "tests", "ddd"},
		},
		{
			name: "prd menciona observabilidade OpenTelemetry adiciona observability",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/go.mod"] = []byte("module example.com\n")
				fsys.Files[prd+"/prd.md"] = []byte("Instrumentar com OpenTelemetry e exportar traces.\n")
				fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			},
			wantRefs: []string{"go-implementation", "tests", "observability"},
		},
		{
			name: "prd menciona HTTP REST adiciona api",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/go.mod"] = []byte("module example.com\n")
				fsys.Files[prd+"/prd.md"] = []byte("Expor endpoint REST via HTTP handler.\n")
				fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			},
			wantRefs: []string{"go-implementation", "tests", "api"},
		},
		{
			name: "techspec menciona goroutine adiciona concurrency",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/go.mod"] = []byte("module example.com\n")
				fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
				fsys.Files[prd+"/techspec.md"] = []byte("Usar goroutine e channel para processar eventos concorrentemente.\n")
			},
			wantRefs: []string{"go-implementation", "tests", "concurrency"},
		},
		{
			name: "prd menciona database SQL adiciona persistence",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/go.mod"] = []byte("module example.com\n")
				fsys.Files[prd+"/prd.md"] = []byte("Persistir dados em database postgres via SQL.\n")
				fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			},
			wantRefs: []string{"go-implementation", "tests", "persistence"},
		},
		{
			name: "prd menciona autenticacao JWT adiciona security",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[workDir+"/go.mod"] = []byte("module example.com\n")
				fsys.Files[prd+"/prd.md"] = []byte("Autenticacao via JWT e autorizacao RBAC.\n")
				fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			},
			wantRefs: []string{"go-implementation", "tests", "security"},
		},
		{
			name: "sem go.mod nem package.json nao adiciona skill de implementacao",
			setupFsys: func(fsys *taskfs.FakeFileSystem, workDir, prd string) {
				fsys.Files[prd+"/prd.md"] = []byte("# PRD\n")
				fsys.Files[prd+"/techspec.md"] = []byte("# TechSpec\n")
			},
			wantRefs: []string{"tests"},
			noRefs:   []string{"go-implementation", "node-implementation", "python-implementation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const workDir = "/fake/project"
			const prd = workDir + "/tasks/prd-test"

			fsys := taskfs.NewFakeFileSystem()
			tt.setupFsys(fsys, workDir, prd)

			refs := resolveReferences(prd, workDir, fsys)
			refSet := make(map[string]bool, len(refs))
			for _, r := range refs {
				refSet[r] = true
			}

			for _, want := range tt.wantRefs {
				if !refSet[want] {
					t.Errorf("esperado %q nas referencias, obtido: %v", want, refs)
				}
			}
			for _, no := range tt.noRefs {
				if refSet[no] {
					t.Errorf("nao esperado %q nas referencias, obtido: %v", no, refs)
				}
			}
		})
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

func TestProcessStartHookSetterInterface(t *testing.T) {
	tools := []string{"claude", "codex", "gemini", "copilot"}
	for _, tool := range tools {
		invoker, err := NewAgentInvoker(tool)
		if err != nil {
			t.Fatalf("NewAgentInvoker(%q): %v", tool, err)
		}
		if _, ok := invoker.(processStartHookSetter); !ok {
			t.Errorf("invoker %q nao implementa processStartHookSetter", tool)
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

func TestAgentLifecycleAcrossTools(t *testing.T) {
	tests := []struct {
		name string
		tool string
	}{
		{name: "claude", tool: "claude"},
		{name: "codex", tool: "codex"},
		{name: "gemini", tool: "gemini"},
		{name: "copilot", tool: "copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			script := "#!/bin/sh\nprintf 'streaming-output\\n'\n"
			writeExecutable(t, dir, tt.tool, script)
			if tt.tool == "claude" {
				writeExecutable(t, dir, "claudiney", script)
			}

			origPath := os.Getenv("PATH")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)

			invoker, err := NewAgentInvoker(tt.tool)
			if err != nil {
				t.Fatalf("NewAgentInvoker(%q): %v", tt.tool, err)
			}

			var liveOut strings.Builder
			setter, ok := invoker.(LiveOutputSetter)
			if !ok {
				t.Fatalf("invoker %q nao implementa LiveOutputSetter", tt.tool)
			}
			setter.SetLiveOutput(&liveOut)

			startCalls := 0
			hookSetter, ok := invoker.(processStartHookSetter)
			if !ok {
				t.Fatalf("invoker %q nao implementa processStartHookSetter", tt.tool)
			}
			hookSetter.SetProcessStartHook(func() { startCalls++ })

			stdout, _, exitCode, err := invoker.Invoke(context.Background(), "executar task", dir, "")
			if err != nil {
				t.Fatalf("Invoke() erro inesperado: %v", err)
			}
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0", exitCode)
			}
			if startCalls != 1 {
				t.Fatalf("hook de start chamado %d vezes, want 1", startCalls)
			}
			if !strings.Contains(stdout, "streaming-output") {
				t.Fatalf("stdout = %q, want conter streaming-output", stdout)
			}
			if !strings.Contains(liveOut.String(), "streaming-output") {
				t.Fatalf("liveOut = %q, want conter streaming-output", liveOut.String())
			}
		})
	}
}

func TestLoopFailureMapping(t *testing.T) {
	tests := []struct {
		name       string
		tool       ToolName
		exitCode   int
		stdout     string
		stderr     string
		invokeErr  error
		postStatus string
		wantCode   ErrorCode
		wantMsg    string
	}{
		{
			name:       "timeout por deadline",
			tool:       ToolCodex,
			invokeErr:  context.DeadlineExceeded,
			postStatus: "pending",
			wantCode:   ErrorToolTimeout,
			wantMsg:    "tempo limite",
		},
		{
			name:       "autenticacao ausente",
			tool:       ToolClaude,
			stderr:     "Not logged in. Please run /login",
			postStatus: "pending",
			wantCode:   ErrorToolAuthRequired,
			wantMsg:    "requer autenticacao",
		},
		{
			name:       "binario ausente",
			tool:       ToolGemini,
			invokeErr:  exec.ErrNotFound,
			postStatus: "pending",
			wantCode:   ErrorToolBinaryMissing,
			wantMsg:    "nao encontrado no PATH",
		},
		{
			name:       "start falha com exit menos um continua binario ausente",
			tool:       ToolCodex,
			exitCode:   -1,
			invokeErr:  exec.ErrNotFound,
			postStatus: "pending",
			wantCode:   ErrorToolBinaryMissing,
			wantMsg:    "nao encontrado no PATH",
		},
		{
			name:       "falha de execucao",
			tool:       ToolCopilot,
			exitCode:   2,
			postStatus: "pending",
			wantCode:   ErrorToolExecutionFailed,
			wantMsg:    "encerrou com falha",
		},
		{
			name:       "status nao concluido com exit zero",
			tool:       ToolCodex,
			postStatus: "in_progress",
			wantCode:   ErrorToolExecutionFailed,
			wantMsg:    "sem concluir a task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := mapLoopFailure(tt.tool, tt.exitCode, tt.stdout, tt.stderr, tt.invokeErr, tt.postStatus)
			if failure == nil {
				t.Fatal("esperado LoopFailure, obteve nil")
			}
			if failure.Code != tt.wantCode {
				t.Fatalf("Code = %q, want %q", failure.Code, tt.wantCode)
			}
			if !strings.Contains(failure.Message, tt.wantMsg) {
				t.Fatalf("Message = %q, want conter %q", failure.Message, tt.wantMsg)
			}
		})
	}

	if failure := mapLoopFailure(ToolClaude, 0, "ok", "", nil, "done"); failure != nil {
		t.Fatalf("mapeamento de sucesso deveria retornar nil, obteve %+v", failure)
	}
}

func TestLoopFailureMappingPreservesCause(t *testing.T) {
	cause := errors.New("processo terminou com erro")
	failure := mapLoopFailure(ToolCodex, 0, "", "", cause, "pending")
	if failure == nil {
		t.Fatal("esperado LoopFailure, obteve nil")
	}
	if !errors.Is(failure, cause) {
		t.Fatalf("errors.Is deveria reconhecer a causa original")
	}
}

func writeExecutable(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("nao foi possivel criar executavel %q: %v", path, err)
	}
}
