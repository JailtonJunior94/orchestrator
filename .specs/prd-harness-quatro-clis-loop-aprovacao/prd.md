# Documento de Requisitos do Produto (PRD)

<!-- spec-version: 4 -->

**Slug:** `harness-quatro-clis-loop-aprovacao`
**Data:** 2026-09-10
**Status:** Fechado — zero questões de escopo em aberto; zero suposições não verificadas

## Visão Geral

O `ai-spec-harness` declara suporte a quatro CLIs de IA — Claude Code, Codex, GitHub Copilot CLI e
Gemini CLI. Três problemas comprometem a promessa de "SDD robusto, eficiente e econômico":

1. **O conjunto de agentes não reflete mais o ecossistema oficial.** O Gemini CLI carrega uma política
   de detecção *opt-in* exclusiva, um invoker legado com aviso de depreciação, assets dedicados
   (`.gemini/`, `GEMINI.md`) e uma trilha de exceções em ~40 arquivos Go, testes e snapshots. É o
   único dos quatro tratado como cidadão de segunda classe pela própria detecção do harness. Em
   paralelo, o **OpenCode** consolidou-se como agente de terminal oficial com servidor ACP
   (`opencode acp`), carga de skills externas, sistema de plugins e um modelo de permissões
   declarativo — e está ausente.

2. **O ciclo de revisão fecha tarefa sem prova real de aprovação.** Hoje há **uma única rodada**:
   `execute-task` Etapa 4 permite `review → bugfix → review`, o `agent-governance` fixa que "a cadeia
   review → bugfix → review é máxima", e o orquestrador chama a auto-revisão uma vez, sem iteração.
   Pior: `APPROVED_WITH_REMARKS` sem tag crítica fecha a tarefa como `done`, e um critério de aceite
   classificado como "não verificável pelo diff" é apenas **registrado como risco** — nunca bloqueia.
   O resultado é um fechamento que parece aprovado sem ter sido verificado.

3. **A paridade declarada não é integralmente comprovada.** A verificação contra fonte primária feita
   para este PRD (§Fatos Verificados) mostrou que o enforcement por hooks tem pré-condições reais e
   não uniformes entre os CLIs — hooks do Codex exigem *trust* persistido; hooks de **projeto** do
   Copilot não são comprováveis pela via estática; o OpenCode possui interruptores de ambiente capazes
   de desligar plugins e skills. Uma matriz de paridade que ignore isso afirma governança que pode não
   estar rodando.

Esta funcionalidade resolve os três em uma entrega única: consolida os **quatro CLIs oficiais**
(Claude Code, Codex, GitHub Copilot CLI, OpenCode) com **comportamento igualitário e comprovado**,
remove o Gemini, e substitui a rodada única por um **loop de remediação determinístico que só encerra
em `APPROVED` com evidência 1:1 por critério de aceite**.

Público-alvo: times que usam o harness para conduzir desenvolvimento orientado a especificação (SDD)
e que precisam confiar que "tarefa concluída" significa "requisito verificado".

## Objetivos

- **O-01 — Paridade igualitária comprovada entre 4 CLIs.** Os quatro agentes compartilham gates,
  enforcement, evidências e vereditos. Sucesso: a matriz de paridade não contém célula `BestEffort`
  nem `N/A` nos pontos canônicos, e **cada célula é sustentada por teste de disparo real**, não por
  leitura de documentação.
- **O-02 — Zero falso positivo de aprovação.** Nenhuma tarefa fecha como `done` sem veredito
  `APPROVED` e sem mapa 1:1 critério de aceite → evidência verificável. Sucesso: 100% dos relatórios
  de execução contêm o mapa completo; qualquer lacuna produz `blocked`.
- **O-03 — Convergência auditável do loop.** Sucesso: todo encerramento de loop registra motivo
  canônico (`approved`, `max_rounds`, `no_convergence`, `empty_diff`, `blocked_input`), com histórico
  por rodada persistido.
- **O-04 — Redução de superfície de manutenção.** Sucesso: zero ocorrências de `gemini` no código de
  produção; `--tools=all` resolve exatamente 4 agentes; nenhuma política de detecção de exceção
  permanece.
- **O-05 — Economia sem perda de rigor.** Sucesso: a revisão da rodada N>1 opera apenas sobre o
  **delta** da remediação; loops improdutivos abortam antes de consumir o teto de rodadas.
- **O-06 — Zero regressão nos agentes remanescentes.** Sucesso: nenhum valor default muda para Claude,
  Codex e Copilot; fluxos existentes produzem resultado idêntico.
- **O-07 — Nenhum gate silenciosamente inativo.** Sucesso: toda pré-condição de enforcement
  (trust de hook, escopo de instalação, interruptor de ambiente) é verificada e reportada; a ausência
  de pré-condição produz estado explícito, nunca sucesso aparente.

## Histórias de Usuário

**US-01 (história única desta entrega — persona primária: engenheiro que opera o harness)**

> **Como** engenheiro que conduz desenvolvimento orientado a especificação com múltiplos CLIs de IA,
> **quero** que o `ai-spec-harness` suporte oficialmente e de forma igualitária Claude Code, Codex,
> GitHub Copilot CLI e OpenCode — com o Gemini removido — e que o ciclo de revisão e correção itere
> automaticamente até um veredito `APPROVED` comprovado por evidência 1:1 de cada regra de negócio e
> critério de aceite,
> **para que** eu possa trocar de agente sem perder gates, observabilidade ou rigor, e confiar que
> "tarefa concluída" significa "requisito verificado" — nunca uma aprovação inventada.

