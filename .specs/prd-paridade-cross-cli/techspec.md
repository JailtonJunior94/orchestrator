<!-- spec-hash-prd: 95a7a64d15e9aa150aa149c10e4cfd0bde28f65e641bb5897c8c4d9c0df9cc83 -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Paridade Absoluta Cross-CLI e Instalação Universal Transparente

> PRD de origem: `.specs/prd-paridade-cross-cli/prd.md` (spec-hash acima).
> Stack: Go 1.26+, cobra, `coder/acp-go-sdk v0.13.0`. Convenções: `AGENTS.md`.
> Esta techspec descreve **como** implementar os requisitos RP-01..RP-04, RI-01..RI-04, RIN-01..RIN-05, RG-01..RG-04 — não reexplica o PRD.

## Resumo Executivo

O PRD comprova que ai-spec-harness e Compozy são peers arquiteturais: não há lacuna fundacional de transporte ACP. As lacunas reais são de **camada de aplicação e governança**, distribuídas em três frentes: (1) **paridade cross-CLI** da normalização de tool-calls e das métricas; (2) **transparência** do instalador (stack-aware, probe, stubs, verify); (3) **governança de runtime** (spec-hash/PRD-first em execução). A estratégia central é fechar gaps por **wiring de infra já existente e testada** (`internal/specdrift`, `internal/detect`, `internal/runtime/probe`, `internal/config/resolver`), introduzindo um pequeno núcleo de domínio para eliminar primitivos crus e structs anêmicas que hoje crescem por driver.

Quatro decisões de modelagem guiam a implementação, todas com **zero-value = comportamento F1 (regressão zero)**: (a) `DriverID` como Value Object e fonte única dos drivers suportados (ADR-020); (b) `MetricSet` VO + `MetricsExtractor` por driver (Strategy) substituindo os 11 campos planos de métrica do `Summary` (ADR-021); (c) guard de governança em runtime reusando `specdrift` no hook `runtime.pre_open` (ADR-022); (d) política de janela CLI-aware via campo estático na `Spec` (ADR-023). O instalador transparente (ADR-024) e o wiring do `RuntimeConfig` hierárquico (ADR-025) completam a entrega; o sunset do legacy mode (ADR-026) é planejado, não executado neste ciclo.

## Arquitetura do Sistema

### Visão Geral dos Componentes

Componentes **novos** (N) e **modificados** (M), por camada:

**Domínio** (regra pura, sem IO):
- `specs.DriverID` (N) — VO; identidade das 4 CLIs; `ParseDriverID`/`ErrUnknownDriver`. ADR-020.
- `specs.ContextWindow` + `WindowClass` (N) — VO de janela de contexto por CLI. ADR-023.
- `events.MetricSet` (N) — VO de métricas canônicas (mínimo comum + `Extra`). ADR-021.
- `events.MetricsExtractor` (N) + `claudeExtractor`/`geminiExtractor`/`nullExtractor` (N/M) — Strategy por driver. ADR-021.
- `memory.WindowPolicy` (N) — domain service stateless: limites de compactação por `WindowClass`. ADR-023.
- `events.normalizationRules` + tabelas YAML (M) — completar `aliases.gemini`, `input_mappings.copilot/gemini`. ADR-020.

**Aplicação** (orquestração, fronteiras):
- `runtime.ACPRunner` (M) — consome `DriverID`, acumula `MetricSet` único, registra `SpecDriftHook`, propaga `WindowClass`. ADR-020/021/022/023.
- `hooks.SpecDriftHook` (N) — guard spec-hash/PRD-first em `runtime.pre_open`. ADR-022.
- `hooks.TokenBudgetHook` (M) — limite por Spec/`WindowClass`. ADR-023.
- `taskloop.BuildRuntimeConfig` (N) — fronteira que mapeia `config.Runtime` resolvido → `runtime.RuntimeConfig`. ADR-025.
- `install.Service.Execute`/`Verify` (M) — stack-aware, probe não-fatal, verify de binário. ADR-024.

**Infraestrutura** (IO, adapters — reuso, sem mudança de contrato):
- `internal/specdrift` (reuso) — `CheckDrift`/`CheckHash`.
- `internal/detect` (reuso) — `DetectLangs`/`DetectTools`.
- `internal/runtime/probe` (reuso) — `EnsureAvailable`.
- `internal/config/resolver` (reuso) — `Resolve` (cascata ADR-016).
- `internal/runtime/persistence/report.go` (M) — `RenderMetricsSection` unificado.
- `internal/telemetry` (reuso) — append-only opt-in (ADR-006).

