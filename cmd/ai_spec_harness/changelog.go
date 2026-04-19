package aispecharness

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/changelog"
	"github.com/spf13/cobra"
)

var (
	changelogVersion string
	changelogDryRun  bool
)

var changelogCmd = &cobra.Command{
	Use:   "changelog [path]",
	Short: "Gera ou atualiza o CHANGELOG.md a partir de conventional commits",
	Long: `Analisa os conventional commits desde o ultimo tag e gera uma secao
no CHANGELOG.md para a versao especificada.

Exemplos:
  ai-spec-harness changelog .
  ai-spec-harness changelog . --version 1.3.0
  ai-spec-harness changelog ./meu-repo --version 1.3.0 --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := "."
		if len(args) > 0 {
			repoPath = args[0]
		}

		date := time.Now().Format("2006-01-02")
		section, err := changelog.GenerateChangelog(repoPath, changelogVersion, date, "")
		if err != nil {
			return err
		}

		if changelogDryRun {
			fmt.Print(section)
			return nil
		}

		changelogPath := filepath.Join(repoPath, "CHANGELOG.md")
		if err := changelog.UpdateChangelog(changelogPath, section); err != nil {
			return fmt.Errorf("updating CHANGELOG.md: %w", err)
		}
		fmt.Printf("CHANGELOG.md atualizado com a versao %s\n", changelogVersion)
		return nil
	},
}

func init() {
	changelogCmd.Flags().StringVar(&changelogVersion, "version", "", "Versao SemVer sem prefixo v (obrigatorio)")
	changelogCmd.Flags().BoolVar(&changelogDryRun, "dry-run", false, "Imprime no stdout sem modificar o arquivo")
	_ = changelogCmd.MarkFlagRequired("version")
	rootCmd.AddCommand(changelogCmd)
}
