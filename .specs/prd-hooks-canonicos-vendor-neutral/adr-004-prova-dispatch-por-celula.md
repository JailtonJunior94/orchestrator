# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Extensão da prova de dispatch por célula ao eixo evento × família
- **Data:** 2026-09-19
- **Status:** Proposta
- **Decisores:** dono do repositório
- **Relacionados:**
  - PRD [`prd.md`](prd.md) — RF-58 (`prd.md:352`), RF-59 (`prd.md:353-356`), RF-61 (`prd.md:364-369`)
  - Techspec [`techspec.md`](techspec.md)
  - [ADR-001](adr-001-hook-contract-fonte-unica-projecoes.md) — `internal/hookcontract` como fonte única dos cinco eventos canônicos
  - [ADR-003 do PRD dependente](../prd-harness-portatil-vendor-neutral/adr-003-capability-matrix-gerada.md) — matriz gerada, nunca editada à mão

## Contexto

### Correção já ocorrida: a prova deixou de ser tautológica

Esta ADR parte de um estado **novo**. A versão anterior deste documento descrevia
`DispatchProvenFromParityTests` como prova tautológica: um parse AST que confirmava a existência de
um método `TestParity_AllTools` com receiver `*ParitySuite` e, com base apenas nisso, declarava
provada a matriz inteira. **Esse defeito foi corrigido no código e não descreve mais o presente.**

O estado atual de `internal/capability/evidence.go` é este. `DispatchProvenFromParityTests`
(`evidence.go:169-175`) resolve a raiz do repositório e delega a `dispatchProvenFromTests`
(`evidence.go:174`) com os parâmetros reais: a suíte `evidenceTestSuite = "ParitySuite"`
(`evidence.go:66`), o teste `EvidenceTest = "TestParitySuite/TestParity_AllTools"`
(`internal/capability/capability.go:23`), a raiz resolvida e o padrão de pacote
`defaultParityPackagePattern = "./internal/parity/..."` (`evidence.go:68`).

`dispatchProvenFromTests` (`evidence.go:140-150`) prova em duas etapas conjuntivas:

1. **Declaração sintática.** Recorta o nome do método após a última `/` (`evidence.go:141-144`) e
   confirma por AST, em `ParityTestMethodExists` (`evidence.go:32-53`), que existe método com aquele
   nome e receiver `*ParitySuite` (`evidence.go:39-50`). Esta era a totalidade da prova antiga.
2. **Execução real.** `declared && executionProof(...)` (`evidence.go:146`). `executionProof`
   (`evidence.go:128-138`) dispara `exec.Command("go", "test", "-json", "-count=1", "-run", ...)`
   no diretório resolvido (`evidence.go:129-130`), com padrão `-run` construído por `runPatternFor`
   (`evidence.go:120-126`), que ancora cada segmento do nome do teste com `^…$` e `regexp.QuoteMeta`.
   O erro de `cmd.Run()` é deliberadamente descartado (`evidence.go:134`): o veredito não vem do exit
   code, vem do stream estruturado.

A leitura do stream é `parseParityTestEvidence` (`evidence.go:76-118`): consome `go test -json` linha
a linha com buffer de até 8 MiB (`evidence.go:83`), descarta o que não for objeto JSON
(`evidence.go:86-90`), ignora eventos de outros testes (`evidence.go:101-103`) e acumula os estados
`pass`, `skip` e `fail` (`evidence.go:104-111`). A regra decisiva está em `evidence.go:114-116`:
**`skip` ou `fail` zeram a prova**, mesmo com `pass` registrado. Não há caminho textual; linha
`--- PASS:` impressa por `t.Logf` não é evento e não produz prova.

`internal/capability/evidence_regression_test.go` — arquivo novo, também não versionado — trava
exatamente essa correção. Ele monta um módulo Go efêmero em `t.TempDir()` (`writeFixtureModule`,
`:17-32`) e usa `mustHaveSyntacticMethod` (`:34-43`) para garantir que o fixture **satisfaz o check
sintático antigo**, deixando a mensagem explícita de que "old behavior relied only on this check"
(`:41`). Sobre esse fixture, três casos: corpo com `t.Skip` nunca prova (`:45-53`), corpo com
`t.Fatal` nunca prova (`:55-63`), e corpo vazio que executa e passa prova (`:65-73`). É o teste de
não regressão da tautologia.

