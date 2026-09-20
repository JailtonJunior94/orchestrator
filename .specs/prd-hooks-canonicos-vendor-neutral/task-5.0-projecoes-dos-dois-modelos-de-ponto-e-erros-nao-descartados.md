# Tarefa 5.0: Projecoes dos dois modelos de ponto, correcao de Kind e erros nao descartados

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Converter os **dois modelos de ponto de hook** que coexistem hoje em projeções derivadas de
`internal/hookcontract`, entregue na tarefa 4.0, **preservando integralmente a superfície pública de
cada um** e instalando um gate que falha se qualquer projeção divergir do contrato.

No mesmo lote entram duas correções que só são seguras agora: o `Kind()` que mente em dois eventos
do dispatcher, e os dois erros de dispatch descartados em `internal/runtime/runner.go`.

Requisitos cobertos: RF-08, RF-11. Dependência: 4.0. Paralelizável com 7.0.

<requirements>
- RF-08 — `specs.CanonicalPoint` e `hooks.Point*` passam a ser projeções derivadas de `hookcontract.EventKind`, com gate automatizado que falha na divergência. A superfície pública dos dois pacotes é preservada: nenhum identificador exportado é renomeado ou removido.
- RF-11 — `SupportUnsupported` é estado próprio. `NewEnforcement` deixa de exigir exatamente três pontos e passa a exigir **cobertura declarada por evento**, onde declarar `SupportUnsupported` é declaração válida e completa.
- Corrigir o bug preexistente de `Kind()` em `PromptBuildEvent` e `ToolCallEvent`, que hoje retorna sempre a constante da fase `pre`.
- Deixar de descartar o erro de dispatch em `internal/runtime/runner.go:428` e `:466`.
- Os dez pontos de teste que asseram a contagem de três entram **no mesmo lote**, nunca em commit separado.
- Zero regressão: `make test lint vet integration coverage` verdes ao fim. Nenhum comportamento observável de produção muda além das duas correções declaradas.
- R-STYLE-001 (hard) — código em inglês, zero comentários, sem prefixo `_` em identificador Go. Os arquivos tocados **contêm comentários em português hoje** (`dispatcher.go:19-20,26,73,78,86`, `runner.go:420-422`): remover os comentários das linhas efetivamente alteradas pelo diff, conforme R-STYLE-001.2.
</requirements>

## Subtarefas

- [ ] 5.1 Mapear, em código, `hookcontract.EventKind` → `specs.CanonicalPoint` (modelo A) e
      `hookcontract.EventKind` → `hooks.Point*` (modelo B). A tradução é de mão única a partir do
      contrato; os dois modelos passam a derivar, não a declarar.
- [ ] 5.2 Relaxar `NewEnforcement` (`internal/runtime/specs/enforcement.go:123`) de "exatamente os
      três pontos de `canonicalPoints`" para "cobertura declarada por evento", aceitando
      `SupportUnsupported` como declaração válida. O laço a substituir é `enforcement.go:137-141`.
- [ ] 5.3 Ajustar, **no mesmo lote**, os dez pontos de teste que codificam a contagem de três
      (tabela verificada na seção "Detalhes de Implementação"). Nenhum deles pode ser adiado: o
      relaxamento sozinho derruba a suíte por `panic` no init do catálogo
      (`registry.go:296` e `registry.go:307`).
- [ ] 5.4 Corrigir `PromptBuildEvent.Kind()` (`internal/runtime/hooks/dispatcher.go:77`) para
      refletir a fase real de despacho, e não retornar sempre `PointPromptPreBuild`.
- [ ] 5.5 Corrigir `ToolCallEvent.Kind()` (`internal/runtime/hooks/dispatcher.go:85`) para refletir
      `Phase`, e não retornar sempre `PointToolCallPreDispatch`.
- [ ] 5.6 Deixar de descartar o erro do dispatch em `internal/runtime/runner.go:428` e `:466`,
      propagando no mesmo padrão já usado em `runner.go:373-381` para os dois pontos de
      `prompt.pre_build` / `prompt.post_build`.
- [ ] 5.7 Criar `tests/integration/hook_contract_projection_test.go`: o gate que falha se qualquer
      um dos dois modelos divergir do contrato — em cardinalidade, em nome ou em ordem.
