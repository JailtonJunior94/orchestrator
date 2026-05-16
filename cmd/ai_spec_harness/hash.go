package aispecharness

import (
	"crypto/sha256"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var hashCmd = &cobra.Command{
	Use:   "hash <file>",
	Short: "Calcula SHA-256 de um arquivo de forma portavel",
	Long: `Calcula o SHA-256 de um arquivo usando a implementacao Go embarcada no
ai-spec-harness, sem depender de binarios externos como sha256sum ou shasum.

Exemplos:
  ai-spec hash tasks/prd-exemplo/prd.md
  ai-spec hash ./techspec.md`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		hash, err := hashFile(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), hash)
		return nil
	},
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("calcular hash de %s: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

func init() {
	rootCmd.AddCommand(hashCmd)
}
