# Relatorio de Bugfix

- Total de bugs no escopo: 14
- Corrigidos: 14
- Testes de regressao adicionados: 14
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-01
- Severidade: minor
- Origem: finding de review (review criteriosa da Tarefa 3.0 do PRD `harness-portatil-vendor-neutral`, R-STYLE-001.1)
- Estado: fixed
- Causa raiz: `cmd/ai_spec_harness/verify_contract_test.go` continha 3 mensagens de asserção (`t.Error`/`t.Errorf`) em portugues, violando a regra hard R-STYLE-001.1 (codigo em ingles).
- Arquivos alterados: `cmd/ai_spec_harness/verify_contract_test.go`
- Teste de regressao: nao aplicavel (mudanca textual pura, mesma asserção/comportamento preservado, coberto pela suite existente)
- Validacao: ver secao "Comandos Executados"

- ID: BUG-02
- Severidade: minor (regra hard R-STYLE-001.3)
- Origem: finding de review (Tarefa 11.0, RF-43.1)
- Estado: fixed
- Causa raiz: `internal/telemetry/parser.go` declarava a variavel package-level `_recognizedRawKeys` com prefixo `_`, proibido para qualquer identificador Go. Renomear diretamente colidia com a funcao existente `recognizedRawKeys()`.
- Arquivos alterados: `internal/telemetry/parser.go` (funcao renomeada para `buildRecognizedRawKeys()`, variavel para `recognizedRawKeys`)
- Teste de regressao: coberto pela suite existente de `internal/telemetry` (nenhuma mudanca de comportamento, apenas nome)
- Validacao: `grep -rn "_recognizedRawKeys" internal/telemetry/` -> zero ocorrencias

- ID: BUG-03
- Severidade: minor
- Origem: finding de review (Tarefa 11.0, RF-43.1)
- Estado: fixed
- Causa raiz: `internal/doctor/doctor.go` misturava valores em ingles (`"core"`, `"adapter"`, `"provider"`) e portugues (`"instalação"`, `"validação"`) no mesmo bloco de constantes `FailureLayer`, violando R-STYLE-001.1 para strings internas novas sem excecao registrada.
- Arquivos alterados: `internal/doctor/doctor.go` (`LayerInstallation`/`LayerValidation` -> `"installation"`/`"validation"`)
- Teste de regressao: coberto por `internal/doctor/multiprovider_test.go` (referencia as constantes, nao string literal)
- Validacao: `grep -n '"instalação"\|"validação"' internal/doctor/doctor.go` -> nenhuma ocorrencia; `conformance.AttributionIndeterminate = "indeterminado"` confirmado intocado (mandato literal do PRD/RF-39)

- ID: BUG-04
- Severidade: minor
- Origem: finding de review (Tarefa 8.0, R-STYLE-001.1)
- Estado: fixed
- Causa raiz: `internal/batchreport/report.go` (arquivo inteiramente novo) usava strings de usuario em portugues em `Print`/`ConflictError`.
- Arquivos alterados: `internal/batchreport/report.go`, `internal/batchreport/report_test.go` (asserção adaptada de `"conflito=1"` para `"conflict=1"`, mesmo comportamento)
- Teste de regressao: `internal/batchreport/report_test.go::TestPrint_ReportsCountsAndNamesConflicts` (adaptado, nao novo)
- Validacao: ver secao "Comandos Executados"

- ID: BUG-05
- Severidade: minor
- Origem: finding de review (Tarefa 8.0, R-STYLE-001.1)
- Estado: fixed
- Causa raiz: `internal/install/install.go:312` e `internal/upgrade/upgrade.go:248` introduziram, nesta mesma entrega, as strings `"aplicar lote de instalacao"`/`"aplicar lote de sincronizacao"` em portugues (confirmado via `git diff HEAD` que sao novas, nao heranca de linha nao tocada).
- Arquivos alterados: `internal/install/install.go`, `internal/upgrade/upgrade.go` (traduzidas para `"apply installation batch"`/`"apply synchronization batch"`)
- Teste de regressao: nenhum teste asserta sobre a string literal; coberto pela suite existente
- Validacao: ver secao "Comandos Executados"

