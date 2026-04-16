package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	"github.com/jailtonjunior/orchestrator/internal/cli"
	"github.com/jailtonjunior/orchestrator/internal/tui"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Pre-scan os.Args for --no-tui before cobra parses flags so that the
	// correct bootstrap wiring (TUI vs plain text) can be selected up front.
	// Cobra still parses --no-tui normally; this scan is read-only.
	noTUI := containsArg(os.Args[1:], "--no-tui")

	app, err := buildApp(noTUI)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "bootstrap error: %v\n", err)
		os.Exit(1)
	}

	rootCmd := cli.NewRootCommand(app, fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date))
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// buildApp creates the dependency graph. When TUI mode is active, TUI-specific
// wiring (TUIPrompter and tuiProgressReporter) is substituted for the default
// terminal prompter and CLI renderer so the engine communicates via channels.
func buildApp(noTUI bool) (*bootstrap.App, error) {
	if tui.ShouldUseTUI(noTUI) {
		wiring := tui.NewWiring()
		app, err := bootstrap.NewWithLoggerOutput(os.Stdin, os.Stdout, wiring.Reporter, wiring.Prompter, io.Discard)
		if err != nil {
			return nil, err
		}
		app.TUIWiring = wiring
		return app, nil
	}

	renderer := cli.NewRenderer(os.Stdout)
	return bootstrap.New(os.Stdin, os.Stdout, renderer, nil)
}

// containsArg reports whether flag is effectively set in args.
// It matches both "--no-tui" (shorthand) and "--no-tui=true" (explicit form)
// while correctly ignoring "--no-tui=false".
func containsArg(args []string, flag string) bool {
	return slices.Contains(args, flag) || slices.Contains(args, flag+"=true")
}
