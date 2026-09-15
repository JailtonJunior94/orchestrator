package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/doctor"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	gitpkg "github.com/JailtonJunior94/ai-spec-harness/internal/git"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor <path>",
		Short: "Diagnostica problemas na instalacao de governanca",
		Long: `Executa verificacoes de saude: repositorio git, symlinks, permissoes, manifesto.

Exemplos:
  ai-spec-harness doctor ./meu-projeto
  ai-spec-harness doctor . -v`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printer := output.New(newCommandEnv().verbose(cmd))
			fsys := fs.NewOSFileSystem()
			mfst := manifest.NewStore(fsys)
			gitRepo := gitpkg.NewCLIRepository()

			checkCodexTrust, _ := cmd.Flags().GetBool("codex-trust")

			svc := doctor.NewService(fsys, printer, mfst, gitRepo)
			return svc.ExecuteWithOptions(args[0], checkCodexTrust)
		},
	}
	cmd.Flags().Bool("codex-trust", false, "Verifica trust de hooks do Codex via RPC read-only hooks/list do codex app-server (executa binario, opt-in explicito)")
	return cmd
}