### O que permanece por fazer: a prova ainda não discrimina por célula

Aqui esta ADR corrige a premissa com que foi encomendada. **A prova ainda não discrimina por
célula.** O closure devolvido por `dispatchProvenFromTests` ignora os dois argumentos:

```
evidence.go:147    return func(provider, capabilityID string) bool {
evidence.go:148        return resolved
evidence.go:149    }
```

`resolved` é um único booleano global, calculado em `evidence.go:146`. O caminho de erro de
`DispatchProvenFromParityTests` (`evidence.go:171-173`) também devolve closure que ignora os
argumentos, fixo em `false`. A diferença entre os dois caminhos é o valor, não a granularidade.

O que mudou, portanto, é o **eixo da falsidade**, não sua ausência: antes a prova era falsa por
construção (existir um método bastava); hoje ela é verdadeira por execução real, mas **uniforme**
sobre as 54 células. Uma célula de `copilot` é declarada provada pelo mesmo `pass` que prova uma
célula de `claude`, ainda que `TestParity_AllTools` não exercite as duas do mesmo modo. A
discriminação por par `(provider, capabilityID)` é trabalho pendente, e é a base sobre a qual o eixo
novo precisa nascer.

### Estado atual da matriz existente

O eixo existente é `provedor × invariante de paridade`. `Generate`
(`internal/capability/capability.go:40-68`) instancia o checker de `internal/parity`
(`capability.go:41`), roda os invariantes (`capability.go:49-50`) e emite uma célula por par
provedor/invariante via `buildCell` (`capability.go:77-104`). A célula é
`{Provider, Capability, Description, State, Reason, Test}` (`capability.go:25-32`).

`testdata/capability-matrix.json` tem hoje **54 células**, **26 capabilities distintas** e quatro
provedores — claude 18, codex 13, opencode 12, copilot 11 — com estados `supported` 32,
`provider capability` 12 e `unknown` 10. As 44 células afirmativas são as que `EvidenceCells`
(`evidence.go:20-30`) marca `Supported`, pela regra `c.State == StateSupported || c.State ==
StateProviderCapability` (`evidence.go:26`).

Os artefatos são gerados por `RenderBoth` (`capability.go:124-130`) e travados por golden test em
`internal/capability/golden_test.go:11-17`, que aponta para `testdata/capability-matrix.json` e
`docs/capability-matrix.md`. O gate é `check-capability-matrix-sync`, definido em `Makefile:90-91`
como `go test ./internal/capability/...`, e executado em CI pelo step "Capability matrix sync gate"
em `.github/workflows/test.yml:119-120`.

### O gate consumidor

`ValidateCapabilityMatrixEvidence` (`internal/runtime/specs/parity_gate.go:110-125`) recebe as
células e a função de prova `CapabilityDispatchProofFunc` (`parity_gate.go:108`). Seu comportamento:
matriz vazia produz a violação `"capability matrix has zero cells"` (`parity_gate.go:111-113`);
células não afirmativas são puladas (`parity_gate.go:117-119`); `dispatchProven == nil` é
**fail-closed** junto com a prova negativa (`parity_gate.go:120-122`), emitindo
`"no dispatch proof test associated"` (`parity_gate.go:121`). O gate está correto e não precisa de
alteração de contrato.

`internal/runtime/specs/parity_gate_capability_test.go` tem quatro testes: `_EmptyMatrix` (`:9`),
`_SupportedWithoutDispatchProof` (`:20`), `_NilDispatchProvenFailsClosed` (`:38`) e
`_ResolvedProofPasses` (`:47`). Honestamente: **nenhum dos quatro exercita discriminação por
célula.** Os dois que fornecem oráculo usam closures constantes — `func(string, string) bool { return
false }` (`:26`) e `func(string, string) bool { return true }` (`:53`) — e passariam sem alteração
com qualquer oráculo uniforme. O mesmo vale para `internal/capability/gate_test.go:26-51`. A única
regra de célula hoje travada por teste é a de que célula não suportada não exige prova
(`gate_test.go:45-51`).

