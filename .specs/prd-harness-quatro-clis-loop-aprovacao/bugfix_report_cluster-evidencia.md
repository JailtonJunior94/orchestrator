# Relatorio de Bugfix

- Total de bugs no escopo: 10
- Corrigidos: 9
- Testes de regressao adicionados: 10
- Pendentes: 1 (BAIXO-9 — falso positivo confirmado; correcao aplicada e revertida, nenhuma mudanca retida)
- Estado final: done

> Leitura honesta do totalizador de testes: dos 10 blocos, **8 ganharam cobertura nova** — 5 funcoes de
> teste Go novas (`sealed_evidence_validation_test.go` x3, `review_verdict_cycle_parity_test.go`,
> `review_criteria_completeness_test.go`) mais casos novos/estendidos em `scripts/test-validators.sh` e
> fixtures de shell. Os blocos **ALTO-4** e **BAIXO-9** nao receberam teste novo: sao cobertos por gates ja
> existentes (reexecucao dos comandos citados no CHANGELOG e `make check-spec-paths` +
> `tests/scripts/check-spec-paths_test.sh`, respectivamente).

## Bugs

- ID: ALTO-1 — prova fisica estruturalmente insatisfazivel
- Severidade: critical
- Origem: RF-14 / RF-55 (contrato de evidencia), finding de review do cluster de evidencia
- Estado: fixed
- Causa raiz: `Orchestrator.ValidateExecutionEvidence` recomputava `base_sha`/`patch_sha256`/`final_state_sha256`
  contra um snapshot da arvore de trabalho capturado **no instante da validacao**. Como o `HEAD` avanca
  depois da execucao de cada tarefa, a comparacao nunca podia fechar. Nao era efeito de arvore suja:
  com `git status --porcelain` vazio e tudo commitado, os 27 resultados selados continuavam reprovando.
  Segunda causa, encadeada: `validate-result --verify-physical` acrescentava o relatorio Markdown ao
  conjunto de exclusoes, de modo que o patch recomposto na verificacao divergia do recomposto na selagem
  e todo selo legitimo falhava. Um selo so e re-auditavel se o verificador nao puder alterar o recorte.
- Arquivos alterados: `internal/taskloop/orchestrator.go`
- Teste de regressao: `internal/taskloop/sealed_evidence_validation_test.go`
  (`TestValidateExecutionEvidenceAceitaSeloAposRepositorioAvancar`,
  `TestValidateExecutionEvidenceRecusaSeloAdulterado`,
  `TestValidateExecutionEvidenceIgnoraExclusoesOperacionaisDoChamador`)
- Validacao: varredura nos 28 relatorios saiu de PASS=0/FAIL=28 para PASS=17/FAIL=11. Direcao adversarial
  provada nos quatro campos (`commit_patch_sha256`, `base_sha`, `commit_sha`, `patch_sha256`): adulterar
  qualquer um reprova; restaurar aprova.

- ID: ALTO-2 — escape v1 cobria o mapa 1:1 (RF-53 violado)
- Severidade: major
- Origem: RF-47 / RF-51 / RF-53
- Estado: fixed
- Causa raiz: em `validate-task-evidence.sh` o branch de contrato v1 consumia toda a cadeia `if/elif` do
  mapa de criterios, pulando o gate. Como nenhum dos 28 relatorios declara `<!-- evidence-contract: v2 -->`,
  100% do universo ficava isento e o gate central de RF-47/RF-51 estava inerte no corpus inteiro.
- Arquivos alterados: `.agents/scripts/validate-task-evidence.sh` (+3 espelhos)
- Teste de regressao: casos `h1`–`h5` de `scripts/test-validators.sh` passam a exercitar o mapa; caso novo
  de opt-out legado que nao faz mapa incompleto passar.