Personas e fluxos secundários cobertos pela mesma história:

- **Mantenedor do harness:** precisa que a remoção do Gemini seja completa e que o OpenCode não
  duplique skills, para não criar drift entre cópias.
- **Revisor humano / tech lead:** precisa que o esgotamento do loop devolva `blocked` com histórico
  legível, nunca um `done` silencioso.
- **Operador de CI:** precisa que os gates rodem fora da sessão interativa, sem depender de o modelo
  "lembrar" de iterar.

Casos de borda cobertos: projeto sem nenhum CLI instalado; projeto legado com `.gemini/` residual;
hooks do Codex sem *trust* concedido; hooks de projeto do Copilot não suportados pelo CLI; OpenCode
executando com plugins ou skills externas desligados por variável de ambiente; loop cuja remediação
não produz diff; loop cujos achados se repetem idênticos; critério de aceite de natureza operacional
que o diff não comprova.

## Fatos Verificados

Verificação contra **fonte primária** em 2026-09-10: binários instalados na máquina de desenvolvimento,
documentação embarcada nos próprios binários, tipos publicados dos pacotes, **testes de disparo real** e
leitura do código do repositório. Estes fatos substituem suposições. A especificação técnica deve
reconfirmá-los e registrar a data.

### Bloco 1 — Agentes e seus mecanismos

| # | Fato | Como foi provado | Impacto |
|---|---|---|---|
| V-01 | OpenCode expõe ACP por **subcomando** `opencode acp`, não por flag; versão `1.18.30` | `opencode acp --help`; `npm view opencode-ai@latest` | A abstração de spec precisa acomodar subcomando (RF-10) |
| V-02 | `opencode acp` **não** aceita `--model` nem flag de permissão | `opencode acp --help` | Modelo e política vêm de `opencode.json` (RF-16, RF-17) |
| V-03 | Plugins de projeto são auto-descobertos em `.opencode/plugin/` **ou** `.opencode/plugins/` | doc embarcada no binário | Adota-se a forma singular; ambas funcionam |
| V-04 | **`tool.execute.before` BLOQUEIA por exceção** — a tool nunca executa, vira `status:"failed"`, a sessão sobrevive e o modelo lê a mensagem | **Teste de disparo real** sob `opencode run` **e** sob `opencode acp` (cliente ACP mínimo); log do plugin sem `AFTER tool=bash`; `session/update` com `status:"failed"` e o texto da exceção | É o mecanismo de enforcement (RF-19). **Corrige a premissa anterior de que o contrato `Promise<void>` impediria o bloqueio** |
| V-05 | O bloqueio é **robusto**: sobrevive a 3 exceções consecutivas na mesma sessão e **intercepta tool executada dentro de subagente `task`** | Teste real sob ACP; `stopReason:"end_turn"` após 4 bloqueios | Cobre a superfície inteira, inclusive delegação (RF-19) |
| V-06 | **`permission.ask` é código morto na 1.18.30** — zero call sites no binário | Enumeração de todos os `trigger("...")` do bundle; a string só aparece na doc embarcada; hook nunca disparou mesmo com *ask* real | **Proibido** basear o gate nele: falharia em silêncio (RF-19) |
| V-07 | O bloco `permission` com pattern `*` **remove a tool do tool-set** enviado ao modelo; padrões finos negam em runtime | Teste real: modelo relatou não ter a ferramenta disponível | Defesa em profundidade e economia de tokens (RF-20) |
| V-08 | **Três interruptores desligam o gate por completo**: flag `--pure`, `OPENCODE_PURE=1` e `OPENCODE_DISABLE_PROJECT_CONFIG=1`. `OPENCODE_DISABLE_DEFAULT_PLUGINS=1` **não** afeta plugin de projeto | Teste real por interruptor; plugin não carregou e o comando executou de verdade | Exige sanitização do ambiente do processo filho **e** handshake ativo (RF-21) |
| V-09 | **`.agents/skills/` de PROJETO é auto-carregado** pelo OpenCode, sem `skills.paths`; e `skills.paths` é **aditivo** | `opencode debug skill` listando a skill de projeto com config vazia; 16 skills ao adicionar o path (duplicação) | Escrever `skills.paths` seria **redundante e nocivo** (RF-13). **Corrige a premissa anterior baseada na tabela incompleta da doc embarcada** |
| V-10 | **`AGENTS.md` é auto-carregado** com upward-walk até a raiz do worktree; `instructions` é desnecessária | Teste real com token único recuperado pelo modelo | O instalador não escreve `instructions` (RF-13) |
| V-11 | `OPENCODE_DISABLE_EXTERNAL_SKILLS=1` mata skills **globais e de projeto** | `opencode debug skill` retornando apenas a built-in | Mais um vetor a cobrir pelo handshake (RF-21) |
| V-12 | **Hooks de PROJETO do Copilot disparam** (`.github/hooks/*.json`), em ambas as grafias de evento — **condicionado a a pasta estar em `trustedFolders`** | **Teste de disparo real** nesta sessão: `PROOF.txt` gravado em pasta trusted; a mesma configuração falhou em pasta untrusted | Resolve o gap anterior; o pré-requisito real é *trust de pasta* (RF-25) |
| V-13 | Hooks do Codex de projeto (`.codex/hooks.json`) são suportados, mas exigem **`trusted_hash`** concedido pela **TUI interativa** (`/hooks`); não há subcomando não-interativo | Enum `HookSource` inclui `project`; strings da TUI; ausência de `codex hooks` | Sem trust, o gate está **inerte** — e hoje não há `[hooks.state]` na config do usuário (RF-24) |
| V-14 | O trust do Codex é **verificável sem conceder**, via RPC `hooks/list` do `codex app-server` (read-only), que retorna `HookTrustStatus` | Schema gerado por `codex app-server generate-json-schema` | Viabiliza RF-24 sem usar a flag de bypass |
| V-15 | Copilot CLI expõe `--acp`; Codex e Claude mantêm seus caminhos já integrados | `copilot --help` | Nenhuma mudança nos três já integrados |
| V-16 | O Codex expõe subcomando nativo `codex review` | `codex --help` | Decisão registrada: **não** será usado (RF-40) |

