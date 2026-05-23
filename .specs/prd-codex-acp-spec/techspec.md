<!-- spec-hash-prd: TBD-via-make-sync-spec-hash -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Codex CLI via ACP Nativo

> **PRD consumido**: [prd.md](./prd.md) (spec-version 1)
> **ADR material**: [013-codex-cli-acp-native](../adr/013-codex-cli-acp-native.md)
> **Precedente direto**: [F1-Copilot techspec](../prd-copilot-acp-spec/techspec.md) — generalização runtime base
> **Insumo de pesquisa**: [docs/research/compozy-adaptation-codex-2026.md](../../docs/research/compozy-adaptation-codex-2026.md)
> **Fase**: 1 de 4 (Compozy 2026 — variante Codex)

## Resumo Executivo

Introduz suporte ao Codex CLI como runtime ACP nativo via novo construtor `specs.Codex()` em `internal/runtime/specs/codex.go`, invocando o adapter `codex-acp` (`@zed-industries/codex-acp@0.14.0`) — distinto do CLI legado `codex` da OpenAI. Espelha estruturalmente o padrão de `claude.go:28-45` e `copilot.go:33-50`, e o registro canônico do compozy em `internal/core/agent/registry_specs.go:106-122`.

A integração reaproveita 100% da infraestrutura ACP existente (`ACPRunner`, `acpClient`, `SessionPersistence`, `ActivityWatchdog`, `events`) já generalizada por F1-Copilot. **A única extensão estrutural** é o suporte a `BootstrapArgs` dinâmico:

1. `internal/runtime/specs/spec.go:12-23` ganha tipo `AccessMode string` (consts `AccessModeRestricted`, `AccessModeFull`) e método `BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string` com default no-op (retorna `nil`). Mudança retrocompatível: Claude/Copilot herdam no-op sem alteração.

2. `internal/runtime/runner.go::Job` ganha campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string`.

3. `internal/runtime/runner.go` em `Run()` chama `spec.BootstrapArgs(...)` e prepend ao argv antes das `FixedArgs`.

4. `cmd/ai_spec_harness/task_loop.go` ganha flags `--reasoning-effort` (default `medium`) e `--access-mode` (default `restricted`), propagadas via `taskloop.Options` até `Job`.

5. `cmd/ai_spec_harness/task_loop.go:21-24` adiciona `"codex": specs.Codex` em `runtimeACPCatalog`. T-14 invertido. T-15 novo cobre as flags.

6. `internal/runtime/probe/probe.go:21-24` adiciona `"codex": ".specs/adr/013-codex-cli-acp-native.md"` em `adrByID`.

ADR-013 declara as decisões. `CODEX.md` raiz é reescrito (esqueleto de 30 linhas → ~80 linhas). `codexInvoker` legado em `internal/taskloop/agent.go:335-351` permanece por 2 versões minor com aviso de depreciação via `sync.Once`. Persistência forense, watchdog e tagged union de eventos (ADR-010) permanecem **inalterados em comportamento**.

## Arquitetura do Sistema

### Visão Geral dos Componentes

**Novos**
- `internal/runtime/specs/codex.go` — Construtor `Codex() Spec` + constantes `CodexNpmPackage`, `CodexNpmVersion="0.14.0"`, `CodexSDKVersion`, `DefaultCodexModel="gpt-5.5"`, `CodexMinNpmVersion="0.12.0"` + função local `codexBootstrapArgs`.
- `internal/runtime/specs/codex_test.go` — Testes de defaults, fallback, pinning, matriz `codexBootstrapArgs`.
- `.specs/adr/013-codex-cli-acp-native.md` — **já entregue** declarando decisão F1-Codex.

**Modificados**
- `internal/runtime/specs/spec.go` — Adicionar tipo `AccessMode string` (`AccessModeRestricted`, `AccessModeFull`); estender `Spec` com campo privado `bootstrapArgs func(string, string, []string, AccessMode) []string` e método público `Spec.BootstrapArgs(...)`. Assinatura de `newSpec` estendida (variant `newSpecWithBootstrap` ou parâmetro opcional). Chamadores existentes (`Claude()`, `Copilot()`) atualizados — passam `nil` para o bootstrap.
- `internal/runtime/specs/claude.go` — Atualizar chamada a `newSpec(...)` passando `bootstrapArgs=nil` (default no-op). Comportamento idêntico ao atual.
- `internal/runtime/specs/copilot.go` — Idem. Passa `bootstrapArgs=nil`.
- `internal/runtime/runner.go` — `ACPRunner.Run()` chama `r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)` e prepend ao argv do launcher antes das `FixedArgs`. Helper `buildRuntimeInitRaw` permanece inalterado (já generalizado em F1-Copilot).
- `internal/runtime/runner.go::Job` — Adicionar campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string`.
- `internal/runtime/probe/probe.go:21-24` — Adicionar `"codex": ".specs/adr/013-codex-cli-acp-native.md"` em `adrByID`.
- `cmd/ai_spec_harness/task_loop.go:21-24` — Adicionar `"codex": specs.Codex` em `runtimeACPCatalog`.
- `cmd/ai_spec_harness/task_loop.go` — Registrar flags `--reasoning-effort` (default `"medium"`, validar enum `low|medium|high`) e `--access-mode` (default `"restricted"`, validar enum `restricted|full`). Propagar via `taskloop.Options.ReasoningEffort`, `Options.AccessMode`, `Options.AddDirs`. Emitir warning único em stderr se `--access-mode=full` for usado.
- `cmd/ai_spec_harness/task_loop_test.go:48-52` — **Inverter T-14** (passa a esperar `wantErr: false` para `--tool codex --runtime acp`). Adicionar T-15 cobrindo combinações com `--reasoning-effort` e `--access-mode`.
- `internal/taskloop/taskloop.go::Service.Execute` — Propagar `Options.ReasoningEffort/AccessMode/AddDirs` para `Job`. Resolver Spec via `runtimeACPCatalog` quando `Runtime == "acp"`.
- `internal/taskloop/agent.go:335-351` — `codexInvoker.Invoke` ganha `sync.Once` para warning de depreciação único por execução do processo (espelhando `copilotInvoker` em F1-Copilot).
- `internal/runtime/acp_integration_test.go` — Sub-suite Codex reusando fake ACP server existente.
- `CODEX.md` raiz — Reescrita conforme [`docs/research/compozy-adaptation-codex-2026.md`](../../docs/research/compozy-adaptation-codex-2026.md) §"Exemplos de Configuração 2026".
- `AGENTS.md` — Tabela de ADRs ganha linha ADR-013.
- `docs/cli-schema.json` — Adicionar enums de `--reasoning-effort` (`low|medium|high`) e `--access-mode` (`restricted|full`).
- `docs/telemetry-feedback-cycle.md` — Documentar cobertura Codex.
- `internal/taskloop/compatibility.go::CompatibilityTable` — Entrada para `codex → [gpt-5.5]` (mínima; outros modelos via `--allow-unknown-model`).

