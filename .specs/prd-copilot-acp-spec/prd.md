# Documento de Requisitos do Produto (PRD) — Copilot CLI via ACP Nativo

<!-- spec-version: 1 -->

> **Insumo de pesquisa**: [docs/research/compozy-adaptation-copilot-2026.md](../../docs/research/compozy-adaptation-copilot-2026.md)
> **ADR substitutiva**: [.specs/adr/012-copilot-cli-acp-native.md](../adr/012-copilot-cli-acp-native.md) (substitui ADR-007)
> **Fase do roadmap**: 1 de 4 (adaptação Compozy 2026 — variante Copilot)
> **Data**: 2026-05-21

## Visão Geral

O `ai-spec-harness` integra Claude Code via protocolo ACP nativo (ADR-009) com persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`), watchdog de inatividade e telemetria opt-in (ADR-006). GitHub Copilot CLI, por contraste, é hoje invocado em modo stateless (`copilot --autopilot --yolo -p <prompt>` em `internal/taskloop/agent.go:381-388`) sem nenhuma dessas garantias — uma limitação registrada em ADR-007 quando o Copilot CLI não expunha modo servidor.

Em 2026 o Copilot CLI passou a expor servidor ACP nativo (`copilot --acp`), conforme demonstrado no `compozy/compozy` em `internal/core/agent/registry_specs.go:222-242`. A premissa técnica de ADR-007 deixou de valer. Esta funcionalidade introduz suporte oficial ao **Copilot CLI como runtime ACP** no harness, atingindo paridade observacional com Claude e formalmente substituindo ADR-007 via [ADR-012](../adr/012-copilot-cli-acp-native.md).

Beneficia desenvolvedores e operadores que usam Copilot CLI para executar tasks e hoje perdem telemetria, persistência forense e watchdog. Também desbloqueia ADR-008 (paridade multi-tool) ao remover a exceção de "BestEffort" que ADR-007 carregava para Copilot. Não introduz novo runtime nem nova camada — reaproveita 100% do stack ACP construído para Claude (`ACPRunner`, `acpClient`, `persistence`, `events`, `watchdog`).

## Objetivos

- **OB-01**: Permitir executar tasks com Copilot CLI via protocolo ACP nativo usando `--tool copilot --runtime acp`.
- **OB-02**: Atingir paridade observacional Copilot ↔ Claude: `events.jsonl`, `tool_calls.md`, `execution_report.md` e telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) cobrem invocações Copilot com os mesmos campos e granularidade que cobrem Claude.
- **OB-03**: Generalizar o runtime layer para deixar de ser Claude-centric: `runtime_init` event e `probe` error template passam a usar metadata do `Spec` resolvido, não constantes Claude hardcoded.
- **OB-04**: Preservar 100% retrocompatibilidade. Invocações sem `--runtime=acp` continuam usando `copilotInvoker` CLI legado em `internal/taskloop/agent.go:381-388`; aviso de depreciação informa migração.
- **OB-05**: Preservar invariantes forenses, watchdog (`ActivityWatchdog` com `CancelCause`) e pinning de SDK (ADR-009) sem regressão para Claude.
- **OB-06**: Substituir formalmente ADR-007 via ADR-012, atualizar `COPILOT.md` e marcar invariantes Copilot em ADR-008 (paridade) como **suportadas** em vez de `BestEffort`.

**Métricas mensuráveis**:
- 100% dos testes de regressão Claude permanecem verdes após generalização do runtime.
- Matriz de teste ACP (`internal/runtime/acp_integration_test.go`) cobre Copilot com ≥ 90% dos casos cobertos para Claude (drop only para casos `--bypass-permissions` específicos do Claude).
- Probe Copilot resolve em < 200ms p95 (binary path) e < 2s p95 (fallback npx) em ambiente padrão.
- Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go` (estes três módulos preservam invariantes existentes).
- Documentação de versão mínima do `copilot` CLI que expõe `--acp` registrada na techspec e em `COPILOT.md`.

## Histórias de Usuário

