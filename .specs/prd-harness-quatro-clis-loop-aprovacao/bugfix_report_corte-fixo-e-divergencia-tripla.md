# Relatorio de Bugfix — corte fixo do contrato de evidencia e divergencia tripla de estado

- Total de bugs no escopo: 2
- Corrigidos: 2
- Testes de regressao adicionados: 2
- Pendentes: nenhum
- Estado final: done

Os dois blocos de bug somam 9 casos de teste novos ou reescritos: 4 casos Go novos, 2 casos Go
reescritos e 3 blocos shell (casos k e l, mais TC20/TC20c/TC24). A divida de evidencia que os
gates passaram a expor esta em Riscos Residuais.

## Bugs

- ID: ALTO-1 — isencao historica ancorada em referencia movel e verificada por existencia
- Severidade: critical
- Origem: RF-53 / RF-55 / RF-56 (contrato de evidencia), risco residual declarado em
  `bugfix_report_cluster-evidencia.md`
- Estado: fixed
- Causa raiz: `internal/evidence/contract.go:20` definia `contractCutRef = "HEAD"` e
  `.agents/scripts/validate-task-evidence.sh:34-41` usava o mesmo `HEAD` com `git cat-file -e`.
  Duas falhas encadeadas. Primeira: o corte andava junto com o repositorio, entao qualquer
  relatorio ganhava a isencao historica **no instante em que era commitado** — trabalho novo
  virava "historico" sem mudar um byte. Segunda: `cat-file -e` prova que o **arquivo existe**
  no ref, nao que o **conteudo** corresponde; um relatorio reescrito no worktree mantinha a
  isencao v1.
  Escolha do corte: a string `evidence-contract` nao existe em nenhuma ref anterior a
  `76f65e1` (`git log --all -S'evidence-contract'`), commit que criou `internal/evidence/contract.go`
  e o marcador v2. Logo, o ultimo estado de arvore em que nenhum relatorio poderia declarar o
  marcador e o pai de `76f65e1`, que e **`0d84ccd`** — o commit fixado. Relatorios selados nesse
  commit sao legitimamente historicos; qualquer coisa criada ou editada depois e cobrada.
  O corte nao foi escolhido para reduzir reprovacao: ele **retira** a isencao de 5 relatorios
  editados apos o contrato existir e a nega permanentemente a todo relatorio futuro.
- Arquivos alterados: `internal/evidence/contract.go`,
  `.agents/scripts/validate-task-evidence.sh` (+3 espelhos)
- Teste de regressao: `internal/evidence/contract_cut_test.go`
  (`TestResolveContract_ExemptionRequiresSealedContentNotMereExistence`,
  `TestResolveContract_TamperedTextIsNotLaunderedByPristineFile`,
  `TestContractCut_IsPinnedCommitNotMovingRef`,
  `TestContractCut_ProductionCodeNeverReassignsTheCut`);
  `internal/evidence/regression_test.go` (`TestResolveContract_CutRuleRejectsNewWorkPosingAsHistorical`
  e `TestResolveContract_CutRefIsNotEnvironmentControlled` reescritos);
  `scripts/test-validators.sh` (caso k);
  `tests/scripts/validate-task-evidence_test.sh` (TC20 reescrito, TC20c novo, TC24 reescrito)
- Validacao: prova adversarial nos dois sentidos. Com o validador de `b098f5d`, um relatorio novo
  de bytes identicos recebe `AVISO: contrato de evidencia v1` assim que e commitado; com o
  validador corrigido, continua reprovando e citando o commit de corte. Mutacoes M1 (corte volta a
  `HEAD`), M2 (volta a checar existencia), M3 (logica ignora o corte e segue `HEAD`), M5 (validador
  antigo) e M6 (corte inalcancavel, isencao removida) reprovam a suite; restaurar aprova.

- ID: ALTO-2 — divergencia tripla de estado em 10 tarefas
- Severidade: major
- Origem: RF-33 / RF-36 (residuo da manobra de editar `tasks.md` para `blocked` a fim de
  contornar o gate de encerramento)
