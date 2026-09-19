# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Canonical Hook Contract v1 como fonte única, com os modelos existentes convertidos em projeções derivadas
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório
- **Relacionados:** PRD [`prd.md`](prd.md) (RF-06, RF-07, RF-08, RF-09, RF-10, RF-11, RF-17), [`techspec.md`](techspec.md), [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md), [ADR-004](adr-004-prova-dispatch-por-celula.md), [ADR-009 do repositório](../adr/009-acp-protocol-adoption.md), [ADR-014 do repositório](../adr/014-claude-cli-acp-native.md)

## Contexto

O repositório tem **dois modelos de ponto de hook coexistindo, e eles não se conhecem**.

O primeiro vive em `internal/runtime/specs/enforcement.go:10-22`: um enum fechado `CanonicalPoint`
com três valores — `PointPreTool`, `PointPostTool`, `PointSessionEnd` — usado para confrontar a
configuração nativa das quatro CLIs e sustentar o gate de paridade. O segundo vive em
`internal/runtime/hooks/dispatcher.go:19-27`: sete constantes string usadas pelo dispatcher
in-process do modo orquestrado ACP, às quais se somam mais sete pontos de memória declarados em
`internal/runtime/hooks/memory_events.go:3-11` — o espaço real do segundo modelo é de catorze pontos,
não sete.

Os dois são **disjuntos em código**: não existe nenhum arquivo que importe `specs.CanonicalPoint` e
`hooks.Point*` juntos. Não há tradução, não há gate que verifique coerência, e nada impede que um
evoluia e o outro não.

Nenhum dos dois expressa o que a User Story exige. Faltam `SessionStart` e `BeforeComplete` como
conceitos nomeados; não há schema versionado de payload; e — o mais grave — **não existe forma de um
provedor declarar que não suporta um evento**. Hoje a ausência de suporte simplesmente não aparece
em lugar nenhum, o que é a definição operacional da falsa paridade que o princípio P07 proíbe.

Três fatos adicionais, todos verificados, moldam a decisão:

1. **`PointSessionEnd` tem o nome errado.** `internal/runtime/specs/registry.go:17-38` mapeia esse
   ponto para `Stop`/`SubagentStop` no Claude e no Codex, `agentStop` no Copilot e `session.idle` no
   OpenCode. A pesquisa na documentação oficial das quatro CLIs confirmou que **todos esses eventos
   são fim de turno**, não fim de sessão — isto é, correspondem a `BeforeComplete`. Claude, Codex e
   Copilot têm eventos `SessionEnd`/`sessionEnd` nativos e distintos, que o repositório **ignora
   integralmente**. Na prática, `.agents/scripts/validate-session-end.sh` roda a cada turno, apesar
   do nome.

   Isso é boa notícia para a migração: o ponto mais difícil dos cinco já está coberto, sob outro
   nome.

2. **A superfície a preservar é pequena.** `NewPointCoverage` e `NewEnforcement` têm exatamente três
   call-sites de produção, todos em `internal/runtime/specs/registry.go:294,306,346`. Fora deles, só
   dois arquivos de teste no mesmo pacote. E `ParseCanonicalPoint` (`enforcement.go:74-85`) **não tem
   nenhum consumidor de produção** — só `enforcement_test.go:110,114`.

3. **Há um bug preexistente de identidade de evento no segundo modelo.**
   `dispatcher.go:77` faz `PromptBuildEvent.Kind()` retornar sempre `prompt.pre_build`, inclusive
   quando o evento é despachado em `prompt.post_build` (`runner.go:377`). O mesmo em
   `dispatcher.go:85`: `ToolCallEvent.Kind()` retorna sempre `tool_call.pre_dispatch`, inclusive em
   `post_complete` (`runner.go:466`). Só o campo `Phase` distingue. Qualquer fonte única terá de
   resolver isso explicitamente.

O custo de não decidir é que a User Story fica inatendível por construção: sem um lugar onde os cinco
eventos existam, RF-06 a RF-11 não têm onde pousar, e RF-08 — reconciliação verificável — não tem
objeto.

## Decisão

Criar `internal/hookcontract` como **fonte única de verdade** dos eventos canônicos, e converter os
dois modelos existentes em **projeções derivadas**, preservando integralmente a superfície pública de
cada um.

O pacote possui, e é o único a possuir:

