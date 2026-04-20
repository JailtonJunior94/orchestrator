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
			changelogVersion = tc.version
			t.Cleanup(func() { changelogVersion = "" })

			err := changelogCmd.RunE(changelogCmd, nil)
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
			updateVersionVersion = tc.version
			t.Cleanup(func() {
				updateVersionVersion = ""
				updateVersionVersionFile = "VERSION"
			})

			err := updateVersionCmd.RunE(updateVersionCmd, nil)
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
