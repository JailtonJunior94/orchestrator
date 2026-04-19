package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/scaffold"
	"github.com/spf13/cobra"
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold <language>",
	Short: "Gera scaffold de uma nova skill de linguagem",
	Long: `Cria a estrutura de uma nova skill de linguagem com SKILL.md,
reference stubs e comando Gemini.

Exemplos:
  ai-spec-harness scaffold rust
  ai-spec-harness scaffold elixir --root ~/ai-spec`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()
		svc := scaffold.NewService(fsys, printer)
		return svc.Execute(args[0], scaffoldRoot)
	},
}

var scaffoldRoot string

func init() {
	scaffoldCmd.Flags().StringVar(&scaffoldRoot, "root", ".", "Diretorio raiz do repositorio de governanca")
	rootCmd.AddCommand(scaffoldCmd)
}
