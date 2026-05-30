package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/bugschema"
	"github.com/spf13/cobra"
)

func newValidateBugsCmd() *cobra.Command {
	var validateBugsSchema string

	cmd := &cobra.Command{
		Use:   "validate-bugs <bugs.json>",
		Short: "Valida um array de bugs contra bug-schema.json",
		Long: `Valida o array JSON de bugs contra o schema canônico de bugs.

Exemplos:
  ai-spec-harness validate-bugs bugs.json
  ai-spec-harness validate-bugs bugs.json --schema /path/to/bug-schema.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			schemaPath := validateBugsSchema
			if schemaPath == "" {
				schemaPath = ".agents/skills/agent-governance/references/bug-schema.json"
			}
			return bugschema.NewValidator().Validate(args[0], schemaPath)
		},
	}

	cmd.Flags().StringVar(&validateBugsSchema, "schema", "", "Caminho alternativo para bug-schema.json")
	return cmd
}
