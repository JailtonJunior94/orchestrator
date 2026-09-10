# Transcript da Modelagem de Dominio

## Contexto Inicial

- **Pedido**: modelar o dominio do Ciclo de Aprovacao (loop review -> bugfix -> review) e do Catalogo de Agentes (4 CLIs) do `ai-spec-harness`, para alimentar a especificacao tecnica do PRD `.specs/prd-harness-quatro-clis-loop-aprovacao/prd.md` (spec-version 2, 51 RFs).
- **Natureza do pedido** (derivada da sessao, nao re-perguntada): novo fluxo (Ciclo de Aprovacao) + revisao de fluxo existente (catalogo de agentes com remocao do Gemini e entrada do OpenCode).
- **Estado do material de apoio**: regras claras. Existem PRD v2 fechado, 13 fatos verificados contra fonte primaria e 11 decisoes de escopo tomadas pelo solicitante.
- **Escopo de codebase**: path local `/Users/jailtonjunior/Git/orchestrator` (Go). Nao e greenfield.
- **Objetivo dominante**: preparar implementacao com zero falso positivo de aprovacao e paridade igualitaria entre Claude Code, Codex, GitHub Copilot CLI e OpenCode.
- **Restricao operacional declarada**: zero regressao nos agentes remanescentes; release MAJOR 2.0.0.

## Confronto com Codebase

Evidencias `path:linha` sustentando o confronto (amostra canonica; lista integral acima):
- `internal/taskloop/bugfix.go:83` — loop de remediacao ja existente
- `internal/runtime/runner_autoreview.go:233` — veredito derivado de texto sintetico
- `internal/runtime/runner.go:216-228` — auto-review one-shot
- `.agents/lib/check-invocation-depth.sh:40` — teto de profundidade default 2
- `internal/metrics/metrics.go:214` — `ToolBudgetsLarge` com entrada unica
- `internal/skills/skills.go:13` — ordem canonica de tools

Riscos de compatibilidade registrados na secao `## Materiais e Evidencias` de `domain-model.md`.

Confronto executado sobre o path local `/Users/jailtonjunior/Git/orchestrator` por quatro agentes
especializados (dois de mapeamento de codigo, dois de verificacao empirica contra binarios instalados).

**Classificacao dos achados:**

