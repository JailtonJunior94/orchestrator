package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/lint"
	"github.com/spf13/cobra"
)

var lintCmd = &cobra.Command{
	Use:   "lint [path]",
	Short: "Verifica governança em arquivos gerados",
	Long: `Detecta problemas de governança no projeto:
  - Placeholders não renderizados {{ em AGENTS.md, CLAUDE.md, GEMINI.md, .codex/config.toml, copilot-instructions.md
  - Versão de governance-schema em AGENTS.md divergente da versão atual do CLI
  - bug-schema.json inválido
  - SKILL.md com frontmatter inválido

Exemplos:
  ai-spec-harness lint .
  ai-spec-harness lint ./meu-projeto`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir := "."
		if len(args) > 0 {
			projectDir = args[0]
		}

		svc := lint.NewService()
		errs, err := svc.Execute(projectDir)
		if err != nil {
			return err
		}

		if len(errs) == 0 {
			n := svc.CountChecks(projectDir)
			fmt.Printf("Lint aprovado: %d verificacoes passaram\n", n)
			return nil
		}

		for _, e := range errs {
			fmt.Println(e.String())
		}
		fmt.Printf("%d erro(s) encontrado(s)\n", len(errs))
		os.Exit(1)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lintCmd)
}