- Estado: fixed
- Causa raiz: `tasks.md` de dois PRDs registrava `blocked` para 10 tarefas cujos relatorios e
  `<id>_execution_result.json` declaravam `done`. Nao era defeito de codigo: era residuo de dados
  de uma sessao anterior, em que apenas `tasks.md` foi editado, sem tocar relatorios nem JSONs.
  A reconciliacao foi feita **de `tasks.md` em direcao a evidencia**, tarefa a tarefa, exigindo os
  tres sustentaculos juntos: veredito que encerra o ciclo (RF-33), `status` do JSON e prova fisica
  valida. Nenhum `verdict=`, `Estado:`, `status` ou `review_verdict` foi reescrito.
- Arquivos alterados: `.specs/prd-harness-quatro-clis-loop-aprovacao/tasks.md`,
  `.specs/prd-memoria-duravel-agentes/tasks.md` (somente a coluna Status de 6 linhas)
- Teste de regressao: `scripts/test-validators.sh` (caso l, cobrindo os tres pares de
  cruzamento de estado, que ate entao nao tinha nenhuma cobertura naquela suite)
- Validacao: 6 tarefas foram para `done` porque os tres sustentaculos as sustentam
  (harness 1.0, 6.0, 8.0, 10.0; memoria-duravel 7.0, 8.0). 4 permanecem `blocked` com motivo
  declarado. O gate de encerramento tem saida **byte-identica** antes e depois da reconciliacao
  (exit 2, bloqueando em 9.0 e 11.0), o que prova que nada foi virado para deixar o gate verde.
  Mutacao M4 (remover o cruzamento `relatorio x tasks.md` do validador) reprova o caso l.

## Reconciliacao tarefa a tarefa

| Tarefa | Veredito encerra? | JSON status | Prova fisica | tasks.md | Motivo |
|---|---|---|---|---|---|
| harness 1.0 | sim (so remarks `low`) | done | OK | blocked -> **done** | tres sustentaculos |
| harness 6.0 | sim (max `medium`) | done | OK | blocked -> **done** | tres sustentaculos |
| harness 8.0 | sim (max `medium`) | done | OK | blocked -> **done** | tres sustentaculos |
| harness 10.0 | sim (max `medium`) | done | OK | blocked -> **done** | tres sustentaculos |
| memoria 7.0 | sim (max `medium`) | done | OK | blocked -> **done** | tres sustentaculos |
| memoria 8.0 | sim (so `low`) | done | OK | blocked -> **done** | tres sustentaculos |
| harness 4.6 | **nao** | done | OK | permanece blocked | `APPROVED_WITH_REMARKS` sem achado com severidade canonica: ausencia de high/critical nao verificavel (RF-33, fail-closed) |
| harness 4.7 | **nao** | done | OK | permanece blocked | idem 4.6 |
| harness 7.0 | **nao** | done | OK | permanece blocked | idem 4.6 |
| harness 11.0 | **nao** | **blocked** | OK | permanece blocked | `review_verdict=changes_requested` por decisao do dono; relatorio diz `done` contra o proprio JSON |
| harness 9.0 | **nao** | blocked | **invalida** | permanece blocked | 3 achados `[blocker]`, `review_verdict=needs_input`, snapshot fisico diverge |

9.0 e 11.0 **confirmadas** bloqueadas por dado, como esperado. 4.6, 4.7 e 7.0 permanecem
bloqueadas por motivo proprio — resultado alem do previsto no escopo e reportado como divida real.

## Comandos Executados

