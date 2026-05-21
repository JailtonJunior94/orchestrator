package aispecharness

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestTaskLoopFlags_Runtime valida as regras de validação da flag --runtime (RF-01, RF-02).
func TestTaskLoopFlags_Runtime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		runtime string
		tool    string
		wantErr bool
		wantMsg string
	}{
		{
			name:    "runtime legacy valido",
			runtime: "legacy",
			tool:    "claude",
			wantErr: false,
		},
		{
			name:    "runtime acp com tool claude valido",
			runtime: "acp",
			tool:    "claude",
			wantErr: false,
		},
		{
			name:    "runtime invalido",
			runtime: "invalid",
			tool:    "claude",
			wantErr: true,
			wantMsg: "exit2",
		},
		{
			name:    "T-13: runtime acp com tool copilot valido (RF-06)",
			runtime: "acp",
			tool:    "copilot",
			wantErr: false,
		},
		{
			name:    "runtime acp com tool codex invalido (RF-02)",
			runtime: "acp",
			tool:    "codex",
			wantErr: true,
			wantMsg: "exit2",
		},
		{
			name:    "T-14: runtime acp com tool gemini invalido — lista ordenada (RF-07)",
			runtime: "acp",
			tool:    "gemini",
			wantErr: true,
			wantMsg: "exit2",
		},
		{
			name:    "runtime acp sem tool invalido (RF-02)",
			runtime: "acp",
			tool:    "",
			wantErr: true,
			// sem tool: cai em "informe --tool" antes ou em validação de runtime+tool
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateRuntimeFlags(tt.runtime, tt.tool, 0)
			if tt.wantErr && err == nil {
				t.Errorf("esperava erro, nao obteve")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("nao esperava erro, obteve: %v", err)
			}
			if tt.wantMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.wantMsg) {
					t.Errorf("erro %q nao contem %q", err.Error(), tt.wantMsg)
				}
			}
		})
	}
}

// TestTaskLoopFlags_ActivityTimeout valida a flag --activity-timeout (RF-07).
func TestTaskLoopFlags_ActivityTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "120s valido",
			timeout: 120 * time.Second,
			wantErr: false,
		},
		{
			name:    "0 desabilita watchdog (valido)",
			timeout: 0,
			wantErr: false,
		},
		{
			name:    "2m valido",
			timeout: 2 * time.Minute,
			wantErr: false,
		},
		{
			name:    "negativo invalido",
			timeout: -1 * time.Second,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateRuntimeFlags("legacy", "claude", tt.timeout)
			if tt.wantErr && err == nil {
				t.Errorf("esperava erro para timeout %v, nao obteve", tt.timeout)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("nao esperava erro para timeout %v, obteve: %v", tt.timeout, err)
			}
		})
	}
}

// TestTaskLoopFlags_Quiet valida que a flag --quiet é definida e parseável.
// A propagação até o acpInvoker é testada em internal/taskloop/acpinvoker_test.go.
func TestTaskLoopFlags_Quiet(t *testing.T) {
	t.Parallel()

	// Verificar que a flag existe no comando.
	f := taskLoopCmd.Flags().Lookup("quiet")
	if f == nil {
		t.Fatal("flag --quiet nao registrada no taskLoopCmd")
	}
	if f.DefValue != "false" {
		t.Errorf("default de --quiet = %q, quero false", f.DefValue)
	}
}

// TestTaskLoopFlags_RuntimeFlagDefault valida o default da flag --runtime.
func TestTaskLoopFlags_RuntimeFlagDefault(t *testing.T) {
	t.Parallel()

	f := taskLoopCmd.Flags().Lookup("runtime")
	if f == nil {
		t.Fatal("flag --runtime nao registrada")
	}
	if f.DefValue != "legacy" {
		t.Errorf("default de --runtime = %q, quero legacy", f.DefValue)
	}
}

