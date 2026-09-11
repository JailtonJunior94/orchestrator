package aispecharness

import (
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/traceability"
	"github.com/spf13/cobra"
)

func newCheckTraceabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-traceability <diretorio-prd>",
		Short: "Verifica a cadeia requisito -> tarefa -> criterio -> evidencia",
		Long: `Deriva o mapa de rastreabilidade a partir de prd.md, tasks.md e dos
relatorios de execucao (<tarefa>_execution_report.md) sob o diretorio informado,
e reprova qualquer RF sem tarefa, tarefa sem relatorio ou criterio, e criterio
sem linha de evidencia.

Criterio reconhecido: item de lista de topo (sem indentacao) iniciado por "- " dentro da secao
"## Criterios de Aceite". Sub-itens indentados, blocos de codigo cercados por crases triplas,
citacoes e listas numeradas nao contam como criterio.

Exemplo:
  ai-spec check-traceability .specs/prd-minha-feature`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]

			info, err := os.Stat(dir)
			if err != nil || !info.IsDir() {
				fmt.Fprintf(os.Stderr, "erro: diretorio nao encontrado: %s\n", dir)
				return newExitError(2)
			}

			traceMap, err := traceability.NewCatalog().BuildMap(dir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "erro: %s\n", err)
				return newExitError(2)
			}

			violations := traceMap.Validate()
			if len(violations) == 0 {
				fmt.Printf("OK: cadeia de rastreabilidade verificada — %d requisitos, %d tarefas.\n",
					len(traceMap.Requirements), len(traceMap.TaskCoverage))
				return nil
			}

			for _, v := range violations {
				fmt.Printf("RUPTURA: %s\n", v.String())
			}
			return newExitError(1)
		},
	}
	return cmd
}
