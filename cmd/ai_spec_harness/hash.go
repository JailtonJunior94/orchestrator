package aispecharness

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JailtonJunior94/ai-spec-harness/internal/specdigest"
)

type hashCommand struct{}

func newHashCmd() *cobra.Command {
	handler := &hashCommand{}
	cmd := &cobra.Command{
		Use:   "hash <file>",
		Short: "Calcula SHA-256 de um arquivo de forma portavel",
		Long: `Calcula o SHA-256 de um arquivo usando a implementacao Go embarcada no
ai-spec-harness, sem depender de binarios externos como sha256sum ou shasum.

Exemplos:
  ai-spec hash .specs/prd-exemplo/prd.md
  ai-spec hash ./techspec.md`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hash, err := handler.hashFile(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), hash)
			return nil
		},
	}
	return cmd
}

func (c *hashCommand) hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("calcular hash de %s: %w", path, err)
	}
	return specdigest.Canonical(data), nil
}
