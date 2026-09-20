# Tarefa 4.0: Contrato canonico de hooks em internal/hookcontract

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar `internal/hookcontract` e `internal/hookaudit`: o pacote que passa a ser a **fonte única** dos
cinco eventos canônicos de hook, do envelope versionado de payload e do resultado tipado de decisão,
conforme [ADR-001](adr-001-hook-contract-fonte-unica-projecoes.md) e
[ADR-002](adr-002-resultado-tipado-traducao-exit-code.md).

**Esta tarefa é PURAMENTE ADITIVA.** Nenhum arquivo do repositório consome `hookcontract` ao final
dela — as projeções entram na tarefa 5.0 e os consumidores nas tarefas 6.0, 10.0, 11.0 e 12.0.
Verificado: `internal/hookcontract/`, `internal/hookpolicy/` e `internal/hookaudit/` **não existem**
hoje. Nenhuma assinatura pública existente muda, nenhum caminho de execução atual é alterado. Isso
torna o **risco de regressão mínimo por construção** — a não-regressão aqui é provada pelo fato de
que a suíte atual passa sem edição em nenhum teste preexistente.

Requisitos cobertos: RF-06, RF-07, RF-09, RF-10, RF-12, RF-13, RF-14, RF-15, RF-16, RF-17.
Dependência: 1.0. Paralelizável: Não.

<requirements>
- RF-06 — `EventKind` é enum fechado de exatamente cinco valores, em fonte única: `EventSessionStart`, `EventBeforeTool`, `EventAfterTool`, `EventBeforeComplete`, `EventSessionEnd`. Nenhum outro lugar do repositório pode declarar o conjunto.
- RF-07 — o pacote `hookcontract` não contém identificador nem literal de provedor. Verificável por gate de AST, não por revisão humana.
- RF-09 — `Envelope` carrega `SchemaVersion` e é decodificado de forma estrita: campo desconhecido produz `ErrUnknownField`, versão não suportada produz `ErrSchemaVersion`. Nunca há aceitação parcial de payload.
- RF-10 — evento não reconhecido produz `ErrUnknownEvent`; nunca é silenciado nem mapeado para um evento vizinho.
- RF-11 (parcial, fechado em 5.0) — `SupportState` tem três valores: `SupportVerified`, `SupportAdapter` e `SupportUnsupported`. `SupportAdapter` exige `limitation` preenchida. `SupportUnsupported` é estado próprio, distinto de sucesso e de falha.
- RF-12, RF-13 — `Result` é VO com construtor validador que **recusa** `DecisionBlock` sem `reason` e sem ao menos um de `policyID` / `gateID`.
- RF-14 — `ToDecision(code int, stderr string, critical bool) Result`: com `critical == true`, qualquer código não mapeado produz `DecisionError`. Fail-closed para hook crítico.
- RF-15 — tradução de exit code acontece em um único ponto: `ExitCodeTranslator`. Os códigos hoje em uso são fixture do teste de não-regressão semântica.
- RF-16 — `internal/hookaudit/` escreve e **lê** `.aispec/hook-decisions.jsonl` (JSONL, append real). O leitor é entregável obrigatório desta tarefa.
- RF-17 — `SchemaVersion` com tabela de migração declarada e teste.
- R-STYLE-001 (hard) — código em inglês, zero comentários, sem prefixo `_` em identificador Go.
- R-DDD-001 — `Result`, `Capability` e `Envelope` são VOs com campos não exportados e construtor validador. Zero struct literal fora de teste e de factory.
- R-ERR-001 — erros sentinela por pacote, wrapping com `%w`, mensagens curtas e estáveis em inglês.
- Zero regressão: nenhum teste preexistente pode ser editado nesta tarefa. Se algum precisar mudar, a tarefa deixou de ser aditiva e isso é defeito de escopo.
</requirements>

## Subtarefas

- [ ] 4.1 Criar `internal/hookcontract/event.go` com `EventKind` (enum fechado de cinco valores,
      `iota + 1`), `Valid()`, `String()` e `ParseEventKind` devolvendo `ErrUnknownEvent`. Assinaturas
      em `techspec.md` seção "Interfaces Chave".
