# Documento de Requisitos do Produto (PRD) — Codex CLI via ACP Nativo

<!-- spec-version: 1 -->

> **Insumo de pesquisa**: [docs/research/compozy-adaptation-codex-2026.md](../../docs/research/compozy-adaptation-codex-2026.md)
> **ADR material**: [tasks/adr/013-codex-cli-acp-native.md](../adr/013-codex-cli-acp-native.md)
> **Precedente direto**: F1-Copilot ([tasks/prd-copilot-acp-spec/](../prd-copilot-acp-spec/)) — gate-keeper de generalização runtime
> **Fase do roadmap**: 1 de 4 (adaptação Compozy 2026 — variante Codex; F2/F3/F4 herdam do roadmap Copilot)
> **Data**: 2026-05-21

## Visão Geral

O `ai-spec-harness` integra Claude (ADR-009, F1-Claude) e Copilot (ADR-012, F1-Copilot) via protocolo ACP nativo com persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`), watchdog de inatividade e telemetria opt-in (ADR-006). Codex hoje é invocável apenas em modo stateless via `codex exec --yolo <prompt>` em `internal/taskloop/agent.go:335-351` — sem nenhuma dessas garantias. O gate em `cmd/ai_spec_harness/task_loop.go:82-97` bloqueia explicitamente `--tool codex --runtime acp` (T-14 em `task_loop_test.go:48-52` valida a rejeição).

Em 2026 o ecossistema ACP ganhou o adapter `@zed-industries/codex-acp` (último stable v0.14.0; mínimo v0.12.0 para `gpt-5.5`), permitindo invocar Codex via JSON-RPC sobre stdio com paridade conceitual a Claude/Copilot. O `compozy/compozy` (`main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`) registra Codex como runtime ACP de primeira classe em `internal/core/agent/registry_specs.go:106-122`, mas com uma divergência estrutural: Codex injeta `model`, `model_reasoning_effort`, feature toggles e sandbox controls via **`BootstrapArgs` dinâmico** (pares `-c key="value"` calculados em tempo de spawn) — não via `FixedArgs` estáticos como Claude/Copilot.

Esta funcionalidade introduz **Codex CLI como runtime ACP** no harness via o adapter `codex-acp`, atingindo paridade observacional total com Claude/Copilot e expondo `--reasoning-effort` e `--access-mode` como flags CLI primárias. ADR-013 declara a decisão. Beneficia operadores que usam Codex e hoje perdem telemetria, persistência forense e watchdog. Também desbloqueia ADR-008 (paridade multi-tool) para o terceiro runtime e fundamenta a extensão da interface `Spec` que destravam futuros runtimes ACP com bootstrap dinâmico (Droid, Gemini quando estabilizarem).

Não introduz novo runtime nem nova camada — reaproveita 100% do stack ACP construído para Claude (F1-Claude) e generalizado por Copilot (F1-Copilot): `ACPRunner`, `acpClient`, `persistence`, `events`, `watchdog`.

## Objetivos

- **OB-01**: Permitir executar tasks com Codex via protocolo ACP nativo usando `--tool codex --runtime acp`.
- **OB-02**: Atingir paridade observacional Codex ↔ Claude ↔ Copilot: `events.jsonl`, `tool_calls.md`, `execution_report.md` e telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) cobrem invocações Codex com mesmos campos e granularidade.
- **OB-03**: Estender a interface `Spec` de forma retrocompatível para comportar `BootstrapArgs` dinâmico — permitindo que Codex injete `model`/`reasoning`/`sandbox` via `-c` flags sem quebrar Claude/Copilot (default no-op).
- **OB-04**: Expor `--reasoning-effort` e `--access-mode` como flags CLI primárias do `task-loop`, propagadas para `Job.ReasoningEffort` e `Job.AccessMode`. Codex consome; Claude/Copilot ignoram (sem efeito).
- **OB-05**: Preservar 100% retrocompatibilidade. Invocações sem `--runtime=acp` continuam usando `codexInvoker` CLI legado em `internal/taskloop/agent.go:335-351`; aviso de depreciação informa migração.
- **OB-06**: Preservar invariantes forenses, watchdog (`ActivityWatchdog` com `CancelCause`), pinning de SDK (ADR-009) e tagged union de eventos (ADR-010) sem regressão para Claude/Copilot.
- **OB-07**: Documentar a confusão de nomenclatura `codex` (CLI legacy da OpenAI) vs `codex-acp` (adapter da Zed Industries) em `CODEX.md`, ADR-013 e mensagens de erro de probe.

**Métricas mensuráveis**:
- 100% dos testes de regressão Claude **e Copilot** permanecem verdes após estender a interface `Spec` com `BootstrapArgs`.
- Matriz de teste ACP (`internal/runtime/acp_integration_test.go`) cobre Codex com ≥ 90% dos casos cobertos para Copilot.
- Probe Codex resolve em < 200ms p95 (binary path) e < 2s p95 (fallback npx) em ambiente padrão.
- Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go` (três módulos preservam invariantes existentes).
- `BootstrapArgs` no-op para Claude/Copilot retorna `nil` em 100% dos casos (validado por testes T-05 e T-19).
- `CodexNpmVersion = "0.14.0"` confirmado disponível no npm registry (validado nesta sessão: `npm view @zed-industries/codex-acp versions` retorna `[..., "0.12.0", "0.13.0", "0.14.0"]`).

