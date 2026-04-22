package taskloop

import (
	"errors"
	"strings"
	"testing"
)

func TestNewExecutionProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		role        string
		tool        string
		model       string
		wantRole    string
		wantTool    string
		wantProvider string
		wantModel   string
		wantErr     error
	}{
		{
			name:         "executor com claude sem model",
			role:         "executor",
			tool:         "claude",
			model:        "",
			wantRole:     "executor",
			wantTool:     "claude",
			wantProvider: "anthropic",
			wantModel:    "",
		},
		{
			name:         "executor com claude com model",
			role:         "executor",
			tool:         "claude",
			model:        "claude-sonnet-4-6",
			wantRole:     "executor",
			wantTool:     "claude",
			wantProvider: "anthropic",
			wantModel:    "claude-sonnet-4-6",
		},
		{
			name:         "reviewer com codex",
			role:         "reviewer",
			tool:         "codex",
			model:        "",
			wantRole:     "reviewer",
			wantTool:     "codex",
			wantProvider: "openai",
			wantModel:    "",
		},
		{
			name:         "executor com gemini",
			role:         "executor",
			tool:         "gemini",
			model:        "gemini-2.5-pro",
			wantRole:     "executor",
			wantTool:     "gemini",
			wantProvider: "google",
			wantModel:    "gemini-2.5-pro",
		},
		{
			name:         "executor com copilot",
			role:         "executor",
			tool:         "copilot",
			model:        "",
			wantRole:     "executor",
			wantTool:     "copilot",
			wantProvider: "github",
			wantModel:    "",
		},
		{
			name:    "role invalido",
			role:    "admin",
			tool:    "claude",
			model:   "",
			wantErr: ErrRoleInvalido,
		},
		{
			name:    "role vazio",
			role:    "",
			tool:    "claude",
			model:   "",
			wantErr: ErrRoleInvalido,
		},
		{
			name:    "tool invalida",
			role:    "executor",
			tool:    "gpt",
			model:   "",
			wantErr: ErrToolInvalida,
		},
		{
			name:    "tool vazia",
			role:    "executor",
			tool:    "",
			model:   "",
			wantErr: ErrToolInvalida,
		},
		{
			name:    "role e tool invalidos — role verificado primeiro",
			role:    "owner",
			tool:    "gpt",
			model:   "",
			wantErr: ErrRoleInvalido,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewExecutionProfile(tc.role, tc.tool, tc.model)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("esperava erro %v, mas nao houve erro", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("erro esperado %v, obteve %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("nao esperava erro, obteve: %v", err)
			}

			if got.Role() != tc.wantRole {
				t.Errorf("Role() = %q, quer %q", got.Role(), tc.wantRole)
			}
			if got.Tool() != tc.wantTool {
				t.Errorf("Tool() = %q, quer %q", got.Tool(), tc.wantTool)
			}
			if got.Provider() != tc.wantProvider {
				t.Errorf("Provider() = %q, quer %q", got.Provider(), tc.wantProvider)
			}
			if got.Model() != tc.wantModel {
				t.Errorf("Model() = %q, quer %q", got.Model(), tc.wantModel)
			}
		})
	}
}

func TestInferProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tool string
		want string
	}{
		{"claude", "anthropic"},
		{"codex", "openai"},
		{"gemini", "google"},
		{"copilot", "github"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tc := range tests {
		t.Run(tc.tool, func(t *testing.T) {
			t.Parallel()
			got := inferProvider(tc.tool)
			if got != tc.want {
				t.Errorf("inferProvider(%q) = %q, quer %q", tc.tool, got, tc.want)
			}
		})
	}
}

func TestResolveProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		tool          string
		execTool      string
		execModel     string
		revTool       string
		revModel      string
		wantNil       bool // true = espera retorno nil (modo simples)
		wantMode      string
		wantExecTool  string
		wantExecModel string
		wantRevTool   string
		wantRevModel  string
		wantRevNil    bool // true = espera Reviewer == nil
		wantErr       error
		wantErrMsg    string // substring da mensagem de erro (quando wantErr == nil mas espera erro)
	}{
		{
			name:    "modo simples — apenas tool",
			tool:    "claude",
			wantNil: true,
		},
		{
			name:          "modo avancado — apenas executor-tool",
			execTool:      "claude",
			wantMode:      "avancado",
			wantExecTool:  "claude",
			wantExecModel: "",
			wantRevTool:   "claude",  // herda executor
			wantRevModel:  "",
		},
		{
			name:          "modo avancado — executor e reviewer distintos",
			execTool:      "claude",
			execModel:     "claude-sonnet-4-6",
			revTool:       "codex",
			revModel:      "gpt-5",
			wantMode:      "avancado",
			wantExecTool:  "claude",
			wantExecModel: "claude-sonnet-4-6",
			wantRevTool:   "codex",
			wantRevModel:  "gpt-5",
		},
		{
			name:          "modo avancado — reviewer herda executor com model",
			execTool:      "gemini",
			execModel:     "gemini-2.5-pro",
			wantMode:      "avancado",
			wantExecTool:  "gemini",
			wantExecModel: "gemini-2.5-pro",
			wantRevTool:   "gemini", // herda
			wantRevModel:  "gemini-2.5-pro", // herda
		},
		{
			name:    "flags conflitantes — tool + executor-tool",
			tool:    "claude",
			execTool: "codex",
			wantErr: ErrFlagsConflitantes,
		},
		{
			name:    "flags conflitantes — tool + reviewer-tool",
			tool:    "claude",
			revTool: "codex",
			wantErr: ErrFlagsConflitantes,
		},
		{
			name:       "executor-model sem executor-tool",
			execModel:  "claude-sonnet-4-6",
			wantErrMsg: "--executor-model requer --executor-tool",
		},
		{
			name:       "reviewer-model sem reviewer-tool",
			execTool:   "claude",
			revModel:   "gpt-5",
			wantErrMsg: "--reviewer-model requer --reviewer-tool",
		},
		{
			name:    "executor-tool invalida",
			execTool: "gpt",
			wantErr: ErrToolInvalida,
		},
		{
			name:     "reviewer-tool invalida",
			execTool: "claude",
			revTool:  "unknown-tool",
			wantErr:  ErrToolInvalida,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveProfiles(tc.tool, tc.execTool, tc.execModel, tc.revTool, tc.revModel)

			// Verificar erro esperado por sentinel
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("esperava erro %v, mas nao houve erro", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("erro esperado %v, obteve %v", tc.wantErr, err)
				}
				return
			}

			// Verificar erro esperado por mensagem
			if tc.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("esperava erro contendo %q, mas nao houve erro", tc.wantErrMsg)
				}
				if msg := err.Error(); !strings.Contains(msg, tc.wantErrMsg) {
					t.Fatalf("esperava mensagem contendo %q, obteve %q", tc.wantErrMsg, msg)
				}
				return
			}

			if err != nil {
				t.Fatalf("nao esperava erro, obteve: %v", err)
			}

			// Modo simples: espera nil
			if tc.wantNil {
				if got != nil {
					t.Errorf("esperava ProfileConfig nil (modo simples), obteve %+v", got)
				}
				return
			}

			if got == nil {
				t.Fatal("esperava ProfileConfig nao-nil, obteve nil")
			}

			if got.Mode != tc.wantMode {
				t.Errorf("Mode = %q, quer %q", got.Mode, tc.wantMode)
			}

			if got.Executor.Tool() != tc.wantExecTool {
				t.Errorf("Executor.Tool() = %q, quer %q", got.Executor.Tool(), tc.wantExecTool)
			}
			if got.Executor.Model() != tc.wantExecModel {
				t.Errorf("Executor.Model() = %q, quer %q", got.Executor.Model(), tc.wantExecModel)
			}
			if got.Executor.Role() != "executor" {
				t.Errorf("Executor.Role() = %q, quer %q", got.Executor.Role(), "executor")
			}

			if tc.wantRevNil {
				if got.Reviewer != nil {
					t.Errorf("esperava Reviewer nil, obteve %+v", got.Reviewer)
				}
				return
			}

			if got.Reviewer == nil {
				t.Fatal("esperava Reviewer nao-nil, obteve nil")
			}
			if got.Reviewer.Tool() != tc.wantRevTool {
				t.Errorf("Reviewer.Tool() = %q, quer %q", got.Reviewer.Tool(), tc.wantRevTool)
			}
			if got.Reviewer.Model() != tc.wantRevModel {
				t.Errorf("Reviewer.Model() = %q, quer %q", got.Reviewer.Model(), tc.wantRevModel)
			}
			if got.Reviewer.Role() != "reviewer" {
				t.Errorf("Reviewer.Role() = %q, quer %q", got.Reviewer.Role(), "reviewer")
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	t.Run("ErrToolInvalida via NewExecutionProfile", func(t *testing.T) {
		t.Parallel()
		_, err := NewExecutionProfile("executor", "nao-existe", "")
		if !errors.Is(err, ErrToolInvalida) {
			t.Errorf("esperava errors.Is(err, ErrToolInvalida) true, obteve err=%v", err)
		}
	})

	t.Run("ErrRoleInvalido via NewExecutionProfile", func(t *testing.T) {
		t.Parallel()
		_, err := NewExecutionProfile("dono", "claude", "")
		if !errors.Is(err, ErrRoleInvalido) {
			t.Errorf("esperava errors.Is(err, ErrRoleInvalido) true, obteve err=%v", err)
		}
	})

	t.Run("ErrFlagsConflitantes via resolveProfiles", func(t *testing.T) {
		t.Parallel()
		_, err := ResolveProfiles("claude", "codex", "", "", "")
		if !errors.Is(err, ErrFlagsConflitantes) {
			t.Errorf("esperava errors.Is(err, ErrFlagsConflitantes) true, obteve err=%v", err)
		}
	})

	t.Run("ErrToolInvalida via resolveProfiles executor-tool invalida", func(t *testing.T) {
		t.Parallel()
		_, err := ResolveProfiles("", "invalid-tool", "", "", "")
		if !errors.Is(err, ErrToolInvalida) {
			t.Errorf("esperava errors.Is(err, ErrToolInvalida) true, obteve err=%v", err)
		}
	})
}