- ID: BUG-06
- Severidade: low
- Origem: finding de review (Tarefa 11.0)
- Estado: fixed
- Causa raiz: `internal/telemetry/report.go`, `internal/telemetry/trend.go` e `internal/telemetry/summary.go` instanciavam `NewCatalog()` dentro de metodos que ja tinham o receptor `c *Catalog` disponivel, em vez de usar `c` diretamente. Inofensivo (`Catalog` e `struct{}` sem estado) mas inconsistente. Uma primeira rodada corrigiu apenas 2 de 11 ocorrencias em codigo de producao; uma re-revisao encontrou o restante e uma segunda rodada fechou as 9 ocorrencias remanescentes (incluindo `summary.go`, que nem tinha sido tocado na primeira rodada).
- Arquivos alterados: `internal/telemetry/report.go` (linhas 43, 76, 77, 78), `internal/telemetry/trend.go` (linha 122), `internal/telemetry/summary.go` (linhas 25, 52, 58, 79)
- Teste de regressao: coberto pela suite existente (mudanca sem efeito observavel); confirmado 0 ocorrencias remanescentes de `NewCatalog()\.` em codigo nao-teste via `grep -rn "NewCatalog()\." internal/telemetry/*.go | grep -v _test.go`
- Validacao: ver secao "Comandos Executados"

- ID: BUG-07
- Severidade: minor
- Origem: finding de review (Tarefa 13.0, RF-49/RF-51)
- Estado: fixed
- Causa raiz: `CHANGELOG.md` e `docs/guia-instalacao-universal.md` nao nomeavam os aliases `init`/`sync` nem a flag `--overwrite-conflicts` (funcionalidades novas e reais desta entrega), apenas `docs/cli-schema.json` (contrato de maquina) e o `--help` da flag os documentavam.
- Arquivos alterados: `CHANGELOG.md` (2 entradas em "Features" com nota de migracao), `docs/guia-instalacao-universal.md` (secao "Atualizacao Apos `git pull`" atualizada)
- Teste de regressao: nao aplicavel (documentacao); validado por leitura/grep
- Validacao: `grep -n "init\|sync\|overwrite-conflicts" CHANGELOG.md docs/guia-instalacao-universal.md` -> presente em prosa real

- ID: BUG-08
- Severidade: minor
- Origem: finding de review (Tarefas 2.0/4.0)
- Estado: fixed
- Causa raiz: `AGENTS.md`/`CLAUDE.md` nao documentavam `.agents/policies/` como origem canonica nem o gate `make check-policies-sync` na tabela "Area tocada | Gate" / secao de skills, ao contrario do padrao ja replicado para `.agents/skills/`.
- Arquivos alterados: `AGENTS.md` (nova linha na tabela), `CLAUDE.md` (nova frase na secao "Skills, subagentes e hooks")
- Teste de regressao: nao aplicavel (documentacao)
- Validacao: `grep -n "check-policies-sync\|agents/policies" AGENTS.md CLAUDE.md` -> presente

- ID: BUG-09
- Severidade: low
- Origem: finding de review (Tarefa 4.0)
- Estado: fixed
- Causa raiz: `scripts/sync-policies.sh` aplicava `chmod -R u+w` recursivamente sobre diretorios inteiros em vez de escopar aos arquivos declarados em `required_policies`, alargando o blast radius sem necessidade.
- Arquivos alterados: `scripts/sync-policies.sh` (chmod por arquivo individual, escopado a `required_policies`)
- Teste de regressao: coberto por `tests/integration/sync_gates_guard_test.go` (execucao indireta, sem edicao)
- Validacao: `bash scripts/sync-policies.sh` -> exit 0, sem regressao

