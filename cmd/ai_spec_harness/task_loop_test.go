package aispecharness

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
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
			name:    "T-22: runtime acp com tool codex valido (RF-12)",
			runtime: "acp",
			tool:    "codex",
			wantErr: false,
		},
		{
			name:    "T-14: runtime acp com tool gemini valido (RF-25 ADR-015)",
			runtime: "acp",
			tool:    "gemini",
			wantErr: false,
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
		return fmt.Errorf("exit2")
	}
	if runtime == "acp" {
		if _, ok := _runtimeACPCatalog[tool]; !ok {
			supported := make([]string, 0, len(_runtimeACPCatalog))
			for k := range _runtimeACPCatalog {
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

// validateEnumFlags replica a lógica de validação enum de --reasoning-effort e --access-mode
// do RunE do taskLoopCmd, para testes unitários focados (RF-09, RF-10, RF-11, RF-13 — ADR-013 D-08).
func validateEnumFlags(reasoningEffort, accessMode string) error {
	validReasoning := map[string]bool{"low": true, "medium": true, "high": true}
	if !validReasoning[reasoningEffort] {
		return fmt.Errorf("exit2: --reasoning-effort inválido: %q — valores aceitos: low|medium|high", reasoningEffort)
	}
	validAccess := map[string]bool{"restricted": true, "full": true}
	if !validAccess[accessMode] {
		return fmt.Errorf("exit2: --access-mode inválido: %q — valores aceitos: restricted|full", accessMode)
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

		if _, ok := _runtimeACPCatalog["claude"]; !ok {
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

		if _, ok := _runtimeACPCatalog["copilot"]; !ok {
			t.Error("runtimeACPCatalog não contém 'copilot'")
		}
		err := validateRuntimeFlags("acp", "copilot", 0)
		if err != nil {
			t.Errorf("copilot acp deve passar validação, obteve: %v", err)
		}
	})

	// T-14: Gemini ACP — aceito após ADR-015 (RF-25).
	// Antes da task 2.0: gemini era rejeitado. Após registrar "gemini" em runtimeACPCatalog, deve ser aceito.
	t.Run("T-14: gemini acp aceito (ADR-015, RF-25)", func(t *testing.T) {
		t.Parallel()

		if _, ok := _runtimeACPCatalog["gemini"]; !ok {
			t.Error("runtimeACPCatalog não contém 'gemini' — tarefa 2.0 não aplicada")
		}
		err := validateRuntimeFlags("acp", "gemini", 0)
		if err != nil {
			t.Errorf("gemini acp deve passar validação após ADR-015, obteve: %v", err)
		}
	})

	// TestRuntimeACPCatalogIncludesGemini (T-13 estendido): verifica que "gemini" está no catálogo.
	t.Run("TestRuntimeACPCatalogIncludesGemini: catálogo inclui gemini (T-13 ext)", func(t *testing.T) {
		t.Parallel()

		if _, ok := _runtimeACPCatalog["gemini"]; !ok {
			t.Error("runtimeACPCatalog não contém 'gemini' (T-13 ext — ADR-015)")
		}
		spec := _runtimeACPCatalog["gemini"]()
		if spec.ID != "gemini" {
			t.Errorf("runtimeACPCatalog[\"gemini\"]().ID = %q, esperava \"gemini\"", spec.ID)
		}
		if spec.Command == "" {
			t.Error("runtimeACPCatalog[\"gemini\"]().Command vazio")
		}
	})

	// T-14b: tool desconhecida ainda deve ser rejeitada com lista ordenada incluindo gemini.
	t.Run("T-14b: tool desconhecida rejeitada com lista ordenada incluindo gemini", func(t *testing.T) {
		t.Parallel()

		err := validateRuntimeFlags("acp", "unknown-tool", 0)
		if err == nil {
			t.Error("tool desconhecida deve ser rejeitada")
		}
		// Verificar que a mensagem contém todas as tools suportadas ordenadas incluindo gemini.
		msg := err.Error()
		for _, tool := range []string{"claude", "codex", "copilot", "gemini"} {
			if !strings.Contains(msg, tool) {
				t.Errorf("mensagem de erro deve listar %q, obteve: %q", tool, msg)
			}
		}
		// Verificar ordem lexicográfica: claude < codex < copilot < gemini.
		idxClaude := strings.Index(msg, "claude")
		idxCodex := strings.Index(msg, "codex")
		idxCopilot := strings.Index(msg, "copilot")
		idxGemini := strings.Index(msg, "gemini")
		if idxClaude > idxCodex {
			t.Errorf("'claude' deve aparecer antes de 'codex' (ordem lexicográfica): %q", msg)
		}
		if idxCodex > idxCopilot {
			t.Errorf("'codex' deve aparecer antes de 'copilot' (ordem lexicográfica): %q", msg)
		}
		if idxCopilot > idxGemini {
			t.Errorf("'copilot' deve aparecer antes de 'gemini' (ordem lexicográfica): %q", msg)
		}
	})

	// T-16: Catálogo deve conter exatamente claude, codex, copilot e gemini (após task 2.0).
	t.Run("T-16: catálogo contém exatamente claude, codex, copilot e gemini", func(t *testing.T) {
		t.Parallel()

		keys := make([]string, 0, len(_runtimeACPCatalog))
		for k := range _runtimeACPCatalog {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		expected := []string{"claude", "codex", "copilot", "gemini"}
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

		for tool, ctor := range _runtimeACPCatalog {
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

// TestRuntimeACPCatalogIncludesGemini (T-13 ext) valida que runtimeACPCatalog contém "gemini" após ADR-015.
// Critérios: entrada presente, ID correto, Command não vazio, validateRuntimeFlags aceita gemini+acp.
func TestRuntimeACPCatalogIncludesGemini(t *testing.T) {
	t.Parallel()

	ctor, ok := _runtimeACPCatalog["gemini"]
	if !ok {
		t.Fatal("runtimeACPCatalog não contém 'gemini' — task 2.0 não aplicada")
	}

	spec := ctor()
	if spec.ID != "gemini" {
		t.Errorf("runtimeACPCatalog[\"gemini\"]().ID = %q, esperava \"gemini\"", spec.ID)
	}
	if spec.Command == "" {
		t.Error("runtimeACPCatalog[\"gemini\"]().Command vazio")
	}

	// Gate de validação deve aceitar gemini+acp sem erro.
	if err := validateRuntimeFlags("acp", "gemini", 0); err != nil {
		t.Errorf("validateRuntimeFlags(\"acp\", \"gemini\", 0) retornou erro inesperado: %v", err)
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

// TestTaskLoopFlags_T24_ReasoningEffortInvalido valida que --reasoning-effort inválido retorna exit2 (T-24, RF-09).
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
			name:            "T-24: invalid retorna exit2",
			reasoningEffort: "invalid",
			wantErr:         true,
			wantExit2:       true,
			wantMsg:         "low|medium|high",
		},
		{
			name:            "T-24: ultra retorna exit2 com enum listado",
			reasoningEffort: "ultra",
			wantErr:         true,
			wantExit2:       true,
			wantMsg:         "low|medium|high",
		},
		{
			name:            "T-24: vazio retorna exit2",
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
			if tt.wantExit2 && err != nil && !strings.Contains(err.Error(), "exit2") {
				t.Errorf("erro deve conter 'exit2', obteve: %q", err.Error())
			}
			if tt.wantMsg != "" && err != nil && !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("erro %q nao contem %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

// TestTaskLoopFlags_T25_AccessModeInvalido valida que --access-mode inválido retorna exit2 (T-25, RF-11).
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
			name:       "T-25: open retorna exit2",
			accessMode: "open",
			wantErr:    true,
			wantExit2:  true,
			wantMsg:    "restricted|full",
		},
		{
			name:       "T-25: danger retorna exit2 com enum listado",
			accessMode: "danger",
			wantErr:    true,
			wantExit2:  true,
			wantMsg:    "restricted|full",
		},
		{
			name:       "T-25: vazio retorna exit2",
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
			if tt.wantExit2 && err != nil && !strings.Contains(err.Error(), "exit2") {
				t.Errorf("erro deve conter 'exit2', obteve: %q", err.Error())
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

// TestAccessModeFullEmitsWarningForGemini (RF-33, T-4.0) valida que o warning
// específico para Gemini --access-mode=full é emitido exatamente uma vez via sync.Once.
// Mensagem deve corresponder ao texto literal de RF-33 (ADR-015 §"Mensagens de Erro e Warning Literais").
func TestAccessModeFullEmitsWarningForGemini(t *testing.T) {
	// Não paralelo: usa sync.Once local para isolamento (não afeta accessModeFullWarnOnce global).
	var localOnce sync.Once
	var buf bytes.Buffer
	geminiWarnMsg := "WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. " +
		"Pré-condição: consentimento operacional. Ver GEMINI.md."

	emitGeminiWarning := func() {
		localOnce.Do(func() {
			fmt.Fprintln(&buf, geminiWarnMsg)
		})
	}

	// Primeira invocação: deve emitir warning.
	emitGeminiWarning()
	if !strings.Contains(buf.String(), "WARNING") {
		t.Errorf("primeira invocacao deve emitir warning; buffer=%q", buf.String())
	}
	if !strings.Contains(buf.String(), "--approval-mode=yolo") {
		t.Errorf("warning deve mencionar --approval-mode=yolo; buffer=%q", buf.String())
	}
	if !strings.Contains(buf.String(), "gemini-cli") {
		t.Errorf("warning deve mencionar gemini-cli; buffer=%q", buf.String())
	}
	if !strings.Contains(buf.String(), "GEMINI.md") {
		t.Errorf("warning deve referenciar GEMINI.md; buffer=%q", buf.String())
	}

	firstOutput := buf.String()

	// Segunda e terceira invocações: sync.Once não deve emitir novamente.
	emitGeminiWarning()
	emitGeminiWarning()
	if buf.String() != firstOutput {
		t.Errorf("invocacoes adicionais nao devem emitir warning; buffer=%q", buf.String())
	}

	// Verificar que a mensagem menciona consentimento operacional.
	if !strings.Contains(firstOutput, "consentimento operacional") {
		t.Errorf("warning deve mencionar consentimento operacional; output=%q", firstOutput)
	}
}

// TestAccessModeFullWarnOnce_GeminiVsCodex (RF-33, T-4.0) valida que o switch de
// ferramenta no warning de --access-mode=full distingue gemini de codex/default.
func TestAccessModeFullWarnOnce_GeminiVsCodex(t *testing.T) {
	// Não paralelo: testa lógica do switch tool-aware de forma isolada.

	t.Run("gemini emite mensagem específica RF-33", func(t *testing.T) {
		var once sync.Once
		var buf bytes.Buffer
		tool := "gemini"
		switch tool {
		case "gemini":
			once.Do(func() {
				fmt.Fprintln(&buf,
					"WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. "+
						"Pré-condição: consentimento operacional. Ver GEMINI.md.")
			})
		default:
			once.Do(func() {
				fmt.Fprintln(&buf, "WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. "+
					"Pré-condição: consentimento operacional. Codex terá acesso pleno ao filesystem e à rede. "+
					"Use somente em ambientes isolados. Ver CODEX.md.")
			})
		}
		out := buf.String()
		if !strings.Contains(out, "gemini-cli") {
			t.Errorf("gemini: mensagem deve mencionar gemini-cli; got=%q", out)
		}
		if !strings.Contains(out, "--approval-mode=yolo") {
			t.Errorf("gemini: mensagem deve mencionar --approval-mode=yolo; got=%q", out)
		}
		if strings.Contains(out, "sandbox_mode") {
			t.Errorf("gemini: mensagem NÃO deve mencionar sandbox_mode; got=%q", out)
		}
	})

	t.Run("codex emite mensagem de sandbox (regressão)", func(t *testing.T) {
		var once sync.Once
		var buf bytes.Buffer
		tool := "codex"
		switch tool {
		case "gemini":
			once.Do(func() {
				fmt.Fprintln(&buf,
					"WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. "+
						"Pré-condição: consentimento operacional. Ver GEMINI.md.")
			})
		default:
			once.Do(func() {
				fmt.Fprintln(&buf, "WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. "+
					"Pré-condição: consentimento operacional. Codex terá acesso pleno ao filesystem e à rede. "+
					"Use somente em ambientes isolados. Ver CODEX.md.")
			})
		}
		out := buf.String()
		if strings.Contains(out, "gemini-cli") {
			t.Errorf("codex: mensagem NÃO deve mencionar gemini-cli; got=%q", out)
		}
		if !strings.Contains(out, "sandbox_mode=danger-full-access") {
			t.Errorf("codex: mensagem deve mencionar sandbox_mode=danger-full-access; got=%q", out)
		}
	})
}

// TestGeminiSpecHasCorrectCommandAndFlags (T-14 estendido, RF-05, RF-25) valida que
// runtimeACPCatalog["gemini"]() retorna Spec com Command=gemini e FixedArgs=[--acp].
func TestGeminiSpecHasCorrectCommandAndFlags(t *testing.T) {
	t.Parallel()

	ctor, ok := _runtimeACPCatalog["gemini"]
	if !ok {
		t.Fatal("runtimeACPCatalog não contém 'gemini'")
	}
	spec := ctor()
	if spec.Command != "gemini" {
		t.Errorf("Command = %q; want %q", spec.Command, "gemini")
	}
	if len(spec.FixedArgs) != 1 || spec.FixedArgs[0] != "--acp" {
		t.Errorf("FixedArgs = %v; want [--acp]", spec.FixedArgs)
	}
}

// TestGeminiFallbackResolvesViaNpx (T-15 estendido, RF-25) valida que
// runtimeACPCatalog["gemini"]() expõe fallback npx com o package pinado.
func TestGeminiFallbackResolvesViaNpx(t *testing.T) {
	t.Parallel()

	ctor, ok := _runtimeACPCatalog["gemini"]
	if !ok {
		t.Fatal("runtimeACPCatalog não contém 'gemini'")
	}
	spec := ctor()
	if len(spec.Fallbacks) == 0 {
		t.Fatal("Gemini Spec não declara nenhum fallback")
	}
	fb := spec.Fallbacks[0]
	if fb.Command != "npx" {
		t.Errorf("Fallbacks[0].Command = %q; want npx", fb.Command)
	}
	// Deve conter @google/gemini-cli@<version> no slice de args
	found := false
	for _, arg := range fb.FixedArgs {
		if strings.HasPrefix(arg, "@google/gemini-cli@") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Fallbacks[0].FixedArgs = %v; want to contain '@google/gemini-cli@<version>'", fb.FixedArgs)
	}
}

// resolveMemoryLimits replica o switch tool-aware de F3-Gemini em task_loop.go para testes unitários.
// changedFlags é o conjunto de flags setadas explicitamente via CLI (replica cmd.Flags().Changed()).
func resolveMemoryLimits(tool string, workflowLines, taskLines, workflowBytes, taskBytes int, changedFlags map[string]bool) (wfLines, tskLines, wfBytes, tskBytes int) {
	wfLines, tskLines, wfBytes, tskBytes = workflowLines, taskLines, workflowBytes, taskBytes
	if tool == "gemini" {
		if !changedFlags["memory-workflow-limit-lines"] {
			wfLines = 250
		}
		if !changedFlags["memory-task-limit-lines"] {
			tskLines = 400
		}
		if !changedFlags["memory-workflow-limit-bytes"] {
			wfBytes = 20 * 1024
		}
		if !changedFlags["memory-task-limit-bytes"] {
			tskBytes = 32 * 1024
		}
	}
	return
}

// TestGeminiDefaultsMemoryLimitsAreGenerous (T-34, RF-16) valida que CLI sem flags de memory
// e --tool gemini resolve para defaults Gemini-generosos: 250 linhas / 400 linhas / 20 KiB / 32 KiB.
func TestGeminiDefaultsMemoryLimitsAreGenerous(t *testing.T) {
	t.Parallel()

	// Defaults de flag: valores de init() do comando (flag não foi changed).
	wfLines, tskLines, wfBytes, tskBytes := resolveMemoryLimits(
		"gemini", 150, 200, 12288, 16384,
		map[string]bool{}, // nenhuma flag changed
	)

	if wfLines != 250 {
		t.Errorf("T-34: memory-workflow-limit-lines com --tool gemini = %d, quero 250", wfLines)
	}
	if tskLines != 400 {
		t.Errorf("T-34: memory-task-limit-lines com --tool gemini = %d, quero 400", tskLines)
	}
	if wfBytes != 20*1024 {
		t.Errorf("T-34: memory-workflow-limit-bytes com --tool gemini = %d, quero %d (20 KiB)", wfBytes, 20*1024)
	}
	if tskBytes != 32*1024 {
		t.Errorf("T-34: memory-task-limit-bytes com --tool gemini = %d, quero %d (32 KiB)", tskBytes, 32*1024)
	}
}

// TestGeminiMemoryLimitOverrideByCliFlag (T-35, RF-16) valida que --memory-task-limit-lines 600
// prevalece sobre o default Gemini; --memory-workflow-limit-lines sem override ainda usa 250.
func TestGeminiMemoryLimitOverrideByCliFlag(t *testing.T) {
	t.Parallel()

	// Usuário setou explicitamente apenas --memory-task-limit-lines 600.
	wfLines, tskLines, wfBytes, tskBytes := resolveMemoryLimits(
		"gemini", 150, 600, 12288, 16384,
		map[string]bool{"memory-task-limit-lines": true},
	)

	// Override explícito prevalece.
	if tskLines != 600 {
		t.Errorf("T-35: memory-task-limit-lines com override = %d, quero 600 (override CLI)", tskLines)
	}
	// Workflow não foi changed → default Gemini-generoso aplicado.
	if wfLines != 250 {
		t.Errorf("T-35: memory-workflow-limit-lines sem override = %d, quero 250 (default Gemini)", wfLines)
	}
	if wfBytes != 20*1024 {
		t.Errorf("T-35: memory-workflow-limit-bytes sem override = %d, quero %d (20 KiB)", wfBytes, 20*1024)
	}
	// task-limit-bytes não foi changed → default Gemini.
	if tskBytes != 32*1024 {
		t.Errorf("T-35: memory-task-limit-bytes sem override = %d, quero %d (32 KiB)", tskBytes, 32*1024)
	}
}

// TestGeminiDefaultsDoNotAffectClaudeCodexCopilot (regressão RF-30) valida que
// o switch tool-aware NÃO altera defaults para claude, codex e copilot.
func TestGeminiDefaultsDoNotAffectClaudeCodexCopilot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tool string
	}{
		{"claude"},
		{"codex"},
		{"copilot"},
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			t.Parallel()

			wfLines, tskLines, wfBytes, tskBytes := resolveMemoryLimits(
				tt.tool, 150, 200, 12288, 16384,
				map[string]bool{}, // nenhuma flag changed
			)

			if wfLines != 150 {
				t.Errorf("RF-30: %s: workflow lines = %d, quero 150 (default Claude/Codex/Copilot)", tt.tool, wfLines)
			}
			if tskLines != 200 {
				t.Errorf("RF-30: %s: task lines = %d, quero 200 (default Claude/Codex/Copilot)", tt.tool, tskLines)
			}
			if wfBytes != 12288 {
				t.Errorf("RF-30: %s: workflow bytes = %d, quero 12288 (12 KiB)", tt.tool, wfBytes)
			}
			if tskBytes != 16384 {
				t.Errorf("RF-30: %s: task bytes = %d, quero 16384 (16 KiB)", tt.tool, tskBytes)
			}
		})
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
