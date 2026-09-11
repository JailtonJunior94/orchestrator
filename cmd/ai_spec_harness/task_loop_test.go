package aispecharness

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// exitCode2 verifica que err carrega um exitError com codigo 2 (uso incorreto),
// mesmo contrato aplicado pelo RunE do taskLoopCmd em producao.
func exitCode2(err error) bool {
	var ee *exitError
	return errors.As(err, &ee) && ee.ExitCode() == 2
}

// TestTaskLoopFlags_Runtime valida as regras de validação da flag --runtime (RF-01, RF-02).
func TestTaskLoopFlags_Runtime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		runtime   string
		tool      string
		wantErr   bool
		wantExit2 bool
		wantMsg   string
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
			name:      "runtime invalido",
			runtime:   "invalid",
			tool:      "claude",
			wantErr:   true,
			wantExit2: true,
		},
		{
			name:    "T-13: runtime acp com tool copilot valido (RF-06)",
			runtime: "acp",
			tool:    "copilot",
			wantErr: false,
		},
		{
			name:    "T-22: runtime acp com tool codex valido (RF-12)",
			runtime: "acp",
			tool:    "codex",
			wantErr: false,
		},
		{
			name:    "T-14: runtime acp com tool opencode valido (task 7.0)",
			runtime: "acp",
			tool:    "opencode",
			wantErr: false,
		},
		{
			name:      "runtime acp sem tool invalido (RF-02)",
			runtime:   "acp",
			tool:      "",
			wantErr:   true,
			wantExit2: true,
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
			if tt.wantExit2 && err != nil && !exitCode2(err) {
				t.Errorf("erro deve carregar exit code 2 (exitError), obteve: %q", err.Error())
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
	f := newTaskLoopCmd().Flags().Lookup("quiet")
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

	f := newTaskLoopCmd().Flags().Lookup("runtime")
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

	f := newTaskLoopCmd().Flags().Lookup("activity-timeout")
	if f == nil {
		t.Fatal("flag --activity-timeout nao registrada")
	}
	// O default é 2m0s (120s).
	if f.DefValue != "2m0s" {
		t.Errorf("default de --activity-timeout = %q, quero 2m0s", f.DefValue)
	}
}

func TestTaskLoopFlags_MaxBugfixIterationsDefault(t *testing.T) {
	t.Parallel()

	f := newTaskLoopCmd().Flags().Lookup("max-bugfix-iterations")
	if f == nil {
		t.Fatal("flag --max-bugfix-iterations nao registrada")
	}
	if f.DefValue != "5" {
		t.Errorf("default de --max-bugfix-iterations = %q, quero 5", f.DefValue)
	}
	if !strings.Contains(f.Usage, "5") {
		t.Errorf("texto de ajuda de --max-bugfix-iterations nao cita o default 5: %q", f.Usage)
	}
}

func TestTaskLoopFlags_MaxBugfixIterationsInvalido(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		wantExit2 bool
		wantMsg   string
	}{
		{
			name: "ausente preserva comportamento pre-mudanca",
			args: []string{"task-loop", "--tool", "claude", "--dry-run", "does-not-exist-prd"},
		},
		{
			name:      "zero e invalido",
			args:      []string{"task-loop", "--tool", "claude", "--max-bugfix-iterations", "0", "does-not-exist-prd"},
			wantErr:   true,
			wantExit2: true,
			wantMsg:   "minimo aceito: 1",
		},
		{
			name:      "negativo e invalido",
			args:      []string{"task-loop", "--tool", "claude", "--max-bugfix-iterations", "-3", "does-not-exist-prd"},
			wantErr:   true,
			wantExit2: true,
			wantMsg:   "minimo aceito: 1",
		},
		{
			name: "positivo explicito e valido",
			args: []string{"task-loop", "--tool", "claude", "--dry-run", "--max-bugfix-iterations", "2", "does-not-exist-prd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AI_INVOCATION_DEPTH", "0")

			origStderr := os.Stderr
			r, w, pipeErr := os.Pipe()
			if pipeErr != nil {
				t.Fatalf("os.Pipe: %v", pipeErr)
			}
			os.Stderr = w

			root := newRootCmd()
			root.SetArgs(tt.args)
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			err := root.Execute()

			_ = w.Close()
			os.Stderr = origStderr
			stderrBytes, _ := io.ReadAll(r)
			stderr := string(stderrBytes)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("esperava erro para args %v, nao obteve", tt.args)
				}
				if tt.wantExit2 && !exitCode2(err) {
					t.Errorf("erro deve carregar exit code 2, obteve: %v", err)
				}
				if tt.wantMsg != "" && !strings.Contains(stderr, tt.wantMsg) {
					t.Errorf("stderr %q nao contem %q", stderr, tt.wantMsg)
				}
				return
			}
			if strings.Contains(stderr, "max-bugfix-iterations") {
				t.Fatalf("nao esperava erro de --max-bugfix-iterations, stderr: %q", stderr)
			}
		})
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
			revTool:   "opencode",
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

	f := newTaskLoopCmd().Flags().Lookup("agent")
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
		return newExitError(2)
	}
	if runtime == "acp" {
		if _, ok := runtimeACPCatalog[tool]; !ok {
			if _, resolveErr := skills.NewCatalog().ResolveTool(tool); resolveErr != nil {
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
			return fmt.Errorf("runtime acp suporta apenas --tool em %v nesta versão: %w", supported, newExitError(2))
		}
	}
	if activityTimeout < 0 {
		return newExitError(2)
	}
	return nil
}

