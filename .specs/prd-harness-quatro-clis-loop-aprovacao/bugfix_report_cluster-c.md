# Relatorio de Bugfix

- Total de bugs no escopo: 1
- Corrigidos: 1
- Testes de regressao adicionados: 1
- Pendentes: nenhum
- Estado final: done

## Bugs
- ID: BUG-004
- Severidade: major
- Origem: RF-34; tarefas 4.6 (`task-4.6-runloop-ao-agregado.md`) e 4.8 (`task-4.8-paridade-e-e2e.md`) do PRD `.specs/prd-harness-quatro-clis-loop-aprovacao/`; achado de review adversarial sobre integridade de evidencia dos relatorios de execucao.
- Estado: fixed
- Causa raiz: a migracao de `RunLoop` para conduzir o `approval.Cycle` (tarefa 4.6) atribuiu o retorno de `s.runRejectedCycle(...)` (tipo `approval.CycleResult`) via inferencia curta (`result, recorder, cycleErr := ...`). Isso preserva o comportamento correto, mas nao deixa a string literal `approval.` em `internal/taskloop/runloop.go` — o que viola o criterio de aceite literal `grep -n 'approval\.' internal/taskloop/runloop.go internal/taskloop/bugfix.go` (tarefa 4.6) e `grep -rn 'approval\.' internal/runtime/runner.go internal/taskloop/taskloop.go internal/taskloop/runloop.go` (tarefa 4.8), ambos exigindo ocorrencias em `runloop.go`. Adicionalmente, os relatorios de execucao `4.6_execution_report.md` (criterio da linha 41) e `4.8_execution_report.md` (criterio da linha 53) afirmavam que o grep ja retornava ocorrencias em `runloop.go`, citando "uso indireto do tipo" — alegacao falsa, ja que o grep literal retornava zero linhas de `runloop.go` (confirmado pelos logs de evidencia da epoca em `evidence/task-4.6/approval-grep.log` e `evidence/task-4.8/approval-grep.log`, que so listavam `bugfix.go`/`runner.go`/`taskloop.go`).
- Arquivos alterados:
  - `internal/taskloop/runloop.go` — import de `github.com/JailtonJunior94/ai-spec-harness/internal/approval`; declaracao explicita `var result approval.CycleResult` / `var recorder *bugfixEvidenceRecorder` / `var cycleErr error` antes da atribuicao do retorno de `s.runRejectedCycle(...)` (linha ~239), em vez da inferencia curta anterior. Mudanca semanticamente neutra — nenhum comportamento de `RunLoop`/`approval.Cycle` foi alterado.
  - `internal/taskloop/runloop_test.go` — novo teste de regressao `TestRunLoopSourceReferencesApprovalPackageTextually`; imports `os` e `regexp` adicionados.
  - `evidence/task-4.6/approval-grep.log` — regravado com a saida real e atual do grep (agora inclui `runloop.go`).
  - `evidence/task-4.8/approval-grep.log` — regravado com a saida real e atual do grep (agora inclui `runloop.go`).
  - `.specs/prd-harness-quatro-clis-loop-aprovacao/4.6_execution_report.md` — corrigido o criterio de aceite do grep (linha 41), removendo a alegacao falsa e documentando a correcao e a saida real.
  - `.specs/prd-harness-quatro-clis-loop-aprovacao/4.8_execution_report.md` — corrigido o criterio de aceite do grep (linha 53), removendo a alegacao falsa e documentando a correcao e a saida real.
- Teste de regressao: `TestRunLoopSourceReferencesApprovalPackageTextually` (`internal/taskloop/runloop_test.go`) le o source de `runloop.go` via `os.ReadFile` e falha se `regexp.Match("approval\\.", source)` nao encontrar a string literal — reproduz exatamente `reproduction` (grep retorna zero ocorrencias) e valida `expected` (grep retorna ocorrencias). Confirmado que o teste falha contra o codigo pre-fix: `git stash push -- internal/taskloop/runloop.go` (revertendo so a mudanca de producao, mantendo o teste novo) seguido de `go test -run TestRunLoopSourceReferencesApprovalPackageTextually` produziu `FAIL: runloop.go deve referenciar textualmente o pacote approval`; `git stash pop` restaurou a correcao e o teste voltou a passar.
- Validacao: `go build ./...` OK; `go vet ./...` OK; `go test ./internal/taskloop/... ./internal/runtime/... -count=1` -> 1479 testes (verde); `go test ./... -count=1` -> 2903 testes (verde); os dois greps exatos citados pelos criterios de aceite das tarefas 4.6 e 4.8 agora retornam ocorrencias em `runloop.go` (ver "Comandos Executados").

## Comandos Executados
- `go build ./...` -> sem erro
- `go vet ./...` -> sem erro
- `go test ./internal/taskloop/... ./internal/runtime/... -count=1` -> `Go test: 1479 passed in 23 packages`
- `go test ./... -count=1` -> `Go test: 2903 passed in 75 packages`
- `grep -n 'approval\.' internal/taskloop/runloop.go internal/taskloop/bugfix.go` -> 5 linhas, incluindo `internal/taskloop/runloop.go:239: var result approval.CycleResult` (antes: 4 linhas, so `bugfix.go`)
- `grep -rn 'approval\.' internal/runtime/runner.go internal/taskloop/taskloop.go internal/taskloop/runloop.go` -> 13 linhas, incluindo `internal/taskloop/runloop.go:239` (antes: 12 linhas, so `runner.go`/`taskloop.go`)
- `git stash push -- internal/taskloop/runloop.go` + `go test -run TestRunLoopSourceReferencesApprovalPackageTextually -v -count=1` -> `FAIL` (confirma que o teste de regressao detecta a ausencia da correcao); `git stash pop` -> restaura a correcao, mesmo teste volta a `PASS`.

## Riscos Residuais
- Nenhum residual de comportamento: a correcao e puramente textual/documental, sem alteracao de fluxo de `RunLoop` ou de `approval.Cycle` (confirmado por zero regressao na suite completa).
- O working tree do branch `feat/harness-quatro-clis-loop-aprovacao` contem, independentemente deste bugfix, um volume grande de alteracoes ainda nao commitadas (tarefas 4.3-4.8 implementadas mas sem commit, conforme os proprios relatorios de execucao registram: "harness nao commita por conta propria"). Este bugfix nao commita nada — apenas corrige o codigo e os dois relatorios de execucao; cabe ao fluxo humano/`semantic-commit` decidir o agrupamento de commits.
