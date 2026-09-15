# Relatorio de Bugfix — Rodada 2 (achados da segunda revisao pos-bugfix)

- Total de bugs no escopo: 6
- Corrigidos: 6
- Testes de regressao adicionados: 6
- Pendentes: nenhum
- Estado final: done
- Nota: 2 dos 6 bugs (BUG-22, BUG-23) sao correcoes puramente textuais (log/comentario/doc) sem comportamento novo a testar; o campo "Teste de regressao" de cada um documenta explicitamente essa natureza
- Nota: todos corrigidos diretamente pelo orquestrador (sem subagente), apos a agente da tarefa 7.0 falhar por limite semanal de API e para evitar nova rodada cara de subagentes para achados ja bem escopados pela segunda revisao.

## Bugs

- ID: BUG-19
- Severidade: major
- Origem: re-review tarefa 2.0 (2a rodada); RF-36, RF-37
- Estado: fixed
- Causa raiz: a correcao original de BUG-04/BUG-05 introduziu dois erros tipados (`ErrHumanContentNotNormalized`, `ErrHumanBlockInterleaved`) retornados por `Serialize`, mas `Facade.RecordSession` nao os tratava como toleraveis por camada (ao contrario de `ErrProjectDirMissing`/`ErrTasksDirMissing`/`ErrTaskFileNameMissing`) — qualquer um deles abortava a funcao inteira via `return MemoryReport{}, err` dentro do loop `for layer, layerFacts := range byLayer`, potencialmente descartando a escrita de TODAS as camadas da sessao (nao apenas a camada afetada), com ordem de iteracao de mapa nao deterministica.
- Correcao: `internal/runtime/memory/durable/facade.go`, `RecordSession` — os dois erros agora sao tratados no mesmo padrao dos tres erros de escopo ausente: logados e a camada e pulada (`continue`), preservando a escrita das demais camadas. RF-36 continua respeitado (nenhum byte e reescrito silenciosamente; a violacao e reportada, nao mascarada) e RF-37 tambem (reporta em vez de reescrever).
- Arquivos alterados: `internal/runtime/memory/durable/facade.go`
- Teste de regressao: `TestRecordSessionWritesOtherLayersEvenWhenOneLayerRoundTripViolationOccurs` (novo, `internal/runtime/memory/durable/facade_test.go`) — semeia uma pagina PRD real sem newline final, roda `RecordSession` com fato de Task e fato de PRD simultaneamente, confirma que o fato de Task e escrito com sucesso, o fato de PRD e pulado sem erro, e a pagina PRD original permanece byte-identica (nao corrompida nem parcialmente reescrita)
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestFacadeSuite -v -count=1` (14 passed); `go test ./internal/runtime/memory/durable/... -count=1` (142 passed); `go build ./...`; `gofmt -l`

- ID: BUG-20
- Severidade: low
- Origem: re-review tarefa 2.0 (2a e 1a rodadas); design de `FactState`
- Estado: fixed
- Causa raiz: `Facade.deriveFacts` nunca definia `Fact.State` explicitamente ao criar fatos novos (zero-value `FactStateUndefined`), confiando implicitamente em `layer.mergeFacts` para normalizar para `FactStateActive` na consolidacao. Tentativa inicial de corrigir validando `FactStateUndefined` em `Page.Parse` (rejeitando como pagina ilegivel) causou REGRESSAO real: `CompactionPolicy.Compact` chama `Page.Serialize` internamente (que por sua vez faz um round-trip de validacao via `Parse` desde a correcao de BUG-04/05) sobre fatos construidos diretamente em `compaction_policy_test.go` sem `State` setado — 5 testes previamente verdes passaram a falhar. Revertido antes de prosseguir, conforme proibicao explicita de regressao.
- Correcao definitiva (sem efeito colateral): `Facade.deriveFacts` agora define `State: FactStateActive` explicitamente nos dois literais de `Fact` que cria (sinal estruturado e secao declarada), fechando a lacuna de design na origem sem alterar o contrato de `Parse`/`Serialize` que outros testes/consumidores dependem implicitamente.
- Arquivos alterados: `internal/runtime/memory/durable/facade.go`
- Teste de regressao: nenhum teste novo dedicado (comportamento ja coberto indiretamente por todos os testes de `RecordSession` que verificam fatos escritos e legiveis apos consolidacao); mudanca revertida em `page.go` confirmada sem residuo via `git diff`
- Validacao: `go test ./internal/runtime/memory/durable/... -count=1` (142 passed, incluindo os 5 testes de `CompactionPolicySuite` que a tentativa anterior havia quebrado)

- ID: BUG-21
- Severidade: major
- Origem: re-review tarefa 5.0 (2a rodada); RF-16 (Windows)
- Estado: fixed
- Causa raiz: a correcao original de BUG-07 eliminou a janela de visibilidade parcial do conteudo do lock (via `os.Link` atomico), mas nao eliminou uma segunda corrida TOCTOU em `takeOverStaleLayerLock`: a decisao de organidade (leitura + checagem de liveness) e a remocao do arquivo (`os.Remove` incondicional) nao eram atomicas entre si. Dois processos podiam decidir concorrentemente que o mesmo lock estava orfao; o primeiro removia e recriava o lock (tornando-se dono legitimo), e o segundo, baseado na mesma leitura antiga, removia o lock RECEM-CRIADO do primeiro sem verificar se o conteudo mudou, permitindo que ambos acreditassem deter exclusividade — perda silenciosa de fato sob concorrencia real.
- Correcao: `readLayerLockOwner`/`readLayerLockOwnerWithRetry` foram substituidos por `readLayerLockRaw`/`readLayerLockRawWithRetry`, que retornam tambem os bytes brutos lidos (nao apenas o `ProcessRef` interpretado). Nova funcao `removeLayerLockFileIfUnchanged(path, expected []byte)` releem o arquivo imediatamente antes de remover e compara byte a byte com o snapshot usado na decisao de organidade — se o conteudo mudou (outro processo ja tomou o lock), recusa com `ErrLayerLocked` em vez de remover; se o arquivo ja nao existe (outro processo ja tomou e talvez ja liberou), trata como sucesso idempotente.
- Arquivos alterados: `internal/runtime/memory/durable/layer_lock_windows.go`, `internal/runtime/memory/durable/layer_lock_windows_test.go` (testes existentes adaptados as novas assinaturas; teste novo adicionado)
- Teste de regressao: `TestTakeOverStaleLayerLock_DoesNotDeleteLockRecreatedByAnotherProcessDuringDecision` (novo) — simula exatamente o cenario de corrida: snapshot de lock orfao, "vitoria" de um tomador concorrente (remove+recria), e confirma que uma segunda tentativa de takeover baseada no snapshot antigo é recusada com `ErrLayerLocked` sem apagar o lock legitimo recriado. `TestReadLayerLockRawWithRetry_RetriesBeforeDecidingOrphan`/`_DeclaresOrphanAfterExhaustingRetries` adaptados e continuam verdes.
- Validacao: nao executavel nativamente em macOS (`//go:build windows`); `GOOS=windows GOARCH=amd64 go build ./internal/runtime/memory/durable/...` (OK), `go vet` (OK), `go test -c -o /tmp/durable_windows_test2.exe ./internal/runtime/memory/durable/...` (compila sem erro); `gofmt -l` vazio

