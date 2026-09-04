package aispecharness

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
	"github.com/spf13/cobra"
)

func newValidateResultCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "validate-result <execution|review|checkpoint> <arquivo-json>",
		Short: "Valida um resultado SDD JSON versionado",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("ler resultado SDD: %w", err)
			}
			expectedTaskID, _ := cmd.Flags().GetString("task-id")
			verifyPhysical, _ := cmd.Flags().GetBool("verify-physical")
			prdDir, _ := cmd.Flags().GetString("prd-dir")
			exclusions, _ := cmd.Flags().GetStringSlice("exclude")
			validator := sdd.NewResultValidator()
			switch args[0] {
			case "execution":
				result, err := validator.ValidateExecutionJSON(content)
				if err != nil {
					return err
				}
				if expectedTaskID != "" && result.TaskID != expectedTaskID {
					return fmt.Errorf("task_id do resultado %q diverge do esperado %q", result.TaskID, expectedTaskID)
				}
				if verifyPhysical {
					if prdDir == "" {
						return fmt.Errorf("--prd-dir e obrigatorio com --verify-physical")
					}
					resultPath, pathErr := filepath.Abs(args[1])
					if pathErr != nil {
						return fmt.Errorf("resolver arquivo de resultado: %w", pathErr)
					}
					exclusions = append(exclusions, resultPath)
					if err := taskloop.NewOrchestrator(sdd.NewStore()).ValidateExecutionEvidence(prdDir, result, exclusions...); err != nil {
						return fmt.Errorf("validar provas fisicas do resultado: %w", err)
					}
				}
			case "checkpoint":
				result, err := validator.ValidateCheckpointJSON(content)
				if err != nil {
					return err
				}
				if expectedTaskID != "" && result.TaskID != expectedTaskID {
					return fmt.Errorf("task_id do checkpoint %q diverge do esperado %q", result.TaskID, expectedTaskID)
				}
			case "review":
				result, err := validator.ValidateReviewJSON(content)
				if err != nil {
					return err
				}
				if expectedTaskID != "" && result.TaskID != expectedTaskID {
					return fmt.Errorf("task_id da revisao %q diverge do esperado %q", result.TaskID, expectedTaskID)
				}
			default:
				return fmt.Errorf("tipo de resultado invalido %q: use execution, review ou checkpoint", args[0])
			}
			fmt.Printf("OK: resultado SDD %s valido\n", args[0])
			return nil
		},
	}
	command.Flags().String("task-id", "", "identificador da tarefa esperado")
	command.Flags().Bool("verify-physical", false, "recompoe e valida patch, estado final e evidencias fisicas")
	command.Flags().String("prd-dir", "", "diretorio do PRD usado na recomposicao fisica")
	command.Flags().StringSlice("exclude", nil, "envelope operacional adicional excluido do patch canonico")
	return command
}