- [ ] 4.2 Criar `internal/hookcontract/capability.go` com `SupportState` (`SupportVerified`,
      `SupportAdapter`, `SupportUnsupported`), o VO `Capability` por par `(provedor, evento)` com
      construtor validador que exige `limitation` não vazia quando o estado é `SupportAdapter`, e a
      interface `CapabilityRegistry`. O campo `provider` é `string` opaca — nunca um enum de nomes de
      provedor, sob pena de quebrar o gate de 4.7.
- [ ] 4.3 Criar `internal/hookcontract/result.go` com `Decision` (cinco valores: `DecisionAllow`,
      `DecisionBlock`, `DecisionWarn`, `DecisionNotApplicable`, `DecisionError`) e o VO `Result` com
      `reason`, `policyID`, `gateID`, `evidence`, `metadata`. `NewResult` recusa `DecisionBlock` sem
      `reason` e sem `policyID`/`gateID`. Construtores auxiliares `NewAllow()` e
      `NewNotApplicable(reason string)`.
- [ ] 4.4 Criar `internal/hookcontract/payload.go` com `Envelope` (`SchemaVersion`, `Event`,
      `Provider`, `SessionID`, `Payload`), `Decoder` e decodificação estrita via
      `json.Decoder.DisallowUnknownFields`. Erros sentinela `ErrUnknownEvent`, `ErrSchemaVersion`,
      `ErrUnknownField`, `ErrProviderUnknown`. Documentar a tabela de migração de
      `SchemaVersion` no `execution_report.md` (RF-17).
- [ ] 4.5 Criar `internal/hookcontract/exitcode.go` com `ExitCodeTranslator`
      (`ToDecision(code int, stderr string, critical bool) Result` e `ToExitCode(Result) int`),
      incluindo a regra fail-closed de RF-14.
- [ ] 4.6 Criar `internal/hookaudit/` com o escritor append-real e o **leitor** de
      `.aispec/hook-decisions.jsonl`, no schema de `techspec.md` seção "Modelos de Dados". Não
      remover nem alterar `.aispec/governance-escapes.log` nesta tarefa — a remoção é tarefa de
      limpeza posterior; aqui os dois coexistem.
- [ ] 4.7 Escrever `TestHookContractHasNoProviderIdentifier`: varre o AST de
      `internal/hookcontract` com `go/parser` e falha se `claude`, `codex`, `copilot` ou `opencode`
      aparecer em qualquer identificador ou literal de string. É o teste que torna RF-07
      verificável.
- [ ] 4.8 Escrever `TestExitCodeTranslator_PreservesCurrentSemantics` com os códigos reais hoje em
      uso como fixture (tabela na seção "Detalhes de Implementação").
- [ ] 4.9 Escrever os demais testes unitários table-driven: `TestEventKind_ParseAndString`,
      `TestCapability_UnsupportedIsNotSuccess`, `TestResult_BlockRequiresReasonAndPolicy`,
      `TestEnvelope_RejectsUnknownField`, `TestEnvelope_RejectsUnsupportedSchemaVersion`,
      `TestEnvelope_RejectsUnknownEvent`, `TestSchemaVersionMigration`, `TestAuditLogRoundTrip`,
      `TestExitCodeTranslator_CriticalUnmappedIsError`. Fuzz sobre `Decode` para RF-09.
- [ ] 4.10 Rodar `make check-mocks`. Se alguma interface nova (`Decoder`, `CapabilityRegistry`,
      `ExitCodeTranslator`) precisar de mock, declará-la em `mockery.yml` na mesma tarefa — gate
      obrigatório da tabela de `AGENTS.md`.
- [ ] 4.11 Registrar no `execution_report.md` a prova de aditividade: `git diff --stat` mostrando
      que nenhum arquivo preexistente foi modificado fora de `mockery.yml`, e a saída de
      `make test lint vet coverage`.

## Detalhes de Implementação

Assinaturas completas em [`techspec.md`](techspec.md), seção "Design de Implementação → Interfaces
Chave". Racional das decisões em [ADR-001](adr-001-hook-contract-fonte-unica-projecoes.md) (fonte
única e projeções) e [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md) (resultado tipado e
tradução única de exit code). **Não duplicar conteúdo desses documentos aqui.**

### Por que um pacote novo em vez de estender `internal/runtime/specs`

Decisão registrada como desvio intencional em `techspec.md` seção "Conformidade com Padrões". A
alternativa aderente — colocar os cinco eventos dentro de `internal/runtime/specs` — foi rejeitada
porque **tornaria o gate de RF-07 impossível**. Verificado no código:

