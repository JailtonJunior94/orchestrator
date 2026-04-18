package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/inspect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <path>",
	Short: "Inspeciona o estado da instalacao de governanca",
	Long: `Exibe detalhes sobre skills instaladas, ferramentas detectadas, e estado do manifesto.

Exemplos:
  ai-spec-harness inspect ./meu-projeto
  ai-spec-harness inspect . -v`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()
		mfst := manifest.NewStore(fsys)
		det := detect.NewFileDetector(fsys)

		svc := inspect.NewService(fsys, printer, mfst, det)
		return svc.Execute(args[0])
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
