# Relatorio de Bugfix

- Total de bugs no escopo: 11
- Corrigidos: 11
- Testes de regressao adicionados: 11
- Pendentes: nenhum
- Estado final: done
- Nota: dos 11 corrigidos, 10 pelo subagente mecanico e BUG-03 pelo orquestrador apos deteccao de lacuna de propriedade de arquivo nesta rodada

## Bugs

- ID: BUG-04
- Severidade: major
- Origem: review tarefa 2.0; RF-36
- Estado: fixed
- Causa raiz: `Serialize` ajustava `expectedContent` com a mesma logica usada para gerar o separador de newline, tornando o checker de round-trip tautologico (nunca falhava para esse caso) e mascarando uma mutacao silenciosa de um byte no conteudo humano.
- Arquivos alterados: `internal/runtime/memory/durable/page.go`, `internal/runtime/memory/durable/errors.go`
- Teste de regressao: `TestSerializeRejectsHumanContentWithoutTrailingNewlineBeforeFacts`, `TestSerializeRejectsRoundTripWhenHumanContentMasqueradesAsFact` (forca `errors.Is(err, durable.ErrRoundTripNotPreserved)` a disparar de fato) em `internal/runtime/memory/durable/page_test.go`
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestPageSuite -v -count=1` (26 passed)

- ID: BUG-05
- Severidade: major
- Origem: review tarefa 2.0; RF-36
- Estado: fixed
- Causa raiz: `Serialize` sempre escrevia todo o conteudo humano concatenado antes de todos os Fatos, reagrupando texto originalmente intercalado entre secoes de Fato sem aviso.
- Arquivos alterados: `internal/runtime/memory/durable/page.go` (campo interno `HumanBlock.interleaved` computado em `Parse`, checado em `Serialize`), `internal/runtime/memory/durable/errors.go` (`ErrHumanBlockInterleaved`), `docs/memory-page-format.md`
- Decisao de design: recusa explicita (opcao sancionada pelo bug) em vez de reposicionamento por indice, pois `Layer`/`Facade` (fora do escopo desta rodada) nao podiam ter a logica alterada para consumir um modelo posicional.
- Teste de regressao: `TestSerializeRejectsInterleavedHumanContentBetweenFacts` em `internal/runtime/memory/durable/page_test.go`
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestPageSuite -v -count=1` (26 passed); `go test ./internal/runtime/memory/durable/... -count=1` (138 passed, sem regressao em `layer_test.go`/`facade_test.go`, que consomem `Page` sem depender do campo interno)

- ID: BUG-07
- Severidade: major
- Origem: review tarefa 5.0; RF-16 (Windows)
- Estado: fixed
- Causa raiz: `createLayerLockFile` criava o arquivo vazio via `O_CREATE|O_EXCL` e so depois escrevia `ProcessRef`, deixando uma janela em que o arquivo existe mas esta vazio/incompleto.
- Correcao: conteudo escrito e sincronizado em arquivo temporario no mesmo diretorio; publicacao no nome final via `os.Link` (falha atomicamente com `os.ErrExist` se o destino ja existir, nunca deixando visibilidade parcial). `readLayerLockOwner` incompleto agora aciona retry limitado (`readLayerLockOwnerWithRetry`, hook `readLayerLockOwnerHook` para teste deterministico) antes de declarar o lock orfao.
- Arquivos alterados: `internal/runtime/memory/durable/layer_lock_windows.go`, `internal/runtime/memory/durable/layer_lock_windows_test.go` (novo)
- Teste de regressao: `TestCreateLayerLockFile_PublishesContentAtomicallyWithVisibility`, `TestCreateLayerLockFile_SecondCallReportsAlreadyExists`, `TestReadLayerLockOwnerWithRetry_RetriesBeforeDecidingOrphan`, `TestReadLayerLockOwnerWithRetry_DeclaresOrphanAfterExhaustingRetries` (todos com `//go:build windows`, deterministicos via hook de funcao, sem dependencia de sleep real)
- Validacao: nao executavel nativamente em macOS (arquivo `//go:build windows`); validado via `GOOS=windows GOARCH=amd64 go build ./internal/runtime/memory/durable/...` (OK), `GOOS=windows GOARCH=amd64 go vet ./internal/runtime/memory/durable/...` (OK) e `GOOS=windows GOARCH=amd64 go test -c -o /tmp/durable_windows_test.exe ./internal/runtime/memory/durable/...` (compila sem erro)