- `internal/runtime/specs/registry.go:17-38` declara `cliHookKeyVocabulary` com as chaves literais
  `"claude"`, `"codex"`, `"copilot"` e `"opencode"`.
- `internal/runtime/specs/registry.go:77-89` declara `agentNativeConfigs` com as mesmas quatro
  chaves literais.

Um pacote que já contém nomes de fornecedor não pode provar por AST que não os contém. O gate de 4.7
falharia no primeiro `go test` se o contrato morasse ali.

### Padrão de VO a seguir

`internal/runtime/specs/enforcement.go:87-110` (`NewPointCoverage`) é o modelo canônico já
estabelecido no repositório: construtor de método do `Catalog`, validação campo a campo com erro
sentinela envolvido por `%w`, retorno de struct com **campos não exportados** e flag `valid`.
`internal/harness/contract.go` é o segundo padrão de referência. Seguir ambos.

### Fixture obrigatória do teste de não-regressão semântica

`TestExitCodeTranslator_PreservesCurrentSemantics` usa os códigos que os validadores praticam hoje.
Todos verificados por leitura direta:

| Origem | Linha verificada | Código |
|---|---|---|
| `.agents/scripts/git-operation-gate.sh` | `:5` — `readonly GIT_OPERATION_BLOCK_EXIT=2` | 2 |
| `.agents/hooks/validate-preload.sh` | `:5` — `readonly PRELOAD_BLOCK_EXIT=2` | 2 |
| `.agents/hooks/validate-session-end.sh` | `:4` — `readonly SESSION_END_BLOCK_EXIT=2` | 2 |
| `.agents/hooks/validate-governance.sh` | `:45` — `exit 1` literal, sem constante | 1 |

O quarto é o outlier que `ExitCodeTranslator` unifica sem alterar comportamento nesta tarefa: o
tradutor precisa mapear **tanto** `1` quanto `2` para `DecisionBlock` na leitura, porque os dois
significam bloqueio hoje. Nenhum script é editado aqui — a substituição do `exit 1` é da tarefa 13.0.

Espelho: `.claude/hooks/validate-governance.sh:45` tem o mesmo `exit 1`;
`.agents/scripts/validate-session-end.sh:4` é o espelho de `.agents/hooks/validate-session-end.sh:4`.

### `internal/hookaudit` — por que o leitor é entregável, não opcional

Estado atual verificado: `.aispec/governance-escapes.log` tem **escritores e nenhum leitor**. Os
três pontos de escrita são `.agents/hooks/validate-preload.sh:85`,
`.agents/scripts/git-operation-gate.sh:45` e `.agents/scripts/git-operation-gate.sh:57` (mais os
espelhos em `.claude/scripts/`). Nenhum `.go`, `.sh` ou `.js` do repositório lê esse arquivo.

Além disso: o formato é TSV gravado por `printf '%s\t%s\t%s\t%s\t%s\n'`
(`git-operation-gate.sh:62-63`) com `$command_text` no meio — **um `\t` dentro do comando corrompe a
linha**; não há rotação; e o diretório está coberto por `.gitignore:34` (`.aispec/`), logo a
evidência nunca entra no histórico. `hook-decisions.jsonl` resolve os três: JSONL é robusto a
qualquer conteúdo no campo, o leitor fecha o ciclo de auditoria e a estrutura permite rotação.

### Restrições de desenho

- R-DDD-001: zero struct literal de `Result`, `Capability` ou `Envelope` fora de teste e de factory.
- R-ERR-001: erros sentinela declarados no pacote; `fmt.Errorf("context: %w", err)` para wrapping.
- R-STYLE-001: inglês, zero comentários, sem `_` como prefixo de identificador. Os termos em
  português do PRD traduzem no código — `Evento` → `Event`, `Resultado` → `Result`,
  `Política` → `Policy`. A rastreabilidade RF↔código fica no `execution_report.md`.
- R-TEST-001: table-driven; `FakeFileSystem` (`internal/fs/fake.go`) no unitário, nunca o FS real;
  `io.Discard` no `Printer`.

## Critérios de Sucesso

