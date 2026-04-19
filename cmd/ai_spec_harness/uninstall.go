package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/uninstall"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <path>",
	Short: "Remove governanca de IA de um projeto",
	Long: `Remove artefatos de governanca instalados pelo comando install.

Exemplos:
  ai-spec-harness uninstall ./meu-projeto
  ai-spec-harness uninstall ./meu-projeto --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()
		svc := uninstall.NewService(fsys, printer)
		return svc.Execute(args[0], uninstallDryRun)
	},
}

var uninstallDryRun bool

func init() {
	uninstallCmd.Flags().BoolVar(&uninstallDryRun, "dry-run", false, "Mostra o que seria removido sem executar")
	rootCmd.AddCommand(uninstallCmd)
}
