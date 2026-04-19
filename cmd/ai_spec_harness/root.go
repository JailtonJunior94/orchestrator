package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/invocation"
	"github.com/spf13/cobra"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "ai-spec-harness",
	Short: "Ferramenta CLI para governanca de IA em projetos de software",
	Long: `ai-spec-harness instala, inspeciona e atualiza pacotes de governanca para ferramentas
de IA (Claude, Gemini, Codex, Copilot) em repositorios de software.

Exemplos:
  ai-spec-harness install ./meu-projeto --tools claude,gemini --langs go,python
  ai-spec-harness upgrade ./meu-projeto
  ai-spec-harness inspect ./meu-projeto
  ai-spec-harness doctor ./meu-projeto`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := invocation.CheckDepth(); err != nil {
			return err
		}
		invocation.IncrementDepth()
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Exibir logs detalhados")
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