- Validacao: nenhum mapa foi inventado — os 28 relatorios ja declaravam `## Criterios de Aceite` com task
  file resolvivel, e nenhum passou a reprovar por mapa incompleto. Direcao adversarial: remover uma linha
  `-> comprovado:` de um relatorio com 9 criterios reprova com
  `criterios de aceite comprovados (8) < definidos na task (9)`; restaurar aprova.

- ID: ALTO-3 — `blocked` como escape de uma palavra
- Severidade: major
- Origem: RF-33 / RF-36
- Estado: fixed
- Causa raiz: `validate-session-end.sh` filtrava `status == "in_progress" || status == "done"`. Mover uma
  tarefa para `blocked` a removia do gate de encerramento, mesmo com relatorio de execucao escrito e
  veredito nao-aprovador. Uma tarefa com relatorio escrito e materialmente ativa.
- Arquivos alterados: `.agents/scripts/validate-session-end.sh` (+11 espelhos)
- Teste de regressao: cinco fixtures cobrindo `blocked` sem relatorio (nao cobra), `blocked` com relatorio
  `APPROVED` (nao cobra), `blocked` com `APPROVED_WITH_REMARKS` (cobra), `blocked` com `REJECTED` (cobra),
  `done` com `APPROVED` (nao regride).
- Validacao: neste proprio repositorio o gate saiu de `exit 0` para `exit 2`, listando as 11 tarefas
  `blocked` com relatorio e veredito `APPROVED_WITH_REMARKS`. **Nao foi enfraquecido para ficar verde.**

- ID: ALTO-4 — CHANGELOG com afirmacao falsa verificavel
- Severidade: major
- Origem: RF-56
- Estado: fixed
- Causa raiz: a secao 2.0.0 afirmava selo em "9 dos 28" (real: 27 dos 28, o commit `15c143b` selou os demais
  depois da redacao) e atribuia a reprovacao fisica a "arvore suja", o que e falso conforme ALTO-1.
- Arquivos alterados: `CHANGELOG.md`
- Teste de regressao: reexecucao de cada comando citado como prova na propria entrada (gate manual RF-56)
- Validacao: cada comando citado como prova foi reexecutado e confere — `ls .specs/*/*_execution_result.json`
  -> 28; `grep -l commit_patch_sha256` -> 27; varredura -> PASS=17/FAIL=11; `check-traceability` -> exit 1.

- ID: MEDIO-5 — divergencia validador<->Ciclo (vetor de fabricacao)
- Severidade: major
- Origem: RF-52
- Estado: fixed
- Causa raiz: `internal/evidence` extraia veredito com `reviewverdict.ParseText`, que pega a primeira
  declaracao e ignora cercas de codigo, enquanto `approval.Translator` respeita cercas e reprova
  contradicao. A divergencia era sempre na direcao perigosa: validador aprovava onde o Ciclo recusava.
- Arquivos alterados: `internal/reviewverdict/document.go` (novo), `internal/evidence/evidence.go`
- Teste de regressao: `internal/evidence/review_verdict_cycle_parity_test.go`
  (`TestReviewVerdictExtractionMatchesApprovalCycle`, 12 casos confrontando as duas implementacoes)
- Validacao: tres vetores medidos antes/depois — veredito so dentro de cerca `APPROVED -> sem veredito
  canonico`; vereditos contraditorios `APPROVED -> BLOCKED`; veredito fora da cerca vencendo o de dentro
  `APPROVED -> REJECTED`. Os tres agora coincidem com o Ciclo.

- ID: MEDIO-6 — RF-51 estruturalmente inverificavel
- Severidade: major
- Origem: RF-51
- Estado: fixed
- Causa raiz: nem `validate-review-evidence.sh` nem `internal/evidence` liam a task file, entao nao
  conseguiam confrontar completude 1:1 — um review cobrindo 1 de 2 criterios passava nos dois.
- Arquivos alterados: `.agents/scripts/validate-review-evidence.sh` (+3 espelhos),
  `internal/evidence/evidence.go`, `internal/evidence/task_gates.go`,
  `.agents/skills/review/assets/review-report-template.md` (+3 espelhos)