### Bloco 2 — Estado do código do harness

| # | Fato | Evidência | Impacto |
|---|---|---|---|
| V-17 | **Já existe um loop** review→bugfix→review, com teto de 3 rodadas e enum de veredito de 4 valores | `internal/taskloop/bugfix.go:83`, teto em `:66`, saída em `:131`; `internal/taskloop/reviewer.go` | O Ciclo **promove** esse loop em vez de criar outro (RF-31) |
| V-18 | O auto-review do runtime é **one-shot** e não conhece o loop | `internal/runtime/runner.go:216-228`; `internal/runtime/runner_autoreview.go:145` | As duas camadas passam a consumir o mesmo agregado (RF-31) |
| V-19 | **O veredito de produção é derivado de texto SINTÉTICO**, não da saída real do revisor | `runner_autoreview.go:233` (`buildReviewOutputFromSummary`) alimenta `parseReviewStatus` (`:60`) | Defeito confirmado; é a raiz do falso positivo (RF-57) |
| V-20 | O Go **sobrescreve** o `review.md` produzido pela skill com um apontador de 3 linhas | `runner_autoreview.go:189` escreve após o child rodar (`:180`); `buildReviewPointer` em `:244` | Defeito confirmado; destruiria a evidência por rodada (RF-58) |
| V-21 | A chave de evento `stop` usada para o Copilot **não existe**; a correta é `agentStop` — e o teste asserta o inverso | `internal/install/install.go:1275-1302` vs lista oficial de eventos; `install_test.go:446` | Defeito confirmado; o hook de encerramento nunca rodou no Copilot (RF-59) |
| V-22 | A desinstalação **não remove 6 arquivos** que a própria instalação cria | `internal/uninstall/uninstall.go:124-137` vs `internal/install/install.go:816-878` | Defeito confirmado (RF-60) |
| V-23 | `AI_REVIEW_PRIOR_SHA` é contrato **órfão**: especificado na skill, nunca exportado por código, hook ou script | `.agents/skills/review/SKILL.md:16`; nenhum exportador no repositório | A revisão incremental nunca funcionou (RF-37) |
| V-24 | O teto de rodadas **não tem flag CLI**: o campo existe e nunca é preenchido | `internal/taskloop/taskloop.go:37` sem escritor em `cmd/ai_spec_harness/task_loop.go` | RF-35 é a primeira exposição desse parâmetro |
| V-25 | **O guarda de profundidade quebra a primeira remediação**: limite default 2, `execute-task` já consome 1, e a correção falha de forma dura | `.agents/lib/check-invocation-depth.sh:40`; `.agents/skills/bugfix/SKILL.md:20` | Sem RF-38 nenhum ciclo passa da rodada 1 |
| V-26 | Já existe **precedente de reset de profundidade por unidade de trabalho** | `.agents/skills/execute-all-tasks/SKILL.md:92` e `:159` | RF-38 replica um padrão já validado, não inventa um |
| V-27 | Não existe validador Go de artefato de revisão | `internal/evidence/evidence.go:32` tem apenas task, bugfix e refactor | Assimetria Go↔shell a corrigir (RF-52) |
| V-28 | `ToolBudgetsLarge` e `inherit_common` têm **entrada única** — ficam vazios ao remover o agente descontinuado | `internal/metrics/metrics.go:214`; `.agents/normalization-rules.yaml:20-21` e o gêmeo embarcado | Degradação silenciosa sem nenhum teste falhar (RF-06) |
| V-29 | **Duas ordens canônicas de agentes coexistem** no código | `internal/skills/skills.go:13` vs `cmd/ai_spec_harness/verify.go:154-155` | Saída não-determinística entre comandos (RF-07) |
| V-30 | Um gate de CI **exige** a presença da string do agente a ser removido | `cmd/ai_spec_harness/cli_contract_test.go:181-184` | Bloqueia a própria remoção; precisa ser o primeiro item alterado (RF-08) |
| V-31 | Existem **dois catálogos ACP espelhados**; remover só um faz cair em fallback silencioso | `cmd/ai_spec_harness/task_loop.go:28-32` e `internal/taskloop/taskloop.go:929-934` (fallback em `:938-940`) | Risco de regressão sem erro (RF-07) |
| V-32 | Nenhum teste do repositório prova **disparo** de hook; todos verificam apenas escrita de arquivo | `install_test.go:2410`, `:2439`; `e2e_install_upgrade_test.go` | A matriz de paridade nunca foi verificada de fato (RF-28) |
| V-33 | O manifesto **não rastreia arquivos instalados individualmente**, e a desinstalação o ignora | `internal/manifest/manifest.go:14-26`; `uninstall.Execute` usa lista fixa | RF-05 exige mudança no manifesto |

## Funcionalidades Core

### F-1 — Remoção total do Gemini (breaking)