- `EventKind` — enum fechado de cinco valores: `EventSessionStart`, `EventBeforeTool`,
  `EventAfterTool`, `EventBeforeComplete`, `EventSessionEnd`.
- `Envelope` — payload com `SchemaVersion` explícito e decodificação estrita: campo desconhecido,
  versão não suportada e evento desconhecido produzem erro tipado, nunca aceitação parcial.
- `Capability` — declaração por par `(provedor, evento)` com `SupportState` de três valores:
  `SupportVerified`, `SupportAdapter` (com `limitation` obrigatoriamente preenchida) e
  `SupportUnsupported`. **`SupportUnsupported` é estado próprio**, distinto de sucesso e de falha.
- `Result` e `Decision` — objeto da [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md).

O escopo da decisão inclui quatro consequências diretas:

**Primeira — inversão de dependência estrita.** `hookcontract` não importa `internal/runtime/specs`,
`internal/runtime/hooks`, `internal/skills` nem qualquer pacote que conheça um provedor. Isso é
verificado por gate, não por convenção: `TestHookContractHasNoProviderIdentifier` varre o AST do
pacote e falha se `claude`, `codex`, `copilot` ou `opencode` aparecer em identificador ou em literal.
RF-07 passa a ser propriedade demonstrável.

**Segunda — `CanonicalPoint` vira projeção.** `specs.CanonicalPoint` mantém nome, valores e toda a
API pública, mas passa a derivar de `EventKind`. `NewEnforcement` deixa de exigir *exatamente três
pontos* e passa a exigir **cobertura declarada para todos os eventos**, onde `SupportUnsupported` é
declaração válida. Essa é a mudança semântica central: hoje a ausência de cobertura é erro
(`ErrIncompleteCoverage`, `enforcement.go:139-143`); depois, a ausência de *declaração* é que será
erro, e a declaração de não suporte será resposta legítima.

**Terceira — o renomeio de `PointSessionEnd`.** O ponto hoje chamado `session-end` passa a se chamar
`BeforeComplete`, que é o que ele sempre foi. O `SessionEnd` real de Claude, Codex e Copilot entra
como evento novo. O OpenCode declara `SupportUnsupported` para `SessionEnd` e `SupportAdapter` com
limitação para `BeforeComplete`, porque `session.idle` é observacional e não bloqueia. O artefato
`validate-session-end.sh` mantém o nome de arquivo para não quebrar os espelhos e os gates de sync;
o renomeio do arquivo é trabalho separado e explicitamente adiado.

**Quarta — correção do `Kind()`.** `PromptBuildEvent.Kind()` e `ToolCallEvent.Kind()` passam a
refletir o ponto real de despacho.

## Alternativas Consideradas

**A1 — Estender `internal/runtime/specs` com os cinco eventos, sem pacote novo.**
*Vantagens:* nenhum pacote a mais; toda a lógica de confronto de configuração nativa já está ali;
menor movimentação de código.
*Desvantagens:* `specs` já importa `internal/skills` e já contém identificadores de provedor em
`cliHookKeyVocabulary` (`registry.go:17-38`) e em `agentNativeConfigs` (`registry.go:77-89`). RF-07
exige que o contrato não importe nomes de fornecedor, e um pacote que já os contém **não pode provar
isso por gate**. A verificação viraria uma allowlist de exceções, que é o oposto de uma garantia.
*Rejeitada* porque tornaria o requisito central não verificável.

**A2 — Estender `internal/runtime/hooks` e fazer `specs` consumir esse pacote.**
*Vantagens:* o dispatcher já tem quatorze pontos e um mecanismo de fan-out testado.
*Desvantagens:* `hooks` é um pacote de *runtime*: seus eventos carregam ponteiro mutável para o
prompt (`dispatcher.go:73`), referência a `events.NormalizedToolCall` e `any` para o `Summary`. São
tipos de execução, não de contrato. Fazer `specs` — que roda em gate de CI, sem runtime — depender
disso inverteria a direção correta da dependência e arrastaria o SDK ACP para dentro do gate de
paridade.
*Rejeitada* por acoplamento indevido.

