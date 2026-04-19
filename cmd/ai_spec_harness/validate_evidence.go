package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/evidence"
	"github.com/spf13/cobra"
)

var validateEvidenceRFIDs []string

var validateEvidenceCmd = &cobra.Command{
	Use:   "validate-evidence <tipo> <arquivo.md>",
	Short: "Valida um relatorio de evidencia Markdown",
	Long: `Valida um relatorio de evidencia Markdown conforme o tipo informado.

Tipos validos: task, bugfix, refactor

Exemplos:
  ai-spec validate-evidence task relatorio-execucao.md
  ai-spec validate-evidence bugfix bugfix-report.md --rf RF-01 --rf RF-02
  ai-spec validate-evidence refactor refactor-report.md`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		kindArg := args[0]
		filePath := args[1]

		var kind evidence.ReportKind
		switch kindArg {
		case "task":
			kind = evidence.KindTask
		case "bugfix":
			kind = evidence.KindBugfix
		case "refactor":
			kind = evidence.KindRefactor
		default:
			fmt.Fprintf(os.Stderr, "tipo invalido: %q. Use: task, bugfix, refactor\n", kindArg)
			_ = cmd.Usage()
			os.Exit(2)
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "erro ao ler arquivo: %v\n", err)
			os.Exit(2)
		}

		result := evidence.Validate(content, kind, validateEvidenceRFIDs)

		for _, f := range result.Findings {
			fmt.Printf("FALTANDO: %s\n", f.Label)
		}

		if result.Pass {
			fmt.Println("Relatorio APROVADO.")
			return nil
		}

		fmt.Printf("%d problemas encontrados. Relatorio REPROVADO.\n", len(result.Findings))
		os.Exit(1)
		return nil
	},
}

func init() {
	validateEvidenceCmd.Flags().StringArrayVar(&validateEvidenceRFIDs, "rf", nil, "ID de requisito para rastreabilidade (repetivel, ex: --rf RF-01 --rf RF-02)")
	rootCmd.AddCommand(validateEvidenceCmd)
}
