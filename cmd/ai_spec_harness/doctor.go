package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/doctor"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	gitpkg "github.com/JailtonJunior94/ai-spec-harness/internal/git"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor <path>",
	Short: "Diagnostica problemas na instalacao de governanca",
	Long: `Executa verificacoes de saude: repositorio git, symlinks, permissoes, manifesto.

Exemplos:
  ai-spec-harness doctor ./meu-projeto
  ai-spec-harness doctor . -v`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()
		mfst := manifest.NewStore(fsys)
		gitRepo := gitpkg.NewCLIRepository()

		svc := doctor.NewService(fsys, printer, mfst, gitRepo)
		return svc.Execute(args[0])
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