- Teste de regressao: `internal/evidence/review_criteria_completeness_test.go`
  (`TestValidateReview_ConfrontaCompletudeContraTaskFile`, confronta Go e shell nos mesmos 4 casos)
- Validacao: mapa 2/2 aprova; mapa 1/2 reprova com `mapa 1:1 incompleto`; task file ausente e task file
  inexistente reprovam fail-closed. Go e shell concordam nos quatro.

- ID: MEDIO-7 — escapes de governanca sem auditoria e cobertura de extensoes incompleta
- Severidade: major
- Origem: finding de review do cluster de evidencia
- Estado: fixed
- Causa raiz: `GOVERNANCE_PRELOAD_CONFIRMED=1` e `GOVERNANCE_PRELOAD_MODE=warn` desligavam o gate
  pre-ferramenta sem deixar registro; e o `case` de extensao so gateava `.go .py .ts .js .tsx .jsx .cs`,
  deixando `.sh`, `.rb`, `.java` e `.sql` com `exit 0` — inclusive os proprios hooks e validadores shell.
- Arquivos alterados: `.agents/hooks/validate-preload.sh` (+1 espelho embarcado), `.gitignore`
- Teste de regressao: sete fixtures cobrindo `.sh`/`.sql`/`.java`/`.rb` gateados, `.md` nao gateado e os
  dois escapes produzindo linha de auditoria.
- Validacao: `.sh` saiu de `exit 0` para `exit 2`; `.md` continua `exit 0`; ambos os escapes gravam
  `timestamp/motivo/alvo/ferramenta/usuario` em `.aispec/governance-escapes.log`.
  **Os espelhos por ferramenta sao delegadores finos por contrato** (`TestPreToolHooksDelegateToCanonicalScript`)
  e foram restaurados — so o canonico carrega logica.

- ID: MEDIO-8 — job de CI que falhava sempre
- Severidade: major
- Origem: `.github/workflows/test.yml:158-174`
- Estado: fixed
- Causa raiz: `scripts/test-sdd-evals.sh` exigia `.specs/prd-sdd-robusto/sdd-state.json`. **Correcao de
  premissa:** o diretorio nao "nunca existiu" — `git log --all` mostra 10 commits e a remocao em
  `91f8cb9 (chore) remove docs`. O script nunca foi atualizado, entao o job `sdd-evals` falhava duro em
  toda execucao, nos tres sistemas operacionais da matriz.
- Arquivos alterados: `scripts/test-sdd-evals.sh`
- Teste de regressao: tres fixtures — escopo existente com checkpoint legado `.yaml` reprova; escopo
  existente sem `sdd-state.json` reprova; escopo ausente ignora o bloco e roda o corpus.
- Validacao: decisao tomada = dar ao script a mesma disciplina de escopo declarado de `check-spec-paths.sh`
  (`SDD_EVALS_PRD_DIR`), **nao** remover o job — o corpus adversarial e o valor real do job e continua
  rodando incondicionalmente (21 fixtures, 20 rejeitadas, 1 controle aceito). `exit 1` -> `exit 0`.

- ID: BAIXO-9 — isencao `.opencode/plugins` em `check-spec-paths.sh`
- Severidade: minor
- Origem: finding de review do cluster de evidencia
- Estado: skipped
- Causa raiz: nao ha defeito. `.opencode/plugin` (singular) **existe**, logo nao precisa de isencao nenhuma.
  A isencao para `.opencode/plugins` (plural) existe porque `prd.md:107` documenta que o OpenCode
  auto-descobre as duas formas upstream, e a forma plural nao e adotada neste repositorio. Trocar plural por
  singular quebrou `make check-spec-paths` com
  `prd.md cita caminho inexistente: .opencode/plugins` — provando que a isencao original estava correta.
- Arquivos alterados: nenhum (revertido; `git diff` limpo em `scripts/check-spec-paths.sh`)
- Teste de regressao: `make check-spec-paths` + `tests/scripts/check-spec-paths_test.sh` (gate existente que
  reprovou a correcao equivocada e provou que a isencao original estava certa)
