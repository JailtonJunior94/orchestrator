package aispecharness

import (
	"fmt"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify [path]",
	Short: "Verifica o estado de instalacao das skills (current/missing/drifted)",
	Long: `Verifica o estado de instalacao das skills de governanca em um projeto.

Reporta cada skill como current (ok), missing (ausente) ou drifted (divergente).
Reusa o comparador de checksum do modulo de upgrade (file-first: le hash do disco).

Sem --global, verifica o diretorio do projeto especificado (ou diretorio atual).
Com --global, verifica a instalacao global em ~/.aispec.

Sem --source, usa as skills canonicas embutidas no binario.

Exemplos:
  ai-spec-harness verify ./meu-projeto
  ai-spec-harness verify . --tools claude,gemini
  ai-spec-harness verify --global
  ai-spec-harness verify ./meu-projeto --source ~/ai-governance`,
	Args: cobra.MaximumNArgs(1),
	RunE: runVerify,
}

var (
	verifyTools  string
	verifyLangs  string
	verifySource string
	verifyGlobal bool
)

func init() {
	verifyCmd.Flags().StringVar(&verifyTools, "tools", "", "Ferramentas a verificar: claude,gemini,codex,copilot ou all (opcional)")
	verifyCmd.Flags().StringVar(&verifyLangs, "langs", "", "Linguagens: go,node,python ou all")
	verifyCmd.Flags().StringVar(&verifySource, "source", "", "Diretorio fonte do repositorio de governanca (opcional; usa embutido se omitido)")
	verifyCmd.Flags().BoolVar(&verifyGlobal, "global", false, "Verifica a instalacao global em ~/.aispec")

	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	projectDir := "."
	if len(args) > 0 {
		projectDir = args[0]
	}

	tools, err := parseToolsFlag(verifyTools)
	if err != nil {
		return err
	}

	langs, err := parseLangsFlag(verifyLangs)
	if err != nil {
		return err
	}

	scope := config.ScopeProject
	if verifyGlobal {
		scope = config.ScopeGlobal
	}

	printer := output.New(verbose)
	fsys := fs.NewOSFileSystem()
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)

	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: projectDir,
		SourceDir:  verifySource,
		Tools:      tools,
		Langs:      langs,
		Scope:      scope,
	})
	if err != nil {
		return fmt.Errorf("verificar instalacao: %w", err)
	}

	if len(items) == 0 {
		printer.Info("Nenhuma skill encontrada para verificar.")
		printer.Info("Dica: use --tools e --langs para especificar o que verificar.")
		return nil
	}

	// Contadores por estado.
	var nCurrent, nMissing, nDrifted int
	for _, item := range items {
		switch item.State {
		case install.VerifyStateCurrent:
			nCurrent++
		case install.VerifyStateMissing:
			nMissing++
		case install.VerifyStateDrifted:
			nDrifted++
		}
	}

	// Exibir resultados.
	printer.Info("Resultado da verificacao:")
	printer.Info("")

	// Agrupar por tool para exibicao mais clara.
	seen := make(map[skills.Tool]bool)
	for _, item := range items {
		if !seen[item.Tool] {
			seen[item.Tool] = true
			if item.Tool != "" {
				printer.Info("  [%s]", item.Tool)
			}
		}
		stateLabel := stateEmoji(item.State)
		printer.Info("    %-40s %s", item.Skill, stateLabel)
	}

	printer.Info("")
	printer.Info("Resumo: %d current, %d missing, %d drifted", nCurrent, nMissing, nDrifted)

	if nMissing > 0 || nDrifted > 0 {
		printer.Info("")
		printer.Info("Dica: execute 'ai-spec-harness install' para corrigir skills faltantes ou divergentes.")
		return fmt.Errorf("%d skill(s) nao current (missing: %d, drifted: %d)", nMissing+nDrifted, nMissing, nDrifted)
	}

	return nil
}

func stateEmoji(state install.VerifyState) string {
	switch state {
	case install.VerifyStateCurrent:
		return "current"
	case install.VerifyStateMissing:
		return "MISSING"
	case install.VerifyStateDrifted:
		return "DRIFTED"
	default:
		return string(state)
	}
}
