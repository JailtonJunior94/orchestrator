# Changelog

## Não publicado

### Mudanças de Regra

- **approval (RF-33, mudança de especificação):** `APPROVED_WITH_REMARKS` volta a encerrar tarefa,
  agora **somente quando nenhum achado é `[HIGH]` ou `[CRITICAL]`**. Basta um achado high/critical
  para o veredito não encerrar e os achados realimentarem a correção — comportamento idêntico ao
  anterior nesse caso. Achados `[MEDIUM]`/`[LOW]` passam a ser **dívida declarada**: a tarefa fecha
  e os achados permanecem registrados no relatório, visíveis e nunca apagados. `APPROVED_WITH_REMARKS`
  **sem nenhum achado declarado não encerra** (fail-closed), porque zero achados pode ser prosa não
  parseada em vez de ausência real de problema. Motivação do dono do repositório: a regra anterior
  reprovava a entrega inteira por uma ressalva de formatação tanto quanto por um defeito real, e a
  severidade é o critério verificável — "ressalva pequena" não é. A decisão passou a ser tomada por
  quem tem os achados em mãos (`Verdict.Closes(findings)` + `NewApprovalProof(verdict, criteriaMap,
  findings)`); o veredito registrado pelo revisor **nunca é reescrito**. Aplicado em paridade no Go
  (`internal/approval`, `internal/evidence`, `internal/taskloop`) e no shell
  (`validate-task-evidence.sh`, `validate-session-end.sh`, `validate-refactor-evidence.sh` e todos os
  espelhos), com o mesmo vocabulário de severidade dos dois lados.

### Correções

- **evidence:** `validate-task-evidence.sh` deixa de reportar `execution-result v2 done incompleto`
  para resultado que apenas **não está concluído**. A prova física agora tem três ramos explícitos:
  `status: done` cobra a prova integralmente; `status` não-done com veredito aprovador **falha duro**
  como tentativa de escape; `status` não-done com veredito não aprovador declara a prova física
  **não aplicável** em linha visível na saída, deixando a reprovação para o gate de veredito.
  "Malformado de verdade" (`schema_version` diferente de 2, campo obrigatório ausente) passou a ter
  mensagens próprias, distintas de "não-done". Alinha o shell ao Go, que já tratava resultado
  não-done como prova física não aplicável (`internal/sdd/state.go`).
- **hooks:** o gate de encerramento (`validate-session-end.sh`) passa a honrar `stop_hook_active` do
  JSON de entrada — antes o stdin era lido e descartado (`cat >/dev/null`), e o gate bloqueava
  indefinidamente, exigindo que o harness passasse por cima após 9 bloqueios consecutivos. Com
  `stop_hook_active: true` o gate sai 0 sem prender a sessão, continuando a imprimir o diagnóstico;
  ausente ou `false`, o comportamento de bloqueio é preservado integralmente. Mesmo tratamento
  aplicado ao `subagent-stop-wrapper.sh` (ponto `SubagentStop`).

## 2.0.0 (2026-09-11)

Release major: consolida os quatro CLIs oficiais (Claude Code, Codex, GitHub Copilot CLI,
OpenCode), remove o Gemini e substitui a rodada única de revisão por um Ciclo de Aprovação
determinístico. Nenhum default muda para Claude, Codex e Copilot além dos listados explicitamente
abaixo — em "Breaking Changes" e na seção "Mudanças de Default Declaradas (RF-62)", que audita a
árvore inteira desta release contra o diff real (O-06). Os demais fluxos permanecem byte-idênticos
(RF-62), comprovado pela suíte de não-regressão e pelos vetores 1 (ambiente do processo filho) e 2
(janela de contexto estática).

### Breaking Changes

- **agents:** remove totalmente o Gemini CLI do conjunto suportado — diretório `.gemini/`,
  `GEMINI.md`, invoker legado com aviso de depreciação, geradores de adaptador dedicados, extrator
  de métricas próprio, testes/fixtures de integração e de paridade dedicados (RF-01, RF-02, task
  10.0). Invocar `--tool gemini` produz erro tipado e explicativo (`skills.RemovedAgentError`,
  distinguível via `errors.As`) citando o conjunto suportado `{claude, codex, copilot, opencode}`
  e apontando para o guia de migração (`docs/migracao-legacy-acp.md#gemini-removido`) — nunca mais
  o erro genérico de valor inválido (RF-03).
- **agents:** adiciona o OpenCode como quarto agente oficial de 1ª classe (ACP nativo via
  subcomando `opencode acp`, versão pinada, sem `@latest`), preenchendo antes da remoção do Gemini
  as duas células de "ocupante único" que ficariam vazias — `ToolBudgetsLarge` em
  `internal/metrics/metrics.go` e `inherit_common` das regras de normalização — cada uma agora
  coberta por gate de não-vacuidade (RF-06, RF-10..RF-18, tasks 6.0/7.0/8.0).
- **approval:** `APPROVED_WITH_REMARKS` **deixa de fechar tarefa como `done`** — a regra antiga que
  permitia isso é removida; os achados da ressalva realimentam a correção. O Ciclo só encerra como
  aprovado com veredito `APPROVED` **e** mapa 1:1 completo de critério de aceite → evidência
  verificável; esgotado o teto de rodadas sem aprovação, o resultado é **sempre `blocked`**, nunca
  `done` (RF-33, RF-36, task 5.0). Quem dependia do comportamento antigo (ressalva não-crítica
  fechando a tarefa) passa a ver `blocked` com histórico de rodadas.
- **cli:** nova flag `--max-bugfix-iterations` (default **5**) expõe pela primeira vez o teto de
  rodadas do Ciclo de Aprovação, antes fixo em código sem nenhum escritor (RF-35, task 5.0).
- **taskloop:** **mudança de default no modo orquestrado, sem opt-in.** No `task-loop`, o Ciclo de
  Aprovação é engatado pela mesma condição que antes disparava a rodada única de revisão —
  perfil de revisor declarado (`--reviewer-*`) e tarefa concluída na iteração
  (`internal/taskloop/taskloop.go`, guarda `opts.Profiles.Reviewer != nil && outcome.RunReviewer`).
  A flag `--auto-review` **não** participa dessa guarda e nunca participou: ela governa apenas a
  auto-revisão do runtime ACP (`ACPRunner`), não o caminho orquestrado. Quem já usava perfil de
  revisor com critérios de aceite declarados sai do fluxo legado (revisão única, teto de 3 rodadas
  de bugfix) e passa ao Ciclo iterativo (teto de 5 rodadas e critério estrito de aprovação) **sem
  precisar ativar nada**. Efeitos observáveis: mais rodadas por tarefa, maior tempo/custo por
  tarefa e tarefas que antes fechavam podem terminar `blocked` (ver a mudança de
  `APPROVED_WITH_REMARKS` acima). Para manter o comportamento anterior, remova o perfil de revisor
  do `task-loop`; para limitar o custo, use `--max-bugfix-iterations 3`.
  O caminho do runtime ACP (`--auto-review`) permanece opt-in e inalterado nesse aspecto.
- **contextgen:** `install` deixa de sobrescrever um `CLAUDE.md` autoral preexistente. O arquivo
  passa a ser **mesclado**: o conteúdo gerado pelo harness é reescrito e todo o conteúdo que não é
  do harness é preservado dentro de um bloco delimitado
  (`<!-- ai-spec-harness:user-content-begin -->` / `...-end -->`), aplicando ao Markdown o mesmo
  princípio já usado em `.codex/config.toml`. O manifesto passa a classificar o arquivo como
  `merged` (não `created`) quando ele preexistia, de modo que `uninstall` **não o remove**
  (RF-05). Efeito observável: em projetos com `CLAUDE.md` próprio, a reinstalação deixa de causar
  perda de dado; o arquivo resultante é maior e contém o bloco preservado.
- **install/uninstall:** `opencode.json` preexistente deixa de ser reindentado. A inserção do bloco
  `permission` passa a ser uma **edição cirúrgica** do texto JSON, preservando byte a byte o
  layout autoral (indentação, alinhamento, ordem de chaves, `$schema`), e a desinstalação remove o
  bloco pelo mesmo mecanismo — `install` seguido de `uninstall` devolve o arquivo **byte-idêntico**
  ao original (RF-13). Reinstalar sobre um arquivo já convergido não reescreve nenhum byte.
- **install/uninstall:** a desinstalação passa a ser **dirigida pelo manifesto** (campo aditivo
  `InstalledFiles`), removendo os arquivos que a instalação mantém sob gestão — antes, 6 arquivos
  escapavam da limpeza (RF-05, RF-60, task 10.0). Manifesto antigo, gravado antes deste campo
  existir, cai em caminho conservador **anunciado explicitamente na saída**, preservando arquivos
  do usuário; nenhuma remoção silenciosa.
  **Precisão sobre o alcance, corrigida ainda nesta release:** o manifesto é preenchido por um
  rastreador de escrita (`internal/install/write_tracker.go`), e um arquivo cuja escrita é no-op
  por convergência não era escrito na 2ª instalação — logo saía de `InstalledFiles` e sobrevivia
  ao `uninstall`. O caso real era `.github/hooks/governance.json`, gravado por
  `RepairCopilotGovernanceHooks` apenas quando há reparo a fazer: `install` ×2 seguido de
  `uninstall` ×2 deixava `.github/hooks/governance.json` e o diretório `.github/hooks/` para trás,
  enquanto `install` ×1 + `uninstall` ×1 limpava tudo. A correção é um **registro explícito de
  arquivo sob gestão**, independente de escrita (`writeTracker.MarkInstalled` +
  `Service.trackInstalled`, chamado no call-site do reparo): o manifesto passa a declarar o que a
  instalação garante convergido, não apenas os bytes que este run gravou. Fixado por
  `internal/uninstall/roundtrip_test.go`
  (`TestRoundtrip_RepeatedInstallUninstall_LeavesOnlyUserFiles`), que reinstala antes de
  desinstalar — a suíte anterior instalava uma única vez e por isso não via o resíduo.
- **copilot:** corrige a chave do hook de encerramento de sessão do Copilot (`stop` → `agentStop`,
  o nome oficial do evento) — o gate de encerramento do Copilot, que **nunca disparava** por causa
  da chave inválida, passa a disparar de fato (RF-59, task 9.0). Quem dependia do gate
  silenciosamente inativo no Copilot passa a vê-lo bloquear encerramento de sessão com tarefa ativa
  sem veredito `APPROVED`.
- **runtime:** o veredito de revisão em produção deixa de ser derivado de texto **sintético**
  (`buildReviewOutputFromSummary`) e passa a consumir a **saída real do revisor**; a escrita do
  relatório de revisão pelo Go deixa de sobrescrever o artefato produzido pela skill (RF-57,
  RF-58, tasks 4.1/4.2). Sessões com `--auto-review`/Ciclo ativado que hoje aprovavam sempre podem
  passar a bloquear — é a correção do falso positivo que motivou esta entrega, não uma regressão.
- **runtime:** o guarda de profundidade de invocação, que impedia qualquer ciclo de correção de
  passar da 1ª rodada, agora reseta a profundidade a cada rodada do Ciclo (RF-38, tasks 4.2/5.0) —
  efeito observável apenas com o Ciclo ativado (opt-in).
- **install:** o matcher dos hooks `PreToolUse`/`PostToolUse` gravado em
  `.claude/settings.local.json` muda de `"Edit|Write"` para `"Edit|Write|apply_patch"`
  (`internal/install/install.go`, `defaultClaudeSettings`). **Alcance real: apenas projetos sem
  `.claude/settings.local.json`.** O instalador só escreve o arquivo quando ele **não existe**
  (`internal/install/install.go:864-866`); existindo, ele apenas lê o conteúdo e, se não encontrar
  `validate-preload.sh`/`validate-governance.sh`, emite um aviso pedindo edição manual
  (`install.go:869-873`). Projetos já instalados **nunca recebem o matcher novo** por reinstalação
  — para ganhá-lo é preciso editar `.claude/settings.local.json` à mão. Onde o matcher chega,
  chamadas de `apply_patch` que escapavam do gate passam a disparar os dois validadores
  (RF-62).
- **contextgen:** o `AGENTS.md` gerado por `install`/`analyze-project` deixa de citar os quatro
  agentes incondicionalmente nas seções "Notas por Ferramenta" e "Matrix de Enforcement" — passa a
  citar **apenas** os agentes efetivamente selecionados (`--tools`), corrigindo também a afirmação
  falsa de que o Copilot não tem hooks nativos (RF-09, task 10.0, subtarefas 10.12/10.13). Projetos
  que selecionam um subconjunto de agentes (ex.: `--tools claude`) verão um `AGENTS.md` menor e
  correto na próxima geração — nenhum agente fora do conjunto selecionado aparece mais.

### Mudanças de Default Declaradas (RF-62)

Auditoria completa da árvore desta release contra o diff real (`git diff --merge-base origin/main`).
Toda mudança de comportamento que ocorre **sem nenhuma configuração do usuário** está listada aqui;
as que já têm entrada própria acima não são repetidas.

**Instalador — o que passa a ser escrito**

- **claude:** matcher dos hooks `PreToolUse`/`PostToolUse` em `.claude/settings.local.json`
  (`internal/install/install.go`, `defaultClaudeSettings`). Muda **somente para projeto novo**: o
  instalador grava o arquivo apenas quando ele não existe; com o arquivo já presente ele só avisa
  (ver a entrada correspondente em "Breaking Changes" para o texto exato e o alcance).
- **codex:** `.codex/hooks.json` **deixa de ser escrito**. O enforcement do Codex passa a depender
  apenas das tabelas de hook do `config.toml`. Instalações antigas mantêm um `hooks.json` obsoleto
  que o instalador não atualiza mais — remova-o manualmente.
- **codex:** o gate de encerramento de sessão do Codex passa a existir, gravado como
  `[[hooks.SessionEnd]]` no `config.toml` (`internal/install/install.go`). É **adição**, não
  migração: contra `origin/main` não havia nenhuma tabela de encerramento no Codex — nem
  `[[hooks.Stop]]`, nem outra. Instalações antigas não tinham esse gate.
- **codex:** `sandbox_mode = "workspace-write"` e `approval_policy = "on-request"` passam a ser
  gravados no **preâmbulo** do `config.toml`, antes de qualquer tabela. Antes eram anexados depois
  de `[[skills.config]]`, posição em que o TOML os absorvia na última tabela — na prática **não
  valiam**. Agora valem de fato.
- **copilot:** o instalador passa a escrever/mesclar `.github/settings.json` com o bloco de hooks de
  governança — arquivo que antes não era criado.
- **all:** o instalador passa a copiar `.agents/hooks/validate-governance.sh` e
  `.agents/hooks/validate-session-end.sh` (validadores tool-neutros novos) em toda instalação.
- **cli:** `install` ganha `--model` (default `""`, no-op) e alimenta a chave `model` de
  `opencode.json`.
- **cli:** `verify --check-codex-trust` e `doctor --codex-trust` (ambos default `false`); a linha de
  resumo do `verify` passa a incluir sempre os contadores `inert` e `unknown` — saída default
  diferente para todo mundo.

**Desinstalador — o que passa a ser tocado**

- **uninstall:** deixa de exigir `.agents/skills/` para rodar. Em diretório sem esse diretório, o
  comando agora prossegue (removendo arquivos rastreados e reescrevendo settings) em vez de falhar
  cedo.
- **uninstall:** passa a **editar** arquivos de configuração do usuário que antes nunca tocava —
  reverse-merge de `.github/settings.json` e `opencode.json`, remoção dos blocos de hook em
  `.claude/settings.json` e limpeza de resíduo legado do Gemini.
