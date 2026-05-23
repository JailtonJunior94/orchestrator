<!-- spec-hash-prd: af3468061964d963ab6d18795514e4063438208b7f929baea5b62ae35dea160a -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Fundação Portátil do ai-spec-harness

## Resumo Executivo

O `ai-spec-harness` já orquestra os 4 CLIs ACP (claude, codex, gemini, copilot) via subprocess
stdio, com `Spec` por runtime, normalização de tool-calls, memória 2-tier, hooks e persistência
file-first. Esta especificação fecha o gap de **maturidade de plataforma** identificado contra o par
arquitetural Compozy, em três frentes que **reusam ao máximo o código existente** em vez de
reescrever: (1) **robustez de runtime** — generalizar a cadeia de fallback launchers, unificar os
parâmetros operacionais num `RuntimeConfig`, adicionar retry/backoff na orquestração e tornar a
sessão ACP observável sob backpressure; (2) **instalador portátil** — auto-detecção de agentes por
binário no PATH, escopo global, e `verify` com estados `current/missing/drifted` reusando o
comparador de checksum do `internal/upgrade`; (3) **config hierárquico** — camada global
`~/.aispec/config.yaml` + projeto com upward-walk e precedência determinística.

A estratégia central é **composição e defaults inertes**: todo novo parâmetro tem zero-value que
preserva o comportamento atual (F1), garantindo regressão zero. A validação de paridade estende o
framework `internal/parity` existente para uma matriz 4×4 cross-CLI + um teste cross-project.
Decisões materiais estão em [ADR-016](adr-016-config-hierarquico-universal.md),
[ADR-017](adr-017-fallback-launcher-chain.md),
[ADR-018](adr-018-runtimeconfig-retry-backpressure.md) e
[ADR-019](adr-019-instalador-portatil-detect-verify.md).

## Arquitetura do Sistema

### Visão Geral dos Componentes

**Modificados:**
- `internal/runtime/specs/spec.go` — `FallbackLauncher` tratado como launcher genérico (mantém forma).
- `internal/runtime/probe/probe.go` — `resolve` itera `spec.Fallbacks` com `NewBinaryLauncher`
  (remove npx-only `extractPackage/extractVersion`). [ADR-017]
- `internal/runtime/types.go` — `Job` **embute** `RuntimeConfig` (composição). [ADR-018]
- `internal/runtime/client/client.go` — backpressure com timeout + contadores
  `slowPublishes/droppedUpdates`; capacidade de canal configurável. [ADR-018]
- `internal/runtime/runner.go` — `Summary` carrega contadores de publicação; `Client` expõe métricas.
- `internal/taskloop/taskloop.go` (`Options`) + `runloop.go` — pool por `Concurrent`/`BatchSize` e
  loop de retry/backoff no invoker (`acpinvoker.go`). [ADR-018]
- `internal/config/runtime.go` — `LoadRuntime` reescrito como wrapper sobre novo `Resolver`. [ADR-016]
- `internal/config/config.go` — `InstallOptions` ganha `Scope`; `Tools` opcional. [ADR-019]
- `internal/install/install.go` — detecção quando `Tools` vazio; escopo global; `Verify`. [ADR-019]
- `cmd/ai_spec_harness/install.go` — `--tools` opcional, `--global`; novo/estendido `verify`. [ADR-019]
- `internal/parity/parity.go` + `parity_test.go` — matriz 4×4 + invariante de fallback + cross-project. [PRD RF-18/19]

**Novos:**
- `internal/config/resolver.go` — `Resolver` (home + upward-walk + merge campo-a-campo). [ADR-016]
- `internal/detect/agent.go` — `AgentDetector` (binário no PATH + dirs de config + arquivos). [ADR-019]
- `internal/runtime/retry.go` (ou em `taskloop`) — classificação transitório/fatal + backoff. [ADR-018]

### Fluxo de Dados

1. **Resolução de config:** CLI start → `config.Resolver.Resolve(cwd, flagOverrides)` →
   built-in ⊕ global(`~/.aispec`) ⊕ projeto(upward-walk) ⊕ flags → `config.Runtime`.