### Precedente de rigor interno

`buildCell` já rebaixa para `StateUnknown` o invariante auto-satisfeito por stub injetado pelo
próprio gerador, com razão explícita citando V-25 (`capability.go:85-89`, mensagem em
`capability.go:87`: `"invariant satisfied by generator-injected stub %q; auto-satisfied by
construction and never sustains a supported cell (V-25)"`). O repositório já trata prova
auto-satisfeita como não-prova na geração da célula; a etapa de prova deve herdar esse mesmo rigor.

### Mecanismo por célula já provado no eixo de pontos canônicos

`internal/runtime/specs/parity_dispatch_proof_test.go:40` mantém `dispatchProofRegistry
map[string]map[specs.CanonicalPoint]dispatchProofCase`, com 12 células — quatro agentes × três
pontos canônicos. Sobre ele há quatro camadas: existência verificada por AST contra
`integrationTestsRelDir = "tests/integration"` (`:24`) em
`TestDispatchProofRegistryNamesExistingIntegrationTests` (`:254`); cobertura obrigatória em
`TestDispatchProofRegistryCoversEveryMandatoryCell` (`:270`); execução real em subprocesso
`go test -tags=integration -count=1 -json -timeout 20m` (`:210-211`, timeout em
`dispatchProofRunTimeout = "20m"`, `:25`) consumida por `parseDispatchEvidence` (`:138`) e exigida
célula a célula por `TestEveryMandatoryCellHasExecutionEvidence` (`:291`); e anti-forja provada em
`parity_dispatch_proof_injection_test.go:24` (`t.Logf` não forja evidência) e `:52` (`skip` revoga
`pass`), com proibição de escape por variável de ambiente em
`dispatch_proof_provenance_test.go:20` e exigência de coleta por subprocesso em `:32`.

Esse é o mecanismo a reusar. Ele é indexado por célula — o que `evidence.go:147-149` ainda não é.

### Estado de versionamento

`git status --short` retorna `?? internal/capability/`, `?? docs/capability-matrix.md`,
`?? testdata/capability-matrix.json` e `?? internal/runtime/specs/parity_gate_capability_test.go`.
**Toda a base sobre a qual esta decisão opera está fora do histórico do git.** Inclui a correção da
tautologia e seu teste de regressão.

### Dependência dura sobre o catálogo de eventos

O catálogo de pontos canônicos em `internal/runtime/specs/enforcement.go:12-16` ainda tem **três**
pontos — `PointPreTool`, `PointPostTool`, `PointSessionEnd` — enumerados em `canonicalPoints`
(`enforcement.go:18-22`), com `ErrIncompleteCoverage` declarando literalmente "does not cover the
three canonical points" (`enforcement.go:26`). A matriz de cinco eventos exigida por RF-59
(`prd.md:353-356`) depende da expansão decidida na ADR-001, que cria `internal/hookcontract` com
`EventKind` de cinco valores (`adr-001-hook-contract-fonte-unica-projecoes.md:69-70`) e converte
`CanonicalPoint` em projeção (`adr-001…:86-91`). O pacote `internal/hookcontract` **ainda não existe
no repositório**. As cinco famílias de hooks — `git-policy`, `quality-gate`, `evidence-gate`,
`checkpoint`, `telemetry` — estão fixadas em `prd.md:171-174`.

## Decisão

Estender ao eixo `provedor × evento canônico × família de hook` o mecanismo de prova por execução já
existente, **reusando-o em vez de reconstruí-lo**, e completar nele a discriminação por célula que
ainda falta. A decisão tem três componentes.

### Componente 1 — Reusar o mecanismo existente e completar a indexação por célula

