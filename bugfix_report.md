# Relatorio de Bugfix

- Total de bugs no escopo: 2
- Corrigidos: 1
- Testes de regressao adicionados: 4
- Pendentes: BUG-3 (bloqueado — ambiente, sem fix de codigo necessario)

## Bugs

- ID: BUG-2
- Severidade: media
- Estado: fixed
- Causa raiz: `internal/taskloop/report.go` usava `exitCode == 0 && postStatus == "done"` como criterio de sucesso. Quando o agente era morto por SIGKILL apos timeout (`exit=-1`) mas ja havia marcado a task como `done`, a condicao `exitCode != 0` classificava a iteracao como falha. O criterio primario de sucesso deve ser o estado observavel da task (`postStatus == "done"`), nao o exit code do processo. O mesmo erro existia na condicao de invocacao do reviewer em `taskloop.go:467`.
- Arquivos alterados: `internal/taskloop/report.go` (logica de classificacao em renderSimples e renderAvancado), `internal/taskloop/taskloop.go` (condicao do reviewer)
- Teste de regressao: `TestReportRenderResumoSection/exit_code_-1_e_post_status_done_—_conta_como_sucesso_(regressao_BUG-2)` (report_test.go); `TestExecuteAdvancedModeReviewerInvokedOnTimeoutWithDone` (taskloop_test.go)
- Validacao: go test ./internal/taskloop/... → PASS; make test → PASS (33 pacotes); go vet → sem issues; make lint → 0 issues

---

- ID: BUG-3
- Severidade: critica (CI) / informativa (dev)
- Estado: blocked
- Causa raiz: ANTHROPIC_API_KEY nao definida no ambiente do subprocesso. O harness ja detecta (`isAuthError`), reporta e aborta corretamente. A heuristica `warnClaudeAuth()` nao emitiu aviso porque `~/.claude/` existe. Nao e bug de codigo — e configuracao de ambiente.
- Arquivos alterados: nenhum
- Teste de regressao: n/a
- Validacao: n/a

## Comandos Executados

- `go test ./internal/taskloop/... -v -run "TestReportRenderResumoSection|TestExecuteAdvancedModeReviewer"` -> PASS (todos os sub-testes)
- `make test` -> ok (33 pacotes, sem falhas)
- `go vet ./internal/taskloop/...` -> sem issues
- `make lint` -> 0 issues (golangci-lint 2.2.1)

## Riscos Residuais

- BUG-1 (tasks/ no .gitignore bloqueia read_file do gemini): sem fix de codigo — requer decisao sobre remoção de tasks/ do .gitignore
- BUG-4 (quota throttling gemini): sem fix de codigo — limitacao de conta; usar timeout padrao (30m) em vez de timeout reduzido de teste
- Invariante preservada: exit!=0 com postStatus nao-done continua contabilizado como falha (coberto por TestReportRenderResumoSection/exit_code_nao_zero_e_post_status_in_progress)

## Estado terminal: done