**A3 — Unificar os dois modelos em um só, eliminando um deles.**
*Vantagens:* fonte única de verdade real, sem projeções; conceitualmente mais limpo.
*Desvantagens:* os dois modelos servem a propósitos genuinamente distintos. `specs` descreve *o que a
CLI oferece e como a configuração nativa o declara*; `hooks` descreve *o que o runtime orquestrado
executa in-process*. Quatro dos sete pontos de `hooks` não têm equivalente em CLI nenhuma —
`prompt.pre_build`, `prompt.post_build`, `session.post_review` e os sete de memória são conceitos do
harness, não do provedor. Unificar forçaria a inventar eventos canônicos falsos para acomodá-los, ou
a perder funcionalidade.
*Rejeitada* porque a unificação destruiria informação.

**A4 — Manter os dois modelos e apenas adicionar um teste que compare os nomes.**
*Vantagens:* custo quase nulo.
*Desvantagens:* um teste de comparação de strings não impede divergência semântica, não cria os dois
eventos faltantes, não introduz payload versionado e não permite declarar não suporte. Atenderia a
aparência de RF-08 e nenhum dos demais.
*Rejeitada* por não resolver o problema.

**A5 — Manter três eventos canônicos em vez de cinco, alegando que é o que as CLIs garantem.**
*Vantagens:* zero migração; nenhum teste quebra.
*Desvantagens:* contraria RF-06 diretamente, e a pesquisa oficial mostrou que três das quatro CLIs
têm `SessionStart` e `SessionEnd` nativos — a limitação seria autoimposta, não técnica.
*Rejeitada.*

## Consequências

### Benefícios Esperados

- Os cinco eventos passam a existir em um lugar só, com semântica documentada por evento.
- A não equivalência entre provedores vira **dado declarado e auditável** em vez de ausência
  silenciosa, atendendo P07 por construção.
- A independência de fornecedor deixa de ser convenção e vira propriedade verificada por gate de AST.
- A divergência entre os dois modelos passa a quebrar o build em vez de passar despercebida.
- Dois bugs preexistentes de identidade de evento são corrigidos com teste.
- O evento mais difícil — `BeforeComplete` — já está coberto nas quatro CLIs, o que reduz
  substancialmente o risco da entrega.

### Trade-offs e Custos

- **Um pacote a mais** e duas camadas de projeção. É complexidade real, justificada por RF-07 ser
  verificável só assim.
- **Um período com três vocabulários em circulação**: `CanonicalPoint`, as constantes de `hooks` e
  `EventKind`. A mitigação é o gate de projeção, não a disciplina.
- **O renomeio conceitual de `session-end` para `BeforeComplete` é confuso por um release**, porque o
  arquivo em disco continua se chamando `validate-session-end.sh`. Essa dívida é assumida
  conscientemente: renomear o arquivo tocaria os quatro espelhos, três listas de sync e doze arquivos
  de teste, sem ganho funcional.
- `SupportUnsupported` exige preencher a matriz para pares que hoje simplesmente não existem —
  trabalho de declaração que não produz funcionalidade imediata.

### Riscos e Mitigações

| Risco | Impacto | Mitigação | Rollback |
|---|---|---|---|
| Relaxar `NewEnforcement` de três pontos fixos causa `panic` no init do catálogo (`registry.go:296,307`), derrubando praticamente toda a suíte | Alto | Os testes que asseram a contagem estão enumerados e são tocados no mesmo lote: `registry_test.go:131,141`, `hooks_parity_matrix_test.go:95,101`, `parity_dispatch_proof_test.go:280`, `precondition_report_test.go:35,66`, `precondition_test.go:333`, `hooks_live/matrix.go:20`, `live_test.go:147,393` | O relaxamento é uma mudança localizada em `enforcement.go`; reverter restaura a exigência de três |
| Ampliar a matriz de 4×3 para 4×5 multiplica as células de prova de dispatch de 12 para 20 | Médio | A [ADR-004](adr-004-prova-dispatch-por-celula.md) trata o mecanismo de prova; células `SupportUnsupported` produzem skip registrado, não exigem teste | Reduzir o conjunto de eventos obrigatórios sem tocar o contrato |
| Corrigir `Kind()` altera comportamento de consumidor não identificado | Médio | Só `dispatcher_test.go:114` usa a constante; a correção entra com teste explícito do ponto real de despacho | Correção isolada, revertível |
| O gate de AST produz falso positivo ao encontrar um nome de provedor em comentário ou em nome de teste | Baixo | R-STYLE-001 já proíbe comentários no código; o gate varre identificadores e literais do pacote, que não contém testes de integração | Ajuste do escopo da varredura |
| Declarar `SupportUnsupported` para OpenCode em `SessionEnd` reduz a cobertura aparente do produto | Baixo | É correção de uma afirmação falsa, não perda. A matriz passa a dizer a verdade | Não aplicável |