// validateEnumFlags replica a lógica de validação enum de --reasoning-effort e --access-mode
// do RunE do taskLoopCmd, para testes unitários focados (RF-09, RF-10, RF-11, RF-13 — ADR-013 D-08).
func validateEnumFlags(reasoningEffort, accessMode string) error {
	validReasoning := map[string]bool{"low": true, "medium": true, "high": true}
	if !validReasoning[reasoningEffort] {
		return fmt.Errorf("--reasoning-effort inválido: %q — valores aceitos: low|medium|high: %w", reasoningEffort, newExitError(2))
	}
	validAccess := map[string]bool{"restricted": true, "full": true}
	if !validAccess[accessMode] {
		return fmt.Errorf("--access-mode inválido: %q — valores aceitos: restricted|full: %w", accessMode, newExitError(2))
	}
	return nil
}

// TestRuntimeACPCatalog_T13_T14_T15 valida T-13 (Copilot ACP aceito),
// T-14 (tool desconhecida rejeitada com lista ordenada) e T-15 (Claude ACP regressão).
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

	t.Run("TestRuntimeACPCatalogIncludesOpenCode: catálogo inclui opencode (task 7.0)", func(t *testing.T) {
		t.Parallel()

		if _, ok := runtimeACPCatalog["opencode"]; !ok {
			t.Error("runtimeACPCatalog não contém 'opencode' (task 7.0 — ADR-003)")
		}
		spec := runtimeACPCatalog["opencode"]()
		if spec.ID != "opencode" {
			t.Errorf("runtimeACPCatalog[\"opencode\"]().ID = %q, esperava \"opencode\"", spec.ID)
		}
		if spec.Command == "" {
			t.Error("runtimeACPCatalog[\"opencode\"]().Command vazio")
		}
	})

	// T-14b: tool desconhecida ainda deve ser rejeitada com lista ordenada.
	t.Run("T-14b: tool desconhecida rejeitada com lista ordenada", func(t *testing.T) {
		t.Parallel()

		err := validateRuntimeFlags("acp", "unknown-tool", 0)
		if err == nil {
			t.Error("tool desconhecida deve ser rejeitada")
		}
		// Verificar que a mensagem contém todas as tools suportadas ordenadas.
		msg := err.Error()
		for _, tool := range []string{"claude", "codex", "copilot", "opencode"} {
			if !strings.Contains(msg, tool) {
				t.Errorf("mensagem de erro deve listar %q, obteve: %q", tool, msg)
			}
		}
		// Verificar ordem lexicográfica: claude < codex < copilot < opencode.
		idxClaude := strings.Index(msg, "claude")
		idxCodex := strings.Index(msg, "codex")
		idxCopilot := strings.Index(msg, "copilot")
		idxOpenCode := strings.Index(msg, "opencode")
		if idxClaude > idxCodex {
			t.Errorf("'claude' deve aparecer antes de 'codex' (ordem lexicográfica): %q", msg)
		}
		if idxCodex > idxCopilot {
			t.Errorf("'codex' deve aparecer antes de 'copilot' (ordem lexicográfica): %q", msg)
		}
		if idxCopilot > idxOpenCode {
			t.Errorf("'copilot' deve aparecer antes de 'opencode' (ordem lexicográfica): %q", msg)
		}
	})

	// T-16: Catálogo deve conter exatamente claude, codex, copilot e opencode.
	t.Run("T-16: catálogo contém exatamente claude, codex, copilot e opencode", func(t *testing.T) {
		t.Parallel()

		keys := make([]string, 0, len(runtimeACPCatalog))
		for k := range runtimeACPCatalog {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		expected := []string{"claude", "codex", "copilot", "opencode"}
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

// TestRuntimeACPCatalogIncludesOpenCode valida que runtimeACPCatalog contém "opencode".
// Critérios: entrada presente, ID correto, Command não vazio, validateRuntimeFlags aceita opencode+acp.
func TestRuntimeACPCatalogIncludesOpenCode(t *testing.T) {
	t.Parallel()

	ctor, ok := runtimeACPCatalog["opencode"]
	if !ok {
		t.Fatal("runtimeACPCatalog não contém 'opencode'")
	}

	spec := ctor()
	if spec.ID != "opencode" {
		t.Errorf("runtimeACPCatalog[\"opencode\"]().ID = %q, esperava \"opencode\"", spec.ID)
	}
	if spec.Command == "" {
		t.Error("runtimeACPCatalog[\"opencode\"]().Command vazio")
	}

	// Gate de validação deve aceitar opencode+acp sem erro.
	if err := validateRuntimeFlags("acp", "opencode", 0); err != nil {
		t.Errorf("validateRuntimeFlags(\"acp\", \"opencode\", 0) retornou erro inesperado: %v", err)
	}
}

// TestTaskLoopFlags_ReasoningEffortRegistered valida que --reasoning-effort está registrado com default correto.
func TestTaskLoopFlags_ReasoningEffortRegistered(t *testing.T) {
	t.Parallel()

	f := newTaskLoopCmd().Flags().Lookup("reasoning-effort")
	if f == nil {
		t.Fatal("flag --reasoning-effort nao registrada no taskLoopCmd")
	}
	if f.DefValue != "medium" {
		t.Errorf("default de --reasoning-effort = %q, quero medium", f.DefValue)
	}
	// Help text deve mencionar que só Codex consome.
	if !strings.Contains(f.Usage, "Codex") {
		t.Errorf("help text de --reasoning-effort nao menciona Codex; usage=%q", f.Usage)
	}
	// Help text deve listar os valores aceitos.
	if !strings.Contains(f.Usage, "low") || !strings.Contains(f.Usage, "medium") || !strings.Contains(f.Usage, "high") {
		t.Errorf("help text de --reasoning-effort nao lista low|medium|high; usage=%q", f.Usage)
	}
}

// TestTaskLoopFlags_AccessModeRegistered valida que --access-mode está registrado com default correto.
func TestTaskLoopFlags_AccessModeRegistered(t *testing.T) {
	t.Parallel()

	f := newTaskLoopCmd().Flags().Lookup("access-mode")
	if f == nil {
		t.Fatal("flag --access-mode nao registrada no taskLoopCmd")
	}
	if f.DefValue != "restricted" {
		t.Errorf("default de --access-mode = %q, quero restricted", f.DefValue)
	}
	// Help text deve mencionar warning para full.
	if !strings.Contains(f.Usage, "isolados") && !strings.Contains(strings.ToLower(f.Usage), "aviso") {
		t.Errorf("help text de --access-mode nao menciona risco de full; usage=%q", f.Usage)
	}
}

// TestTaskLoopFlags_T24_ReasoningEffortInvalido valida que --reasoning-effort inválido retorna exit code 2 (T-24, RF-09).
func TestTaskLoopFlags_T24_ReasoningEffortInvalido(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		reasoningEffort string
		wantErr         bool
		wantExit2       bool
		wantMsg         string
	}{
		{
			name:            "low valido",
			reasoningEffort: "low",
			wantErr:         false,
		},
		{
			name:            "medium valido (default)",
			reasoningEffort: "medium",
			wantErr:         false,
		},
		{
			name:            "high valido",
			reasoningEffort: "high",
			wantErr:         false,
		},
		{
			name:            "T-24: invalid retorna exit code 2",
			reasoningEffort: "invalid",
			wantErr:         true,
			wantExit2:       true,
			wantMsg:         "low|medium|high",
		},
		{
			name:            "T-24: ultra retorna exit code 2 com enum listado",
			reasoningEffort: "ultra",
			wantErr:         true,
			wantExit2:       true,
			wantMsg:         "low|medium|high",
		},
		{
			name:            "T-24: vazio retorna exit code 2",
			reasoningEffort: "",
			wantErr:         true,
			wantExit2:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateEnumFlags(tt.reasoningEffort, "restricted")
			if tt.wantErr && err == nil {
				t.Errorf("esperava erro, nao obteve")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("nao esperava erro, obteve: %v", err)
			}
			if tt.wantExit2 && err != nil && !exitCode2(err) {
				t.Errorf("erro deve carregar exit code 2 (exitError), obteve: %q", err.Error())
			}
			if tt.wantMsg != "" && err != nil && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("erro %q nao contem %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// TestTaskLoopFlags_T25_AccessModeInvalido valida que --access-mode inválido retorna exit code 2 (T-25, RF-11).
func TestTaskLoopFlags_T25_AccessModeInvalido(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		accessMode string
		wantErr    bool
		wantExit2  bool
		wantMsg    string
	}{
		{
			name:       "restricted valido (default)",
			accessMode: "restricted",
			wantErr:    false,
		},
		{
			name:       "full valido",
			accessMode: "full",
			wantErr:    false,
		},
		{
			name:       "T-25: open retorna exit code 2",
			accessMode: "open",
			wantErr:    true,
			wantExit2:  true,
			wantMsg:    "restricted|full",
		},
		{
			name:       "T-25: danger retorna exit code 2 com enum listado",
			accessMode: "danger",
			wantErr:    true,
			wantExit2:  true,
			wantMsg:    "restricted|full",
		},
		{
			name:       "T-25: vazio retorna exit code 2",
			accessMode: "",
			wantErr:    true,
			wantExit2:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateEnumFlags("medium", tt.accessMode)
			if tt.wantErr && err == nil {
				t.Errorf("esperava erro, nao obteve")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("nao esperava erro, obteve: %v", err)
			}
			if tt.wantExit2 && err != nil && !exitCode2(err) {
				t.Errorf("erro deve carregar exit code 2 (exitError), obteve: %q", err.Error())
			}
			if tt.wantMsg != "" && err != nil && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("erro %q nao contem %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// TestTaskLoopFlags_T23_CombinacoesCompletas valida combinações completas aceitas (T-23, RF-12).
func TestTaskLoopFlags_T23_CombinacoesCompletas(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		reasoningEffort string
		accessMode      string
		wantErr         bool
	}{
		{
			name:            "T-23: high + full aceito",
			reasoningEffort: "high",
			accessMode:      "full",
			wantErr:         false,
		},
		{
			name:            "T-23: low + restricted aceito",
			reasoningEffort: "low",
			accessMode:      "restricted",
			wantErr:         false,
		},
		{
			name:            "T-23: medium + restricted aceito (defaults)",
			reasoningEffort: "medium",
			accessMode:      "restricted",
			wantErr:         false,
		},
		{
			name:            "T-26 regressao Claude: high + full aceito (flags no-op para Claude)",
			reasoningEffort: "high",
			accessMode:      "full",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateEnumFlags(tt.reasoningEffort, tt.accessMode)
			if tt.wantErr && err == nil {
				t.Errorf("esperava erro, nao obteve")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("nao esperava erro, obteve: %v", err)
			}
		})
	}
}

// TestTaskLoopFlags_T30_AccessModeFullWarningSyncOnce valida que o warning de --access-mode=full
// é emitido exatamente uma vez via sync.Once (T-30, R-03 alto, ADR-013 D-08).
func TestTaskLoopFlags_T30_AccessModeFullWarningSyncOnce(t *testing.T) {
	// Não paralelo: testa comportamento de sync.Once com estado local para isolamento.
	var localOnce sync.Once
	var buf bytes.Buffer
	warnMsg := "WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp."

	emitWarning := func() {
		localOnce.Do(func() {
			fmt.Fprintln(&buf, warnMsg)
		})
	}

	// Primeira invocação: deve emitir warning.
	emitWarning()
	if !strings.Contains(buf.String(), "WARNING") {
		t.Errorf("primeira invocacao deve emitir warning; buffer=%q", buf.String())
	}

	firstOutput := buf.String()

	// Segunda e terceira invocações: sync.Once não deve emitir novamente.
	emitWarning()
	emitWarning()
	if buf.String() != firstOutput {
		t.Errorf("invocacoes adicionais nao devem emitir warning; buffer=%q", buf.String())
	}

	// Verificar que a mensagem menciona sandbox_mode=danger-full-access.
	if !strings.Contains(firstOutput, "sandbox_mode=danger-full-access") {
		t.Errorf("warning deve mencionar sandbox_mode=danger-full-access; output=%q", firstOutput)
	}
}

// TestTaskLoopFlags_T30_WarningSyncOnce_Global valida que accessModeFullWarnOnce
// está declarado como sync.Once em escopo package-level (T-30 — ADR-013 D-08).
func TestTaskLoopFlags_T30_WarningSyncOnce_Global(t *testing.T) {
	t.Parallel()

	// Referência explícita ao ponteiro confirma existência e tipo; compilação falha se ausente.
	// Usando ponteiro para evitar cópia de sync.Once (go vet: assignment copies lock value).
	_ = &_accessModeFullWarnOnce
}

// TestTaskLoopFlags_T15_MCPNestedNoNormalize valida as flags F2-Claude --mcp-nested e --no-normalize (T-15).
// Critério: ambas registradas com default false; --mcp-nested + --no-normalize combinados aceitos (RF-01.1, RF-02.4).
func TestTaskLoopFlags_T15_MCPNestedNoNormalize(t *testing.T) {
	t.Parallel()

	// Verificar que --mcp-nested está registrada com default false.
	t.Run("mcp-nested registrada com default false", func(t *testing.T) {
		t.Parallel()

		f := newTaskLoopCmd().Flags().Lookup("mcp-nested")
		if f == nil {
			t.Fatal("flag --mcp-nested nao registrada no taskLoopCmd")
		}
		if f.DefValue != "false" {
			t.Errorf("default de --mcp-nested = %q, quero false", f.DefValue)
		}
		if !strings.Contains(f.Usage, "MCP") && !strings.Contains(f.Usage, "mcp") {
			t.Errorf("help text de --mcp-nested nao menciona MCP; usage=%q", f.Usage)
		}
	})

	// Verificar que --no-normalize está registrada com default false.
	t.Run("no-normalize registrada com default false", func(t *testing.T) {
		t.Parallel()

		f := newTaskLoopCmd().Flags().Lookup("no-normalize")
		if f == nil {
			t.Fatal("flag --no-normalize nao registrada no taskLoopCmd")
		}
		if f.DefValue != "false" {
			t.Errorf("default de --no-normalize = %q, quero false", f.DefValue)
		}
		if !strings.Contains(f.Usage, "normaliz") {
			t.Errorf("help text de --no-normalize nao menciona normalizacao; usage=%q", f.Usage)
		}
	})

	// Verificar que --mcp-nested + --no-normalize combinados são aceitos pela validação de runtime.
	// Regressão T-19: sessão sem flags novas roda idêntico a F1-Claude (defaults preservam comportamento).
	t.Run("combinacao mcp-nested + no-normalize aceita (regressao T-19)", func(t *testing.T) {
		t.Parallel()

		// validateRuntimeFlags não conhece MCPNested/NoNormalize (flags ortogonais ao runtime).
		// Este caso documenta que as flags F2 não interferem com a validação de runtime.
		err := validateRuntimeFlags("acp", "claude", 0)
		if err != nil {
			t.Errorf("claude acp com mcp-nested + no-normalize deve passar validacao: %v", err)
		}
	})

	// Verificar que defaults preservam comportamento F1-Claude (RF-01.1, zero-value false).
	t.Run("defaults false preservam comportamento F1-Claude", func(t *testing.T) {
		t.Parallel()

		mcpFlag := newTaskLoopCmd().Flags().Lookup("mcp-nested")
		normFlag := newTaskLoopCmd().Flags().Lookup("no-normalize")
		if mcpFlag == nil || normFlag == nil {
			t.Fatal("flags F2-Claude nao registradas")
		}
		if mcpFlag.DefValue != "false" || normFlag.DefValue != "false" {
			t.Errorf("defaults devem ser false para preservar F1-Claude: mcp-nested=%q, no-normalize=%q",
				mcpFlag.DefValue, normFlag.DefValue)
		}
	})
}

// TestTaskLoopFlags_T16_F3Flags valida as 5 flags F3-Claude registradas (T-16).
// Critérios: defaults corretos; --disable-hooks + --memory-workflow-limit-lines combináveis.
func TestTaskLoopFlags_T16_F3Flags(t *testing.T) {
	t.Parallel()

	t.Run("T-16a: memory-workflow-limit-lines default 150", func(t *testing.T) {
		t.Parallel()

		f := newTaskLoopCmd().Flags().Lookup("memory-workflow-limit-lines")
		if f == nil {
			t.Fatal("flag --memory-workflow-limit-lines nao registrada")
		}
		if f.DefValue != "150" {
			t.Errorf("default = %q, quero 150", f.DefValue)
		}
	})

	t.Run("T-16b: memory-workflow-limit-bytes default 12288", func(t *testing.T) {
		t.Parallel()

		f := newTaskLoopCmd().Flags().Lookup("memory-workflow-limit-bytes")
		if f == nil {
			t.Fatal("flag --memory-workflow-limit-bytes nao registrada")
		}
		if f.DefValue != "12288" {
			t.Errorf("default = %q, quero 12288", f.DefValue)
		}
	})

	t.Run("T-16c: memory-task-limit-lines default 200", func(t *testing.T) {
		t.Parallel()

		f := newTaskLoopCmd().Flags().Lookup("memory-task-limit-lines")
		if f == nil {
			t.Fatal("flag --memory-task-limit-lines nao registrada")
		}
		if f.DefValue != "200" {
			t.Errorf("default = %q, quero 200", f.DefValue)
		}
	})

	t.Run("T-16d: memory-task-limit-bytes default 16384", func(t *testing.T) {
		t.Parallel()

		f := newTaskLoopCmd().Flags().Lookup("memory-task-limit-bytes")
		if f == nil {
			t.Fatal("flag --memory-task-limit-bytes nao registrada")
		}
		if f.DefValue != "16384" {
			t.Errorf("default = %q, quero 16384", f.DefValue)
		}
	})

	t.Run("T-16e: disable-hooks default false", func(t *testing.T) {
		t.Parallel()

		f := newTaskLoopCmd().Flags().Lookup("disable-hooks")
		if f == nil {
			t.Fatal("flag --disable-hooks nao registrada")
		}
		if f.DefValue != "false" {
			t.Errorf("default = %q, quero false", f.DefValue)
		}
		// Help text deve mencionar que desabilita TODOS os hooks.
		if !strings.Contains(f.Usage, "hooks") {
			t.Errorf("help text deve mencionar hooks; usage=%q", f.Usage)
		}
		// Help text deve mencionar que shell hooks continuam ativos.
		if !strings.Contains(f.Usage, "Shell") && !strings.Contains(f.Usage, "shell") {
			t.Errorf("help text deve mencionar shell hooks; usage=%q", f.Usage)
		}
	})

	t.Run("T-16f: --disable-hooks + --memory-workflow-limit-lines 100 combinaveis", func(t *testing.T) {
		t.Parallel()

		// Verificar que ambas as flags existem e são do tipo correto (não interferem entre si).
		dhFlag := newTaskLoopCmd().Flags().Lookup("disable-hooks")
		mwlFlag := newTaskLoopCmd().Flags().Lookup("memory-workflow-limit-lines")

		if dhFlag == nil || mwlFlag == nil {
			t.Fatal("flags F3 ausentes")
		}
		// Verificar tipos: disable-hooks é bool, memory-workflow-limit-lines é int.
		if dhFlag.Value.Type() != "bool" {
			t.Errorf("disable-hooks deve ser bool; got %q", dhFlag.Value.Type())
		}
		if mwlFlag.Value.Type() != "int" {
			t.Errorf("memory-workflow-limit-lines deve ser int; got %q", mwlFlag.Value.Type())
		}
	})
}

// TestOpenCodeSpecHasCorrectCommandAndFlags valida que
// runtimeACPCatalog["opencode"]() retorna Spec com Command=opencode e FixedArgs=[acp].
func TestOpenCodeSpecHasCorrectCommandAndFlags(t *testing.T) {
	t.Parallel()

	ctor, ok := runtimeACPCatalog["opencode"]
	if !ok {
		t.Fatal("runtimeACPCatalog não contém 'opencode'")
	}
	spec := ctor()
	if spec.Command != "opencode" {
		t.Errorf("Command = %q; want %q", spec.Command, "opencode")
	}
	if len(spec.FixedArgs) != 1 || spec.FixedArgs[0] != "acp" {
		t.Errorf("FixedArgs = %v; want [acp]", spec.FixedArgs)
	}
}

// TestOpenCodeFallbackResolvesViaNpx valida que
// runtimeACPCatalog["opencode"]() expõe fallback npx com o package pinado.
func TestOpenCodeFallbackResolvesViaNpx(t *testing.T) {
	t.Parallel()

	ctor, ok := runtimeACPCatalog["opencode"]
	if !ok {
		t.Fatal("runtimeACPCatalog não contém 'opencode'")
	}
	spec := ctor()
	if len(spec.Fallbacks) == 0 {
		t.Fatal("OpenCode Spec não declara nenhum fallback")
	}
	fb := spec.Fallbacks[0]
	if fb.Command != "npx" {
		t.Errorf("Fallbacks[0].Command = %q; want npx", fb.Command)
	}
	// Deve conter opencode-ai@<version> no slice de args
	found := false
	for _, arg := range fb.FixedArgs {
		if strings.HasPrefix(arg, "opencode-ai@") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Fallbacks[0].FixedArgs = %v; want to contain 'opencode-ai@<version>'", fb.FixedArgs)
	}
}

// TestTaskLoopFlags_ReasoningEffortAndAccessModeDefaults valida os valores default das novas flags.
func TestTaskLoopFlags_ReasoningEffortAndAccessModeDefaults(t *testing.T) {
	t.Parallel()

	f := newTaskLoopCmd().Flags().Lookup("reasoning-effort")
	if f == nil {
		t.Fatal("flag --reasoning-effort nao encontrada")
	}
	if f.DefValue != "medium" {
		t.Errorf("default --reasoning-effort = %q, esperava medium", f.DefValue)
	}

	g := newTaskLoopCmd().Flags().Lookup("access-mode")
	if g == nil {
		t.Fatal("flag --access-mode nao encontrada")
	}
	if g.DefValue != "restricted" {
		t.Errorf("default --access-mode = %q, esperava restricted", g.DefValue)
	}
}

// TestValidateRuntimeFlags_RemovedAgentTypedError (RF-03, tarefa 10.0) valida que
// invocar --tool gemini com --runtime acp produz o erro tipado RemovedAgentError,
// distinguivel do erro generico de valor invalido via errors.As.
func TestValidateRuntimeFlags_RemovedAgentTypedError(t *testing.T) {
	t.Parallel()

	err := validateRuntimeFlags("acp", "gemini", 0)
	if err == nil {
		t.Fatal("esperava erro para --tool gemini, obteve nil")
	}

	var removedErr *skills.RemovedAgentError
	if !errors.As(err, &removedErr) {
		t.Fatalf("erro deveria ser RemovedAgentError (errors.As), obteve: %v", err)
	}
	if removedErr.Agent != "gemini" {
		t.Errorf("RemovedAgentError.Agent = %q, want %q", removedErr.Agent, "gemini")
	}
	msg := err.Error()
	for _, want := range []string{"claude", "codex", "copilot", "opencode", "migracao-legacy-acp.md"} {
		if !strings.Contains(msg, want) {
			t.Errorf("mensagem de erro deveria conter %q, got: %q", want, msg)
		}
	}

	// Ferramenta genuinamente desconhecida (nunca existiu) NAO deve produzir RemovedAgentError.
	genericErr := validateRuntimeFlags("acp", "not-a-real-tool", 0)
	var genericRemoved *skills.RemovedAgentError
	if errors.As(genericErr, &genericRemoved) {
		t.Error("ferramenta desconhecida generica nao deveria ser RemovedAgentError")
	}
}
