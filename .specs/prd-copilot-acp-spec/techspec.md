<!-- spec-hash-prd: c2cf759b8a13f6d110bf1d89b5f79d0254da32ff58e010ba86c45edd538696e3 -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Copilot CLI via ACP Nativo

> **PRD consumido**: [prd.md](./prd.md) (spec-version 1)
> **ADR substitutiva**: [012-copilot-cli-acp-native](../adr/012-copilot-cli-acp-native.md) (substitui ADR-007)
> **Insumo de pesquisa**: [docs/research/compozy-adaptation-copilot-2026.md](../../docs/research/compozy-adaptation-copilot-2026.md)
> **Fase**: 1 de 4 (Compozy 2026 — variante Copilot)

## Resumo Executivo

Introduz suporte ao GitHub Copilot CLI como runtime ACP nativo via novo construtor `specs.Copilot()` em `internal/runtime/specs/copilot.go`, espelhando o padrão de `claude.go:24-42` e o registro canônico do compozy em `internal/core/agent/registry_specs.go:222-242`. A integração reaproveita 100% da infraestrutura ACP existente (`ACPRunner`, `acpClient`, `SessionPersistence`, `ActivityWatchdog`, `events`); não cria nova camada nem dependência externa.

O ponto cirúrgico desta entrega é **generalizar dois pontos hoje Claude-specific** do runtime:

1. `internal/runtime/runner.go:113-120` hardcoda `specs.ClaudeSDKVersion` e `specs.ClaudeNpmVersion` no payload de `runtime_init`. Será substituído por metadata derivada do `Spec` resolvido.
2. `internal/runtime/probe/probe.go:69-82` assume `specs.ClaudeNpmPackage` no template de erro e referencia ADR-009 hardcoded. Será parametrizado pelo `Spec` recebido.

Em paralelo, `cmd/ai_spec_harness/task_loop.go:77` deixa de bloquear `--runtime=acp` para tools diferentes de Claude. Uma tabela `runtimeACPCatalog map[string]func() specs.Spec` torna-se a fonte de verdade para quais tools podem usar ACP nesta versão (Claude e Copilot). O `copilotInvoker` legado em `internal/taskloop/agent.go:381-388` permanece por uma versão com aviso de depreciação.

ADR-012 substitui formalmente ADR-007 (workaround stateless); `COPILOT.md` raiz é reescrito; AGENTS.md ganha referência a ADR-012. Persistência forense, watchdog e tagged union de eventos (ADR-010) permanecem inalterados em comportamento.

## Arquitetura do Sistema

### Visão Geral dos Componentes

**Novos**
- `internal/runtime/specs/copilot.go` — Construtor `Copilot() Spec` + constantes `CopilotNpmPackage`, `CopilotNpmVersion`, `CopilotSDKVersion`, `CopilotMinCLIVersion`.
- `internal/runtime/specs/copilot_test.go` — Testes de defaults da Spec, fallback npx, pinning de constantes.
- `cmd/ai_spec_harness/runtime_acp_catalog.go` (ou inline em `task_loop.go`) — Tabela `runtimeACPCatalog` mapeando tool name → função de catálogo.
- `.specs/adr/012-copilot-cli-acp-native.md` — **já entregue** declarando supersedure de ADR-007.

**Modificados**
- `internal/runtime/specs/spec.go` — Adicionar métodos `Spec.SDKVersion() string`, `Spec.NPMVersion() string`, `Spec.NPMPackage() string` (consumidos por runner e probe). Spec ganha três campos novos (`sdkVersion`, `npmVersion`, `npmPackage`) preenchidos pelo construtor `newSpec`. Assinatura de `newSpec` é estendida; chamadores existentes (`Claude()`) são atualizados.
- `internal/runtime/specs/claude.go` — Atualizar chamada a `newSpec(...)` passando `ClaudeSDKVersion`, `ClaudeNpmVersion`, `ClaudeNpmPackage`.
- `internal/runtime/runner.go:113-120` — `buildRuntimeInitRaw` e o evento `runtime_init` deixam de hardcodar constantes Claude. Recebem `spec.SDKVersion()`, `spec.NPMVersion()` (ou valores vazios quando launcher é `binary` e Spec não declara — decisão Q7 do PRD: continuar carregando metadata derivada do Spec).
- `internal/runtime/probe/probe.go:69-82` — `errMsgTemplate` parametrizado pelo binário do `Spec` (não literal "claude-agent-acp") e pelo `Spec.NPMPackage()`/`NPMVersion()`. Referência ao ADR no remédio torna-se metadata do Spec (`Spec.ADRReference() string` ou constante no package probe mapeando ID→ADR).
- `internal/runtime/runner.go` — `ACPRunner` continua agnóstico (já é). Apenas o helper `buildRuntimeInitRaw` muda assinatura para receber versões dinamicamente.
- `cmd/ai_spec_harness/task_loop.go:67-81` — Lógica de validação de `--runtime=acp` consulta `runtimeACPCatalog` em vez de comparar literal contra `"claude"`. Mensagem de erro lista catálogo dinamicamente.
- `internal/taskloop/taskloop.go` — `Service.Execute` consulta `runtimeACPCatalog` (injetado ou exposto via package `runtime`) quando `Runtime == "acp"`. Resolve `Spec` correspondente ao `Tool` e instancia `ACPRunner`.
- `internal/taskloop/agent.go:381-388` — `copilotInvoker.Invoke` emite log WARNING uma única vez por execução do processo (via `sync.Once`) anunciando depreciação.
- `COPILOT.md` raiz — Reescrita conforme [`docs/research/compozy-adaptation-copilot-2026.md`](../../docs/research/compozy-adaptation-copilot-2026.md) §"Exemplos de Configuração 2026".
- `AGENTS.md` — Tabela de ADRs ganha linha ADR-012; ADR-007 marcada como "substituída por ADR-012". Tabela de runtimes/tools suportados (se existir) é atualizada.
- `docs/adr/007-copilot-cli-stateless-workaround.md` — Cabeçalho ganha `**Status:** Substituída por ADR-012`. Conteúdo histórico preservado.
- `docs/telemetry-feedback-cycle.md` — Documenta que invariantes de telemetria cobrem `tool=copilot` quando `--runtime=acp --tool=copilot`.
- `docs/cli-schema.json` — Verificar se enum de `--tool` ou `--runtime` precisa atualização; caso enum aceito já cubra "copilot" sob "acp", nenhuma mudança necessária.
- `internal/runtime/acp_integration_test.go` — Sub-suite Copilot reusando fake ACP server existente.