- **HU-01**: Como **operador da CLI**, quero invocar `ai-spec-harness task-loop --tool copilot --runtime acp .specs/prd-minha-feature` e ver o harness conectar via ACP, produzir `events.jsonl` linha-a-linha e gerar `execution_report.md` no padrão Claude.
- **HU-02**: Como **operador**, quero que quando o binário `copilot` não está no PATH, o harness tente `npx --yes @github/copilot@<pin> --acp` como fallback automático sem mudança de comando.
- **HU-03**: Como **operador**, quero ver erro claro quando nem `copilot` nem `npx` estão disponíveis, com os três remédios (instalar `copilot`; instalar pacote npm; usar `--runtime=legacy`).
- **HU-04**: Como **operador legado**, quero que `ai-spec-harness task-loop --tool copilot .specs/...` (sem `--runtime=acp`) continue funcionando exatamente como hoje (via `copilotInvoker`), com aviso de depreciação registrado em log.
- **HU-05**: Como **mantenedor**, quero que telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) registre invocações Copilot ACP com `tool=copilot`, `launcher=binary|npx`, `npm_version` e `sdk_version` reais — não constantes Claude.
- **HU-06**: Como **mantenedor**, quero que ADR-007 seja marcada como substituída por ADR-012 e que `COPILOT.md` documente o caminho ACP como recomendado.
- **HU-07**: Como **operador**, quero que `ActivityWatchdog` cancele sessões Copilot inativas com o mesmo timeout configurável que cancela sessões Claude.
- **HU-08**: Como **operador**, quero suporte transparente a tabela de compatibilidade Copilot (`internal/taskloop/compatibility.go`) — modelos Copilot continuam validados pelo mesmo mecanismo de `--allow-unknown-model`.

## Funcionalidades Core

### F-01: Spec Copilot ACP em catálogo
Novo construtor `specs.Copilot()` em `internal/runtime/specs/copilot.go` retornando uma `Spec` com `Command="copilot"`, `FixedArgs=["--acp"]`, `Fallbacks=[npx --yes @github/copilot@<pin> --acp]`. Constantes `CopilotNpmPackage`, `CopilotNpmVersion`, `CopilotSDKVersion` (sincronizada com `go.mod` via processo de ADR-009), `CopilotMinCLIVersion` (documentada).

### F-02: Generalização do `runtime_init` event
`internal/runtime/runner.go:113-120` deixa de hardcodar `specs.ClaudeSDKVersion`/`ClaudeNpmVersion`. Métodos novos no value object `Spec` (`SDKVersion() string`, `NPMVersion() string`, `NPMPackage() string`) ou metadata no `Launcher` resolvido permitem que `runtime_init` carregue versões do runtime efetivamente em uso (Claude ou Copilot).

### F-03: Generalização do `probe` error template
`internal/runtime/probe/probe.go:69-82` deixa de assumir `specs.ClaudeNpmPackage`. Template de erro consome metadata do `Spec` recebido; ID do ADR referenciado no remédio também passa a ser parametrizado (ADR-009 para Claude, ADR-012 para Copilot).

### F-04: Roteamento ACP por tool no CLI
`cmd/ai_spec_harness/task_loop.go:77` deixa de bloquear `--tool != claude` quando `--runtime=acp`. Tabela `runtimeACPCatalog map[string]func() specs.Spec` (registro `"claude" → specs.Claude`, `"copilot" → specs.Copilot`) determina quais tools podem usar ACP. Erro claro quando tool não está no catálogo ACP.

### F-05: Wiring de Copilot no taskloop service
`internal/taskloop/taskloop.go` (`Service.Execute`) resolve Spec via `runtimeACPCatalog` quando `Runtime == "acp"` e `Tool == "copilot"`. Reusa `ACPRunner` existente sem nova camada.

### F-06: Matriz de teste ACP estendida
`internal/runtime/acp_integration_test.go` ganha matriz Copilot reusando a fake ACP server existente em `internal/runtime/client/client_test.go`. Casos cobertos: open OK, prompt, tool calls, agent message, completion, cancel por timeout, erro de launcher unavailable, fallback npx.

### F-07: Aviso de depreciação no caminho legado
`copilotInvoker` em `internal/taskloop/agent.go:381-388` ganha log de warning na primeira invocação: "Copilot CLI legado em uso; migrar para --runtime=acp (ver ADR-012)". Caminho legado é mantido por uma versão (a definir na techspec).

### F-08: Reescrita de `COPILOT.md`
`COPILOT.md` raiz é reescrito: seção "Modo Recomendado (2026): Copilot via ACP" é a primeira; seção "Modo Legado" é marcada como deprecated com timeline de remoção. Referências cruzadas para ADR-012 e ADR-007.

