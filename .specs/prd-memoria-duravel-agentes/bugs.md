# Bugs — Ciclo review -> bugfix -> review (PRD memoria-duravel-agentes)

Consolidado a partir de 10 subagentes de revisão independentes (um por tarefa 1.0-10.0).
Vereditos: 3.0 APPROVED; 1.0, 2.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0 REJECTED.

Severidade traduzida via `.agents/skills/agent-governance/references/severity-mapping.md`
(critical->critical, high->major, medium/low->minor). Nível original preservado no campo `Origem`.

## BUG-01 [critical]
- Origem: review tarefa 7.0; RF (ADR-001 do PRD, precedência de invariantes: segredo > não-perda de fato > bastão > orçamento)
- file: internal/runtime/memory/durable/facade.go
- line: 243-247
- reproduction: duas sessões concorrentes no mesmo projeto; sessão B falha `claimBaton` (`ErrBatonAlreadyClaimed`) e `RecordSession` retorna erro imediatamente sem gravar nenhum fato.
- expected: falha de bastão nunca deve impedir a escrita/consolidação de fatos; deve apenas marcar campo do relatório (`BatonClaimed=false`), nunca abortar `Consolidate`.
- actual: `RecordSession` aborta a escrita inteira quando `claimBaton` falha, invertendo a precedência declarada.

## BUG-02 [critical]
- Origem: review tarefas 4.0 e 9.0; RF-23, RF-25
- file: internal/runtime/memory/durable/facade.go (ContinuityHandoff em memória) + cmd/ai_spec_harness/memory.go (sidecar de handoff nunca lido pela Facade)
- reproduction: duas invocações concorrentes de `task-loop` (processos distintos), cada uma cria sua própria `Facade` com `ContinuityHandoff` vazio; ambas "reivindicam" o bastão com sucesso. Um operador que roda `memory handoff claim` não impede sessão orquestrada concorrente de prosseguir.
- expected: `Facade` e o comando `memory handoff` devem compartilhar a mesma fonte de verdade persistida em disco, sob lock, garantindo dono único do bastão entre processos orquestrados e entre CLI humana e runtime.
- actual: dois relógios de lease independentes que nunca se enxergam — RF-23 ("exatamente uma sessão detém o lease") não é garantido no caminho real.

## BUG-03 [critical]
- Origem: review tarefa 10.0; RF-20, O-8
- file: internal/runtime/memory/durable/recovery_bench_test.go
- line: 17
- reproduction: rodar `make bench`; `FacadeConfig` do benchmark só define `ProjectDir`, nunca `TasksDir` — escrita nas camadas PRD/Task falha silenciosamente (log, não erro), `FactsByLayer` fica vazio.
- expected: benchmark deve popular efetivamente ~1000 páginas, verificar que o volume foi de fato escrito/recuperado, e falhar (`b.Fatalf`) se o volume não for alcançado ou se p95 > 200ms.
- actual: benchmark mede `BuildContext` sobre estado vazio (0 fatos); o número de p95 reportado não prova RF-20.

## BUG-04 [major]
- Origem: review tarefa 2.0; RF-36
- file: internal/runtime/memory/durable/page.go
- line: 158-164 (Serialize)
- reproduction: serializar página com conteúdo humano sem newline final e Fatos a acrescentar.
- expected: `ErrRoundTripNotPreserved` deve ser retornado sempre que a serialização não reproduzir exatamente o `human.Content` original recebido.
- actual: a função ajusta `expectedContent` para acomodar a própria mutação, tornando a sentinela inatingível; um byte é acrescentado silenciosamente ao conteúdo humano.

## BUG-05 [major]
- Origem: review tarefa 2.0; RF-36
- file: internal/runtime/memory/durable/page.go
- line: 52-149 (Parse/Serialize)
- reproduction: página com conteúdo humano intercalado entre duas seções de Fato; serializar novamente após `Parse`.
- expected: a posição relativa do conteúdo humano em relação aos Fatos deve ser preservada, ou a escrita deve recusar explicitamente quando não puder preservar (nunca reordenar silenciosamente).
- actual: todos os blocos humanos são reagrupados em um único bloco antes de todos os Fatos, deslocando texto de autoria humana sem aviso.