**Inalterados (invariante de fase)**
- `internal/runtime/persistence/*` — Persistência forense intacta.
- `internal/runtime/watchdog.go` — `ActivityWatchdog` intacto.
- `internal/runtime/client/client.go` — `acpClient` intacto.
- `internal/runtime/client/client_test.go` — fake server intacto (reusado pela sub-suite Copilot).
- `internal/runtime/events/*` — Eventos e conversão SDK→domínio intactos. Tagged union ADR-010 preservado: nenhum kind novo adicionado.

### Relacionamentos e Fluxo de Dados

```
CLI: --tool copilot --runtime acp
         │
         ▼
cmd/ai_spec_harness/task_loop.go
   • valida --runtime in {legacy, acp}
   • valida --tool in runtimeACPCatalog quando runtime==acp
   • monta taskloop.Options{Tool: "copilot", Runtime: "acp", ...}
         │
         ▼
taskloop.Service.Execute(opts)
   • opts.Runtime == "acp" →
      specCtor := runtimeACPCatalog[opts.Tool]
      spec := specCtor()  // specs.Copilot()
      runner := airuntime.NewACPRunner(spec, opts...)
   • opts.Runtime == "legacy" → copilotInvoker (com WARNING uma vez)
         │
         ▼
airuntime.ACPRunner.Run(ctx, job)
   • probe.EnsureAvailable(ctx, spec) →
        - tenta spec.Command no PATH
        - fallback spec.Fallbacks (npx --yes @github/copilot@<pin> --acp)
        - erro template usa spec.NPMPackage()/NPMVersion() + ADR-012
   • buildRuntimeInitRaw(launcher.Kind(), cmd, args, spec.SDKVersion(), spec.NPMVersion())
   • events.NewRuntimeInit(... spec.SDKVersion(), spec.NPMVersion() ...)
   • acpClient.Open + fan-out (inalterado)
   • ActivityWatchdog (inalterado)
   • SessionPersistence (inalterado)
         │
         ▼
audit/<run>/events.jsonl + tool_calls.md + execution_report.md
   (mesmos campos e granularidade que Claude)
```

## Design de Implementação

### Interfaces Chave

```go
// internal/runtime/specs/spec.go — assinatura estendida
type Spec struct {
    ID             string
    DisplayName    string
    Command        string
    FixedArgs      []string
    Fallbacks      []FallbackLauncher
    AccessModeFlag string
    // NOVOS — metadata para runtime_init e probe error
    sdkVersion string
    npmVersion string
    npmPackage string
}

func newSpec(
    id, displayName, command string,
    fixedArgs []string,
    fallbacks []FallbackLauncher,
    accessModeFlag string,
    sdkVersion, npmVersion, npmPackage string, // NOVOS
) Spec

// Acessores públicos
func (s Spec) SDKVersion() string  { return s.sdkVersion }
func (s Spec) NPMVersion() string  { return s.npmVersion }
func (s Spec) NPMPackage() string  { return s.npmPackage }
```