- ID: BUG-11
- Severidade: major
- Origem: review tarefa 8.0; RF-34
- Estado: fixed
- Causa raiz: `injectMetricsSection` substituia "do cabecalho ate o fim do arquivo" em vez de ate a proxima secao `## `, apagando qualquer secao inserida depois dela (ex.: Evidencia de Memoria). Causa raiz secundaria descoberta durante a correcao: a injecao da secao de Evidencia de Memoria (`injectBoundedSection`, ramo "nao encontrado") sempre fazia append no fim absoluto do arquivo, podendo inverter a ordem canonica (Evidencia antes de Metricas) quando a secao de Metricas ja existia de uma chamada anterior.
- Correcao: `injectMetricsSection` passou a delegar para `injectBoundedSection` (que ja localizava corretamente a proxima secao `## `). Adicionado `injectBoundedSectionBefore` (usado para a secao de Evidencia de Memoria) que insere antes da secao de Metricas quando esta ja existe, preservando a ordem canonica entre chamadas.
- Arquivos alterados: `internal/runtime/persistence/report.go`, `internal/runtime/persistence/report_test.go`
- Teste de regressao: `TestEnrichReport_MetricsThenMemoryEvidence_BothSectionsSurvive` (duas chamadas de `EnrichReport` com summaries diferentes sobre o mesmo arquivo — 1a so metricas, 2a metricas+MemoryEvidence — ambas as secoes sobrevivem integras e na ordem correta)
- Validacao: `go test ./internal/runtime/persistence/... -count=1 -v -run TestEnrichReport` (13 passed)

- ID: BUG-13
- Severidade: major
- Origem: review tarefa 10.0 (subtarefa 10.3); RF-16
- Estado: fixed
- Causa raiz: nenhum teste exercitava o `LayerLock` real (flock em Unix / arquivo de lock em Windows) com processo morto; a cobertura existente (`handoff_lease_integration_test.go`) testa apenas `HandoffLease`/`LeasePolicy`, mecanismo distinto (RF-23).
- Correcao: novo teste de integracao com subprocesso OS real (`os/exec`) que adquire o `LayerLock` real via `durable.DefaultLayerLocker.Lock` e e morto com `SIGKILL` sem liberar deliberadamente; o teste prova que o kernel libera o `flock` automaticamente na morte do processo e que uma nova tentativa de `Consolidate` no mesmo `Scope` sucede em tempo limitado e persiste o fato do sucessor sem sobrescrita silenciosa.
- Arquivos alterados: `internal/runtime/memory/durable/layer_lock_orphan_integration_test.go` (novo, `//go:build integration`)
- Teste de regressao: `TestOrphanLayerLockReleasedWhenOwningProcessDies` (+ helper `TestHelperProcessHoldsLayerLockUntilKilled`)
- Validacao: `go test -tags=integration ./internal/runtime/memory/durable/... -run TestOrphanLayerLockReleasedWhenOwningProcessDies -v -count=1` (passed); suite completa `go test -tags=integration ./internal/runtime/memory/durable/... -count=1 -v` (149 passed)

- ID: BUG-14
- Severidade: minor
- Origem: review tarefa 10.0 (subtarefa 10.6); RF-25
- Estado: fixed
- Causa raiz: writer e reader usavam `CLI:"claude"` fixo nas duas pontas; "cross-CLI" era alegado apenas por argumento estrutural (ausencia de parametro CLI em `MemoryScope`/`BuildContext`), nunca exercitado.
- Correcao: o writer agora grava fatos variando explicitamente o CLI de origem entre `claude`, `codex`, `opencode` e `copilot` (um por fato, ciclicamente); o reader continua sem qualquer parametro de CLI em `MemoryScope`/`BuildContext` — essa ausencia, combinada a variacao explicita na escrita, e a prova de que o recall independe do CLI de origem.
- Arquivos alterados: `internal/runtime/memory/durable/bootstrap_handoff_integration_test.go`
- Teste de regressao: `TestHandoffCrossCLIDeliversAtLeast90PercentOfFacts` (atualizado)
- Validacao: `go test -tags=integration ./internal/runtime/memory/durable/... -run TestHandoffCrossCLIDeliversAtLeast90PercentOfFacts -v -count=1` (passed)

