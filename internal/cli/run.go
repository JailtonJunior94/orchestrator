package cli

import (
	"fmt"
	"os"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	"github.com/spf13/cobra"
)

// NewRunCommand creates the `orq run` command.
func NewRunCommand(app *bootstrap.App) *cobra.Command {
	var inlineInput string
	var fileInput string

	cmd := &cobra.Command{
		Use:   "run <workflow>",
		Short: "Executa um workflow built-in",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if inlineInput != "" && fileInput != "" {
				return fmt.Errorf("--input e --file não podem ser usados juntos")
			}

			var input string
			switch {
			case inlineInput != "":
				input = inlineInput
			case fileInput != "":
				data, err := os.ReadFile(fileInput)
				if err != nil {
					return fmt.Errorf("falha ao ler arquivo de input %q: %w", fileInput, err)
				}
				input = string(data)
			default:
				return fmt.Errorf("é obrigatório informar --input ou --file")
			}

			_, err := app.Runtime.Run(cmd.Context(), args[0], input)
			return translateError(err)
		},
	}

	cmd.Flags().StringVar(&inlineInput, "input", "", "Input inline do workflow")
	cmd.Flags().StringVarP(&fileInput, "file", "f", "", "Arquivo com o input do workflow")
	return cmd
}