## Histórias de Usuário

- **HU-01**: Como **operador da CLI**, quero invocar `ai-spec-harness task-loop --tool codex --runtime acp tasks/prd-minha-feature` e ver o harness conectar via ACP, produzir `events.jsonl` linha-a-linha e gerar `execution_report.md` no padrão Claude/Copilot.
- **HU-02**: Como **operador**, quero passar `--reasoning-effort {low,medium,high}` para controlar o esforço de raciocínio do Codex, propagado como `-c model_reasoning_effort="<level>"` no `codex-acp`.
- **HU-03**: Como **operador**, quero passar `--access-mode {restricted,full}` para escolher entre modo sandboxed (default) e modo full-access (com warning explícito). `full` aciona `-c approval_policy="never" -c sandbox_mode="danger-full-access" -c web_search="live"`.
- **HU-04**: Como **operador**, quero que quando o binário `codex-acp` não está no PATH, o harness tente `npx --yes @zed-industries/codex-acp@<pin>` como fallback automático sem mudança de comando.
- **HU-05**: Como **operador**, quero ver erro claro quando nem `codex-acp` nem `npx` estão disponíveis, com os três remédios (instalar `codex-acp` via `npm install -g @zed-industries/codex-acp@latest`; instalar pacote npm direto; usar `--runtime=legacy`) e referência a ADR-013.
- **HU-06**: Como **operador legado**, quero que `ai-spec-harness task-loop --tool codex tasks/...` (sem `--runtime=acp`) continue funcionando como hoje (via `codexInvoker` invocando `codex exec --yolo`), com aviso de depreciação em log.
- **HU-07**: Como **mantenedor**, quero que telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) registre invocações Codex ACP com `tool=codex`, `launcher=binary|npx`, `npm_version=0.14.0`, `sdk_version=v0.13.0` reais.
- **HU-08**: Como **mantenedor**, quero que `--reasoning-effort high --access-mode full --tool claude --runtime acp` seja aceito mas **sem efeito** sobre o spawn de Claude (que tem `BootstrapArgs` no-op) — validado por teste T-19.
- **HU-09**: Como **mantenedor**, quero que o ADR-013 seja referenciado no índice em `AGENTS.md` e em `CODEX.md`, e que a tabela `adrByID` em `internal/runtime/probe/probe.go` aponte `"codex"` para ADR-013.
- **HU-10**: Como **operador**, quero que `ActivityWatchdog` cancele sessões Codex inativas com o mesmo timeout configurável que cancela sessões Claude/Copilot.

## Funcionalidades Core

### F-01: Extensão da interface `Spec` com `BootstrapArgs`
`internal/runtime/specs/spec.go:12-23` ganha tipo `AccessMode string` (consts `AccessModeRestricted`, `AccessModeFull`) e método `BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string` no `Spec` com default no-op (retorna `nil`). Mudança retrocompatível: Claude/Copilot herdam o no-op sem alteração de comportamento.