O eixo atual (`provedor × invariante de paridade`, 54 células, gate `Makefile:90-91` e step
`.github/workflows/test.yml:119-120`) **permanece como está em tudo o que já é correto**: a prova por
execução real de `go test -json` (`evidence.go:128-138`), a revogação por `skip` ou `fail`
(`evidence.go:114-116`), o fail-closed do gate (`parity_gate.go:120-122`) e os golden tests
(`golden_test.go:11-17`). Nada disso é reconstruído.

O que se completa é a granularidade. O closure uniforme de `evidence.go:147-149` passa a resolver por
par `(provider, capabilityID)`, sustentado por uma tabela de mapeamento célula → teste, na forma já
provada por `dispatchProofRegistry` (`parity_dispatch_proof_test.go:40`). A prova de uma célula é
verdadeira apenas quando as três condições valem conjuntamente:

1. existe entrada de mapeamento nomeando um teste para **aquela** célula;
2. esse teste existe de fato, confirmado por AST, como em `parity_dispatch_proof_test.go:254`;
3. a execução real reportou `pass` para a chave daquela célula pelo stream `go test -json`, com
   `skip` e `fail` revogando, como já vale em `evidence.go:114-116`.

A assinatura `specs.CapabilityDispatchProofFunc` (`parity_gate.go:108`) não muda, de modo que o gate
consumidor não é tocado.

### Componente 2 — Criar a segunda matriz, coexistente

Criar os artefatos `testdata/hook-capability-matrix.json` e `docs/hook-capability-matrix.md`, com
eixo `provedor × evento canônico × família de hook`: cinco eventos × cinco famílias × quatro
provedores. A fonte de dados é a declaração de capability de `internal/hookcontract` prevista na
ADR-001 (`adr-001…:73-75`), nunca edição manual — restrição herdada da
[ADR-003 do PRD dependente](../prd-harness-portatil-vendor-neutral/adr-003-capability-matrix-gerada.md).

A segunda matriz **coexiste** com `testdata/capability-matrix.json`. Os eixos são semanticamente
distintos e não são unificados.

Os estados declarados são três, alinhados a `adr-001…:73-75`:

- `verified` — prova indexada, nas três condições do componente 1;
- `adapter` — suportado por adaptação, com o campo `limitation` **obrigatoriamente preenchido**,
  atendendo à exigência de RF-59 de limitações explícitas (`prd.md:355`);
- `unsupported` — estado próprio, distinto de sucesso e de falha.

Regras normativas: nenhuma célula é `verified` sem prova indexada; célula `unsupported` **não exige
prova** — a regra que `gate_test.go:45-51` já trava para o eixo atual vale igualmente aqui, e é o que
mantém o custo de execução finito; e capability específica de um provedor não é promovida ao núcleo
universal (`prd.md:355-356`), separação hoje materializada em `capability.go:97-101`.

### Componente 3 — Substituir o fallback silencioso por erro tipado

O caminho de erro de `DispatchProvenFromParityTests` (`evidence.go:171-173`) devolve hoje um closure
fixo em `false` quando `repoRootFromWorkingDir` (`evidence.go:152-167`) não localiza `go.mod` subindo
a partir do diretório de trabalho (erro em `evidence.go:163`). Isso é **fail-closed** e, portanto,
seguro: toda célula vira não-provada e o gate falha alto. O problema não é de segurança, é de
diagnóstico — o operador lê `"no dispatch proof test associated"` (`parity_gate.go:121`) em 44
células e não tem como distinguir "nenhuma prova existe" de "não achei a raiz do repositório".

`DispatchProvenFromParityTests` passa a devolver `(specs.CapabilityDispatchProofFunc, error)`, com o
erro de `repoRootFromWorkingDir` encadeado no formato `fmt.Errorf("context: %w", err)`. O chamador —
hoje `gate_test.go:62` — reporta a falha de ambiente como falha de ambiente. O comportamento
fail-closed é preservado: erro de resolução continua impedindo qualquer célula de ser considerada
provada.

