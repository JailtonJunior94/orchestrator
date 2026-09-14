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
e reprova qualquer RF sem tarefa, tarefa sem relatorio ou criterio, criterio
sem linha de evidencia e mapa 1:1 incompleto entre a task file e o relatorio.

Tarefa com status blocked que ja tem relatorio de execucao escrito segue cobrada.
A isencao de contrato de evidencia v1 (historico) cobre somente a forma estrita da
evidencia por criterio; o mapa 1:1 continua cobrado (RF-53).

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

			violations, notices := traceMap.Report()

			for _, n := range notices {
				fmt.Printf("AVISO: %s\n", n.String())
			}

			verified := traceMap.VerifiedTaskCount()

			if len(violations) == 0 {
				fmt.Printf("OK: cadeia de rastreabilidade verificada — %d requisitos, %d tarefas (%d verificada(s), %d isenta(s)/nao confrontada(s)).\n",
					len(traceMap.Requirements), len(traceMap.TaskCoverage), verified, len(traceMap.TaskCoverage)-verified)
				return nil
			}

			for _, v := range violations {
				fmt.Printf("RUPTURA: %s\n", v.String())
			}
			fmt.Printf("%d ruptura(s), %d tarefa(s) verificada(s) de %d, %d isencao(oes) declarada(s).\n",
				len(violations), verified, len(traceMap.TaskCoverage), len(notices))
			return newExitError(1)
		},
	}
	return cmd
}
