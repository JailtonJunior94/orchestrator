package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/upgrade"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <path>",
	Short: "Atualiza skills de governanca em um projeto",
	Long: `Verifica ou atualiza skills de governanca comparando versoes e checksums.

Exemplos:
  ai-spec-harness upgrade ./meu-projeto --source ~/ai-spec
  ai-spec-harness upgrade ./meu-projeto --source ~/ai-spec --check
  ai-spec-harness upgrade ./meu-projeto --source ~/ai-spec --langs go,node`,
	Args: cobra.ExactArgs(1),
	RunE: runUpgrade,
}

var (
	upgradeCheckOnly bool
	upgradeLangs     string
	upgradeSource    string
)

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Apenas verifica sem alterar arquivos")
	upgradeCmd.Flags().StringVar(&upgradeLangs, "langs", "", "Filtrar por linguagens: go,node,python")
	upgradeCmd.Flags().StringVar(&upgradeSource, "source", "", "Diretorio fonte do repositorio de governanca (obrigatorio)")

	_ = upgradeCmd.MarkFlagRequired("source")

	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	langs, err := parseLangsFlag(upgradeLangs)
	if err != nil {
		return err
	}

	printer := output.New(verbose)
	fsys := fs.NewOSFileSystem()
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)

	svc := upgrade.NewService(fsys, printer, mfst, adpt, ctxg)

	return svc.Execute(config.UpgradeOptions{
		ProjectDir: args[0],
		SourceDir:  upgradeSource,
		CheckOnly:  upgradeCheckOnly,
		Langs:      langs,
	})
}