Partes impactadas: `internal/capability/evidence.go`, `internal/capability/capability.go`,
`internal/capability/gate_test.go`, `internal/runtime/specs/parity_gate_capability_test.go`, os
artefatos gerados, e o subcomando `doctor` de RF-60 (`prd.md:357-360`).

## Alternativas Consideradas

### A1 — Unificar as duas matrizes em um único artefato

Descrição: colapsar `testdata/capability-matrix.json` e a matriz nova de RF-59 em um artefato só.

Vantagens: um artefato para gerar, versionar e ler; um golden test; um gate.

Desvantagens: os eixos são semanticamente distintos. O existente indexa por invariante de paridade
(`capability.go:81`, `cell.Capability = inv.ID`); o novo indexa por evento canônico × família de
hook. Unificar produz o produto cartesiano de eixos independentes, gerando células sem significado —
um invariante de paridade cruzado com uma família de hook não descreve nada — e quebra os golden
tests que travam o formato atual (`golden_test.go:11-17`, `gate_test.go:68`, `:84`, `:100`).

Motivo da rejeição: perda de significado semântico e quebra de gate existente sem ganho
correspondente. Rejeitada.

### A2 — Reconstruir o mecanismo de prova do zero para o eixo novo

Descrição: escrever um mecanismo de prova próprio para a matriz de eventos × famílias, independente
do que existe em `internal/capability/evidence.go` e em `parity_dispatch_proof_test.go`.

Vantagens: liberdade de desenho sem restrição do formato atual; nenhuma refatoração do eixo antigo.

Desvantagens: o mecanismo existente já é correto no que importa — execução real
(`evidence.go:128-138`), revogação por `skip`/`fail` (`evidence.go:114-116`), anti-forja demonstrada
com teste de regressão (`parity_dispatch_proof_injection_test.go:24`, `:52`) e proibição de escape
por variável de ambiente (`dispatch_proof_provenance_test.go:20`). Reconstruir duplicaria lógica
crítica de segurança em dois lugares, com duas superfícies de erro e duas oportunidades de reintrodução
do modo de falha forjável. Duplicação de lógica crítica é exatamente o que a User Story existe para
eliminar.

Motivo da rejeição: duplicação de lógica crítica já provada, sem ganho. Rejeitada.

### A3 — Manter o fallback silencioso devolvendo `false`

Descrição: preservar `evidence.go:171-173` como está, sem propagar erro.

Vantagens: é fail-closed e, portanto, seguro — nenhuma célula é promovida indevidamente. Custo zero.
A assinatura da função permanece de um único valor de retorno.

Desvantagens: o resultado é indistinguível de "nenhuma prova existe". Um erro de ambiente — invocar o
gate fora da árvore do repositório, por exemplo — produz 44 violações com o texto
`"no dispatch proof test associated"` (`parity_gate.go:121`), que aponta para a causa errada. O
operador é levado a procurar testes ausentes quando o defeito é de diretório de trabalho. Diagnóstico
enganoso é pior que diagnóstico ausente.

Motivo da rejeição: rejeitada por auditabilidade (RNF05, `prd.md:431`), não por segurança. A
segurança do fallback é real e é preservada pela decisão.

### A4 — Marcar as células do eixo novo como `verified` por herança das provas do eixo antigo

Descrição: considerar provada qualquer célula `(provedor, evento, família)` cujo provedor já tenha
células afirmativas provadas na matriz de invariantes de paridade.

Vantagens: custo de implementação próximo de zero; a segunda matriz nasce verde.

Desvantagens: é reintroduzir tautologia em um eixo novo. Um `pass` de `TestParity_AllTools` não diz
nada sobre o despacho do evento `SessionStart` na família `checkpoint` para o OpenCode. Herança de
prova entre eixos é a mesma falácia que `evidence_regression_test.go:34-43` trava para o eixo antigo,
aplicada um nível acima. Também contraria o precedente de `buildCell`, que rebaixa a célula
auto-satisfeita por construção (`capability.go:85-89`).

Motivo da rejeição: reintroduziria, em eixo novo, o defeito que o eixo antigo acaba de corrigir.
Rejeitada.