O Gemini deixa de ser agente suportado. A remoção é física — código, assets, testes, snapshots,
documentação — não uma depreciação faseada. A invocação por nome produz erro de migração dirigido. A
desinstalação limpa apenas o que o harness instalou. **Importância:** elimina a única exceção de
política de detecção e devolve simetria ao conjunto de agentes, pré-condição para "igualitário" ter
significado testável. O confronto com o código mostrou que a remoção esvazia duas estruturas e ativa
um gate de CI que exige a própria string a remover — a entrega trata os dois.

### F-2 — OpenCode como agente oficial de 1ª classe via ACP

Detecção automática, runtime ACP nativo por subcomando com fallback de launcher, versão pinada e
paridade observacional completa. **Importância:** completa o conjunto oficial com o agente cujo modelo
de extensão melhor se encaixa no que o harness já assume — e cujo bloqueio já foi provado equivalente
ao dos demais, inclusive dentro de subagentes.

### F-3 — Governança OpenCode por reuso nativo comprovado

Ficou provado que o OpenCode carrega `AGENTS.md` e `.agents/skills/` de projeto **nativamente**, e que
declarar `skills.paths` **duplicaria** a árvore. O instalador, portanto, não copia skills, não cria
symlink e não escreve `skills.paths` nem `instructions`. Escreve apenas o bloco `permission` e deposita
o plugin de governança onde ele é auto-descoberto. **Importância:** é a menor pegada possível no
projeto do usuário, e cada arquivo não escrito é um ponto de drift que não existe.

### F-4 — Enforcement igualitário, comprovado e não-desligável

Os quatro agentes negam ação proibida e bloqueiam encerramento indevido, cada um pelo mecanismo do seu
CLI, todos delegando aos **mesmos scripts canônicos**. Cada célula da matriz de paridade exige teste de
disparo real — hoje nenhuma tem. Toda pré-condição de enforcement (pasta confiável, hash de hook
confiado, ausência de interruptor) é verificada, e a sessão é recusada quando o gate não pode operar.
**Importância:** transforma "suporta 4 CLIs" em "comporta-se igual nos 4 CLIs" — e impede que a matriz
afirme governança que não está rodando, que é o estado atual em dois dos quatro agentes.

### F-5 — Ciclo de Aprovação determinístico

O ciclo de remediação torna-se um agregado de domínio próprio, sem dependência de protocolo, CLI ou
filesystem, consumido tanto pelo runtime ACP quanto pelo orquestrador legado. Ele promove o loop já
existente, é dono da regra de parada, encerra apenas em `APPROVED` com prova, e cada rodada executa em
sessão nova. **Importância:** enforcement por prompt é best-effort; aprovação de tarefa é forte demais
para depender disso. E hoje o ciclo nem chega à rodada 2, porque o guarda de profundidade o interrompe.

### F-6 — Gate anti-falso-positivo de critérios de aceite

Cada critério exige uma linha de evidência verificável; critério sem evidência ou marcado como não
verificável **proíbe** a aprovação, e o gate é **inescapável**. **Importância:** é o ponto exato onde o
fluxo atual permite afirmar aprovação sem demonstrá-la.

### F-7 — Correção dos defeitos que sustentam o falso positivo

Quatro defeitos confirmados por evidência são corrigidos como parte da entrega, porque são causa-raiz
do problema, não vizinhos dele: o veredito derivado de texto sintético, a sobrescrita da evidência de
revisão, a chave de evento inválida que deixou o gate de encerramento do Copilot inerte, e a
desinstalação incompleta. **Importância:** sem eles, a entrega descreveria um comportamento que o
código continuaria não tendo.

## Requisitos Funcionais

### Bloco A — Remoção do Gemini

- **RF-01:** O conjunto canônico de agentes passa a ser exatamente `{claude, codex, copilot, opencode}`.
  Toda superfície que hoje enumera o Gemini deixa de aceitá-lo: seleção de ferramenta, catálogo de
  runtimes ACP, catálogo de drivers, perfis de execução, tabela de compatibilidade de modelos e tabela
  de orçamento de tokens por fluxo.
- **RF-02:** Todos os artefatos exclusivos do agente removido são apagados do repositório e dos assets
  embarcados: diretório de configuração, arquivo de governança raiz, rotina de instalação dedicada,
  geradores de adaptadores, settings default, invoker legado, extrator de métricas dedicado, testes de
  integração dedicados e fixture de paridade.
- **RF-03:** Invocar o agente removido por nome produz erro **tipado e explicativo**, citando o conjunto
  suportado e apontando o guia de migração — nunca o erro genérico de "valor inválido".
- **RF-04:** A documentação é reconciliada: a ADR do runtime removido é marcada como **Substituída**
  (preservada no histórico), e a tabela de governança por ferramenta, o README, a matriz de degradação,
  a matriz de confiabilidade e o guia de instalação refletem os 4 agentes. Changelog, ADRs, PRDs
  anteriores, auditorias e evidências de execução são **históricos** e não são reescritos.
- **RF-05:** A desinstalação remove **exclusivamente** os arquivos que o harness instalou. Como o
  manifesto hoje não rastreia arquivos individualmente (V-33), ele passa a fazê-lo, e a desinstalação
  passa a consumi-lo como fonte de verdade em vez de lista fixa. Arquivos do usuário são preservados; o
  diretório só é removido se ficar vazio; a operação é idempotente e não-fatal quando nada existir.
- **RF-06:** As estruturas que ficariam vazias com a remoção (V-28) recebem a entrada correspondente ao
  novo agente, **e** ganham teste que **falha** se qualquer uma delas ficar vazia no futuro. Sem esse
  teste, esvaziá-las degrada o comportamento sem nenhum sinal.