- **manifest:** `InstalledFiles` deixa de ser uma lista estática esperada e passa a ser preenchido
  por rastreador de escrita real, com o campo novo `MergedFiles`. O conjunto removido pelo
  `uninstall` é materialmente maior — inclui arquivos de skill e adaptador que a lista estática
  nunca citou.

**Hooks e validadores (os quatro agentes)**

- **session-end:** o matcher de veredito passa a ser **ancorado** —
  `^(veredito|verdict)\s*:\s*APPROVED\s*$`. Além da remoção de `APPROVED_WITH_REMARKS` (já
  declarada), a âncora `^…$` é nova: relatório com `Veredito: APPROVED (com ressalvas)` ou com o
  veredito no meio da linha agora bloqueia o encerramento onde antes passava.
- **claude:** `validate-preload.sh` passa a sair com código **2** em vez de 1. É o código que faz o
  Claude Code bloquear a chamada e devolver o stderr ao modelo — o hook passa a **bloquear de
  fato** em vez de apenas errar. A invocação de `.agents/scripts/hook-prereq-gate.sh` nas edições
  gated e a presença de `*.cs` na lista de extensões gated **não** são novidades desta release: já
  estavam no hook contra `origin/main` (`git show ba80b56:.claude/hooks/validate-preload.sh`,
  linhas 48 e 60).
- **codex/copilot:** `validate-preload.sh` deixa de bloquear **toda** chamada de ferramenta e passa
  a bloquear apenas arquivos de código (ou `file_path` vazio). **Relaxamento:** com
  `GOVERNANCE_PRELOAD_CONFIRMED` não definido, editar `.md`/`.yaml` agora é permitido.
- **codex/copilot:** a variável de opt-out muda de `CODEX_GOVERNANCE_MODE`/`COPILOT_GOVERNANCE_MODE`
  para **`GOVERNANCE_HOOK_MODE`**. Quem exportava a variável antiga perde o opt-out em silêncio e
  volta ao modo `fail`.
- **codex/copilot:** os hooks de governança passam a ler JSON por stdin em vez de `$1`, e o matcher
  de caminho governado passa a aceitar forma relativa (`AGENTS.md`,
  `.agents/skills/*/SKILL.md`, …). Edições reportadas com caminho relativo — o caso comum de
  `apply_patch` — passam a ser bloqueadas onde antes escapavam.
- **lib:** `parse-hook-input.sh` passa a extrair caminho de corpo de `apply_patch` e das chaves
  `arguments`/`filePath`/`target_file`/`path`/`file`. É o mecanismo que faz os dois itens acima
  morderem: hooks que antes saíam 0 por não resolver `file_path` agora resolvem e enforçam.
- **task-evidence:** `validate-task-evidence.sh` só roda o confronto 1:1 de critérios quando o
  relatório declara `Estado: done` (**relaxamento** para relatórios não-`done`); em contrapartida,
  relatório `done` sem task file resolvível passa a falhar duro, sem o escape
  `AI_SDD_STRICT_EVIDENCE` (**endurecimento**).
- **review-evidence:** `validate-review-evidence.sh` passa a **exigir incondicionalmente** a seção
  `Mapa de Criterios de Aceite` no relatório de revisão, com pelo menos uma linha de critério, um
  marcador válido (`atendido` / `nao atendido` / `nao verificavel`) e uma linha de evidência em uma
  das três formas de RF-48 (comando + saída, `arquivo:linha`, teste + resultado). Critério marcado
  `nao verificavel` reprova (RF-49). **Endurecimento sem opt-out:** relatório de revisão que era
  aceito antes desta release — inclusive todo relatório gerado contra o validador de
  `origin/main`, que não conhecia a seção — passa a falhar até ganhar o mapa
  (`.agents/scripts/validate-review-evidence.sh`, RF-47/RF-48/RF-49/RF-51).
  **Correção posterior, na mesma release:** o endurecimento existia no script mas **não era
  cobrado por nenhum gate**. A suíte `tests/scripts/validate-review-evidence_test.sh` era órfã —
  existia, passava e não era referenciada pelo `Makefile`, de modo que desligar as rejeições do
  validador não quebrava nada. A suíte passou a ser executada por `make test-validators`
  (`Makefile:104`) e o inventário de suítes órfãs passou a ser verificado por
  `tests/integration/sync_gates_guard_test.go` (`TestNoOrphanValidatorTestSuites`). Só a partir
  daí o endurecimento é de fato inegociável.
- **all:** o instalador passa a gravar também `.agents/scripts/validate-session-end.sh` (entrada
  nova em `agentsScriptsFiles`, `internal/install/install.go`). Como os validadores de
  `.agents/scripts/` são tool-neutros e instalados sempre, o gate canônico de encerramento de
  sessão passa a existir em **toda** instalação, independentemente dos agentes selecionados.
- **catalog:** o script canônico do ponto PostTool muda de `.agents/hooks/post-execute-task.sh` para
  `.agents/hooks/validate-governance.sh` para os quatro agentes.

**OpenCode**

- **plugin:** `.opencode/plugin/governance.js` é **novo nesta release** — contra `origin/main` o
  arquivo não existia, nem no repositório nem como asset embarcado. Tudo abaixo descreve o
  comportamento inicial do plugin, não uma mudança sobre um comportamento anterior.
- **plugin:** `tool.execute.before` é o único ponto que **bloqueia**: roda
  `.agents/scripts/hook-prereq-gate.sh` para as ferramentas de `MUTATING_TOOLS` (`bash`, `write`,
  `edit`, `multiedit`, `patch`, `apply_patch`) e lança exceção quando o validador reprova ou
  estoura o timeout (timeout = negação, nunca aprovação). `session.idle` roda
  `.agents/scripts/validate-session-end.sh` e também lança exceção quando reprova.
- **plugin:** `tool.execute.after` **observa mas não bloqueia**. Ele executa
  `.agents/hooks/validate-governance.sh` após cada chamada e, quando o validador reprova, apenas
  emite `console.warn` (`GOVERNANCE OBSERVED at tool.execute.after: …`) — a chamada já ocorreu e
  nada é revertido. Não tratar esse ponto como enforcement.
- **runtime:** `OpenCodeKillSwitchVars` ganha `OPENCODE_DISABLE_DEFAULT_PLUGINS` (quarta variável
  sanitizada antes da orquestração), além das três enumeradas acima.

**Runtime e memória durável**

- **runtime:** falha de auto-revisão passa a marcar `ReviewStatus = "blocked"`. Antes, uma falha de
  infraestrutura na revisão apenas imprimia em stderr e a sessão mantinha o status anterior.
- **memory:** com `--durable-memory` ligado, a sessão passa a **reivindicar bastão**, subir uma
  goroutine de heartbeat e liberar na saída por default; TTL do lease = 30 min, heartbeat = TTL/2.
- **memory:** o orçamento de contexto default da memória durável fica resolvido **por classe de
  janela** (`internal/runtime/memory/durable/budget_policy.go:10-12,48-56`):
  `DefaultTotalBudgetTokens = 300` e `largeBudgetMultiplier = 2`. As classes são um par fechado
  (`internal/runtime/specs/window.go:11-15`, limiar ≥ 1M tokens), então o orçamento efetivo é
  **300 tokens para `WindowStandard` e 600 para `WindowLarge`**, repartidos 50/30/20 entre
  Projeto/PRD/Tarefa (150/90/60 e 300/180/120).
  **Não é redução de um default publicado:** o pacote `internal/runtime/memory/durable` inteiro é
  novo nesta release — contra `origin/main` (`ba80b56`) o arquivo não existe. Valores intermediários
  que circularam durante o desenvolvimento desta branch (`6000` / multiplicador `3`, efetivo
  18 000 para `WindowLarge`) **nunca foram publicados** e não constituem comportamento anterior de
  ninguém. Quem precisar de mais orçamento configura `BudgetConfig.TotalTokens` explicitamente.
- **memory:** os padrões de redação de segredo ficam mais abrangentes
  (`Authorization:\s*[^\r\n]+`, `SESSION_ID`, `^api_key=.+$`) — mais conteúdo é redigido nas
  páginas persistidas.
- **config:** as chaves `max_bugfix_iterations`, `handoff_lease_ttl` e `durable_memory_enabled`
  passam a ser resolvíveis pela cascata de configuração (`~/.aispec/config.yaml` e config de
  workspace), não só por flag.

**contextgen e gates**

- **contextgen:** `.codex/config.toml` passa a ser **mesclado** com o arquivo existente em vez de
  sobrescrito (ver também a entrada de `CLAUDE.md` acima).
- **contextgen:** o conjunto de diretórios ignorados na detecção de stack troca `.gemini` por
  `.opencode`.
- **ci:** `make test-hooks-live` **continua sendo nightly e continua não sendo gate de merge**
  (`.github/workflows/hooks-live.yml`, `schedule: cron "30 6 * * *"` + `workflow_dispatch`; o gate
  de PR é `.github/workflows/test.yml`, que não executa as build tags `acp_live`/`hooks_live`). O
  que muda é que o job perde `continue-on-error` e passa a exigir o ambiente protegido `hooks-live`
  com `AISPEC_HOOKS_LIVE=1` e credenciais reais (`ANTHROPIC_API_KEY`/`OPENAI_API_KEY`/`GH_TOKEN`),
  falhando duro sem elas — falha visível no nightly, não bloqueio de merge. O alvo `test-hooks-live`
  é **novo nesta release** (não existia no `Makefile` contra `origin/main`) e nasce já declarando
  "NAO e gate de merge" (`Makefile:130-131`); o workflow e o job declaram o mesmo. Não houve
  correção de comentário — não havia comentário anterior a corrigir. O workflow passa também a
  instalar os quatro CLIs nativos via `npm install --global`, todos com **versão pinada** (RF-11):
  `@github/copilot@1.0.51` e `opencode-ai@1.18.30` acompanham `CopilotNpmVersion` e
  `OpenCodeNpmVersion` em `internal/runtime/specs/`;
  `@anthropic-ai/claude-code@2.1.270` e `@openai/codex@0.154.0` são os CLIs nativos (distintos dos
  adaptadores ACP) e **também têm constante correspondente no registro** —
  `ClaudeCodeNpmVersion` (`internal/runtime/specs/claude.go:27`) e `CodexCLINpmVersion`
  (`internal/runtime/specs/codex.go:47`). O confronto entre as versões pinadas nos workflows e as
  quatro constantes do registro é automático: `internal/runtime/specs/workflow_pins_test.go`.
- **chore:** `MOCKERY_VERSION` `v3.7.4` → `v3.8.0` (`Makefile`, `scripts/check-mocks.sh`).

**Auditoria complementar — defaults que faltavam nesta seção**

Itens levantados por auditoria independente contra o código, depois da primeira redação desta
seção. Cada um foi reconfirmado no arquivo citado antes de ser escrito aqui.

- **task-evidence:** o **contrato de evidência v2 é always-on**. Não há flag, env var ou chave de
  config que o desligue: `.agents/scripts/validate-task-evidence.sh:22-48` parte de
  `contract_version=1` e, na ausência da prova de historicidade, faz `missing=1; contract_version=2`
  incondicionalmente; o gêmeo Go `internal/evidence/contract.go:55-67` (`ResolveContract`) também
  não lê nenhuma variável, e `internal/evidence/evidence.go:73` o invoca em todo relatório.
- **task-evidence:** o opt-out `AI_SDD_STRICT_EVIDENCE` foi **removido por completo** dos
  validadores. Contra `origin/main` o script fazia `strict_evidence="${AI_SDD_STRICT_EVIDENCE:-1}"`
  com um ramo `legacy_escape()`; hoje a variável **não é lida em lugar nenhum**. Todas as
  ocorrências restantes do nome são (a) texto de mensagem de erro dizendo que ela não reabre o
  gate e (b) casos de teste negativos que provam que `=0` não reabre nada. Quem exportava
  `AI_SDD_STRICT_EVIDENCE=0` perde o escape **em silêncio** — o comando não avisa que a variável
  deixou de ter efeito.
- **task-evidence:** o validador passa a **depender do estado do git**. O corte de contrato usa o
  ref fixo `HEAD` (`.agents/scripts/validate-task-evidence.sh:34-47`; `internal/evidence/contract.go:20`,
  `const contractCutRef = "HEAD"`) e resolve a historicidade do relatório por `git rev-parse
  --git-dir`, `git ls-files --full-name` e `git cat-file -e "HEAD:<path>"`. **Fora de repositório
  git, com o relatório não rastreado, ou apenas staged, a isenção histórica não é concedida**: o
  relatório é forçado ao contrato v2 estrito e o validador reprova. O mesmo relatório validava
  antes sem nenhuma dependência de git.
- **install:** `AGENTS.md` e `.github/copilot-instructions.md` passam a ser **mesclados**, não
  sobrescritos — o mesmo tratamento já declarado acima para `CLAUDE.md` e `.codex/config.toml`,
  mas que faltava ser declarado para estes dois. `AGENTS.md`: `internal/install/install.go:859`
  (`writeMergedMarkdownFromSource`, implementado em `install.go:1560-1579` sobre
  `contextgen.MergeUserContentMarkdown`); antes era `CopyFile` puro.
  `.github/copilot-instructions.md`: `internal/contextgen/contextgen.go:133`
  (`g.writeMergedMarkdown`); antes era `WriteFile` puro. Precisão sobre o que "mesclar" significa
  aqui (`internal/contextgen/claude_merge.go:6-7,39-45`): preserva-se o conteúdo dentro do bloco
  `<!-- ai-spec-harness:user-content-begin --> / ...-end -->` e o restante é **regerado** — não é
  merge textual de três vias. Consequência de manifesto: preexistindo, os arquivos são
  classificados `merged` e o `uninstall` **não os remove** (ver Parte "Guia de migração").
- **taskloop:** tarefa **sem critérios de aceite aborta o run** e tem o status reescrito para
  `needs_input`. `internal/taskloop/task_status_writer.go:26-28` transforma "zero critérios" em erro
  (`mapa 1:1 nao confrontavel (RF-47/RF-51)`); `internal/taskloop/taskloop.go:583-594` define
  `PostStatus = statusNeedsInput`, chama `forceTaskStatus` — que **reescreve o front-matter do task
  file e a linha correspondente da tabela em `tasks.md`** — grava `StopReason` e faz `break`,
  encerrando o loop. Limite do alcance, verificado: o gate vive dentro do ramo
  `opts.Profiles.Reviewer != nil && outcome.RunReviewer` (`taskloop.go:577`) — sem perfil de
  revisor declarado, não há abort. Efeito observável para quem usa perfil de revisor: um PRD com
  uma única tarefa sem seção de critérios para o run inteiro e tem artefatos de planejamento
  reescritos no disco.
- **claude:** o hook de preload passa a **bloquear quando não consegue extrair caminho nenhum** do
  payload. `.claude/hooks/validate-preload.sh` virou delegador do canônico
  `.agents/hooks/validate-preload.sh`, onde o caminho vazio cai no ramo permissivo do `case`
  (`:51-54`) e é barrado em `:69-75` com `exit 2` e a mensagem "ausencia de alvo e negacao, nunca
  aprovacao". Contra `origin/main` o hook fazia o oposto: `[[ -n "$file_path" ]] || exit 0`.
  A ausência da lib de parsing também passou a bloquear (`:25-28`), onde antes era `exit 0`.