### F-09: Telemetria enriquecida
Evento `runtime_init` no telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) ganha tool real (`tool=copilot` quando aplicável). Pipeline de relatório (`ai-spec-harness telemetry report`) agrega por tool.

## Requisitos Funcionais

- **RF-01**: Criar `specs.Copilot()` retornando `Spec` com `ID="copilot"`, `DisplayName="GitHub Copilot CLI (ACP)"`, `Command="copilot"`, `FixedArgs=["--acp"]`.
- **RF-02**: Spec Copilot expõe pelo menos um `FallbackLauncher` com `Command="npx"`, `FixedArgs=["--yes", CopilotNpmPackage+"@"+CopilotNpmVersion, "--acp"]`.
- **RF-03**: Constantes `CopilotNpmPackage`, `CopilotNpmVersion`, `CopilotSDKVersion`, `CopilotMinCLIVersion` declaradas em `internal/runtime/specs/copilot.go` com documentação de política de atualização equivalente à de Claude (ADR-009).
- **RF-04**: `runtime_init` event em `internal/runtime/runner.go` deixa de hardcodar `specs.ClaudeSDKVersion`/`ClaudeNpmVersion`. Versões passam a vir do `Spec` resolvido.
- **RF-05**: `probe` error template em `internal/runtime/probe/probe.go` deixa de assumir `specs.ClaudeNpmPackage`. Template é parametrizado pelo `Spec` recebido. Referência ao ADR no remédio também é parametrizada.
- **RF-06**: `cmd/ai_spec_harness/task_loop.go` aceita `--tool copilot --runtime acp`. Tabela `runtimeACPCatalog` é a fonte de verdade para quais tools podem usar ACP nesta versão.
- **RF-07**: Quando `--runtime acp` é usado com tool fora do catálogo ACP, harness retorna erro claro listando tools suportadas em ACP nesta versão.
- **RF-08**: `internal/taskloop/taskloop.go` (`Service.Execute`) roteia Copilot via `ACPRunner` quando `Runtime == "acp"`; via `copilotInvoker` quando `Runtime == "legacy"` (default).
- **RF-09**: `copilotInvoker` legado emite log WARNING na primeira invocação por execução do processo (não a cada task) anunciando depreciação e apontando para ADR-012.
- **RF-10**: `internal/runtime/acp_integration_test.go` ganha sub-suite Copilot reusando fake ACP server. Cobertura mínima: open, prompt, ≥ 2 tipos de tool call, agent message, completion, cancel por `ActivityWatchdog`.
- **RF-11**: `internal/runtime/specs/copilot_test.go` cobre defaults da Spec, ordenação de fallback, política de versionamento (constantes não-vazias, versão Copilot pinada SemVer-like).
- **RF-12**: Persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e `ActivityWatchdog` permanecem inalterados em comportamento e código fonte para o caso Claude. Diff zero em `internal/runtime/persistence/` e `internal/runtime/watchdog.go`.
- **RF-13**: Reescrever `COPILOT.md` raiz com seção "Modo Recomendado (2026): Copilot via ACP" primeira e "Modo Legado" deprecada.
- **RF-14**: Atualizar `AGENTS.md` adicionando linha na tabela de ADRs (ADR-012) e marcando ADR-007 como "substituída por ADR-012".
- **RF-15**: Atualizar `docs/cli-schema.json` se o enum de `--tool` ou `--runtime` for tocado. Caso contrário, validar que schema atual já cobre o caso.
- **RF-16**: Atualizar `docs/telemetry-feedback-cycle.md` documentando que invariantes Copilot ACP cobrem os mesmos kinds que Claude.
- **RF-17**: Tabela `internal/taskloop/compatibility.go:CompatibilityTable` é a fonte de verdade para validação de `--model` quando `--tool=copilot`. Mesma semântica de `--allow-unknown-model` aplica-se.
- **RF-18**: Cache de probe em `internal/runtime/probe/probe.go` continua keyed por `spec.ID` — múltiplas invocações Copilot em uma sessão CLI reutilizam o launcher resolvido sem re-probe.
- **RF-19**: ADR-007 (`docs/adr/007-copilot-cli-stateless-workaround.md`) ganha cabeçalho "**Status:** Substituída por ADR-012" sem deletar o conteúdo histórico.
- **RF-20**: ADR-012 (já escrito em `.specs/adr/012-copilot-cli-acp-native.md`) é referenciada no índice em `AGENTS.md` e em `COPILOT.md`.
- **RF-21**: Telemetria opt-in registra `runtime_init` com `tool=copilot` quando aplicável; campo `launcher` distingue `binary` de `npx`.

