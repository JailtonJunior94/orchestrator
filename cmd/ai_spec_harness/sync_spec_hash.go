package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/specdrift"
	"github.com/spf13/cobra"
)

var syncSpecHashCmd = &cobra.Command{
	Use:   "sync-spec-hash <tasks.md>",
	Short: "Sincroniza os hashes de spec em tasks.md com os arquivos atuais de prd.md e techspec.md",
	Long: `Recalcula os SHA-256 de prd.md e techspec.md e atualiza (ou insere) os
comentarios <!-- spec-hash-prd: ... --> e <!-- spec-hash-techspec: ... --> em tasks.md.

Use este comando sempre que editar prd.md ou techspec.md para manter os hashes
registrados em tasks.md sincronizados e evitar que o check-spec-drift falhe.

Exemplos:
  ai-spec sync-spec-hash docs/tasks.md
  ai-spec sync-spec-hash ./tasks.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		tasksPath := args[0]

		if _, err := os.Stat(tasksPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "erro: arquivo nao encontrado: %s\n", tasksPath)
			os.Exit(2)
		}

		if err := specdrift.SyncSpecHash(tasksPath); err != nil {
			fmt.Fprintf(os.Stderr, "erro: %s\n", err)
			os.Exit(2)
		}

		fmt.Printf("OK: hashes sincronizados em %s\n", tasksPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncSpecHashCmd)
}
