package taskloop

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNewAgentInvoker valida que a factory retorna o invoker correto para cada ferramenta
// e erro tipado para ferramenta invalida.
func TestNewAgentInvoker(t *testing.T) {
	tests := []struct {
		name       string
		tool       string
		wantBinary string
		wantErr    bool
	}{
		{name: "claude", tool: "claude", wantBinary: "claude", wantErr: false},
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
func TestInvokerArgs(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		prompt   string
		wantArgs []string // argumentos esperados apos o binario, em ordem
	}{
		{
			name:   "claudeInvoker flags corretas (com --bare)",
			tool:   "claude",
			prompt: "execute task",
			wantArgs: []string{
				"--dangerously-skip-permissions", "--print", "--bare", "-p", "execute task",
			},
		},
		{
			name:   "codexInvoker flags corretas (exec + dangerously-bypass)",
			tool:   "codex",
			prompt: "execute task",
			wantArgs: []string{
				"exec", "--dangerously-bypass-approvals-and-sandbox", "-p", "execute task",
			},
		},
		{
			name:   "geminiInvoker flags inalteradas",
			tool:   "gemini",
			prompt: "execute task",
			wantArgs: []string{
				"--yolo", "-p", "execute task",
			},
		},
		{
			name:   "copilotInvoker flags corretas (com --autopilot, ordem correta)",
			tool:   "copilot",
			prompt: "execute task",
			wantArgs: []string{
				"--autopilot", "--yolo", "-p", "execute task",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cria diretorio temporario com fake binary para a ferramenta.
			dir := t.TempDir()
			writeFakeBinary(t, dir, tt.tool)

			// Coloca o fake binary na frente do PATH original.
			origPath := os.Getenv("PATH")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)

			invoker, err := NewAgentInvoker(tt.tool)
			if err != nil {
				t.Fatalf("NewAgentInvoker(%q): %v", tt.tool, err)
			}

			stdout, _, exitCode, err := invoker.Invoke(context.Background(), tt.prompt, dir)
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