## Experiência do Usuário

UX é primariamente backend/CLI; materializa-se em três pontos:

1. **CLI — comando recomendado**:
   ```
   ai-spec-harness task-loop \
     --tool copilot \
     --runtime acp \
     .specs/prd-minha-feature
   ```
   Comportamento idêntico ao Claude: stream humano em stdout (suprimível via `--quiet`), `events.jsonl`/`tool_calls.md`/`execution_report.md` em `audit/` ou `evidenceDir` configurado.

2. **CLI — erro de launcher unavailable** (mensagem RF-05 generalizada):
   ```
   copilot não encontrado. Install copilot CLI; OR install @github/copilot@<pin> via npm; OR use --runtime=legacy. Veja .specs/adr/012-copilot-cli-acp-native.md
   ```

3. **CLI — aviso de depreciação legado**:
   ```
   WARNING: Copilot CLI em modo legado (sem ACP). Migrar para --runtime=acp.
            O modo legado será removido em vX.Y.Z. Ver ADR-012.
   ```
   Emitido uma única vez por execução do processo (não por task).

4. **`COPILOT.md` raiz** reescrito conforme exemplo em [`docs/research/compozy-adaptation-copilot-2026.md`](../../docs/research/compozy-adaptation-copilot-2026.md) §"Exemplos de Configuração 2026".

## Restrições Técnicas de Alto Nível

- **Linguagem e protocolo**: Go, mantendo `coder/acp-go-sdk` como SDK ACP (ADR-009). Versão sincronizada com `go.mod` via `scripts/sync-acp-sdk-version.sh`.
- **Reuso obrigatório**: `ACPRunner`, `acpClient` (`internal/runtime/client/client.go`), `SessionPersistence`, `ActivityWatchdog` e `events` package são reusados sem modificação semântica. Apenas extensões aditivas em `runner.go`/`probe.go` para deixar de ser Claude-specific.
- **Pinning de SDK e pacote npm**: `CopilotNpmVersion` é constante Go pinada (não `@latest`); atualização via processo `audit/` (espelha `ClaudeNpmVersion` em `claude.go:8-22`). `CopilotSDKVersion` mantida em sincronia com `go.mod`.
- **Filesystem abstraction**: leitura de configs e write de artefatos forenses continuam sobre `fs.FileSystem` (ADR-002).
- **Invariantes preservadas**:
  - Persistência forense intacta (`internal/runtime/persistence/`).
  - `ActivityWatchdog` intacto (`internal/runtime/watchdog.go`).
  - `acpClient` intacto em comportamento de subprocess management e fan-out de eventos.
  - Tabela de compatibilidade tool↔model (`internal/taskloop/compatibility.go`) continua autoritativa.
  - ADR-009 (pinning SDK) e ADR-010 (tagged union de eventos) inalterados.
- **Compatibilidade**: caminho legado (`copilotInvoker` em `internal/taskloop/agent.go:381-388`) é mantido por uma versão com aviso de depreciação. Remoção é decisão de versão futura, não desta fase.
- **Segurança (R-SEC-001)**: subprocess `copilot --acp` segue mesmas regras de Claude — sem shell, args via `exec.Command` com slice. Auth Copilot pressupõe `gh auth status` válido; ausência produz erro claro sem expor token.
- **Limites operacionais**: probe Copilot deve resolver em < 200ms p95 (binary) e < 2s p95 (npx fallback) em ambiente padrão.
- **Telemetria**: aditiva. Campos `tool`, `launcher`, `npm_version`, `sdk_version` em `runtime_init` ganham cardinalidade `tool=copilot` mas sem novo kind de evento (ADR-010 invariante preservada).
- **Documentação versão mínima**: techspec e `COPILOT.md` documentam versão mínima do `copilot` CLI que expõe `--acp` (a confirmar via `copilot --version` em ambiente de desenvolvimento e via release notes upstream).

## Fora de Escopo

