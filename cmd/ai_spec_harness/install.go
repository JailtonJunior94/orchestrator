package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/gitref"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/spf13/cobra"
)

type installCommand struct{}

func newInstallCmd() *cobra.Command {
	handler := &installCommand{}
	cmd := &cobra.Command{
		Use:   "install <path>",
		Short: "Instala governanca de IA em um projeto",
		Long: `Instala o pacote de governanca para ferramentas de IA em um projeto alvo.

Sem --source, usa as skills canonicas embutidas no binario.

Exemplos:
  ai-spec-harness install ./meu-projeto --tools claude,gemini --langs go,python
  ai-spec-harness install ./meu-projeto --tools all --langs all --mode copy
  ai-spec-harness install ./meu-projeto --tools claude --dry-run
  ai-spec-harness install ./meu-projeto --tools all --langs all --source ~/ai-governance
  ai-spec-harness install ./meu-projeto --tools all --ref v1.0.0`,
		Args: cobra.ExactArgs(1),
		RunE: handler.run,
	}

	cmd.Flags().String("tools", "", "Ferramentas para instalar: claude,gemini,codex,copilot ou all (opcional; detecta automaticamente se omitido)")
	cmd.Flags().String("langs", "", "Linguagens: go,node,python ou all")
	cmd.Flags().String("mode", "symlink", "Modo de instalacao: symlink ou copy")
	cmd.Flags().Bool("dry-run", false, "Mostra o que seria criado sem executar")
	cmd.Flags().String("source", "", "Diretorio fonte do repositorio de governanca (opcional; usa embutido se omitido)")
	cmd.Flags().String("ref", "", "Referencia git (tag, branch, SHA) para usar como fonte; forca --mode copy (mutualmente exclusivo com --source)")
	cmd.Flags().Bool("no-context", false, "Desabilita geracao de governanca contextual")
	cmd.Flags().String("codex-profile", "full", "Perfil de skills para Codex: full ou lean")
	cmd.Flags().String("focus-paths", "", "Prioriza deteccao de toolchain proximo desses arquivos, separados por virgula (util em monorepos). Alternativa: env FOCUS_PATHS")
	cmd.Flags().Bool("global", false, "Instala globalmente em ~/.aispec (ADR-019 RF-07)")
	cmd.Flags().Bool("follow-external-symlinks", false, "Permite escrever em .agents/skills quando symlink aponta para fora do projeto")
	return cmd
}

func (c *installCommand) run(cmd *cobra.Command, args []string) error {
	projectDir := args[0]
	installTools, _ := cmd.Flags().GetString("tools")
	installLangs, _ := cmd.Flags().GetString("langs")
	installMode, _ := cmd.Flags().GetString("mode")
	installDryRun, _ := cmd.Flags().GetBool("dry-run")
	installSource, _ := cmd.Flags().GetString("source")
	installRef, _ := cmd.Flags().GetString("ref")
	installNoCtx, _ := cmd.Flags().GetBool("no-context")
	installCodexProfile, _ := cmd.Flags().GetString("codex-profile")
	installFocusPaths, _ := cmd.Flags().GetString("focus-paths")
	installGlobal, _ := cmd.Flags().GetBool("global")
	installFollowExternalSymlinks, _ := cmd.Flags().GetBool("follow-external-symlinks")

	if installRef != "" && installSource != "" {
		return fmt.Errorf("--ref e --source sao mutuamente exclusivos")
	}

	scope := config.ScopeProject
	if installGlobal {
		scope = config.ScopeGlobal
	}

	// --tools e opcional: vazio => auto-detect via AgentDetector (ADR-019 + RF-06).
	// Quando presente, e override explicito (precede deteccao automatica).
	tools, err := newFlagHelper().parseToolsFlag(installTools)
	if err != nil {
		return err
	}

	langs, err := newFlagHelper().parseLangsFlag(installLangs)
	if err != nil {
		return err
	}

	linkMode, ok := skills.NewCatalog().ParseLinkMode(installMode)
	if !ok {
		return fmt.Errorf("modo invalido: %s (use symlink ou copy)", installMode)
	}

	printer := output.New(newCommandEnv().verbose(cmd))

	sourceDir := installSource
	if installRef != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("obtendo diretorio atual: %w", err)
		}
		resolved, err := gitref.NewResolver().Resolve(cwd, installRef)
		if err != nil {
			return err
		}
		defer resolved.Cleanup()
		sourceDir = resolved.Dir
		if linkMode != skills.LinkCopy && cmd.Flags().Changed("mode") {
			printer.Warn("--ref ignora --mode %s: forcando copy", installMode)
		}
		linkMode = skills.LinkCopy
	}

	fsys := fs.NewOSFileSystem()
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)

	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	return svc.Execute(config.InstallOptions{
		ProjectDir:             projectDir,
		SourceDir:              sourceDir,
		Tools:                  tools,
		Langs:                  langs,
		LinkMode:               linkMode,
		DryRun:                 installDryRun,
		GenerateCtx:            !installNoCtx,
		CodexProfile:           installCodexProfile,
		FocusPaths:             newFlagHelper().parseFocusPaths(installFocusPaths),
		Scope:                  scope,
		FollowExternalSymlinks: installFollowExternalSymlinks,
	})
}
