package cli

import (
	"fmt"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	"github.com/spf13/cobra"
)

// NewListCommand creates the `orq list` command.
func NewListCommand(app *bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista workflows disponíveis",
		RunE: func(cmd *cobra.Command, _ []string) error {
			workflows, err := app.Runtime.ListWorkflows(cmd.Context())
			if err != nil {
				return translateError(err)
			}

			for _, workflow := range workflows {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), workflow); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
