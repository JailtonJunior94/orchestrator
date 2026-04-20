package aispecharness

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRequireFlag(t *testing.T) {
	cases := []struct {
		name      string
		setValue  string // vazio = nao chamar Set (flag ausente)
		wantErr   bool
		wantMsg   string
	}{
		{
			name:    "flag ausente: erro com 'e obrigatoria'",
			wantErr: true,
			wantMsg: "e obrigatoria",
		},
		{
			name:     "flag presente mas vazia: erro com 'nao pode ficar vazia'",
			setValue: "",
			wantErr:  true,
			wantMsg:  "nao pode ficar vazia",
		},
		{
			name:     "flag com valor: sem erro",
			setValue: "claude",
			wantErr:  false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("tools", "", "")

			if tc.setValue != "" || (tc.wantErr && tc.wantMsg == "nao pode ficar vazia") {
				// Simula --tools "" via cobra para marcar Changed=true com valor vazio
				_ = cmd.Flags().Set("tools", tc.setValue)
			}

			err := requireFlag(cmd, "tools", "ai-spec-harness install ./proj --tools claude")
			if tc.wantErr && err == nil {
				t.Fatal("esperava erro, nao obteve")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("nao esperava erro, obteve: %v", err)
			}
			if tc.wantErr && !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("mensagem de erro: got %q, quer conter %q", err.Error(), tc.wantMsg)
			}
		})
	}
}

func TestRequireFlag_ExemploNoErro(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("tools", "", "")

	err := requireFlag(cmd, "tools", "ai-spec-harness install ./proj --tools claude")
	if err == nil {
		t.Fatal("esperava erro")
	}
	if !strings.Contains(err.Error(), "Exemplo:") {
		t.Errorf("mensagem deve conter 'Exemplo:', got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "ai-spec-harness install ./proj --tools claude") {
		t.Errorf("mensagem deve conter o exemplo real, got: %q", err.Error())
	}
}