**Inalterados (invariante de fase)**
- `internal/runtime/persistence/*` — Persistência forense intacta.
- `internal/runtime/watchdog.go` — `ActivityWatchdog` intacto.
- `internal/runtime/client/client.go` — `acpClient` intacto.
- `internal/runtime/client/client_test.go` — fake server intacto (reusado pela sub-suite Codex).
- `internal/runtime/events/*` — Eventos e conversão SDK→domínio intactos. Tagged union ADR-010 preservado: nenhum kind novo.
- `internal/runtime/specs/claude_test.go`, `copilot_test.go`, `spec_test.go` — Testes existentes passam sem alteração (default no-op).

### Relacionamentos e Fluxo de Dados

```
CLI: --tool codex --runtime acp --reasoning-effort high --access-mode full
         │
         ▼
cmd/ai_spec_harness/task_loop.go
   • valida --runtime in {legacy, acp}
   • valida --tool in runtimeACPCatalog quando runtime==acp
   • valida --reasoning-effort in {low, medium, high}
   • valida --access-mode in {restricted, full}
   • se --access-mode=full → emitir warning único em stderr
   • monta taskloop.Options{Tool: "codex", Runtime: "acp",
                            ReasoningEffort: "high", AccessMode: AccessModeFull, ...}
         │
         ▼
taskloop.Service.Execute(opts)
   • opts.Runtime == "acp" →
      specCtor := runtimeACPCatalog[opts.Tool]
      spec := specCtor()  // specs.Codex()
      runner := airuntime.NewACPRunner(spec, opts...)
      job := Job{
        Model:           opts.ExecutorModel,        // "gpt-5.5" (default)
        ReasoningEffort: opts.ReasoningEffort,      // "high"
        AccessMode:      opts.AccessMode,           // AccessModeFull
        AddDirs:         opts.AddDirs,
        ...
      }
   • opts.Runtime == "legacy" → codexInvoker (com WARNING uma vez)
         │
         ▼
airuntime.ACPRunner.Run(ctx, job)
   • probe.EnsureAvailable(ctx, spec) →
        - tenta spec.Command no PATH ("codex-acp")
        - fallback spec.Fallbacks (npx --yes @zed-industries/codex-acp@0.14.0)
        - erro template usa spec.NPMPackage()/NPMVersion() + ADR-013
   • bootstrap := spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)
     // Para Codex: ["-c", `model="gpt-5.5"`, "-c", `model_reasoning_effort="high"`,
     //              "-c", "features.code_mode=false", "-c", "features.code_mode_only=false",
     //              "-c", `approval_policy="never"`, "-c", `sandbox_mode="danger-full-access"`,
     //              "-c", `web_search="live"`]
     // Para Claude/Copilot: nil
   • argv := append(bootstrap, spec.FixedArgs...)
   • cmd := exec.Command(launcher.Command, append(launcher.Args, argv...)...)
   • buildRuntimeInitRaw(launcher.Kind(), launcherCmd, launcherArgs, spec.SDKVersion(), spec.NPMVersion())
   • events.NewRuntimeInit(...) — já generalizado em F1-Copilot
   • acpClient.Open + fan-out (inalterado)
   • ActivityWatchdog (inalterado)
   • SessionPersistence (inalterado)
         │
         ▼
audit/<run>/events.jsonl + tool_calls.md + execution_report.md
   (mesmos campos e granularidade que Claude/Copilot)
```

## Design de Implementação

### Interfaces Chave

```go
// internal/runtime/specs/spec.go — extensão de Spec
// Tipo novo: AccessMode
type AccessMode string

const (
    AccessModeRestricted AccessMode = "restricted"
    AccessModeFull       AccessMode = "full"
)

// BootstrapArgsFunc é a assinatura para spec-specific bootstrap arg builders.
// Default no-op (nil) preserva comportamento de Claude/Copilot.
type BootstrapArgsFunc func(model, reasoning string, addDirs []string, mode AccessMode) []string

// Spec estendido — campo privado adicional
type Spec struct {
    ID             string
    DisplayName    string
    Command        string
    FixedArgs      []string
    Fallbacks      []FallbackLauncher
    AccessModeFlag string
    sdkVersion     string
    npmVersion     string
    npmPackage     string
    // NOVO — Codex usa; Claude/Copilot preserva nil (no-op)
    bootstrapArgs BootstrapArgsFunc
}

// BootstrapArgs retorna args a serem prepended ao argv do launcher.
// Default (Claude/Copilot): retorna nil — sem prepend.
// Codex: retorna ["-c", "model=\"...\"", ...] via codexBootstrapArgs.
func (s Spec) BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string {
    if s.bootstrapArgs == nil {
        return nil
    }
    return s.bootstrapArgs(model, reasoning, addDirs, mode)
}

// newSpecWithBootstrap — variant constructor para specs com bootstrap dinâmico.
// Mantém newSpec original sem mudança de assinatura (preserva Claude/Copilot).
func newSpecWithBootstrap(
    id, displayName, command string,
    fixedArgs []string,
    fallbacks []FallbackLauncher,
    accessModeFlag string,
    sdkVersion, npmVersion, npmPackage string,
    bootstrapArgs BootstrapArgsFunc, // pode ser nil
) Spec {
    s := newSpec(id, displayName, command, fixedArgs, fallbacks, accessModeFlag,
                 sdkVersion, npmVersion, npmPackage)
    s.bootstrapArgs = bootstrapArgs
    return s
}
```

