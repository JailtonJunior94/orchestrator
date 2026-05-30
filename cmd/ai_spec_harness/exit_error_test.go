package aispecharness

import (
	"errors"
	"fmt"
	"testing"
)

// TestExitCodeFor verifica que ExitCodeFor extrai o codigo de saida de um
// exitError e usa 1 como padrao para erros comuns, preservando o contrato de
// codigos de saida que antes era garantido por chamadas diretas a os.Exit.
func TestExitCodeFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "exitError codigo 1", err: newExitError(1), want: 1},
		{name: "exitError codigo 2", err: newExitError(2), want: 2},
		{name: "erro comum usa padrao 1", err: errors.New("falha qualquer"), want: 1},
		{name: "exitError envolvido", err: fmt.Errorf("contexto: %w", newExitError(2)), want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewExitResolver().CodeFor(tt.err); got != tt.want {
				t.Errorf("CodeFor(%v) = %d, esperado %d", tt.err, got, tt.want)
			}
		})
	}
}

// TestExitErrorImplementsError garante que exitError satisfaz error e expoe o
// codigo via ExitCode, permitindo que main traduza o codigo sem reimprimir a
// mensagem (que ja foi emitida pelo handler).
func TestExitErrorImplementsError(t *testing.T) {
	var err error = newExitError(2)
	if err.Error() == "" {
		t.Error("exitError.Error() nao deve ser vazio")
	}

	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatal("exitError deve ser recuperavel via errors.As")
	}
	if ee.ExitCode() != 2 {
		t.Errorf("ExitCode() = %d, esperado 2", ee.ExitCode())
	}
}