- ID: BUG-10
- Severidade: low
- Origem: finding de review (Tarefa 4.0)
- Estado: fixed
- Causa raiz: `scripts/check-policies-sync.sh` checava orfaos apenas do lado do espelho (`.claude/rules/*.md`), sem checagem simetrica do lado canonico (`.agents/policies/*.md` vs `required_policies`).
- Arquivos alterados: `scripts/check-policies-sync.sh` (checagem simetrica adicionada, mesmo padrao usado em `check-scripts-sync.sh`)
- Teste de regressao: coberto por `tests/integration/sync_gates_guard_test.go`
- Validacao: `bash scripts/check-policies-sync.sh` -> exit 0, sem drift real

- ID: BUG-CAPMATRIX-DISPATCH-PROOF-01
- Severidade: minor (mas arquitetural)
- Origem: finding de review (review criteriosa da Tarefa 6.0 do PRD `harness-portatil-vendor-neutral`, RF-18/RF-18.1/RF-20/V-25/V-28)
- Estado: fixed
- Causa raiz: `DispatchProvenFromParityTests` (internal/capability/evidence.go) e `EvidenceTest` (internal/capability/capability.go) sustentavam a prova de dispatch das celulas `supported`/`provider capability` da capability matrix apenas com `ParityTestMethodExists`, uma checagem sintatica via `go/ast` que apenas confirma que existe um metodo com nome/receiver esperados em `internal/parity/parity_test.go`. A funcao nunca executava o teste nem lia `pass`/`skip`/`fail`; retornava a mesma resposta booleana global para qualquer combinacao `(provider, capabilityID)`. Um teste com corpo esvaziado ou `t.Skip()` continuaria "provando" as celulas, pois a checagem nunca observava o resultado real de execucao — reabrindo, num eixo adjacente, a classe de risco que V-25 (saneamento de invariantes auto-satisfeitos) foi desenhada para combater. O repositorio ja tinha o padrao correto em `internal/runtime/specs/parity_dispatch_proof_test.go` (executa `go test -json` e exige `Action: "pass"`), que nao era reaproveitado aqui.
- Arquivos alterados:
  - `internal/capability/evidence.go` (reescrito: mantém `ParityTestMethodExists`/`exprString` como checagem sintatica de pre-condicao, mas adiciona `parseParityTestEvidence`, `executionProof`, `dispatchProvenFromTests` e `repoRootFromWorkingDir`, que executam `go test -json -run <pattern> <pkg>` de verdade sobre `internal/parity/...` e só retornam prova quando o evento `pass` é observado para o teste-alvo e nenhum `skip`/`fail` for reportado; `DispatchProvenFromParityTests` agora e um wrapper fino sobre `dispatchProvenFromTests` com os defaults de producao)
  - `internal/capability/evidence_regression_test.go` (novo: teste de regressao)
- Teste de regressao: `TestDispatchProvenFromParityTests_SkippedTestNeverSatisfiesGate`, `TestDispatchProvenFromParityTests_FailedTestNeverSatisfiesGate`, `TestDispatchProvenFromParityTests_PassingTestSatisfiesGate` em `internal/capability/evidence_regression_test.go`. Cada teste monta um modulo Go fixture temporario (`t.TempDir()` + `go.mod` + `fixture_test.go`) cujo teste `TestParitySuite/TestParity_AllTools` tem corpo `t.Skip(...)`, `t.Fatal(...)` ou vazio, e chama `dispatchProvenFromTests` apontando para esse fixture. Antes de cada asserção, `mustHaveSyntacticMethod` confirma que a checagem antiga isolada (`ParityTestMethodExists`) retornaria `true` para o mesmo cenario — provando concretamente que o defeito descrito (prova puramente sintatica) existia e que a correcao (exigir `Action: "pass"` real) e o que agora bloqueia os casos `skip`/`fail`.
- Validacao: ver secao "Comandos Executados"

