package aispecharness

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/specdrift"
	"github.com/spf13/cobra"
)

var checkSpecDriftCmd = &cobra.Command{
	Use:   "check-spec-drift <tasks.md>",
	Short: "Verifica cobertura de requisitos e drift de spec em relacao a tasks.md",
	Long: `Verifica se os IDs de requisitos (RF-nn, REQ-nn) definidos em prd.md e/ou
techspec.md estao cobertos em tasks.md, e se os hashes dos arquivos de spec
coincidem com os registrados em tasks.md.

Exemplos:
  ai-spec check-spec-drift docs/tasks.md
  ai-spec check-spec-drift ./tasks.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tasksPath := args[0]

		if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "erro: arquivo nao encontrado: %s\n", tasksPath)
			os.Exit(2)
		}

		dir := filepath.Dir(tasksPath)

		report, err := specdrift.CheckDrift(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro: %s\n", err)
			os.Exit(2)
		}

		for _, cov := range report.Coverage {
			if len(cov.MissingIDs) > 0 {
				fmt.Printf("DRIFT: %s → %s: IDs faltantes: %s\n",
					cov.SourceFile, cov.TargetFile, strings.Join(cov.MissingIDs, ", "))
			}
		}

		for _, h := range report.Hashes {
			if !h.Match {
				if h.NoHashFound {
					fmt.Printf("AVISO: hash de %s nao encontrado em tasks.md\n", h.File)
				} else {
					fmt.Printf("DRIFT: hash de %s divergente (esperado: %s, atual: %s)\n",
						h.File, h.ExpectedHash, h.ActualHash)
				}
			}
		}

		if report.Pass {
			fmt.Println("OK: sem drift detectado.")
			return nil
		}

		os.Exit(1)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkSpecDriftCmd)
}