- **claude:** o matcher de caminho governado passa a casar **forma relativa**
  (`.claude/hooks/validate-governance.sh:41` inclui `AGENTS.md`, `.agents/skills/*/SKILL.md` e
  `.agents/skills/*/references/*.md` ao lado das formas `*/…`). Contra `origin/main` só existiam as
  formas com prefixo, e uma edição reportada como `AGENTS.md` puro passava sem validação. É o
  mesmo endurecimento já declarado para Codex/Copilot, que faltava para o Claude.
- **contextgen:** `.codex/config.toml` passa a ser **pulado** quando já contém conteúdo gerado pelo
  harness. `internal/contextgen/contextgen.go:143-156` só grava quando o arquivo é ilegível/ausente
  **ou** quando `HasCodexGeneratedContent` é falso (`internal/contextgen/codex_merge.go:34-36`);
  contra `origin/main` era `WriteFile` incondicional. Efeito observável e contraintuitivo: um
  `.codex/config.toml` que já carrega o bloco gerado **nunca é atualizado pelo `contextgen`** —
  mudanças de conteúdo gerado em versões futuras não chegam nele por esse caminho. O `install`
  ainda o mescla por outro caminho (`internal/install/install.go:1063-1071`).
- **install:** `install` passa a **falhar duro** quando o runtime do plugin do OpenCode não está no
  `PATH`. `internal/install/install.go:1172-1174` faz `LookPath(specs.OpenCodePluginRuntime)` como
  primeira coisa de `installOpenCode` e retorna `ErrOpenCodePluginRuntimeMissing`
  (`install.go:1165`). Não havia verificação equivalente contra `origin/main`. O `verify` reporta a
  mesma sonda de forma não-fatal, como `VerifyStateMissing` (`install.go:424-435`).
- **gates:** `scripts/check-spec-paths.sh:32` ganha `.opencode/plugins` na lista `_OPTIONAL`
  (aplicada em `:65`), isentando o caminho da verificação de existência. **Declarado com a
  ressalva:** o diretório real, em todo o código e na árvore, é `.opencode/plugin` no **singular**
  (`internal/install/install.go:1176,1189,1515`; arquivo versionado
  `.opencode/plugin/governance.js`). A isenção como escrita cobre um caminho que não existe, e o
  caminho que existe não está isento.

**Ondas 5–7 — endurecimentos posteriores, dentro da mesma release**

Mudanças aplicadas depois da primeira redação deste changelog, levantadas do diff contra
`origin/main` **e** da árvore não commitada. Todas observáveis; todas reconfirmadas no arquivo
citado.

- **evidence:** contrato de evidência **versionado v1/v2 com regra de corte por ref git**. O marcador
  `<!-- evidence-contract: v2 -->` força v2; sem marcador, a isenção histórica só é concedida se o
  relatório estiver **versionado no ref de corte** — `git ls-files --full-name` seguido de
  `git cat-file -e "HEAD:<path>"`. O ref é fixo: `internal/evidence/contract.go:20`
  (`const contractCutRef = "HEAD"`) e `.agents/scripts/validate-task-evidence.sh:34`
  (`cut_ref="HEAD"`). Relatório não commitado, apenas staged, ou fora de git, cai em v2 estrito com
  `relatorio sem marcador de contrato nao e evidencia historica — nao esta versionado em HEAD`
  (`contract.go:63-65`).
