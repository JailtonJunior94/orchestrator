package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/harness"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hookpolicy"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/spf13/cobra"
)

func newGitScopeCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "git-scope [path]",
		Short: "Gera ou verifica o escopo de operacoes git derivado do contrato do harness",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			filesystem := fs.NewOSFileSystem()
			loader := harness.NewDefaultLoader(filesystem)
			printer := output.New(newCommandEnv().verbose(cmd))
			service := hookpolicy.NewArtifactService(filesystem, loader, printer)
			if check {
				return service.Check(root)
			}
			_, err := service.Generate(root)
			return err
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Falha se o escopo git gerado divergir do contrato do harness")
	return cmd
}
