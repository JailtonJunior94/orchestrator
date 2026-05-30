package aispecharness

import (
	"strings"
	"testing"
)

// TestChangelogVersion_Validation verifica as mensagens de validacao da flag --version em changelog.
func TestChangelogVersion_Validation(t *testing.T) {
	cases := []struct {
		name    string
		version string // vazio = nao definir (ausente)
		wantMsg string
	}{
		{
			name:    "flag ausente",
			version: "",
			wantMsg: "flag --version e obrigatoria",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newChangelogCmd()
			if tc.version != "" {
				if err := cmd.Flags().Set("version", tc.version); err != nil {
					t.Fatal(err)
				}
			}

			err := cmd.RunE(cmd, nil)
			if err == nil {
				t.Fatal("esperava erro")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("erro: got %q, quer conter %q", err.Error(), tc.wantMsg)
			}
			if !strings.Contains(err.Error(), "Exemplo:") {
				t.Errorf("erro deve conter 'Exemplo:', got: %q", err.Error())
			}
		})
	}
}

// TestUpdateVersion_FlagValidation verifica mensagens amigaveis de flag obrigatoria.
func TestUpdateVersion_FlagValidation(t *testing.T) {
	cases := []struct {
		name    string
		version string
		wantMsg string
	}{
		{
			name:    "flag ausente retorna mensagem obrigatoria",
			version: "",
			wantMsg: "flag --version e obrigatoria",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newUpdateVersionCmd()
			if tc.version != "" {
				if err := cmd.Flags().Set("version", tc.version); err != nil {
					t.Fatal(err)
				}
			}

			err := cmd.RunE(cmd, nil)
			if err == nil {
				t.Fatal("esperava erro")
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("erro: got %q, quer conter %q", err.Error(), tc.wantMsg)
			}
			if !strings.Contains(err.Error(), "Exemplo:") {
				t.Errorf("erro deve conter 'Exemplo:', got: %q", err.Error())
			}
		})
	}
}