| Achado | Classe | Evidencia |
|---|---|---|
| Ja existe um loop review->bugfix->review implementado | `confirmado` | `internal/taskloop/bugfix.go:83` (`BugfixLoop.Run`), teto em `:66` (`DefaultMaxBugfixIterations = 3`), condicao de saida em `:131` |
| Enum de veredito de 4 valores ja existe | `confirmado` | `internal/taskloop/reviewer.go` (`ReviewVerdict`); mapeamento severidade->veredito em `.agents/skills/review/SKILL.md:59-70` |
| Auto-review do runtime e one-shot e nao conhece o loop | `confirmado` | `internal/runtime/runner.go:216-228` chama `runAutoReview` uma vez; `internal/runtime/runner_autoreview.go:145` |
| Veredito de producao e derivado de texto SINTETICO, nao da saida real do revisor | `confirmado` | `internal/runtime/runner_autoreview.go:233` (`buildReviewOutputFromSummary`) alimenta `parseReviewStatus` (`:60`) |
| `review.md` da skill e sobrescrito por um apontador de 3 linhas do Go | `confirmado` | `runner_autoreview.go:189` escreve apos o child rodar (`:180`); `buildReviewPointer` em `:244` |
| `AI_REVIEW_PRIOR_SHA` e contrato prompt-only orfao | `confirmado` | especificado em `.agents/skills/review/SKILL.md:16`; nenhum exportador em Go, hook ou script |
| Teto de rodadas nao tem flag CLI | `confirmado` | `internal/taskloop/taskloop.go:37` (`Options.MaxBugfixIterations`) nunca preenchido por `cmd/ai_spec_harness/task_loop.go` |
| Depth guard quebra a primeira remediacao | `confirmado` | `.agents/lib/check-invocation-depth.sh:40` (`AI_INVOCATION_MAX` default 2) + `bugfix/SKILL.md:20` (falha dura); `execute-task` ja consome 1 |
| Precedente de reset de profundidade por unidade de trabalho | `confirmado` | `.agents/skills/execute-all-tasks/SKILL.md:92` e `:159` (`export AI_INVOCATION_DEPTH=0` por subagente) |
| `internal/invocation/invocation.go` e codigo morto para o loop | `confirmado` | `Guard.CheckDepth` (`:24`) e `Guard.IncrementDepth` (`:35`) sem caller em `internal/runtime` ou `internal/taskloop` |
| Gate de evidencia ja rejeita veredito fora do conjunto canonico | `confirmado` | `.agents/scripts/validate-task-evidence.sh:198-204` (hoje aceita `APPROVED` e `APPROVED_WITH_REMARKS`) |
| Nao existe `validateReview` no validador Go | `confirmado` | `internal/evidence/evidence.go:32` tem `validateTask`, `validateBugfix`, `validateRefactor` |
| `ToolBudgetsLarge` tem uma unica entrada (gemini) | `confirmado` | `internal/metrics/metrics.go:214` |
| `inherit_common` tem um unico item (gemini) | `confirmado` | `.agents/normalization-rules.yaml:20-21` e `internal/runtime/events/normalization-rules.yaml:20-21` |
| Duas ordens canonicas de tools coexistem | `confirmado` | `internal/skills/skills.go:13` (`[Claude, Gemini, Codex, Copilot]`) vs `cmd/ai_spec_harness/verify.go:154-155` (`[Claude, Codex, Copilot, Gemini]`) |
| Gate de CI exige a string "gemini" no schema do CLI | `confirmado` | `cmd/ai_spec_harness/cli_contract_test.go:181-184` |
| Chave de evento `stop` do Copilot e invalida (correto: `agentStop`) | `confirmado` | `internal/install/install.go:1275-1302` (`defaultCopilotHooks`) vs lista oficial de eventos; `install_test.go:446` asserta o inverso |
| Uninstall nao remove 6 arquivos que o proprio install cria | `confirmado` | `internal/uninstall/uninstall.go:124-137` cobre parcialmente o que `install.go:816-878` escreve |
| Manifesto nao rastreia arquivos instalados individualmente | `confirmado` | `internal/manifest/manifest.go:14-26`; `uninstall.Execute` usa lista hardcoded e ignora o manifesto |
| Hooks de PROJETO do Copilot disparam, condicionados a pasta trusted | `confirmado` | teste de disparo real nesta sessao: `PROOF.txt` com `FIRED_LOWER` e `FIRED_PASCAL` em pasta sob `trustedFolders` |
| Hooks do Codex exigem `trusted_hash` concedido por TUI interativa | `confirmado` | enum `HookSource` inclui `project`; trust persistido em `~/.codex/config.toml` `[hooks.state.*]`; nao ha subcomando `codex hooks` |
| Trust do Codex e verificavel sem conceder | `confirmado` | RPC `hooks/list` do `codex app-server`, read-only, retorna `HookTrustStatus` |
| `tool.execute.before` do OpenCode BLOQUEIA por excecao | `confirmado` | prova por execucao: tool vira `status:"error"`, `AFTER` nunca loga, modelo le a mensagem; 7 call sites no bundle |
| `permission.ask` do OpenCode e codigo morto na 1.18.30 | `refutado` (como mecanismo) | zero call sites `trigger("permission.ask")` no binario; hook nunca disparou em teste com ask real |
| `.agents/skills/` de PROJETO e auto-carregado pelo OpenCode | `confirmado` | `opencode debug skill` lista a skill de projeto sem `skills.paths`; `skills.paths` e aditivo |
| `AGENTS.md` e auto-carregado pelo OpenCode com upward-walk | `confirmado` | prova por execucao com token unico; sem chave `instructions` |

**Suspeitos e nao resolvidos nesta fase:** comportamento do bloqueio sob `opencode acp` (todos os testes foram
sob `opencode run`) e efeito dos interruptores `--pure` / `OPENCODE_PURE` / `OPENCODE_DISABLE_DEFAULT_PLUGINS`
sobre plugins de projeto. Verificacao empirica em andamento.

## Rodada 1 - Linguagem e Fronteiras

**Perguntas em multipla escolha (4) e respostas do solicitante:**

1. *Termo canonico do fluxo principal* -> **Ciclo de Aprovacao**.
   Nomeia o fluxo pela invariante que ele protege ("so fecha em APPROVED"), nao pela mecanica.
   Rejeitadas: `Ciclo de Remediacao` (mente no caso feliz de zero achados), `Loop de Revisao`
   (vocabulario de mecanismo, nao de negocio), `Veredito de Tarefa` (perde a nocao de processo iterativo).

2. *Quantidade de bounded contexts* -> **Dois contextos**: `Catalogo de Agentes` e `Ciclo de Aprovacao`.
   Justificativa aceita: ritmos de mudanca distintos — o catalogo muda quando o ecossistema de CLIs muda;
   o ciclo muda quando a regra de qualidade muda. Evidencia permanece artefato de saida do Ciclo, sem
   virar contexto proprio. Rejeitadas: contexto unico (reproduz o acoplamento que tornou a remocao do
   Gemini cara), tres contextos (um contexto so de I/O nao carrega decisao de negocio propria).