### Fronteiras entre Aplicação, Domínio e Infraestrutura

```
                 ┌─────────────────────────── Aplicação ───────────────────────────┐
  flags CLI ───▶ │  taskloop.BuildRuntimeConfig ──▶ runtime.ACPRunner.Run(Job)      │
  config.yaml ─▶ │            │                          │                          │
                 │            │                  ┌────────┴─────────┐               │
                 │            ▼                  ▼                  ▼               │
                 │   config.Resolver      hooks.Dispatcher    events loop          │
                 └────────────┼──────────────────┼──────────────────┼─────────────┘
                              │                   │                  │
        ┌─────────── Domínio ─┼───────────────────┼──────────────────┼───────────┐
        │  ContextWindow   DriverID   normalizationRules   MetricsExtractor       │
        │  WindowPolicy    (VO)       (alias/input)        ─▶ MetricSet (VO)       │
        └─────────────────────┼───────────────────┼──────────────────┼───────────┘
                              │                   │                  │
        ┌─── Infraestrutura ──┼───────────────────┼──────────────────┼───────────┐
        │  config/resolver   specdrift.CheckDrift  probe.EnsureAvailable          │
        │  (FS, HOME)        (FS: prd/tasks.md)    (PATH, npx)   persistence (FS) │
        └────────────────────────────────────────────────────────────────────────┘
```

Regras de fronteira (R-DDD-001):
- **Domínio não conhece IO/serialização.** `MetricSet`, `DriverID`, `ContextWindow`, `WindowPolicy`, `normalizationRules` operam sobre dados já lidos; o parsing de YAML/JSON e a leitura de arquivos ficam na borda (loader, extractor recebe `json.RawMessage` já obtido pelo SDK).
- **Aplicação orquestra.** `ACPRunner` resolve `DriverID` na fronteira (ao montar Spec), seleciona `MetricsExtractor`, dispara hooks. Não contém regra de transição de domínio.
- **Infra é substituível.** `specdrift`/`detect`/`probe`/`resolver` são consumidos por interface ou função pura; testáveis com `FakeFileSystem`/doubles.

## Modelagem de Domínio

O bounded context é **"Paridade Cross-CLI & Instalação Universal"**. Por ser um serviço Go/CLI (sem banco), o domínio é pequeno e focado em invariantes de paridade e governança, não em persistência transacional.

### Value Objects

| VO | Pacote | Invariante protegida | Construtor / validação |
|---|---|---|---|
| `DriverID` | `runtime/specs` | É um dos 4 drivers canônicos; imutável; identidade por valor | `ParseDriverID(string) (DriverID, error)` → `ErrUnknownDriver` se fora do conjunto |
| `ContextWindow` | `runtime/specs` | `MaxTokens >= 0`; classe derivada de forma estável | literal de catálogo; `Class() WindowClass` |
| `WindowClass` | `runtime/specs` | Conjunto fechado: `WindowStandard` \| `WindowLarge` | enum tipado (não string solta) |
| `MetricSet` | `runtime/events` | Contadores ≥ 0; imutável após build; soma associativa | `Merge(other) MetricSet`; `IsZero() bool` |