```go
// internal/runtime/specs/copilot.go — novo arquivo
package specs

const (
    // CopilotNpmPackage é o nome do pacote npm canônico do Copilot CLI ACP.
    CopilotNpmPackage = "@github/copilot"

    // CopilotNpmVersion é a versão npm pinada do @github/copilot.
    // Pinada conforme ADR-009 §"Decisão": constante Go atualizada somente via audit/.
    // Não alterar para @latest. Para atualizar: registrar decisão em audit/ e atualizar manualmente.
    CopilotNpmVersion = "0.1.0" // Q4 — confirmar versão exata via npm registry no momento da implementação

    // CopilotSDKVersion é a versão do coder/acp-go-sdk sincronizada com go.mod.
    // Mantida em sincronia por scripts/sync-acp-sdk-version.sh.
    // Não editar manualmente — use make sync-acp-sdk-version.
    CopilotSDKVersion = "v0.13.0" // mesmo SDK do Claude (mesma versão)

    // CopilotMinCLIVersion é a versão mínima do binário `copilot` que expõe --acp.
    // Documentação: https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server
    CopilotMinCLIVersion = "X.Y.Z" // Q1 — confirmar via release notes upstream
)

// Copilot retorna a Spec do runtime GitHub Copilot CLI via ACP nativo.
// Binário canônico: "copilot" com FixedArgs=["--acp"].
// Fallback: npx --yes @github/copilot@<CopilotNpmVersion> --acp.
// AccessModeFlag vazio (sem flag análoga a --bypass-permissions do Claude no v0).
func Copilot() Spec {
    return newSpec(
        "copilot",
        "GitHub Copilot CLI (ACP)",
        "copilot",
        []string{"--acp"},
        []FallbackLauncher{
            {
                Command:   "npx",
                FixedArgs: []string{"--yes", CopilotNpmPackage + "@" + CopilotNpmVersion, "--acp"},
            },
        },
        "",
        CopilotSDKVersion,
        CopilotNpmVersion,
        CopilotNpmPackage,
    )
}
```

```go
// internal/runtime/runner.go — generalização de buildRuntimeInitRaw
func buildRuntimeInitRaw(launcher, command string, args []string, sdkVersion, npmVersion string) ([]byte, error) {
    return json.Marshal(map[string]any{
        "launcher":    launcher,
        "command":     command,
        "args":        args,
        "sdk_version": sdkVersion,
        "npm_version": npmVersion,
    })
}

// Chamada em ACPRunner.Run muda de:
//   buildRuntimeInitRaw(launcher.Kind(), launcherCmd, launcherArgs, specs.ClaudeSDKVersion, specs.ClaudeNpmVersion)
// para:
//   buildRuntimeInitRaw(launcher.Kind(), launcherCmd, launcherArgs, r.spec.SDKVersion(), r.spec.NPMVersion())

// Idem events.NewRuntimeInit — recebe spec.SDKVersion() / spec.NPMVersion() em vez de constantes Claude.
```

```go
// internal/runtime/probe/probe.go — generalização do erro
// errMsgTemplate passa a ser função:
func formatLauncherUnavailable(spec specs.Spec, adrPath string) string {
    return fmt.Sprintf(
        "%s não encontrado. Install %s; OR install %s@%s via npm; OR use --runtime=legacy. Veja %s",
        spec.Command, spec.Command, spec.NPMPackage(), spec.NPMVersion(), adrPath,
    )
}

// Mapping ID→ADR fica em uma tabela no package probe:
var adrByID = map[string]string{
    "claude":  ".specs/adr/009-acp-protocol-adoption.md",
    "copilot": ".specs/adr/012-copilot-cli-acp-native.md",
}
```

```go
// cmd/ai_spec_harness/task_loop.go — tabela runtimeACPCatalog
var runtimeACPCatalog = map[string]func() specs.Spec{
    "claude":  specs.Claude,
    "copilot": specs.Copilot,
}

// Validação substituída:
if runtime == "acp" {
    effectiveTool := tool
    if effectiveTool == "" {
        effectiveTool = execTool
    }
    if _, ok := runtimeACPCatalog[effectiveTool]; !ok {
        supported := make([]string, 0, len(runtimeACPCatalog))
        for k := range runtimeACPCatalog {
            supported = append(supported, k)
        }
        sort.Strings(supported)
        _, _ = fmt.Fprintf(os.Stderr,
            "runtime acp suporta apenas --tool em %v nesta versão\n", supported)
        return fmt.Errorf("exit2")
    }
}
```

```go
// internal/taskloop/agent.go — aviso de depreciação no copilotInvoker
var copilotLegacyWarnOnce sync.Once

func (c *copilotInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
    copilotLegacyWarnOnce.Do(func() {
        _, _ = fmt.Fprintln(os.Stderr,
            "WARNING: Copilot CLI em modo legado (sem ACP). Migrar para --runtime=acp. " +
            "O modo legado será removido em vX.Y.Z. Ver ADR-012.")
    })
    args := make([]string, 0, 6)
    if model != "" {
        args = append(args, "--model", model)
    }
    args = append(args, "--autopilot", "--yolo", "-p", prompt)
    return runCmd(ctx, workDir, c.liveOut, "copilot", args...)
}
```

### Modelos de Dados

Nenhum modelo de dados novo. Apenas extensão do value object `Spec` (3 campos privados + 3 acessores públicos), retrocompatível porque construtores externos passam pelos catálogos (`Claude()`, `Copilot()`) — não há instanciação por literal fora do package.

