# Relatorio de Bugfix

- Total de bugs no escopo: 2
- Corrigidos: 2
- Testes de regressao adicionados: 2
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-012a
- Severidade: major
- Origem: RF-40, RF-57 (tarefa 4.1 — `translateReviewStatus` traduz veredito fail-closed via `internal/approval.Translator`) e tarefa 4.5 (contrato de texto bruto: so reconhece linha canonica `Verdict: <token>`)
- Estado: fixed
- Causa raiz: o fixture do teste `TestClaudeCrossWaveSmokeE2E` (T-INT-06) foi escrito antes da tarefa 4.1/4.5 introduzirem o contrato de texto bruto. O mock `reviewFn` retornava texto livre `"APPROVED — nenhuma issue critica encontrada"`, sem a linha canonica `Verdict: APPROVED` exigida pelo `Translator.Translate` (`internal/approval/translator.go`). O `Translator` e fail-closed por design: qualquer texto sem a linha canonica cai em `VerdictBlocked`. O teste nao estava testando um bug de producao — estava testando um formato de mock desatualizado.
- Arquivos alterados: `tests/integration/claude_2026_e2e_test.go` (linha ~695: retorno do mock `reviewFn` em `TestClaudeCrossWaveSmokeE2E`)
- Teste de regressao: o proprio `TestClaudeCrossWaveSmokeE2E` (T-INT-06), apos ajuste do fixture para `"Verdict: APPROVED\n\nNenhuma issue critica encontrada."`, passa a exercitar corretamente o caminho de aprovacao do `Translator` e valida `summary.ReviewStatus == "ok"`.
- Validacao: `go build ./...`, `go vet ./...`, `go test -tags=integration ./tests/integration/... -run "TestClaudeCrossWaveSmokeE2E|TestClaudeAutoReviewBlocksOnHardIssueE2E" -v -count=1` — 2 passed.

- ID: BUG-012b
- Severidade: major
- Origem: tarefa 4.2 (D2 — `writeRoundReviewEvidence` grava o texto bruto da saida do revisor em `review.md`, substituindo o antigo pointer renderizado `"# Auto-Review\n\nReviewStatus: %s\n\n..."`)
- Estado: fixed
- Causa raiz: o teste `TestClaudeAutoReviewBlocksOnHardIssueE2E` (T-INT-05) afirmava `strings.Contains(string(content), "ReviewStatus: blocked")` sobre o conteudo de `review.md`. Esse formato de pointer foi deliberadamente removido na tarefa 4.2 — `review.md` agora contem o texto bruto retornado pelo revisor, sem cabecalho `ReviewStatus:`. A asserção testava um formato de arquivo que nao existe mais por design. A asserção separada sobre o campo estruturado `summary.ReviewStatus == "blocked"` (linha ~620) ja estava correta e nao foi tocada.
- Arquivos alterados: `tests/integration/claude_2026_e2e_test.go` (linha ~589: mock `reviewFn` passou a incluir a linha canonica `Verdict: BLOCKED` antes do texto `[HARD]`, para que o `Translator` classifique corretamente o veredito como bloqueado; linha ~639: asserção de conteudo do arquivo trocada de `"ReviewStatus: blocked"` para `"[HARD] eval() detected"`, que e o texto bruto real gravado em `review.md` pelo mock do revisor)
- Teste de regressao: o proprio `TestClaudeAutoReviewBlocksOnHardIssueE2E` (T-INT-05), apos o ajuste, valida que `review.md` contem o texto bruto exato do revisor (incluindo a marca `[HARD]`) e que `summary.ReviewStatus == "blocked"` — cobrindo tanto o campo estruturado quanto a evidencia em texto bruto por rodada (D2).
- Validacao: `go build ./...`, `go vet ./...`, `go test -tags=integration ./tests/integration/... -run "TestClaudeCrossWaveSmokeE2E|TestClaudeAutoReviewBlocksOnHardIssueE2E" -v -count=1` — 2 passed.

## Comandos Executados

- `go build ./...` -> sucesso, sem erros.
- `go vet ./...` -> sucesso, sem warnings.
- `go test -tags=integration ./tests/integration/... -run "TestClaudeCrossWaveSmokeE2E|TestClaudeAutoReviewBlocksOnHardIssueE2E" -v -count=1` -> 2 passed in 2 packages.
- `go test -tags=integration ./... -count=1` -> 3203 passed, 3 failed, 4 skipped em 77 pacotes. Falhas remanescentes fora de escopo:
  - `internal/parity` [build failed]: `E2EParitySuite` indefinido em `internal/parity/e2e_parity_test.go` — falha pre-existente introduzida no commit `900d8af`, muito anterior a este PRD e a este cluster de correcao; confirmado fora de escopo por instrucao explicita da tarefa.
  - `internal/install` (`TestIntegration_7_5_ScenarioG_Monorepo`, `TestIntegration_7_5_ProbeNonFatal_BinaryAbsent`) — nao relacionadas ao contrato de veredito/evidencia do auto-review (Bloco D). Confirmado que nao sao causadas pela correcao deste cluster: a arvore de trabalho do repositorio contem um volume extenso de alteracoes concorrentes de outras tarefas/clusters (`internal/runtime/specs/opencode.go`, `internal/install/install_opencode_test.go`, etc., ja modificados antes desta sessao), e a correcao aplicada neste cluster tocou exclusivamente `tests/integration/claude_2026_e2e_test.go`. `git stash`/`git stash pop` isolado desse arquivo confirmou que o `go build` do pacote `internal/install`/`internal/runtime/specs` ja apresenta instabilidade independente das mudancas deste cluster.

## Riscos Residuais

- Nenhum risco residual no escopo deste cluster (BUG-012a/BUG-012b). As duas correcoes sao restritas a fixtures/asserções de teste E2E; nenhum comportamento de producao (`translateReviewStatus`, `writeRoundReviewEvidence`, `approval.Translator`) foi alterado.
- Falhas remanescentes em `internal/parity` (build quebrado, commit `900d8af`) e `internal/install` (2 testes) permanecem fora do escopo deste cluster e devem ser tratadas por bugfix dedicado ou pelo cluster responsavel pelas tarefas de OpenCode (7.0/8.0), que estao em andamento concorrente na mesma arvore de trabalho.