3. *Restricao dominante que arbitra trade-offs* -> **Correcao fail-closed**.
   Diante de duvida, bloquear em vez de aprovar. Rejeitadas: economia de tokens (compra economia com
   risco de aprovacao mal verificada), compatibilidade com legado (impediria endurecer justamente os
   gates frouxos que a entrega precisa mudar).

4. *Ownership do conceito Agente* -> **Agregado `Catalogo de Agentes`, com Agente como Value Object imutavel**
   (identidade, spec ACP, mecanismo de enforcement, janela de contexto). Adicionar ou remover agente
   passa a ser uma mudanca no catalogo. Rejeitadas: cada agente como entidade autonoma (mantem a
   enumeracao espalhada; nada garante cobertura uniforme dos pontos canonicos), agente como mera
   configuracao (perde validacao do conjunto fechado, que e a invariante sustentadora da paridade).

## Rodada 2 - Workflow e Comportamento

**Perguntas em multipla escolha (4) e respostas do solicitante:**

1. *Eventos de dominio do Ciclo* -> **Rodada + encerramento**:
   `RodadaIniciada`, `RodadaConcluida` (veredito + contagem de achados por severidade), `CicloEncerrado`
   (motivo canonico). Alinha com RF-35/RF-36 e com o `events.jsonl` ja emitido pelo harness.
   Rejeitadas: so `CicloEncerrado` (perde a linha do tempo intra-ciclo, que e o insumo de diagnostico de
   nao-convergencia), granularidade fina por criterio/achado (custo de persistencia desproporcional ao
   ganho, ja coberto pelo mapa 1:1 no relatorio).

2. *Estado ilegal mais perigoso* -> **Ciclo encerrado sem veredito**.
   Deve ser impossivel construir um Ciclo em estado "encerrado com sucesso" sem `Veredito = APPROVED` e
   mapa 1:1 completo anexados. O falso positivo deixa de ser bug de fluxo e passa a ser estado
   inconstruivel. Rejeitadas: rodada alem do teto (limite operacional, ja tratado como `blocked`),
   evidencia sem comando executado (regra de validacao de conteudo; sozinha nao impede fechar sem veredito).

3. *Tratamento da nao-convergencia* -> **Transicao terminal explicita** com motivo canonico, nao erro tecnico.
   Distingue "resultado legitimo de negocio que o humano decide" de "falha de infraestrutura que merece retry".
   Rejeitadas: erro de dominio lancado (chamador tende a aplicar retry, comportamento errado aqui),
   campo de texto livre (abre espaco para motivo inventado; RF-35 exige conjunto canonico fechado).

4. *Gatilho e comando dominante* -> **`SubmeterParaAprovacao`**, comando unico; as rodadas sao decisao
   interna do agregado, que e dono da regra de parada. Impede um orquestrador externo "esquecer" de
   iterar ou parar cedo. Rejeitadas: `ExecutarRodada` controlado por fora (devolve a regra de parada
   para fora do dominio — e exatamente como o fluxo falha hoje), dois comandos iniciar/continuar
   (suporta retomada, mas exige persistir estado intermediario; custo nao justificado nesta entrega).

## Rodada 3 - Regras e Invariantes

**Perguntas em multipla escolha (4) e respostas do solicitante:**

1. *Modelagem do Veredito* -> **Value Object com construtor validante**, campo interno nao exportado,
   zero-value invalido. Reusa o padrao ja existente em `internal/runtime/specs/driver.go:14` (`DriverID`).
   Rejeitadas: enum com `iota` (nao valida na fronteira, e o veredito chega como texto do agente),
   string validada no uso (validacao difusa; contradiz a postura fail-closed).

2. *Estrategia de erro* -> **Resultado tipado + sentinelas**. Encerramento do ciclo, inclusive
   nao-convergente, e RESULTADO; erro fica para falha real de infraestrutura, com sentinelas
   comparaveis por `errors.Is` (regra 5.10 de `go-implementation`). Rejeitadas: tudo como erro tipado
   (o codigo de retry existente passaria a reprocessar reprovacoes legitimas), codigos de status sem
   erro (perde integracao com o tratamento de erro idiomatico ja usado no runner).

3. *Base da fingerprint* -> **conjunto normalizado e ordenado de `{severidade, arquivo, identificador
   da regra}`**, com SHA-256, ignorando numero de linha e texto livre. Justificativa reforcada pelo
   confronto: `id` do bug e atribuido pelo agente e nao e estavel entre rodadas; `line` desloca a cada
   patch. Rejeitadas: hash do texto integral (falso negativo garantido — qualquer variacao de redacao
   muda o hash), contagem por severidade (falso positivo — conjuntos diferentes com mesma contagem).