### Endpoints de API

N/A — funcionalidade de CLI/biblioteca.

## Pontos de Integração

**Internos**:
- `internal/runtime/specs/` — Novo `copilot.go` adicionado ao catálogo.
- `internal/runtime/runner.go` — Consome `spec.SDKVersion()`/`NPMVersion()` em `buildRuntimeInitRaw`.
- `internal/runtime/probe/probe.go` — Consome `spec.Command`/`NPMPackage()`/`NPMVersion()` no template de erro; tabela `adrByID` referencia ADR-012 para Copilot.
- `internal/runtime/events/event.go` — `NewRuntimeInit` continua aceitando `sdkVersion`/`npmVersion` como parâmetros (não muda assinatura) — apenas a origem dos valores muda no caller.
- `internal/taskloop/taskloop.go` — `Service.Execute` ganha branch para resolver `Spec` via catálogo ACP quando `Runtime == "acp"`.
- `internal/taskloop/agent.go` — `copilotInvoker` ganha `sync.Once` para warning.
- `cmd/ai_spec_harness/task_loop.go` — Nova tabela `runtimeACPCatalog`; validação revisada.

**Externos**:
- Binário `copilot` (GitHub Copilot CLI) versão `CopilotMinCLIVersion`+ no PATH, OU
- `npx` no PATH + acesso ao registry npm para `@github/copilot@<CopilotNpmVersion>`
- `gh auth status` válido com token Copilot (pré-condição operacional).

## Abordagem de Testes

### Testes Unitários

| ID | Componente | Cenário | Resultado esperado |
|---|---|---|---|
| T-01 | `specs.Copilot()` | Construtor retorna Spec com ID e Command corretos | `spec.ID == "copilot"`, `spec.Command == "copilot"`, `spec.FixedArgs == ["--acp"]` |
| T-02 | `specs.Copilot()` | Spec carrega metadata SDK/NPM corretas | `spec.SDKVersion() == CopilotSDKVersion`, `spec.NPMVersion() == CopilotNpmVersion`, `spec.NPMPackage() == CopilotNpmPackage` |
| T-03 | `specs.Copilot()` | Spec declara fallback npx único | `len(spec.Fallbacks) == 1`, `spec.Fallbacks[0].Command == "npx"`, args contém `--acp` ao final |
| T-04 | `specs.Copilot()` | Versão pinada não é literal "latest" | `CopilotNpmVersion != "latest"` e não-vazia |
| T-05 | `specs.Claude()` | Spec Claude continua íntegra após estender `newSpec` | Suíte existente de `claude_test.go` passa sem alteração |
| T-06 | `probe.EnsureAvailable` | Spec Copilot, binário ausente, npx ausente | erro contém `"copilot não encontrado"`, `"@github/copilot@..."`, `"--runtime=legacy"`, `"012-copilot-cli-acp-native.md"` |
| T-07 | `probe.EnsureAvailable` | Spec Copilot, binário presente | retorna `BinaryLauncher("copilot", ...)` com FixedArgs de Spec |
| T-08 | `probe.EnsureAvailable` | Spec Copilot, binário ausente, npx presente | retorna `NpxLauncher("@github/copilot", CopilotNpmVersion)` |
| T-09 | `runner.buildRuntimeInitRaw` | Spec Copilot resolvido | payload contém `sdk_version=CopilotSDKVersion`, `npm_version=CopilotNpmVersion` |
| T-10 | `ACPRunner.Run` (integration) | Spec Copilot + fake ACP server | `runtime_init` event carrega versões Copilot; `events.jsonl` produzido idêntico em estrutura a Claude |
| T-11 | `ACPRunner.Run` (integration) | Spec Copilot + fake ACP server emite ≥ 2 tool calls | `tool_calls.md` agregado correto; `execution_report.md` com counts certos |
| T-12 | `ACPRunner.Run` (integration) | Spec Copilot + fake ACP server inativo | `ActivityWatchdog` cancela via `CancelCause(ErrActivityTimeout)`; CancelReason = `activity_timeout` |
| T-13 | `cmd.task_loop` | `--tool copilot --runtime acp` | aceita, roteia para `ACPRunner` |
| T-14 | `cmd.task_loop` | `--tool gemini --runtime acp` | erro listando tools suportados em ACP (`[claude copilot]`) |
| T-15 | `cmd.task_loop` | `--tool claude --runtime acp` | aceita (regressão; comportamento atual preservado) |
| T-16 | `cmd.task_loop` | `--tool copilot` (sem `--runtime`) | usa default `legacy` → `copilotInvoker` legado |
| T-17 | `copilotInvoker.Invoke` | Primeira invocação por processo | WARNING emitido em stderr exatamente uma vez |
| T-18 | `copilotInvoker.Invoke` | Segunda invocação no mesmo processo | nenhum WARNING adicional emitido (`sync.Once`) |
| T-19 | `specs.Spec` | `SDKVersion`/`NPMVersion`/`NPMPackage` de Spec novo (criado fora do catálogo, hipotético) | retornam valores passados em `newSpec`; campos privados não podem ser setados via literal |
| T-20 | `probe.adrByID` | Lookup por ID conhecido | `adrByID["claude"] == ".specs/adr/009-..."`, `adrByID["copilot"] == ".specs/adr/012-..."` |
| T-21 | `probe.adrByID` | ID desconhecido | fallback para mensagem genérica sem ADR (ou ADR raiz `.specs/adr/`) |
| T-22 | Regressão | Suíte completa de `internal/runtime/...` | 100% verde após generalização de `runner.go` e `probe.go` |
| T-23 | Regressão | Suíte completa de `internal/taskloop/...` | 100% verde após mudança em `copilotInvoker` |
| T-24 | Regressão | Suíte de `cmd/ai_spec_harness/...` | 100% verde após tabela `runtimeACPCatalog` |