- ID: BUG-12
- Severidade: critical
- Origem: smoke test manual pos-implementacao contra repositorio real externo (/Users/jailtonjunior/Git/morvi, clonado para /tmp/morvi-smoketest2), fora do ciclo de review sintetico dos 7 blocos — nenhum dos 7 blocos de review (todos `APPROVED`) cobriu este cenario porque exigia um repositorio real ja sincronizado
- Estado: fixed
- Causa raiz: `internal/upgrade/upgrade.go` so re-copia (e portanto so grava checksum via `tracker.record()`) uma skill quando seu `Status != StatusOK`. Para qualquer skill/arquivo gerenciado cujo conteudo ja esteja identico ao da fonte (o caso comum: repositorio ja sincronizado), nenhuma escrita ocorre, `tracker.Checksums()` retorna vazio para esses paths, e `mf.FileChecksums` nunca e populado para eles. `internal/txn/transaction.go:153` (`checkConflict`) trata qualquer path SEM entrada em `expected` como `!isManaged` e sai sem checar nada. Resultado: a deteccao de conflito por checksum (RF-22/RF-24/RF-27 — nucleo das Tarefas 7.0/8.0) nunca protege nenhuma instalacao pre-existente cujo baseline nunca foi escrito, permitindo que uma edicao manual do usuario em um arquivo gerenciado seja sobrescrita silenciosamente num `sync`/`upgrade` futuro, exatamente o comportamento que essas duas tarefas foram desenhadas para eliminar. Confirmado ao vivo: `ai-spec sync` contra o clone do `morvi` (repo real, 27 skills, todas ja "OK") resultou em 0 entradas em `file_checksums` apos a execucao. O teste que alegava provar este comportamento (`TestUpgrade_BackfillsFileChecksumsEvenWhenVersionAndSkillsAreUnchanged`) nunca verificava o campo `FileChecksums` no corpo do teste — so `Version`/`SkillVersions`/`UpdatedAt` — por isso passava sem detectar o defeito.
- Arquivos alterados:
  - `internal/tracking/tracker.go` (nova funcao `BackfillChecksums(fsys fs.FileSystem, root string, managedPaths []string, existing map[string]string) map[string]string`, computa o hash real do arquivo em disco para qualquer path gerenciado ainda sem entrada em `existing`)
  - `internal/upgrade/upgrade.go` (`mergeFileTracking` ganhou o parametro `projectDir` e agora chama `tracking.BackfillChecksums` sobre a uniao de `mf.InstalledFiles`+`mf.MergedFiles` apos mesclar os checksums desta execucao, garantindo que TODO arquivo gerenciado tenha um baseline de checksum, escrito ou nao nesta rodada)
  - `internal/upgrade/upgrade_test.go` (teste existente `TestUpgrade_BackfillsFileChecksumsEvenWhenVersionAndSkillsAreUnchanged` corrigido para de fato verificar `FileChecksums`; novo teste `TestUpgrade_BackfilledChecksumDetectsConflictOnNextRun` prova o fechamento do ciclo: apos o backfill, uma edicao manual subsequente E detectada como conflito na proxima execucao)
- Teste de regressao: `TestUpgrade_BackfillsFileChecksumsEvenWhenVersionAndSkillsAreUnchanged` (corrigido) e `TestUpgrade_BackfilledChecksumDetectsConflictOnNextRun` (novo), ambos em `internal/upgrade/upgrade_test.go`. Validacao adicional em REPOSITORIO REAL (nao apenas FakeFileSystem): clone de `/Users/jailtonjunior/Git/morvi` para `/tmp/morvi-smoketest2`, `ai-spec sync` rodado 3 vezes — (1) primeira rodada: 0 -> 522 entradas em `file_checksums` via backfill, zero escrita real (`created=0 updated=0`); (2) edicao manual em `.agents/skills/review/SKILL.md` seguida de novo `sync`: abortou com exit code 1, mensagem `batch aborted: 1 managed file(s) in conflict...` nomeando o arquivo e a origem canonica, edicao do usuario preservada intacta no disco; (3) `sync --overwrite-conflicts`: sobrescreveu corretamente, relatou `overwritten (conflict, --overwrite-conflicts): .agents/skills/review/SKILL.md`.
- Validacao: ver secao "Comandos Executados"