## BUG-06 [major]
- Origem: review tarefas 4.0 e 9.0 (Finding 2 de 9.0); RF-23
- file: cmd/ai_spec_harness/memory.go (runHandoffClaim, saveHandoffLease, runHandoffRelease)
- line: 457-486
- reproduction: dois processos reais chamam `memory handoff claim` simultaneamente sobre o mesmo lease órfão.
- expected: leitura+decisão+escrita do lease deve ser atômica (lock ou compare-and-swap); segunda reivindicação concorrente deve ser recusada, nunca sobrescrever silenciosamente.
- actual: race TOCTOU sem lock algum ao redor do ciclo leitura-decisão-escrita.

## BUG-07 [major]
- Origem: review tarefa 5.0; RF-16 (Windows)
- file: internal/runtime/memory/durable/layer_lock_windows.go
- line: 14-51
- reproduction: dois processos Windows disputando o lock da mesma camada no instante entre criação do arquivo de lock (`O_CREATE|O_EXCL`) e gravação do conteúdo (`ProcessRef`).
- expected: o conteúdo do `ProcessRef` deve estar visível atomicamente com a criação do arquivo (ex.: escrever em temp + rename), ou "conteúdo incompleto/ilegível" deve ser tratado como possível escrita em andamento (retry), nunca como órfão imediato.
- actual: segundo processo lê o lock incompleto do dono ativo, interpreta como ilegível, remove-o sem checar liveness, e cria seu próprio lock — dois processos escrevendo concorrentemente na mesma página.

## BUG-08 [major]
- Origem: review tarefa 7.0; RF-28
- file: internal/config/resolver.go (mergeInto) + internal/taskloop/runtimeconfig.go (optionsToConfigOverrides)
- line: 206-211 (resolver.go)
- reproduction: config global/workspace liga `durable_memory_enabled=true`; flag CLI `--durable-memory=false`.
- expected: `flags > workspace > global > defaults` (RF-28) — flag explícita deve poder desligar a feature mesmo com camada inferior ligando.
- actual: merge "sticky-true" nunca permite que `false` sobrescreva `true` de camada inferior; a flag não consegue desativar a feature.

## BUG-09 [major]
- Origem: review tarefa 7.0; RF-09
- file: internal/runtime/runner.go (recordDurableMemorySession) / internal/runtime/memory/durable/facade.go
- reproduction: rodar sessão completa via ACPRunner com `--durable-memory`.
- expected: RF-09 exige fonte dupla — sinal estruturado (garantido) + seção declarada pela sessão (enriquece). `SessionFacts.DeclaredSection` deve ser populada a partir de conteúdo real da sessão quando presente.
- actual: `DeclaredSection` nunca é populada em produção — a segunda fonte de captura declarada no requisito não existe no caminho real.

## BUG-10 [minor]
- Origem: review tarefa 7.0; MD-001 (contenção de decisão de domínio na fachada)
- file: internal/runtime/memory/durable/facade.go
- line: 301-334 (deriveFacts)
- reproduction: ler `deriveFacts`.
- expected: a decisão de qual `Durability` cada tipo de sinal recebe deve estar em um colaborador/política nomeada, não como lógica condicional embutida na fachada.
- actual: classificação de domínio (kind->durability) embutida via `if`/`kind` literal diretamente na fachada.

## BUG-11 [major]
- Origem: review tarefa 8.0; RF-34
- file: internal/runtime/persistence/report.go (injectBoundedSection / injectMetricsSection)
- reproduction: chamar `EnrichReport` duas vezes sobre o mesmo `execution_report.md` — primeira só com métricas, segunda com `MemoryEvidence` preenchido (cenário real de retry em `internal/taskloop/acpinvoker.go`, que reusa o mesmo `EvidenceDir` entre tentativas).
- expected: seções nomeadas anteriores devem ser preservadas; nova seção deve ser inserida sem apagar seções intermediárias já existentes.
- actual: reinjeção de métricas substitui "do cabeçalho até o fim do arquivo", apagando a seção de evidência de memória recém-inserida.

## BUG-12 [major]
- Origem: review tarefa 9.0 (Finding 2); segurança/integridade de escrita (techspec)
- file: cmd/ai_spec_harness/memory.go (runMigrate, saveHandoffLease)
- reproduction: symlink plantado em `<tasks-dir>/memory/MEMORY.md` ou no sidecar de handoff.
- expected: toda escrita em caminho derivado de configuração deve chamar `fs.RefuseExternalSymlink` antes de gravar (mesmo padrão já usado em `layer.go:400`).
- actual: `runMigrate` e o trio `handoff status/claim/release` escrevem sem essa proteção.

