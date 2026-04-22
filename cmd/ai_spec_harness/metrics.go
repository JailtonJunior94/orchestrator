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

Por default usa a heuristica chars/3.5 (sem dependencia externa).
Com --precise, usa tiktoken cl100k_base (~15% mais preciso, requer download inicial do modelo BPE).

Exemplos:
  ai-spec-harness metrics
  ai-spec-harness metrics ~/ai-spec --format json
  ai-spec-harness metrics --precise`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootDir := "."
		if len(args) > 0 {
			rootDir = args[0]
		}
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()

		var tok metrics.Tokenizer
		if metricsPrecise {
			var usingTiktoken bool
			tok, usingTiktoken = metrics.NewPreciseTokenizer()
			if usingTiktoken {
				printer.Info("Tokenizer: tiktoken/cl100k_base (preciso)")
			} else {
				printer.Info("Tokenizer: chars/3.5 (fallback — tiktoken nao disponivel)")
			}
		} else {
			tok = metrics.NewCharEstimator()
		}

		svc := metrics.NewService(fsys, printer, tok)
		return svc.Execute(rootDir, metricsFormat, metricsBrief)
	},
}

var metricsFormat string
var metricsPrecise bool
var metricsBrief bool

func init() {
	metricsCmd.Flags().StringVar(&metricsFormat, "format", "table", "Formato de saida: table ou json")
	metricsCmd.Flags().BoolVar(&metricsPrecise, "precise", false, "Usa tiktoken cl100k_base para contagem precisa de tokens (~15% mais preciso)")
	metricsCmd.Flags().BoolVar(&metricsBrief, "brief", false, "Estima economia com modo brief (apenas TL;DR de referencias)")
	rootCmd.AddCommand(metricsCmd)
}