4. *Sede da politica de parada* -> **Value Object `PoliticaDeAprovacao` imutavel e injetado**,
   construido a partir da hierarquia de config existente (flags > workspace > global > defaults).
   O agregado continua dono de APLICAR a politica. Rejeitadas: constantes no agregado (RF-31 exige
   teto configuravel), politica como interface Strategy (indirecao sem fronteira consumidora real;
   `go-implementation` manda preferir tipo concreto).

## Rodada 4 - Tipos e Integracoes

**Perguntas em multipla escolha (4) e respostas do solicitante:**

1. *Onde mora o agregado* -> **pacote de dominio proprio** (ex.: `internal/approval`), com o agregado e
   os VOs (`Veredito`, `PoliticaDeAprovacao`, `Fingerprint`, `MotivoDeParada`) e **zero dependencia** de
   ACP, CLI ou filesystem. `internal/taskloop` e `internal/runtime` passam a consumi-lo. Rejeitadas:
   dentro de `internal/taskloop` (acopla dominio a orquestracao e inverte a direcao de dependencia para
   `internal/runtime`), dentro de `internal/runtime` (amarra a regra ao protocolo ACP e exclui o caminho legacy).

2. *Fronteira externa mais sensivel* -> **o texto do revisor**. A traducao "saida em linguagem natural
   do agente -> Veredito tipado" e explicita, validada e fail-closed: sem veredito canonico declarado no
   texto, o resultado e `BLOCKED`, nunca aprovacao por ausencia de marcadores. Decisao sustentada pelo
   achado `confirmado` de que hoje o parser le texto sintetico. Rejeitadas: filesystem de evidencia
   (consequencia, nao entrada de decisao), git/diff (erro detectavel, ao contrario do veredito falso).

3. *Consistencia entre rodadas* -> **forte em memoria durante a execucao** (rodada corrente, conjunto de
   fingerprints vistas, SHA anterior) + **evidencia append-only imutavel e numerada por rodada**, nunca
   sobrescrita — corrige por desenho o bug confirmado de `review.md` sobrescrito. Limite aceito
   explicitamente: crash perde o ciclo em andamento e ele recomeca. Rejeitadas: estado persistido por
   rodada (exige versionar formato, tratar corrupcao e definir semantica de retomada — nenhum requisito
   pede), sem estado (fingerprint e contagem de rodadas exigem memoria entre rodadas).

4. *Postura de custo e operacao* -> **abortar cedo e barato**: checagens baratas antes das caras — diff
   vazio e fingerprint repetida abortam sem gastar sessao de revisao; rodada N>1 revisa apenas o delta.
   O pior caso so ocorre havendo progresso real por rodada. Rejeitadas: sempre rodar o ciclo completo
   (queima orcamento em cenario irrecuperavel), orcamento de tokens por ciclo (segundo motivo de parada
   nao-deterministico, enfraquece reprodutibilidade).

## Decisoes Registradas

Decisoes efetivas tomadas pelo solicitante nas quatro rodadas, consolidadas em `domain-model.md`:

1. Termo canonico do fluxo: **Ciclo de Aprovacao**.
2. Dois bounded contexts: **Catalogo de Agentes** e **Ciclo de Aprovacao**; evidencia e saida do Ciclo.
3. Restricao dominante: **correcao fail-closed**.
4. Ownership de Agente: **agregado Catalogo**, com Agente como **Value Object imutavel**.
5. Eventos: **RodadaIniciada, RodadaConcluida, CicloEncerrado**.
6. Estado ilegal alvo: **Ciclo encerrado com sucesso sem veredito e sem mapa 1:1**.
7. Nao-convergencia: **transicao terminal explicita com motivo canonico**, nao erro tecnico.
8. Comando dominante: **SubmeterParaAprovacao**; o agregado e dono da regra de parada.
9. Veredito: **Value Object com construtor validante**, zero-value invalido.
10. Erros: **resultado tipado + sentinelas** para falha de infraestrutura.
11. Fingerprint: **{severidade, arquivo, regra} normalizados**, sem linha nem id.
12. Politica de parada: **Value Object PoliticaDeAprovacao injetado**.
13. Sede do agregado: **pacote de dominio proprio**.
14. Fronteira mais sensivel: **o texto do revisor**, com camada anticorrupcao fail-closed.
15. Consistencia: **forte em memoria**; **evidencia append-only imutavel** por rodada.
16. Postura de custo: **abortar cedo e barato**.

Verificacoes empiricas concluidas apos a Rodada 4, sem impacto no modelo (afetam mecanismo, nao regra):
o bloqueio por excecao do agente OpenCode **foi provado sob o runtime ACP**, e **tres interruptores de
ambiente desligam o gate por completo**. Ambos dizem respeito a `Mecanismo de Enforcement` e
`PreCondicaoDeEnforcement`, conceitos que o modelo ja declara.