## Consequências

### Benefícios Esperados

- **Reuso de mecanismo já provado.** A prova por execução real, a revogação por `skip`/`fail` e o
  anti-forja testado passam a cobrir o segundo eixo sem serem reescritos. A superfície de lógica
  crítica não cresce proporcionalmente ao número de eixos.
- **A segunda matriz nasce com rigor desde a primeira célula.** Nenhuma célula `verified` sem prova
  indexada, desde o primeiro commit do artefato — não há janela em que o artefato exista com
  semântica frouxa a ser apertada depois.
- **Erro de ambiente passa a ser distinguível de ausência de prova.** O operador lê a causa real em
  vez de 44 violações apontando para a causa errada.
- **A lacuna de granularidade é fechada.** O closure uniforme de `evidence.go:147-149` deixa de ser o
  elo restante entre uma prova honesta por execução e uma afirmação por célula.
- **RF-60 (`prd.md:357-360`) ganha fonte de verdade confiável** para reportar estado por família e
  por evento, em vez de derivar diagnóstico de prova uniforme.

### Trade-offs e Custos

- **Mapeamento explícito de até 100 células.** Cinco eventos × cinco famílias × quatro provedores.
  Cada célula `verified` exige entrada de mapeamento nomeando teste real. O crescimento de 12
  entradas (`parity_dispatch_proof_test.go:40`) para essa ordem torna a tabela um artefato volumoso
  de manutenção manual.
- **Custo de execução.** A prova por execução é cara e já domina o tempo do gate de paridade, com
  timeout declarado de 20 minutos (`parity_dispatch_proof_test.go:25`). Cada célula `verified` do
  eixo novo amplia o conjunto de chaves exigidas e, portanto, o conjunto de testes que precisam
  existir e passar. Compartilhar a coleta entre eixos mitiga o custo de processo, não o de cobertura.
- **Dois artefatos de matriz sob sincronia recorrente.** `testdata/capability-matrix.json` +
  `docs/capability-matrix.md` e `testdata/hook-capability-matrix.json` +
  `docs/hook-capability-matrix.md`, cada par com seu golden test e sua regeneração. Todo evento,
  família ou provedor novo toca dois lugares.
- **Mudança de assinatura pública.** `DispatchProvenFromParityTests` passa a devolver erro,
  obrigando o chamador a tratá-lo (`gate_test.go:62`). É ruptura pequena e contida, mas é ruptura.

### Riscos e Mitigações

**Risco (a) — `internal/capability/` está untracked e pode ser perdido.**
Impacto: perda integral da base sobre a qual esta ADR opera, incluindo a correção da tautologia e
`evidence_regression_test.go`. `git status --short` confirma `?? internal/capability/`,
`?? docs/capability-matrix.md`, `?? testdata/capability-matrix.json` e
`?? internal/runtime/specs/parity_gate_capability_test.go`.
Mitigação: **commitar esses caminhos antes de iniciar qualquer trabalho derivado desta decisão.**
Nenhuma tarefa começa sobre árvore de trabalho não versionada. Sem esse commit, a ADR referencia
código que não existe no histórico.

**Risco (b) — a expansão de três para cinco pontos canônicos é pré-requisito não entregue.**
Impacto: bloqueante. O catálogo em `enforcement.go:12-16` tem três pontos e `internal/hookcontract`
não existe. Sem a decisão da ADR-001 implementada (`adr-001…:69-70`), o eixo `evento × família` não
tem domínio de eventos e a matriz nova não pode ser gerada.
Mitigação: declarar dependência dura e sequenciar — nenhuma tarefa do componente 2 inicia antes da
entrega do `EventKind` de cinco valores. O componente 1 e o componente 3 são independentes dessa
dependência e podem avançar antes.

