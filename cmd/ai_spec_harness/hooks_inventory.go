package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hookinventory"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/spf13/cobra"
)

type hooksInventoryCommand struct{}

func newHooksCmd() *cobra.Command {
	command := &hooksInventoryCommand{}
	cmd := &cobra.Command{Use: "hooks", Short: "Inspeciona hooks de governanca"}
	cmd.AddCommand(command.newInventoryCmd())
	cmd.AddCommand(newGitScopeCmd())
	return cmd
}

func (c *hooksInventoryCommand) newInventoryCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "inventory [path]",
		Short: "Gera ou verifica o inventario de hooks",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root := "."
			if len(args) == 1 {
				root = args[0]
			}
			service := hookinventory.NewService(fs.NewOSFileSystem(), output.New(newCommandEnv().verbose(cmd)))
			if check {
				return service.Check(root)
			}
			_, err := service.Generate(root)
			return err
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "Falha se o inventario estiver divergente ou incompleto")
	return cmd
}
