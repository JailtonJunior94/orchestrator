package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/bugschema"
	"github.com/spf13/cobra"
)

var validateBugsSchema string

var validateBugsCmd = &cobra.Command{
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
		return bugschema.Validate(args[0], schemaPath)
	},
}

func init() {
	validateBugsCmd.Flags().StringVar(&validateBugsSchema, "schema", "", "Caminho alternativo para bug-schema.json")
	rootCmd.AddCommand(validateBugsCmd)
}