Os itens abaixo **não** fazem parte desta Fase 1. São PRDs futuros documentados em [`docs/research/compozy-adaptation-copilot-2026.md`](../../docs/research/compozy-adaptation-copilot-2026.md) §"Continuidade":

- **F2 — Memória 2-níveis prompt-driven** (`MEMORY.md` workflow + `task_N.md` task-local com compactação prompt-driven).
- **F3 — Hook System Go in-process** (33 hooks canônicos, capability gating, manifesto TOML para extensões subprocess).
- **F4 — TUI Bubble Tea + daemon HTTP** (`charm.land/bubbletea/v2`, `AttachRemote`, stream cursor, journal append-before-publish).
- **MCP server reservado `run_agent`** (compozy `internal/core/agents/mcpserver/`) — depende de Agent Registry (já entregue em F1 anterior).
- **Generalização para Codex/Gemini/Droid via ACP** — depende de SDKs upstream estabilizarem; reabrir quando ≥ 1 SDK adicional disponível.
- **Remoção do `copilotInvoker` legado** — decisão de versão futura, não desta fase.
- **Mudança em modo avançado** (`--executor-tool=copilot --reviewer-tool=copilot`) — esta fase cobre apenas modo simples (`--tool copilot`); modo avançado é incremento aditivo posterior, se demandado.
- **Mudanças em `.github/copilot-instructions.md`** — mantido funcional para Copilot Chat no editor; não toca neste PRD.
- **Mudanças em CI workflow** (além de validar matriz Copilot quando disponível em runner) — fora de escopo desta fase.

## Suposições e Questões em Aberto

**Suposições assumidas** (devem ser validadas no TechSpec):

- A1: O Copilot CLI versão `X.Y.Z`+ expõe `--acp` com semântica idêntica ao Claude (`@agentclientprotocol/claude-agent-acp@0.1.0`) — protocolo JSON-RPC sobre stdio, mesmos kinds de eventos. **Validação**: testar contra fake ACP server reusando matriz Claude.
- A2: O pacote npm `@github/copilot` existe e suporta `--acp` quando invocado via `npx --yes @github/copilot@<pin> --acp`. **Validação**: checar registry npm e release notes.
- A3: Auth do Copilot CLI (token `gh`) é pré-condição operacional, não responsabilidade do harness. Ausência produz erro do subprocess que o harness reporta sem inventar fluxo de auth.
- A4: `ActivityWatchdog` com timeout default de 120s funciona para Copilot — eventos `agent.on_session_update` chegam com cadência compatível.
- A5: Generalizar `runtime_init` para versões metadata-driven (não constantes hardcoded) não quebra consumidores downstream (telemetria, dashboards, ADR-010 tagged union).
- A6: `copilotInvoker` legado pode coexistir com modo ACP no mesmo binário sem conflito de flag (já coexistem hoje; apenas o roteamento muda).

**Questões em aberto** (não bloqueantes para PRD; serão resolvidas no TechSpec):

- Q1: Versão mínima exata do `copilot` CLI que expõe `--acp` — precisa ser confirmada via release notes do GitHub Copilot CLI ou via `copilot --version` em ambiente teste.
- Q2: Modelo default do Copilot quando `--model` não é passado — usar default do CLI (passar args vazio) ou definir constante `DefaultCopilotModel`?
- Q3: Existe flag análoga a `--bypass-permissions` no Copilot CLI? Se sim, qual `AccessModeFlag` declarar na Spec? Se não, `AccessModeFlag=""` é aceitável.
- Q4: Versão pinada inicial de `@github/copilot` para `CopilotNpmVersion` — precisa de verificação no registry npm. Proposta: usar major.minor latest stable do registry no momento da implementação.
- Q5: Quanto tempo (quantas versões do harness) manter `copilotInvoker` legado antes de remover? Proposta: 2 versões minor (deprecação na vX.Y; remoção em vX.Y+2).
- Q6: Promover invariantes Copilot em ADR-008 de `BestEffort` para `Required` nesta PR ou em PR separado? Proposta: nesta PR como parte de F1 (RF-14 cobre AGENTS.md; ADR-008 update fica em task separada se necessário).
- Q7: Eventos `runtime_init` carregam `npm_version` e `sdk_version` mesmo quando launcher é `binary` (sem npx)? Hoje sim (Claude) mas com valores Claude. Proposta: continuar carregando, derivando do `Spec` resolvido independente do launcher kind.