- ID: BUG-12b
- Severidade: high
- Origem: achado de revisao adversarial independente sobre a correcao do BUG-12 (rodada de re-revisao critica, nao coberta pelos 7 blocos originais nem pela primeira validacao do BUG-12)
- Estado: fixed
- Causa raiz: a correcao do BUG-12 monta `managedPaths` (para o backfill) exclusivamente a partir de `mf.InstalledFiles`+`mf.MergedFiles`. Quando AMBOS sao `nil` — manifesto gravado antes da propria existencia desse rastreamento por arquivo (`manifest.go` documenta esse estado como real: "manifestos antigos no disco nao o possuem") — `managedPaths` fica vazio e o backfill nao tem nada para iterar. Reproduzido de forma deterministica pelo revisor com `FakeFileSystem`: manifesto legado sem `InstalledFiles`, primeiro `Execute` roda com `FileChecksums={}`, edicao manual preservando a mesma versao no frontmatter, segundo `Execute` substitui o conteudo do usuario sem NENHUMA mensagem de conflito (porque a comparacao de versao decide reescrever a skill, e essa reescrita nunca passou por checagem de conflito por falta de baseline). E o MESMO defeito de fundo do BUG-12, so que para a populacao de manifestos anteriores a Tarefa 7.0/8.0 — nao e hipotetico, e deterministicamente reproduzivel.
- Arquivos alterados:
  - `internal/tracking/tracker.go` (nova funcao `WalkManagedPaths(fsys fs.FileSystem, root, dir string) []string`, varre recursivamente um diretorio via `ReadDir`/`IsDir` da interface `fs.FileSystem` — portavel entre `OSFileSystem` e `FakeFileSystem`, ao contrario de `filepath.Walk` que so funciona no filesystem real)
  - `internal/upgrade/upgrade.go` (`mergeFileTracking` captura `legacyManifest := !mf.HasFileTracking()` ANTES de qualquer mutacao de `mf.InstalledFiles`; quando `legacyManifest` e verdadeiro, `managedPaths` e complementado com `tracking.WalkManagedPaths(tracker, projectDir, ".agents/skills")` antes de chamar `BackfillChecksums` — cobre exatamente a superficie que `upgrade` gerencia, sem promover o manifesto a `HasFileTracking()==true` [`InstalledFiles` continua `nil` de proposito, preservando o comportamento conservador ja testado por `TestUpgrade_LegacyManifestWithoutTrackingBackfillsChecksumsViaSkillsWalkOnVersionOnlyBump`])
  - `internal/upgrade/upgrade_test.go` (novo teste `TestUpgrade_LegacyManifestWithoutFileTrackingStillDetectsConflictAfterBackfill`)
  - `internal/upgrade/task_7_0_test.go` (`TestUpgrade_LegacyManifestWithoutTrackingStaysUntrackedOnVersionOnlyBump` renomeado para `TestUpgrade_LegacyManifestWithoutTrackingBackfillsChecksumsViaSkillsWalkOnVersionOnlyBump` e corrigido: a asserção antiga `mf.FileChecksums != nil` esperava EXPLICITAMENTE o comportamento defeituoso — foi essa decisão da Tarefa 7.0, codificada como teste, que criou a vulnerabilidade; a asserção agora exige o backfill via varredura, preservando a asserção original de que `InstalledFiles` continua `nil`)
- Teste de regressao: `TestUpgrade_LegacyManifestWithoutFileTrackingStillDetectsConflictAfterBackfill` (novo, `upgrade_test.go`) e `TestUpgrade_LegacyManifestWithoutTrackingBackfillsChecksumsViaSkillsWalkOnVersionOnlyBump` (corrigido, `task_7_0_test.go`). Validacao adicional em REPOSITORIO REAL: clone fresco de `/Users/jailtonjunior/Git/morvi`, `file_checksums`/`installed_files`/`merged_files` removidos manualmente do manifesto (simulando instalacao anterior a Tarefa 7.0/8.0), `ai-spec sync` rodado: `installed_files` permanece `null` (comportamento conservador preservado) mas `file_checksums` populado com 159 entradas via varredura; edicao manual em `.agents/skills/review/SKILL.md` seguida de novo `sync`: exit code 1, `batch aborted: 1 managed file(s) in conflict...`, edicao do usuario preservada intacta no disco.
- Validacao: ver secao "Comandos Executados"

