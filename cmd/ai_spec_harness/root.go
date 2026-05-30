package aispecharness

import (
	"errors"
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/invocation"
	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
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
			if err := invocation.NewGuard().CheckDepth(); err != nil {
				return err
			}
			invocation.NewGuard().IncrementDepth()
			return nil
		},
	}

	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Exibir logs detalhados")
	cmd.AddCommand(newChangelogCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newHashCmd())
	cmd.AddCommand(newInspectCmd())
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newLintCmd())
	cmd.AddCommand(newMetricsCmd())
	cmd.AddCommand(newPrerequisitesCmd())
	cmd.AddCommand(newScaffoldCmd())
	cmd.AddCommand(newSemverNextCmd())
	cmd.AddCommand(newSkillBumpCmd())
	cmd.AddCommand(newSkillsCmd())
	cmd.AddCommand(newCheckSpecDriftCmd())
	cmd.AddCommand(newSyncSpecHashCmd())
	cmd.AddCommand(newTaskLoopCmd())
	cmd.AddCommand(newTelemetryCmd())
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newUpdateVersionCmd())
	cmd.AddCommand(newUpgradeCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newValidateBugsCmd())
	cmd.AddCommand(newValidateEvidenceCmd())
	cmd.AddCommand(newVerifyCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newWrapperCmd())
	return cmd
}

type commandEnv struct{}

func newCommandEnv() *commandEnv {
	return &commandEnv{}
}

func (e *commandEnv) verbose(cmd *cobra.Command) bool {
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	return verbose
}

func Execute() error {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		var ee *exitError
		if !errors.As(err, &ee) {
			fmt.Fprintln(os.Stderr, err)
		}
		return err
	}
	return nil
}
