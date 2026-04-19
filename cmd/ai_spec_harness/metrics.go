package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/metrics"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/spf13/cobra"
)

var metricsCmd = &cobra.Command{
	Use:   "metrics [path]",
	Short: "Reporta metricas de contexto para governanca",
	Long: `Calcula e exibe metricas de tokens estimados por baseline, fluxo
e perfil de carga de skills e referencias.

Exemplos:
  ai-spec-harness metrics
  ai-spec-harness metrics ~/ai-spec --format json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootDir := "."
		if len(args) > 0 {
			rootDir = args[0]
		}
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()
		svc := metrics.NewService(fsys, printer)
		return svc.Execute(rootDir, metricsFormat)
	},
}

var metricsFormat string

func init() {
	metricsCmd.Flags().StringVar(&metricsFormat, "format", "table", "Formato de saida: table ou json")
	rootCmd.AddCommand(metricsCmd)
}