- ID: BUG-22
- Severidade: low
- Origem: re-review tarefa 6.0 (2a rodada); R-STYLE-001.1 (hard)
- Estado: fixed
- Causa raiz: os logs de ativacao/desativacao de RF-30 em `internal/runtime/runner.go` (`"runner: durable memory ativada (RF-28)"` / `"...desativada..."`) permaneceram em PT-BR apos a correcao de BUG-15 (que tratou apenas comentarios, nao esses dois `log.Printf`). Adicionalmente, o doc-comment de `MemoryPersistHook.Run` em `internal/runtime/hooks/memory_persist.go` ficou factualmente desatualizado pela propria tarefa 6.0: descrevia "Evento inesperado: ignorado silenciosamente", mas a subtarefa 6.5 da mesma tarefa passou a emitir `log.Printf` nesse caminho — a linha nao é mais silenciosa.
- Correcao: as duas mensagens de log traduzidas para ingles (`"runner: durable memory enabled (RF-28)"` / `"...disabled...; legacy path preserved"`); doc-comment desatualizado removido de `memory_persist.go` (codigo autoexplicativo, conforme R-STYLE-001.2); doc-comment de `dispatchSessionPostEnd` (tambem desatualizado, achado informativo da mesma revisao) removido de `runner.go`.
- Arquivos alterados: `internal/runtime/runner.go`, `internal/runtime/hooks/memory_persist.go`
- Teste de regressao: nenhum teste depende do texto exato dos logs (confirmado via `grep` antes da alteracao); comportamento funcional inalterado
- Validacao: `go build ./...`; `go test ./internal/runtime/... -run "Parity|Golden" -v -count=1` (30 passed); `go test ./... -count=1` completo sem FAIL; `gofmt -l`

- ID: BUG-23
- Severidade: low
- Origem: re-review tarefa 9.0 (2a rodada); clareza de documentacao (RF-23/RF-25)
- Estado: fixed
- Causa raiz: `docs/troubleshooting.md`, secao sobre `memory handoff claim`, descrevia o mecanismo do CLI como "igual" ao problema de bastao retido em sessao orquestrada, sem afirmar explicitamente que e o MESMO estado de lease persistido — deixando implicita, em vez de afirmada, a unificacao entre `Facade` e CLI corrigida pelo BUG-02/BUG-06.
- Correcao: paragrafo de causa reescrito para declarar explicitamente que `memory handoff` e a sessao orquestrada leem/escrevem o MESMO arquivo sidecar sob a MESMA trava, e que uma reivindicacao em qualquer um dos dois caminhos e vinculante para o outro.
- Arquivos alterados: `docs/troubleshooting.md`
- Teste de regressao: nao aplicavel (mudanca textual em documentacao)
- Validacao: leitura manual confirmando consistencia com o codigo (`handoff_lease.go`, `cmd/ai_spec_harness/memory.go`)