### F-02: Spec Codex ACP em catálogo
Novo construtor `specs.Codex()` em `internal/runtime/specs/codex.go` retornando uma `Spec` com `ID="codex"`, `DisplayName="Codex (ACP)"`, `Command="codex-acp"`, `FixedArgs=nil`, `Fallbacks=[npx --yes @zed-industries/codex-acp@<pin>]`, `AccessModeFlag=""`, e `BootstrapArgs: codexBootstrapArgs` (função local). Constantes `CodexNpmPackage`, `CodexNpmVersion="0.14.0"`, `CodexSDKVersion="v0.13.0"`, `DefaultCodexModel="gpt-5.5"`, `CodexMinNpmVersion="0.12.0"` (gating documentado pelo compozy para `gpt-5.5`).

### F-03: Função `codexBootstrapArgs`
Função local em `internal/runtime/specs/codex.go` replicando fielmente `compozy/internal/core/agent/registry_specs.go:247-278`. Emite pares `-c key="value"`:
- `model="<name>"` quando `model` ≠ vazio
- `model_reasoning_effort="<level>"` quando `reasoning` ≠ vazio
- Sempre: `features.code_mode=false`, `features.code_mode_only=false`
- Em `AccessModeFull` adiciona: `approval_policy="never"`, `sandbox_mode="danger-full-access"`, `web_search="live"`

### F-04: Generalização do runner para consumir `BootstrapArgs`
`internal/runtime/runner.go` em `Run()` chama `r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)` e faz prepend ao argv do launcher antes das `FixedArgs`. Comportamento atual de Claude/Copilot preservado (no-op retorna `nil`, prepend de `nil` é idempotente).

