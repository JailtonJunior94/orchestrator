package tasks_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd/tasks"
)

const validPRD = `## Requisitos Funcionais

- RF-01: primeiro requisito.
- RF-02: segundo requisito.

## Fora de Escopo
`

const validTasks = `## Tarefas

| # | Título | Status | Dependências | Paralelizável | Ownership |
|---|---|---|---|---|---|
| 1.0 | Base | done | — | Não | internal/base |
| 2.0 | Fluxo | pending | 1.0 | Sim | internal/flow |

## Cobertura de Requisitos

| Tarefa | Requisitos cobertos |
|---|---|
| 1.0 | RF-01 |
| 2.0 | RF-02 |
`

func TestParserParse(t *testing.T) {
	tests := []struct {
		name    string
		prd     string
		tasks   string
		wantErr string
	}{
		{name: "documento valido", prd: validPRD, tasks: validTasks},
		{name: "requisito sem cobertura", prd: validPRD, tasks: strings.Replace(validTasks, "| 2.0 | RF-02 |\n", "", 1), wantErr: "RF-02 sem cobertura"},
		{name: "ciclo local", prd: validPRD, tasks: strings.Replace(validTasks, "| 1.0 | Base | done | — |", "| 1.0 | Base | done | 2.0 |", 1), wantErr: "ciclo de dependencias"},
		{name: "dependencia inexistente", prd: validPRD, tasks: strings.Replace(validTasks, "| 2.0 | Fluxo | pending | 1.0 |", "| 2.0 | Fluxo | pending | 9.0 |", 1), wantErr: "nao existe"},
		{name: "cross PRD explicito", prd: validPRD, tasks: strings.Replace(validTasks, "pending | 1.0", "pending | prd-outro:1.0", 1)},
		{name: "ownership ausente para paralelismo", prd: validPRD, tasks: strings.Replace(validTasks, "| 2.0 | Fluxo | pending | 1.0 | Sim | internal/flow |", "| 2.0 | Fluxo | pending | 1.0 | Sim | — |", 1), wantErr: "sem ownership"},
		{name: "ownership sobreposto", prd: validPRD, tasks: strings.Replace(validTasks, "internal/flow", "internal/base/sub", 1), wantErr: "ownership sobreposto"},
		{name: "ownership com traversal", prd: validPRD, tasks: strings.Replace(validTasks, "internal/flow", "../fora", 1), wantErr: "ownership invalido"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := tasks.NewParser().Parse([]byte(test.prd), []byte(test.tasks))
			if test.wantErr == "" {
				if err != nil {
					t.Fatalf("erro inesperado: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("erro = %v, esperado conter %q", err, test.wantErr)
			}
		})
	}
}