- [ ] 5.8 Escrever os testes unitários das duas correções de `Kind()` e dos dois pontos de
      propagação de erro, provando que o erro que antes era silencioso agora chega ao chamador.
- [ ] 5.9 Verificar que o pacote novo de integração cai em um job que o CI executa
      (`.github/workflows/test.yml`, job `integration`) — armadilha de "gate órfão" herdada do PRD
      dependente e registrada em `tasks.md` seção "Riscos de Integração".
- [ ] 5.10 Rodar `make test lint vet integration coverage` e persistir as saídas no
      `execution_report.md`.

## Detalhes de Implementação

Racional completo em [ADR-001](adr-001-hook-contract-fonte-unica-projecoes.md) (fonte única e
projeções derivadas) e em [`techspec.md`](techspec.md) seção "Sequenciamento de Desenvolvimento →
Fase 2". **Não duplicar aqui.**

### Estado verificado dos dois modelos

**Modelo A — `internal/runtime/specs`.** Enum fechado `CanonicalPoint` com três valores declarados em
`enforcement.go:12-16` (`PointPreTool`, `PointPostTool`, `PointSessionEnd`) e o slice
`canonicalPoints` em `enforcement.go:18-22`. Usado para confrontar a configuração nativa das quatro
CLIs e sustentar o gate de paridade.

**Modelo B — `internal/runtime/hooks`.** Sete constantes string em `dispatcher.go:19-27`
(`PointRuntimePreOpen` a `PointSessionPostReview`), às quais somam-se **mais sete** pontos de memória
em `memory_events.go:3-11` (`PointMemoryFactRecorded` a `PointMemoryBatonTransferred`). **O espaço
real do modelo B é de catorze pontos, não sete** — fato confirmado por leitura e registrado na
ADR-001. Qualquer projeção que trate o modelo B como tendo sete pontos está errada.

**Os dois são disjuntos.** Nenhum arquivo do repositório importa `specs.CanonicalPoint` e
`hooks.Point*` juntos. Não existe tradução, não existe gate de coerência — que é exatamente o que
esta tarefa instala.

### Superfície de mudança de `NewEnforcement` — pequena e enumerada

Os call-sites de produção de `NewPointCoverage` e `NewEnforcement` são **exatamente três**, todos em
`internal/runtime/specs/registry.go`, e todos verificados:

| Linha | Chamada |
|---|---|
| `registry.go:294` | `enf, err := c.NewEnforcement(coverage)` dentro de `canonicalEnforcement` (`:288`) |
| `registry.go:306` | `cov, err := c.NewPointCoverage(...)` dentro de `mustCoverage` (`:301`) |
| `registry.go:346` | `enf, err = c.NewEnforcement(enf.Coverage(), preconditions...)` |

As falhas nesses pontos são `panic`, não erro devolvido: `registry.go:296` e `registry.go:307`. Por
isso relaxar `NewEnforcement` sem ajustar os testes no mesmo lote derruba a suíte inteira no init do
catálogo — risco R-01 de `techspec.md`.

`ParseCanonicalPoint` (`enforcement.go:74-85`) **não tem consumidor de produção** — apenas teste.
Verificado por busca no repositório. É o ponto de menor risco para receber a tradução a partir do
contrato.

### Correção de linha da descrição original

A descrição de origem apontava `NewEnforcement` em `enforcement.go:139-143`. **Verificado e
corrigido:** a função começa em `enforcement.go:123`; o laço que exige os três pontos fixos ocupa
`enforcement.go:137-141`. A linha 139 é apenas o `return` de erro dentro desse laço.

### Os dez pontos de teste que asseram a contagem de três

Todos verificados por leitura direta. Note que três deles **não são um literal `!= 3`** e por isso
exigem atenção diferente:

| Arquivo:linha | Conteúdo verificado | Natureza |
|---|---|---|
| `internal/runtime/specs/registry_test.go:131` | `if len(points) != 3 {` | Literal |
| `internal/runtime/specs/registry_test.go:141` | `if len(enf.Coverage()) != len(points) {` | Derivado de `points` |
| `internal/install/hooks_parity_matrix_test.go:95` | `if len(points) != 3 {` | Literal |
| `internal/install/hooks_parity_matrix_test.go:101` | `expectedCells := len(mandatoryMatrixAgents) * len(points)` | Aritmética derivada |
| `internal/runtime/specs/parity_dispatch_proof_test.go:280` | `if len(byPoint) != len(points) {` | Derivado |
| `internal/install/precondition_report_test.go:35` | `if len(items) != 3 {` | Literal |
| `internal/install/precondition_report_test.go:66` | `want := []string{"PreToolUse", "PostToolUse", "Stop"}` | **Não é `!= 3`**: é a lista das três chaves nativas de `claude`, codificada literalmente |
| `internal/runtime/precondition/precondition_test.go:333` | `if len(report.Points) != 3 {` | Literal |
| `tests/integration/hooks_live/matrix.go:20` | `cells := make([]MatrixCell, 0, len(LiveAgentIDs)*len(catalog.CanonicalPoints()))` | **Não é `!= 3`**: já deriva de `CanonicalPoints()`. Afetado por cardinalidade, não por literal |
| `tests/integration/hooks_live/live_test.go:147` | `case specs.PointPreTool:` | **Não é `!= 3`**: é um `switch` que precisa cobrir todos os pontos; ampliar sem tratar o novo caso produz comportamento mudo |
| `tests/integration/hooks_live/live_test.go:393` | `func nativePrompt(point specs.CanonicalPoint) string` | Mesma natureza de `:147` — despacho por ponto |

São onze linhas em dez arquivos/pontos distintos. Os `switch` de `live_test.go` são o risco real: um
`switch` sem `default` explícito sobre um enum ampliado falha em silêncio, não em vermelho.

### Bug preexistente de `Kind()`

Verificado:

- `dispatcher.go:77` — `func (e PromptBuildEvent) Kind() string { return PointPromptPreBuild }`.
  Retorna `prompt.pre_build` **mesmo quando despachado em `post_build`**: `runner.go:377` faz
  `disp.Dispatch(ctx, hooks.PointPromptPostBuild, hooks.PromptBuildEvent{...})` com o mesmo tipo.
- `dispatcher.go:85` — `func (e ToolCallEvent) Kind() string { return PointToolCallPreDispatch }`.
  Retorna `tool_call.pre_dispatch` **mesmo em `post_complete`**: `runner.go:466` despacha
  `hooks.PointToolCallPostComplete` com `ToolCallEvent{Phase: "post_complete"}`.

Em ambos, **só o campo `Phase` distingue as fases**, e `PromptBuildEvent` não tem sequer esse campo
(`dispatcher.go:72-75` declara apenas `Prompt` e `Spec`). A correção precisa decidir entre
acrescentar `Phase` a `PromptBuildEvent` ou derivar o `Kind` do ponto de despacho — a decisão fica no
`execution_report.md` com a justificativa.

Risco R-11 de `techspec.md`: a mudança pode alterar comportamento de consumidor não identificado. A
mitigação verificada é que o único uso de constante de ponto em `dispatcher_test.go` é a linha `:114`
(`hooks.PointSessionPostEnd`), que não é afetada. A correção entra com teste explícito.

### Erros de dispatch descartados

Verificado — as duas únicas ocorrências de descarte no arquivo:

```
runner.go:428    _ = disp.Dispatch(ctx, hooks.PointToolCallPreDispatch, hooks.ToolCallEvent{Phase: "pre_dispatch"})
runner.go:466    _ = disp.Dispatch(ctx, hooks.PointToolCallPostComplete, hooks.ToolCallEvent{Phase: "post_complete"})
```

Contraste no mesmo arquivo: `runner.go:371-381` propaga o erro dos dois dispatches de `prompt.*` com
`fmt.Errorf("runner: hook prompt.pre_build: %w", err)`. O padrão já existe; os dois pontos de
tool-call é que fogem dele.

**Por que corrigir AQUI e não depois.** Hoje **nenhum hook de produção está registrado nesses dois
pontos**, então propagar o erro é uma mudança inerte: não há caminho em que ela altere o resultado
de uma execução real. Se a correção esperar a tarefa 10.0 registrar o primeiro quality gate ali, ela
deixa de ser inerte e passa a **bloquear o que hoje não bloqueia** — risco R-12 de `techspec.md`,
classificado como Alto. A janela de segurança é agora.

### Preservação de superfície pública

