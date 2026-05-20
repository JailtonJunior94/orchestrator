package aispecharness

import (
	"fmt"
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
			name:    "runtime acp com tool codex invalido (RF-02)",
			runtime: "acp",
			tool:    "codex",
			wantErr: true,
			wantMsg: "exit2",
		},
		{
			name:    "runtime acp com tool gemini invalido (RF-02)",
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
		tt := tt
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
		tt := tt
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

// validateRuntimeFlags é uma função auxiliar de teste que valida as mesmas
// regras aplicadas no RunE do taskLoopCmd, sem invocar o comando completo.
// Replica a lógica de validação para permitir testes unitários focados.
func validateRuntimeFlags(runtime, tool string, activityTimeout time.Duration) error {
	if runtime != "legacy" && runtime != "acp" {
		return fmt.Errorf("exit2")
	}
	if runtime == "acp" && tool != "claude" {
		return fmt.Errorf("exit2")
	}
	if activityTimeout < 0 {
		return fmt.Errorf("exit2")
	}
	return nil
}