Critérios DDD aplicados: cada VO se autovalida, é imutável por design e expressa regra própria (encapsular primitivo — OC #3 / R-DDD-001). `WindowClass` substitui comparação de strings soltas (State Pattern leve).

```go
// runtime/specs/driver.go
type DriverID struct{ value string } // não exportar campo (R-DDD-001 §Entidades)

func ParseDriverID(s string) (DriverID, error) {
    switch s {
    case "claude", "codex", "copilot", "gemini":
        return DriverID{value: s}, nil
    default:
        return DriverID{}, fmt.Errorf("%w: %q", ErrUnknownDriver, s)
    }
}
func (d DriverID) String() string { return d.value }
```

```go
// runtime/events/metricset.go
type MetricSet struct {
    totalTokens     int
    cacheReadTokens int
    thinkingTokens  int
    extra           map[string]int // campos driver-específicos (effective_context_tokens, ...)
}

func (m MetricSet) Merge(o MetricSet) MetricSet { /* soma campo a campo + extra */ }
func (m MetricSet) IsZero() bool                { /* todos zero e extra vazio */ }
func (m MetricSet) Fields() []MetricField        { /* só não-zero, ordenado — para render */ }
```

### Entidades / Agregados

Este contexto tem **um agregado raiz operacional**: a `Spec` (catálogo de runtime por CLI), que passa a ser a fonte única de `DriverID` + `ContextWindow`. Não há entidades com ciclo de vida persistido (sem store transacional) — coerente com o domínio de uma CLI. As "entidades" relevantes são efêmeras por sessão:

- **`Spec` (aggregate root de configuração de driver)** — já existe; construída **apenas via construtores de catálogo** (`Claude()`, `Codex()`, ...), nunca por literal externo (R-DDD-001 §Proibido). Passa a expor `DriverID()` e `ContextWindow()`. Centraliza as capacidades por CLI.
- **`Job` (parâmetros de sessão)** — DTO de execução; isento de OC #8 (config). Ganha propagação de `WindowClass` e flag `SkipDriftGuard`.
- **`Summary` (resultado de sessão)** — passa a compor `Metrics MetricSet` em vez de 11 campos planos.

`normalizationRules` é tratado como **agregado de configuração** (carregado da borda, validado por `version`, com regra de herança `resolveInherit` centralizada) — alterações nas tabelas alias/input só ocorrem via o loader, nunca por mutação externa.

### Domain Services

- `events.ExtractorFor(DriverID) MetricsExtractor` — seleção da estratégia (stateless).
- `memory.WindowPolicy.LimitsFor(WindowClass) memory.Limits` — decisão de compactação (stateless, combina janela + defaults).
- `resolveInherit(*normalizationRules)` — regra de herança de aliases (já existente, stateless).

### Fail Fast (R-DDD-001 §Fail Fast)

- `ParseDriverID` rejeita driver inválido na fronteira (antes do loop de eventos).
- `BuildRuntimeConfig` rejeita `Timeout` malformado antes de `Run()`.
- `SpecDriftHook` aborta no `runtime.pre_open` (antes de `c.Open`) em drift/PRD não rastreável.

## Design de Implementação

### Interfaces Chave

```go
// runtime/events — Strategy de extração de métricas por driver (ADR-021).
type MetricsExtractor interface {
    Extract(raw json.RawMessage) MetricSet
}

// runtime/specs — capacidades expostas pela Spec (agregado de driver).
// (métodos novos; Spec continua construída só por catálogo)
func (s Spec) DriverID() DriverID            // ADR-020
func (s Spec) ContextWindow() ContextWindow  // ADR-023

// runtime/hooks — guard de governança (ADR-022); implementa a interface Hook existente.
type SpecDriftHook struct{ /* lê Job.TasksDir via RuntimePreOpenEvent */ }
func (h *SpecDriftHook) Name() string { return "spec_drift" }
func (h *SpecDriftHook) Run(ctx context.Context, evt Event) error // ErrSpecDrift | ErrPRDUntracked

// taskloop — fronteira de aplicação: cascata de config → RuntimeConfig (ADR-025).
func BuildRuntimeConfig(resolved config.Runtime) (runtime.RuntimeConfig, error)

// memory — domain service de política de janela (ADR-023).
type WindowPolicy interface {
    LimitsFor(class specs.WindowClass, base Limits) Limits
}
```

### Modelos de Dados

**`Summary` (modificado — ADR-021):** remover `CacheReadTokens`, `CacheCreationTokens`, `ThinkingTokens`, `ToolCallsNormalizedCount`, `GeminiCacheReadTokens`, `GeminiEffectiveContextTokens`, `GeminiPromptTokensBilled`, `GeminiThoughtsTokens`; adicionar `Metrics MetricSet`. (`ToolCallsNormalizedCount` migra para `Extra["tool_calls_normalized"]` ou campo próprio se preferido — manter no VO.)

**`Job` (modificado):** `SkipDriftGuard bool` (ADR-022); `WindowClass` derivado da Spec (propagado, não setado pelo usuário).

**`normalization-rules.yaml` (modificado — ADR-020):**
```yaml
aliases:
  gemini:            # RP-04: tabela explícita (não só inherit_common)
    bash: bash
    read_file: read
    write_file: write
    str_replace_editor: edit
input_mappings:
  copilot:           # RP-01: hoje ausente
    run:
      command: command   # ou "# no-op verificado" se confirmado
  gemini:            # RP-01: hoje ausente
    bash:
      command: command
```

**`VerifyItem` (estendido — ADR-024):** adicionar tipo/kind `binary` (por CLI: `current`/`missing`) ao lado dos itens de skill.

### Mapeamento Requisito → Decisão → Teste

| Req (PRD) | Decisão (ADR) | Componente | Teste de aceitação |
|---|---|---|---|
| RP-01 input_mappings 4 CLIs | ADR-020 | `normalization-rules.yaml`, `normalizeInput` | golden por driver: campo canônico idêntico |
| RP-02 métricas mínimas unificadas | ADR-021 | `MetricSet`, `RenderMetricsSection` | report omite ausentes; sem campo divergente |
| RP-03 invariância cross-CLI | ADR-020/022 | `internal/parity` | mesma fixture → mesmo set de `normalized_name` + forma de evento (4 CLIs) |
| RP-04 alias Gemini explícito | ADR-020 | `aliases.gemini` | tabela explícita vence; teste de herança |
| RI-01 stack-aware install | ADR-024 | `install.Execute` + `DetectLangs` | repo Go/Node/Python → skill correta sem flag |
| RI-02 probe no install | ADR-024 | `install` + `probe.EnsureAvailable` | binário ausente → warning não-fatal |
| RI-03 stubs por CLI | ADR-024 | `installClaude/Codex/Gemini/Copilot` | reexecução idempotente; setup funcional |
| RI-04 verify cobre binário | ADR-024 | `Verify`/`VerifyItem` | item `binary` current/missing por CLI |
| RIN-01 wiring RuntimeConfig | ADR-025 | `BuildRuntimeConfig`, `taskloop` | precedência flags>workspace>global>defaults idêntica nas 4 CLIs |
| RIN-02 input-normalization completa | ADR-020 | `normalize.go` + YAML | cobertura Copilot/Gemini |
| RIN-03 harmonização de métricas | ADR-021 | `MetricsExtractor`, `report.go` | extractor por driver + fallback mínimo comum |
| RIN-04 janela grande | ADR-023 | `ContextWindow`, `WindowPolicy`, `TokenBudgetHook` | Gemini (large) não compacta em limites F1 |
| RIN-05 sunset legacy | ADR-026 | (plano) | depreciação reforçada; critério de remoção |
| RG-01 spec-hash em runtime | ADR-022 | `SpecDriftHook` | drift aborta antes de `c.Open` |
| RG-02 PRD-first enforce | ADR-022 | `SpecDriftHook` | `tasks.md` sem hash → `ErrPRDUntracked` |
| RG-03 invariantes ADR-008 em CI | ADR-022 | `internal/parity` + CI | gate obrigatório por CLI |
| RG-04 telemetria opt-in | (preservar) | `internal/telemetry` | sem `GOVERNANCE_TELEMETRY` ⇒ nada enviado |

## Pontos de Integração

- **CLIs ACP externas** (claude/codex/copilot/gemini) via `coder/acp-go-sdk` — sem mudança de transporte. Autenticação delegada à própria CLI (R-SEC-001: credenciais nunca no harness).
- **`npx` (fallback launcher)** — usado pelo probe (ADR-017); tratado como input não confiável (timeout curto, warning não-fatal).
- **Filesystem** — leitura de `prd.md`/`tasks.md` (drift guard), `config.yaml` (resolver), `normalization-rules.yaml` (override de projeto). Toda escrita auditável (R-SEC-001); paths via `filepath`/`os.UserHomeDir`, nunca hardcoded.

## Abordagem de Testes

### Testes Unitários

Componentes-chave (table-driven, `FakeFileSystem`, `io.Discard` em `Printer`):
- **`DriverID`**: parse válido/ inválido (`ErrUnknownDriver`), idempotência de `String()`.
- **`MetricSet`**: `Merge` associativo/comutativo nos contadores, `IsZero`, `Fields` só não-zero e ordenado.
- **`MetricsExtractor`**: cada extractor com payload completo, parcial e `usage` ausente → zero-value; `nullExtractor` sempre zero.
- **`SpecDriftHook`**: drift (`ErrSpecDrift`), `NoHashFound` (`ErrPRDUntracked`), `TasksDir==""` (no-op), `SkipDriftGuard` (bypass).
- **`BuildRuntimeConfig`**: timeout vazio→zero/F1, timeout válido→parse, timeout malformado→erro; precedência via `Resolver`.
- **`WindowPolicy`/`ContextWindow`**: standard vs large; zero-value→F1.
- **Normalização**: golden por driver para `aliases.gemini` e `input_mappings.copilot/gemini`; `RawName/RawInput` byte-identical; `--no-normalize` recupera pré-normalização.

Mock só para fronteiras externas (subprocesso ACP via `acpfake`/doubles existentes); domínio testado direto.

### Testes de Integração

> Avaliação dos critérios do template:
> - [x] Fronteiras de IO críticas (filesystem do install/verify, leitura de spec/tasks para drift) onde mocks não garantem correção.
> - [x] Risco real de "unit passa, integração falha": idempotência do install P/M/G e probe de binário dependem do FS/PATH reais.
> - [x] Custo proporcional: o projeto **já** usa build tag `integration` + `t.TempDir()` (sem testcontainers; não há Postgres/Redis/Kafka).
>
> Decisão: **sim** — manter e estender a suíte de integração existente com build tag `//go:build integration` e `t.TempDir()`. **Não** introduzir testcontainers (sem dependência de serviço externo).

- **Install P/M/G** (`internal/install/install_integration_test.go`): repo vazio (P), repo Go/Node/Python (M), monorepo multi-agente (G) → reexecução converge a 100% `current`; `verify` reporta skills + binário.
- **Probe não-fatal**: ambiente sem o binário → install conclui com warning; verify marca `missing`.

### Testes E2E / Paridade (RP-03, RG-03)

- **Suíte `internal/parity`** derivada de ADR-008: uma task fixture executada (via `acpfake`) contra os 4 drivers asserta igualdade do conjunto de `normalized_name` e da forma de evento. Promovida a **gate de CI obrigatório por CLI** (test.yml). Determinística — sem rede real (R-TEST-001).

## Sequenciamento de Desenvolvimento

### Ordem de Build

1. **Núcleo de domínio (sem IO):** `DriverID` + `ErrUnknownDriver` (ADR-020); `MetricSet` (ADR-021); `ContextWindow`/`WindowClass` (ADR-023). *Primeiro — sem dependências, habilita o resto.*
2. **Extractors + harmonização de métricas:** `MetricsExtractor`/`ExtractorFor`, refactor `runEventLoop` para `MetricSet` único, `Summary.Metrics`, `RenderMetricsSection` (ADR-021). *Depende de (1).*
3. **Paridade de normalização:** `aliases.gemini`, `input_mappings.copilot/gemini`, propagação de `DriverID` (ADR-020). *Depende de (1).*
4. **Guard de governança:** `SpecDriftHook` + wiring no dispatcher (ADR-022). *Independente; depende de `specdrift` (pronto).*
5. **Wiring RuntimeConfig:** `BuildRuntimeConfig` + integração no `taskloop` (ADR-025). *Depende de `resolver` (pronto).*
6. **Janela CLI-aware:** popular `ContextWindow` no catálogo, `TokenBudgetHook` por Spec, `WindowPolicy` na memória (ADR-023). *Depende de (1).*
7. **Install transparente:** stack-aware, probe não-fatal, stubs, verify de binário (ADR-024). *Independente das demais.*
8. **Suíte de paridade RP-03 + gate CI (RG-03)** e **plano de sunset (ADR-026, sem remoção).** *Por último — valida o conjunto.*

### Dependências Técnicas

- Nenhuma infra nova. Reuso de `specdrift`, `detect`, `probe`, `resolver`, `telemetry`, `acpfake` (todos presentes e testados).
- Sem novas dependências diretas no `go.mod`.

## Monitoramento e Observabilidade

- **Telemetria opt-in (ADR-006/RG-04):** `MetricSet` alimenta `internal/telemetry` apenas com `GOVERNANCE_TELEMETRY=1`; `IsZero()` ⇒ nada logado. Relatório via `ai-spec telemetry report`.
- **`execution_report.md`:** seção de métricas unificada (omite campos ausentes); seção `## Runtime ACP` inalterada.
- **Logs/stderr:** warnings não-fatais (probe ausente, unknown ACP events) seguem o padrão atual; erros de guard (`ErrSpecDrift`/`ErrPRDUntracked`) são acionáveis (apontam `ai-spec sync-spec-hash`/`check-spec-drift`).
- Sem métricas Prometheus/dashboards Grafana (CLI, não serviço long-running).

## Considerações Técnicas

### Decisões Chave

- [ADR-020](adr-020-driverid-vo-normalizacao-paridade.md) — `DriverID` VO + paridade de normalização (alias Gemini explícito + input_mappings Copilot/Gemini).
- [ADR-021](adr-021-metricset-vo-extractor-por-driver.md) — `MetricSet` VO + `MetricsExtractor` por driver + render mínimo-comum.
- [ADR-022](adr-022-guard-governanca-runtime-spec-hash.md) — guard de governança em runtime (spec-hash/drift + PRD-first) no `runtime.pre_open`.
- [ADR-023](adr-023-window-policy-cli-aware.md) — política de janela CLI-aware via campo estático na `Spec`.
- [ADR-024](adr-024-instalacao-transparente-stack-aware.md) — instalação transparente stack-aware (probe, stubs, verify de binário).
- [ADR-025](adr-025-runtimeconfig-wiring-acprunner.md) — wiring do `RuntimeConfig` hierárquico no `ACPRunner`.
- [ADR-026](adr-026-sunset-legacy-mode.md) — sunset planejado do legacy mode (sem remoção neste ciclo).

### Object Calisthenics (heurísticas aplicadas)

Tratadas como heurística, não dogma (`object-calisthenics-go/references/rules.md`); aplicadas só onde reduzem complexidade real. `go-implementation` prevalece em conflito (governança R-GOV-001).

| Regra OC | Aplicação nesta spec | Onde **não** aplicar (registro) |
|---|---|---|
| #3 Encapsular primitivos de domínio | `DriverID`, `WindowClass` (string crua → VO com regra) | Não encapsular `Timeout`/contadores além de `MetricSet` — `RuntimeConfig` já é DTO coeso |
| #4 Coleções de primeira classe | `MetricSet` substitui 11 campos planos de métrica no `Summary` | `[]VerifyItem`, `[]string` triviais permanecem slices |
| #2 Early return / sem `else` | `ParseDriverID`, `SpecDriftHook.Run`, `BuildRuntimeConfig` (guard clauses) | — |
| #1 Uma indentação por função | `runEventLoop` perde 8 acumuladores ao delegar a `MetricSet.Merge` (menos branching) | Não fragmentar `Run()` além dos helpers já extraídos |
| #6 Nomes não-opacos | `DriverID`, `ContextWindow`, `WindowPolicy`, `MetricsExtractor` (papéis explícitos) | — |
| #7/#8 Entidades pequenas / poucos campos | `Summary` para de crescer por driver (1 campo `Metrics` vs 8) | DTOs (`Job`, `Spec`, `config.Runtime`) isentos — representam estado/config real |
| #9 Sem getters/setters mecânicos | `MetricSet.Fields()`/`Merge()` (comportamento, não getter); campos não exportados | Getters de catálogo (`SDKVersion()`, `DriverID()`) aceitos: contrato público estável do pacote |

Interromper a aplicação quando exigir quebra de API pública ou aumentar indireção sem reduzir acoplamento (ex.: não criar VO para cada contador isolado).

### Riscos Conhecidos

| Risco | Impacto | Mitigação |
|---|---|---|
| Refactor de `Summary` (8 campos → `MetricSet`) toca `persistence`/`evidence` | Médio — regressão em relatórios | Troca mecânica; golden tests de report por driver; header Claude compatível |
| `SpecDriftHook` falso-positivo em uso ad-hoc (sem PRD) | Médio — bloqueia sessão legítima | No-op quando `Job.TasksDir==""`; `SkipDriftGuard`; `--disable-hooks` |
| Probe no install lento sem rede (npx) | Baixo — viola RF-11 (<30s) | Timeout curto; degrada para warning; nunca bloqueia |
| Janela estática desatualizada quando provider muda | Baixo — subutiliza/excede janela | Dado de catálogo versionado; override por config; revisão a cada bump de SDK/CLI |
| Drift entre `config.Runtime` e `runtime.RuntimeConfig` | Baixo | Mapeamento e normalização centralizados (`BuildRuntimeConfig`/`ApplyDefaults`) + teste |
| Override de projeto `normalization-rules.yaml` sem novas chaves | Baixo | Loader tolerante (passthrough); suíte RP-03 roda contra embedded default |
| "Depreciação eterna" do legacy (RIN-05) | Baixo | Critério objetivo de remoção em ADR-026 |

### Conformidade com Padrões

Regras de `.claude/rules/` e referências de governança aplicáveis:
- **R-GOV-001** (governança/precedência): `go-implementation` > `object-calisthenics-go` em conflito; OC como heurística de revisão.
- **R-DDD-001**: VOs autovalidados e imutáveis; agregado `Spec` só via catálogo; fail-fast; domínio sem IO/serialização.
- **R-ERR-001**: sentinelas tipados (`ErrUnknownDriver`, `ErrSpecDrift`, `ErrPRDUntracked`) + `fmt.Errorf("...: %w", err)`; validação na fronteira (`BuildRuntimeConfig`); nada de `panic` recuperável; mensagens PT-BR acionáveis.
- **R-SEC-001**: credenciais delegadas à CLI; paths via `filepath`/`os.UserHomeDir`; input externo (payload ACP, npx, arquivos) tratado como não confiável; sem segredos em logs/telemetria.
- **R-TEST-001**: table-driven; `FakeFileSystem` (unit) e `t.TempDir()` (integration); determinístico (sem rede/sleep); caminho feliz + falha; cobertura ≥ 75% (CI).
- **AGENTS.md** (invariantes): PRD-first (reforçado por ADR-022); spec-hash (este header + RG-01); evidência obrigatória; isolamento de contexto.

### Arquivos Relevantes e Dependentes

**Domínio (novos/modificados):**
- `internal/runtime/specs/driver.go` (N), `internal/runtime/specs/window.go` (N), `internal/runtime/specs/spec.go` (M), `internal/runtime/specs/{claude,codex,copilot,gemini}.go` (M — popular `ContextWindow`)
- `internal/runtime/events/metricset.go` (N), `internal/runtime/events/extractor.go` (N), `internal/runtime/events/convert.go` (M), `internal/runtime/events/gemini_metrics.go` (M), `internal/runtime/events/normalize.go` (M), `internal/runtime/events/normalization-rules.yaml` (M)
- `internal/runtime/memory/window_policy.go` (N), `internal/runtime/memory/store.go` (M)

**Aplicação:**
- `internal/runtime/runner.go` (M — `DriverID`, `MetricSet`, hook, window), `internal/runtime/summary.go` (M), `internal/runtime/types.go` (M — `Job.SkipDriftGuard`)
- `internal/runtime/hooks/spec_drift.go` (N), `internal/runtime/hooks/token_budget.go` (M), `internal/runtime/hooks/dispatcher.go` (uso)
- `internal/runtime/persistence/report.go` (M — `RenderMetricsSection`)
- `internal/taskloop/runtimeconfig.go` (N — `BuildRuntimeConfig`), `internal/taskloop/runloop.go` (M)
- `internal/install/install.go` (M — stack-aware, probe, verify)

**Infra reusada (sem mudança de contrato):**
- `internal/specdrift/specdrift.go`, `internal/detect/{detect,architecture}.go`, `internal/runtime/probe/probe.go`, `internal/config/resolver.go`, `internal/telemetry/telemetry.go`, `internal/runtime/acpfake/server.go`

**Testes (novos/estendidos):**
- `internal/runtime/specs/driver_test.go`, `internal/runtime/events/metricset_test.go`, `internal/runtime/events/extractor_test.go`, `internal/runtime/hooks/spec_drift_test.go`, `internal/taskloop/runtimeconfig_test.go`, `internal/runtime/memory/window_policy_test.go`, `internal/parity/*` (RP-03), `internal/install/install_integration_test.go` (P/M/G)

**Governança/docs a atualizar:**
- `AGENTS.md` (Normalização, Métricas, Instalação, invariante 2), `CLAUDE.md`, `docs/config-hierarchy.md`, `docs/guia-instalacao-universal.md`, `docs/troubleshooting.md`