```go
// internal/runtime/specs/codex.go — novo arquivo
package specs

import "strconv"

// Constantes do runtime Codex via ACP adapter @zed-industries/codex-acp.
// Política de atualização (ADR-009 + ADR-013):
//   - CodexNpmVersion e CodexSDKVersion são constantes Go pinadas. Nunca usar @latest.
//   - CodexNpmVersion só é alterada via processo audit/ (.specs/templates/skill-upgrade-decision.md).
//   - CodexSDKVersion é mantida em sincronia com go.mod por scripts/sync-acp-sdk-version.sh.
const (
    // CodexNpmPackage é o nome do pacote npm canônico do Codex ACP adapter.
    // NÃO confundir com `codex` (CLI legacy da OpenAI). codex-acp é o adapter da Zed Industries.
    CodexNpmPackage = "@zed-industries/codex-acp"

    // CodexNpmVersion é a versão npm pinada do @zed-industries/codex-acp.
    // Pinada conforme ADR-013 D-06: constante Go atualizada somente via audit/.
    // Validada via `npm view @zed-industries/codex-acp versions` em 2026-05-21:
    // 0.14.0 é o último stable; 0.12.0 é o mínimo para gpt-5.5 (gating do compozy).
    CodexNpmVersion = "0.14.0"

    // CodexMinNpmVersion é a versão mínima do codex-acp que suporta gpt-5.5
    // (gating documentado por compozy/internal/core/agent/registry_compat.go::codexModelRequirements).
    // Informacional; probe não valida versão runtime.
    CodexMinNpmVersion = "0.12.0"

    // CodexSDKVersion é a versão do coder/acp-go-sdk sincronizada com go.mod.
    // Mesma do Claude/Copilot. Não editar manualmente — use make sync-acp-sdk-version.
    CodexSDKVersion = "v0.13.0"

    // DefaultCodexModel é o modelo default quando --model não é passado.
    // Espelha compozy/internal/core/model/constants.go:15.
    DefaultCodexModel = "gpt-5.5"
)

// Codex retorna a Spec do runtime Codex via codex-acp adapter.
// Binário canônico: "codex-acp" (NÃO "codex" — esse é o CLI legacy da OpenAI).
// Fallback: npx --yes @zed-industries/codex-acp@<CodexNpmVersion>.
// FixedArgs vazio — toda configuração via BootstrapArgs em tempo de spawn.
// AccessModeFlag vazio — Codex passa access via -c approval_policy=..., não flag dedicada.
func Codex() Spec {
    return newSpecWithBootstrap(
        "codex",
        "Codex (ACP)",
        "codex-acp",
        nil, // FixedArgs vazio
        []FallbackLauncher{
            {
                Command:   "npx",
                FixedArgs: []string{"--yes", CodexNpmPackage + "@" + CodexNpmVersion},
            },
        },
        "", // AccessModeFlag vazio
        CodexSDKVersion,
        CodexNpmVersion,
        CodexNpmPackage,
        codexBootstrapArgs,
    )
}

// codexBootstrapArgs replica compozy/internal/core/agent/registry_specs.go:247-278.
// Emite pares -c key="value" para model, reasoning, feature toggles e sandbox.
func codexBootstrapArgs(model, reasoning string, _ []string, mode AccessMode) []string {
    args := make([]string, 0, 14)
    if model != "" {
        args = appendCodexOverrides(args, "model="+strconv.Quote(model))
    }
    if reasoning != "" {
        args = appendCodexOverrides(args, "model_reasoning_effort="+strconv.Quote(reasoning))
    }
    args = appendCodexOverrides(args,
        "features.code_mode=false",
        "features.code_mode_only=false",
    )
    if mode == AccessModeFull {
        args = appendCodexOverrides(args,
            `approval_policy="never"`,
            `sandbox_mode="danger-full-access"`,
            `web_search="live"`,
        )
    }
    return args
}

// appendCodexOverrides adiciona cada override como par "-c <override>".
func appendCodexOverrides(args []string, overrides ...string) []string {
    for _, o := range overrides {
        args = append(args, "-c", o)
    }
    return args
}
```

```go
// internal/runtime/runner.go — Job estendido
type Job struct {
    Prompt           string
    WorkDir          string
    EvidenceDir      string
    Model            string
    ActivityTimeout  time.Duration
    Quiet            bool
    // NOVOS (F1-Codex; defaults preservam comportamento Claude/Copilot)
    ReasoningEffort  string             // "" para Claude/Copilot; "low|medium|high" para Codex
    AccessMode       specs.AccessMode   // AccessModeRestricted (default) ou AccessModeFull
    AddDirs          []string           // diretórios adicionais para Codex (SupportsAddDirs=true)
}

// ACPRunner.Run — extensão para consumir BootstrapArgs
func (r *ACPRunner) Run(ctx context.Context, job Job) (Summary, error) {
    launcher, err := r.prober.EnsureAvailable(ctx, r.spec)
    if err != nil {
        return Summary{}, err
    }
    launcherCmd, launcherArgs := launcher.Command()

    // F1-Codex: prepend bootstrap args antes das FixedArgs
    bootstrap := r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)
    argv := append([]string{}, launcherArgs...)
    if len(bootstrap) > 0 {
        argv = append(argv, bootstrap...)
    }
    argv = append(argv, r.spec.FixedArgs...)
    // ... resto do método inalterado: exec.Command(launcherCmd, argv...), acpClient, watchdog, persistence

    initRaw, _ := buildRuntimeInitRaw(launcher.Kind(), launcherCmd, argv,
        r.spec.SDKVersion(), r.spec.NPMVersion())
    // ... (já generalizado em F1-Copilot)
}
```

```go
// cmd/ai_spec_harness/task_loop.go — flags + catálogo
var runtimeACPCatalog = map[string]func() specs.Spec{
    "claude":  specs.Claude,
    "copilot": specs.Copilot,
    "codex":   specs.Codex, // F1-Codex
}

// Em init() do taskLoopCmd:
taskLoopCmd.Flags().String("reasoning-effort", "medium",
    "Esforço de raciocínio (Codex): low | medium | high (default medium)")
taskLoopCmd.Flags().String("access-mode", "restricted",
    "Modo de acesso (Codex): restricted (default) | full (atenção: desabilita sandbox)")

// Em RunE, após parsing dos flags:
reasoningEffort, _ := cmd.Flags().GetString("reasoning-effort")
accessMode, _ := cmd.Flags().GetString("access-mode")

// Validação enum
validReasoning := map[string]bool{"low": true, "medium": true, "high": true}
if !validReasoning[reasoningEffort] {
    return fmt.Errorf("--reasoning-effort inválido %q (low|medium|high)", reasoningEffort)
}
validAccess := map[string]bool{"restricted": true, "full": true}
if !validAccess[accessMode] {
    return fmt.Errorf("--access-mode inválido %q (restricted|full)", accessMode)
}

// Warning único para --access-mode=full
var accessModeFullWarnOnce sync.Once
if accessMode == "full" {
    accessModeFullWarnOnce.Do(func() {
        _, _ = fmt.Fprintln(os.Stderr,
            "WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. " +
            "Pré-condição: consentimento operacional. Codex terá acesso pleno ao filesystem " +
            "e à rede. Use somente em ambientes isolados. Ver CODEX.md.")
    })
}

// Propagar via Options
opts := taskloop.Options{
    // ... existentes
    ReasoningEffort: reasoningEffort,
    AccessMode:      specs.AccessMode(accessMode),
}
```

