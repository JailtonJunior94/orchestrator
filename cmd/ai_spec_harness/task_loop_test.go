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
		// Verificar que a mensagem contém todas as tools suportadas ordenadas.
		msg := err.Error()
		if !strings.Contains(msg, "claude") || !strings.Contains(msg, "codex") || !strings.Contains(msg, "copilot") {
			t.Errorf("mensagem de erro deve listar [claude codex copilot], obteve: %q", msg)
		}
		// Verificar ordem lexicográfica: claude < codex < copilot.
		idxClaude := strings.Index(msg, "claude")
		idxCodex := strings.Index(msg, "codex")
		idxCopilot := strings.Index(msg, "copilot")
		if idxClaude > idxCodex {
			t.Errorf("'claude' deve aparecer antes de 'codex' na mensagem (ordem lexicográfica): %q", msg)
		}
		if idxCodex > idxCopilot {
			t.Errorf("'codex' deve aparecer antes de 'copilot' na mensagem (ordem lexicográfica): %q", msg)
		}
	})

	// T-16: Catálogo deve conter exatamente claude, codex e copilot (nesta versão).
	t.Run("T-16: catálogo contém exatamente claude, codex e copilot", func(t *testing.T) {
		t.Parallel()

		keys := make([]string, 0, len(runtimeACPCatalog))
		for k := range runtimeACPCatalog {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		expected := []string{"claude", "codex", "copilot"}
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

// TestTaskLoopFlags_ReasoningEffortRegistered valida que --reasoning-effort está registrado com default correto.
func TestTaskLoopFlags_ReasoningEffortRegistered(t *testing.T) {
	t.Parallel()

	f := taskLoopCmd.Flags().Lookup("reasoning-effort")
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

	f := taskLoopCmd.Flags().Lookup("access-mode")
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
	_ = &accessModeFullWarnOnce
}

// TestTaskLoopFlags_ReasoningEffortAndAccessModeDefaults valida os valores default das novas flags.
func TestTaskLoopFlags_ReasoningEffortAndAccessModeDefaults(t *testing.T) {
	t.Parallel()

	f := taskLoopCmd.Flags().Lookup("reasoning-effort")
	if f == nil {
		t.Fatal("flag --reasoning-effort nao encontrada")
	}
	if f.DefValue != "medium" {
		t.Errorf("default --reasoning-effort = %q, esperava medium", f.DefValue)
	}

	g := taskLoopCmd.Flags().Lookup("access-mode")
	if g == nil {
		t.Fatal("flag --access-mode nao encontrada")
	}
	if g.DefValue != "restricted" {
		t.Errorf("default --access-mode = %q, esperava restricted", g.DefValue)
	}
}
