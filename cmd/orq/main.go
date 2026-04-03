package main

import (
	"fmt"
	"os"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	"github.com/jailtonjunior/orchestrator/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	renderer := cli.NewRenderer(os.Stdout)
	app, err := bootstrap.New(os.Stdin, os.Stdout, renderer)
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