- `go build ./...` -> exit 0
- `go vet ./...` -> exit 0
- `go test ./... -count=1 -p 1` -> exit 0
- `go test -tags=integration ./... -count=1 -p 1` -> exit 0 (62 pacotes ok, 0 FAIL)
- `golangci-lint run` -> exit 0, 0 issues
- `golangci-lint run --build-tags integration` -> exit 0, 0 issues
- `make budget` -> exit 0
- `make check-mocks` -> exit 0
- `make test-check-mocks` -> exit 0
- `make check-skills-sync` -> exit 0
- `make check-hooks-sync` -> exit 0
- `make check-scripts-sync` -> exit 0
- `make check-spec-paths` -> exit 0
- `make test-validators` -> exit 0 (53 + 60 casos)
- `make test-hooks` -> exit 0
- `make test-portable-skills` -> exit 0
- `bash .agents/scripts/validate-task-evidence.sh` nos 28 relatorios -> PASS=21 FAIL=7
  (antes: PASS=17 FAIL=11)
- `ai-spec check-traceability` (harness) -> exit 1; deixou de ser `gate_vacuous`: 5 tarefas
  verificadas de 18, 68 rupturas `criterion_without_evidence`
- `ai-spec check-traceability` (memoria-duravel) -> exit 1, segue `gate_vacuous` 0/10
- gate de encerramento -> exit 2, bloqueia em 9.0 e 11.0 (identico ao baseline)

## Suposicoes

- O corte legitimo e o pai do commit que introduziu o contrato v2, e nao o inicio do PRD:
  cobrar o marcador de relatorios escritos semanas antes de ele existir seria retroatividade, e
  RF-53 pressupoe que um escape legado legitimo exista — ele apenas nao cobre o mapa 1:1 nem o
  veredito. O defeito nunca foi a existencia da isencao, e sim a mobilidade da sua fronteira.
- Adicionar `<!-- evidence-contract: v2 -->` aos relatorios que perderam a isencao seria
  **declarar em nome do autor** que a evidencia cumpre o contrato estrito. Nao foi feito: o
  marcador e declaracao do executor, nao do corretor.

## Riscos Residuais

- **7 dos 28 relatorios continuam reprovando** (antes eram 11). Motivos reais, sem suavizacao:
  4.6 e 7.0 (RF-33 fail-closed + perda de isencao), 4.7 (RF-33 fail-closed), 4.8 e 8.0 (apenas
  perda de isencao: conteudo editado apos o corte sem migrar para v2), 9.0 (achados `[blocker]` +
  perda de isencao), 11.0 (relatorio diz `done` contra o proprio JSON `blocked`).
- **4.8 e 8.0 sao reprovacoes novas em substancia zero**: passam em todos os gates materiais e
  falham somente por nao declararem o marcador v2 depois de terem sido editados apos o corte.
  Resolucao correta: o autor do relatorio acrescenta o marcador.
- **O corte fixado nao existe em clone raso.** `.github/workflows/test.yml` usa
  `actions/checkout@v6` sem `fetch-depth`, entao `0d84ccd` nao esta presente no CI e nenhuma
  isencao seria concedida la. Hoje isso e inofensivo porque o CI nao valida os relatorios do
  proprio repositorio; se algum passo passar a validar, e preciso `fetch-depth: 0`. O caso TC20c
  pula explicitamente nessa condicao em vez de reprovar por motivo falso.
- **O gate de encerramento e o validador de evidencia discordam em 4.6, 4.7 e 7.0.** O gate lê
  `review_verdict` do JSON (`approved`) e nao as bloqueia; o validador lê `verdict=` mais as
  severidades declaradas no relatorio e conclui, por RF-33 fail-closed, que o ciclo nao encerra.
  O validador e o mais estrito e o mais correto. Fechar essa assimetria esta fora do escopo deste
  cluster e deveria virar item proprio.
- **O cruzamento de estado das tres fontes existe apenas no validador shell.** Nao ha equivalente
  em `internal/evidence` — mesma classe de lacuna de paridade Go/shell ja registrada para V-27.
- **`check-traceability` do PRD harness saiu de 1 ruptura para 68.** Nao e regressao: antes o
  universo era integralmente isento e o gate era vacuo. Agora 5 tarefas sao de fato confrontadas e
  as 68 rupturas sao criterios cujas linhas de evidencia nao satisfazem as tres formas de RF-48.