- **RF-07:** As duas ordens canônicas de agentes coexistentes (V-29) são unificadas em uma só, e os dois
  catálogos ACP espelhados (V-31) passam a ter fonte única ou verificação de sincronia que falha na
  divergência — o fallback silencioso atual é removido.
- **RF-08:** O gate de contrato de CI que exige a string do agente removido (V-30) é invertido para o
  novo agente antes de qualquer outra alteração, sob pena de bloquear a própria entrega.
- **RF-09:** Testes e snapshots dedicados são removidos ou regenerados; contagens fixas de agentes e
  asserções por índice de slice são revistas. `--tools=all` resolve exatamente os 4 agentes.

### Bloco B — OpenCode oficial

- **RF-10:** O OpenCode é declarado runtime ACP nativo com binário `opencode` e **subcomando** `acp`
  (V-01). A abstração de spec suporta a forma subcomando sem caso especial no runner. O launcher de
  fallback usa `npx --yes <pacote oficial>@<versão pinada> acp`.
- **RF-11:** As versões são **constantes pinadas**; `@latest` é proibido. Alteração exige registro de
  decisão em auditoria. A versão do SDK ACP permanece sincronizada com o `go.mod` pelo mecanismo vigente.
- **RF-12:** A detecção trata o OpenCode como 1ª classe, com os mesmos três sinais dos demais: binário
  no `PATH`, diretório de configuração do usuário, ou sinal de projeto. Qualquer sinal isolado basta.
  **Nenhuma política de detecção *opt-in* específica por agente permanece no harness.**
- **RF-13:** A instalação para OpenCode **não** copia skills, **não** cria symlink e **não** escreve
  `skills.paths` nem `instructions` — todos comprovadamente redundantes (V-09, V-10), e `skills.paths`
  seria ativamente nocivo por duplicar a árvore. Escreve apenas o bloco `permission` no `opencode.json`,
  preservando `$schema` e todos os campos preexistentes, de forma idempotente.
- **RF-14:** O plugin de governança é depositado no diretório de plugins de projeto, onde é
  auto-descoberto sem entrada de configuração (V-03).
- **RF-15:** Os validadores canônicos de evidência são instalados pelo mesmo caminho tool-neutro dos
  demais agentes: um projeto somente-OpenCode tem exatamente os mesmos gates que um somente-Claude.
- **RF-16:** O modo de acesso é expresso pelo bloco `permission` e/ou pela negociação do protocolo —
  **não** por flag, que o subcomando não oferece (V-02). É proibido presumir flag não confirmada.
- **RF-17:** O modelo é declarado pela chave `model` do `opencode.json`; a flag de modelo do harness
  mapeia para essa chave. A janela de contexto é **derivada do modelo resolvido** por tabela versionada;
  sem modelo resolvível, aplica-se fallback conservador **declarado e testado**.
- **RF-18:** O OpenCode alcança **paridade observacional completa**: eventos, renderização de tool-calls,
  relatório de execução, watchdog, telemetria opt-in e normalização com tabela de alias própria.

### Bloco C — Enforcement igualitário

- **RF-19:** No OpenCode, o enforcement primário é o hook de pré-ferramenta que **lança exceção**,
  delegando a decisão aos scripts canônicos — comprovadamente bloqueante sob ACP e dentro de subagentes
  (V-04, V-05). É **proibido** basear o gate no hook de permissão, que é código morto (V-06). A mensagem
  da exceção é redigida como instrução corretiva, porque o modelo a lê e reage a ela.
- **RF-20:** Como defesa em profundidade, o bloco `permission` nega declarativamente o que é
  categoricamente proibido, aproveitando que o padrão total remove a ferramenta do conjunto oferecido ao
  modelo (V-07). O valor "perguntar" é **proibido** em orquestração: seu comportamento diverge entre
  modos de execução.
- **RF-21:** O gate do OpenCode não pode ser desligável. Duas camadas obrigatórias: (a) o harness nunca
  passa a flag de modo puro e **sanitiza o ambiente do processo filho que ele mesmo cria**, removendo os
  interruptores conhecidos (V-08, V-11) — sem alterar o ambiente do usuário; (b) **handshake ativo**: o
  plugin sinaliza sua carga e a sessão é **abortada** se o sinal não chegar antes do primeiro prompt.
  A camada (b) valida o **efeito** e por isso cobre também vetores futuros desconhecidos.
- **RF-22:** Os quatro agentes cobrem os três pontos canônicos — pré-ferramenta, pós-ferramenta e
  encerramento de sessão — cada um pelo mecanismo nativo do seu CLI, todos apontando para os **mesmos
  scripts canônicos**. Nenhuma lógica de gate é reimplementada por agente.
- **RF-23:** Se o runtime necessário ao plugin não estiver disponível, a instalação e a verificação
  **falham de forma ruidosa**. Degradação silenciosa é proibida.
- **RF-24:** A verificação detecta se os hooks do Codex possuem **trust persistido** (V-13), usando o
  canal read-only já disponível (V-14). Sem trust, o estado é `drifted` com a instrução exata de
  concessão. O harness **nunca** concede o trust pelo usuário e **nunca** usa a flag de bypass.
- **RF-25:** A verificação detecta se o diretório do projeto está na lista de pastas confiáveis do
  Copilot (V-12). Fora dela, os hooks de projeto não disparam e o estado é reportado como não-ativo.