// TestTaskLoopFlags_ActivityTimeoutDefault valida o default da flag --activity-timeout.
func TestTaskLoopFlags_ActivityTimeoutDefault(t *testing.T) {
	t.Parallel()

	f := taskLoopCmd.Flags().Lookup("activity-timeout")
	if f == nil {
		t.Fatal("flag --activity-timeout nao registrada")
	}
	// O default é 2m0s (120s).
	if f.DefValue != "2m0s" {
		t.Errorf("default de --activity-timeout = %q, quero 2m0s", f.DefValue)
	}
}

// TestTaskLoopFlags_AgentExclusivity valida exclusividade de --agent com --tool e modo avancado (T-20, T-21, D-06).
func TestTaskLoopFlags_AgentExclusivity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		agentName string
		tool      string
		execTool  string
		revTool   string
		wantErr   bool
		errMsg    string
	}{
		// T-20: --agent + --tool deve gerar erro (ErrFlagsConflitantes).
		{
			name:      "T-20: agent + tool gera conflito",
			agentName: "foo",
			tool:      "claude",
			wantErr:   true,
			errMsg:    "mutuamente exclusivas",
		},
		// T-21: --agent + --executor-tool deve gerar erro de conflito.
		{
			name:      "T-21: agent + executor-tool gera conflito",
			agentName: "foo",
			execTool:  "codex",
			wantErr:   true,
			errMsg:    "mutuamente exclusivas",
		},
		// --agent + --reviewer-tool deve gerar erro de conflito.
		{
			name:      "agent + reviewer-tool gera conflito",
			agentName: "foo",
			revTool:   "gemini",
			wantErr:   true,
			errMsg:    "mutuamente exclusivas",
		},
		// --agent sozinho: sem conflito de exclusividade (pode falhar por outros motivos, mas nao aqui).
		{
			name:      "agent sozinho: sem conflito de exclusividade",
			agentName: "foo",
			wantErr:   false,
		},
		// --agent + --model é permitido (RF-13): flags de override nao conflitam.
		{
			name:      "agent + sem tool/execTool/revTool: aceito",
			agentName: "myagent",
			tool:      "",
			execTool:  "",
			revTool:   "",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAgentFlags(tt.agentName, tt.tool, tt.execTool, tt.revTool)
			if tt.wantErr && err == nil {
				t.Errorf("esperava erro, nao obteve")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("nao esperava erro, obteve: %v", err)
			}
			if tt.errMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("erro %q nao contem %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestTaskLoopFlags_AgentFlagRegistered valida que --agent esta registrado no taskLoopCmd.
func TestTaskLoopFlags_AgentFlagRegistered(t *testing.T) {
	t.Parallel()

	f := taskLoopCmd.Flags().Lookup("agent")
	if f == nil {
		t.Fatal("flag --agent nao registrada no taskLoopCmd")
	}
	if f.DefValue != "" {
		t.Errorf("default de --agent = %q, quero string vazia", f.DefValue)
	}
	// Verificar que o help text menciona a relacao com --tool.
	if !strings.Contains(f.Usage, "--tool") {
		t.Errorf("help text de --agent nao menciona --tool; usage=%q", f.Usage)
	}
}

// validateAgentFlags é uma função auxiliar de teste que valida as mesmas
// regras de exclusividade aplicadas no RunE do taskLoopCmd para a flag --agent.
func validateAgentFlags(agentName, tool, execTool, revTool string) error {
	if agentName != "" && (tool != "" || execTool != "" || revTool != "") {
		return fmt.Errorf("--agent e mutuamente exclusivo com --tool, --executor-tool e --reviewer-tool: flags de modo simples e avancado sao mutuamente exclusivas")
	}
	return nil
}

// validateRuntimeFlags é uma função auxiliar de teste que valida as mesmas
// regras aplicadas no RunE do taskLoopCmd, sem invocar o comando completo.
// Replica a lógica de validação para permitir testes unitários focados.
// Usa runtimeACPCatalog como fonte de verdade — mesma tabela do RunE (D-04).
func validateRuntimeFlags(runtime, tool string, activityTimeout time.Duration) error {
	if runtime != "legacy" && runtime != "acp" {
		return fmt.Errorf("exit2")
	}
	if runtime == "acp" {
		if _, ok := runtimeACPCatalog[tool]; !ok {
			supported := make([]string, 0, len(runtimeACPCatalog))
			for k := range runtimeACPCatalog {
				supported = append(supported, k)
			}
			sort.Strings(supported)
			return fmt.Errorf("exit2: runtime acp suporta apenas --tool em %v nesta versão", supported)
		}
	}
	if activityTimeout < 0 {
		return fmt.Errorf("exit2")
	}
	return nil
}

// TestRuntimeACPCatalog_T13_T14_T15 valida T-13 (Copilot ACP aceito),
// T-14 (Gemini ACP rejeitado com lista ordenada) e T-15 (Claude ACP regressão).
func TestRuntimeACPCatalog_T13_T14_T15(t *testing.T) {
	t.Parallel()

	// T-15: Claude ACP — regressão (comportamento atual preservado).
	t.Run("T-15: claude acp aceito (regressão)", func(t *testing.T) {
		t.Parallel()

		if _, ok := runtimeACPCatalog["claude"]; !ok {
			t.Error("runtimeACPCatalog não contém 'claude' — regressão")
		}
		err := validateRuntimeFlags("acp", "claude", 0)
		if err != nil {
			t.Errorf("claude acp deve passar validação, obteve: %v", err)
		}
	})

	// T-13: Copilot ACP — aceito (nova entrada no catálogo).
	t.Run("T-13: copilot acp aceito", func(t *testing.T) {
		t.Parallel()

		if _, ok := runtimeACPCatalog["copilot"]; !ok {
			t.Error("runtimeACPCatalog não contém 'copilot'")
		}
		err := validateRuntimeFlags("acp", "copilot", 0)
		if err != nil {
			t.Errorf("copilot acp deve passar validação, obteve: %v", err)
		}
	})

	// T-14: Gemini ACP — rejeitado com lista ordenada [claude copilot].
	t.Run("T-14: gemini acp rejeitado com lista ordenada", func(t *testing.T) {
		t.Parallel()

		err := validateRuntimeFlags("acp", "gemini", 0)
		if err == nil {
			t.Error("gemini acp deve ser rejeitado")
		}
		// Verificar que a mensagem contém ambas as tools suportadas ordenadas.
		msg := err.Error()
		if !strings.Contains(msg, "claude") || !strings.Contains(msg, "copilot") {
			t.Errorf("mensagem de erro deve listar [claude copilot], obteve: %q", msg)
		}
		// Verificar que 'claude' aparece antes de 'copilot' (ordem lexicográfica).
		idxClaude := strings.Index(msg, "claude")
		idxCopilot := strings.Index(msg, "copilot")
		if idxClaude > idxCopilot {
			t.Errorf("'claude' deve aparecer antes de 'copilot' na mensagem (ordem lexicográfica): %q", msg)
		}
	})

	// Catálogo deve conter exatamente claude e copilot (nesta versão).
	t.Run("catálogo contém exatamente claude e copilot", func(t *testing.T) {
		t.Parallel()

		keys := make([]string, 0, len(runtimeACPCatalog))
		for k := range runtimeACPCatalog {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		expected := []string{"claude", "copilot"}
		if len(keys) != len(expected) {
			t.Errorf("catálogo tem %d entradas, esperava %d: %v", len(keys), len(expected), keys)
			return
		}
		for i, k := range keys {
			if k != expected[i] {
				t.Errorf("catálogo[%d] = %q, esperava %q", i, k, expected[i])
			}
		}
	})

	// Construtores devem ser não-nil e retornar Specs corretas.
	t.Run("construtores do catálogo retornam Specs válidas", func(t *testing.T) {
		t.Parallel()

		for tool, ctor := range runtimeACPCatalog {
			spec := ctor()
			if spec.ID != tool {
				t.Errorf("runtimeACPCatalog[%q]().ID = %q, esperava %q", tool, spec.ID, tool)
			}
			if spec.Command == "" {
				t.Errorf("runtimeACPCatalog[%q]().Command vazio", tool)
			}
		}
	})
}