```go
// internal/taskloop/agent.go:335-351 — codexInvoker com sync.Once warning
var codexLegacyWarnOnce sync.Once

func (c *codexInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
    codexLegacyWarnOnce.Do(func() {
        _, _ = fmt.Fprintln(os.Stderr,
            "WARNING: Codex CLI legado (codex exec --yolo) em uso. " +
            "Migrar para --runtime=acp (binário codex-acp, pacote @zed-industries/codex-acp). " +
            "O modo legado será removido em 2 versões minor. Ver ADR-013.")
    })
    args := make([]string, 0, 5)
    args = append(args, "exec")
    if model != "" {
        args = append(args, "--model", model)
    }
    args = append(args, "--yolo", prompt)
    return runCmd(ctx, workDir, c.liveOut, "codex", args...)
}
```

### Modelos de Dados

Dois adicionais ao value object layer (todos retrocompatíveis):

1. `AccessMode` (string type alias) em `internal/runtime/specs/spec.go` — apenas `Restricted` e `Full` nesta fase.
2. `BootstrapArgsFunc` (function type) em `internal/runtime/specs/spec.go` — assinatura para spec-specific bootstrap builders.

Sem mudanças em `events`, `Job` ganha 3 campos opcionais (defaults preservam Claude/Copilot).

### Endpoints de API

N/A — funcionalidade de CLI/biblioteca.

## Pontos de Integração

**Internos**:
- `internal/runtime/specs/` — Novo `codex.go` + extensão de `spec.go`.
- `internal/runtime/runner.go` — Consome `spec.BootstrapArgs(...)` em `Run()`; `Job` ganha 3 campos.
- `internal/runtime/probe/probe.go` — Tabela `adrByID["codex"] = ".specs/adr/013-..."`.
- `internal/runtime/events/event.go` — Inalterado (já generalizado).
- `internal/taskloop/taskloop.go` — `Service.Execute` propaga `ReasoningEffort/AccessMode/AddDirs` para `Job`.
- `internal/taskloop/agent.go` — `codexInvoker.Invoke` ganha `sync.Once` warning.
- `internal/taskloop/compatibility.go` — Entrada `codex → [gpt-5.5]` na `CompatibilityTable`.
- `cmd/ai_spec_harness/task_loop.go` — Catálogo + flags + validação enum + warning full.

**Externos**:
- Binário `codex-acp` (Zed Industries adapter) versão `CodexMinNpmVersion`+ no PATH, OU
- `npx` no PATH + acesso ao registry npm para `@zed-industries/codex-acp@0.14.0`
- Auth Codex/Zed válido (pré-condição operacional; varia conforme o adapter).

## Abordagem de Testes

### Testes Unitários

