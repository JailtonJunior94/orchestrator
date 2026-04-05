package cli

import (
	"fmt"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	"github.com/spf13/cobra"
)

// NewListCommand creates the `orq list` command.
// In a TTY it launches the interactive TUI list; otherwise it prints plain text.
func NewListCommand(app *bootstrap.App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Lista workflows disponíveis",
		RunE: func(cmd *cobra.Command, _ []string) error {
			noTUI, _ := cmd.Root().PersistentFlags().GetBool("no-tui")
			if shouldUseTUI(noTUI) {
				summaries, err := app.Runtime.ListWorkflowDetails(cmd.Context())
				if err != nil {
					return translateError(err)
				}
				selected, tuiErr := runWorkflowListTUI(summaries)
				if tuiErr != nil {
					return tuiErr
				}
				if selected != "" {
					return translateError(executeWorkflowCommand(cmd, app, selected, ""))
				}
				return nil
			}

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
