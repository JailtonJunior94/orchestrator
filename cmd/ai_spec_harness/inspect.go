package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/inspect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect <path>",
	Short: "Inspeciona o estado da instalacao de governanca",
	Long: `Exibe detalhes sobre skills instaladas, ferramentas detectadas, e estado do manifesto.

Exemplos:
  ai-spec-harness inspect ./meu-projeto
  ai-spec-harness inspect . -v
  ai-spec-harness inspect ./monorepo --focus-paths services/go-api/handler.go`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		printer := output.New(verbose)
		fsys := fs.NewOSFileSystem()
		mfst := manifest.NewStore(fsys)
		det := detect.NewFileDetector(fsys)

		svc := inspect.NewService(fsys, printer, mfst, det)
		if err := svc.Execute(args[0]); err != nil {
			return err
		}

		// Deteccao de toolchain com suporte a focus-paths
		focusPaths := parseFocusPaths(inspectFocusPaths)
		tcDetector := detect.NewToolchainDetector(fsys)
		if len(focusPaths) > 0 {
			tcDetector.FocusPaths = focusPaths
		}

		toolchain := tcDetector.Detect(args[0])
		if len(toolchain) > 0 {
			if len(focusPaths) > 0 {
				printer.Info("Toolchain detectado (priorizado por --focus-paths %v):", focusPaths)
			} else {
				printer.Info("Toolchain detectado:")
			}
			for _, lang := range []string{"go", "node", "python", "unknown"} {
				entry, ok := toolchain[lang]
				if !ok {
					continue
				}
				printer.Info("  %s:", lang)
				if entry.Fmt != "" {
					printer.Info("    fmt:  %s", entry.Fmt)
				}
				if entry.Test != "" {
					printer.Info("    test: %s", entry.Test)
				}
				if entry.Lint != "" {
					printer.Info("    lint: %s", entry.Lint)
				}
			}
		}

		return nil
	},
}

var inspectFocusPaths string

func init() {
	inspectCmd.Flags().StringVar(&inspectFocusPaths, "focus-paths", "", "Prioriza deteccao de toolchain proximo desses arquivos, separados por virgula (util em monorepos). Alternativa: env FOCUS_PATHS")
	rootCmd.AddCommand(inspectCmd)
}