## Plano de Implementação

1. **Criar `internal/hookcontract` de forma puramente aditiva** — `EventKind`, `Envelope`,
   `Capability`, `SupportState`, com testes de tabela e fuzz no decoder. Nenhum consumidor ainda.
   Nesta etapa nada pode quebrar, por construção.
2. **Instalar o gate de independência de fornecedor** (`TestHookContractHasNoProviderIdentifier`),
   que deve passar desde o primeiro commit do pacote.
3. **Relaxar `NewEnforcement`** para cobertura declarada por evento, em lote único com os dez
   arquivos de teste que asseram a contagem de três.
4. **Converter `specs.CanonicalPoint` em projeção**, preservando nome, valores e API. Remover
   `ParseCanonicalPoint` ou reapontá-lo — não tem consumidor de produção, é o ponto de menor risco.
5. **Converter os pontos de `hooks` em projeção** e corrigir `Kind()` nos dois tipos afetados.
6. **Instalar o gate de projeção** (`hook_contract_projection_test.go`), que falha se qualquer um dos
   dois modelos divergir do contrato.
7. **Declarar `SessionStart` e o `SessionEnd` real** em `cliHookKeyVocabulary`, com os estados de
   suporte confirmados pela documentação oficial de cada CLI.
8. **Corrigir os caminhos de configuração nativa** identificados como errados ou incompletos:
   `.github/copilot/settings.json` no lugar de `.github/settings.json`, e as camadas
   `.codex/hooks.json` que o registro hoje ignora.

A adoção está concluída quando o gate de projeção estiver verde, a matriz declarar os vinte pares
`(provedor, evento)` com estado explícito, e nenhum par estiver sem declaração.

## Monitoramento e Validação

- **Sinal primário:** `hook_contract_projection_test.go` verde no CI. É o único indicador que prova a
  premissa central da decisão — que projeções derivadas não divergem.
- **Sinal secundário:** contagem de pares `(provedor, evento)` sem declaração igual a zero, exposta
  por `ai-spec hooks capabilities`.
- **Telemetria:** entradas `hook.decision` e `hook.duration_ms` por evento em `.agents/telemetry.log`,
  no formato `<ts> chave=valor` já estabelecido pela ADR-006 do repositório, opt-in por
  `GOVERNANCE_TELEMETRY=1`. Nenhuma infraestrutura de métricas nova é introduzida.
- **Critério de sucesso:** adicionar um evento canônico novo exige tocar exatamente um pacote.
- **Critério para revisar ou reverter:** se manter as duas projeções exigir mais de um caso especial
  por modelo, a premissa de que projeção é mais barata que unificação terá falhado, e a alternativa
  A3 deve ser reavaliada com o custo real em mãos.

## Impacto em Documentação e Operação

- `docs/hooks-canonicos.md` — documento novo com a semântica de cada evento e a matriz por provedor.
- `docs/runtime-claude-capabilities.md` — atualizar a descrição dos pontos.
- `docs/degradation-matrix.md` — acrescentar os pares `SupportUnsupported` e `SupportAdapter`.
- `AGENTS.md` e `CLAUDE.md` — a afirmação de que `validate-governance.sh` "bloqueia edição de
  `AGENTS.md` e de `SKILL.md`" precisa ser corrigida: a documentação oficial das quatro CLIs confirma
  que `AfterTool` **não bloqueia em nenhuma delas**, e o script roda depois da edição. Ele informa,
  não impede.
- `docs/troubleshooting.md` — diagnóstico de divergência entre contrato e adapter.
- Onboarding: a distinção entre evento canônico, chave nativa e artefato instalado passa a ser
  conceito de primeira classe.

## Revisão Futura

Revisar quando ocorrer qualquer um destes eventos:

- uma quinta CLI entrar no harness;
- qualquer provedor introduzir um evento cuja semântica não caiba nos cinco canônicos — o caso mais
  provável é `PermissionRequest`, que existe em Claude, Codex e Copilot, **é bloqueante**, e hoje o
  repositório não usa em nenhuma delas;
- o OpenCode passar a oferecer evento de fim de sessão ou bloqueio em `BeforeComplete`, o que
  eliminaria a única limitação estrutural declarada;
- o número de pares `(provedor, evento)` ultrapassar o que uma tabela em Markdown sustenta com
  clareza.
