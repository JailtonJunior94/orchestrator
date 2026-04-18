package aispecharness

import (
	"fmt"
	"strings"

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

var installCmd = &cobra.Command{
	Use:   "install <path>",
	Short: "Instala governanca de IA em um projeto",
	Long: `Instala o pacote de governanca para ferramentas de IA em um projeto alvo.

Exemplos:
  ai-spec-harness install ./meu-projeto --tools claude,gemini --langs go,python
  ai-spec-harness install ./meu-projeto --tools all --langs all --mode copy
  ai-spec-harness install ./meu-projeto --tools claude --dry-run`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

var (
	installTools  string
	installLangs  string
	installMode   string
	installDryRun bool
	installSource string
	installNoCtx  bool
)

func init() {
	installCmd.Flags().StringVar(&installTools, "tools", "", "Ferramentas para instalar: claude,gemini,codex,copilot ou all (obrigatorio)")
	installCmd.Flags().StringVar(&installLangs, "langs", "", "Linguagens: go,node,python ou all")
	installCmd.Flags().StringVar(&installMode, "mode", "symlink", "Modo de instalacao: symlink ou copy")
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Mostra o que seria criado sem executar")
	installCmd.Flags().StringVar(&installSource, "source", "", "Diretorio fonte do repositorio de governanca (obrigatorio)")
	installCmd.Flags().BoolVar(&installNoCtx, "no-context", false, "Desabilita geracao de governanca contextual")

	_ = installCmd.MarkFlagRequired("tools")
	_ = installCmd.MarkFlagRequired("source")

	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	projectDir := args[0]

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
	fsys := fs.NewOSFileSystem()
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)

	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	return svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   installSource,
		Tools:       tools,
		Langs:       langs,
		LinkMode:    linkMode,
		DryRun:      installDryRun,
		GenerateCtx: !installNoCtx,
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