2. **Install/bootstrap:** `Tools` vazio → `AgentDetector.Detect()` → agentes alvo →
   `install.Service.Execute(Scope, Mode)` → assets (go:embed) materializados → manifesto.
   `Verify` → comparador de checksum → estados por skill/agente.
3. **Execução:** `taskloop.Options` → monta `Job{RuntimeConfig}` → `acpInvoker` (loop retry/backoff)
   → `ACPRunner.Run` → `probe.EnsureAvailable` (canônico→fallbacks) → `client.Open` → loop de eventos
   com backpressure observável → `Summary` (inclui slow/dropped/retry).

## Design de Implementação

### Interfaces Chave

```go
// internal/config/resolver.go — ADR-016
type Resolver interface {
    // Resolve aplica precedência built-in < global(~/.aispec) < projeto(upward-walk) < overrides.
    Resolve(ctx context.Context, cwd string, overrides Runtime) (Runtime, error)
}

// internal/detect/agent.go — ADR-019
type AgentDetector interface {
    // Detect retorna os agentes presentes (binário ACP no PATH e/ou dirs/arquivos de config).
    Detect(ctx context.Context, opts DetectOptions) ([]skills.Tool, error)
}

// internal/install — ADR-019 (verify file-first)
type VerifyState string // "current" | "missing" | "drifted"
type VerifyItem struct {
    Tool  skills.Tool
    Skill string
    State VerifyState
}
func (s *Service) Verify(opts config.InstallOptions) ([]VerifyItem, error)

// internal/runtime/client — ADR-018 (sessão observável)
type Client interface {
    Open(ctx context.Context, launcher specs.Launcher, prompt string) error
    Updates() <-chan events.Event
    Err() error
    Close() error
    SlowPublishes() uint64   // novo
    DroppedUpdates() uint64   // novo
}

// internal/runtime/retry.go — ADR-018
type RetryClassifier interface {
    // IsTransient decide se um erro de sessão é reexecutável (ex.: launcher/IO/inatividade)
    // — falhas como ErrPermissionDenied retornam false.
    IsTransient(err error) bool
}
```

### Modelos de Dados

```go
// internal/runtime/types.go — ADR-018 (composição; defaults inertes preservam F1)
type RuntimeConfig struct {
    Timeout                events.ActivityTimeout // mapeia ActivityTimeout atual
    MaxRetries             int                    // 0 = uma tentativa (F1)
    RetryBackoffMultiplier float64                // <=0 => sem espera
    Concurrent             int                    // <=0 => 1 (sequencial, F1)
    BatchSize              int                    // <=0 => 1 (F1)
}
func (c *RuntimeConfig) ApplyDefaults()

type Job struct {
    Prompt, WorkDir, EvidenceDir string
    RuntimeConfig                // EMBUTIDO (substitui ActivityTimeout solto via mapeamento)
    Model, ReasoningEffort string
    AccessMode specs.AccessMode
    AddDirs []string
    // ... campos F2–F5 preservados: TaskFileName, MCPNested, NoNormalize,
    //     MemoryLimits, DisableHooks, TasksDir, AutoReview, Quiet
}

// internal/config/runtime.go — ADR-016 (novas chaves operacionais opcionais)
type Runtime struct {
    TasksRoot, PRDPrefix, EvidenceDir string
    CoverageThreshold float64
    LanguageDefault string
    // novas (zero-value => default/F1):
    Timeout string  `yaml:"timeout"`
    MaxRetries int  `yaml:"max_retries"`
    RetryBackoffMultiplier float64 `yaml:"retry_backoff_multiplier"`
    Concurrent int `yaml:"concurrent"`
    BatchSize int  `yaml:"batch_size"`
    DefaultTool string `yaml:"default_tool"`
}

// internal/config/config.go — ADR-019
type InstallScope string // "project" (default) | "global"
type InstallOptions struct {
    ProjectDir, SourceDir string
    Tools []skills.Tool   // OPCIONAL: vazio => auto-detect
    Langs []skills.Lang
    LinkMode skills.LinkMode
    Scope InstallScope     // novo
    DryRun, GenerateCtx bool
    CodexProfile string
    FocusPaths []string
}
```

### Endpoints de API

