package cli

import (
	"fmt"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	"github.com/spf13/cobra"
)

// NewRootCommand creates the root ORQ command tree.
func NewRootCommand(app *bootstrap.App, version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "orq",
		Short:   "ORQ - AI workflow orchestrator CLI",
		Long:    "ORQ runs AI-assisted development workflows, integrating multiple agents (Claude CLI, Copilot CLI) into a single, controlled and auditable pipeline.",
		Version: version,
	}

	rootCmd.AddCommand(NewRunCommand(app))
	rootCmd.AddCommand(NewContinueCommand(app))
	rootCmd.AddCommand(NewListCommand(app))
	rootCmd.AddCommand(NewInstallCommand(app))

	return rootCmd
}

func installServiceOrError(app *bootstrap.App) error {
	if app == nil || app.Install == nil {
		return fmt.Errorf("install service is not configured")
	}
	return nil
}
