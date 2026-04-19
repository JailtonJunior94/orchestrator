package aispecharness

import (
	"fmt"
	"os"
	"strings"

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

var installCmd = &cobra.Command{
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
	RunE: runInstall,
}

var (
	installTools        string
	installLangs        string
	installMode         string
	installDryRun       bool
	installSource       string
	installRef          string
	installNoCtx        bool
	installCodexProfile string
	installFocusPaths   string
)

func init() {
	installCmd.Flags().StringVar(&installTools, "tools", "", "Ferramentas para instalar: claude,gemini,codex,copilot ou all (obrigatorio)")
	installCmd.Flags().StringVar(&installLangs, "langs", "", "Linguagens: go,node,python ou all")
	installCmd.Flags().StringVar(&installMode, "mode", "symlink", "Modo de instalacao: symlink ou copy")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Mostra o que seria criado sem executar")
	installCmd.Flags().StringVar(&installSource, "source", "", "Diretorio fonte do repositorio de governanca (opcional; usa embutido se omitido)")
	installCmd.Flags().StringVar(&installRef, "ref", "", "Referencia git (tag, branch, SHA) para usar como fonte; forca --mode copy (mutualmente exclusivo com --source)")
	installCmd.Flags().BoolVar(&installNoCtx, "no-context", false, "Desabilita geracao de governanca contextual")
	installCmd.Flags().StringVar(&installCodexProfile, "codex-profile", "full", "Perfil de skills para Codex: full ou lean")
	installCmd.Flags().StringVar(&installFocusPaths, "focus-paths", "", "Prioriza deteccao de toolchain proximo desses arquivos, separados por virgula (util em monorepos). Alternativa: env FOCUS_PATHS")

	_ = installCmd.MarkFlagRequired("tools")

	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	projectDir := args[0]

	if installRef != "" && installSource != "" {
		return fmt.Errorf("--ref e --source sao mutuamente exclusivos")
	}

	tools, err := parseToolsFlag(installTools)
	if err != nil {
		return err
	}

	langs, err := parseLangsFlag(installLangs)
	if err != nil {
		return err
	}

	linkMode, ok := skills.ParseLinkMode(installMode)
	if !ok {
		return fmt.Errorf("modo invalido: %s (use symlink ou copy)", installMode)
	}

	printer := output.New(verbose)

	sourceDir := installSource
	if installRef != "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("obtendo diretorio atual: %w", err)
		}
		resolved, err := gitref.Resolve(cwd, installRef)
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
		ProjectDir:   projectDir,
		SourceDir:    sourceDir,
		Tools:        tools,
		Langs:        langs,
		LinkMode:     linkMode,
		DryRun:       installDryRun,
		GenerateCtx:  !installNoCtx,
		CodexProfile: installCodexProfile,
		FocusPaths:   parseFocusPaths(installFocusPaths),
	})
}

func parseToolsFlag(raw string) ([]skills.Tool, error) {
	if raw == "" {
		return nil, fmt.Errorf("flag --tools e obrigatoria")
	}
	if raw == "all" {
		return skills.AllTools, nil
	}
	var tools []skills.Tool
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		t, ok := skills.ParseTool(s)
		if !ok {
			return nil, fmt.Errorf("ferramenta invalida: %s (opcoes: claude, gemini, codex, copilot, all)", s)
		}
		tools = append(tools, t)
	}
	return tools, nil
}

// parseFocusPaths converte a flag --focus-paths (comma-separated) ou a env var
// FOCUS_PATHS (newline ou comma-separated) em uma slice de caminhos.
func parseFocusPaths(raw string) []string {
	if raw == "" {
		raw = os.Getenv("FOCUS_PATHS")
	}
	if raw == "" {
		return nil
	}
	var paths []string
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

func parseLangsFlag(raw string) ([]skills.Lang, error) {
	if raw == "" || raw == "none" {
		return nil, nil
	}
	if raw == "all" {
		return skills.AllLangs, nil
	}
	var langs []skills.Lang
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		l, ok := skills.ParseLang(s)
		if !ok {
			return nil, fmt.Errorf("linguagem invalida: %s (opcoes: go, node, python, all)", s)
		}
		langs = append(langs, l)
	}
	return langs, nil
}