| ID | Componente | Cenário | Resultado esperado |
|---|---|---|---|
| T-01 | `specs.Codex()` | Construtor retorna Spec com ID e Command corretos | `spec.ID == "codex"`, `spec.Command == "codex-acp"`, `spec.FixedArgs == nil` |
| T-02 | `specs.Codex()` | Spec carrega metadata SDK/NPM corretas | `spec.SDKVersion() == CodexSDKVersion`, `spec.NPMVersion() == CodexNpmVersion`, `spec.NPMPackage() == CodexNpmPackage` |
| T-03 | `specs.Codex()` | Spec declara fallback npx único | `len(spec.Fallbacks) == 1`, `spec.Fallbacks[0].Command == "npx"`, args contém `@zed-industries/codex-acp@0.14.0` |
| T-04 | `specs.Codex()` | Versão pinada não é literal "latest" | `CodexNpmVersion != "latest"` e não-vazia; `CodexMinNpmVersion <= CodexNpmVersion` (semver) |
| T-05 | `specs.Claude()`, `specs.Copilot()` | Specs herdadas continuam íntegras após extensão de Spec | Suítes existentes de `claude_test.go` e `copilot_test.go` passam sem alteração. `BootstrapArgs(...)` retorna `nil` para ambos |
| T-06 | `codexBootstrapArgs` | model vazio, reasoning vazio, AccessModeRestricted | `args == ["-c", "features.code_mode=false", "-c", "features.code_mode_only=false"]` |
| T-07 | `codexBootstrapArgs` | model "gpt-5.5", reasoning "medium", AccessModeRestricted | inclui `-c model="gpt-5.5"`, `-c model_reasoning_effort="medium"`, `-c features.code_mode=false`, `-c features.code_mode_only=false`; **não** inclui sandbox |
| T-08 | `codexBootstrapArgs` | model "gpt-5.5", reasoning "high", AccessModeFull | inclui todos os anteriores + `-c approval_policy="never"`, `-c sandbox_mode="danger-full-access"`, `-c web_search="live"` |
| T-09 | `codexBootstrapArgs` | reasoning "low" | `-c model_reasoning_effort="low"` (não rejeita; valida apenas em CLI) |
| T-10 | `Spec.BootstrapArgs` | Claude (no-op) | retorna `nil` |
| T-11 | `Spec.BootstrapArgs` | Copilot (no-op) | retorna `nil` |
| T-12 | `Spec.BootstrapArgs` | Codex | delega para `codexBootstrapArgs`; resultado conforme T-06..T-09 |
| T-13 | `probe.EnsureAvailable` | Spec Codex, binário ausente, npx ausente | erro contém `"codex-acp não encontrado"`, `"@zed-industries/codex-acp@0.14.0"`, `"--runtime=legacy"`, `"013-codex-cli-acp-native.md"` |
| T-14 | `probe.EnsureAvailable` | Spec Codex, binário presente | retorna `BinaryLauncher("codex-acp", ...)` com FixedArgs vazio |
| T-15 | `probe.EnsureAvailable` | Spec Codex, binário ausente, npx presente | retorna `NpxLauncher("@zed-industries/codex-acp", "0.14.0")` |
| T-16 | `probe.adrByID` | Lookup `"codex"` | `adrByID["codex"] == ".specs/adr/013-codex-cli-acp-native.md"` |
| T-17 | `ACPRunner.Run` (integration) | Spec Codex + fake ACP server, AccessModeRestricted | spawn args contêm `-c model="..."`, `-c model_reasoning_effort="..."`, `-c features.code_mode=false`; **não** contêm `sandbox_mode` |
| T-18 | `ACPRunner.Run` (integration) | Spec Codex + fake ACP server, AccessModeFull | spawn args contêm todos os de T-17 + `sandbox_mode="danger-full-access"`, `approval_policy="never"`, `web_search="live"` |
| T-19 | `ACPRunner.Run` (integration) | Spec Claude (regressão) | spawn args **não** contêm `-c` flags (BootstrapArgs no-op) |
| T-20 | `ACPRunner.Run` (integration) | Spec Codex + fake server emite ≥ 2 tool calls | `tool_calls.md` agregado correto; `execution_report.md` com counts certos |
| T-21 | `ACPRunner.Run` (integration) | Spec Codex + fake server inativo | `ActivityWatchdog` cancela via `CancelCause(ErrActivityTimeout)`; CancelReason = `activity_timeout` |
| T-22 | `cmd.task_loop` | `--tool codex --runtime acp` (T-14 invertido) | aceita, roteia para `ACPRunner` com `Job.ReasoningEffort="medium"`, `Job.AccessMode=AccessModeRestricted` (defaults) |
| T-23 | `cmd.task_loop` | `--tool codex --runtime acp --reasoning-effort high --access-mode full` (T-15 novo) | aceita; `Job.ReasoningEffort="high"`; `Job.AccessMode=AccessModeFull`; warning único emitido em stderr |
| T-24 | `cmd.task_loop` | `--reasoning-effort invalid` | erro `exit2` com mensagem listando enum aceito |
| T-25 | `cmd.task_loop` | `--access-mode invalid` | erro `exit2` com mensagem listando enum aceito |
| T-26 | `cmd.task_loop` | `--tool claude --reasoning-effort high --access-mode full --runtime acp` | aceito (regressão); Job recebe os valores mas BootstrapArgs no-op os ignora (T-19) |
| T-27 | `cmd.task_loop` | `--tool codex` (sem `--runtime`) | usa default `legacy` → `codexInvoker` legado |
| T-28 | `codexInvoker.Invoke` | Primeira invocação por processo | WARNING emitido em stderr exatamente uma vez referenciando ADR-013 |
| T-29 | `codexInvoker.Invoke` | Segunda invocação no mesmo processo | nenhum WARNING adicional (`sync.Once`) |
| T-30 | `cmd.task_loop` | `--access-mode full` (com qualquer tool) | warning único de full-access emitido em stderr |
| T-31 | Regressão | Suíte completa de `internal/runtime/...` | 100% verde após extensão de `Spec` e `Job` |
| T-32 | Regressão | Suíte completa de `internal/taskloop/...` | 100% verde após mudança em `codexInvoker` |
| T-33 | Regressão | Suíte de `cmd/ai_spec_harness/...` | 100% verde após adição ao catálogo e flags novas |
| T-34 | `compatibility.go` | `validateModelForTool("codex", "gpt-5.5")` | aceito sem `--allow-unknown-model` |
| T-35 | `compatibility.go` | `validateModelForTool("codex", "gpt-4")` (modelo arbitrário) | rejeitado sem `--allow-unknown-model`; aceito com flag |

**Mocks**: apenas `fake.Client` em `internal/runtime/client/client_test.go` (já existente) e `fs.FileSystem` quando necessário. Sem mock de `exec.Command` — testes de `probe` usam `LookPather` injetada (já é o padrão).

### Testes de Integração

> Avaliação contra critérios do template:
> - [x] Fronteira IO crítica onde mocks não garantem correção? **Sim** — o protocolo ACP sobre stdio é a costura entre subprocess `codex-acp` e SDK; fake server cobre semântica mas não o adapter real da Zed Industries.
> - [ ] Incidente prévio com mocks divergindo? **Não** — fake server já cobre Claude/Copilot com paridade observacional sólida (ADR-008).
> - [x] Custo de containers proporcional ao risco? **Proporcional** — testes E2E real opt-in via env `CODEX_E2E=1`, não bloqueiam CI.
>
> **Decisão**: integration tests dentro da matriz do fake ACP server (T-17 a T-21) são obrigatórios. E2E real com `codex-acp` é opt-in (`CODEX_E2E=1`), executado em smoke test manual e capturado em `audit/`.

### Testes E2E

Smoke test manual (não automatizado em CI nesta fase):

1. Pré-condição: `codex-acp --version` ≥ `CodexMinNpmVersion` ("0.12.0"), auth Codex/Zed válido.
2. Comando (modo restricted, default):
   ```bash
   GOVERNANCE_TELEMETRY=1 ai-spec-harness task-loop \
     --tool codex \
     --runtime acp \
     --reasoning-effort medium \
     --activity-timeout 120s \
     .specs/prd-codex-acp-spec
   ```
3. Comando (modo full, com warning):
   ```bash
   ai-spec-harness task-loop \
     --tool codex --runtime acp \
     --reasoning-effort high --access-mode full \
     .specs/prd-codex-acp-spec
   ```
4. Validações:
   - `events.jsonl` contém pelo menos: `runtime_init`, `agent_message`, `tool_call_start`, `tool_call_end`, `completion`.
   - `runtime_init` carrega `tool=codex`, `npm_version=0.14.0`, `sdk_version=v0.13.0`, `launcher=binary|npx`.
   - `tool_calls.md` agrega corretamente.
   - `execution_report.md` registra `Launcher: binary` ou `Launcher: npx`, `EventsCount > 0`, `UnknownEventsCount == 0`.
   - `.agents/telemetry.log` contém entrada `skill=runtime_init tool=codex launcher=...`.
   - Modo full: warning único emitido em stderr antes da invocação.
5. Evidência capturada em `audit/<timestamp>-codex-acp-smoke/`.