- **RF-26:** A regra de **não executar binários permanece intacta na detecção**. A verificação de
  pré-condições — que exige comunicação com um CLI — pertence ao comando de diagnóstico, que é invocado
  explicitamente pelo usuário. As duas responsabilidades ficam em superfícies separadas.
- **RF-27:** É criado um gate canônico de **encerramento** que bloqueia o fim da sessão quando existir
  tarefa ativa sem veredito `APPROVED` registrado, presente nos quatro agentes — cobrindo também o uso
  interativo, onde o orquestrador não está no caminho.
- **RF-28:** A matriz de paridade é verificada por um gate de build que **falha** quando um agente deixa
  de cobrir um ponto canônico, aponta para script divergente, **ou não possui teste de disparo real
  associado** — hoje nenhuma célula tem (V-32).
- **RF-29:** O espelhamento dos scripts canônicos permanece verificado pelo gate de sincronização,
  agora cobrindo o gate de encerramento e o plugin do OpenCode. As contagens fixas de espelhos nos
  scripts de verificação são atualizadas.

### Bloco D — Ciclo de Aprovação

- **RF-30:** O Ciclo de Aprovação é modelado como **agregado em pacote de domínio próprio**, sem
  dependência de protocolo ACP, CLI ou filesystem, e é a fonte de verdade da contagem de rodadas, do
  veredito corrente e do critério de parada. O runtime ACP e o orquestrador legado passam ambos a
  consumi-lo. As skills descrevem o mesmo comportamento; havendo divergência, prevalece o agregado.
- **RF-31:** O loop já existente (V-17) é **promovido** a esse agregado, não substituído por outro. Suas
  quatro lacunas são fechadas: remarks realimentam o ciclo, detecção de não-convergência, aborto por
  ausência de mudança e exportação do ponto de corte por rodada.
- **RF-32:** No modo orquestrado, o Ciclo é ativado pela mesma flag opt-in que hoje ativa a auto-revisão
  (default desligado, preservando não-regressão). O que muda é que, ativado, ele **itera**. No fluxo de
  skill, onde a revisão já é obrigatória, o Ciclo vale sempre.
- **RF-33:** O Ciclo encerra como aprovado **exclusivamente** com veredito `APPROVED` **e** mapa 1:1
  completo. `APPROVED_WITH_REMARKS` **não** encerra: seus achados realimentam a correção. A regra atual
  que fecha tarefa com remarks não-críticos é **removida**.
- **RF-34:** Cada rodada de revisão e de correção executa em **sessão/subagente novo**, recebendo apenas
  os achados e o delta, cumprindo a invariante de isolamento de contexto. Contexto acumulado através de
  rodadas é proibido: é o vetor direto de aprovação alucinada.
- **RF-35:** O teto de rodadas é **5** por default, configurável por flag e por arquivo de configuração,
  respeitando a precedência vigente. É a **primeira exposição** desse parâmetro (V-24), o que exige
  percorrer toda a cadeia de propagação e acrescentar a chave ao merge campo a campo — omitir o merge
  faz a configuração ser silenciosamente ignorada.
- **RF-36:** Esgotado o teto sem `APPROVED`, o resultado é **`blocked`** — nunca `done`. O retorno
  identifica o motivo canônico e devolve a decisão ao humano.
- **RF-37:** O Ciclo aborta antes de consumir o teto por dois mecanismos independentes: **fingerprint**
  repetida em rodadas consecutivas e **ausência de mudança** após a correção. A fingerprint é calculada
  sobre o conjunto ordenado de `{severidade, arquivo, identificador da regra}`, **ignorando número de
  linha e identificador atribuído pelo agente**, que comprovadamente não são estáveis entre rodadas.
- **RF-38:** O Ciclo é **iterativo no mesmo nível de invocação, não recursivo**. Cada rodada abre com a
  profundidade de invocação **resetada**, replicando o padrão já validado no orquestrador de PRD (V-26).
  Sem isso nenhum ciclo passa da rodada 1 (V-25). O texto de governança que declara a cadeia
  `review → bugfix → review` como máxima é substituído pelo limite de rodadas, e a restrição
  correspondente na skill de correção é atualizada para dizer que quem reinvoca é o orquestrador.
- **RF-39:** A revisão da rodada N>1 opera **somente sobre o delta**, exportando o ponto de corte por
  rodada pela variável já especificada e hoje órfã (V-23). O orçamento de revisão vigente continua
  aplicável a cada rodada.
- **RF-40:** A revisão é sempre a skill `review` do harness, idêntica nos quatro agentes. Capacidades
  nativas de revisão dos CLIs (V-16) **não** são usadas, para não introduzir veredito fora do formato
  canônico que o Ciclo controla e audita.
- **RF-41:** Cada encerramento registra um **motivo canônico** de conjunto fechado: aprovado, limite de
  rodadas, não convergiu, sem mudança, entrada bloqueada.
- **RF-42:** Cada rodada persiste evidência **própria, numerada e imutável**; o endereço de escrita
  deriva do número da rodada, de modo que uma rodada nunca sobrescreva outra. O relatório consolida
  rodadas executadas, veredito de cada uma, contagem por severidade, fingerprint e motivo de parada.
- **RF-43:** O comportamento do Ciclo é **idêntico nos quatro agentes**: mesmos vereditos, motivos,
  estrutura de evidência e teto default.
- **RF-44:** A recursão permanece hard-bloqueada: a sessão de revisão disparada pelo Ciclo nunca dispara
  um novo Ciclo dentro de si.