### F-05: Extensão de `Job` com novos campos
`internal/runtime/runner.go::Job` ganha campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string`. Defaults: `""`, `AccessModeRestricted`, `nil`. Campos opcionais — ausência preserva comportamento atual.

### F-06: Flags CLI `--reasoning-effort` e `--access-mode`
`cmd/ai_spec_harness/task_loop.go` ganha duas flags novas:
- `--reasoning-effort` (string, default `"medium"`, aceita `low|medium|high`)
- `--access-mode` (string, default `"restricted"`, aceita `restricted|full`)
Propagadas via `taskloop.Options` até `Job.ReasoningEffort` / `Job.AccessMode`. Validação de valores aceitos em `task_loop.go` antes de propagar.

### F-07: Registro Codex em `runtimeACPCatalog`
`cmd/ai_spec_harness/task_loop.go:21-24` ganha `"codex": specs.Codex` no catálogo. Linha 82-97 (gate) passa a aceitar Codex automaticamente sem mudança de código (mensagem de erro lista o catálogo dinamicamente).

### F-08: Wiring de Codex no taskloop service
`internal/taskloop/taskloop.go::Service.Execute` resolve Spec via `runtimeACPCatalog` quando `Runtime == "acp"` e `Tool == "codex"`. Reusa `ACPRunner` existente. Propaga `ReasoningEffort`, `AccessMode`, `AddDirs` para `Job`.

### F-09: Inversão de T-14 e novo T-15
`cmd/ai_spec_harness/task_loop_test.go:48-52` (T-14, atualmente valida rejeição de `--tool codex --runtime acp`) é **invertido** para esperar aceitação. Novo caso T-15: `--tool codex --runtime acp --reasoning-effort high --access-mode full` aceito e flags propagadas corretamente.

### F-10: Matriz de teste ACP estendida
`internal/runtime/acp_integration_test.go` ganha sub-suite Codex reusando o fake ACP server existente. Casos cobertos: open OK, prompt, ≥ 2 tipos de tool call, agent message, completion, cancel por timeout, erro de launcher unavailable, fallback npx, validação de que `BootstrapArgs` produz `-c` flags no spawn correto.

### F-11: Aviso de depreciação no caminho legado
`codexInvoker` em `internal/taskloop/agent.go:335-351` ganha log WARNING uma única vez por execução do processo (via `sync.Once`): "Codex CLI legado (`codex exec --yolo`) em uso. Migrar para --runtime=acp (binário `codex-acp`). Ver ADR-013." Espelha padrão de F1-Copilot.

### F-12: Reescrita de `CODEX.md`
`CODEX.md` raiz (esqueleto atual de 30 linhas) é reescrito (~80-100 linhas): seção "Modo Recomendado (2026): Codex via ACP" primeira; seção "Modo Legado" deprecada; documentação explícita da confusão `codex` vs `codex-acp`; descrição das flags `--reasoning-effort`/`--access-mode`; warning sobre `--access-mode=full`.

### F-13: Telemetria enriquecida
Evento `runtime_init` no telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) ganha tool real (`tool=codex` quando aplicável), `npm_version=0.14.0`, `sdk_version=v0.13.0`. Sem novo kind de evento (preserva ADR-010).

### F-14: Tabela `adrByID` em probe
`internal/runtime/probe/probe.go:21-24` adiciona `"codex": "tasks/adr/013-codex-cli-acp-native.md"`. Mensagem de erro de probe Codex referencia ADR-013 explicitamente.

## Requisitos Funcionais

- **RF-01**: Criar `specs.Codex()` retornando `Spec` com `ID="codex"`, `DisplayName="Codex (ACP)"`, `Command="codex-acp"`, `FixedArgs=nil`, `BootstrapArgs=codexBootstrapArgs`.
- **RF-02**: Spec Codex expõe pelo menos um `FallbackLauncher` com `Command="npx"`, `FixedArgs=["--yes", CodexNpmPackage+"@"+CodexNpmVersion]`.
- **RF-03**: Constantes `CodexNpmPackage="@zed-industries/codex-acp"`, `CodexNpmVersion="0.14.0"`, `CodexSDKVersion="v0.13.0"`, `CodexMinNpmVersion="0.12.0"`, `DefaultCodexModel="gpt-5.5"` declaradas em `internal/runtime/specs/codex.go` com política de atualização equivalente a Claude/Copilot (ADR-009).
- **RF-04**: Interface `Spec` em `internal/runtime/specs/spec.go` ganha método `BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string` com default no-op (retorna `nil`).
- **RF-05**: Tipo `AccessMode string` declarado em `internal/runtime/specs/spec.go` com consts `AccessModeRestricted = "restricted"`, `AccessModeFull = "full"`.
- **RF-06**: Função `codexBootstrapArgs` em `internal/runtime/specs/codex.go` replica fielmente o compozy: emite `-c model=...`, `-c model_reasoning_effort=...`, `-c features.code_mode=false`, `-c features.code_mode_only=false`, e em `AccessModeFull` adiciona `-c approval_policy="never"`, `-c sandbox_mode="danger-full-access"`, `-c web_search="live"`.
- **RF-07**: `internal/runtime/runner.go::Job` ganha campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string`. Defaults preservam comportamento atual de Claude/Copilot.
- **RF-08**: `internal/runtime/runner.go` em `Run()` chama `spec.BootstrapArgs(...)` e prepend ao argv do launcher antes das `FixedArgs`. Para Claude/Copilot (no-op `nil`), prepend é idempotente.
- **RF-09**: `cmd/ai_spec_harness/task_loop.go` registra flag `--reasoning-effort` (default `"medium"`, valores aceitos: `low|medium|high`).
- **RF-10**: `cmd/ai_spec_harness/task_loop.go` registra flag `--access-mode` (default `"restricted"`, valores aceitos: `restricted|full`).
- **RF-11**: Validação de valores de `--reasoning-effort` e `--access-mode` em `task_loop.go` antes de propagar. Valores inválidos retornam erro `exit2` com mensagem clara.
- **RF-12**: `runtimeACPCatalog` em `cmd/ai_spec_harness/task_loop.go:21-24` ganha entrada `"codex": specs.Codex`.
- **RF-13**: `task_loop_test.go:48-52` (T-14) **invertido**: `--tool codex --runtime acp` passa a ser aceito. T-15 novo cobre `--reasoning-effort high --access-mode full`.
- **RF-14**: `internal/runtime/probe/probe.go:21-24` ganha `"codex": "tasks/adr/013-codex-cli-acp-native.md"` em `adrByID`.
- **RF-15**: `internal/taskloop/taskloop.go` roteia Codex via `ACPRunner` quando `Runtime == "acp"`. Propaga `Options.ReasoningEffort`/`AccessMode`/`AddDirs` para `Job`.
- **RF-16**: `codexInvoker` legado em `internal/taskloop/agent.go:335-351` emite log WARNING único por execução (via `sync.Once`) referenciando ADR-013 e binário `codex-acp`.
- **RF-17**: `internal/runtime/acp_integration_test.go` ganha sub-suite Codex reusando fake ACP server. Cobertura mínima: open, prompt, ≥ 2 tipos de tool call, agent message, completion, cancel por `ActivityWatchdog`, validação de spawn args (`-c model="gpt-5.5"`, etc.).
- **RF-18**: `internal/runtime/specs/codex_test.go` cobre defaults da Spec, fallback npx, constantes pinadas SemVer-like, e função `codexBootstrapArgs` com matriz: model vazio, model setado, reasoning vazio/low/medium/high, accessMode restricted/full.
- **RF-19**: Persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e `ActivityWatchdog` permanecem inalterados em comportamento e código fonte. Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go`.
- **RF-20**: Reescrever `CODEX.md` raiz com seção "Modo Recomendado (2026)" primeira; "Modo Legado" deprecada; warning sobre `--access-mode=full`; documentação da confusão `codex` vs `codex-acp`.
- **RF-21**: Atualizar `AGENTS.md` adicionando linha na tabela de ADRs (ADR-013).
- **RF-22**: Atualizar `docs/cli-schema.json` adicionando enums de `--reasoning-effort` e `--access-mode`.
- **RF-23**: Atualizar `docs/telemetry-feedback-cycle.md` documentando que invariantes Codex ACP cobrem os mesmos kinds que Claude/Copilot.
- **RF-24**: Tabela `internal/taskloop/compatibility.go::CompatibilityTable` ganha entrada para `tool=codex` (ou aceita `--allow-unknown-model` para modelos não catalogados — manter semântica atual).
- **RF-25**: Cache de probe em `internal/runtime/probe/probe.go` continua keyed por `spec.ID` — múltiplas invocações Codex em uma sessão CLI reutilizam o launcher resolvido sem re-probe.
- **RF-26**: Regressão obrigatória para Claude e Copilot: 100% dos testes de `internal/runtime/specs/claude_test.go` e `copilot_test.go` permanecem verdes após F-01 (extensão da interface `Spec`).
- **RF-27**: Telemetria opt-in registra `runtime_init` com `tool=codex` quando aplicável; campo `launcher` distingue `binary` de `npx`.

## Experiência do Usuário

UX é primariamente backend/CLI; materializa-se em quatro pontos:

1. **CLI — comando recomendado**:
   ```
   ai-spec-harness task-loop \
     --tool codex \
     --runtime acp \
     --reasoning-effort medium \
     --access-mode restricted \
     tasks/prd-minha-feature
   ```
   Comportamento idêntico a Claude/Copilot: stream humano em stdout (suprimível via `--quiet`), `events.jsonl`/`tool_calls.md`/`execution_report.md` em `audit/` ou `evidenceDir` configurado.

2. **CLI — comando full-access (com warning)**:
   ```
   ai-spec-harness task-loop \
     --tool codex --runtime acp \
     --reasoning-effort high --access-mode full \
     tasks/prd-minha-feature
   ```
   `--access-mode=full` aciona warning único em stderr: `"WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. Pré-condição: consentimento operacional. Ver CODEX.md."`

3. **CLI — erro de launcher unavailable** (mensagem RF-14 generalizada):
   ```
   codex-acp não encontrado. Install codex-acp; OR install @zed-industries/codex-acp@0.14.0 via npm; OR use --runtime=legacy. Veja tasks/adr/013-codex-cli-acp-native.md
   ```

4. **CLI — aviso de depreciação legado** (`codexInvoker`):
   ```
   WARNING: Codex CLI legado (codex exec --yolo) em uso. Migrar para --runtime=acp (binário codex-acp). Ver ADR-013.
   ```
   Emitido uma única vez por execução do processo (não por task).

5. **`CODEX.md` raiz** reescrito conforme exemplo em [`docs/research/compozy-adaptation-codex-2026.md`](../../docs/research/compozy-adaptation-codex-2026.md) §"Exemplos de Configuração 2026".

## Restrições Técnicas de Alto Nível

- **Linguagem e protocolo**: Go, mantendo `coder/acp-go-sdk` como SDK ACP (ADR-009). Versão sincronizada com `go.mod` via `scripts/sync-acp-sdk-version.sh`.
- **Reuso obrigatório**: `ACPRunner`, `acpClient` (`internal/runtime/client/client.go`), `SessionPersistence`, `ActivityWatchdog` e `events` package são reusados sem modificação semântica. Apenas extensões aditivas em `runner.go` (consumir `BootstrapArgs`) e em `Spec` (adicionar método).
- **Pinning de SDK e pacote npm**: `CodexNpmVersion = "0.14.0"` constante Go pinada (não `@latest`); atualização via processo `audit/` (espelha `ClaudeNpmVersion` em `claude.go:8-22` e `CopilotNpmVersion` em `copilot.go:8-27`). `CodexSDKVersion = "v0.13.0"` mantida em sincronia com `go.mod` (mesma do Claude/Copilot).
- **Filesystem abstraction**: leitura de configs e write de artefatos forenses continuam sobre `fs.FileSystem` (ADR-002).
- **Invariantes preservadas**:
  - Persistência forense intacta (`internal/runtime/persistence/`).
  - `ActivityWatchdog` intacto (`internal/runtime/watchdog.go`).
  - `acpClient` intacto em comportamento de subprocess management e fan-out de eventos.
  - Tabela de compatibilidade tool↔model (`internal/taskloop/compatibility.go`) continua autoritativa.
  - ADR-009 (pinning SDK), ADR-010 (tagged union), ADR-011 (Agent Registry), ADR-012 (Copilot ACP) inalterados.
- **Compatibilidade**: caminho legado (`codexInvoker` em `internal/taskloop/agent.go:335-351`) é mantido por 2 versões minor com aviso de depreciação. Remoção é decisão de versão futura, não desta fase. Mesma política de F1-Copilot (Q5 de ADR-012).
- **Segurança (R-SEC-001)**: subprocess `codex-acp` segue mesmas regras de Claude/Copilot — sem shell, args via `exec.Command` com slice. **`AccessModeFull` é opt-in explícito**: warning único em stderr antes de propagar `sandbox_mode="danger-full-access"`. Ausência de consentimento operacional não é detectável programaticamente — confiamos no consentimento via flag.
- **Limites operacionais**: probe Codex deve resolver em < 200ms p95 (binary) e < 2s p95 (npx fallback) em ambiente padrão.
- **Telemetria**: aditiva. Campos `tool`, `launcher`, `npm_version`, `sdk_version` em `runtime_init` ganham cardinalidade `tool=codex` mas sem novo kind de evento (ADR-010 invariante preservada).
- **Documentação versão mínima**: techspec e `CODEX.md` documentam `CodexMinNpmVersion=0.12.0` (mínimo do compozy para `gpt-5.5`) e `CodexNpmVersion=0.14.0` (pin atual, último stable em 2026-05-21).
- **Mensagem de erro distingue `codex` vs `codex-acp`**: probe error em `internal/runtime/probe/probe.go` deve referenciar **`codex-acp`** (não `codex`) para evitar confusão com CLI legado.

## Fora de Escopo

Os itens abaixo **não** fazem parte desta Fase 1-Codex. Estão documentados como follow-ups em [`docs/research/compozy-adaptation-codex-2026.md`](../../docs/research/compozy-adaptation-codex-2026.md) §"Roadmap":

- **F2-Codex — Tool name aliasing** (`search_query` → `web_search`, `image_query` → `image_search` em `internal/runtime/events/convert.go`). Decisão registrada como follow-up opcional, não bloqueante.
- **F3-Codex — `$CODEX_HOME` resolver no install flow**. Pequeno (~50 LoC) mas independente do runtime path; vira PR próprio quando o install flow merger.
- **Herdadas do roadmap Copilot** (cobertas sem código adicional após F1-Codex):
  - F2-Copilot — Memória 2-níveis prompt-driven
  - F3-Copilot — Hook System Go in-process (33 hooks canônicos)
  - F4-Copilot — TUI Bubble Tea + daemon HTTP
- **MCP server reservado `run_agent`** (compozy `internal/core/agents/mcpserver/`) — depende de Agent Registry (F1 anterior já entregue).
- **Generalização para Droid/Gemini via ACP** — F1-Codex destrava a interface `Spec.BootstrapArgs(...)` mas implementação de cada novo runtime é PRD separado.
- **Remoção do `codexInvoker` legado** — decisão de versão futura (mesma política de F1-Copilot).
- **Mudança em modo avançado** (`--executor-tool=codex --reviewer-tool=codex`) — esta fase cobre apenas modo simples (`--tool codex`); modo avançado é incremento aditivo posterior se demandado.
- **Validação runtime da versão de `codex-acp`** (ex: `codex-acp --version >= 0.12.0`) — probe não valida versão; assume disponibilidade quando `LookPath` resolve. Documentar limitação em `CODEX.md`.
- **Mudanças em CI workflow** (além de validar matriz Codex quando `codex-acp` disponível em runner) — fora de escopo desta fase. Smoke real é manual.

## Suposições e Questões em Aberto

**Suposições assumidas** (devem ser validadas no TechSpec):

- A1: O adapter `codex-acp >= 0.12.0` expõe ACP com semântica idêntica ao Claude/Copilot — protocolo JSON-RPC sobre stdio, mesmos kinds de eventos. **Validação**: testar contra fake ACP server reusando matriz Claude/Copilot. **Validação parcial**: docs em https://github.com/zed-industries/codex-acp confirmam compatibilidade ACP.
- A2: O pacote npm `@zed-industries/codex-acp@0.14.0` existe e suporta o protocolo de overrides `-c`. **Validação concluída em 2026-05-21**: `npm view @zed-industries/codex-acp versions` confirmou versões `[..., 0.12.0, 0.13.0, 0.14.0]` publicadas. 0.14.0 é o último stable.
- A3: Auth do Codex/`codex-acp` (token OpenAI ou conta Zed) é pré-condição operacional, não responsabilidade do harness. Ausência produz erro do subprocess que o harness reporta.
- A4: `ActivityWatchdog` com timeout default de 120s funciona para Codex — eventos `agent.on_session_update` chegam com cadência compatível.
- A5: Estender interface `Spec` com `BootstrapArgs(...)` não quebra Claude/Copilot (validado por T-05/T-19 — no-op retorna `nil`).
- A6: `codexInvoker` legado pode coexistir com modo ACP no mesmo binário sem conflito de flag (já coexistem hoje; apenas o roteamento muda).
- A7: Flags `--reasoning-effort` e `--access-mode` passadas com `--tool claude` ou `--tool copilot` são aceitas mas sem efeito (BootstrapArgs no-op retorna `nil`). Documentar isso em help text.

**Questões em aberto** (não bloqueantes para PRD; serão resolvidas no TechSpec):

- Q1: Mensagem do warning de `--access-mode=full`. Proposta: explícita o suficiente para evitar uso acidental ("ATENÇÃO: sandbox totalmente desabilitado; Codex tem acesso pleno ao filesystem e à rede. Use somente em ambientes isolados.")
- Q2: Modelo default do Codex quando `--model` não passado. Proposta: usar `DefaultCodexModel="gpt-5.5"` (mesmo default do compozy) propagado via `--executor-model` quando não especificado.
- Q3: Tabela de compatibilidade `internal/taskloop/compatibility.go::CompatibilityTable` — incluir entrada para `tool=codex` com modelos catalogados (`gpt-5.5`) ou apenas confiar em `--allow-unknown-model`? Proposta: incluir entrada mínima `codex → [gpt-5.5]`; outros modelos requerem flag.
- Q4: Quanto tempo manter `codexInvoker` legado antes de remover? Proposta: 2 versões minor (alinhado a Q5 de ADR-012).
- Q5: `--reasoning-effort` aceita valores literais `low/medium/high` ou também aceita strings arbitrárias (Codex API pode evoluir)? Proposta: validar contra enum fixo nesta fase; relaxar se necessário em PR futuro.
- Q6: `BootstrapArgs` deve ser método na struct `Spec` (com campo função privado) ou campo público `Spec.BootstrapArgs func(...)`? Proposta: campo privado + acessor para preservar imutabilidade do value object (R-DDD-001).
- Q7: `runtime_init` carrega `npm_version=CodexNpmVersion` mesmo quando launcher é `binary` (sem npx)? Proposta: sim, consistente com Claude/Copilot (Q7 de ADR-012).
- Q8: Validação de versão mínima do `codex-acp` (semver `>= 0.12.0`) — implementar em F1-Codex ou adiar? Proposta: adiar para F2-Codex (não bloqueante); F1-Codex assume disponibilidade quando `LookPath` resolve.