- **evidence:** **remoção do knob `AI_EVIDENCE_CONTRACT_CUT_REF`.** O ref de corte deixou de ser
  controlável por ambiente. A variável não é lida por nenhum código de produção; sobrevive apenas
  em dois testes que existem para **provar que ela é ignorada**:
  `tests/scripts/validate-task-evidence_test.sh:423` (caso `TC24-cut-ref-ignora-env`) e
  `internal/evidence/regression_test.go:172`
  (`TestResolveContract_CutRefIsNotEnvironmentControlled`, "o ref de corte e HEAD fixo; o env nao
  pode estreita-lo"). Quem estreitava o corte por env perde o controle em silêncio.
- **evidence:** **endurecimento da forma (a)** de evidência (comando + saída). Registro trivial
  deixa de ser prova: veredito canônico (`pass`/`fail`/`exit N`) passa; lista de trivialidades
  (`ok`, `done`, `feito`, `sim`, `aprovado`, `n/a`, …) reprova; menos de dois campos reprova; o
  resto precisa casar sinal (dígito, caminho, extensão de arquivo-fonte ou nome `Test*`). São
  **três implementações independentes** — `internal/evidence/form.go:31-46,71-73`
  (`evidencia nao registra saida significativa do comando — registro trivial nao e prova (RF-48)`),
  `internal/approval/evidence.go:475-495` e `.agents/scripts/validate-review-evidence.sh:52-60,201-202`
  (`substantive_record()`). Precisão sobre a contagem: a implementação **shell** é replicada
  byte-a-byte em **quatro** caminhos (`.agents/scripts`, `.claude/scripts` e os dois espelhos em
  `internal/embedded/assets/`); "três" refere-se às implementações, não aos arquivos.
- **evidence:** **comparação de arquivo por caminho, não por basename.** Um arquivo declarado como
  revisado só casa com a referência por igualdade ou por sufixo de segmento completo
  (`internal/evidence/evidence.go:346-351`, `samePathOrSuffix`;
  `.agents/scripts/validate-review-evidence.sh:62-76`, `path_is_reviewed()`). O caso decisivo está
  fixado em `internal/evidence/review_form_hardening_test.go:80-104`
  (`TestReferencedFileIsComparedByPathNotBasename`): declarar `internal/evidence/form.go` **não**
  cobre `internal/inventado/form.go:22`. Antes, basename igual em diretório diferente passava.
- **runtime:** **o fallback textual do veredito virou fail-closed.** `Catalog.parseVerdict`
  (`internal/taskloop/reviewer.go:225-227`) deixou de ter corpo próprio e delega a
  `approval.NewTranslator().Translate(raw)`. Contra `origin/main` o método tinha 44 linhas e, quando
  a linha dedicada de veredito não casava, varria o texto inteiro por substring
  (`containsAnyPattern(lower, "approved", "aprovado")`) — qualquer menção em prosa produzia
  `VerdictApproved`. Esse ramo foi **removido**. Agora, ausência de linha dedicada de veredito
  retorna `VerdictBlocked` (`internal/approval/translator.go:16-23`) e token não reconhecido cai no
  `default: return VerdictBlocked` (`:46-47`). **Ausência de veredito é bloqueio, nunca aprovação.**
- **gates:** `check-skills-sync` e `check-scripts-sync` **deixam de aprovar por vacuidade**. Em
  `scripts/check-skills-sync.sh:108-123` a lista de libs exigidas passa a ser **declarada**, não
  derivada de glob, e a ausência do diretório canônico vira `DRIFT lib: diretorio canonico ausente`
  — antes, esvaziar `.agents/lib/` fazia o bloco inteiro não comparar nada e o gate aprovava.
  Espelho ausente virou `DRIFT: mirror nao existe` onde era `WARN` sem drift (`:49-51`), e a
  iteração foi invertida para percorrer o conjunto **canônico**, de modo que skill nunca espelhada
  agora é drift. Em `scripts/check-scripts-sync.sh:42-46`, validador canônico ausente virou
  `MISSING: <path> (validador canonico ausente)` + drift, onde antes havia um `continue` puro.
- **gates:** **a prova de disparo migrou para `go test -json`.** A evidência por célula passa a ser
  extraída do stream JSON de uma execução real da suíte de integração
  (`internal/runtime/specs/parity_dispatch_proof_test.go:209-211`,
  `exec.Command("go", "test", "-tags=integration", …, "-json", …)`), e
  `internal/runtime/specs/dispatch_proof_provenance_test.go:20-30` **reprova o próprio código-fonte**
  se ele contiver `AISPEC_DISPATCH_PROOF_LOG`, `dispatchProofLogEnvVar` ou `os.Getenv(` — "a
  evidência de disparo tem de vir de execução real, nunca de arquivo ou variável que o revisor
  possa fornecer". Ressalva de precisão: `make test-hooks-live` (`Makefile:132-133`) **não** usa
  `-json`; a migração é da prova de paridade de disparo, não do alvo live.
- **hooks:** **`AI_SDD_LEGACY_HOOK_CONTRACT` deixa de fechar tarefa sem prova.** Contra `origin/main`,
  `AI_SDD_LEGACY_HOOK_CONTRACT=1` desviava do `ai-spec validate-result execution` inteiro e o
  caminho legado não mencionava veredito, aprovação nem critérios — relatório vazio saía 0. Hoje
  (`.agents/hooks/post-execute-task.sh:262-286`) a variável desvia **apenas a forma** do contrato
  (YAML vs JSON v2); o **desfecho** passa pelo validador canônico nos dois caminhos. Com
  `status == done`, o hook resolve `validate-task-evidence.sh` em cascata e falha se o validador
  estiver ausente (`:278`), se o relatório estiver ausente/vazio (`:280`) ou se o validador
  reprovar (`:283`): `FAIL RF-53: relatorio de execucao nao comprova aprovacao —
  AI_SDD_LEGACY_HOOK_CONTRACT nao reabre este gate`.
- **claude:** o matcher de `defaultClaudeSettings()` passa a cobrir também **`Bash`** (e
  `NotebookEdit`): `"Edit|Write"` → `"Bash|Edit|Write|NotebookEdit|apply_patch"`
  (`internal/install/install.go:1269` PreToolUse e `:1280` PostToolUse; golden em
  `internal/uninstall/settings_fixture_test.go:7,18`). Vale o mesmo alcance da entrada de matcher em
  "Breaking Changes": o instalador só grava `.claude/settings.local.json` quando ele não existe.
- **opencode:** **`bash` sem alvo de arquivo passa a ser validado.** Em
  `.opencode/plugin/governance.js:289`, quando a chamada não produz alvos de arquivo mas carrega
  texto de comando, o plugin valida o próprio comando
  (`commandPayload` → `{"tool_input":{"command":…}}`, `:135-137`, com chave de memo própria em
  `:143-145`). Quando **nem** alvo **nem** comando são extraíveis, `resolveValidationTargets`
  (`:115-129`) nega: "absence of target is treated as denial, never approval". O ponto pós-ferramenta
  espelha isso (`:227-234`). O asset embarcado é byte-idêntico e a paridade é cobrada por
  `scripts/check-skills-sync.sh:288-300`.
- **opencode:** **ferramenta desconhecida é negada sob orquestração.**
  `.opencode/plugin/governance.js:274-282`: fora de `MUTATING_TOOLS` e fora de `READ_ONLY_TOOLS`,
  se `isOrchestrated()` (`AISPEC_OPENCODE_ORCHESTRATED == "1"`, `:37-39`) a chamada **lança** —
  "an unclassified tool is treated as denial under orchestration, never approval; classify it in
  governance.js before retrying" (`:41-47`). Fora de orquestração, degrada para um aviso único por
  ferramenta. Efeito observável: ferramenta nova do OpenCode passa a quebrar a sessão orquestrada
  até ser classificada no plugin.
- **copilot:** **rota de reparo do `governance.json` obsoleto.**
  `internal/upgrade/copilot_governance.go` passa a migrar as chaves mortas
  `stop`/`Stop`/`sessionEnd`/`SessionEnd` (`:17`) para a chave oficial `agentStop` (`:97-108`),
  criando o arquivo com os defaults quando ausente (`:52`), mesclando as entradas canônicas
  de-duplicadas por JSON exato (`:118-121,148`) e preferindo edição cirúrgica
  (`SetJSONTopLevelKey`) à remarshalização (`:126-128`). Idempotente e convergente. Invocado por
  `internal/upgrade/upgrade.go:171` e `internal/install/install.go:1137`. É o complemento do fix de
  `stop` → `agentStop` declarado em "Breaking Changes": **instalações antigas são reparadas**, não
  só as novas ficam corretas.
- **evidence (estado da árvore, não mudança de código):** o **selo de evidência** está presente em
  **27 dos 28** resultados de execução do repositório — `ls .specs/*/*_execution_result.json` → 28;
  `grep -l commit_patch_sha256 .specs/*/*_execution_result.json | wc -l` → 27. O único sem selo é
  `.specs/prd-harness-quatro-clis-loop-aprovacao/9.0_execution_result.json`, que está em
  `status: blocked` e por isso não é selável (`SealEvidence` recusa status diferente de `done`).
  Declarado com a ressalva: **nenhum gate emite esse número** — é propriedade observada da árvore,
  não saída de suíte. Correção de uma afirmação anterior desta mesma seção, que dizia "9 dos 28":
  o número estava desatualizado porque o commit `15c143b` selou os demais depois da redação.
  Os dois comandos acima foram reexecutados e conferem com o número declarado (RF-56).
- **specs (evidência, não mudança de código):** **evidência fabricada foi removida dos relatórios de
  execução.** A concentração está em
  `.specs/prd-harness-quatro-clis-loop-aprovacao/9.0_execution_report.md`: saiu o bloco inteiro de
  `## Critérios de Aceite` no formato `<critério> -> comprovado: evidence/task-9.0/<arquivo>.log`,
  substituído por linhas honestas (`Parcialmente verificado`, e `Bloqueado: RF-28 exige doze
  execuções observadas pelos CLIs nativos. A evidência anterior só registrava a presença de
  'GOVERNANCE', não a negação e a terminação da ação; não é prova suficiente`); o estado foi de
  `done` para `blocked`; e `9.0_execution_result.json` perdeu `"status": "done"` e
  `"review_verdict": "approved"`. Natureza do que saiu: **afirmações critério→evidência cujos
  artefatos citados não as sustentam** — vários logs citados em `evidence/task-9.0/` têm **0 bytes**
  (`build.log`, `vet.log`), e uma das linhas removidas afirmava que
  `.github/workflows/hooks-live.yml` continha `continue-on-error: true`, string que não existe no
  workflow. Autocorreções do mesmo tipo estão registradas em `4.6_execution_report.md` e
  `4.8_execution_report.md` ("era falso — o grep literal retornava zero linhas"). A contagem de
  "201 tokens" citada internamente **não foi reconferida** e por isso não é afirmada aqui.

### Features

- **domain:** novo pacote `internal/approval` — agregado do Ciclo de Aprovação sem dependência de
  protocolo ACP, CLI ou filesystem; dono da contagem de rodadas, veredito corrente e critério de
  parada; consumido por `Service.Execute`, `RunLoop` e `ACPRunner` (RF-30, RF-31, RF-34, RF-41,
  RF-42, RF-43, RF-44, RF-45, tasks 2.0, 4.1–4.8).
- **domain:** mapa 1:1 critério de aceite → evidência como dado verificável e fail-closed; critério
  "não verificável pelo diff" proíbe `APPROVED` (RF-46..RF-54, task 3.0).
- **opencode:** enforcement não-desligável — hook de pré-ferramenta por exceção, permissões
  declarativas, sanitização do ambiente do processo filho contra os interruptores conhecidos
  (`OPENCODE_PURE`, `OPENCODE_DISABLE_PROJECT_CONFIG`, `OPENCODE_DISABLE_EXTERNAL_SKILLS`) e
  handshake ativo do plugin de governança antes do primeiro prompt (RF-19..RF-21, task 8.0).
- **opencode:** janela de contexto derivada do modelo por tabela versionada (casamento exato e por
  maior prefixo), com fallback conservador declarado e testado para modelo não resolvível — sem
  efeito para Claude/Codex/Copilot, cuja janela permanece estática (RF-17, task 7.0).
- **hooks:** gate canônico de encerramento de sessão presente nos quatro agentes, bloqueando
  encerramento com tarefa ativa sem veredito `APPROVED` (RF-27, task 9.0).
- **cli:** nova família de comandos `ai-spec memory` para operação humana da memória durável de
  agentes, com **oito subcomandos diretos** — `show`, `search <termo>`, `export`, `compact`,
  `migrate`, `promote`, `restore` e `handoff` — e **três subcomandos aninhados** sob
  `memory handoff`: `status`, `claim` e `release` (`cmd/ai_spec_harness/memory.go`, declarados em
  `docs/cli-schema.json`). A família opera diretamente sobre os colaboradores do subsistema
  (`Layer`, políticas e `HandoffLease`), **sem passar pela fachada** usada pelo runtime (MD-001).
  Superfície puramente aditiva: nenhum comando preexistente tem contrato, flags ou saída
  alterados, e nada aqui depende de `--durable-memory` estar ligado no `task-loop`. As mudanças de
  runtime da memória durável continuam declaradas na seção "Runtime e memória durável" (RF-62).
- **traceability:** `ai-spec check-traceability <diretorio-prd>` deriva e verifica a cadeia
  requisito → tarefa → critério → evidência a partir dos próprios artefatos (RF-55, task 11.0).
- **catalog:** registro único de agentes (`internal/runtime/specs.Registry`) — elimina as duas
  ordens canônicas divergentes e os dois catálogos ACP espelhados que hoje coexistiam (RF-07,
  task 6.0).

### Fixes

- **evidence (gate estruturalmente insatisfazível):** `Orchestrator.ValidateExecutionEvidence`
  (`internal/taskloop/orchestrator.go`) deixa de recomputar a prova física contra um snapshot da
  árvore de trabalho capturado **no instante da validação**. Quando o resultado está selado
  (`commit_sha` + `commit_patch_sha256`), a validação passa a recompor o patch a partir do **ponto
  de corte git em que a tarefa foi executada** (`VerifySealedEvidence`, range `base_sha..commit_sha`),
  e o artefato de patch é conferido pelo digest registrado no fechamento. Efeito medido: a varredura
  nos 28 relatórios versionados saiu de **PASS=0 / FAIL=28** para **PASS=17 / FAIL=11**, e as 11
  reprovações remanescentes são substantivas (veredito `APPROVED_WITH_REMARKS`), não estruturais.
  O caminho não selado preserva o comportamento anterior byte a byte.
  Verificação adversarial: adulterar `commit_patch_sha256`, `base_sha`, `commit_sha` ou
  `patch_sha256` faz o gate reprovar nos quatro casos.
- **evidence (determinismo do selo):** a verificação de um resultado selado passa a usar sempre o
  mesmo conjunto de exclusões usado na selagem, derivado do próprio resultado
  (`<prd-dir>/<task_id>_execution_result.json`), em vez de herdar as exclusões operacionais do
  chamador. Antes, `validate-result --verify-physical` acrescentava o relatório Markdown ao conjunto
  de exclusões, o que mudava o patch recomposto e fazia todo selo legítimo divergir — um selo só é
  re-auditável se o verificador não puder alterar o recorte.
- **evidence (RF-53, escape de compatibilidade):** o escape de contrato v1 em
  `validate-task-evidence.sh` deixa de cobrir o **mapa 1:1 de critérios de aceite**. RF-53 diz
  explicitamente que a isenção legada não cobre o mapa 1:1; na prática os 28 relatórios do
  repositório (100% do universo, nenhum com marcador `<!-- evidence-contract: v2 -->`) passavam com
  o gate central de RF-47/RF-51 inerte. A isenção v1 agora cobre somente a forma da evidência.
  Nenhum mapa foi inventado: os 28 relatórios já declaravam `## Critérios de Aceite` com task file
  resolvível, e nenhum passou a reprovar por mapa incompleto. Verificação adversarial: remover uma
  linha `-> comprovado:` de um relatório com 9 critérios faz o gate reprovar com
  `critérios de aceite comprovados (8) < definidos na task (9)`.
- **evidence (RF-33/RF-36, `blocked` como escape):** `validate-session-end.sh` filtrava apenas
  `in_progress` e `done`, de modo que mover uma tarefa para `blocked` a removia do gate de
  encerramento. Uma tarefa `blocked` **com relatório de execução escrito** é materialmente ativa e
  passa a ter o desfecho cobrado; uma tarefa `blocked` sem relatório continua fora do gate, porque
  não há nada a cobrar. Efeito medido **neste próprio repositório**: o gate saiu de `exit 0` para
  `exit 2`, apontando as 11 tarefas `blocked` que têm relatório com veredito
  `APPROVED_WITH_REMARKS`. Esse é o comportamento correto e não foi enfraquecido para ficar verde.

- **evidence (RF-52, paridade validador↔Ciclo):** a extração de veredito dos validadores passa a usar
  a mesma semântica do Ciclo de Aprovação. `reviewverdict.ParseDocument` (novo) ignora vereditos
  declarados dentro de cercas de código e devolve `BLOCKED` quando há vereditos contraditórios —
  exatamente o que `approval.Translator` já fazia. `internal/evidence` usava `ParseText`, que pegava
  a primeira declaração e não enxergava cercas. Efeito medido nos três vetores perigosos (validador
  aprovava, Ciclo recusava): veredito só dentro de cerca `APPROVED → sem veredito canônico`;
  vereditos contraditórios `APPROVED → BLOCKED`; veredito fora da cerca vencendo o de dentro
  `APPROVED → REJECTED`. Travado por `TestReviewVerdictExtractionMatchesApprovalCycle`, que confronta
  as duas implementações em 12 casos.
- **evidence (RF-51, mapa 1:1 inverificável):** `validate-review-evidence.sh` e `internal/evidence`
  não liam a task file, então não conseguiam confrontar completude — um review cobrindo 1 de 2
  critérios passava nos dois. O relatório de review passa a declarar `- Task file: <caminho>`
  (campo novo, obrigatório) e os dois validadores confrontam a contagem de linhas do mapa contra os
  critérios da task, fail-closed quando a task não é resolvível. Paridade Go↔shell provada nos
  quatro casos por `TestValidateReview_ConfrontaCompletudeContraTaskFile`.
- **governance (escapes sem auditoria):** `GOVERNANCE_PRELOAD_CONFIRMED=1` e
  `GOVERNANCE_PRELOAD_MODE=warn` continuam desligando o gate pré-ferramenta, mas agora deixam
  registro: cada uso anexa `timestamp / motivo / alvo / ferramenta / usuário` em
  `.aispec/governance-escapes.log` (configurável por `GOVERNANCE_ESCAPE_LOG`) e emite `AUDITORIA:`
  em stderr. O escape deixa de ser silencioso.
- **governance (cobertura de extensões):** o gate pré-ferramenta cobria apenas
  `.go .py .ts .js .tsx .jsx .cs`; `.sh`, `.rb`, `.java` e `.sql` saíam com `exit 0` — inclusive os
  próprios hooks e validadores shell deste repositório, que ficavam fora do gate que eles mesmos
  implementam. A lista passa a cobrir shell, Ruby, Java, SQL, Rust, Kotlin, Swift, PHP, C/C++,
  Scala, Elixir, Perl, Lua e as variantes de módulo JS/TS. Artefato `.md` continua fora.
- **ci (job que falhava sempre):** `scripts/test-sdd-evals.sh` exigia
  `.specs/prd-sdd-robusto/sdd-state.json`, diretório removido no commit `91f8cb9 (chore) remove
  docs`. O script nunca foi atualizado, então o job `sdd-evals` do workflow de testes falhava duro
  em toda execução, nos três sistemas operacionais da matriz. O bloco de estado SDD passa a ter
  escopo declarado (`SDD_EVALS_PRD_DIR`) e é ignorado quando o diretório não existe, com aviso
  explícito em stderr; o corpus adversarial — o valor real do job — continua rodando
  incondicionalmente (21 fixtures, 20 rejeitadas, 1 controle aceito). Quando o escopo existe, o gate
  continua reprovando checkpoint legado e estado ausente.

- **events:** o OpenCode passa a ter **tabela de alias própria** em `normalization-rules.yaml`,
  derivada de `MUTATING_TOOLS` do próprio plugin de governança (`bash`, `write`, `edit`,
  `multiedit`, `patch`, `apply_patch`), mais `input_mappings.opencode` para a chave de comando do
  `bash`. Antes o driver herdava `common_aliases` (`read_file`/`write_file`/`str_replace_editor`),
  nomes que o OpenCode nunca emite — na prática `normalized_name` era sempre igual a `raw_name` e a
  paridade de renderização de tool-calls era nominal (RF-18). `inherit_common` continua declarado e
  não-vazio (RF-06); a tabela explícita tem precedência. O espelho
  `.agents/normalization-rules.yaml` volta a ser byte-idêntico ao embarcado, com gate próprio.
- **parity:** o agente OpenCode ganha invariantes de verdade — `OC01` (plugin
  `.opencode/plugin/governance.js` instalado e registrando `tool.execute.before`), `OC02`
  (`opencode.json` com bloco `permission` sem `"ask"` e sem `deny` geral em ferramenta exigida) e
  `OC03` (`opencode.json` nunca escreve `skills`/`instructions`) — além de entrar no mapa do
  invariante cross-tool `X01` (via `AGENTS.md`, seu artefato canônico) e no `INV-32`, com fixture
  `tests/fixtures/parity/opencode_*.jsonl`. Antes, nenhum invariante declarava
  `AppliesTo=opencode` e a suíte de paridade do agente era verde por vacuidade.
- **scripts:** `sync-acp-sdk-version.sh` passa a sincronizar **todas** as constantes `*SDKVersion`
  de `internal/runtime/specs/` com o `go.mod` (antes só `ClaudeSDKVersion`), varrendo o diretório
  em vez de listar arquivos — agente novo é coberto sem editar o script. Teste novo compara o
  `go.mod` com o `SDKVersion()` de todo o registro de agentes (RF-11).
- **gates:** `scripts/check-spec-paths.sh` passa a **falhar** quando o escopo é vazio, em vez de
  sair 0 com "nada a verificar" — um gate sem alvo aprova por vacuidade. O alvo `make
  check-spec-paths` declara os PRDs ativos em `SPEC_PATH_TARGETS`; `SPEC_PATHS_ROOT` permite
  controlar a raiz da varredura (RF-63).
- **hooks:** `.opencode/plugin/governance.js` passa a existir como espelho de raiz obrigatório do
  asset embarcado, **versionado no git**; `check-hooks-sync.sh` deixa de comparar o plugin
  condicionalmente e `sync-hooks.sh` passa a materializar o espelho. Como o gate agora falha
  (`MISSING: .opencode/plugin/governance.js`) quando o arquivo não está na árvore, deixá-lo fora
  do controle de versão quebraria o CI em clone limpo.
- **install:** chave de evento do hook de encerramento do Copilot corrigida (ver Breaking Changes).
- **uninstall:** desinstalação incompleta corrigida via manifesto por arquivo (ver Breaking
  Changes).
- **runtime:** veredito sintético e sobrescrita do artefato de revisão corrigidos (ver Breaking
  Changes).

### Documentation

- **migration:** guia de migração do Gemini removido em
  `docs/migracao-legacy-acp.md#gemini-removido`, referenciado pelo erro tipado de invocação.
- **sdd:** rastreabilidade requisito → tarefa → critério → evidência ancorada por hash
  (`spec-hash-prd`/`spec-hash-techspec` em `tasks.md`), verificável via `ai-spec check-spec-drift`
  e `ai-spec check-traceability` (RF-55, task 11.0).

### Estado Conhecido desta Release

Três fatos verificados sobre a **própria entrega**, declarados aqui por decisão do dono do
repositório. Nenhum está mascarado e nenhum foi corrigido antes da publicação.

**1. A cadeia de rastreabilidade desta entrega não é verificável.**

`ai-spec check-traceability .specs/prd-harness-quatro-clis-loop-aprovacao` sai **1**, com:

```
RUPTURA: gate_vacuous: 18 tarefa(s) no escopo, 0 verificada(s) — gate vacuo: nenhuma tarefa teve
seus criterios confrontados contra a forma estrita de evidencia ...
1 ruptura(s), 0 tarefa(s) verificada(s) de 18, 18 isencao(oes) declarada(s).
```

As 18 tarefas foram executadas **antes** de o critério estrito de evidência existir, e por isso
caem todas na isenção de contrato v1 (`AVISO: historical_evidence_contract`). A isenção existe
justamente para **não fabricar evidência retroativa** — o que seria a violação mais grave de
RF-56. O preço é explícito e está sendo pago com o nome certo: **o gate não verificou nada nesta
entrega**. O universo integralmente isento não é cadeia verificada e não pode ser lido como
aprovação — é o próprio validador quem diz isso, e é por isso que ele sai 1 em vez de 0.
O gate passa a verificar de verdade a partir da **primeira tarefa executada sob o contrato v2**.

**2. Onze relatórios de execução reprovam no validador canônico por veredito inválido.**

Varredura completa nos **28** relatórios versionados
(`for f in $(git ls-files | grep _execution_report.md); do bash .agents/scripts/validate-task-evidence.sh "$f"; done`)
com a árvore limpa: **17 aprovam, 11 reprovam**. Os 11 são **1.0, 4.6, 4.7, 6.0, 7.0, 8.0, 9.0,
10.0 e 11.0** do PRD `prd-harness-quatro-clis-loop-aprovacao` e **7.0 e 8.0** do PRD
`prd-memoria-duravel-agentes`, todos com:

```
FALTANDO: veredito do reviewer não encerra o ciclo de aprovação: APPROVED_WITH_REMARKS
(RF-53: a isenção histórica cobre a forma da evidência, nunca o desfecho; somente APPROVED encerra).
```

Esses onze encerram com `APPROVED_WITH_REMARKS` — critério que o PRD desta release passa a declarar
**inválido** para fechar tarefa (ver "Breaking Changes"). A regra nova reprova retroativamente os
relatórios produzidos sob a regra velha, **inclusive o 11.0**, que é o relatório que declara esta
própria entrega. É dívida conhecida e assumida, não mascarada: nenhum relatório foi reescrito para
passar no gate. O `11.0` acumula um segundo defeito real: o critério
`changelog-breaking-changes-section-complete` não referencia evidência declarada.

**Correção de uma afirmação falsa publicada antes nesta mesma seção.** A redação anterior dizia que
os relatórios reprovavam também em `prova fisica invalida` "porque o estado final recomputado
diverge do declarado **enquanto a árvore está suja**". Isso era **falso e foi verificado como
falso**: com `git status --porcelain` vazio e tudo commitado, os 27 relatórios selados continuavam
reprovando. A causa real era um defeito estrutural do próprio gate, corrigido nesta entrega —
`Orchestrator.ValidateExecutionEvidence` recomputava a prova física contra um snapshot da árvore
capturado **no instante da validação**, de modo que qualquer avanço do `HEAD` posterior à execução
da tarefa tornava a comparação impossível de fechar. O gate era **estruturalmente insatisfazível**,
não uma consequência de sujeira na árvore. Ver a entrada correspondente em "Bug Fixes".

**3. O Ciclo de Aprovação nunca foi executado neste repositório.**

Não existe nenhum diretório `evidence/**/review/round-N/` na árvore — a busca por diretórios
`round-*` e por `review/` sob `evidence/` não retorna nada. Toda a evidência dos **RF-30 a RF-45**
(agregado `internal/approval`, contagem de rodadas, veredito corrente, critério de parada, reset de
profundidade por rodada, condução por `Service.Execute`/`RunLoop`/`ACPRunner`) é de **teste
unitário e de integração**. A suíte verde comprova que o código faz o que o teste pede; ela **não**
comprova uso em produção. O Ciclo entra nesta release sem nenhuma rodada real registrada.

> **Nota de proveniência:** as entradas acima foram compiladas a partir dos relatórios de execução
> de cada tarefa (`.specs/prd-harness-quatro-clis-loop-aprovacao/*_execution_report.md`) e da
> especificação técnica, não apenas do histórico `git log` — a maior parte do trabalho desta
> release estava presente na árvore de trabalho, ainda não commitada, no momento da preparação
> deste changelog. Nenhum hash de commit é citado onde a mudança correspondente ainda não foi
> commitada.

## 1.1.0 (2026-09-05)

### Features
- **gates:** reprova artefato de contrato que cita caminho inexistente (abf3247)

### Documentation
- **sdd:** reconcilia a techspec com a estrutura real do codigo (f7f53fe)

## 1.0.2 (2026-09-05)

### Bug Fixes
- **skills:** selo de evidencia nao pode bloquear projeto sem contrato SDD (e1bfdeb)

## 1.0.1 (2026-09-05)

### Bug Fixes
- **semver:** detecta BREAKING CHANGE apenas como footer (2bb3001)

## 1.0.0 (2026-09-05)

### Features

### Documentation
- **sdd:** integra o selo ao fluxo e documenta o ciclo de vida (b3b176a)

### Breaking Changes
- **scripts:** fecha o gate de aceite fail-open por padrao (b2508a1)

## 0.30.0 (2026-09-05)

### Features
- **sdd:** torna a evidencia auditavel apos o commit (RF-14) (c5a27a4)

### Chores
- **deps:** atualiza Go para 1.27.1 e golangci-lint para 2.13.2 (d239d85)

## 0.29.3 (2026-09-05)

### Bug Fixes
- **fs:** torna FakeFileSystem agnostico ao separador de caminho (62b15f4)
- **sdd:** estabiliza digest de spec em checkout com CRLF (c5fb328)
- **tests:** repara fixtures do gate de hooks e remove dependencia de rtk (2049a4d)
- **scripts:** torna gate de aceite fail-closed em awk byte-oriented (68ac9c8)
- **taskloop:** elimina panic no parser com header de tabela parcial (44b8a03)

### Documentation
- **sdd:** agenda modo estrito como padrao para v0.31.0 (119eac8)
- **sdd:** registra rodada de prontidao para producao (BUG-127..130) (551ce98)

### CI
- torna fuzz e cobertura por pacote bloqueantes (09fccbe)
- completa alvos de fuzz e adiciona gate de skills portateis (2c9c4cd)

## 0.29.2 (2026-09-01)

### Bug Fixes
- **ci:** torna smoke de adaptadores portável (8d94675)

## 0.29.1 (2026-09-01)

### Bug Fixes
- **sdd:** rejeita paths absolutos em qualquer plataforma (fc80388)

## 0.29.0 (2026-09-01)

### Features
- **sdd:** implementa fluxo robusto e verificável (09c837b)

## 0.28.1 (2026-08-26)

### Bug Fixes
- **ci:** confia no tap homebrew no setup-ai-spec e corrige comandos de instalacao (1dd0bb9)

## 0.28.0 (2026-08-26)

### Features
- **skills:** gate skills --verify com verificacao de hash SHA-256 (c9f61b6)

### Bug Fixes
- **cli:** exit code 2 real para uso incorreto no task-loop (de463a9)

### Documentation
- corrige guia de instalacao e alinha referencias ao gate skills --verify (0add30a)

### Chores
- remove .pyc rastreado e ignora caches e temp de sessao (9130549)

## 0.27.1 (2026-06-02)

### Bug Fixes
- **codex,tests:** hooks como MatcherGroup e snapshot tests consistentes (d8e9bc3)

### Documentation
- **readme:** adiciona secao Instalacao em outros repositorios (b8933f5)

## 0.27.0 (2026-06-02)

### Features
- **governance:** descoberta cirurgica cross-CLI com gates inegociaveis (8bb379d)

### Chores
- **mocks:** regenera mock FileSystem apos correcao do FakeFileSystem (8ba6718)

## 0.26.0 (2026-05-30)

### Features
- **skills:** endurecimento production-proof da cadeia de governanca (57afd0c)
- **install:** paridade cross-CLI de validadores e metadado category (7c8d78e)

### Bug Fixes
- **skillbump:** pular skill removida desde a ultima tag (bbd30ab)
- **production-proof:** corrige 14 achados cross-tool na causa raiz e remove bubbletea (a34df86)

### Documentation
- **specs:** artefatos PRD-first da auditoria production-proof (380a3b0)

### Build
- gate de sync de scripts e suite de validadores (41e15d2)

## 0.25.0 (2026-05-30)

### Features
- **skills:** reforça node/python/dotnet com regras [HARD], lazy-loading e arquitetura (v1.2.0) (03a1560)

### Documentation
- **changelog:** registra reforço das skills node/python/dotnet e remove prompts obsoletos (c998e75)

## 0.24.2 (2026-05-30)

### Bug Fixes
- **ci:** exclui pacotes de mocks gerados do calculo de cobertura (5a3edf9)

## 0.24.1 (2026-05-30)

### Refactoring
- **go:** aplica Regras Estritas 0–7 ao código (R1, R5.2, R5.11, R5.16, R5.26) (900d8af)

### Documentation
- **go-implementation:** completa integração das Regras Estritas 0–7 (v1.2.0) (e09c09f)

## 0.24.0 (2026-05-30)

### Features
- **skills:** adiciona skill dotnet-csharp-implementation (efc8127)

### Documentation
- **prompts:** remove prompts compozy obsoletos e adiciona skills de implementacao (444936f)

## [Unreleased]

### Correções
- **scripts:** o gate de critérios de aceite voltava `exit 0` para relatórios com critério não comprovado em runners com `awk` byte-oriented (mawk, padrão no Linux). A classe de bracket multibyte `Crit[eé]rios` nunca casava, o gate se desligava sozinho e a evidência passava sem prova — fail-open no invariante que RF-01/RF-03 exigem fail-closed
- **sdd:** digest de spec deixava de bater em checkout com `core.autocrlf=true` (padrão do Git for Windows), marcando artefatos aprovados como `stale` sem ninguém tê-los editado. Afetava qualquer usuário de Windows, não só a CI
- **fs:** `FakeFileSystem` comparava containment de diretório com `/` fixo e ficava cego ao separador nativo do Windows, quebrando a descoberta de agentes
- **taskloop:** `panic: index out of range` em `ParseTasksFile` com header de tabela parcial

### Funcionalidades
- **scripts:** `AI_SDD_STRICT_EVIDENCE=1` fecha os escapes de compatibilidade de `validate-task-evidence.sh` (NFR-01), fazendo falhar o relatório cuja task file não seja resolvível ou cuja task não declare seção de critérios

### Funcionalidades
- **sdd:** `ai-spec seal-evidence` vincula a evidência de uma tarefa ao commit que a contém (RF-14). A prova de fechamento depende da árvore de trabalho viva e evapora no commit; o selo grava `commit_sha` + `commit_patch_sha256` e torna a evidência re-auditável indefinidamente, a partir apenas dos dois SHAs

### Correções
- **sdd:** um artefato aprovado e depois editado ficava travado para sempre — constava como `approved` (reaprovação recusada) mas com digest divergente (todo downstream falhando), e `invalidate --from` marca os descendentes, nunca a origem. Um PRD aprovado não podia ser emendado

### Funcionalidades
- **scripts:** o gate de critérios de aceite passa a ser **fail-closed por padrão** (0.31.0). Um relatório cuja task file não seja resolvível, ou cuja task não declare critérios, agora falha em vez de emitir aviso. `AI_SDD_STRICT_EVIDENCE=0` reabre o legado apenas para migração, com aviso explícito de que a evidência assim validada não comprova os critérios

### Depreciações
- **scripts:** os escapes de legado do gate de critérios de aceite passam a emitir aviso com prazo explícito e **serão removidos em `v0.31.0`**, quando o modo estrito vira o padrão. A janela de duas versões menores concedida por NFR-01 cobre `0.29` e `0.30`. Para validar desde já, exporte `AI_SDD_STRICT_EVIDENCE=1`

### CI
- **test.yml:** `fuzz` e `coverage por pacote` deixam de ser `continue-on-error` e passam a reprovar o job — o panic em `ParseTasksFile` estava no corpus e escapou por semanas porque a CI o encontrava e seguia verde
- **test.yml:** passo de fuzz recupera paridade com `make fuzz` (9 alvos) e `make test-portable-skills` passa a rodar

### Funcionalidades
- **skills:** adiciona skill `dotnet-csharp-implementation` com SKILL.md + 17 referências para implementação .NET 10/C# 14, espelhada em `.agents/`, `.claude/`, `.github/` e `internal/embedded/assets/`
- **detect:** adiciona detecção automática de projetos .NET/C# (`.csproj`, `*.sln`, `.NET`)
- **prerequisites:** adiciona validação de pré-requisitos para .NET (`dotnet`, `dotnet-sdk`)
- **contextgen:** adiciona geração de contexto específico para .NET/C#
- **skills:** registra `dotnet-csharp-implementation` no catálogo interno de skills
- **analyze-project:** atualiza template e script de governança para suportar .NET/C#

### Documentação
- **skills(node/python/dotnet-implementation):** reforça as três skills ao nível de robustez e economia da `go-implementation`, atacando os gaps da auditoria comparativa (bump `1.1.0` → `1.2.0` em cada uma):
  - adiciona seção **Regras Estritas Obrigatórias** com severidade `[HARD]` (bloqueante de merge) em cada `SKILL.md` — R0–R7 em node/python e R0–R6 em dotnet — derivadas das orientações inline e elevadas a contrato
  - adiciona **Checklist de Validação** na Etapa 5 de cada skill, com gates concretos e greps de regressão (type check, lint, testes, format)
  - adiciona bloco **TL;DR** (summary/keywords/load-when) em 17/17 referências de node e 17/17 de python (antes 0/17), restaurando o lazy-loading por complexidade esperado por `agent-governance`
  - adiciona nota de **teto por complexidade** (trivial/standard/complex) nas três `SKILL.md`
  - expande `node/architecture.md` e `python/architecture.md` com monólito modular, monorepo/workspaces, projetos legados e escala pequeno/médio/grande
  - propaga o canônico `.agents/skills/` para os mirrors `.claude/skills`, `.github/skills` e `internal/embedded/assets/.agents/skills` via `scripts/sync-skills.sh` (drift zero em `make check-skills-sync`)
- **prompts:** expande `go-implementation-strict-rules.md` com regras estritas de implementação Go
- **skills(go-implementation):** incorpora por completo as Regras Estritas 0–7 à skill (bump `1.1.0` → `1.2.0`):
  - adiciona em `references/build.md` o **Checklist de Validação (R0–R7)** que o `SKILL.md` (Etapa 5) já referenciava mas não existia — corrige referência pendente; inclui greps de R0 (`init()`), R1 (funções standalone), R3 (`mockery.yml`/mocks atualizados), R4 (`suite.Suite`/`SetupTest`/`suite.Run`), R5/R6 (`os.Exit`/`log.Fatal`/`panic`/globals/context em struct) e R7 (`interface{}`), mais os gates `go build/vet/test -race`, `golangci-lint` e `mockery --dry-run`
  - adiciona em `references/interfaces.md` a **Regra 6** explícita: R6.1 (`context.Context` como 1º parâmetro, nunca em struct, propagação obrigatória), R6.2 (tipos concretos por padrão), R6.3 (interface no pacote consumidor) e R6.5 (tabela de decisão sentinel vs tipo customizado)
  - propaga o canônico `.agents/skills/` para os mirrors `.claude/skills`, `.github/skills` e `internal/embedded/assets/.agents/skills` via `scripts/sync-skills.sh` (drift zero em `make check-skills-sync`)
- **agents:** atualiza `AGENTS.md` e `GEMINI.md` embarcados com referência à nova skill

### Refatoração
- **internal/cmd:** aplica integralmente as Regras Estritas Go 0–7 ao código (alterações *behavior-preserving*, sem mudança de comportamento público):
  - **R1:** converte funções standalone em métodos de struct em todos os pacotes `internal/` e `cmd/`, com holders dedicados por pacote (24 arquivos `r1_catalog.go`/`r1_receiver.go`); zero funções standalone remanescentes (exceto `New*`/`main`/`Execute`)
  - **R5.2:** adiciona 38 assertions de interface em tempo de compilação (`var _ Iface = (*T)(nil)`)
  - **R5.11:** garante comma-ok em type assertion no `internal/runtime/probe` (erro defensivo no caminho de falha)
  - **R5.16:** remove `os.Exit` fora de `main` — novo `exit_error.go` (tipo `exitError` + `ExitResolver.CodeFor`); os 14 handlers Cobra retornam erro tipado e `main` passa a ser o único chamador de `os.Exit` (códigos de saída preservados)
  - **R5.26:** prefixo `_` em 120 globais não-exportados de pacote (sentinelas `errX` isentos conforme exceção da regra)

### Correções
- **runtime/mcpserver:** sincroniza 6 goroutines de teste via canal de done, eliminando data race detectado por `go test -race`
- **integration/token-budget:** rebaseline consciente dos sentinelas de orçamento da skill `go-implementation` após crescimento legítimo de conteúdo (preserva o conteúdo da skill)
- **production-proof:** corrige os 14 achados da validação cross-tool na causa raiz — robustez production-ready sem falso positivo:
  - **install/verify↔upgrade:** `sourceSkills()` passa a exigir `SKILL.md` para contar um diretório como skill, reconciliando `verify` e `upgrade` (diretórios auxiliares como `tests/` deixam de ser contados)
  - **install/upgrade:** `upgrade` instala skills ausentes em vez de apenas reportar; `detect` reconhece monorepo via `--focus-paths`
  - **fs/symlink:** novo `symlink_guard.go` recusa escrita através de symlink externo por padrão (proteção contra travessia para repositórios fora do alvo)
  - **lint:** isenta versão de skills de terceiros da verificação de versão própria
  - **specdrift:** `check-spec-drift` aceita caminho de pasta de spec e limpa marcadores `PENDING-RUN`; `create-tasks` cobre RFs sem tarefa
  - **runtime:** mensagem de `permission_denied` aponta para `--access-mode full`; help de `--runtime` inclui `gemini`
- **skills(bubbletea):** remove 100% a skill `bubbletea` dos caminhos operacionais (diretório, `skills-lock.json`, referências em `README.md`/`CLAUDE.md`)

### Testes
- **config:** migra testes para `testify/suite` table-driven (`config_types_test.go`, `runtime_suite_test.go`); remove `config_test.go` e `runtime_test.go`
- **production-proof:** adiciona testes de regressão para os achados corrigidos (`install_test.go`, `upgrade_test.go`, `lint_test.go`, `detect_test.go`, `specdrift/sync_test.go`, `spec_drift_test.go`, `fs/fake_test.go`)

### Infraestrutura
- **mocks:** adiciona `mockery.yml`, `scripts/check-mocks.sh` e `scripts/normalize-mocks.sh`, além dos mocks gerados por pacote em `*/mocks/`

## 0.23.4 (2026-05-25)

### Bug Fixes
- **ci:** corrige pin npm quebrado do claude-agent-acp no acp-live (5a96f59)

### CI
- atualiza actions para runtime Node 24 (deprecacao Node 20) (fdb8326)

## 0.23.3 (2026-05-25)

### Bug Fixes
- **release:** propaga skill bumps para mirrors e destrava check-skills-sync (d14bb2f)

### Documentation
- **readme:** corrige falso positivo no playbook de atualizacao (71ff5ea)
- **readme:** adiciona playbook do ciclo de vida da governanca (df14163)
- **readme:** documenta atualizacao real do financialcontrol-api (707e292)

### Tests
- **client:** elimina flaky em RequestPermissionCancelsPrompt (803e9f1)

## 0.23.2 (2026-05-23)

### Bug Fixes
- **sync:** nao marcar skills/lib como read-only (quebrava o git) (7dbdc0b)

## 0.23.1 (2026-05-23)

### Refactoring
- **sdd:** migra root de artefatos de tasks/ para .specs/ (833bdd1)

### Chores
- **cleanup:** remove artefatos obsoletos e traces versionados (f503e3f)

## 0.23.0 (2026-05-23)

### Features
- **paridade:** suite RP-03 + gate CI de paridade + plano de sunset (task 8.0) (cc298a9)
- **paridade:** paridade cross-CLI e instalacao universal transparente (c10f798)
- **foundation:** implement portable foundation (RF-01..RF-19) (f152ead)
- **runtime:** add Gemini ACP runtime support (30cc945)
- **codex:** implementar F1-Codex via ACP nativo (ADR-013) (822ae74)
- **cli:** integrate Copilot ACP catalog and --agent flag in task-loop (a6f5501)
- **agents:** add declarative agent registry package and integration (3017077)
- **runtime:** add Copilot ACP spec and generalize newSpec/runner/probe (24e0081)
- **cli:** add --runtime/--activity-timeout/--quiet flags, telemetry fields and acp_live CI job (4aaab6f)
- **runtime:** add ACPRunner application service with integration tests (5f80510)
- **runtime:** add ACP client, fake server and human renderer (633fd26)
- **runtime/events:** translate acp.SessionUpdate to domain Event (120e3ab)
- **runtime/persistence:** add events.jsonl writer, tool_calls.md and report enricher (01e5c1e)
- **runtime/probe:** resolve claude-agent-acp launcher with npx fallback and cache (de60bb4)
- **runtime/specs:** add Claude spec with launcher fallbacks and sync script (c6f3aa2)
- **runtime:** add activity watchdog and error sentinels (7cc780f)
- **runtime/events:** add domain events model with VOs and state pattern (a01793f)

### Bug Fixes
- **fs:** copia/remove de assets read-only (source imutavel) + cobertura execute-all-tasks (45e429c)
- **acp:** validação real cross-CLI (claude/copilot/codex) — corrige hangs e bloqueios (b446cde)
- **governance:** spec_drift no-op quando tasks.md ausente (741d2f6)
- **tests:** resolve staticcheck lint failures on codex branch (0febf6a)
- **skills:** re-sync execute-all-tasks mirrors with canonical source (ced8e34)
- **agents:** resolve staticcheck findings in tests (058fc4e)

### Documentation
- **readme:** runtime ACP cross-CLI — processo + uso por ferramenta (claude/codex/copilot/gemini) (40f9537)
- **codex-acp:** add ADR-013 + PRD/TechSpec/Tasks for F1-Codex (708d00b)
- **research:** add Compozy adaptation analyses and governance drift (ca8d87e)
- **acp-runtime:** fix ADR-009 and ADR-010 labels in AGENTS.md table (dea381b)
- **acp-runtime:** finalize README, CHANGELOG, ADR-009/010 acceptance (72f841a)

### Chores
- **deps:** pin github.com/coder/acp-go-sdk v0.13.0 (ADR-009) (2132dae)
- **tasks:** mark task 5.0 done and add execution report (c339f93)

### Tests
- **acp-live:** smoke de handshake honesto e estrito com binário real (f2e7a92)
- **runtime:** estabilizar testes de goroutine-leak do watchdog (BF-06) (4d8c427)

### CI
- incluir tests/integration (e2e) no gate de merge (8feb873)

## [Unreleased] — Gemini CLI via ACP nativo (F0..F5-Gemini, ADR-015)

### Features

- **feat(gemini): F0 spec registration via gemini --acp (ADR-015)** — novo `internal/runtime/specs/gemini.go` com `Command="gemini"`, `FixedArgs=["--acp"]`, `BootstrapArgs=nil`; constantes `GeminiNpmPackage`, `GeminiNpmVersion="0.43.0"`, `DefaultGeminiModel="gemini-2.5-pro"`; registro em `runtimeACPCatalog`; mapeamento D-05 `AccessMode → --approval-mode`; probe estendido com T-13/T-14/T-15/T-16
- **feat(gemini): F1 paridade ACP E2E + wrapper deprecation** — habilita `--runtime acp --tool gemini` end-to-end; `internal/runtime/client/client.go` e `runner.go` (tool-agnosticos) recebem Gemini via catalog; sub-suite Gemini em `acp_integration_test.go`; smoke test `tests/integration/gemini_acp_smoke_test.go` (skipavel via `-short`); warning de deprecation `sync.Once` em `wrapper.go` quando modo legado invocado; `docs/cli-schema.json` enum atualizado
- **feat(gemini): F2 normalization cascata + MCP nested-agent** — `.agents/normalization-rules.yaml` e `internal/runtime/events/normalization-rules.yaml` ganham entrada `gemini: { inherit: common, overrides: {} }` (Gemini herda tabela comum, sem aliases especificos); `internal/runtime/events/normalize.go` resolve `inherit: common`; testes T-32/T-33 validam normalizacao; MCP nested-agent cascata automaticamente via `internal/runtime/mcpserver/` (tool-agnostico)
- **feat(gemini): F3 memory defaults generosos (250/400) + hooks cascata** — `cmd/ai_spec_harness/task_loop.go` aplica defaults Gemini-generosos quando `--tool gemini` e flags nao foram setadas explicitamente (workflow: 250 lin/20 KiB; task: 400 lin/32 KiB vs 150/200 defaults Claude); hooks in-process Go cascateiam automaticamente via `internal/runtime/hooks/dispatcher.go` (tool-agnostico); testes T-34/T-35
- **feat(gemini): F4 metricas Gemini-2026 (cache_read, effective_context, prompt_billed, thoughts)** — novo `internal/runtime/events/gemini_metrics.go` com `ExtractGeminiMetrics`/`LogGeminiMetrics`; `Summary` acumula `GeminiCacheReadTokens`, `GeminiEffectiveContextTokens`, `GeminiPromptTokensBilled`, `GeminiThoughtsTokens`; `execution_report.md` ganha secao "Metricas Gemini-2026"; telemetria opt-in via `GOVERNANCE_TELEMETRY=1` com entries `gemini.*`; extracao defensiva — ausencia nao bloqueia evidence; testes T-36/T-37/T-38
- **feat(gemini): F5 auto-review opt-in cascata** — flag `--auto-review` ja e tool-agnostica apos F1-Gemini; `runner_autoreview.go` cascatea sem codigo novo; testes T-39 validam `ReviewStatus=blocked` em issues `[HARD]`; INFO de custo amplificado documentado para Gemini com janela 1M+

### Chores

- **chore(deps): pin @google/gemini-cli@0.43.0** — constante `GeminiNpmVersion="0.43.0"` em `internal/runtime/specs/gemini.go`; validada via `npm view @google/gemini-cli version dist-tags` em 2026-05-22 (dist-tag `latest=0.43.0`); politica de pinning conforme ADR-015 D-02 (audit/ para atualizacoes futuras, nunca `@latest`)

## [Unreleased] — Claude-CLI 2026 waves F2–F5 (RF-01–RF-06, ADR-014)

### Features

- **feat(claude-2026): F2 — MCP nested + normalize** — adiciona `internal/runtime/mcpserver/` com tool `run_agent` via stdio MCP e `internal/runtime/events/normalize.go` com tabelas de alias por ferramenta; `events.jsonl` ganha `normalized_name`/`raw_name`; INV-30 e INV-31 adicionados a suite de paridade
- **feat(claude-2026): F3 — memory store 2-tier + hooks dispatcher** — adiciona `internal/runtime/memory/` (limites 150 linhas / 12 KB workflow + 200 linhas / 16 KB task) e `internal/runtime/hooks/` com 6 pontos canonicos (governance, token_budget, memory_persist); runner.go integra memoria e hooks via flags `--memory-workflow-limit-lines`, `--memory-task-limit-lines`, `--disable-hooks`
- **feat(claude-2026): F4 — metricas Claude-2026** — exporta `ExtractClaudeMetrics` e `LogClaudeMetrics` em `internal/runtime/events/metrics.go`; Summary acumula `cache_read_tokens`, `cache_creation_tokens`, `thinking_tokens`; telemetria opt-in via `GOVERNANCE_TELEMETRY=1`
- **feat(claude-2026): F5 — auto-review opt-in** — flag `--auto-review` (default-off) dispara nova ACPRunner apos sessao principal com skill `review` + git diff; resultado em `evidence/<task>/review.md`; issues `[HARD]` → `ReviewStatus=blocked`; recursao hard-bloqueada no child Job; hook `session.post_review` extensivel

## [Unreleased] — ACP runtime for Claude (RF-01–RF-16, ADR-009, ADR-010)

### Features

- **runtime/events:** add domain events model with VOs and state pattern (a01793f)
- **runtime:** add activity watchdog and error sentinels (7cc780f)
- **runtime/specs:** add Claude spec with launcher fallbacks and sync script (c6f3aa2)
- **runtime/probe:** resolve claude-agent-acp launcher with npx fallback and cache (de60bb4)
- **runtime/persistence:** add events.jsonl writer, tool_calls.md and report enricher (01e5c1e)
- **runtime/events:** translate acp.SessionUpdate to domain Event (120e3ab)
- **runtime:** add ACP client, fake server and human renderer (633fd26)
- **runtime:** add ACPRunner application service with integration tests (5f80510)
- **cli:** add --runtime/--activity-timeout/--quiet flags, telemetry fields and acp_live CI job (4aaab6f)

### Chores

- **deps:** pin github.com/coder/acp-go-sdk v0.13.0 (ADR-009) (2132dae)
- **tasks:** mark task 5.0 done and add execution report (c339f93)

### Documentation

- **acp-runtime:** add README section, accept ADR-009 and ADR-010, add CHANGELOG entry

## 0.22.4 (2026-05-19)

### Bug Fixes
- **ci:** restore shellcheck and skill budget gates (c46eca1)

### Documentation
- **readme:** document mandatory install and upgrade workflow (3357367)

## 0.22.3 (2026-05-19)

### Bug Fixes
- **setup-ai-spec:** baixar tarball com nome original para sha256sum -c verificar corretamente (8ffd97c)

## 0.22.2 (2026-05-19)

### Bug Fixes
- **setup-ai-spec:** usar github.token em vez de secrets.GITHUB_TOKEN na composite action (f66f93c)

## 0.22.1 (2026-05-18)

### Bug Fixes
- **setup-ai-spec,metrics:** autentica GitHub API, corrige brew link e forward slash em SkippedDirs (3748e75)

## [Unreleased]

### Bug Fixes
- **ci/lint:** instalar `shellcheck` explicitamente nos runners Linux e macOS antes do step de análise shell para eliminar falha por dependência ausente no job `Lint`
- **execute-all-tasks:** reduzir o texto base do `SKILL.md` para recolocar a skill abaixo do budget de 4.000 tokens exigido pelos testes de integração
- **action/setup-ai-spec:** autenticar chamada à GitHub API com `GITHUB_TOKEN` no step Linux para evitar rate limit (60 req/h → 5.000 req/h) em runners compartilhados
- **action/setup-ai-spec:** remover `|| true` do `brew link` no restore de cache macOS para expor falha total em vez de silenciar
- **metrics:** substituir `filepath.Join` por concatenação com forward slash em `SkippedDirs` para garantir separador consistente em qualquer SO

### Tests
- **metrics:** adicionar teste de regressão `TestGather_SkippedDirs_AlwaysForwardSlash` para invariante de forward slash em `SkippedDirs`
- **metrics:** refatorar condição de verificação JSON de comparação por nome para campo semântico `checkJSON` na tabela de testes
- **metrics:** registrar diretório pai explicitamente no setup de `TestGather_SkippedDirs_AlwaysForwardSlash`
- **integration:** restaurar a passagem de `TestTokenBudget_EmbeddedSkills` após o ajuste do budget da skill `execute-all-tasks`

## 0.22.0 (2026-05-18)

### Features
- **sdd-readiness:** fecha scorecard 100/100 com Action dual-channel, hook spec-drift e fix metrics (5153ad3)

### Documentation
- overhaul de governança e suporte ao SDD (9b94734)
- **sdd:** consolida estratégia de desenvolvimento de alta performance (8e60e21)

## 0.21.1 (2026-05-18)

### Bug Fixes
- **gemini,codex:** avoid command and agent-role conflicts (59a7e2e)

### Chores
- **sync:** propaga bumps de skills v0.21.0 para mirrors (d83d619)
- **sync:** propaga execute-all-tasks v1.6.0 para mirrors (c88b6dd)

## 0.21.0 (2026-05-16)

### Features
- **governance:** fecha lacunas multi-tool e distribui vendor .agents/lib (504ec70)

## 0.20.0 (2026-05-15)

### Features
- **execute-all-tasks:** orquestrador PRD com hooks de enforcement (1e35ecb)

## 0.19.1 (2026-05-08)

### Bug Fixes
- **integration:** atualiza skill budgets apos hardening de invocacao (9635f4d)

## 0.19.0 (2026-05-08)

### Features
- **config:** adiciona runtime config carregada de .claude/config.yaml (a5701da)
- **triggers:** sistema de triggers de revisao por linguagem (fbbbc38)

### Documentation
- adiciona matriz de degradacao, equivalencia runloop e tests scripts (9e037ee)

### Chores
- **skills:** melhora robustez de invocacao e validacao de schema (66d1eec)

## 0.18.1 (2026-05-08)

### Bug Fixes
- **taskloop:** serializa stdout+stderr concorrentes em liveOut do runCmd (246427c)
- **upgrade:** elimina falso-positivo SCHEMA DIVERGENTE em upgrade --check (9ab214b)
- **version:** elimina data race em Version mutado por testes paralelos (2a385cf)
- **fs:** CopyFile sobrescreve destino read-only existente (aadd61c)
- **fs:** WriteFile sobrescreve arquivos read-only existentes (ba2efba)

## 0.18.0 (2026-05-06)

### Features
- **taskloop:** enriquece revisao consolidada com contexto PRD/spec e isola diff do bugfix (38555c1)

## 0.17.0 (2026-05-05)

### Features
- **taskloop:** adiciona Service.RunLoop com selector, acceptance, evidence, final reviewer, bugfix loop e reservation planner (5467f6b)

## [Unreleased]

### Features
- **ci:** adiciona Action setup-ai-spec com canal dual brew/curl (refs tarefas 4.0/5.0/6.0)
- **hook:** adiciona bloco spec-drift ao pre-commit existente (refs tarefas 2.0/3.0)
- **adapters/task-executor:** os 4 generators (Claude/GitHub/Gemini/Codex) agora emitem o bloco YAML literal do contrato de retorno (`status`/`report_path`/`summary`) no agent file de `task-executor`. Fecha falso-positivo `failed: contract violation` em `execute-all-tasks` quando o subagent retornava prosa em vez de YAML. Auditoria A02. Regressão: `TestGenerate_executeTaskYAMLContract_allTools` cobre os 4 tools + guard `TestGenerate_nonExecuteTaskHasNoYAMLContract`.
- **install/hooks:** nova função `copyToolValidationHooks` distribui `validate-preload.sh` e `validate-governance.sh` para Codex e Copilot (paridade total com Claude/Gemini). Auditoria A01. Hooks novos: `.codex/hooks/validate-governance.sh`, `.github/hooks/validate-preload.sh`, `.github/hooks/validate-governance.sh` (padrão env-based, espelhando convenção Gemini). Regressão: `TestInstall_Copilot_CopiesValidationHooks`, `TestInstall_Codex_CopiesGovernanceHook`.
- **install/.agents/lib:** novo passo 2.6 distribui vendor canônico shell (`check-invocation-depth.sh`, `parse-hook-input.sh`) para `<projeto>/.agents/lib/`. Skills e hooks deixam de depender exclusivamente do mirror legado `scripts/lib/` — cascata `.agents/lib/` → `scripts/lib/` resolve no primeiro. Regressão: `TestInstall_DistributesAgentsLib`, `TestInstall_AgentsLib_AbsentSource_NoError`.
- **execute-all-tasks:** nova skill orquestradora de PRD que spawna um subagent fresh por tarefa para isolar contexto (≤100 tokens/task no orquestrador), respeita o DAG declarado em `tasks.md`, paraleliza waves marcadas como `Paralelizável` quando o tool ativo suporta nativamente, halt-first em status não-`done`, retomada idempotente. v1.5.1.
- **adapters:** `GenerateGeminiAgents` e `GenerateCodexAgents` geram `task-executor` agent files automaticamente para Gemini (`.gemini/agents/`) e Codex (`.codex/agents/`), espelhando o padrão de Claude/Copilot — paridade multi-tool dos 8 agents processuais.
- **hooks/orchestrator:** 4 hooks bash de enforcement programático (`post-execute-task.sh`, `pre-execute-all-tasks.sh`, `post-wave.sh`, `subagent-stop-wrapper.sh`) cobrindo F2 (evidence path), F13 (path absoluto), F17 (PREFLIGHT_DONE), F18 (cross-PRD spec-hash), F24 (escalation de remark crítico), F25 (checkpoint), F27 (ciclo cross-PRD), F29 (gaps numéricos), F31 (orchestrator checkpoint), F35 (git revert).
- **install/SubagentStop:** `defaultClaudeSettings()` registra `subagent-stop-wrapper.sh` como SubagentStop hook com matcher `task-executor` — auto-invocação no Claude Code sem dependência de o LLM lembrar.
- **create-tasks v1.5.0:** templates `tasks-template.md` e `task-template.md` ganham coluna/seção `Skills` mandatória; Etapa 4.1 faz descoberta agnóstica de skills processuais via match semântico contra `description` em `.agents/skills/`; Etapa 5.5 valida sincronia entre coluna e seção.
- **create-technical-specification v1.2.0:** adiciona seção `## Resolução de paths` espelhando o padrão das outras 4 skills core (`AI_TASKS_ROOT`/`AI_PRD_PREFIX` documentados) — fecha lacuna de paridade documental que poderia divergir paths em repos com `tasks_root` customizado. Auditoria A03.
- **create-technical-specification v1.1.0:** template injeta `<!-- spec-hash-prd: ... -->` no header (rastreabilidade do PRD consumido + drift detection downstream).
- **create-prd v1.2.1:** Etapa 1.4 detecta artefatos downstream (techspec, tasks, reports) e exige confirmação humana (best-effort enforcement) antes de editar PRD existente.
- **execute-task v1.4.0:** carrega skills processuais declaradas em `## Skills Necessárias` (descoberta agnóstica); Stage 4 escala `APPROVED_WITH_REMARKS` com tags `[critical|security|blocker|high]` para `BLOCKED`; Stage 5 escreve checkpoint YAML antes de mutar `tasks.md` (proteção contra crash mid-flight).
- **scripts/test-hooks:** test harness com 14 asserts empíricos validando F18/F27/F29/F35/F25 com fixtures sintéticos descartáveis.
- **scripts/sync-hooks + check-hooks-sync:** automação de sincronia canônico→mirrors (.agents, .gemini, .codex, .github, embedded) com pre-commit hook + step CI.
- **docs/execute-all-tasks-guide.md:** guia operacional de 570 linhas cobrindo prompts copy-paste por tool, anti-padrões, fluxos end-to-end e tabela comparativa de capacidades por tool.
- **taskloop:** orquestrador `Service.RunLoop` com `TaskSelector`, `AcceptanceGate`, `EvidenceRecorder`, `FinalReviewer`, `BugfixLoop` e `ReservationPlanner` (RF-01 a RF-08).
- **taskloop:** RF-08(a) — decisão `ActionImplement` em `APPROVED_WITH_REMARKS` reentra o `BugfixLoop` com limite rígido de 3 iterações e escalonamento humano.
- **taskloop:** telemetria `implement_promoted` por finding promovido a `Critical` via decisão `Implement`.
- **taskloop:** `buildFinalReviewInput` injeta contexto estruturado (PRD, TechSpec, Tasks, task executada, lote concluído) no payload do reviewer consolidado — reviewer passa a ter acesso aos artefatos de especificação sem precisar inferir caminhos.
- **taskloop/bugfix:** `splitReviewContext` / `attachReviewContext` preservam o cabeçalho de contexto entre iterações do `BugfixLoop`: o contexto segue para o reviewer mas não polui o prompt do `BugfixInvoker`.

### Bug Fixes
- **metrics:** ignora subdiretórios sem SKILL.md com aviso em vez de falhar (ref tarefa 1.0)
- **codex/adapters:** `task-executor.toml` agora usa `developer_instructions` no lugar de `instructions`, tanto no arquivo distribuído quanto no generator. Isso elimina o aviso `Ignoring malformed agent role definition` no Codex. Regressão: `TestGenerateCodexAgents_withSkill`.
- **gemini/adapters:** wrappers em `.gemini/commands/` passam a usar namespace `workspace.*.toml` e o generator remove automaticamente os nomes legados sem prefixo. Isso evita colisão com comandos nativos das skills no Gemini CLI. Regressão: `TestGenerateGemini_removesLegacyCommandName`, `TestE2E_GeminiInstall_TomlContent`.
- **execute-task:** Stage 2 não dispara mais `needs_input` em tarefas non-code (docs, configs YAML/JSON, SQL, shell, MD); detecção de linguagem agora condicional ao diff (F1).
- **execute-all-tasks:** validação 4-pass do YAML retornado por subagent (formato canônico, status canônico, evidência física via `realpath` + `[ -s ]`, consistência com `tasks.md`) — fecha alucinação de path e crash silencioso (F2/F13/F25).
- **execute-all-tasks:** wait-all-then-halt em waves paralelas previne race em `tasks.md` (F3); orientação explícita para subagents usarem `flock -x` ou rename atômico em writes concorrentes.
- **execute-all-tasks:** regex canônicos estritos para `Status`, `Dependências` (com suporte cross-PRD `<slug>/<id>`), `Paralelizável` — sem parsing tolerante (F7/F12/F20).
- **execute-all-tasks:** soft timeout (result-discard, não kill) configurável via `AI_TASK_TIMEOUT_SECONDS` ou comentário `<!-- task-timeout-seconds: N -->` no task file (F10/F21).
- **execute-task:** pre-flight gates condicionais via `AI_PREFLIGHT_DONE=1` evitam re-execução redundante de `ai-spec skills --verify` em cada subagent quando orquestrador já validou (F8/F17).
- **scripts/lib/check-invocation-depth.sh:** valida `AI_TOOL` ∈ `{claude, codex, gemini, copilot}`; valor inválido → unset (modo agnóstico) com warn (F30).
- **scripts/check-skills-sync.sh:** loga `SKIP:` explicitamente quando skill é ignorada por falta de SKILL.md no canônico (F34).
- **internal/install:** `copyOrchestratorHooks` preserva permissão `+x` via `os.Chmod 0o755` após `CopyFile` (F40).
- **taskloop/reviewer:** `parseVerdict` ancorado em linha dedicada (`Verdict:`/`Veredito:`) para evitar falso positivo de `BLOCKED`/`REJECTED` por palavras-chave em texto livre.
- **taskloop/reviewer:** `partitionDiff` subdivide seção única oversize em hunks `@@` repetindo o cabeçalho do arquivo; truncamento explícito quando hunk único excede `maxDiffPartitionSize`.
- **taskloop/runloop:** ramo `bfErr` não-exhausted agora emite `final_review_verdict` preservando paridade com `VerdictRejected`.
- **taskloop/reviewer:** prompt do reviewer consolidado inclui `BLOCKED` como veredito válido — impedia que o agente retornasse veredito de bloqueio em contextos sem condições de aprovar.
- **taskloop/bugfix:** `BugfixLoop.Run` passava o `reviewInput` completo (com cabeçalho de contexto) ao `BugfixInvoker`; agora extrai e entrega apenas o diff puro, evitando contaminação do prompt de correção.

### Documentation
- **execute-all-tasks v1.5.1:** corrige linha 131 da tabela "Mapeamento por Tool" — Copilot agora descreve discovery honestamente (`.github/skills/` criado por `ai-spec install` espelhando `.agents/skills/`; demais mirrors opcionais) em vez de prometer auto-descoberta de mirrors que não existem em fresh install. Auditoria A04.
- **docs/prompts/audit-workflow-skills.md:** novo prompt de auditoria rigorosa dos 5 skills core (PRD/TechSpec/Tasks/ExecuteTask/ExecuteAllTasks) cobrindo rastreabilidade, carregamento sob demanda, paridade multi-tool e robustez.
- **docs/execute-all-tasks-guide.md:** novo guia operacional de 570 linhas — pré-requisitos obrigatórios, anatomia dos inputs (tasks.md, task files), regras invioláveis, prompts copy-paste por tool (Claude/Codex/Gemini/Copilot), regras DTPS para paralelismo, fluxos end-to-end e tabela comparativa de capacidades por tool.

### Chores
- **scripts/sync-skills + check-skills-sync:** sync agora cobre 4 hooks orquestrador para 9 mirrors (`.claude/.codex/.gemini/.github/.agents` + 5 embedded), vendor `.agents/lib/` para `internal/embedded/assets/.agents/lib/`; check valida zero drift de hooks orquestrador (4×9), presença de hooks de validação por tool (2×8) e paridade dupla das libs (legacy + embedded).
- **install:** `BaseSkills` inclui `execute-all-tasks`; `installCodex`/`installGemini`/`installClaude`/`installCopilot` distribuem `task-executor` agent files + 4 hooks orquestrador para 5 dirs (.claude, .agents, .gemini, .codex, .github).
- **install:** `defaultClaudeSettings()` registra `SubagentStop` matcher `task-executor` para auto-invocação do wrapper de validação programática.
- **contextgen + upgrade:** allowlists internas incluem `execute-all-tasks` para distribuição correta no Codex `config.toml`.
- **scripts:** novos `check-skills-sync.sh`, `check-hooks-sync.sh`, `sync-hooks.sh`, `test-hooks.sh`, `git-hooks/pre-commit`; integrados a Makefile (`check-skills-sync`, `check-hooks-sync`, `test-hooks`) e workflow `.github/workflows/test.yml`.
- **.specs/prd-portability-parity + prd-taskloop-execution-validation:** migração para v1.4+ template (coluna `Skills` em tasks.md, seção `## Skills Necessárias` em todos os 17 task files).

## 0.16.0 (2026-04-25)

### Features
- **specdrift:** add sync-spec-hash command and fix taskloop pipeline blockers (b0f005f)

## 0.15.1 (2026-04-24)

### Bug Fixes
- **taskloop/parser:** detectar colunas Status e Dependências dinamicamente no header da tabela (3e1ec70)

## 0.15.0 (2026-04-24)

### Features
- **taskloop:** add bugfix phase, unlimited iterations and context-enriched prompts (75601c6)

## 0.14.3 (2026-04-24)

### Bug Fixes
- **taskloop/parser:** deduplica IDs em ParseTasksFile para evitar bloqueio de tasks elegiveis (6f92a95)

## 0.14.2 (2026-04-24)

### Bug Fixes
- **skills:** add missing version and effects.md to bubbletea skill (524df10)

## 0.14.1 (2026-04-24)

### Bug Fixes
- **taskloop:** corrige overwrite de postStatus e extrai classifyIterationOutcome (db28c51)

## 0.14.0 (2026-04-23)

### Features
- **taskloop:** enforce isolation and resume in-progress tasks (ccb8639)

## 0.13.0 (2026-04-22)

### Features
- **cli:** add skill-bump command, version --skills flag and skill_versions manifest field (ac90988)

## [Unreleased]

### Bug Fixes
- **taskloop:** corrige sobrescrita incondicional de `postStatus` pelo `tasks.md` — o fallback para `tasks.md` agora so ocorre quando o task file nao atualizou o status (`postStatus == preStatus`), evitando que tarefas concluidas corretamente sejam marcadas como "status inalterado" e puladas
- **taskloop/parser:** `ParseTasksFile` agora deduplica entradas por ID, mantendo apenas a primeira ocorrencia — tabelas auxiliares (ex: "Cobertura de Requisitos") com os mesmos IDs numericos corrompiam o `statusMap` e bloqueavam tasks elegiveis (`pending` com deps satisfeitas) impedindo o `task-loop` de encontrar trabalho executavel
- **taskloop/parser:** `ParseTasksFile` detecta dinamicamente os indices das colunas Status e Dependencias a partir do header da tabela markdown — tabelas com coluna Arquivo entre Titulo e Status deslocavam os indices fixos, fazendo o parser ler o nome do arquivo como status e classificar todas as tasks como inelegiveis
- **taskloop/isolation:** `extractTaskRows` ignora silenciosamente linhas com menos de 5 colunas em vez de retornar erro — tabelas auxiliares como "Cobertura de Requisitos" que compartilham IDs numericos bloqueavam o snapshot de isolamento antes de cada iteracao do task-loop
- **taskloop/agent:** `detectReferences` nao identifica mais projetos Go como Python — o padrao `"pip"` foi substituido por `"pip install"`, `"pip3"` e outros sufixos especificos para evitar falso positivo com a palavra `"pipeline"`

### Refactor
- **taskloop:** extrai `classifyIterationOutcome` — funcao pura sem parametro de ferramenta que centraliza a logica de decisao de resultado de iteracao (skip, abort, note, runReviewer), tornando-a isoladamente testavel
- **taskloop/agent:** `BuildPrompt` migrado para template `go:embed` (`executor_template.tmpl`) com criterios de execucao nao-negociaveis embutidos; `BuildPromptContext` extrai o sumario de arquitetura da `techspec.md` e detecta automaticamente as referencias relevantes a carregar (`go-implementation`, `ddd`, `security`, `tests`)
- **taskloop/reviewer:** `review_template.tmpl` ampliado com campos `CompletedTasks` e `RiskAreas`, focos de revisao estruturados (corretude, regressao, seguranca, testes, divida tecnica) e formato de saida esperada com veredicto final; `detectRiskAreas` detecta automaticamente areas de risco (performance, seguranca, contratos, concorrencia, persistencia) a partir da techspec e do diff
- **taskloop/reviewer:** `BuildBugfixPrompt` construido a partir do template `go:embed` (`bugfix_template.tmpl`) com placeholders para achados da revisao, diff original e contexto da task
- **version:** `ResolveFromExecutable()` resolve o arquivo `VERSION` adjacente ao binario seguindo symlinks, substituindo a leitura via `ReadVersionFile(sourceDir)` no `install` e `upgrade`
- **skills:** `isValidSemver` renomeado para `IsValidSemver` (exportado) para reutilizacao no pacote `upgrade`

### Tests
- **taskloop:** adiciona matriz de paridade semantica (10 cenarios x 4 ferramentas = 40 sub-testes), testes de isolamento de sessao entre tasks, testes de determinismo de prompt, testes de integracao com mock binaries e testes de reproducao do bug de status

### Features
- **specdrift:** novo comando `sync-spec-hash <tasks.md>` que recalcula os SHA-256 de `prd.md` e `techspec.md` e atualiza ou insere os comentarios `<!-- spec-hash-{label}: ... -->` em `tasks.md`; idempotente — nao reescreve o arquivo quando os hashes ja estao corretos
- **templates:** `prd-template.md` passa a incluir secao `## Requisitos Funcionais` com formato `RF-nn` obrigatorio; `tasks-template.md` passa a incluir comentarios `spec-hash` e secao `## Cobertura de Requisitos` — ambos atualizados nos quatro locais de copia (`.agents/`, `.claude/`, `.github/`, `internal/embedded/`)
- **taskloop:** retoma tasks com status efetivo `in_progress` antes de abrir novas pendentes, reconciliando `tasks.md` com o status real do arquivo individual da task
- **taskloop:** adiciona guardrails de isolamento para executor e reviewer; o loop aborta e restaura snapshot quando o agente altera outras rows de `tasks.md`, outros arquivos de task ou arquivos protegidos do PRD
- **taskloop:** suporta `MaxIterations == 0` como modo de iteracoes ilimitadas; o log de inicio exibe "ilimitado" no lugar do numero e o loop continua ate nao haver tasks pendentes
- **taskloop:** fase de bugfix automatica pos-revisao — quando o reviewer retorna exit code != 0, o executor e reinvocado via `invokeBugfix` com prompt de bugfix estruturado; a fase tem guardrails de isolamento proprios e o resultado e registrado em `BugfixResult` na iteracao
- **taskloop/report:** `BugfixResult` adicionado a `IterationResult`; o report avancado exibe sub-linha de duracao/exit-code do bugfix e secao detalhada com output e nota de isolamento
- **skills:** adiciona a skill externa `bubbletea` ao `skills-lock.json`
- **skill-bump:** novo comando `skill-bump <path>` que detecta skills alteradas desde a ultima tag via `git diff` e atualiza automaticamente o campo `version` no frontmatter `SKILL.md`; suporta `--dry-run` para inspecionar sem alterar arquivos
- **version:** flag `--skills` exibe versoes das skills; aceita `embedded`, `installed` ou sem valor (ambos) — ex.: `ai-spec-harness version --skills`
- **manifest:** campo `skill_versions` registra o mapa `skill → version` no manifesto durante `install` e `upgrade`
- **upgrade:** modo `--check` exibe divergencia entre a versao do CLI e a versao registrada no manifesto quando diferem

### Documentation
- remove arquivos de prompt obsoletos de `docs/prompt/` e `docs/prompts/`; adiciona `docs/prompts/taskloop-paridade-multiagente.md`

## 0.12.0 (2026-04-22)

### Features
- **taskloop:** add advanced mode with independent executor and reviewer profiles (fdbfee0)

## 0.11.2 (2026-04-21)

### Bug Fixes
- **taskloop:** support task-X.Y- filename convention in matchesTaskPrefix (39d1f2d)

## [0.11.2] - 2026-04-21

### Fixed
- **taskloop:** `matchesTaskPrefix` agora reconhece a convenção `task-X.Y-descricao.md` além das convenções `X-` e `X.Y-`, evitando que tasks com esse padrão sejam ignoradas e bloqueiem toda a cadeia de dependências no task-loop

## 0.11.1 (2026-04-21)

### Bug Fixes
- **taskloop:** isolate Unix syscalls behind build tags for Windows cross-compilation (1a4cfec)

## [0.11.1] - 2026-04-21

### Fixed
- **taskloop:** isolate Unix-only syscall fields (`Setpgid`, `syscall.Kill`) behind `//go:build !windows` to fix cross-compilation failure for `windows_amd64` and `windows_arm64` targets

## 0.11.0 (2026-04-21)

### Features
- **taskloop:** add report summary, update agent flags and inject AGENTS.md in prompt (43d50d0)

### Documentation
- **readme:** add task execution alternatives without task-loop (1ff3f75)
- **readme:** explain task-loop as automatic equivalent of per-session discipline (005c373)
- **readme:** add table of contents with anchor links for all sections (ff32669)
- **readme:** add execute-task per-session discipline with context reset and fidelity rules (adcacaa)
- **readme:** add mandatory skills usage guide with contracts and checklists (e104cab)

## [0.11.0] - 2026-04-21

### Added
- **taskloop:** seção "Resumo" no relatório com contagem de tasks por status (sucesso, puladas, falhadas)
- **taskloop:** instrução explícita de leitura de AGENTS.md no prompt do agente (RF-04, contrato de carga base)
- **taskloop:** flag `--bare` no Claude para pular carregamento automático de CLAUDE.md
- **taskloop:** atualizar flags do Codex para `exec --dangerously-bypass-approvals-and-sandbox`
- **taskloop:** atualizar flags do Copilot para `--autopilot --yolo`

### Fixed
- **taskloop:** guard clause em `ReadTaskFileStatus` para campos de status vazios ou com whitespace

### Changed
- **taskloop:** substituir `os.ReadFile` e `os.Stat` por `fs.FileSystem` em parser e taskloop para testabilidade com FakeFileSystem

### Documentation
- **readme:** adicionar sumário, guia mandatório de skills, disciplina execute-task por sessão e alternativas de execução sem task-loop
- **prompts:** adicionar prompt de execução sequencial automatizada de tasks

## 0.10.4 (2026-04-20)

### Bug Fixes
- **taskloop:** kill process group on cancel and update codex flags (39505f4)

### Documentation
- **readme:** add execute-task and task-loop usage examples (0bb6688)
- **readme:** reorganize sections and expand usage guides (9b4016b)

## 0.10.3 (2026-04-20)

### Bug Fixes
- **brew:** use Ruby chmod method to satisfy Homebrew FormulaAudit (d500dfe)

## 0.10.2 (2026-04-20)

### Bug Fixes
- **brew:** add chmod around xattr to fix post_install permission denied (a52e58a)

## 0.10.1 (2026-04-20)

### Bug Fixes
- **ci:** corrigir smoke test com comando inexistente detect (2c9aeb6)

## 0.10.0 (2026-04-20)

### Features
- **cli:** adicionar skills check, telemetria trend e lint strict — v0.10.0 (3ef9792)

### Documentation
- **readme:** documentar todas as opcoes de correcao do Gatekeeper do macOS (30a3785)
- **readme:** documentar workaround do Gatekeeper do macOS para binario nao assinado (b9d1823)

## 0.10.0 (2026-04-20)

### Added

- **skills check:** novo comando para verificar versões de skills externas contra `skills-lock.json`; detecta upgrades compatíveis (minor/patch) e potenciais quebras de interface (major bump) (`cmd/ai_spec_harness/skills.go`, `internal/skillscheck/`)
- **inspect --brief / --complexity:** exibe referências carregadas por skill por nível de complexidade via `contextgen.Loader` (`cmd/ai_spec_harness/inspect.go`)
- **telemetry report --trend:** evolução semanal de invocações nas últimas 4 semanas em formato texto ou JSON (`internal/telemetry/trend.go`)
- **telemetry report --budget-check:** verifica budget de invocações por skill com saída em texto ou JSON
- **telemetry report --top-skills:** ranking de skills por volume de uso
- **lint --strict:** trata invariantes `BestEffort` de paridade como erros; sem a flag, exibe apenas avisos (`cmd/ai_spec_harness/lint.go`)
- **contextgen.Loader:** carrega referências por skill com suporte a `brief` e `complexity` (`internal/contextgen/loader.go`)
- **requireFlag:** validação de flags obrigatórias com mensagem amigável em PT-BR e exemplo de uso real (`cmd/ai_spec_harness/flags.go`)
- **CodeQL:** workflow de análise de segurança estática adicionado ao CI (`.github/workflows/codeql.yml`)
- **hook validate-token-budget:** verificação de budget de tokens integrada ao Claude Code (`.claude/hooks/validate-token-budget.sh`)
- **docs/troubleshooting.md:** guia de 12 problemas comuns com sintoma, causa, solução e verificação
- **docs/skill-schema.json:** schema JSON para validação de `SKILL.md`

### Changed

- **lint:** detecta tools e langs automaticamente e verifica invariantes `BestEffort` de paridade em toda execução
- **inspect:** integra `contextgen.Loader` para exibir referências por skill nos modos `--brief` e `--complexity`
- **telemetry report:** expandido com três novos modos de saída (`--trend`, `--budget-check`, `--top-skills`)
- **test.yml:** pipeline de CI expandido com novos targets, cobertura por pacote e testes de integração
- **Makefile:** novos targets para fuzz, cobertura e validação de schema
- **skills references:** atualizações em `agent-governance`, `go-implementation` e `object-calisthenics-go`

### Tests

- Fuzz tests adicionados em `internal/config`, `internal/detect` e `internal/manifest`
- Testes de integração para budget de tokens e skills externas (`internal/integration/token_budget_skill_test.go`)
- Contrato CLI expandido com novos subcomandos (`cmd/ai_spec_harness/cli_contract_test.go`)
- Novos testes: `flags_test.go`, `validation_test.go`, `skillscheck_test.go`, `trend_test.go`, `loader_test.go`

---

## 0.9.2 (2026-04-20)

### Bug Fixes
- **brew:** remover quarentena do Gatekeeper via post_install na Formula (3e21aab)

## 0.9.1 (2026-04-20)

### Bug Fixes
- **ci:** corrigir dirty state do GoReleaser por semver_output.txt no workspace (7540c7f)

### Documentation
- **readme:** atualizar instalacao para Formula (brew install) (32206f2)

## [Unreleased]

### Added

- **taskloop — modo avancado (executor + reviewer independentes):** flags `--executor-tool`, `--executor-model`, `--reviewer-tool` e `--reviewer-model` permitem configurar agentes distintos para execucao e revisao; modo simples via `--tool` permanece inalterado e retro-compativel
- **taskloop — `ExecutionProfile` (Value Object):** representa a configuracao de um papel (executor ou reviewer) com validacao fail-fast no construtor; campos `role`, `tool`, `provider` e `model`
- **taskloop — `CompatibilityTable`:** cataloga combinacoes ferramenta+modelo reconhecidas (Claude, Codex, Gemini, Copilot); flag `--allow-unknown-model` para aceitar combinacoes fora do catalogo sem erro
- **taskloop — `BuildReviewPrompt` com `go:embed`:** gera prompt de revisao a partir de template embutido (`review_template.tmpl`) ou de template customizado via `--reviewer-prompt-template`
- **taskloop — deteccao de auth error e guidance por ferramenta:** `isAuthError` detecta padroes de falha de autenticacao no output do agente; `authGuidance` retorna instrucao especifica por ferramenta; `warnClaudeAuth` alerta antes do loop quando ANTHROPIC_API_KEY esta ausente
- **taskloop — `LiveOutputSetter` interface:** permite injetar um `io.Writer` para streaming de output do agente em tempo real; usado internamente por `runCmd` para tee do stdout
- **taskloop — fallback model nativo por papel:** flags `--executor-fallback-model` e `--reviewer-fallback-model` passam `--fallback-model` ao claudeInvoker; outros invokers ignoram silenciosamente
- **taskloop — fallback tool pre-loop:** flag `--fallback-tool` para validacao de disponibilidade antes do inicio do loop principal
- **taskloop — relatorio modo avancado:** `ReviewResult` armazena resultado da revisao; `Report` ganha campos `Mode`, `ExecutorProfile` e `ReviewerProfile`; renderizacao separada para modo simples e avancado com coluna Papel
- **taskloop — dry-run avancado:** exibe modo, perfis resolvidos com status de compatibilidade, template de revisao e preview do prompt para a primeira task elegivel
- **metrics — modo `brief` no gather:** estimativa de tokens por referencia de skill em modo resumido (150 tokens fixos por entrada) em vez de contagem real do arquivo completo
- **docs — `docs/task-loop-reference.md`:** guia consolidado de flags, heuristicas e alternativas do task-loop (referenciado no README)
- **docs — `docs/skills-usage-guide.md`:** guia de uso das skills com contratos de entrada, prompts mandatorios e criterios de aceite
- **docs — prompts de execucao sequencial:** prompts em `docs/prompts/` para execucao automatizada de tasks com modelos por papel
- Nova skill `finalize-changelog-readme-push` para consolidar atualizacao de `CHANGELOG.md`, revisao de `README.md`, `git add .`, commit semantico e `git push` com guardrails de confirmacao

### Fixed

- **ci:** corrigir dirty state do GoReleaser causado por `semver_output.txt` não rastreado no workspace git

### Tests

- Expandir cobertura de testes unitários em `detect`, `install`, `metrics`, `scaffold`, `uninstall`, `upgrade` e `wrapper`
- Adicionar testes de integração para skills externas e orçamento de tokens
- Adicionar benchmarks para `metrics`, `parity` e `skills/schema`
- Adicionar contrato CLI em `cmd/ai_spec_harness/cli_contract_test.go`

### CI

- Atualizar `test.yml` com melhorias no pipeline de testes
- Adicionar script `scripts/check-package-coverage.sh` para verificação de cobertura por pacote

### Docs

- Adicionar ADR-006: telemetria opt-in com append-only log
- Adicionar ADR-007: workaround stateless para Copilot CLI
- Adicionar ADR-008: paridade multi-tool com 29 invariantes semânticas em 3 níveis
- Adicionar `.aiignore` e `.claudeignore` para controle de contexto dos agentes
- Atualizar governança operacional em `AGENTS.md`, `CLAUDE.md`, `CODEX.md`, `COPILOT.md` e `GEMINI.md`
- Expandir `docs/cli-schema.json` com novos comandos
- Atualizar `Makefile` com novos targets

## 0.9.0 (2026-04-20)

### Features
- **release:** migrar Homebrew de Cask para Formula (c77f61a)

### CI
- **release:** adicionar step para corrigir ordem de stanzas no Homebrew Cask após GoReleaser (9143e49)

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.8.0] - 2026-04-20