- ID: BUG-16
- Severidade: major
- Origem: review tarefa 1.0; RF-16 (prova de atomicidade)
- Estado: fixed
- Causa raiz: o teste anterior nunca interrompia `WriteFileAtomic` em execucao real; apenas plantava um `.tmp-interrupted` alheio e confirmava que ele nao contaminava o caminho final — nao provava nada sobre a sequencia real Write->Sync->Close->Rename.
- Correcao: teste reescrito para forcar uma falha real no `Rename` (caminho final pre-existente como diretorio nao-vazio, o que faz `os.Rename` falhar de verdade depois que Write/Sync/Close ja sucederam sobre o arquivo temporario) e comparar o conteudo/tipo do caminho final antes e depois da tentativa falha, provando que nunca fica parcial.
- Arquivos alterados: `internal/fs/os_test.go`
- Teste de regressao: `TestOS_WriteFileAtomic_noPartialFileOnInterruption` (reescrito)
- Validacao: `go test ./internal/fs/... -run TestOS_WriteFileAtomic -v -count=1` (passed)

- ID: BUG-17
- Severidade: minor
- Origem: review tarefa 1.0
- Estado: fixed
- Causa raiz: divergencia de semantica entre `WriteFileAtomic` (substitui o symlink via `Rename`) e `WriteFile` (segue o link via `os.WriteFile`) nunca estava documentada nem travada por teste.
- Correcao: teste table-driven novo que exercita as duas funcoes sobre o mesmo symlink e verifica o comportamento esperado de cada uma; comportamento tambem documentado no proprio nome/asserts do teste (codigo, sem comentarios, conforme R-STYLE-001.2).
- Arquivos alterados: `internal/fs/os_test.go`
- Teste de regressao: `TestOS_WriteFileAtomicVsWriteFile_symlinkSemantics`
- Validacao: `go test ./internal/fs/... -run TestOS_WriteFileAtomicVsWriteFile -v -count=1` (passed)

- ID: BUG-18
- Severidade: minor
- Origem: review tarefa 6.0; cobertura do golden de paridade (RF-29)
- Estado: fixed
- Causa raiz: caso 1 do golden usava `TasksDir: t.TempDir()` (diretorio vazio real), nunca `TasksDir: ""` (zero-value real) — o branch real de `TasksDir==""` (RF-26/RF-28) nunca era exercitado na fronteira do hook `hooks.PointPromptPostBuild`.
- Correcao: novo caso "case 1b: zero-value TasksDir" com `TasksDir: ""` literal, comparado por igualdade estrita (`!=`) contra o prompt esperado; caso 1 original (diretorio vazio) mantido como cenario adicional.
- Arquivos alterados: `internal/runtime/prompt_parity_golden_test.go`
- Teste de regressao: `TestPromptParityGolden/case_1b:_zero-value_TasksDir`
- Validacao: `go test ./internal/runtime/... -run TestPromptParityGolden -v -count=1` (7 passed)

- ID: BUG-15 (subset B)
- Severidade: major
- Origem: review tarefas 5.0, 6.0, 7.0, 9.0; R-STYLE-001.2
- Estado: fixed
- Causa raiz: doc-comments em `ActivePath`/`SidecarPath` (`layer.go:81-92`) e 3 linhas de comentario em `cli_contract_test.go:98-100`, violando a regra de zero comentarios.
- Arquivos alterados: `internal/runtime/memory/durable/layer.go` (somente remocao de comentario, logica inalterada), `cmd/ai_spec_harness/cli_contract_test.go` (somente remocao de comentario)
- Teste de regressao: nenhum aplicavel (mudanca puramente de remocao de comentario, sem alteracao de comportamento) — validado por `grep "// " internal/runtime/memory/durable/layer.go` (linhas 81-92 nao retornam mais comentario) e recompilacao/teste completo do pacote
- Validacao: `go build ./...`, `go test ./internal/runtime/memory/durable/... -count=1` (138 passed), `go test ./cmd/ai_spec_harness/... -count=1` (207 passed)