**Mocks**: apenas `fake.Client` em `internal/runtime/client/client_test.go` (já existente) e `fs.FileSystem` quando necessário. Sem mock de `exec.Command` — testes de `probe` usam `LookPather` injetada (já é o padrão).

### Testes de Integração

> Avaliação contra critérios do template:
> - [x] Fronteira IO crítica onde mocks não garantem correção? **Sim** — o protocolo ACP sobre stdio é a costura entre subprocess e SDK; fake server cobre semântica mas não o `--acp` real do Copilot CLI.
> - [ ] Incidente prévio com mocks divergindo? **Não** — fake server já cobre Claude com paridade observacional sólida (ADR-008).
> - [x] Custo de containers proporcional ao risco? **Proporcional** — testes E2E real são opt-in via env `COPILOT_E2E=1`, não bloqueiam CI.
>
> **Decisão**: integration tests dentro da matriz do fake ACP server (T-10/T-11/T-12) são obrigatórios. E2E real com `copilot --acp` é opt-in (`COPILOT_E2E=1`), executado em smoke test manual e capturado em `audit/`.

### Testes E2E

Smoke test manual (não automatizado em CI nesta fase):

1. Pré-condição: `copilot --version` ≥ `CopilotMinCLIVersion`, `gh auth status` válido.
2. Comando:
   ```bash
   GOVERNANCE_TELEMETRY=1 ai-spec-harness task-loop \
     --tool copilot \
     --runtime acp \
     --activity-timeout 120s \
     .specs/prd-copilot-acp-spec
   ```
3. Validações:
   - `events.jsonl` contém pelo menos: `runtime_init`, `agent_message`, `tool_call_start`, `tool_call_end`, `completion`.
   - `tool_calls.md` agrega corretamente.
   - `execution_report.md` registra `Launcher: binary` ou `Launcher: npx`, `EventsCount > 0`, `UnknownEventsCount == 0`.
   - `.agents/telemetry.log` contém entrada `skill=runtime_init tool=copilot launcher=...`.
4. Evidência capturada em `audit/<timestamp>-copilot-acp-smoke/`.

## Sequenciamento de Desenvolvimento

### Ordem de Build

1. **Estender `specs.Spec` value object** (3 campos privados + 3 acessores) e atualizar `newSpec` signature. Atualizar `claude.go` para passar novos parâmetros. Rodar `internal/runtime/specs/claude_test.go` para regressão. Mitigação do risco R-01.
2. **Criar `internal/runtime/specs/copilot.go`** com constantes pinadas e construtor. Testes T-01 a T-04.
3. **Atualizar `internal/runtime/specs/copilot_test.go`** cobrindo defaults e fallback. Testes T-19.
4. **Generalizar `internal/runtime/runner.go:113-120`** — `buildRuntimeInitRaw` e `events.NewRuntimeInit` recebem versões via `spec.SDKVersion()/NPMVersion()`. Testes T-05/T-09 (regressão Claude + extensão Copilot).
5. **Generalizar `internal/runtime/probe/probe.go:69-82`** — `errMsgTemplate` parametrizado; tabela `adrByID` para mapping ID→ADR. Testes T-06/T-07/T-08/T-20/T-21.
6. **Tabela `runtimeACPCatalog` em `cmd/ai_spec_harness/task_loop.go`** — Substitui literal `"claude"`. Mensagem de erro lista catálogo. Testes T-13/T-14/T-15.
7. **Wiring no `Service.Execute`** (`internal/taskloop/taskloop.go`) — Branch ACP consulta `runtimeACPCatalog`. Roteia Copilot para `ACPRunner` quando `Runtime == "acp"`. Testes T-16 (regressão legado).
8. **Sub-suite Copilot em `internal/runtime/acp_integration_test.go`** — Reusa fake ACP server. Testes T-10/T-11/T-12.
9. **Aviso de depreciação em `copilotInvoker`** (`internal/taskloop/agent.go:381-388`) com `sync.Once`. Testes T-17/T-18.
10. **Reescrever `COPILOT.md` raiz** — Seção "Modo Recomendado (2026): Copilot via ACP" primeira; "Modo Legado" deprecada. RF-13.
11. **Atualizar `AGENTS.md`** — Linha ADR-012 na tabela; ADR-007 marcada como substituída; tabela de runtimes/tools atualizada se aplicável. RF-14.
12. **Atualizar `docs/adr/007-...md`** com cabeçalho `**Status:** Substituída por ADR-012`. RF-19.
13. **Atualizar `docs/cli-schema.json`** se necessário (validar enum atual). RF-15.
14. **Atualizar `docs/telemetry-feedback-cycle.md`** documentando cobertura Copilot. RF-16.
15. **Smoke test manual + captura de evidência** em `audit/<timestamp>-copilot-acp-smoke/`. Suíte completa de testes (T-22/T-23/T-24).

