package aispecharness

import (
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
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
  ai-spec-harness inspect ./monorepo --focus-paths services/go-api/handler.go
  ai-spec-harness inspect . --brief
  ai-spec-harness inspect . --complexity=standard`,
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

		// Modo contexto: exibe referencias carregadas por nivel de complexidade
		if inspectComplexity != "" || inspectBrief {
			level := contextgen.ComplexityLevel(inspectComplexity)
			if level == "" {
				level = contextgen.ComplexityComplex
			}
			opts := contextgen.LoadOptions{Brief: inspectBrief, Complexity: level}
			loader := contextgen.NewLoader(fsys)
			skillsDir := args[0] + "/.agents/skills"
			if entries, err := fsys.ReadDir(skillsDir); err == nil {
				printer.Info("")
				printer.Info("Referencias por skill (complexity=%s, brief=%v):", level, inspectBrief)
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					refs, err := loader.LoadSkillReferences(skillsDir+"/"+e.Name(), opts)
					if err != nil {
						continue
					}
					printer.Info("  %s: %d referencias carregadas", e.Name(), len(refs))
				}
			}
		}

		return nil
	},
}

var inspectFocusPaths string
var inspectBrief bool
var inspectComplexity string

func init() {
	inspectCmd.Flags().StringVar(&inspectFocusPaths, "focus-paths", "", "Prioriza deteccao de toolchain proximo desses arquivos, separados por virgula (util em monorepos). Alternativa: env FOCUS_PATHS")
	inspectCmd.Flags().BoolVar(&inspectBrief, "brief", false, "Carrega apenas blocos TL;DR das referencias (economiza tokens)")
	inspectCmd.Flags().StringVar(&inspectComplexity, "complexity", "", "Filtra referencias por nivel de complexidade: trivial, standard, complex (default: complex)")
	rootCmd.AddCommand(inspectCmd)
}
