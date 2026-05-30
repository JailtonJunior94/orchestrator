package aispecharness

import (
	"errors"
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillbump"
	"github.com/spf13/cobra"
)

type skillBumpCommand struct{}

func newSkillBumpCmd() *cobra.Command {
	handler := &skillBumpCommand{}
	var skillBumpDryRun bool

	cmd := &cobra.Command{
		Use:   "skill-bump <path>",
		Short: "Atualiza versao de skills cujo conteudo mudou desde a ultima tag",
		Long: `Compara skills entre a ultima tag e HEAD e atualiza os campos version no frontmatter.

Exemplos:
  ai-spec-harness skill-bump .
  ai-spec-harness skill-bump . --dry-run
  ai-spec-harness skill-bump ./meu-projeto`,
		Args: cobra.ExactArgs(1),
		RunE: handler.run,
	}

	cmd.Flags().BoolVar(&skillBumpDryRun, "dry-run", false, "Exibe as mudancas sem alterar arquivos")
	return cmd
}

func (c *skillBumpCommand) run(cmd *cobra.Command, args []string) error {
	repoPath := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	printer := output.New(newCommandEnv().verbose(cmd))
	fsys := fs.NewOSFileSystem()
	svc := skillbump.NewService(fsys, printer)

	results, err := svc.Execute(repoPath, ".agents/skills", dryRun)
	if err != nil {
		if errors.Is(err, skillbump.ErrNoChanges) {
			fmt.Fprintln(cmd.OutOrStdout(), "nenhuma skill com mudanca detectada")
			return nil
		}
		if errors.Is(err, skillbump.ErrNoTagFound) {
			return fmt.Errorf("nenhuma tag v* encontrada: execute skill-bump apos a primeira release")
		}
		return err
	}

	for _, r := range results {
		if dryRun {
			fmt.Fprintf(cmd.OutOrStdout(), "[dry-run] %s: %s -> %s (%s)\n", r.SkillName, r.PreviousVersion, r.NewVersion, r.Reason)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s -> %s\n", r.SkillName, r.PreviousVersion, r.NewVersion)
		}
	}

	return nil
}