- Validacao: `make check-spec-paths` -> exit 0, "12 OK, 0 FAIL" apos a reversao.

- ID: BAIXO-10 — `uninstall` nao restaura a chave `stop` do Copilot
- Severidade: minor
- Origem: RF-59
- Estado: fixed
- Causa raiz: a migracao RF-59 renomeia `stop`/`Stop`/`sessionEnd`/`SessionEnd` para `agentStop` preservando
  conteudo e indentacao; `uninstall` nao desfaz o rename. Comportamento intencional — `stop` nao e ponto de
  extensao reconhecido pelo Copilot CLI, e reverter devolveria um hook que nunca dispara.
- Arquivos alterados: `docs/migracao-legacy-acp.md`
- Teste de regressao: suite existente de `internal/upgrade` + `internal/uninstall` (verde; documenta-se o
  comportamento que ela ja trava, sem alterar codigo)
- Validacao: secao nova descreve o diff esperado, como conferi-lo e por que nao ha igualdade byte a byte.

## Comandos Executados
- `go test ./... -count=1 -p 1` -> exit 0 (60 pacotes ok, 0 FAIL)
- `go test -tags=integration ./... -count=1 -p 1` -> exit 0 (62 pacotes ok, 0 FAIL)
- `bash scripts/test-validators.sh` -> exit 0 (Passaram: 30 | Falharam: 0)
- `bash scripts/test-hooks.sh` -> exit 0 (50 asserts OK, 0 FAIL)
- `make check-spec-paths` -> exit 0 (12 OK, 0 FAIL)
- `make check-skills-sync` -> exit 0
- `make check-scripts-sync` -> exit 0 (29 validadores em sync, drift 0)
- `make check-hooks-sync` -> exit 0 (7/7 mirrors do gate de encerramento)
- `make test-sdd-evals` -> exit 0 (21 fixtures, 20 rejeitadas, 1 aceita)
- varredura nos 28 `_execution_report.md` -> PASS=17, FAIL=11
- `bash .agents/scripts/validate-session-end.sh` -> exit 2 (11 tarefas cobradas)
- `ai-spec check-traceability .specs/prd-harness-quatro-clis-loop-aprovacao` -> exit 1 (gate vacuo)

## Riscos Residuais
- **11 relatorios continuam reprovando por veredito `APPROVED_WITH_REMARKS`** (1.0, 4.6, 4.7, 6.0, 7.0, 8.0,
  9.0, 10.0, 11.0 do PRD harness; 7.0 e 8.0 do PRD memoria-duravel). E divida real, nao defeito de gate.
  Nenhum relatorio foi reescrito para passar (RF-56). O `11.0` acumula um segundo defeito material: o
  criterio `changelog-breaking-changes-section-complete` nao referencia evidencia declarada.
- **`ai-spec check-traceability` continua saindo 1 com gate vacuo** (18 tarefas, 0 verificadas). Mantem os
  escapes `historical_evidence_contract` e `task_blocked` em `internal/traceability`, pacote **fora do escopo
  declarado** deste cluster. E o analogo Go dos ALTO-2 e ALTO-3 e deveria receber o mesmo tratamento.
- **`scripts/check-traceability.sh` nao existe** em nenhuma ref (confirmado por `git log --all`). O comando
  valido e o subcomando `ai-spec check-traceability`. A referencia ao script no enunciado do cluster e falsa.
- **Campo `- Task file:` passa a ser obrigatorio no relatorio de review.** Raio de impacto zero hoje (nao ha
  nenhum review report versionado no repositorio), mas e mudanca de contrato do template.
- **A arvore de trabalho e compartilhada com outros agentes ativos.** `git status --porcelain` mostra 108
  entradas; 42 sao minhas. Nao foi possivel deixar a arvore contendo apenas os meus arquivos.
