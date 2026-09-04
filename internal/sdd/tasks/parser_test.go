package tasks_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd/tasks"
)

const validPRD = `## Requisitos Funcionais

- RF-01: primeiro requisito.
- RF-02: segundo requisito.

## Requisitos Não Funcionais

- NFR-01: requisito de produção.

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
| 2.0 | RF-02, NFR-01 |
`

func TestParserParse(t *testing.T) {
	tests := []struct {
		name    string
		prd     string
		tasks   string
		wantErr string
	}{
		{name: "documento valido", prd: validPRD, tasks: validTasks},
		{name: "requisito sem cobertura", prd: validPRD, tasks: strings.Replace(validTasks, "| 2.0 | RF-02, NFR-01 |\n", "", 1), wantErr: "RF-02 sem cobertura"},
		{name: "NFR sem cobertura", prd: validPRD, tasks: strings.Replace(validTasks, ", NFR-01", "", 1), wantErr: "NFR-01 sem cobertura"},
		{name: "ciclo local", prd: validPRD, tasks: strings.Replace(validTasks, "| 1.0 | Base | done | — |", "| 1.0 | Base | done | 2.0 |", 1), wantErr: "ciclo de dependencias"},
		{name: "dependencia inexistente", prd: validPRD, tasks: strings.Replace(validTasks, "| 2.0 | Fluxo | pending | 1.0 |", "| 2.0 | Fluxo | pending | 9.0 |", 1), wantErr: "nao existe"},
		{name: "cross PRD sem contexto", prd: validPRD, tasks: strings.Replace(validTasks, "pending | 1.0", "pending | prd-outro:1.0", 1), wantErr: "requer diretorio"},
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

func TestParserParseAtResolvesCrossPRDFailClosed(t *testing.T) {
	root := t.TempDir()
	current := filepath.Join(root, "prd-atual")
	external := filepath.Join(root, "prd-outro")
	if err := os.MkdirAll(current, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	externalTasks := []byte("external tasks\n")
	if err := os.WriteFile(filepath.Join(external, "tasks.md"), externalTasks, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(externalTasks)
	currentTasks := strings.Replace(validTasks, "pending | 1.0", "pending | prd-outro:1.0", 1)
	missingTasks := strings.Replace(currentTasks, "prd-outro:1.0", "prd-ausente:1.0", 1)
	if _, err := tasks.NewParser().ParseAt(current, []byte(validPRD), []byte(missingTasks)); err == nil || !strings.Contains(err.Error(), "ausente") {
		t.Fatalf("PRD externo ausente deveria falhar fechado: %v", err)
	}

	tests := []struct {
		name         string
		status       string
		sha          string
		dependencies string
		omitTask     bool
		wantErr      string
	}{
		{name: "destino aprovado e done", status: "done", sha: hex.EncodeToString(digest[:])},
		{name: "destino nao concluido", status: "blocked", sha: hex.EncodeToString(digest[:]), wantErr: "nao esta done"},
		{name: "task externa ausente", status: "done", sha: hex.EncodeToString(digest[:]), omitTask: true, wantErr: "nao esta done"},
		{name: "hash stale", status: "done", sha: strings.Repeat("a", 64), wantErr: "stale"},
		{name: "ciclo cross PRD", status: "done", sha: hex.EncodeToString(digest[:]), dependencies: `"prd-atual:2.0"`, wantErr: "ciclo cross-PRD"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dependencies := test.dependencies
			if dependencies == "" {
				dependencies = ""
			}
			tasksState := fmt.Sprintf(`{"1.0":{"status":%q,"dependencies":[%s]}}`, test.status, dependencies)
			if test.omitTask {
				tasksState = `{}`
			}
			state := fmt.Sprintf(`{"schema_version":2,"artifacts":{"tasks":{"sha256":%q,"approved":true}},"tasks":%s}`, test.sha, tasksState)
			if err := os.WriteFile(filepath.Join(external, "sdd-state.json"), []byte(state), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := tasks.NewParser().ParseAt(current, []byte(validPRD), []byte(currentTasks))
			if test.wantErr == "" && err != nil {
				t.Fatalf("dependencia válida falhou: %v", err)
			}
			if test.wantErr != "" && (err == nil || !strings.Contains(err.Error(), test.wantErr)) {
				t.Fatalf("erro=%v, esperado %q", err, test.wantErr)
			}
		})
	}
}

func TestParserAcceptsCanonicalTasksTableBeforeSections(t *testing.T) {
	tasksContent := strings.Replace(validTasks, "## Tarefas\n\n", "", 1)
	if _, err := tasks.NewParser().Parse([]byte(validPRD), []byte(tasksContent)); err != nil {
		t.Fatalf("tabela canônica no topo deveria ser aceita: %v", err)
	}
}