- **RF-45:** Os estados terminais do Ciclo são distintos de erro de infraestrutura: não convergir,
  esgotar rodadas e não produzir mudança são **resultados**, e não devem acionar o caminho de retry.

### Bloco E — Anti-falso-positivo

- **RF-46:** A tradução entre o texto do revisor e o veredito é uma **camada anticorrupção explícita e
  fail-closed**: texto sem veredito canônico declarado resulta em `BLOCKED`. É **proibido** inferir
  aprovação pela ausência de marcadores negativos, que é o comportamento atual.
- **RF-47:** A revisão produz, para **toda** tarefa ativa, um **mapa 1:1** entre cada critério de aceite
  e uma linha de evidência verificável. O confronto é **incondicional**.
- **RF-48:** Uma linha de evidência é válida somente como (a) comando executado **com saída registrada**;
  (b) referência `arquivo:linha` presente no diff revisado; ou (c) nome de teste com resultado
  registrado. Qualquer outra forma é **inválida** e o critério conta como não atendido.
- **RF-49:** Critério classificado como "não verificável pelo diff" **proíbe** o veredito `APPROVED`,
  virando achado de severidade alta que realimenta o Ciclo.
- **RF-50:** Critério não atendido é achado de severidade **mínima alta**, implicando `REJECTED` pelo
  mapeamento determinístico vigente.
- **RF-51:** Os validadores canônicos verificam o mapa 1:1 de forma **fail-closed**: tarefa não
  resolvível, ausência de seção de critérios ou mapa incompleto **falham**.
- **RF-52:** O validador Go de evidência ganha a rotina de revisão hoje inexistente (V-27), restaurando
  a paridade entre a verificação em Go e a verificação em shell.
- **RF-53:** O escape de compatibilidade legado **não** cobre o mapa 1:1 nem o critério `APPROVED`
  estrito. Não existe caminho legítimo para fechar tarefa sem prova de aprovação.
- **RF-54:** As expressões de validação não podem usar classes de colchetes com caracteres multibyte;
  devem usar alternação. A invariante é travada por caso de teste sob locale de bytes.
- **RF-55:** A cadeia requisito → tarefa → critério → evidência é verificável de ponta a ponta, e a
  integridade entre PRD, especificação técnica e tarefas continua ancorada por hash.
- **RF-56:** É proibido afirmar aprovação, execução de comando ou cobertura de critério que não tenha
  ocorrido. Na ausência de evidência, o resultado é `blocked` ou `needs_input`.

### Bloco F — Defeitos confirmados a corrigir

- **RF-57:** A revisão passa a produzir e consumir a **saída real do revisor**, eliminando o texto
  sintético que hoje alimenta o parser (V-19). Enquanto esse defeito existir, todo veredito de produção
  é ficção e nenhum requisito deste PRD é verificável na prática.
- **RF-58:** A escrita do artefato de revisão pelo Go deixa de **sobrescrever** o relatório produzido
  pela skill (V-20). Com evidência por rodada, o defeito destruiria o histórico do Ciclo.
- **RF-59:** A chave de evento inválida usada para o Copilot é corrigida para o nome oficial, e a
  asserção de teste invertida é corrigida junto (V-21) — hoje o teste protege o comportamento errado.
- **RF-60:** A desinstalação passa a remover todos os arquivos que a instalação cria (V-22), condição
  necessária para RF-05 ser verdadeiro.

### Bloco G — Release e não-regressão

- **RF-61:** A entrega é publicada como **major**, com seção de mudanças incompatíveis no changelog e
  guia de migração do agente removido.
- **RF-62:** Os três agentes remanescentes preservam comportamento observável em todos os fluxos que não
  ativam as novas capacidades. Toda mudança de default é declarada explicitamente no changelog.
- **RF-63:** O gate que reprova artefato de contrato citando caminho inexistente permanece verde após a
  entrega.

## Experiência do Usuário

O usuário primário interage por linha de comando. Fluxos afetados:

- **Instalação:** `install` sem flags detecta automaticamente os agentes presentes entre os quatro
  suportados e instrumenta cada um pelo seu mecanismo nativo. Quando um agente exigir ação do usuário
  para que o gate funcione — trust de hook no Codex, pasta confiável no Copilot — a ação é **pedida
  explicitamente**, nunca presumida. O bootstrap em repositório vazio permanece dentro da meta de tempo
  vigente e a operação continua idempotente.
- **Verificação:** `verify` reporta, por skill e por agente, o estado `current` / `missing` / `drifted`,
  agora incluindo o OpenCode, o gate de encerramento e o **estado real de cada pré-condição de
  enforcement** — um hook que não pode disparar nunca aparece como ativo.
- **Migração:** quem usava o agente removido recebe erro dirigido com o caminho de migração; o resíduo
  instalado é limpo pela desinstalação, preservando arquivos do usuário.
- **Execução de tarefa:** o usuário acompanha o Ciclo por rodadas, com veredito de cada uma. Se a tarefa
  não convergir, recebe `blocked` com histórico completo — não um sucesso ambíguo.

Acessibilidade e legibilidade: a saída de cada rodada deve ser distinguível (número, veredito, contagem
por severidade, motivo de parada), de modo que o estado do Ciclo seja compreensível sem abrir os
artefatos de evidência.

## Restrições Técnicas de Alto Nível

- **Protocolo:** integração pelo protocolo ACP já adotado, com o SDK Go já consumido. Sem protocolo novo.
- **Formas de invocação divergentes:** três agentes expõem ACP como flag; o OpenCode como subcomando. A
  abstração acomoda ambas sem caso especial no runner.