Nenhum identificador exportado de `internal/runtime/specs` ou `internal/runtime/hooks` pode ser
renomeado, removido ou ter assinatura alterada nesta tarefa. `PointPreTool`, `PointPostTool`,
`PointSessionEnd`, as sete constantes de `dispatcher.go` e as sete de `memory_events.go` continuam
existindo com os mesmos nomes e os mesmos valores string. O que muda é **de onde o valor deriva**,
não o valor.

## Critérios de Sucesso

- [ ] Os dois modelos derivam de `hookcontract.EventKind` e a superfície pública de ambos está
      byte-a-byte preservada nos identificadores exportados (RF-08).
- [ ] `tests/integration/hook_contract_projection_test.go` existe, passa, e **falha
      comprovadamente** quando um ponto é adicionado a um modelo e não ao contrato — demonstrar a
      execução vermelha na evidência.
- [ ] `NewEnforcement` aceita cobertura declarada por evento, com `SupportUnsupported` como
      declaração válida, e não exige mais exatamente três pontos (RF-11).
- [ ] `PromptBuildEvent.Kind()` e `ToolCallEvent.Kind()` refletem a fase real de despacho, com
      teste que falharia na implementação anterior.
- [ ] `runner.go:428` e `:466` propagam o erro de dispatch; teste prova que um hook que falha nesses
      pontos agora chega ao chamador.
- [ ] Os onze pontos de teste da tabela foram ajustados **no mesmo commit** do relaxamento. Nenhum
      `panic` no init do catálogo.
- [ ] Os `switch` de `tests/integration/hooks_live/live_test.go:147,393` tratam explicitamente todo
      ponto novo — nenhum caso cai em silêncio.
- [ ] O pacote de integração novo é executado por um job real do CI (`.github/workflows/test.yml`).

### Gates de não-regressão — obrigatórios, requisito inegociável

- [ ] `make test` — verde.
- [ ] `make lint` — verde.
- [ ] `make vet` — verde.
- [ ] `make integration` — verde (build tag `integration`; área tocada: `tests/integration/`).
- [ ] `make coverage` — 75% total e 70% por pacote crítico preservados.
- [ ] **Prova de preservação de superfície:** diff dos símbolos exportados de
      `internal/runtime/specs` e `internal/runtime/hooks` antes e depois, vazio para remoções e
      renomeações. Persistir no `execution_report.md`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários — projeção do modelo A e do modelo B a partir de `hookcontract`;
      `NewEnforcement` com cobertura declarada e com `SupportUnsupported`; `Kind()` de
      `PromptBuildEvent` e de `ToolCallEvent` nas duas fases; propagação de erro nos dois pontos de
      `runner.go`. Os onze pontos de teste preexistentes ajustados continuam verdes.
- [ ] Testes de integração — `tests/integration/hook_contract_projection_test.go`: gate que falha se
      qualquer um dos dois modelos divergir do contrato em cardinalidade, nome ou ordem. Mais a
      suíte `tests/integration/hooks_live/` ajustada.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

**A criar:**
- `tests/integration/hook_contract_projection_test.go`

**A modificar:**
- `internal/runtime/specs/enforcement.go:12-22,74-85,123,137-141`
- `internal/runtime/specs/registry.go:288-299,346`
- `internal/runtime/hooks/dispatcher.go:19-27,77,85`
- `internal/runtime/hooks/memory_events.go:3-11`
- `internal/runtime/runner.go:428,466`
- `internal/runtime/specs/registry_test.go:131,141`
- `internal/install/hooks_parity_matrix_test.go:95,101`
- `internal/runtime/specs/parity_dispatch_proof_test.go:280`
- `internal/install/precondition_report_test.go:35,66`
- `internal/runtime/precondition/precondition_test.go:333`
- `tests/integration/hooks_live/matrix.go:20`
- `tests/integration/hooks_live/live_test.go:147,393`
- `.github/workflows/test.yml` — apenas se o pacote novo não cair em job existente

**Consumido, não modificado (entregue na tarefa 4.0):**
- `internal/hookcontract/`

**Leitura obrigatória:**
- `.specs/prd-hooks-canonicos-vendor-neutral/prd.md` (RF-08, RF-11)
- `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md` (Fase 2; riscos R-01, R-11, R-12)
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-001-hook-contract-fonte-unica-projecoes.md`

**Referência de padrão de propagação de erro:**
- `internal/runtime/runner.go:371-381`