### Breaking Changes

- Resetar repositório para catálogo de skills (`65f25ae`)

### Added

- Implementar CLI Go de governança de IA (`7083197`)
- Adicionar distribuição via Homebrew e expandir CLI de governança (`937c15b`)
- Adicionar assets embutidos e validações de spec (`aa62ac9`)
- Separar modelo de custo em três eixos e adicionar gate de regressão — `metrics` (`c004882`)
- Adicionar workflow de dry-run e comandos semver-next e changelog — `release` (`1cd2b1a`)
- Adicionar pacote para resolver git refs em diretório temporário — `gitref` (`e8b29dc`)
- Adicionar flag `--ref` para install e upgrade a partir de git ref (`45e0ce3`)
- Adicionar scoring por focus-paths e suporte a monorepo Python — `detect` (`786d145`)
- Adicionar wrapper e verificação de pré-requisitos de skills (`61b18ae`)
- Expand skills baseline and document task loop flow — `governance` (`6237f5f`)
- Adicionar feedback loop de telemetria, spec-driven e governança multi-agente (`4d7a780`)
- Adicionar Codex, Copilot e parser de telemetria (`66dd041`)

### Fixed

- Usar /tmp para semver_output e ajustar validação de working tree — `release-dry-run` (`e5c3529`)
- Alinhar flags de autonomia total para todas as ferramentas — `taskloop` (`9517323`)
- Corrigir bad substitution ao interpolar mensagem de commit no bash — `ci` (`3c3b0d9`)