- **Mecanismos de enforcement divergentes:** cada CLI tem o seu. A paridade é de **efeito**, não de
  implementação; a lógica de decisão permanece em scripts canônicos únicos.
- **Verificação em fonte primária, obrigatória:** toda característica de CLI externo afirmada na
  especificação técnica deve ser confirmada contra fonte primária, e **comportamento de bloqueio exige
  teste de disparo real**. Característica não confirmada não pode ser implementada por suposição.
- **Pinagem de versão:** proibido `@latest`. Versões em constantes versionadas, alteradas apenas por
  decisão registrada em auditoria.
- **Segurança operacional:** a detecção não executa binários; a verificação de pré-condições, que exige
  comunicação com CLI, vive no comando de diagnóstico. O harness não concede trust pelo usuário, não
  altera o ambiente do usuário, e não realiza ação destrutiva de git ou publicação remota sem pedido
  explícito. Telemetria permanece opt-in.
- **Sanitização de ambiente:** o harness controla o ambiente dos processos que ele próprio cria e o
  sanitiza contra interruptores de governança conhecidos; o ambiente do usuário permanece intocado.
- **Escrita em configuração de terceiros:** ao mesclar o `opencode.json` do projeto, preserva `$schema`
  e todos os campos preexistentes, é idempotente e nunca remove configuração do usuário.
- **Assets embarcados e espelhamento:** novos artefatos seguem o mecanismo vigente e o espelhamento
  verificado por gate.
- **Precedência de configuração:** parâmetros novos respeitam a hierarquia vigente.
- **Domínio isolado:** o agregado do Ciclo não depende de protocolo, CLI ou filesystem, de modo que suas
  invariantes sejam testáveis sem subir sessão nem tocar disco.

## Fora de Escopo

- Adicionar outros runtimes ACP do ecossistema.
- Remover ou reescrever o runtime legado para os agentes remanescentes.
- Atualizar a versão do SDK ACP ou migrar de protocolo.
- Migrar histórico, sessões ou memória de usuários do agente removido.
- Criar arquivo de governança raiz dedicado ao OpenCode — comprovadamente desnecessário.
- Usar capacidades nativas de revisão dos CLIs em substituição à skill de revisão.
- Aposentar o escape de compatibilidade legado além de restringi-lo.
- Retomada de Ciclo após interrupção ou crash: um Ciclo interrompido recomeça.
- Interface gráfica ou painel do progresso do Ciclo.
- Tornar a telemetria obrigatória ou alterar seu modelo de consentimento.

## Suposições e Questões em Aberto

**Nenhuma questão de escopo, produto ou mecanismo permanece em aberto, e nenhuma suposição material
permanece não verificada.**

Todas as suposições das versões 1 e 2 foram resolvidas por verificação contra fonte primária ou teste de
disparo real. Três delas foram **refutadas**, e a refutação mudou o desenho:

| Suposição anterior | Desfecho | Efeito no desenho |
|---|---|---|
| Bloqueio por exceção não seria possível (contrato `Promise<void>`) | **Refutada** (V-04, V-05) | Passou a ser o mecanismo primário de enforcement |
| O hook de permissão seria o mecanismo oficial de negação | **Refutada** (V-06) | Proibido — é código morto e falharia em silêncio |
| Skills de projeto não seriam auto-carregadas | **Refutada** (V-09) | Instalador deixa de escrever configuração redundante e nociva |
| Hooks de projeto do Copilot não seriam suportados | **Resolvida** (V-12) | São suportados; o pré-requisito real é pasta confiável |
| Não haveria loop no repositório | **Refutada** (V-17) | O Ciclo promove o loop existente em vez de criar outro |

Permanece **um limite conhecido e declarado**, que não é lacuna de decisão nem suposição silenciosa:
dois interruptores de ambiente foram testados sob o modo de execução direta e não sob o runtime ACP,
tendo o modo direto e o ACP apresentado comportamento idêntico em todos os demais testes. A defesa
adotada (RF-21) **não depende desse resultado**, porque o handshake ativo valida o efeito e não a causa —
razão pela qual a lacuna não se propaga para o desenho. A especificação técnica deve, ainda assim,
replicar os dois testes sob ACP e registrar o resultado.

## Histórico de Decisões

- **v1 → v2:** incorporadas 11 decisões de escopo e 13 fatos verificados contra fonte primária.
- **v3 → v4:** correcao de rastreabilidade: cinco ponteiros da tabela de Fatos Verificados
  apontavam para IDs deslocados em duas posicoes no Bloco F, efeito colateral da insercao de RF-55 e
  RF-56 no Bloco E. Nenhum requisito mudou; apenas a referencia V->RF foi reconciliada.
- **v2 → v3:** incorporados 33 fatos verificados — parte deles por **teste de disparo real** — e 11
  decisões adicionais. Correções materiais: o enforcement do OpenCode passa a se basear no hook de
  pré-ferramenta com exceção (provado) em vez do hook de permissão (código morto); o instalador deixa de
  escrever configuração de skills e instruções (provadamente redundante e nociva); o Ciclo passa a
  promover o loop já existente em vez de criar outro; acrescentados requisitos de sanitização de
  ambiente e handshake ativo, verificação de pré-condições de trust, separação entre detecção e
  diagnóstico, guardas contra estruturas esvaziadas, unificação de ordens canônicas e correção de quatro
  defeitos confirmados que sustentam o falso positivo. O modelo de domínio que sustenta o Bloco D está
  em `discoveries/domain-loop-de-aprovacao-e-catalogo-de-agentes-cli/`.