**Risco (c) — explosão de custo por exigir prova de células não suportadas.**
Impacto: se `unsupported` exigisse prova, as até 100 células demandariam prova integral, e o gate se
tornaria impagável em tempo de CI, levando ao seu desligamento.
Mitigação: **declarar normativamente que célula `unsupported` não exige prova**, replicando a regra
já travada por `gate_test.go:45-51` e pelo `continue` de `parity_gate.go:117-119`. O custo escala com
o número de células afirmativas, não com o tamanho do produto cartesiano.

**Risco (d) — o gate ser desligado por custo de tempo.**
Impacto: mecanismo que existe e não roda é indistinguível de mecanismo inexistente.
Mitigação: tratada na seção de monitoramento como critério explícito de revisão desta decisão.

Plano de rollback: reverter o componente 3 restaura o closure de `evidence.go:171-173` e é seguro
(continua fail-closed). Reverter o componente 1 restaura o oráculo uniforme e **precisa ser
registrado como dívida técnica**, com prazo e responsável, porque recoloca as 54 células sob prova
não discriminada.

## Plano de Implementação

1. **Commitar** `internal/capability/`, `docs/capability-matrix.md`,
   `testdata/capability-matrix.json` e `internal/runtime/specs/parity_gate_capability_test.go`. Nada
   abaixo se sustenta sem isso.
2. **Componente 3 primeiro.** Alterar `DispatchProvenFromParityTests` (`evidence.go:169-175`) para
   devolver `(specs.CapabilityDispatchProofFunc, error)`, encadeando o erro de
   `repoRootFromWorkingDir` (`evidence.go:163`), e ajustar o chamador em `gate_test.go:62`. É a
   mudança de menor risco e independe da ADR-001.
3. **Adicionar o teste que falta.** Estender `parity_gate_capability_test.go` com caso que falha se o
   oráculo não discriminar por célula — o cenário que nenhum dos quatro testes atuais cobre
   (`:9`, `:20`, `:38`, `:47`).
4. **Componente 1.** Introduzir a tabela de mapeamento célula → teste e substituir o closure uniforme
   de `evidence.go:147-149` pela resolução por par `(provider, capabilityID)`, mantendo intactas a
   execução (`evidence.go:128-138`) e a leitura de stream (`evidence.go:76-118`). Mapear as 44
   células afirmativas atuais **antes** de trocar o oráculo, no mesmo lote. Célula sem teste real
   identificado não é mapeada à força: é rebaixada e a lacuna fica registrada.
5. **Componente 2.** Após a entrega do `EventKind` de cinco valores da ADR-001, gerar
   `testdata/hook-capability-matrix.json` e `docs/hook-capability-matrix.md` a partir da declaração
   de capability de `internal/hookcontract`, com os estados `verified`, `adapter` (com `limitation`
   obrigatória) e `unsupported`, e golden test espelhando `golden_test.go:11-17` e
   `gate_test.go:68-112` (drift só-JSON, drift só-Markdown, correspondência exata).
6. **Estender o gate de CI.** Incluir a nova matriz no alvo `check-capability-matrix-sync`
   (`Makefile:90-91`), preservando o step existente em `.github/workflows/test.yml:119-120`.
7. **Ligar ao `doctor`** (RF-60, `prd.md:357-360`), que passa a consultar a matriz nova para reportar
   estado por família e por evento, por provedor.

Dependências: o passo 5 depende da ADR-001 (`EventKind` de cinco valores,
`adr-001…:69-70`), hoje não entregue — `enforcement.go:12-16` ainda declara três pontos. O passo 6
depende do 5; o passo 7 depende do 6. Os passos 2, 3 e 4 são independentes da ADR-001.

Critério de conclusão da adoção: nenhuma célula `verified` ou afirmativa sem entrada de mapeamento,
sem confirmação por AST e sem `pass` de execução; o gate falhando quando qualquer uma das três
condições é removida; e o erro de resolução de raiz reportado com causa própria, distinta de
`"no dispatch proof test associated"`.

## Monitoramento e Validação

Sinais a acompanhar:

- Contagem de células por estado nos dois artefatos, observada em diff a cada alteração. Baseline do
  eixo existente: 54 células, `supported` 32, `provider capability` 12, `unknown` 10.