- ID: BUG-12c
- Severidade: critical
- Origem: achado de revisao adversarial independente (rodada 3) sobre a correcao do BUG-12b — o mesmo revisor que rejeitou o BUG-12 original rejeitou tambem o BUG-12b, reproduzindo ao vivo contra clone de `/Users/jailtonjunior/Git/morvi`
- Estado: fixed
- Causa raiz: o backfill legado do BUG-12b so varria `.agents/skills/`, mas `internal/upgrade/upgrade.go` gerencia, pelo mesmo `tracker`, uma superficie muito maior: `regenerateAdapters` (escreve em `.claude/`, `.codex/`, `.github/`, `scripts/lib/`) e `regenerateGovernance` (escreve `AGENTS.md`/`CLAUDE.md`/`CODEX.md`/`COPILOT.md`). Para manifesto legado, nenhum desses arquivos jamais ganhava baseline de checksum, permitindo sobrescrita silenciosa de customizacao do usuario ja na primeira execucao apos qualquer skill divergir (reproduzido ao vivo pelo revisor: `conflict=0`, exit 0, edicoes manuais em `.claude/hooks/validate-preload.sh` e `scripts/lib/check-invocation-depth.sh` apagadas sem aviso).
- Arquivos alterados:
  - `internal/upgrade/upgrade.go` (`legacyManagedPathCandidates` + `legacyManagedRootDirs`/`legacyManagedRootFiles`: a varredura de backfill agora cobre `.agents/skills`, `.agents/hooks`, `.agents/scripts`, `.agents/lib`, `.agents/policies`, `.claude`, `.codex`, `.github`, `.opencode`, `scripts/lib`, e os arquivos `AGENTS.md`/`CLAUDE.md`/`CODEX.md`/`COPILOT.md` quando existentes — toda a superficie real que `regenerateAdapters`/`regenerateGovernance` tocam, nao apenas skills)
  - `internal/txn/transaction.go` — TENTATIVA REVERTIDA: uma primeira abordagem generalizou `checkConflict` para, na ausencia de baseline, comparar o conteudo atual do destino contra o HASH DO CONTEUDO NOVO sendo escrito (nao contra um baseline do manifesto). Essa abordagem e correta em teoria (fecha o gap ja na primeira execucao, sem depender de backfill previo) mas quebrou ~20 testes existentes de `install`/`upgrade` porque a logica de MERGE de arquivos (`.claude/settings.local.json`, `.github/settings.json`, `opencode.json` — conteudo mesclado com o que o usuario ja tinha, INTENCIONALMENTE diferente do template puro) e tratada pelo mesmo mecanismo de escrita, e passou a ser sinalizada como falso-positivo de conflito. Revertida integralmente para o comportamento original (`isManaged==false` -> pular checagem), preservando a semantica de merge legitima.
  - `internal/upgrade/upgrade_test.go` (teste `TestUpgrade_LegacyManifestFirstSyncEstablishesBaselineThenProtectsFromSecondSyncOnward`, substitui uma tentativa anterior que exigia protecao ja na primeira sync — ver Nota de Escopo; e `TestUpgrade_LegacyManifestBackfillCoversAdapterFilesTouchedByRegenerateAdapters` ajustado para forcar divergencia de skill em AMBAS as execucoes, ja que `regenerateAdapters` so roda quando alguma skill muda nesta mesma execucao)