- [ ] Os cinco eventos canônicos existem em fonte única e `EventKind` é enum fechado (RF-06).
- [ ] `TestHookContractHasNoProviderIdentifier` passa e falha comprovadamente quando um nome de
      provedor é introduzido no pacote (demonstrar com uma execução deliberadamente vermelha
      registrada na evidência).
- [ ] `Envelope` rejeita campo desconhecido, versão não suportada e evento desconhecido com o erro
      sentinela correspondente. Zero caminho de aceitação parcial (RF-09, RF-10, RF-17).
- [ ] `NewResult` recusa `DecisionBlock` sem `reason` e sem `policyID`/`gateID` (RF-12, RF-13).
- [ ] `ToDecision` com `critical == true` e código não mapeado devolve `DecisionError` (RF-14).
- [ ] `TestExitCodeTranslator_PreservesCurrentSemantics` passa com a fixture dos quatro códigos
      reais (RF-15).
- [ ] `.aispec/hook-decisions.jsonl` tem escritor e leitor, com round-trip provado (RF-16).
- [ ] `SupportUnsupported` é estado próprio, distinto de sucesso e de falha, provado por teste.

### Gates de não-regressão — obrigatórios, requisito inegociável

- [ ] `make test` — verde.
- [ ] `make lint` — verde.
- [ ] `make vet` — verde.
- [ ] `make check-mocks` — verde (área tocada: `mockery.yml`, tabela de `AGENTS.md`).
- [ ] `make coverage` — 75% total e 70% por pacote crítico preservados; `internal/hookcontract` e
      `internal/hookaudit` nascem acima do piso.
- [ ] **Prova de aditividade:** `git diff --name-only` não lista nenhum arquivo `.go` preexistente.
      Nenhum teste preexistente editado. Se algum precisou mudar, a tarefa falhou o critério de
      escopo e deve ser reavaliada antes de `done`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários — `TestEventKind_ParseAndString`,
      `TestCapability_UnsupportedIsNotSuccess`, `TestResult_BlockRequiresReasonAndPolicy`,
      `TestEnvelope_RejectsUnknownField`, `TestEnvelope_RejectsUnsupportedSchemaVersion`,
      `TestEnvelope_RejectsUnknownEvent`, `TestSchemaVersionMigration`,
      `TestExitCodeTranslator_CriticalUnmappedIsError`,
      `TestExitCodeTranslator_PreservesCurrentSemantics`, `TestAuditLogRoundTrip`,
      `TestHookContractHasNoProviderIdentifier`, mais fuzz sobre `Decode`.
- [ ] Testes de integração — não se aplicam nesta tarefa: o pacote é aditivo e não tem consumidor.
      A integração entra em 5.0 (`tests/integration/hook_contract_projection_test.go`). Registrar
      essa ausência como decisão consciente no `execution_report.md`, não como lacuna.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

**A criar:**
- `internal/hookcontract/event.go`
- `internal/hookcontract/capability.go`
- `internal/hookcontract/result.go`
- `internal/hookcontract/payload.go`
- `internal/hookcontract/exitcode.go`
- `internal/hookcontract/*_test.go` (inclui o gate de AST)
- `internal/hookaudit/` (escritor, leitor e testes)

**A modificar (apenas se surgir interface a mockar):**
- `mockery.yml`

**Leitura obrigatória — contrato e racional:**
- `.specs/prd-hooks-canonicos-vendor-neutral/prd.md` (RF-06, RF-07, RF-09, RF-10, RF-12 a RF-17)
- `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md` (Interfaces Chave, Modelos de Dados)
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-001-hook-contract-fonte-unica-projecoes.md`
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-002-resultado-tipado-traducao-exit-code.md`

**Leitura de referência — padrão de VO e prova do desvio de pacote:**
- `internal/runtime/specs/enforcement.go:87-110` — modelo de construtor validador
- `internal/harness/contract.go` — segundo padrão de VO
- `internal/runtime/specs/registry.go:17-38,77-89` — nomes de provedor que impedem hospedar o
  contrato em `specs`

**Fixture do teste de não-regressão (somente leitura nesta tarefa):**
- `.agents/scripts/git-operation-gate.sh:5,45,57,62-63`
- `.agents/hooks/validate-preload.sh:5,85`
- `.agents/hooks/validate-session-end.sh:4`
- `.agents/hooks/validate-governance.sh:45`
- `.gitignore:34`