## Sequenciamento de Desenvolvimento

### Ordem de Build

1. **Estender `specs.Spec` com `AccessMode` e `BootstrapArgs`** — adicionar tipo `AccessMode`, função-tipo `BootstrapArgsFunc`, campo privado `bootstrapArgs`, método `BootstrapArgs(...)`, constructor `newSpecWithBootstrap`. Rodar `claude_test.go`, `copilot_test.go`, `spec_test.go` para regressão. Mitigação do risco **R-01** crítico. T-05, T-10, T-11.
2. **Criar `internal/runtime/specs/codex.go`** com constantes pinadas e construtor `Codex()`. Função local `codexBootstrapArgs` + helper `appendCodexOverrides`. Testes T-01..T-04, T-06..T-09, T-12.
3. **Estender `internal/runtime/runner.go::Job`** com campos `ReasoningEffort`, `AccessMode`, `AddDirs`. Defaults preservam Claude/Copilot. T-19.
4. **Modificar `internal/runtime/runner.go::Run`** para chamar `spec.BootstrapArgs(...)` e prepend ao argv. T-17, T-18, T-19.
5. **Adicionar entrada `"codex"` em `adrByID`** (`probe.go:21-24`). T-13, T-16.
6. **Adicionar `"codex": specs.Codex`** em `runtimeACPCatalog` (`task_loop.go:21-24`). T-22.
7. **Registrar flags `--reasoning-effort` e `--access-mode`** em `task_loop.go` + validação enum + warning único para `full`. T-23, T-24, T-25, T-30.
8. **Inverter T-14** em `task_loop_test.go:48-52` e adicionar T-15 (combinação completa). T-22, T-23.
9. **Propagar `ReasoningEffort/AccessMode/AddDirs`** via `taskloop.Options` em `internal/taskloop/taskloop.go::Service.Execute`. T-26.
10. **Aviso de depreciação em `codexInvoker`** (`internal/taskloop/agent.go:335-351`) com `sync.Once`. T-28, T-29.
11. **Adicionar entrada `codex → [gpt-5.5]`** em `internal/taskloop/compatibility.go::CompatibilityTable`. T-34, T-35.
12. **Sub-suite Codex em `internal/runtime/acp_integration_test.go`** reusando fake ACP server. T-17..T-21.
13. **Reescrever `CODEX.md` raiz** — "Modo Recomendado (2026)" primeira, "Modo Legado" deprecada, warning `--access-mode=full`, distinção `codex` vs `codex-acp`. RF-20.
14. **Atualizar `AGENTS.md`** — linha ADR-013. RF-21.
15. **Atualizar `docs/cli-schema.json`** — enums `--reasoning-effort`, `--access-mode`. RF-22.
16. **Atualizar `docs/telemetry-feedback-cycle.md`** — cobertura Codex. RF-23.
17. **Smoke test manual + captura de evidência** em `audit/<timestamp>-codex-acp-smoke/`. Suíte completa de testes (T-31, T-32, T-33).

### Dependências Técnicas

- Nenhuma dependência externa nova no `go.mod`. SDK ACP (`coder/acp-go-sdk@v0.13.0`) já presente.
- Binário externo `codex-acp` (Zed Industries adapter) ou `npx` + acesso npm a `@zed-industries/codex-acp@0.14.0` — pré-condição operacional, não dependência de build.
- ADR-013 já entregue (`.specs/adr/013-codex-cli-acp-native.md`).
- F1-Copilot (`prd-copilot-acp-spec/`) já entregue como precedente arquitetural — esta techspec assume runner/probe/Spec já generalizados.

## Monitoramento e Observabilidade

- **Telemetria opt-in** (`GOVERNANCE_TELEMETRY=1`, ADR-006): evento `runtime_init` ganha cardinalidade `tool=codex` quando aplicável. Não há novo kind de evento (preserva ADR-010).
- **Log estruturado**:
  - `WARNING` em `stderr`: depreciação do `codexInvoker` legado (uma vez por execução).
  - `WARNING` em `stderr`: `--access-mode=full` ativado (uma vez por execução).
  - `INFO` em `stderr`: launcher resolvido (`Launcher: binary` ou `Launcher: npx`), já existente.
- **Dashboards Grafana**: nenhum painel novo. Painéis existentes que filtram por `tool=claude|copilot` ganham granularidade `tool=codex` aditivamente. Tool name aliasing (`search_query` → `web_search`) **não** implementado nesta fase — telemetria Codex usará nomes nativos (decisão Q3-research, F2-Codex follow-up).
- **Captura forense**: `audit/<run>/events.jsonl` + `tool_calls.md` + `execution_report.md` produzidos com mesma estrutura que Claude/Copilot. `Launcher` field distingue `binary` de `npx`. Spawn args na payload de `runtime_init` permitem auditar quais `-c` overrides foram aplicados.

## Considerações Técnicas

### Decisões Chave

Cada decisão material abaixo está registrada em ADR-013 ([`.specs/adr/013-codex-cli-acp-native.md`](../adr/013-codex-cli-acp-native.md)):

- **D-01 — Adoção do adapter `codex-acp`**: o binário canônico para Codex via ACP é `codex-acp` (Zed Industries, npm `@zed-industries/codex-acp`), não `codex` (CLI legacy da OpenAI). Confusão de nomenclatura documentada em `CODEX.md` e na mensagem de erro de probe.
- **D-02 — Extensão da interface `Spec` com `BootstrapArgs`**: campo privado `bootstrapArgs BootstrapArgsFunc` + método público `BootstrapArgs(...)`. Default `nil` (no-op) preserva Claude/Copilot. R-DDD-001 respeitado (Spec continua value object imutável). Risco R-01.
- **D-03 — `newSpecWithBootstrap` como variant constructor**: mantém `newSpec` original sem mudança de assinatura. Claude/Copilot inalterados. Codex usa o variant. Alternativa rejeitada: estender `newSpec` em si quebrava encapsulamento dos testes existentes.
- **D-04 — `runtimeACPCatalog` continua em `cmd/ai_spec_harness/`** (decidido em F1-Copilot D-04): tabela é responsabilidade do CLI, não do catálogo de specs. Codex apenas adiciona uma entrada.
- **D-05 — Caminho legado mantido por 2 versões minor**: `codexInvoker` em `internal/taskloop/agent.go:335-351` continua disponível com warning de depreciação. Remoção é decisão de versão futura (Q4 do PRD; espelha F1-Copilot Q5).
- **D-06 — Pinning de Codex npm package**: `CodexNpmVersion = "0.14.0"` constante Go pinada (validada via `npm view @zed-industries/codex-acp versions` em 2026-05-21). `CodexMinNpmVersion = "0.12.0"` informacional (mínimo do compozy para `gpt-5.5`). Atualização via processo `audit/`.
- **D-07 — `AccessModeFlag` vazio para Codex**: Codex passa access via `-c approval_policy=...` (em `BootstrapArgs`), não flag dedicada como `--bypass-permissions` do Claude. Mantém Spec consistente.
- **D-08 — Warning único para `--access-mode=full`**: opt-in explícito via flag não é suficiente; warning em stderr na primeira invocação por execução (via `sync.Once`) reforça consentimento operacional.
- **D-09 — Tool name aliasing adiado para F2-Codex**: aliasing de `search_query` → `web_search` é cosmético e não bloqueia paridade observacional core. Telemetria Codex usa nomes nativos nesta fase.
- **D-10 — Mínima entrada em `CompatibilityTable`**: `codex → [gpt-5.5]` apenas. Outros modelos requerem `--allow-unknown-model`. Crescer tabela é PR futuro quando Codex tiver mais modelos suportados.