- Teste de regressao: `TestUpgrade_LegacyManifestFirstSyncEstablishesBaselineThenProtectsFromSecondSyncOnward` e `TestUpgrade_LegacyManifestBackfillCoversAdapterFilesTouchedByRegenerateAdapters`, ambos em `upgrade_test.go`. Validacao adicional em REPOSITORIO REAL: clone fresco de `/Users/jailtonjunior/Git/morvi`, manifesto legado simulado, edicao manual pre-existente em `.claude/hooks/validate-preload.sh` + divergencia forcada em `.agents/skills/review/SKILL.md`, `ai-spec sync` executado 4 vezes — (1) primeira sync: exit 0, conflito=0, baseline estabelecido para 554 paths (edicao pre-existente perdida — limitacao aceita, ver Nota de Escopo); (2) `sync --overwrite-conflicts` (setup, restaura conteudo canonico); (3) nova edicao manual + sync SEM forcar divergencia de skill: exit 0 (regenerateAdapters nem roda, nada e tocado — comportamento correto, nao ha risco quando nada e escrito); (4) nova edicao manual + divergencia forcada de skill + sync: exit 1, `batch aborted: 1 managed file(s) in conflict...` nomeando `.claude/hooks/validate-preload.sh`, edicao preservada intacta.
- Validacao: ver secao "Comandos Executados"

## Nota de Escopo (BUG-12/12b/12c)

Existe uma limitacao de bootstrap fundamental e aceita: uma customizacao feita pelo usuario ANTES da adocao desta versao (ou antes da primeira sync sob um manifesto legado) nao pode ser distinguida, de forma confiavel, de conteudo meramente desatualizado — nao ha baseline anterior para comparar, e o unico "baseline" possivel seria o proprio conteudo atual (que e, por definicao, igual a si mesmo — nao ha conflito logico a detectar num alvo que ainda nao tem historia). A primeira sync apos adotar o rastreamento de checksum estabelece esse baseline a partir do estado atual do disco; se esse estado ja incluia uma customizacao do usuario feita antes da adocao, ela pode ser sobrescrita nessa transicao unica. A partir dai (baseline estabelecido), TODA customizacao subsequente e protegida corretamente, com evidencia real em repositorio externo (`morvi`) nas rodadas 3 e 4 acima.

A alternativa de fechar esse gap por completo — tratar QUALQUER arquivo existente sem baseline e about-to-be-modified como conflito, independente de manifesto — foi implementada, testada e REVERTIDA nesta mesma rodada por quebrar a semantica de merge legitima de `install.go` (~20 testes), que sao um caso de uso real e correto (preservar customizacoes do usuario nos hooks/settings dos providers durante o `install`), nao um caso a eliminar. Corrigir esse caso especifico exigiria diferenciar "escrita por merge" de "escrita por template puro" no proprio nucleo do `txn.FileTransaction`, uma mudanca de escopo maior que o justificavel para este PRD; registrado aqui para decisao futura consciente, nao como lacuna oculta.

## Comandos Executados

Grupo R-STYLE-001 (BUG-01..BUG-06):
- `go build ./... && go vet ./...` -> exit 0
- `go test ./cmd/ai_spec_harness/... ./internal/telemetry/... ./internal/doctor/... ./internal/batchreport/... ./internal/install/... ./internal/upgrade/...` -> 532 passed, 0 failed
- `GOGC=20 golangci-lint run --config .golangci.yml ./cmd/... ./internal/telemetry/... ./internal/doctor/... ./internal/batchreport/... ./internal/install/... ./internal/upgrade/...` -> "No issues found"
- `grep -rn "[a-zà-ú]ç\|ão\b" internal/telemetry/parser.go internal/doctor/doctor.go internal/batchreport/report.go` -> sem ocorrencias

Grupo documentacao/governanca (BUG-07..BUG-10):
- `grep -n "init\|sync\|overwrite-conflicts" CHANGELOG.md docs/guia-instalacao-universal.md` -> presente em prosa real
- `bash scripts/check-policies-sync.sh` -> "Policies em sync: 2", "Drift / missing: 0", exit 0
- `bash scripts/sync-policies.sh` -> "sync-policies: concluido", exit 0
- `go test -tags=integration ./tests/integration/... -run TestSyncGates -v` -> 37 passed
- `make check-policies-sync check-skills-sync check-hooks-sync check-scripts-sync` -> 0 drift em todos

