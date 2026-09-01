package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
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
	return command
}
