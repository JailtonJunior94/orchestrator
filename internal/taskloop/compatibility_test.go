package taskloop

import (
	"errors"
	"testing"
)

func TestCompatibilityTable_IsSupported(t *testing.T) {
	t.Parallel()

	table := NewCompatibilityTable()

	tests := []struct {
		name  string
		tool  string
		model string
		want  bool
	}{
		// combinacoes catalogadas — devem retornar true
		{name: "claude + claude-opus-4-7", tool: "claude", model: "claude-opus-4-7", want: true},
		{name: "claude + claude-opus-4-6", tool: "claude", model: "claude-opus-4-6", want: true},
		{name: "claude + claude-sonnet-4-6", tool: "claude", model: "claude-sonnet-4-6", want: true},
		{name: "claude + claude-haiku-4-5", tool: "claude", model: "claude-haiku-4-5", want: true},
		{name: "codex + gpt-5.4", tool: "codex", model: "gpt-5.4", want: true},
		{name: "codex + gpt-5.4-mini", tool: "codex", model: "gpt-5.4-mini", want: true},
		{name: "codex + gpt-5.3-codex", tool: "codex", model: "gpt-5.3-codex", want: true},
		{name: "codex + gpt-5.3-codex-spark", tool: "codex", model: "gpt-5.3-codex-spark", want: true},
		{name: "gemini + auto", tool: "gemini", model: "auto", want: true},
		{name: "gemini + pro", tool: "gemini", model: "pro", want: true},
		{name: "gemini + flash", tool: "gemini", model: "flash", want: true},
		{name: "gemini + flash-lite", tool: "gemini", model: "flash-lite", want: true},
		{name: "gemini + gemini-2.5-pro", tool: "gemini", model: "gemini-2.5-pro", want: true},
		{name: "gemini + gemini-2.5-flash", tool: "gemini", model: "gemini-2.5-flash", want: true},
		{name: "gemini + gemini-2.5-flash-lite", tool: "gemini", model: "gemini-2.5-flash-lite", want: true},
		{name: "gemini + gemini-3-pro-preview", tool: "gemini", model: "gemini-3-pro-preview", want: true},
		{name: "copilot + claude-sonnet-4.5", tool: "copilot", model: "claude-sonnet-4.5", want: true},
		{name: "copilot + claude-sonnet-4.6", tool: "copilot", model: "claude-sonnet-4.6", want: true},
		{name: "copilot + gpt-5.4", tool: "copilot", model: "gpt-5.4", want: true},
		{name: "copilot + gpt-5.4-mini", tool: "copilot", model: "gpt-5.4-mini", want: true},
		{name: "copilot + haiku-4.5", tool: "copilot", model: "haiku-4.5", want: true},
		{name: "copilot + auto", tool: "copilot", model: "auto", want: true},
		// model vazio — sempre valido para ferramenta conhecida
		{name: "claude + model vazio", tool: "claude", model: "", want: true},
		{name: "codex + model vazio", tool: "codex", model: "", want: true},
		{name: "gemini + model vazio", tool: "gemini", model: "", want: true},
		{name: "copilot + model vazio", tool: "copilot", model: "", want: true},
		// combinacoes invalidas — modelo nao catalogado
		{name: "claude + modelo desconhecido", tool: "claude", model: "gpt-5.4", want: false},
		{name: "codex + modelo desconhecido", tool: "codex", model: "claude-sonnet-4-6", want: false},
		{name: "gemini + modelo desconhecido", tool: "gemini", model: "gpt-5.4", want: false},
		{name: "copilot + modelo desconhecido", tool: "copilot", model: "gemini-2.5-pro", want: false},
		// ferramenta desconhecida — sempre false
		{name: "ferramenta desconhecida com model", tool: "unknown-tool", model: "gpt-5.4", want: false},
		{name: "ferramenta desconhecida model vazio", tool: "unknown-tool", model: "", want: false},
		{name: "ferramenta vazia", tool: "", model: "claude-sonnet-4-6", want: false},
		{name: "ferramenta vazia model vazio", tool: "", model: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := table.IsSupported(tc.tool, tc.model)
			if got != tc.want {
				t.Errorf("IsSupported(%q, %q) = %v, quer %v", tc.tool, tc.model, got, tc.want)
			}
		})
	}
}

