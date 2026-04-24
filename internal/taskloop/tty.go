package taskloop

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// UIMode representa a forma de apresentacao do task-loop.
type UIMode string

const (
	UIModeAuto  UIMode = "auto"
	UIModeTUI   UIMode = "tui"
	UIModePlain UIMode = "plain"
)

func (m UIMode) Validate() error {
	switch m {
	case "", UIModeAuto, UIModeTUI, UIModePlain:
		return nil
	default:
		return fmt.Errorf("modo de UI invalido: %q", m)
	}
}

// ParseUIMode valida e normaliza a flag --ui.
func ParseUIMode(raw string) (UIMode, error) {
	mode := UIMode(strings.TrimSpace(strings.ToLower(raw)))
	if mode == "" {
		return UIModeAuto, nil
	}
	if err := mode.Validate(); err != nil {
		return "", err
	}
	return mode, nil
}

func normalizeRequestedUIMode(mode UIMode) (UIMode, error) {
	if mode == "" {
		return UIModeAuto, nil
	}
	if err := mode.Validate(); err != nil {
		return "", err
	}
	return mode, nil
}

func normalizeEffectiveUIMode(mode UIMode, requested UIMode, capabilities TerminalCapabilities) (UIMode, error) {
	if mode == "" {
		return ResolveUIMode(requested, capabilities)
	}
	if err := mode.Validate(); err != nil {
		return "", err
	}
	return mode, nil
}

// TerminalCapabilities descreve as capacidades minimas do terminal atual.
type TerminalCapabilities struct {
	Interactive       bool
	Width             int
	Height            int
	SupportsAltScreen bool
}

// TerminalDetector detecta capacidades do terminal sem acoplar a CLI a implementacao concreta.
type TerminalDetector interface {
	Detect(stdout io.Writer, stderr io.Writer) TerminalCapabilities
}

type stdTerminalDetector struct {
	lookupEnv  func(string) string
	isTerminal func(io.Writer) bool
}

// NewTerminalDetector cria o detector padrao do processo local.
func NewTerminalDetector() TerminalDetector {
	return stdTerminalDetector{
		lookupEnv:  os.Getenv,
		isTerminal: isTerminalWriter,
	}
}

func (d stdTerminalDetector) Detect(stdout io.Writer, stderr io.Writer) TerminalCapabilities {
	interactive := d.isTerminal(stdout)
	if !interactive {
		return TerminalCapabilities{}
	}
	return TerminalCapabilities{
		Interactive:       true,
		Width:             parseTerminalDimension(d.lookupEnv("COLUMNS")),
		Height:            parseTerminalDimension(d.lookupEnv("LINES")),
		SupportsAltScreen: true,
	}
}

// ResolveUIMode decide o modo efetivo segundo as regras da tech spec.
func ResolveUIMode(requested UIMode, capabilities TerminalCapabilities) (UIMode, error) {
	switch requested {
	case "", UIModeAuto:
		if capabilities.Interactive {
			return UIModeTUI, nil
		}
		return UIModePlain, nil
	case UIModePlain:
		return UIModePlain, nil
	case UIModeTUI:
		if capabilities.Interactive {
			return UIModeTUI, nil
		}
		return "", fmt.Errorf("%w: use --ui=plain ou um TTY compativel", ErrInteractiveUnavailable)
	default:
		return "", fmt.Errorf("modo de UI invalido: %q", requested)
	}
}

func parseTerminalDimension(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0
	}
	return value
}

func isTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
