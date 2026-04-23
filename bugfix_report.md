# Relatorio de Bugfix

- Total de bugs no escopo: 1
- Corrigidos: 1
- Testes de regressao adicionados: 5
- Pendentes: nenhum
- Estado final: done

## Bugs
- ID: BUG-TASKLOOP-ISOLATION-001
- Severidade: critical
- Estado: fixed
- Causa raiz: o snapshot de isolamento protegia apenas `tasks.md`, arquivos de task detectados no diretório raiz e `prd.md`/`techspec.md`. Além disso, a heurística `isTrackedTaskFile` tratava praticamente qualquer `.md` do PRD como task file, deixando arquivos arbitrários e caminhos aninhados fora da proteção correta. Isso permitia que mudanças indevidas no PRD vazassem entre iterações e sessões.
- Arquivos alterados: `internal/taskloop/isolation.go`, `internal/taskloop/isolation_test.go`, `internal/taskloop/taskloop.go`, `internal/taskloop/taskloop_test.go`
- Teste de regressao:
  - `TestValidateTaskIsolationRejectsArbitraryPRDFileMutation`
  - `TestValidateReviewerIsolationRejectsArbitraryNestedPRDFileCreation`
  - `TestRestoreTaskIsolationSnapshotAtRemovesUnexpectedProtectedPRDFiles`
  - `TestExecuteRejectsArbitraryPRDFileMutationForAllProviders`
  - `TestExecuteRejectsReviewerArbitraryPRDMutation`
- Validacao:
  - `go test ./internal/taskloop/...` -> `ok  	github.com/JailtonJunior94/ai-spec-harness/internal/taskloop	2.323s`
  - `go test ./...` -> `ok` em toda a árvore
  - `go vet ./...` -> sem issues
  - `make lint` -> `0 issues.`

## Comandos Executados
- `gofmt -w internal/taskloop/isolation.go internal/taskloop/isolation_test.go internal/taskloop/taskloop.go internal/taskloop/taskloop_test.go` -> ok
- `go test ./internal/taskloop/...` -> ok
- `go test ./...` -> ok
- `go vet ./...` -> ok
- `make lint` -> ok

## Riscos Residuais
- O escopo corrigido protege arquivos arbitrários do PRD e subdiretórios, mas não altera a política funcional de quais artefatos do executor são explicitamente permitidos (`execution_report`, `bugfix_report`, `report.md`).
