package aispecharness

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
	"github.com/spf13/cobra"
)

type stubTerminalDetector struct {
	capabilities taskloop.TerminalCapabilities
}

func (d stubTerminalDetector) Detect(stdout, stderr io.Writer) taskloop.TerminalCapabilities {
	return d.capabilities
}

func newTaskLoopUITestCommand(t *testing.T) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{Use: "task-loop"}
	addTaskLoopUIFlags(cmd)
	return cmd
}

func TestTaskLoopUIFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		want    taskloop.UIMode
		wantErr string
	}{
		{
			name: "default usa auto",
			want: taskloop.UIModeAuto,
		},
		{
			name: "aceita ui plain",
			args: []string{"--ui=plain"},
			want: taskloop.UIModePlain,
		},
		{
			name: "no-ui vira plain",
			args: []string{"--no-ui"},
			want: taskloop.UIModePlain,
		},
		{
			name:    "no-ui conflita com ui diferente de plain",
			args:    []string{"--ui=tui", "--no-ui"},
			wantErr: "--no-ui nao pode ser combinado com --ui=tui",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newTaskLoopUITestCommand(t)
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() erro inesperado: %v", err)
			}

			got, err := requestedTaskLoopUIMode(cmd)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("erro = %v, esperado %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if got != tt.want {
				t.Fatalf("requestedTaskLoopUIMode() = %q, esperado %q", got, tt.want)
			}
		})
	}
}

func TestTaskLoopUIResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		capabilities  taskloop.TerminalCapabilities
		wantRequest   taskloop.UIMode
		wantEffective taskloop.UIMode
		wantErr       error
	}{
		{
			name:          "auto vira tui em terminal interativo",
			capabilities:  taskloop.TerminalCapabilities{Interactive: true},
			wantRequest:   taskloop.UIModeAuto,
			wantEffective: taskloop.UIModeTUI,
		},
		{
			name:          "auto degrada para plain fora de tty",
			capabilities:  taskloop.TerminalCapabilities{},
			wantRequest:   taskloop.UIModeAuto,
			wantEffective: taskloop.UIModePlain,
		},
		{
			name:          "ui plain preserva fallback textual",
			args:          []string{"--ui=plain"},
			capabilities:  taskloop.TerminalCapabilities{Interactive: true},
			wantRequest:   taskloop.UIModePlain,
			wantEffective: taskloop.UIModePlain,
		},
		{
			name:         "tui falha sem terminal interativo",
			args:         []string{"--ui=tui"},
			capabilities: taskloop.TerminalCapabilities{},
			wantErr:      taskloop.ErrInteractiveUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := newTaskLoopUITestCommand(t)
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() erro inesperado: %v", err)
			}

			requested, effective, capabilities, err := resolveTaskLoopUIMode(
				cmd,
				stubTerminalDetector{capabilities: tt.capabilities},
				bytes.NewBuffer(nil),
				bytes.NewBuffer(nil),
			)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("erro = %v, esperado errors.Is(..., %v)", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if requested != tt.wantRequest {
				t.Fatalf("requested = %q, esperado %q", requested, tt.wantRequest)
			}
			if effective != tt.wantEffective {
				t.Fatalf("effective = %q, esperado %q", effective, tt.wantEffective)
			}
			if capabilities != tt.capabilities {
				t.Fatalf("capabilities = %+v, esperado %+v", capabilities, tt.capabilities)
			}
		})
	}
}

// TestTaskLoopUIResolutionTUIExplicit valida que --ui=tui com terminal interativo
// resolve para TUI com sucesso e preserva as capacidades detectadas.
func TestTaskLoopUIResolutionTUIExplicit(t *testing.T) {
	t.Parallel()

	cmd := newTaskLoopUITestCommand(t)
	if err := cmd.ParseFlags([]string{"--ui=tui"}); err != nil {
		t.Fatalf("ParseFlags() erro inesperado: %v", err)
	}

	caps := taskloop.TerminalCapabilities{Interactive: true, Width: 120, Height: 40, SupportsAltScreen: true}
	requested, effective, capabilities, err := resolveTaskLoopUIMode(
		cmd,
		stubTerminalDetector{capabilities: caps},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("resolveTaskLoopUIMode() erro inesperado: %v", err)
	}
	if requested != taskloop.UIModeTUI {
		t.Errorf("requested = %q, esperado UIModeTUI", requested)
	}
	if effective != taskloop.UIModeTUI {
		t.Errorf("effective = %q, esperado UIModeTUI", effective)
	}
	if !capabilities.Interactive {
		t.Error("capabilities.Interactive deveria ser true para terminal interativo")
	}
}

// TestTaskLoopUIResolutionAutoNonInteractiveDegradesFully valida que auto degrada
// para plain quando nao ha TTY, mesmo com Width/Height preenchidos (terminal parcial).
func TestTaskLoopUIResolutionAutoNonInteractiveDegradesFully(t *testing.T) {
	t.Parallel()

	cmd := newTaskLoopUITestCommand(t)
	if err := cmd.ParseFlags(nil); err != nil {
		t.Fatalf("ParseFlags() erro inesperado: %v", err)
	}

	// Terminal com Width/Height mas sem Interactive (ex: CI com pseudo-tty parcial)
	caps := taskloop.TerminalCapabilities{Interactive: false, Width: 80, Height: 24}
	_, effective, _, err := resolveTaskLoopUIMode(
		cmd,
		stubTerminalDetector{capabilities: caps},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("resolveTaskLoopUIMode() erro inesperado: %v", err)
	}
	if effective != taskloop.UIModePlain {
		t.Errorf("effective = %q, esperado UIModePlain para terminal nao interativo", effective)
	}
}

// TestTaskLoopUIResolutionNoUIFlagPreservesPlain valida que --no-ui forca plain
// mesmo quando o terminal e interativo.
func TestTaskLoopUIResolutionNoUIFlagPreservesPlain(t *testing.T) {
	t.Parallel()

	cmd := newTaskLoopUITestCommand(t)
	if err := cmd.ParseFlags([]string{"--no-ui"}); err != nil {
		t.Fatalf("ParseFlags() erro inesperado: %v", err)
	}

	caps := taskloop.TerminalCapabilities{Interactive: true, Width: 120, Height: 40}
	requested, effective, _, err := resolveTaskLoopUIMode(
		cmd,
		stubTerminalDetector{capabilities: caps},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("resolveTaskLoopUIMode() erro inesperado: %v", err)
	}
	if requested != taskloop.UIModePlain {
		t.Errorf("requested = %q, esperado UIModePlain apos --no-ui", requested)
	}
	if effective != taskloop.UIModePlain {
		t.Errorf("effective = %q, esperado UIModePlain apos --no-ui", effective)
	}
}
