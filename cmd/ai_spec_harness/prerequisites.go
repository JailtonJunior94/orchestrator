package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/prerequisites"
	"github.com/spf13/cobra"
)

var prerequisitesCmd = &cobra.Command{
	Use:   "prerequisites <skill-name> [path]",
	Short: "Verifica pre-condicoes de uma skill antes de executa-la",
	Long: `Verifica se os arquivos necessarios para uma skill estao presentes no projeto.

Exit codes:
  0 — todos os pre-requisitos satisfeitos
  1 — um ou mais pre-requisitos ausentes
  2 — uso incorreto (skill desconhecida ou argumentos invalidos)

Skills suportadas:
  go-implementation          requer go.mod ou go.work
  node-implementation        requer package.json
  python-implementation      requer pyproject.toml, setup.py ou requirements.txt
  create-tasks               requer prd.md e techspec.md
  execute-task               requer tasks.md
  create-technical-specification requer prd.md
  bugfix                     bugs.json (opcional, apenas aviso)

Exemplos:
  ai-spec prerequisites go-implementation ./meu-projeto
  ai-spec prerequisites create-tasks .
  ai-spec prerequisites python-implementation /caminho/absoluto`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		skill := args[0]
		projectDir := "."
		if len(args) == 2 {
			projectDir = args[1]
		}

		fsys := fs.NewOSFileSystem()
		passed, results, err := prerequisites.Verify(skill, projectDir, fsys)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Erro:", err)
			os.Exit(2)
		}

		fmt.Printf("Verificando pre-requisitos para skill %q em %s\n\n", skill, projectDir)

		for _, r := range results {
			var status string
			switch {
			case r.Found:
				status = "OK  "
			case r.Optional:
				status = "AVISO"
			default:
				status = "FALHA"
			}

			qualifier := ""
			if r.Optional {
				qualifier = " (opcional)"
			}
			fmt.Printf("  [%s] %s%s\n", status, r.Label, qualifier)
		}

		fmt.Println()
		if passed {
			fmt.Printf("Resultado: skill %q pode ser executada\n", skill)
			return nil
		}

		fmt.Fprintf(os.Stderr, "Resultado: pre-requisitos ausentes para skill %q\n", skill)
		os.Exit(1)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(prerequisitesCmd)
}