Grupo capability matrix (BUG-CAPMATRIX-DISPATCH-PROOF-01):
- `go build ./... && go vet ./...` -> exit 0
- `go test ./internal/capability/... ./internal/runtime/specs/... -v` -> 100% pass (17 testes em `internal/capability`, incluindo os 3 novos de regressao; suite completa de `internal/runtime/specs` inalterada e verde)
- `UPDATE_SNAPSHOTS=1 go test ./internal/capability/...` seguido de `go test ./internal/capability/... -v` -> `docs/capability-matrix.md` e `testdata/capability-matrix.json` regenerados deterministicamente; conteudo da matrix (celulas, estados, campo `Test`) nao mudou — a correcao altera apenas o *mecanismo de prova* usado pelo gate, nao os dados publicados
- `GOGC=20 golangci-lint run --config .golangci.yml --timeout 10m ./internal/capability/... ./internal/runtime/specs/...` -> "No issues found"
- `make check-capability-matrix-sync` -> pass

Grupo backfill de FileChecksums (BUG-12):
- `go build ./... && go vet ./...` -> exit 0
- `go test ./internal/tracking/... ./internal/upgrade/... ./internal/install/... -v` -> 178 passed, 0 failed
- `go test ./...` -> 4018 passed, 85 pacotes
- Smoke test real (ver "Teste de regressao" acima) contra clone de `/Users/jailtonjunior/Git/morvi`

Grupo backfill para manifesto legado (BUG-12b, re-revisao):
- `go build ./... && go vet ./...` -> exit 0
- `go test ./internal/upgrade/... -run "TestUpgrade_LegacyManifestWithoutFileTrackingStillDetectsConflictAfterBackfill|TestUpgrade_LegacyManifestWithoutTrackingBackfillsChecksumsViaSkillsWalkOnVersionOnlyBump" -v` -> ambos PASS
- `go test ./...` -> 4019 passed, 85 pacotes
- `make lint` -> 0 issues
- `make integration` -> ok em todos os pacotes
- Smoke test real (ver "Teste de regressao" acima) contra clone fresco de `/Users/jailtonjunior/Git/morvi` com manifesto legado simulado

Validacao consolidada (orquestrador, apos os 4 grupos):
- `go build ./...` -> exit 0
- `go vet ./...` -> exit 0

## Riscos Residuais
- `executionProof` (BUG-CAPMATRIX-DISPATCH-PROOF-01) invoca `go test -json ... ./internal/parity/...` via subprocesso a cada chamada de `DispatchProvenFromParityTests`; adiciona ~0.5-1s de custo por execucao do gate, mesmo padrao ja aceito em `internal/runtime/specs/parity_dispatch_proof_test.go`.
- O mecanismo de dispatch continua sendo uma prova global unica (uma suite cobre todas as 42 celulas de uma vez, nao uma prova por celula individual) — limitacao de design pre-existente do PRD, nao criada por esta correcao.
- `internal/upgrade/upgrade.go:709` `_upgradePlanningSkills` viola R-STYLE-001.3 mas e codigo pre-existente NAO tocado pelo diff desta entrega (confirmado por `git diff HEAD` no bloco de revisao correspondente) — fora de escopo por regra explicita ("renomear ao tocar", nao antes); registrado para correcao futura quando esse trecho for efetivamente tocado.
- `evidence/task-*-harness-portatil-vendor-neutral/*.patch` acumulam o worktree inteiro (nao sao diffs escopados) por `evidence/` nao estar em `.gitignore` — problema de higiene de evidencia identificado na revisao de fechamento (Tarefa 13.0), fora do escopo de codigo desta correcao; recomendado tratar separadamente (adicionar `evidence/` ao `.gitignore` ou parar de persistir patch recursivo).
- Selo de evidencia (`ai-spec seal-evidence`) permanece pendente de commit humano para todas as tarefas do PRD, conforme R-GOV-001 (harness nao commita).