Não aplicável (CLI/biblioteca Go). Superfície CLI afetada:
- `ai-spec install [path] [--tools ...] [--global] [--mode symlink|copy] [--dry-run]` — `--tools` opcional.
- `ai-spec verify [path] [--global]` — reporta `current/missing/drifted` (novo ou estende `inspect`).

## Pontos de Integração

- **`os.UserHomeDir`** para `~/.aispec/` e dirs globais por-agente — paths normalizados/validados,
  nunca hardcoded (R-SEC-001). Ausência de `$HOME` degrada (global opt-in) sem abortar projeto.
- **Binários ACP no PATH** (`exec.LookPath` via `internal/runtime/specs`/`probe`) como sinal primário
  de detecção de agente — sem executar os binários (LookPath apenas).
- **`coder/acp-go-sdk`** inalterado; mudanças de sessão ficam no wrapper `internal/runtime/client`.
- **Manifesto `.ai_spec_harness.json`** e comparador de checksum de `internal/upgrade` reusados pelo `Verify`.

## Abordagem de Testes

### Testes Unitários
- **config.Resolver:** table-driven — precedência (flags>projeto>global>default), upward-walk de
  subdiretório, ausência de global, arquivo malformado (erro), regressão (== `LoadRuntime` legado).
  FakeFileSystem; injetar home dir.
- **probe.resolve (ADR-017):** canônico presente; canônico ausente + fallback genérico (FixedArgs
  literais); cadeia de múltiplos fallbacks; argv idêntico ao baseline por spec (RF-05).
- **RuntimeConfig.ApplyDefaults / retry (ADR-018):** defaults inertes (F1); erro transitório injetado
  reexecuta até `MaxRetries` com backoff; `ErrPermissionDenied` não reexecuta.
- **client backpressure (ADR-018):** canal cheio com timeout=0 → drop + `droppedUpdates++`;
  timeout>0 → aguarda e contabiliza `slowPublishes`; defaults preservam contagem de eventos atual.
- **AgentDetector (ADR-019):** binário no PATH (LookPath fake) detecta; só arquivo de projeto;
  repo vazio sem binários → vazio; flag override.
- **install.Verify (ADR-019):** mapeamento `StatusOK→current`, `Missing→missing`,
  `Outdated|ContentDivergent→drifted`; install→install→verify == 100% current.

### Testes de Integração
> Critérios avaliados: o harness tem fronteira de IO crítica (filesystem real, symlinks, PATH) onde
> FakeFileSystem não cobre symlink/permites reais → **sim**; risco de divergência unit↔real em
> symlink/UserHomeDir → **sim**. **Integration tests recomendados** (build tag `integration`,
> `t.TempDir()`), sem testcontainers (não há banco/fila).

- **Bootstrap portátil (RF-11):** `t.TempDir()` vazio + PATH com binários fake → `install` sem
  `--tools` → `verify` 100% `current`; medir tempo < 30s.
- **Idempotência (RF-09):** install 2× no mesmo TempDir → verify 100% `current`; symlinks válidos.
- **Escopo global (RF-07):** `HOME` apontado para TempDir → install `--global` materializa em `~/.aispec`/dirs globais.

### Testes E2E
- **Paridade cross-CLI 4×4 (RF-18):** estender `internal/parity` para gerar combinações table-driven
  das 4 tools (e subconjuntos) e validar invariantes em cada uma.
- **Cross-project (RF-18):** novo `internal/parity/e2e_parity_test.go` — instalar num repo temporário
  e validar invariantes via `parity.Invariants()`.
- **Fallback launcher (RF-19):** invariante/teste — binário direto ausente → launcher fallback →
  resultado idêntico ao do binário direto.

## Sequenciamento de Desenvolvimento

### Ordem de Build
1. **config.Resolver (ADR-016)** — base transversal; nada depende de execução. Wrapper `LoadRuntime`.
2. **Fallback genérico (ADR-017)** — isolado em `probe`/`specs`; habilita RF-19.
3. **RuntimeConfig + backpressure + retry (ADR-018)** — depende de (1) para chaves de config;
   composição em `Job`, contadores no client, retry no invoker, pool no runloop.
