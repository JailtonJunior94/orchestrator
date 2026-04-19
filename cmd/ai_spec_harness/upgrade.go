package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/gitref"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/upgrade"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade <path>",
	Short: "Atualiza skills de governanca em um projeto",
	Long: `Verifica ou atualiza skills de governanca comparando versoes e checksums.

Sem --source, compara com as skills canonicas embutidas no binario.

Exemplos:
  ai-spec-harness upgrade ./meu-projeto
  ai-spec-harness upgrade ./meu-projeto --check
  ai-spec-harness upgrade ./meu-projeto --source ~/ai-governance
  ai-spec-harness upgrade ./meu-projeto --source ~/ai-governance --langs go,node
  ai-spec-harness upgrade ./meu-projeto --ref v1.1.0 --check
  ai-spec-harness upgrade ./meu-projeto --ref v1.1.0`,
	Args: cobra.ExactArgs(1),
	RunE: runUpgrade,
}

var (
	upgradeCheckOnly bool
	upgradeLangs     string
	upgradeSource    string
	upgradeRef       string
)

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeCheckOnly, "check", false, "Apenas verifica sem alterar arquivos")
	upgradeCmd.Flags().StringVar(&upgradeLangs, "langs", "", "Filtrar por linguagens: go,node,python")
	upgradeCmd.Flags().StringVar(&upgradeSource, "source", "", "Diretorio fonte do repositorio de governanca (opcional; usa embutido se omitido)")
	upgradeCmd.Flags().StringVar(&upgradeRef, "ref", "", "Referencia git (tag, branch, SHA) para usar como fonte (mutualmente exclusivo com --source)")

	rootCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	if upgradeRef != "" && upgradeSource != "" {
		return fmt.Errorf("--ref e --source sao mutuamente exclusivos")
	}

	langs, err := parseLangsFlag(upgradeLangs)
	if err != nil {
		return err
	}

	printer := output.New(verbose)

	sourceDir := upgradeSource
	if upgradeRef != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("obtendo diretorio atual: %w", err)
		}
		resolved, err := gitref.Resolve(cwd, upgradeRef)
		if err != nil {
			return err
		}
		defer resolved.Cleanup()
		sourceDir = resolved.Dir
	}

	fsys := fs.NewOSFileSystem()
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)

	svc := upgrade.NewService(fsys, printer, mfst, adpt, ctxg)

	return svc.Execute(config.UpgradeOptions{
		ProjectDir: args[0],
		SourceDir:  sourceDir,
		CheckOnly:  upgradeCheckOnly,
		Langs:      langs,
	})
}
