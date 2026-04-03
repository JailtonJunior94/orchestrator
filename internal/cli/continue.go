package cli

import (
	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	"github.com/spf13/cobra"
)

// NewContinueCommand creates the `orq continue` command.
func NewContinueCommand(app *bootstrap.App) *cobra.Command {
	var runID string

	cmd := &cobra.Command{
		Use:   "continue",
		Short: "Retoma o último workflow pausado",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := app.Runtime.Continue(cmd.Context(), runID)
			return translateError(err)
		},
	}

	cmd.Flags().StringVar(&runID, "run-id", "", "ID explícito da run a ser retomada")
	return cmd
}