### Dependências Técnicas

- Nenhuma dependência externa nova no `go.mod`. SDK ACP (`coder/acp-go-sdk@v0.13.0`) já presente.
- Binário externo `copilot` (Copilot CLI) ou `npx` + acesso npm a `@github/copilot@<pin>` — pré-condição operacional, não dependência de build.
- ADR-012 já entregue (`.specs/adr/012-copilot-cli-acp-native.md`).

## Monitoramento e Observabilidade

- **Telemetria opt-in** (`GOVERNANCE_TELEMETRY=1`, ADR-006): evento `runtime_init` ganha cardinalidade `tool=copilot` quando aplicável. Não há novo kind de evento (preserva ADR-010).
- **Log estruturado**:
  - `WARNING` em `stderr`: depreciação do `copilotInvoker` legado (uma vez por execução).
  - `WARNING` em `stderr`: `agent requested permission` — herdado de Claude, mensagem agora também relevante a Copilot quando aplicável (texto mantido genérico ou parametrizado se Copilot tiver flag análoga — Q3).
  - `INFO` em `stderr`: launcher resolvido (`Launcher: binary` ou `Launcher: npx`), já existente.
- **Dashboards Grafana**: nenhum painel novo. Painéis existentes que filtram por `tool=claude` ganham granularidade `tool=copilot` aditivamente.
- **Captura forense**: `audit/<run>/events.jsonl` + `tool_calls.md` + `execution_report.md` produzidos com mesma estrutura que Claude. `Launcher` field distingue `binary` de `npx`.

## Considerações Técnicas

### Decisões Chave

Cada decisão material abaixo está registrada em ADR-012 ([`.specs/adr/012-copilot-cli-acp-native.md`](../adr/012-copilot-cli-acp-native.md)):

- **D-01 — Supersedure de ADR-007**: ADR-007 deixa de valer porque sua premissa técnica (CLI stateless) deixou de ser verdade em 2026. Marcada como substituída por ADR-012; conteúdo histórico preservado.
- **D-02 — Reuso total do stack ACP**: `ACPRunner`, `acpClient`, `SessionPersistence`, `ActivityWatchdog` são reusados sem modificação semântica. Apenas `runner.go:113-120` e `probe/probe.go:69-82` são generalizados (deixam de hardcodar constantes Claude). Risco R-02.
- **D-03 — Spec ganha metadata privada + acessores públicos**: extensão retrocompatível porque construtores são funções de catálogo (`Claude()`, `Copilot()`). Nenhum chamador instancia `Spec{...}` por literal (R-DDD-001 já enforça isso).
- **D-04 — `runtimeACPCatalog` em `cmd/ai_spec_harness/`** (não em `internal/runtime/specs/`): a tabela é responsabilidade do CLI (decidir quais tools podem usar `--runtime=acp` nesta versão), não do catálogo de specs. Specs são unidades atômicas.
- **D-05 — Caminho legado mantido por uma versão**: `copilotInvoker` em `internal/taskloop/agent.go:381-388` continua disponível com warning de depreciação. Remoção é decisão de versão futura (Q5).
- **D-06 — Pinning de Copilot npm package**: `CopilotNpmVersion` constante Go pinada (não `@latest`), mesma política de ADR-009 para `ClaudeNpmVersion`. Atualização via processo `audit/`.
- **D-07 — `AccessModeFlag` vazio**: Copilot CLI no v0 não documenta flag análoga a `--bypass-permissions` do Claude. Spec declara `AccessModeFlag=""`. Revisar se documentação upstream do Copilot atualizar (Q3).
- **D-08 — Documentar versão mínima**: `CopilotMinCLIVersion` constante Go; `COPILOT.md` documenta. Verificação em runtime fica fora de escopo desta fase (probe não valida versão do binário — assume disponibilidade quando `LookPath` resolve).
- **D-09 — Tabela `adrByID` em probe**: mapping de `Spec.ID` para path do ADR é local ao package `probe`. Evita acoplamento entre `specs` e a estrutura de docs ADRs.