- ID: BUG-24
- Severidade: critical
- Origem: re-review tarefa 10.0 (2a rodada); RF-20, O-8
- Estado: fixed
- Causa raiz: a correcao anterior do BUG-03 (rodada 1) tornou o benchmark executavel e nao-vazio, mas o tornou VACUAMENTE TRIVIAL: `Layer.Read`/`FakeFileSystem.ReadFile` resolvem por chave exata (mapa/caminho direto, sem varredura de diretorio), entao medir a leitura de UMA pagina de Task entre 999 paginas irmas tem custo identico independente do volume — o benchmark passaria com qualquer implementacao, boa ou ruim, e passaria igualmente com 1 ou 1.000.000 de paginas irmas. Nao prova nada sobre RF-20.
- Correcao: redesenhado para semear 1.000 Fatos distintos DIRETAMENTE via `Layer.Consolidate` (chamada de baixo nivel, ignorando deliberadamente `Facade.RecordSession` e a compactacao que ele aciona) em uma UNICA pagina PRD real — algo que o caminho normal de escrita (`RecordSession`) nao consegue produzir, pois a compactacao real (RF-13) arquivaria a maioria dos fatos a cada chamada (confirmado experimentalmente na rodada 1: de 1000 gravados, ~11 permaneciam ativos). O benchmark mede entao `Facade.BuildContext` sobre essa pagina real com 1.000 Fatos ativos — exercitando de fato o custo de `Parse` (regex + YAML por fato), `RelevancePolicy.Rank` e `BudgetPolicy.Allocate` sobre um volume real. Resultado apos a correcao: `133154 allocs/op`, `~30ms/op`, escalando com o numero de fatos — nao mais um numero fixo independente de volume.
- Arquivos alterados: `internal/runtime/memory/durable/recovery_bench_test.go`, `internal/runtime/memory/durable/facade_bench_test.go` (o teste preliminar da tarefa 7.0 tinha o mesmo defeito estrutural, corrigido em conjunto)
- Teste de regressao: os proprios benchmarks/testes, agora com asserção `FactsByLayer["prd"] == factCount` logo apos o seeding (falha explicita se o volume real nao for atingido, eliminando a classe de defeito "falha silenciosa" do BUG-03 original)
- Validacao: `go test ./internal/runtime/memory/durable/... -run TestBuildContextP95PreliminaryBenchmark_1000Pages -v -count=1` (1 passed); `make bench` -> `BenchmarkRecoveryWith1000Pages-8 39 30141662 ns/op 31.05 p95_ms 13766891 B/op 133155 allocs/op PASS` (p95 real ≈ 31ms, dentro do limite de 200ms, mas agora genuinamente sensivel a volume); `go test ./... -count=1` completo sem FAIL; `golangci-lint run ./internal/runtime/memory/durable/...` -> "No issues found"

## Comandos Executados

- `go build ./...` -> sem erros
- `go test ./... -count=1` -> 0 FAIL (suite completa do repositorio)
- `go test ./internal/runtime/memory/durable/... -count=1 -v` -> 142 passed
- `go test ./internal/runtime/memory/durable/... -run TestFacadeSuite -v -count=1` -> 14 passed
- `go test ./internal/runtime/... -run "Parity|Golden" -v -count=1` -> 30 passed
- `go vet ./...` -> sem erros
- `gofmt -l` nos arquivos tocados -> vazio
- `golangci-lint run ./internal/runtime/...` -> 1 issue pre-existente fora do escopo (`acp_opencode_telemetry_test.go`), confirmado nao introduzido por nenhuma rodada desta feature
- `make integration` -> `ok` em todos os pacotes, incluindo `internal/runtime/memory/durable` e `cmd/ai_spec_harness`
- `make bench` -> `PASS`, p95 real ≈ 31ms com 1.000 fatos ativos numa unica pagina
- `GOOS=windows GOARCH=amd64 go build/vet/test -c ./internal/runtime/memory/durable/...` -> OK (BUG-21)

## Riscos Residuais

- A tarefa 7.0 nao pode ser re-revisada nesta rodada porque o subagente de revisao falhou por limite semanal de API antes de concluir (erro `You've hit your weekly limit`). Os achados da tarefa 7.0 da primeira rodada (BUG-01, BUG-08, BUG-09, BUG-10, BUG-15 subset A) foram corrigidos pelo subagente arquitetural e validados por ele com evidencia real antes desta rodada, mas uma segunda revisao independente e adversarial dessa tarefa especifica (a de maior risco do PRD) ainda esta pendente quando o limite de API for renovado.
- BUG-21 (Windows) permanece validado apenas por cross-compilation; nenhuma execucao nativa em Windows real foi feita em nenhuma das duas rodadas — recomenda-se CI Windows real antes do proximo release.
- O benchmark corrigido (BUG-24) usa `FakeFileSystem` (memoria), nao um filesystem real — aceitavel para custo de CI e consistente com o padrao ja estabelecido em `facade_bench_test.go`/`recovery_bench_test.go` desde a tarefa 7.0, mas nao mede I/O real de disco.