4. **Instalador portátil (ADR-019)** — `AgentDetector`, escopo global, `Verify`; depende de (1) para
   precedência de `--tools`.
5. **Paridade 4×4 + cross-project (RF-18/19)** — valida (2),(3),(4) end-to-end.
6. **Guia de Instalação Universal + docs/ADR sync** — entregável final.

### Dependências Técnicas
- Nenhuma infra externa nova. Go 1.26+, cobra, yaml.v3 (já presentes). Binários ACP no PATH apenas
  para testes de integração (mockáveis via LookPath fake nos unitários).

## Monitoramento e Observabilidade
- `Summary`/telemetria opt-in (ADR-006) ganham `slow_publishes`, `dropped_updates`, `retry_attempts`.
- Logs: tentativas de retry (nível warn), fallback acionado (info, já em `runtime_init`), agentes
  detectados no install (info), itens `drifted`/`missing` no verify (warn).
- Métricas Prometheus não se aplicam (CLI local); telemetria continua append-only opt-in.

## Considerações Técnicas

### Decisões Chave
- [ADR-016] Config hierárquico universal (global+projeto, upward-walk, precedência, YAML, file-first).
- [ADR-017] Fallback launcher chain genérico (remove npx-only).
- [ADR-018] RuntimeConfig por composição + retry/backoff na orquestração + sessão backpressure observável.
- [ADR-019] Instalador com auto-detecção por binário, escopo global e verify reusando checksum do upgrade.

### Riscos Conhecidos
- **Não-determinismo por concorrência:** mitigado por default sequencial (`Concurrent=1`) e respeito
  a dependências de tasks.
- **Retry sobre efeito não-idempotente:** retry restrito à fase de sessão ACP, antes de efeitos
  persistentes irreversíveis; classificador transitório/fatal explícito.
- **Mudança de timing do canal de eventos:** defaults (cap 64, timeout 0) preservam comportamento;
  teste de regressão de contagem de eventos.
- **`$HOME` ausente em CI:** global é opt-in e degrada com erro explícito; projeto permanece default.
- **Detecção de agente falso-positivo:** `verify` valida a pós-condição; install registra o que foi feito.

### Conformidade com Padrões
- `.claude/rules/governance.md` (R-GOV-001): precedência de regras, evidência obrigatória.
- `security.md` (R-SEC-001): paths via `os.UserHomeDir` normalizados/validados, sem hardcode; subprocess
  com args explícitos (já vigente em `client.startProcess`); inputs externos (config) parseados/validados.
- AGENTS.md: PT-BR; `fmt.Errorf("ctx: %w", err)`; DI por construtor, zero estado global; interfaces no
  consumidor; testes table-driven + FakeFileSystem (unit) e `t.TempDir()` (integration); cobertura ≥ 75%.
- ADR-001 (go:embed) preservado como fonte de assets; ADR-006 (telemetria opt-in) para novos contadores.

### Arquivos Relevantes e Dependentes
- Runtime: `internal/runtime/specs/spec.go`, `claude.go`, `codex.go`, `gemini.go`, `copilot.go`;
  `internal/runtime/probe/probe.go`; `internal/runtime/types.go`; `internal/runtime/client/client.go`;
  `internal/runtime/runner.go`; (novo) `internal/runtime/retry.go`.
- Taskloop: `internal/taskloop/taskloop.go`, `runloop.go`, `acpinvoker.go`.
- Config: `internal/config/runtime.go`, `config.go`; (novo) `internal/config/resolver.go`.
- Install/detect: `internal/install/install.go`; `internal/detect/detect.go`; (novo)
  `internal/detect/agent.go`; `internal/upgrade/*` (checksum reuse); `cmd/ai_spec_harness/install.go`,
  `inspect.go`, `doctor.go`.
- Assets/skills: `internal/embedded/embedded.go`, `internal/adapters/adapters.go`, `internal/skills/skills.go`.
- Paridade: `internal/parity/parity.go`, `parity_test.go`; (novo) `internal/parity/e2e_parity_test.go`.
- Docs: `AGENTS.md`, `CLAUDE.md`, `docs/` (hierarquia de config + guia de instalação universal).