**Alternativas rejeitadas** (detalhe em ADR-013):
- A: Manter `codexInvoker` legado — rejeitado porque deixa Codex fora de paridade ACP indefinidamente.
- B: Hardcodar `codex-acp` args em `runner.go` — rejeitado porque cria Claude/Copilot/Codex-centrismo no runner e bloqueia Droid/Gemini futuros.
- C: Aguardar SDK upstream `@openai/codex-agent-acp` em Go — não existe; adapter Zed é o caminho real.
- D: Wrapper Go que invoque `codex` legado e traduza para JSON-RPC — reimplementa protocolo ACP; alta complexidade.

### Riscos Conhecidos

| ID | Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|---|
| R-01 | Regressão Claude/Copilot ao estender interface `Spec` | Baixa | **Crítico** | Ordem de Build #1 estende `Spec` com no-op default e roda `claude_test.go`, `copilot_test.go`, `spec_test.go` antes de qualquer outra mudança. T-05, T-10, T-11, T-19 garantem. |
| R-02 | `runtime_init` quebra consumidores ao mudar origem de spawn args (agora contém `-c` overrides) | Baixa | Médio | Payload `runtime_init` mantém estrutura; apenas o conteúdo de `args` muda quando Codex. T-17/T-18 valida. ADR-010 invariante preservada. |
| R-03 | `--access-mode=full` usado acidentalmente em ambiente compartilhado | Média | **Alto** | Warning único em stderr antes de propagar; documentação explícita em `CODEX.md`; default permanece `restricted`. D-08. |
| R-04 | Versão `@zed-industries/codex-acp@0.14.0` indisponível no momento da implementação | Baixa | Médio | Validado em 2026-05-21 via `npm view`. Constante pinada; falha graciosa via probe error (T-13). |
| R-05 | Auth Codex/Zed inválido → erro genérico do subprocess | Alta | Baixo | Documentar pré-condição em `CODEX.md`. Erro do subprocess capturado por `acpClient` e reportado em `events.jsonl`. |
| R-06 | Tool name aliasing ausente → dashboards multi-tool confusos | Média | Baixo | Documentado como follow-up (F2-Codex). Telemetria Codex usa nomes nativos (`search_query`, `image_query`). Decisão D-09. |
| R-07 | `BootstrapArgs` retorna args incorretos para edge case (model com aspas, reasoning inválido) | Baixa | Médio | `strconv.Quote` escapa corretamente. Validação de `--reasoning-effort` no CLI antes de propagar. T-06..T-09 cobre matriz. |
| R-08 | Quebra de invariantes forenses ao mudar `runner.go::Run()` | Baixa | **Crítico** | Diff zero esperado em `internal/runtime/persistence/*`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go`. T-31 valida. Revisão obrigatória do diff. |
| R-09 | Aviso de depreciação ruidoso em loops longos | Baixa | Baixo | `sync.Once` garante emissão única por execução. T-28/T-29 validam. |
| R-10 | Confusão `codex` vs `codex-acp` causa instalação errada | Média | Baixo | Mensagem de erro de probe explícita; `CODEX.md` documenta; ADR-013 §"Contexto" explica. |

### Conformidade com Padrões

Regras aplicáveis de `.claude/rules/` e `.agents/skills/agent-governance/references/`:

- **R-GOV-001** (`.claude/rules/governance.md`): este techspec respeita precedência — `agent-governance` carregado, suposições explicitadas, ADR-013 documenta decisões. Conflito hard/guideline: D-04 (catálogo em `cmd/` não em `specs/`) é guideline herdada de F1-Copilot.
- **R-DDD-001** (`agent-governance/references/ddd.md`): `Spec` permanece value object imutável; campo `bootstrapArgs` é privado; método `BootstrapArgs(...)` é read-only delegando para a função armazenada. Construtor `newSpecWithBootstrap` é único caminho de injeção (R-DDD-001 enforça).
- **R-SEC-001** (`agent-governance/references/security.md`): subprocess `codex-acp` segue mesmas regras de Claude/Copilot — sem shell, args via slice. `strconv.Quote` escapa valores em `-c` overrides corretamente. **`AccessModeFull` é opt-in explícito** via flag + warning; sandbox/approval/web_search overrides nunca chegam ao adapter sem consentimento.
- **R-ERR-001** (`agent-governance/references/error-handling.md`): erros sentinela existentes (`ErrLauncherUnavailable`, `ErrActivityTimeout`) reaproveitados. Sem novos sentinelas. Erros de validação de flag retornam `exit2` sem wrap (consistente com T-14 original).
- **R-TEST-001** (`agent-governance/references/testing.md`): tabela de testes (35 casos T-01..T-35) cobre positivos e negativos. FakeFileSystem + fake ACP server cobrem fronteiras de IO.
- **ADR-002**: filesystem abstraction preservada.
- **ADR-005**: spec-hash do PRD registrado no cabeçalho desta techspec (TBD via `make sync-spec-hash`).
- **ADR-006**: telemetria opt-in append-only; campos aditivos.
- **ADR-008**: paridade multi-tool — promove Codex de ausente para suportado. ADR-008 referenciada; atualização documental em `AGENTS.md` (RF-21).
- **ADR-009**: pinning SDK e npm package — `CodexNpmVersion`/`CodexSDKVersion` seguem mesma política. `CodexSDKVersion` sincronizada via `make sync-acp-sdk-version`.
- **ADR-010**: tagged union de eventos preservada — `runtime_init` continua sendo o único kind afetado pela mudança de spawn args; nenhum kind novo.
- **ADR-011**: Agent Registry preservado — pode coexistir (`--agent foo` com `runtime.ide=codex` em AGENT.md → Spec resolvida via `runtimeACPCatalog["codex"]`).
- **ADR-012**: F1-Copilot — esta techspec assume runner/probe/Spec já generalizados por ADR-012 como precedente.
- **ADR-013**: nova; declara decisões D-01 a D-10.

### Arquivos Relevantes e Dependentes

**Novos**:
- `internal/runtime/specs/codex.go`
- `internal/runtime/specs/codex_test.go`
- `.specs/adr/013-codex-cli-acp-native.md` (já entregue)

**Modificados**:
- `internal/runtime/specs/spec.go` — adicionar `AccessMode`, `BootstrapArgsFunc`, campo privado `bootstrapArgs`, método `BootstrapArgs(...)`, `newSpecWithBootstrap`
- `internal/runtime/specs/claude.go` — passar `bootstrapArgs=nil` via `newSpec` (sem mudança comportamental)
- `internal/runtime/specs/copilot.go` — idem
- `internal/runtime/specs/spec_test.go` — testes `BootstrapArgs` no-op para Claude/Copilot (T-10, T-11)
- `internal/runtime/runner.go` — `Job` ganha 3 campos; `Run()` chama `BootstrapArgs(...)` e prepend
- `internal/runtime/runner_test.go` — atualizar fixtures se necessário; T-19 (regressão Claude)
- `internal/runtime/probe/probe.go:21-24` — `adrByID["codex"]`
- `internal/runtime/probe/probe_test.go` — T-13, T-14, T-15, T-16
- `internal/runtime/acp_integration_test.go` — sub-suite Codex (T-17..T-21)
- `internal/taskloop/taskloop.go` — propagar `ReasoningEffort/AccessMode/AddDirs`
- `internal/taskloop/taskloop_test.go` — T-26
- `internal/taskloop/agent.go` — `codexInvoker.Invoke` com `sync.Once` warning
- `internal/taskloop/agent_test.go` — T-28, T-29
- `internal/taskloop/compatibility.go` — entrada `codex → [gpt-5.5]`
- `internal/taskloop/compatibility_test.go` — T-34, T-35
- `cmd/ai_spec_harness/task_loop.go` — flags + catálogo + validação enum + warning full
- `cmd/ai_spec_harness/task_loop_test.go` — T-14 invertido, T-15 novo, T-22..T-25, T-27, T-30, T-33
- `CODEX.md` raiz — reescrita
- `AGENTS.md` — linha ADR-013
- `docs/cli-schema.json` — enums `--reasoning-effort`, `--access-mode`
- `docs/telemetry-feedback-cycle.md` — cobertura Codex

**Inalterados (invariante de fase)**:
- `internal/runtime/persistence/*`
- `internal/runtime/watchdog.go`
- `internal/runtime/client/client.go`
- `internal/runtime/client/client_test.go`
- `internal/runtime/events/event.go`
- `internal/runtime/events/convert.go`

---

## Resolução de Suposições e Questões em Aberto do PRD

| ID | Item | Resolução |
|---|---|---|
| A1 | `codex-acp >= 0.12.0` expõe ACP com semântica idêntica ao Claude/Copilot | **Confirmação pendente** via integração com fake ACP server (T-17..T-21). Docs upstream confirmam compatibilidade ACP. |
| A2 | `@zed-industries/codex-acp@0.14.0` existe e suporta `-c` overrides | **Confirmado** em 2026-05-21 via `npm view @zed-industries/codex-acp versions` — versões publicadas incluem 0.12.0, 0.13.0, 0.14.0. |
| A3 | Auth Codex/Zed é pré-condição operacional | **Confirmado**. Documentado em `CODEX.md` (RF-20). Falhas reportadas via `acpClient` + `events.jsonl`. |
| A4 | `ActivityWatchdog` com 120s funciona para Codex | **Confirmado** por design — watchdog é semaforado por `Touch()` em qualquer evento. T-21 valida. |
| A5 | Estender `Spec` com `BootstrapArgs` não quebra Claude/Copilot | **Confirmado** por design (no-op default). T-05, T-10, T-11, T-19. |
| A6 | `codexInvoker` legado coexiste com ACP | **Confirmado** — roteamento por `opts.Runtime`. T-27. |
| A7 | Flags `--reasoning-effort`/`--access-mode` com Claude/Copilot são aceitas sem efeito | **Confirmado** — BootstrapArgs no-op ignora. T-26 valida. |
| Q1 | Mensagem do warning `--access-mode=full` | **Resolvido**: "WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. Pré-condição: consentimento operacional. Codex terá acesso pleno ao filesystem e à rede. Use somente em ambientes isolados. Ver CODEX.md." |
| Q2 | Modelo default do Codex | **Resolvido** (D-06): `DefaultCodexModel="gpt-5.5"` propagado quando `--executor-model` ausente. |
| Q3 | Tabela de compatibilidade Codex | **Resolvido** (D-10): mínima `codex → [gpt-5.5]`. Outros via `--allow-unknown-model`. T-34, T-35. |
| Q4 | Tempo de manutenção do `codexInvoker` legado | **Resolvido** (D-05): 2 versões minor (espelha F1-Copilot). |
| Q5 | `--reasoning-effort` enum vs string arbitrária | **Resolvido**: validação enum `low\|medium\|high` em CLI. T-24. |
| Q6 | `BootstrapArgs` como método vs campo público | **Resolvido** (D-02): campo privado + método público (R-DDD-001). |
| Q7 | `runtime_init` carrega `npm_version` em launcher `binary` | **Resolvido**: sim, consistente com F1-Copilot Q7 (sempre derivado do Spec). |
| Q8 | Validação runtime de versão `codex-acp >= 0.12.0` | **Resolvido** (adiado): F1-Codex assume disponibilidade quando `LookPath` resolve; validação semver fica para F2-Codex se necessário. Documentado em `CODEX.md`. |