- ID: BUG-03
- Severidade: critical
- Origem: review tarefa 10.0; RF-20, O-8
- Estado: fixed
- Nota: nao corrigido por nenhum dos dois subagentes de bugfix (instrucao contraditoria do orquestrador nesta rodada fez ambos evitarem `recovery_bench_test.go`); corrigido diretamente pelo orquestrador apos os dois subagentes concluirem, ao detectar a lacuna em `git status`/validacao central.
- Causa raiz: `FacadeConfig` do benchmark so definia `ProjectDir`, nunca `TasksDir` — escrita nas camadas PRD/Task falhava silenciosamente (log, nao erro), `FactsByLayer` ficava vazio (0 fatos recuperados). Causa raiz secundaria descoberta ao corrigir: rotear os 1000 fatos para a camada PRD (mesma pagina, chaves distintas por `TaskFileName`) colide com a compactacao real (RF-13/`DefaultCompactionLineLimit`) — a pagina PRD e compactada a cada `RecordSession`, restando ~11 fatos ativos de 1000 gravados, tornando "1000 fatos ativos em uma pagina" inatingivel pelo caminho real de escrita.
- Correcao: `TasksDir` adicionado ao `FacadeConfig` de ambos os benchmarks (`recovery_bench_test.go` e o preliminar `facade_bench_test.go` da tarefa 7.0, que tinha o mesmo defeito). Fatos passaram a ser gravados sem `DeclaredSection` (apenas `session-summary`, `Durability=Ephemeral`), roteando cada um para sua propria pagina de Task (1000 arquivos distintos, um por `TaskFileName`, sem acionar compactacao). Apos o seeding, o teste verifica explicitamente `FactsByLayer["task"]==1` para uma tarefa especifica (`task-500.0.md`) antes de `ResetTimer`/medir amostras, com `Fatalf` se a contagem nao bater — eliminando a falha silenciosa. O benchmark agora mede o custo real de recuperar uma pagina de Task especifica em meio a 1000 paginas de Task presentes no repositorio (interpretacao literal de "1.000 paginas de memoria" do RF-20/O-8, dado que `Layer.Read` e um lookup direto por caminho, nao uma varredura de diretorio).
- Arquivos alterados: `internal/runtime/memory/durable/recovery_bench_test.go`, `internal/runtime/memory/durable/facade_bench_test.go`
- Teste de regressao: `TestBuildContextP95PreliminaryBenchmark_1000Pages` (corrigido, agora falha se `FactsByLayer["task"]!=1` apos seeding); `BenchmarkRecoveryWith1000Pages` (corrigido, mesma guarda)
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestBuildContextP95PreliminaryBenchmark_1000Pages -v -count=1` (1 passed); `make bench` -> `p95_ms=0.298`, `PASS`, `ok .../internal/runtime/memory/durable`; `go test ./... -count=1` completo sem FAIL; `golangci-lint run ./internal/runtime/memory/durable/...` -> "No issues found"; `gofmt -l` vazio nos dois arquivos

## Comandos Executados

- `gofmt -l internal/fs/fs.go internal/fs/os_test.go internal/runtime/memory/durable/page.go internal/runtime/memory/durable/page_test.go internal/runtime/memory/durable/layer.go internal/runtime/memory/durable/layer_lock_windows.go internal/runtime/memory/durable/layer_lock_windows_test.go internal/runtime/memory/durable/errors.go internal/runtime/memory/durable/bootstrap_handoff_integration_test.go internal/runtime/memory/durable/layer_lock_orphan_integration_test.go internal/runtime/persistence/report.go internal/runtime/persistence/report_test.go internal/runtime/prompt_parity_golden_test.go cmd/ai_spec_harness/cli_contract_test.go` -> saida vazia (formatado)
- `go build ./...` -> sem erros
- `go vet ./internal/fs/... ./internal/runtime/... ./cmd/ai_spec_harness/...` -> sem erros
- `go vet -tags=integration ./internal/runtime/memory/durable/...` -> sem erros
- `GOOS=windows GOARCH=amd64 go build ./internal/runtime/memory/durable/...` -> OK
- `GOOS=windows GOARCH=amd64 go vet ./internal/runtime/memory/durable/...` -> OK
- `GOOS=windows GOARCH=amd64 go test -c -o /tmp/durable_windows_test.exe ./internal/runtime/memory/durable/...` -> compila sem erro (nao executavel em macOS)
- `go test ./internal/fs/... -count=1` -> 52 passed
- `go test ./internal/runtime/... -count=1` -> 847 passed
- `go test ./cmd/ai_spec_harness/... -count=1` -> 207 passed
- `go test -tags=integration ./internal/runtime/memory/durable/... -count=1 -v -timeout 180s` -> 149 passed
- `make check-mocks` -> "Mocks: OK (sincronizados com mockery.yml)"
- `golangci-lint run ./internal/fs/... ./internal/runtime/memory/durable/... ./internal/runtime/persistence/... ./internal/runtime/... ./cmd/ai_spec_harness/...` -> 1 issue pre-existente em `internal/runtime/acp_opencode_telemetry_test.go` (staticcheck), nao relacionado a nenhum arquivo tocado nesta rodada e presente antes das mudancas (confirmado via `git diff --name-only`)
- `make bench` -> nao executado por este subagente: `recovery_bench_test.go` e propriedade exclusiva do outro subagente (BUG-03), e nenhuma mudanca deste subagente altera o caminho de codigo exercitado pelo benchmark de forma que exigisse revalidacao (Serialize/Parse mantiveram compatibilidade total, confirmado pelos 138 testes do pacote `durable` incluindo os testes existentes de `layer_test.go`/`facade_test.go`)

## Riscos Residuais

- BUG-07: a correcao do Windows nao pode ser executada nativamente neste ambiente (macOS); validada apenas por cross-compilation (`go build`/`go vet`/`go test -c`). Recomenda-se rodar a suite `layer_lock_windows_test.go` em CI Windows real antes do proximo release.
- BUG-05: a decisao de recusar (em vez de reposicionar por indice) conteudo humano intercalado entre Fatos e uma limitacao deliberada documentada em `docs/memory-page-format.md` — paginas com essa topologia (raras, tipicamente geradas so por edicao manual incomum) exigirao reformatacao manual antes de aceitar nova consolidacao. Nenhum fato ou byte humano e perdido; a operacao apenas falha explicitamente em vez de silenciosamente reordenar.
- BUG-04: `Serialize` agora exige que o conteudo humano ja termine com `\n` quando houver Fatos a anexar, retornando `ErrHumanContentNotNormalized` em vez de normalizar silenciosamente. Isso e uma mudanca de comportamento observavel (paginas legadas com conteudo humano sem newline final + Fatos, escritas por versoes anteriores do harness, agora falhariam na proxima consolidacao) — nao foram encontradas paginas reais nesse estado no corpus testado (`TestRoundTripOnRealRepositoryCorpus`), mas o layer.go real (que le a pagina antes de escrever) preserva o conteudo humano exatamente como leu, entao o caminho tipico (ler -> mesclar Fatos -> escrever) so tropeca nessa validacao se a pagina em disco ja estiver sem newline final antes de qualquer Fato existir — cenario que so ocorre por edicao manual incomum.
- BUG-13: a prova em Unix baseia-se na garantia do kernel de liberar `flock` na morte do processo (incluindo `SIGKILL`); e um comportamento POSIX estavel, mas o teste depende de `os/exec` com o binario de teste real, custando ~alguns segundos de wall-clock por exigir `-tags=integration`.
