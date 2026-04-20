package aispecharness

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	updateVersionVersion     string
	updateVersionVersionFile string
)

var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

var updateVersionCmd = &cobra.Command{
	Use:   "update-version",
	Short: "Atualiza o arquivo VERSION com validacao semver",
	Long: `Atualiza o arquivo VERSION com a versao especificada, validando o formato semver.

Exemplos:
  ai-spec update-version --version 1.2.3
  ai-spec update-version --version 1.2.3 --version-file path/to/VERSION`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireFlag(cmd, "version", "ai-spec update-version --version 1.2.3"); err != nil {
			return err
		}

		v := strings.TrimSpace(updateVersionVersion)
		if !semverPattern.MatchString(v) {
			return fmt.Errorf("formato de versao invalido %q: esperado MAJOR.MINOR.PATCH sem prefixo 'v'", v)
		}

		if err := os.WriteFile(updateVersionVersionFile, []byte(v+"\n"), 0644); err != nil {
			return fmt.Errorf("escrevendo arquivo de versao: %w", err)
		}

		fmt.Printf("VERSION atualizado para %s\n", v)
		return nil
	},
}

func init() {
	updateVersionCmd.Flags().StringVar(&updateVersionVersion, "version", "", "Versao SemVer sem prefixo v (obrigatorio, ex: 1.2.3)")
	updateVersionCmd.Flags().StringVar(&updateVersionVersionFile, "version-file", "VERSION", "Caminho para o arquivo de versao")
	rootCmd.AddCommand(updateVersionCmd)
}