func TestCompatibilityTable_Models(t *testing.T) {
	t.Parallel()

	table := NewCompatibilityTable()

	tests := []struct {
		name      string
		tool      string
		wantLen   int
		wantFirst string
		wantNil   bool
	}{
		{name: "claude", tool: "claude", wantLen: 4, wantFirst: "claude-opus-4-7"},
		{name: "codex", tool: "codex", wantLen: 4, wantFirst: "gpt-5.4"},
		{name: "gemini", tool: "gemini", wantLen: 8, wantFirst: "auto"},
		{name: "copilot", tool: "copilot", wantLen: 6, wantFirst: "claude-sonnet-4.5"},
		{name: "ferramenta desconhecida retorna nil", tool: "desconhecida", wantNil: true},
		{name: "ferramenta vazia retorna nil", tool: "", wantNil: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := table.Models(tc.tool)

			if tc.wantNil {
				if got != nil {
					t.Errorf("Models(%q) = %v, quer nil", tc.tool, got)
				}
				return
			}

			if len(got) != tc.wantLen {
				t.Errorf("len(Models(%q)) = %d, quer %d", tc.tool, len(got), tc.wantLen)
			}

			if len(got) > 0 && got[0] != tc.wantFirst {
				t.Errorf("Models(%q)[0] = %q, quer %q", tc.tool, got[0], tc.wantFirst)
			}
		})
	}
}

func TestCompatibilityTable_ValidateCombination(t *testing.T) {
	t.Parallel()

	table := NewCompatibilityTable()

	tests := []struct {
		name    string
		tool    string
		model   string
		wantErr error
	}{
		{
			name:    "combinacao valida — claude + claude-sonnet-4-6",
			tool:    "claude",
			model:   "claude-sonnet-4-6",
			wantErr: nil,
		},
		{
			name:    "model vazio em ferramenta conhecida — valido",
			tool:    "gemini",
			model:   "",
			wantErr: nil,
		},
		{
			name:    "modelo invalido para ferramenta conhecida",
			tool:    "claude",
			model:   "gpt-5.4",
			wantErr: ErrModeloIncompativel,
		},
		{
			name:    "ferramenta desconhecida com model",
			tool:    "nao-existe",
			model:   "algum-modelo",
			wantErr: ErrModeloIncompativel,
		},
		{
			name:    "ferramenta desconhecida model vazio",
			tool:    "nao-existe",
			model:   "",
			wantErr: ErrModeloIncompativel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := table.ValidateCombination(tc.tool, tc.model)

			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("esperava erro %v, mas nao houve erro", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("errors.Is(err, %v) = false; err = %v", tc.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("nao esperava erro, obteve: %v", err)
			}
		})
	}
}

func TestErrModeloIncompativel_Sentinela(t *testing.T) {
	t.Parallel()

	table := NewCompatibilityTable()

	t.Run("wrapping contextual e verificavel via errors.Is", func(t *testing.T) {
		t.Parallel()
		err := table.ValidateCombination("claude", "modelo-inexistente")
		if !errors.Is(err, ErrModeloIncompativel) {
			t.Errorf("errors.Is(err, ErrModeloIncompativel) = false; err = %v", err)
		}
	})

	t.Run("ferramenta desconhecida model vazio gera ErrModeloIncompativel", func(t *testing.T) {
		t.Parallel()
		err := table.ValidateCombination("inexistente", "")
		if !errors.Is(err, ErrModeloIncompativel) {
			t.Errorf("errors.Is(err, ErrModeloIncompativel) = false; err = %v", err)
		}
	})
}