**Alternativas rejeitadas** (detalhe em ADR-012):
- A: Manter ADR-007 (workaround stateless) — rejeitado porque premissa técnica deixou de valer.
- B: Aguardar SDK upstream `@github/copilot-agent-acp` em Go — não existe em 2026-05; bloqueia entrega.
- C: Plugin custom para `gh` — alta complexidade e manutenção contínua.
- D: Hardcodar Spec Copilot em `runner.go` em vez de generalizar Spec — rejeitado porque mantém Claude-centrismo e bloqueia futuros runtimes ACP (Codex/Gemini/Droid).

### Riscos Conhecidos

| ID | Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|---|
| R-01 | Regressão Claude ao estender `newSpec` signature | Baixa | Alto | Ordem de Build #1 atualiza `claude.go` e roda `claude_test.go` antes de qualquer outra mudança. T-05 garante. |
| R-02 | `runtime_init` event quebra consumidores downstream ao mudar origem das versões | Baixa | Médio | Payload mantém mesmos campos (`sdk_version`, `npm_version`), apenas com cardinalidade Copilot adicional. T-09/T-10 valida. ADR-010 invariante preservada. |
| R-03 | Versão mínima do Copilot CLI mal documentada → erros opacos para usuário | Média | Médio | `CopilotMinCLIVersion` constante Go + documentação em `COPILOT.md` (RF-13). Validação de versão fica fora desta fase (D-08) mas erro de probe é claro (T-06). |
| R-04 | Auth Copilot inválido → erro genérico do subprocess | Alta | Baixo | Documentar pré-condição `gh auth status` em `COPILOT.md` (D-08). Erro do subprocess é capturado por `acpClient` e reportado em `events.jsonl`. |
| R-05 | `@github/copilot@<pin>` indisponível no npm registry no momento da implementação | Média | Médio | Q4 do PRD pendente: confirmar versão exata via `npm view @github/copilot versions` no momento da implementação. Falha de probe é caso explícito (T-06). |
| R-06 | Aviso de depreciação ruidoso em loops longos | Baixa | Baixo | `sync.Once` garante emissão única por execução. T-17/T-18 validam. |
| R-07 | Tabela `runtimeACPCatalog` cresce e exige refactor | Baixa | Baixo | Tabela local ao `cmd/`; mudança futura é simples (adicionar entrada). Não cria acoplamento com specs. |
| R-08 | Quebra de invariantes forenses ao mudar `buildRuntimeInitRaw` | Baixa | **Crítico** | Diff zero esperado em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`. T-22 valida regressão. Revisão obrigatória do diff. |

### Conformidade com Padrões

Regras aplicáveis de `.claude/rules/` e `.agents/skills/agent-governance/references/`:

- **R-GOV-001** (`.claude/rules/governance.md`): este techspec respeita precedência — `agent-governance` carregado, suposições explicitadas, ADR-012 documenta decisões. Conflito hard/guideline: D-04 (catálogo em `cmd/` não em `specs/`) é guideline, justificada (separação de responsabilidades).
- **R-DDD-001** (`agent-governance/references/ddd.md`): `Spec` permanece value object imutável; campos novos (`sdkVersion`, `npmVersion`, `npmPackage`) são privados; acessores são read-only. Construtor `newSpec` continua sendo a única forma de criar `Spec`. R-DDD-001 enforça que `Copilot()` deve usar `newSpec`, não literal `Spec{...}`.
- **R-SEC-001** (`agent-governance/references/security.md`): subprocess `copilot --acp` segue mesmas regras de Claude — sem shell, args via slice. Sem expansão de variáveis no template de erro. Pacote npm `@github/copilot` é confiável (publicado pelo GitHub).
- **R-ERR-001** (`agent-governance/references/error-handling.md`): erros sentinela existentes (`ErrLauncherUnavailable`, `ErrActivityTimeout`, `ErrPermissionDenied`) reaproveitados. Sem novos sentinelas.
- **R-TEST-001** (`agent-governance/references/testing.md`): tabela de testes (24 casos) cobre positivos e negativos. FakeFileSystem + fake ACP server cobrem fronteiras de IO.
- **ADR-002**: filesystem abstraction preservada.
- **ADR-005**: spec-hash do PRD registrado no cabeçalho desta techspec (`c2cf759b...`).
- **ADR-006**: telemetria opt-in append-only; campos aditivos.
- **ADR-008**: paridade multi-tool — promove Copilot CLI de `BestEffort` para suportado (decisão Q6: avaliar se update do ADR-008 entra nesta PR ou em PR separada — proposta: atualização documental em `AGENTS.md` aqui; ADR-008 conteúdo fica como está se decisão for granular).
- **ADR-009**: pinning SDK e npm package — `CopilotNpmVersion`/`CopilotSDKVersion` seguem mesma política. `CopilotSDKVersion` sincronizada via `make sync-acp-sdk-version`.
- **ADR-010**: tagged union de eventos preservada — `runtime_init` continua sendo o único kind afetado pela mudança, apenas o payload ganha cardinalidade Copilot.
- **ADR-011**: Agent Registry preservado — esta fase não conflita; pode coexistir (`--agent foo` com `runtime.ide=copilot` em AGENT.md → Spec resolvida via `runtimeACPCatalog["copilot"]`).
- **ADR-012**: nova; declara supersedure de ADR-007 e regista decisões D-01 a D-09.

### Arquivos Relevantes e Dependentes

**Novos**:
- `internal/runtime/specs/copilot.go`
- `internal/runtime/specs/copilot_test.go`
- `.specs/adr/012-copilot-cli-acp-native.md` (já entregue)

**Modificados**:
- `internal/runtime/specs/spec.go` — value object estendido com metadata privada + acessores
- `internal/runtime/specs/claude.go` — atualizar chamada a `newSpec` (assinatura nova)
- `internal/runtime/specs/claude_test.go` — verificar regressão (sem mudança de teste se assinatura preservar comportamento via wrapper)
- `internal/runtime/runner.go` — `buildRuntimeInitRaw` e chamada a `events.NewRuntimeInit` consomem `spec.SDKVersion()`/`NPMVersion()`
- `internal/runtime/runner_test.go` — atualizar fixtures se necessário
- `internal/runtime/probe/probe.go` — `errMsgTemplate` → função; tabela `adrByID`
- `internal/runtime/probe/probe_test.go` — atualizar fixtures para Spec Copilot
- `internal/runtime/acp_integration_test.go` — sub-suite Copilot
- `internal/taskloop/taskloop.go` — branch ACP consulta `runtimeACPCatalog`
- `internal/taskloop/taskloop_test.go` — caso Copilot ACP
- `internal/taskloop/agent.go` — `copilotInvoker.Invoke` com `sync.Once` warning
- `internal/taskloop/agent_test.go` — T-17/T-18 do warning único
- `cmd/ai_spec_harness/task_loop.go` — tabela `runtimeACPCatalog`; validação revisada
- `cmd/ai_spec_harness/task_loop_test.go` — T-13/T-14/T-15
- `COPILOT.md` raiz — reescrita
- `AGENTS.md` — linha ADR-012; ADR-007 marcada substituída
- `docs/adr/007-copilot-cli-stateless-workaround.md` — header status atualizado
- `docs/telemetry-feedback-cycle.md` — cobertura Copilot
- `docs/cli-schema.json` — validar enum (provavelmente sem mudança)

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
| A1 | Copilot CLI versão X.Y.Z+ expõe `--acp` com semântica idêntica ao Claude | **Confirmação pendente** via release notes upstream; validada por T-10/T-11/T-12 contra fake ACP server. Versão mínima documentada em `CopilotMinCLIVersion` (D-08). |
| A2 | `npx --yes @github/copilot@<pin> --acp` é fallback válido | **Confirmação pendente** via `npm view @github/copilot versions` (Q4/R-05). Falha graciosamente via probe error (T-06). |
| A3 | Auth Copilot é pré-condição operacional | **Confirmado**. Documentado em `COPILOT.md` (RF-13). Falhas reportadas via `acpClient` + `events.jsonl`. |
| A4 | `ActivityWatchdog` com 120s funciona para Copilot | **Confirmado** por design — watchdog é semaforado por `Touch()` em qualquer evento, independente de origem. T-12 valida. |
| A5 | Generalizar `runtime_init` não quebra downstream | **Confirmado** — payload mantém mesmos campos; apenas origem dos valores muda. T-09/T-10 + ADR-010 invariante. |
| A6 | `copilotInvoker` legado coexiste com ACP | **Confirmado** — roteamento por `opts.Runtime`. T-15/T-16 validam. |
| Q1 | Versão mínima do `copilot` CLI com `--acp` | **A confirmar na implementação** (Ordem de Build #1 ou antes). Constante `CopilotMinCLIVersion = "X.Y.Z"` placeholder; preencher antes de mergear. |
| Q2 | Modelo default do Copilot quando `--model` não passado | **Resolvido**: deixar Copilot decidir (passar args sem `--model`). Sem constante `DefaultCopilotModel` nesta fase. |
| Q3 | Flag análoga a `--bypass-permissions` no Copilot | **Resolvido** (D-07): `AccessModeFlag=""`. Reavaliar quando upstream documentar. |
| Q4 | Versão pinada inicial de `@github/copilot` | **A confirmar na implementação** via `npm view @github/copilot versions`. Constante `CopilotNpmVersion = "0.1.0"` placeholder. |
| Q5 | Tempo de manutenção do `copilotInvoker` legado | **Resolvido** (D-05): 2 versões minor. Deprecação na vX.Y; remoção em vX.Y+2. Versões exatas registradas em CHANGELOG quando mergear. |
| Q6 | Promover Copilot em ADR-008 para `Required` | **Resolvido**: nesta PR atualiza `AGENTS.md` (linha de ADRs + tabela de tools). Update de ADR-008 fica como follow-up tático se necessário. |
| Q7 | `runtime_init` carrega versões mesmo em launcher `binary` | **Resolvido** (D-02): sim, sempre derivadas do Spec resolvido. T-09 valida. |