- Contagem de células afirmativas **sem** entrada de mapeamento — deve ser permanentemente zero.
- Tempo de parede do gate de prova por execução, hoje limitado por `dispatchProofRunTimeout = "20m"`
  (`parity_dispatch_proof_test.go:25`), acompanhado nos jobs de `.github/workflows/test.yml`.
- Violações emitidas por `ValidateCapabilityMatrixEvidence` (`parity_gate.go:110-125`), separadas por
  causa: ausência de prova versus falha de resolução de raiz.

Critérios de sucesso:

- Zero células `verified` sem prova indexada nos dois eixos.
- O gate **falha** quando uma célula é promovida manualmente a `verified` no artefato sem a entrada
  de mapeamento correspondente. Este é o teste negativo que valida a decisão; sem ele a adoção não
  está comprovada.
- O gate **falha com causa distinta** quando executado fora da árvore do repositório, em vez de
  reportar ausência de prova.
- `evidence_regression_test.go:45`, `:55` e `:65` continuam verdes, isto é, `skip` e `fail` continuam
  nunca provando e a passagem real continua provando.
- O anti-forja de `parity_dispatch_proof_injection_test.go:24` e `:52` continua verde e continua
  sendo teste de regressão real.

Critério de revisão ou reversão: se o tempo do gate por execução passar a dominar o CI a ponto de o
gate ser desligado, adiado ou marcado como não bloqueante, **a decisão precisa ser revista com
estratégia de amostragem declarada** — qual subconjunto de células é provado por execução em cada
corrida, com que rotação, e qual o intervalo máximo entre duas provas da mesma célula. Amostragem não
declarada é equivalente a prova uniforme e não é aceitável.

## Impacto em Documentação e Operação

- `docs/capability-matrix.md` — a semântica da coluna `Test` passa a refletir prova indexada por
  célula; o cabeçalho gerado por `RenderMarkdown` (`capability.go:140` em diante), que instrui a
  regeneração por `UPDATE_SNAPSHOTS=1 go test ./internal/capability/...`, precisa registrar a nova
  condição de prova.
- `docs/hook-capability-matrix.md` — artefato **novo**, com o eixo `evento × família × provedor` e os
  estados `verified`, `adapter` e `unsupported`, incluindo a coluna `limitation` obrigatória para
  `adapter`.
- `docs/degradation-matrix.md` — alinhar a semântica de degradação aos estados novos, em especial
  `adapter` com `limitation` obrigatória e `unsupported` como estado próprio.
- `docs/evidence-gates.md` — registrar que a prova de dispatch é por execução real e por célula, e
  que falha de resolução de raiz é reportada como erro próprio.
- Seção de hooks do `doctor` (RF-60, `prd.md:357-360`) — saída, exit codes e texto de diagnóstico
  passam a citar estado por família e por evento.
- Runbook de CI — registrar o custo de tempo do gate por execução e o novo diagnóstico de falha de
  ambiente, para que não seja confundido com ausência de teste.

## Revisão Futura

Revisar esta decisão quando qualquer uma destas condições ocorrer:

- O número de células do eixo novo **ultrapassar 100** — limite superior declarado (cinco eventos ×
  cinco famílias × quatro provedores). Acima disso, o custo de manutenção da tabela e o de execução
  exigem reavaliação do mecanismo.
- Uma **CLI nova** entrar no conjunto de provedores, elevando o eixo de quatro para cinco ou mais e
  multiplicando as células por família.
- O catálogo de eventos canônicos mudar além dos cinco de RF-59 (`prd.md:353-356`), ou uma sexta
  família de hook ser admitida contra a restrição de `prd.md:171-174`.
- A ADR-001 ser rejeitada ou substituída — sem `internal/hookcontract` e sem os cinco `EventKind`, o
  componente 2 desta decisão perde objeto e esta ADR precisa ser reescrita.
- O gate por execução ser desligado ou amostrado, conforme o critério de revisão acima; nesse caso a
  substituição desta ADR por outra, que declare a estratégia de amostragem, é obrigatória.
