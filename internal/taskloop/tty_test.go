package taskloop

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestResolveUIMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		requested    UIMode
		capabilities TerminalCapabilities
		want         UIMode
		wantErr      error
	}{
		{
			name:         "auto usa tui quando terminal e interativo",
			requested:    UIModeAuto,
			capabilities: TerminalCapabilities{Interactive: true},
			want:         UIModeTUI,
		},
		{
			name:         "auto degrada para plain quando terminal nao e interativo",
			requested:    UIModeAuto,
			capabilities: TerminalCapabilities{},
			want:         UIModePlain,
		},
		{
			name:         "plain preserva modo textual",
			requested:    UIModePlain,
			capabilities: TerminalCapabilities{Interactive: true},
			want:         UIModePlain,
		},
		{
			name:         "tui permanece tui quando terminal e interativo",
			requested:    UIModeTUI,
			capabilities: TerminalCapabilities{Interactive: true},
			want:         UIModeTUI,
		},
		{
			name:      "tui falha quando terminal nao suporta modo iterativo",
			requested: UIModeTUI,
			wantErr:   ErrInteractiveUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ResolveUIMode(tt.requested, tt.capabilities)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("erro = %v, esperado errors.Is(..., %v)", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolveUIMode() = %q, esperado %q", got, tt.want)
			}
		})
	}
}

func TestTerminalDetector(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		isTerminal bool
		env        map[string]string
		want       TerminalCapabilities
	}{
		{
			name:       "nao marca terminal quando writer nao e interativo",
			isTerminal: false,
			env:        map[string]string{"COLUMNS": "120", "LINES": "40"},
			want:       TerminalCapabilities{},
		},
		{
			name:       "coleta largura e altura quando terminal e interativo",
			isTerminal: true,
			env:        map[string]string{"COLUMNS": "120", "LINES": "40"},
			want: TerminalCapabilities{
				Interactive:       true,
				Width:             120,
				Height:            40,
				SupportsAltScreen: true,
			},
		},
		{
			name:       "ignora dimensoes invalidas",
			isTerminal: true,
			env:        map[string]string{"COLUMNS": "x", "LINES": "-1"},
			want: TerminalCapabilities{
				Interactive:       true,
				SupportsAltScreen: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := stdTerminalDetector{
				lookupEnv: func(key string) string {
					return tt.env[key]
				},
				isTerminal: func(io.Writer) bool {
					return tt.isTerminal
				},
			}

			got := detector.Detect(bytes.NewBuffer(nil), bytes.NewBuffer(nil))
			if got != tt.want {
				t.Fatalf("Detect() = %+v, esperado %+v", got, tt.want)
			}
		})
	}
}