## BUG-13 [major]
- Origem: review tarefa 10.0 (subtarefa 10.3); RF-16
- file: internal/runtime/memory/durable/handoff_lease_integration_test.go (cobertura incorreta) — falta teste real para layer_lock_unix.go / layer_lock_windows.go
- reproduction: nenhum teste, unitário ou de integração, exercita `takeOverStaleLayerLock` (Windows) ou o `flock` real (Unix) do `LayerLock` com processo morto.
- expected: subtarefa 10.3 exige provar que o mecanismo real de RF-16 (`LayerLock`) detecta lock órfão e registra a tomada; `HandoffLease` (RF-23) é um mecanismo diferente e não substitui essa prova.
- actual: 10.3 testa apenas `HandoffLease`/`LeasePolicy`, deixando o `LayerLock` real sem cobertura de cenário de processo morto.

## BUG-14 [minor]
- Origem: review tarefa 10.0 (subtarefa 10.6); RF-25
- file: internal/runtime/memory/durable/bootstrap_handoff_integration_test.go (TestHandoffCrossCLIDeliversAtLeast90PercentOfFacts)
- reproduction: ler o teste — writer e reader usam `CLI:"claude"` fixo nas duas pontas.
- expected: teste deve variar o valor de CLI entre escrita e leitura para provar neutralidade real.
- actual: CLI nunca varia; "cross-CLI" é alegado por argumento estrutural, não exercitado.

## BUG-15 [major]
- Origem: review tarefas 5.0, 6.0, 7.0, 9.0; R-STYLE-001.2 (hard, zero comentários)
- file (subset A — arquiteturais, ver BUG-01/02/08/09/10): internal/runtime/memory/durable/facade.go (arquivo inteiro), internal/runtime/memory_port.go, internal/runtime/memory/durable/facade_bench_test.go, internal/runtime/memory/durable/facade_test.go:34, internal/runtime/types.go, internal/config/resolver.go, internal/config/runtime.go, internal/taskloop/taskloop.go, internal/taskloop/acpinvoker.go, cmd/ai_spec_harness/task_loop.go, internal/runtime/runner.go (linhas 158-161, 243-245, 758-760)
- file (subset B — mecânicos): internal/runtime/memory/durable/layer.go:81-92 (ActivePath/SidecarPath), cmd/ai_spec_harness/cli_contract_test.go:98-100
- reproduction: `grep "//"` nesses arquivos nas linhas tocadas por esta feature (excluindo `//go:build`).
- expected: zero comentários (doc-comments inclusive) em código criado/editado por esta feature, por R-STYLE-001.2 (hard, sem exceção documentada além de shebang/`go:build`).
- actual: dezenas de comentários novos, incluindo doc-comments em toda API exportada nova e um comentário em PT-BR misturado com código em inglês.

## BUG-16 [major]
- Origem: review tarefa 1.0; RF-16 (prova de atomicidade)
- file: internal/fs/os_test.go
- line: 117-152 (TestOS_WriteFileAtomic_noPartialFileOnInterruption)
- reproduction: ler o teste — ele nunca interrompe `WriteFileAtomic` em execução real; apenas cria um arquivo `.tmp-interrupted` alheio e confirma que ele não contamina o caminho final.
- expected: o teste deve provar que uma falha real a meio da sequência Write->Sync->Close->Rename nunca deixa o caminho final com conteúdo parcial/corrompido, comparado ao conteúdo pré-existente nesse caminho.
- actual: o teste não exercita nenhum caminho de falha real da função.

## BUG-17 [minor]
- Origem: review tarefa 1.0
- file: internal/fs/fs.go (WriteFileAtomic vs WriteFile — semântica de symlink)
- reproduction: caminho final é um symlink.
- expected: o comportamento diante de symlink deve ser documentado e travado por teste (`WriteFileAtomic` substitui o link via `Rename`; `WriteFile` segue o link via `os.WriteFile`).
- actual: divergência de semântica entre os dois métodos da mesma interface, sem teste que documente/trave a decisão.

## BUG-18 [minor]
- Origem: review tarefa 6.0; cobertura do golden de paridade (RF-29)
- file: internal/runtime/prompt_parity_golden_test.go (caso 1)
- reproduction: caso 1 usa `TasksDir: t.TempDir()` (diretório vazio real), não `TasksDir: ""`.
- expected: o golden deve cobrir o branch real de `TasksDir==""` (zero-value, RF-26/RF-28) com igualdade estrita byte a byte, não apenas um diretório vazio.
- actual: o branch `TasksDir==""` nunca é exercitado pelo golden na fronteira do hook (`hooks.PointPromptPostBuild`).
