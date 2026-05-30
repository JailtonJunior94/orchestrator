package aispecharness

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var _semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func newUpdateVersionCmd() *cobra.Command {
	var updateVersionVersion string
	updateVersionVersionFile := "VERSION"

	cmd := &cobra.Command{
		Use:   "update-version",
		Short: "Atualiza o arquivo VERSION com validacao semver",
		Long: `Atualiza o arquivo VERSION com a versao especificada, validando o formato semver.

Exemplos:
  ai-spec update-version --version 1.2.3
  ai-spec update-version --version 1.2.3 --version-file path/to/VERSION`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := newFlagHelper().requireFlag(cmd, "version", "ai-spec update-version --version 1.2.3"); err != nil {
				return err
			}

			v := strings.TrimSpace(updateVersionVersion)
			if !_semverPattern.MatchString(v) {
				return fmt.Errorf("formato de versao invalido %q: esperado MAJOR.MINOR.PATCH sem prefixo 'v'", v)
			}

			if err := os.WriteFile(updateVersionVersionFile, []byte(v+"\n"), 0644); err != nil {
				return fmt.Errorf("escrevendo arquivo de versao: %w", err)
			}

			fmt.Printf("VERSION atualizado para %s\n", v)
			return nil
		},
	}

	cmd.Flags().StringVar(&updateVersionVersion, "version", "", "Versao SemVer sem prefixo v (obrigatorio, ex: 1.2.3)")
	cmd.Flags().StringVar(&updateVersionVersionFile, "version-file", "VERSION", "Caminho para o arquivo de versao")
	return cmd
}
