# Análise de Adaptação ao Padrão Compozy — Foco Codex-CLI 2026

> **Status**: Pesquisa concluída — base para futuro PRD `prd-codex-acp-spec` (F1-Codex)
> **Data**: 2026-05-21
> **Fonte primária (Compozy)**: leitura via `gh` do repositório [`compozy/compozy`](https://github.com/compozy/compozy) — branch `main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`
> **Fonte primária (harness)**: árvore atual de `/Users/jailtonjunior/Git/orchestrator` na branch `feat/acp-runtime-claude`
> **Pesquisa irmã**: [`docs/research/compozy-adaptation-copilot-2026.md`](compozy-adaptation-copilot-2026.md) (F1-Copilot entregue; F2–F4 desta pesquisa referenciam o roadmap dela)
> **Pesquisa correlata**: [`docs/research/compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md) (análise genérica, 10 dimensões)
> **Prompt de origem**: [`docs/prompts/compozy-adaptation-research-codex.md`](../prompts/compozy-adaptation-research-codex.md)

---

## Sumário Executivo

O Codex-CLI no compozy é tratado como runtime ACP de primeira classe, lado a lado com Claude, Copilot, Droid, Gemini, OpenCode, Pi e Cursor (`internal/core/agent/registry_specs.go:106-122`). Diferente do Copilot — onde `BootstrapArgs` retorna `nil` —, o Codex injeta hiperparâmetros de runtime via pares `-c key="value"` em uma função dedicada (`codexBootstrapArgs`, `registry_specs.go:247-278`). Esses pares carregam **model**, **reasoning effort**, **feature toggles** e, no modo de acesso pleno, controles de **sandbox / approval policy / web search**. Esta é a singularidade do Codex no ecossistema ACP 2026.

Como o stack ACP do `ai-spec-harness` já foi generalizado para Copilot (Probe, Runner e Summary tool-agnósticos após ADR-012), **adicionar Codex é ~80% reaproveitamento + ~20% peculiaridades Codex**. Os ~20% específicos são: (a) `BootstrapArgs` dinâmico (model + reasoning + feature toggles + sandbox); (b) gating de versão `@zed-industries/codex-acp >= 0.12.0`; (c) tool name aliasing (`search_query` → `web_search`, `image_query` → `image_search`); (d) honrar `$CODEX_HOME` env var com fallback `~/.codex`; (e) binário canônico chamado `codex-acp` (não `codex`).

Recomendação operacional: tratar **F1-Codex** (Codex ACP Spec) como próximo PRD candidato após F1-Copilot encerrar. F2 (memória 2-níveis), F3 (hook system) e F4 (TUI Bubble Tea) descritos no doc Copilot **aplicam-se igualmente a Codex sem trabalho adicional** quando o `Spec` Codex existir — a generalização do runtime ACP é a fundação compartilhada. Esta pesquisa **não** executa F1-Codex (sem código, sem PRD, sem ADR-013); apenas o ancora.

Correções factuais ao prompt original estão documentadas em §"Correções ao prompt original": o "spec-hash" é conceito do harness, não do compozy; o servidor MCP reservado expõe **apenas** `run_agent` (não tools de skills); e o binário ACP do Codex é `codex-acp` (pacote `@zed-industries/codex-acp`), distinto do `codex` da OpenAI — uma confusão de nomenclatura comum em 2026.

---

## Correções ao prompt original

O prompt em `docs/prompts/compozy-adaptation-research-codex.md` carrega quatro premissas que esta pesquisa precisou corrigir contra o código real do compozy (validado via `gh api`):

1. **"Spec-hash do Orchestrator integrado à interface de feedback do Codex"** (Fase 4 do prompt) — o `spec-hash` é um conceito do `ai-spec-harness` (veja `internal/skills/frontmatter.go` e ADR-005), não existe no compozy. O compozy expõe `model_reasoning_effort` e `access_mode` como hiperparâmetros de runtime, não um hash contractual. A integração proposta na Fase 4 do prompt precisa ser reformulada como "preservar spec-hash do harness ao introduzir Codex" — não há contraparte do lado compozy.

2. **"Servidor MCP interno para expor nossas skills (`create-prd`, `execute-task`) como ferramentas nativas para o Codex"** (Fase 3 do prompt) — Compozy expõe **apenas** `run_agent` via MCP em `internal/core/agents/mcpserver/server.go` (constante `reservedToolName = "run_agent"`). Skills **não** são expostas como tools MCP em nenhum runtime do compozy, inclusive Codex. A Fase 3 do prompt precisa ser reformulada como "MCP para hand-off recursivo entre agentes Codex", não como exposição de skills.

3. **"Codex-CLI nativo ACP"** — confirmado, mas via binário `codex-acp` (pacote npm `@zed-industries/codex-acp`), **distinto do `codex` da OpenAI**. Esta é a confusão de nomenclatura mais comum em 2026: `codex` (CLI da OpenAI, stateless) vs `codex-acp` (adapter ACP da Zed Industries que envolve o motor Codex). O harness atual em `internal/taskloop/agent.go:333-351` invoca `codex exec --yolo`, ou seja, o CLI legado — o futuro Spec Codex deve invocar `codex-acp`, não `codex`.

4. **"Compactação de contexto inspirada no Compozy"** (Fase 2 do prompt) — Compozy implementa medição de limites (`workflow 150 linhas / 12 KiB`; `task 200 linhas / 16 KiB`) em `internal/core/memory/store.go` mas **não compacta automaticamente**. O builder de prompt em `internal/core/prompt/prd.go` injeta diretiva "compact the flagged memory files before proceeding" no system prompt e o LLM executa a compactação. **Princípio prompt-driven**, não code-driven. Esta correção já está documentada no doc irmão para Copilot e aplica-se identicamente a Codex.

Essas correções não invalidam o roadmap proposto pelo prompt; apenas alinham as recomendações ao código-fonte real do compozy.

---

## Achados por Dimensão

### 1. Codex Spec com `BootstrapArgs` dinâmico (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/agent/registry_specs.go:106-122`:

```go
model.IDECodex: {
    ID:                 model.IDECodex,
    DisplayName:        "Codex",
    SetupAgentName:     "codex",
    DefaultModel:       model.DefaultCodexModel,
    Command:            "codex-acp",
    SupportsAddDirs:    true,
    UsesBootstrapModel: true,
    Fallbacks: []Launcher{
        {
            Command:   "npx",
            FixedArgs: []string{"--yes", "@zed-industries/codex-acp"},
        },
    },
    DocsURL:       "https://github.com/zed-industries/codex-acp",
    InstallHint:   "Install or update the Codex ACP adapter with `npm install -g @zed-industries/codex-acp@latest`, then expose `codex-acp` on PATH.",
    BootstrapArgs: codexBootstrapArgs,
},
```

Três campos divergem do Spec Copilot (`registry_specs.go:222-242`, referenciado no doc irmão):

- `Command: "codex-acp"` (não `codex`) — adapter ACP da Zed Industries, distinto do CLI Codex da OpenAI.
- `UsesBootstrapModel: true` — único entre os runtimes; Claude/Copilot/Gemini têm `false` ou ausente.
- `BootstrapArgs: codexBootstrapArgs` — função, não `nil`. Veja §"BootstrapArgs" abaixo.

**ai-spec-harness** — `internal/runtime/specs/spec.go:12-23`:

```go
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
}
```

**Lacuna crítica**: a struct `Spec` atual **não tem** suporte a `BootstrapArgs` dinâmico nem ao flag `UsesBootstrapModel`. Para Codex isso é bloqueante. As funções `Claude()` (`claude.go:28-45`) e `Copilot()` (`copilot.go:33-50`) só passam dados estáticos para `newSpec`; nenhuma delas precisa computar argv em função de model/reasoning/access.

**Impacto**: enquanto Claude e Copilot conseguem viver dentro do contrato atual de `Spec`, Codex requer extensão da interface. Sem `BootstrapArgs` dinâmico, o binário `codex-acp` é invocado sem injeção de model/reasoning e cai no default (`gpt-5.5` com reasoning effort indefinido), perdendo paridade com o compozy.

**Gap técnico**: criar `internal/runtime/specs/codex.go` espelhando `claude.go:28-45` e `copilot.go:33-50`, mas o `newSpec` constructor precisa ser estendido (ou um novo `newSpecWithBootstrap`) para aceitar `BootstrapArgs func(model, reasoning, addDirs string, accessMode AccessMode) []string`. O runner (`internal/runtime/runner.go`) precisa chamar essa função antes de montar argv e prepend o resultado às `FixedArgs`.

---

### 2. `codexBootstrapArgs` e injeção via `-c key="value"` (Compozy ✅ | harness ❌)

**Compozy** — `internal/core/agent/registry_specs.go:247-278`:

```go
var codexManagedRuntimeConfigOverrides = []string{
    "features.code_mode=false",
    "features.code_mode_only=false",
}

func codexBootstrapArgs(modelName, reasoningEffort string, _ []string, accessMode string) []string {
    args := make([]string, 0, 14)
    if selected := strings.TrimSpace(modelName); selected != "" {
        args = appendCodexConfigOverrides(args, "model="+strconv.Quote(selected))
    }
    if effort := strings.TrimSpace(reasoningEffort); effort != "" {
        args = appendCodexConfigOverrides(args, "model_reasoning_effort="+strconv.Quote(effort))
    }
    args = appendCodexConfigOverrides(args, codexManagedRuntimeConfigOverrides...)
    if accessMode == model.AccessModeFull {
        args = appendCodexConfigOverrides(
            args,
            `approval_policy="never"`,
            `sandbox_mode="danger-full-access"`,
            `web_search="live"`,
        )
    }
    return args
}

func appendCodexConfigOverrides(args []string, overrides ...string) []string {
    for _, override := range overrides {
        if strings.TrimSpace(override) == "" {
            continue
        }
        args = append(args, "-c", override)
    }
    return args
}
```

Cada override vira um par `-c key="value"`, interpretado pelo `codex-acp` como sobrescrita de config. Exemplo de invocação completa em modo full-access (extraído de `TestCodexBootstrapArgsSetManagedRuntimeOverrides`):

```
codex-acp \
  -c model="gpt-5.5" \
  -c model_reasoning_effort="high" \
  -c features.code_mode=false \
  -c features.code_mode_only=false \
  -c approval_policy="never" \
  -c sandbox_mode="danger-full-access" \
  -c web_search="live"
```

Comparação com Droid (`registry_specs.go` outro entry — também `UsesBootstrapModel`): Droid usa `--model X --reasoning-effort Y` como flags flat, **não** pares `-c`. Codex é único nesse formato.

**ai-spec-harness**: o invoker Codex legado em `internal/taskloop/agent.go:333-351` invoca `codex exec --yolo <prompt>` (sem ACP, sem model, sem reasoning, sem sandbox). Não há equivalente ao `codexBootstrapArgs` no stack ACP.

```go
func (c *codexInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
    args := make([]string, 0, 5)
    args = append(args, "exec")
    if model != "" {
        args = append(args, "--model", model)
    }
    args = append(args, "--yolo", prompt)
    return runCmd(ctx, workDir, c.liveOut, "codex", args...)
}
```

Note que (a) usa binário `codex`, não `codex-acp`; (b) usa `--model X` flat, não `-c model="X"`; (c) não tem reasoning effort; (d) não tem access mode; (e) não passa pelo runner ACP, portanto sem `events.jsonl` / `tool_calls.md` / watchdog / telemetria.

**Impacto**: paridade comportamental com Compozy exige que o futuro Spec Codex no harness reproduza fielmente o formato `-c key="value"` — não basta passar `--model X`. Sandbox e reasoning são parte do contrato do `codex-acp >= 0.12.0`.

**Gap técnico**: F1-Codex precisa de uma função `codexBootstrapArgs(model, reasoning, addDirs, accessMode)` em `internal/runtime/specs/codex.go`, espelhando o compozy. O runner chama essa função e prepend o resultado às `FixedArgs` antes de spawnar o processo.

---

### 3. Gating de versão `codex-acp >= 0.12.0` (Compozy ✅ | harness 🟡 pinning estático)

**Compozy** — `internal/core/agent/registry_compat.go`:

```go
var codexModelRequirements = map[string]runtimeModelRequirement{
    "gpt-5.5": {
        RuntimeCommand:     "codex-acp",
        RuntimeDisplayName: "codex-acp",
        PackageName:        "@zed-industries/codex-acp",
        MinVersion:         "0.12.0",
        UpgradeCommand:     "npm install -g @zed-industries/codex-acp@latest",
    },
}
```

A versão mínima é validada por modelo: `gpt-5.5` exige `codex-acp >= 0.12.0` porque versões anteriores não suportam a sintaxe `-c key="value"`. Modelos não registrados no mapa são rejeitados em tempo de probe.

**ai-spec-harness** — `internal/runtime/specs/copilot.go:8-27` pina por constante Go sem validação semver runtime:

```go
const (
    CopilotNpmPackage = "@github/copilot"
    CopilotNpmVersion = "1.0.51"
    CopilotSDKVersion = "v0.13.0"
    CopilotMinCLIVersion = "1.0.51"
)
```

O `CopilotMinCLIVersion` é informacional (documentado no header) mas não há gate runtime. O probe (`internal/runtime/probe/probe.go`) só checa existência do binário, não a versão.

**Impacto**: a estratégia atual do harness (constante pinada, sem semver check) é defensável para Claude/Copilot porque seus pacotes têm contratos estáveis. Para Codex o risco é maior: `codex-acp < 0.12.0` falha silenciosamente em runtime ao receber `-c` flags, com mensagens de erro pouco informativas.

**Decisão recomendada** (a registrar em ADR-013 futura): seguir o padrão de constante pinada (`CodexNpmVersion = "0.12.0"`) e adicionar verificação opcional `codex-acp --version` no probe quando o binário canônico for resolvido (não no fallback npx, que já pina a versão via `@X.Y.Z`). Documentar em techspec do PRD futuro.

**Gap técnico**: parte de F1-Codex. Constantes em `internal/runtime/specs/codex.go` espelhando `copilot.go:8-27`.

---

### 4. Tool name aliasing por driver (Compozy ✅ Codex-único | harness ❌)

**Compozy** — `internal/core/agent/tool_call_name.go::driverToolTitleAlias`:

```go
func driverToolTitleAlias(driverID string, token string) (string, bool) {
    switch driverID {
    case model.IDECodex:
        switch token {
        case "search_query":
            return toolNameWebSearch, true
        case "image_query":
            return toolNameImageSearch, true
        }
    case model.IDEClaude, model.IDECursor, model.IDEDroid, model.IDEOpenCode, model.IDEPi, model.IDEGemini:
        // Use common aliases only.
    }
    return "", false
}
```

Codex é o **único** runtime com aliasing por driver no compozy. Os tool names internos do Codex (`search_query`, `image_query`) são mapeados para nomes canônicos (`toolNameWebSearch`, `toolNameImageSearch`) antes de serem renderizados na UI ou persistidos em telemetria.

**ai-spec-harness** — o decoder de eventos em `internal/runtime/events/` é agnóstico de driver: nomes de tools são propagados como vêm do SDK ACP, sem alias.

**Impacto**: dashboards multi-tool que esperam `web_search` canônico verão `search_query` em sessões Codex, dificultando agregação cross-runtime. Para uso single-tool a divergência é cosmética.

**Decisão recomendada** (a registrar em F2-Codex como follow-up, não bloqueante de F1): introduzir mapa `aliasesByDriver map[string]map[string]string` em `internal/runtime/events/convert.go` aplicado antes da persistência. Custo baixo (~30 LoC + tabela). Alternativa: ignorar e documentar a divergência na techspec de F1-Codex.

**Gap técnico**: F2-Codex (opcional, baixa prioridade). Não bloqueia F1.

---

### 5. `$CODEX_HOME` e diretório de skills (Compozy ✅ | harness 🟡 documentado, sem resolver)

**Compozy** — `internal/setup/agents.go:125-127`:

```go
codeXHome := strings.TrimSpace(options.CodeXHome)
if codeXHome == "" {
    codeXHome = filepath.Join(homeDir, ".codex")
}
```

E o helper `codexPath` em `agents.go:219`:

```go
func codexPath(path string) pathSpec {
    return pathSpec{root: envRootCodeX, path: path}
}
```

Registro do agente Codex em `agents.go:271`:

```go
universalAgent("codex", "Codex", codexPath("skills"), codexPath(""), absolutePath("/etc/codex"))
```

Convenção: skills em `$CODEX_HOME/skills` (default `~/.codex/skills`); config root em `$CODEX_HOME`; fallback de sistema em `/etc/codex` (containers).

**ai-spec-harness** — `CODEX.md` (esqueleto atual, 30 linhas) já documenta a expectativa:

> 3. `.codex/config.toml` lista as skills habilitadas para resolucao e upgrade via harness.

Mas **não há resolver de path** que honre `$CODEX_HOME`. O instalador (referenciado no CHANGELOG como `copyToolValidationHooks` e `GenerateCodexAgents`) distribui hooks e agent files em paths relativos a `.codex/` no projeto; nada lê a env var.

**Impacto**: ambientes onde o usuário override-eia `$CODEX_HOME` (ex: containers com `$CODEX_HOME=/opt/codex`) verão skills serem instaladas em `~/.codex/skills` mas o `codex-acp` ler de `/opt/codex/skills` — quebra silenciosa.

**Gap técnico**: F3-Codex (~50 LoC). Adicionar helper `resolveCodexHome()` em `internal/runtime/specs/codex.go` ou `internal/setup/codex.go` que honre `$CODEX_HOME` com fallback `~/.codex`, e usar no install flow + nas mensagens de erro de probe.

---

### 6. Reasoning effort e AccessMode como parâmetros de CLI (Compozy ✅ | harness ❌)

**Compozy** — `codexBootstrapArgs(modelName, reasoningEffort string, _ []string, accessMode string) []string`. Os valores chegam do `Job`/`Execution` mais acima na pilha — testes em `internal/core/run/executor/execution_acp_test.go` mostram fixtures com `ReasoningEffort: "medium"`, `"high"`, `"low"`, e `AccessMode: model.AccessModeFull` ou `model.AccessModeRestricted`.

**ai-spec-harness** — `cmd/ai_spec_harness/task_loop.go:189-192` lista as flags ACP runtime atuais:

```go
taskLoopCmd.Flags().String("runtime", "legacy", "Runtime de invocacao: legacy (default) ou acp (tools suportados: claude, copilot)")
taskLoopCmd.Flags().Duration("activity-timeout", 120*time.Second, "Timeout de inatividade do agente ACP (0 = desabilitado); aceita time.Duration: 90s, 2m")
taskLoopCmd.Flags().Bool("quiet", false, "Suprime stream humano (stdout); jsonl e warnings continuam")
```

**Não há** `--reasoning-effort` nem `--access-mode`. O `Job` em `internal/runtime/runner.go` também não tem campos correspondentes.

**Impacto**: Codex no harness rodaria com reasoning effort default (indefinido) e access mode default. Para paridade comportamental com compozy, F1-Codex precisa expor essas duas flags + propagá-las via `Job` até `Spec.BootstrapArgs`.

**Gap técnico**: parte de F1-Codex. Duas flags novas + dois campos novos em `Job` + extensão da interface `Spec` para aceitar `BootstrapArgs(model, reasoning, addDirs, accessMode)`.

---

### 7. CLI gating bloqueia Codex em `--runtime=acp` (harness)

**ai-spec-harness** — `cmd/ai_spec_harness/task_loop.go:21-24`:

```go
var runtimeACPCatalog = map[string]func() specs.Spec{
    "claude":  specs.Claude,
    "copilot": specs.Copilot,
}
```

E o gate em `task_loop.go:82-97`:

```go
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

Codex não está no `runtimeACPCatalog`. Teste T-14 em `cmd/ai_spec_harness/task_loop_test.go:48-52` valida explicitamente a rejeição:

```go
{
    name:    "runtime acp com tool codex invalido (RF-02)",
    runtime: "acp",
    tool:    "codex",
    wantErr: true,
    wantMsg: "exit2",
},
```

**Impacto**: usuário que tente `ai-spec task-loop --tool codex --runtime acp <prd>` recebe erro com `exit2`. Configuração esperada para F1-Codex.

**Gap técnico**: F1-Codex inverte T-14 (passa a esperar `wantErr: false`) e adiciona `"codex": specs.Codex` ao catálogo. Também registra `"codex": "tasks/adr/013-codex-cli-acp-native.md"` em `internal/runtime/probe/probe.go:21-24`.

---

## Gap Map Consolidado

Legenda: 🟢 implementado · 🟡 parcial · 🔴 ausente · ⭐ vantagem do harness a preservar

| # | Feature | Status Orchestrator | Padrão Compozy | Gap Técnico | Fase |
|---|---|---|---|---|---|
| 1 | Codex via ACP nativo (`codex-acp`) | 🔴 invoker `codex exec --yolo` (stateless) | 🟢 `codex-acp` + npx fallback (`registry_specs.go:106-122`) | Criar `specs/codex.go`; estender interface `Spec` com `BootstrapArgs`; aceitar codex em `runtimeACPCatalog` | **F1-Codex** |
| 2 | `BootstrapArgs` dinâmico (-c key="value") | 🔴 | 🟢 `codexBootstrapArgs` (`registry_specs.go:247-278`) | Função em `specs/codex.go` + extensão da interface; runner prepend ao argv | **F1-Codex** |
| 3 | Reasoning effort como flag CLI | 🔴 | 🟢 propagado via `Job.ReasoningEffort` | Flag `--reasoning-effort` em `task_loop.go`; campo no `Job` | **F1-Codex** |
| 4 | AccessMode (restricted/full) | 🔴 | 🟢 propagado via `Job.AccessMode`; full adiciona sandbox/approval/web_search | Tipo `AccessMode` em `specs/spec.go`; flag `--access-mode`; lógica em `codexBootstrapArgs` | **F1-Codex** |
| 5 | Version gating `codex-acp >= 0.12.0` | 🔴 | 🟢 `registry_compat.go::codexModelRequirements` | Constante `CodexNpmVersion` + check opcional `--version` no probe | **F1-Codex** |
| 6 | Tool name aliasing por driver | 🔴 | 🟢 `tool_call_name.go::driverToolTitleAlias` (Codex-único) | Mapa `aliasesByDriver` em `events/convert.go` (~30 LoC) | F2-Codex (opcional) |
| 7 | `$CODEX_HOME` resolver | 🔴 | 🟢 `setup/agents.go:125` | Helper `resolveCodexHome()` no install flow | F3-Codex |
| 8 | CLI gating aceita Codex em `--runtime=acp` | 🔴 (T-14 bloqueia) | 🟢 (Codex em catálogo) | `runtimeACPCatalog["codex"] = specs.Codex`; inverter T-14 | **F1-Codex** |
| 9 | `CODEX.md` documenta modo ACP 2026 | 🟡 esqueleto de 30 linhas | 🔴 não existe em compozy | Reescrever `CODEX.md` com §"Modo Recomendado (2026)" | **F1-Codex** |
| 10 | `.codex/config.toml` versionado | 🔴 referenciado mas não comitado | 🔴 não comitado em compozy (config dinâmica via `-c`) | Criar template + installer flow | F1-Codex (com F3 opcional) |
| 11 | Persistência forense (`events.jsonl`, `tool_calls.md`) para Codex | 🔴 | 🔴 (compozy usa OTel/Grafana) | Reaproveitar runner generalizado (zero código novo) | **F1-Codex** (herda) |
| 12 | ADR-013 e PRD `prd-codex-acp-spec` | 🔴 | n/a | Escrever ADR-013 (template `tasks/adr/000-template.md`) + PRD com TechSpec + Tasks | **F1-Codex** |

**Critérios de esforço**: Baixo ≤ 1 sprint; Médio 1–2 sprints; Alto ≥ 2 sprints com pesquisa.

---

## Roadmap de Adaptação (Codex-específico)

### F1-Codex — Codex ACP Spec (próximo PRD candidato)

**Escopo**:

- Estender `internal/runtime/specs/spec.go:12-23`: adicionar tipo `AccessMode string` (constantes `AccessModeRestricted = "restricted"`, `AccessModeFull = "full"`) e método opcional `BootstrapArgs(model, reasoning string, addDirs []string, accessMode AccessMode) []string` no `Spec` com default no-op (retorna `nil` para Claude/Copilot por backward-compat).
- Novo `internal/runtime/specs/codex.go` espelhando `claude.go:28-45` e `copilot.go:33-50`:
  - `CodexNpmPackage = "@zed-industries/codex-acp"`
  - `CodexNpmVersion = "0.12.0"` (pinado conforme ADR-009; só atualizado via audit/)
  - `CodexSDKVersion = "v0.13.0"` (mesma do Claude/Copilot, em go.mod)
  - `DefaultCodexModel = "gpt-5.5"`
  - Função `Codex()` retornando `Spec` com `Command: "codex-acp"`, `FixedArgs: nil`, fallback `npx --yes @zed-industries/codex-acp@<CodexNpmVersion>`, `AccessModeFlag: ""` (Codex passa access via `-c`, não flag dedicada), e `BootstrapArgs: codexBootstrapArgs` (função local que replica o compozy).
- Função `codexBootstrapArgs` em `internal/runtime/specs/codex.go`:
  - Emite `-c model="<name>"` quando model presente
  - Emite `-c model_reasoning_effort="<level>"` quando reasoning presente
  - Sempre emite `-c features.code_mode=false -c features.code_mode_only=false`
  - Em `AccessModeFull` adiciona `-c approval_policy="never" -c sandbox_mode="danger-full-access" -c web_search="live"`
- Generalizar `internal/runtime/runner.go`: em `Run()`, chamar `r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)` e prepend ao argv da launcher antes das `FixedArgs`. Claude/Copilot herdam comportamento atual (default no-op retorna `nil`).
- Estender `internal/runtime/runner.go::Job` com campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string`.
- Adicionar `"codex": specs.Codex` ao `runtimeACPCatalog` em `cmd/ai_spec_harness/task_loop.go:21-24`.
- Inverter T-14 em `cmd/ai_spec_harness/task_loop_test.go:48-52` (passa a esperar `wantErr: false`) e adicionar T-15 cobrindo `--reasoning-effort high --access-mode restricted`.
- Adicionar `"codex": "tasks/adr/013-codex-cli-acp-native.md"` em `internal/runtime/probe/probe.go:21-24`.
- Flags novas em `cmd/ai_spec_harness/task_loop.go`:
  - `--reasoning-effort` (default `"medium"`; valores aceitos: `low`, `medium`, `high`)
  - `--access-mode` (default `"restricted"`; valores aceitos: `restricted`, `full`)
  - Propagar via `taskloop.Options` até `Job.ReasoningEffort` / `Job.AccessMode`.
- Novo `internal/runtime/specs/codex_test.go` espelhando `copilot_test.go`:
  - T-01 `TestCodexSpecDefaults` — valida ID, Command (`codex-acp`), FixedArgs (vazio)
  - T-02 `TestCodexSpecMetadata` — valida SDKVersion, NPMVersion, NPMPackage
  - T-03 `TestCodexSpecFallback` — valida fallback `npx --yes @zed-industries/codex-acp@0.12.0`
  - T-04 `TestCodexBootstrapArgs` — table-driven com casos: sem model, com model, com reasoning, full access
  - T-05 `TestCodexAccessModeFullAddsSandboxOverrides` — valida triplet sandbox/approval/web_search
- Reescrever `CODEX.md` (raiz, 30 linhas atuais → ~80 linhas) com §"Modo Recomendado (2026)" descrevendo invocação `codex-acp` via harness, hooks de validação em `.codex/hooks/`, skills em `$CODEX_HOME/skills` (ver §"Exemplos de Configuração 2026" abaixo).
- ADR-013 ([`tasks/adr/013-codex-cli-acp-native.md`](../../tasks/adr/013-codex-cli-acp-native.md), a criar) documentando: (a) escolha de `codex-acp` sobre `codex` legado; (b) pinning `@zed-industries/codex-acp@0.12.0`; (c) extensão da interface `Spec` com `BootstrapArgs`; (d) novas flags CLI; (e) decisão de adiar aliasing (F2-Codex).
- PRD `tasks/prd-codex-acp-spec/` com TechSpec + Tasks decompostas conforme padrão `prd-copilot-acp-spec/`.

**Esforço**: Baixo–Médio. **Risco**: Médio (extensão da interface `Spec` afeta Claude/Copilot; mitigar com default no-op).

**Dependências**: F1-Copilot (ADR-012, especs.Copilot) entregue como precedente arquitetural. Generalização de runner/probe já feita.

**Pré-requisito de viabilidade**: confirmar que `@zed-industries/codex-acp@0.12.0` está disponível via npm registry e suporta os 6 overrides `-c` documentados. Documentar versão na techspec.

**Critério de aceitação**: `ai-spec task-loop --tool codex --runtime acp --reasoning-effort high --access-mode restricted tasks/prd-X` gera `events.jsonl`, `tool_calls.md`, `execution_report.md` paritários aos modos Claude/Copilot.

### F2-Codex — Tool name aliasing (opcional, baixa prioridade)

**Escopo bruto**:
- Mapa `aliasesByDriver map[string]map[string]string` em `internal/runtime/events/convert.go`
- Entradas: `{"codex": {"search_query": "web_search", "image_query": "image_search"}}`
- Aplicar antes da persistência em `events.jsonl` e antes da agregação em `tool_calls.md`
- Telemetria preserva o nome canônico

**Riscos**: dashboards/relatórios pré-F2-Codex que esperam `search_query` em sessões Codex quebram. Mitigação: registrar mudança em CHANGELOG + flag opcional `--preserve-tool-names` para compatibilidade.

**Dependências**: F1-Codex entregue.

**Esforço**: Baixo (~30 LoC + testes).

### F3-Codex — `$CODEX_HOME` resolver no install flow

**Escopo bruto**:
- Helper `resolveCodexHome() string` em `internal/setup/codex.go` (novo arquivo)
- Honra env `$CODEX_HOME` com fallback `~/.codex`
- Usar em `copyToolValidationHooks` e `GenerateCodexAgents` (já referenciados no CHANGELOG)
- Atualizar `CODEX.md` com nota explicando precedência

**Riscos**: ambientes legados que não setam `$CODEX_HOME` mas têm `~/.codex` populado preservam comportamento atual (fallback é compatível).

**Dependências**: F1-Codex entregue (Spec define onde skills deveriam viver).

**Esforço**: Baixo (~50 LoC).

### F4-Codex — Referência ao roadmap Copilot

As fases F2 (memória 2-níveis), F3 (hook system 33-canônicos) e F4 (TUI Bubble Tea + daemon) descritas em [`compozy-adaptation-copilot-2026.md`](compozy-adaptation-copilot-2026.md) §"Roadmap" **aplicam-se igualmente a Codex sem trabalho adicional**. Após F1-Codex, sessões Codex consomem automaticamente o stack memory-first, hook dispatcher e TUI quando esses forem entregues. Não duplicar pesquisa nem PRDs.

---

## Exemplos de Configuração 2026

### `CODEX.md` raiz — reescrita F1-Codex

```markdown
# Codex — ai-spec-harness

## Modo Recomendado (2026): Codex via ACP

Em 2026 o adapter `codex-acp` (`@zed-industries/codex-acp`) passou a expor o motor
Codex via Agent Client Protocol nativo. O harness suporta esse modo via
`--runtime=acp --tool=codex`.

### Pré-requisitos
- `codex-acp` versão >= 0.12.0 (verifique com `codex-acp --version`)
- Variáveis Codex configuradas conforme docs do `@zed-industries/codex-acp`
- Alternativa: `npx --yes @zed-industries/codex-acp@0.12.0` (fallback automático)

### Uso

\`\`\`bash
ai-spec-harness task-loop \
  --tool codex \
  --runtime acp \
  --reasoning-effort medium \
  --access-mode restricted \
  tasks/prd-minha-feature
\`\`\`

A sessão produz os mesmos artefatos forenses do modo Claude/Copilot:
- `events.jsonl` (linha-a-linha de eventos ACP)
- `tool_calls.md` (agregado de tool calls; nomes Codex `search_query`/`image_query`
  preservados até F2-Codex; depois aliado para `web_search`/`image_search`)
- `execution_report.md` (summary final)

Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) registra invocações com `tool=codex`.

### Flags Codex-específicas
- `--reasoning-effort {low,medium,high}` — controla `model_reasoning_effort`
  injetado via `-c model_reasoning_effort="<level>"` no `codex-acp`
- `--access-mode {restricted,full}` — `full` adiciona overrides de sandbox
  (`approval_policy=never`, `sandbox_mode=danger-full-access`, `web_search=live`)

### Diretório de skills
O `codex-acp` lê skills de `$CODEX_HOME/skills` (default `~/.codex/skills`).
O instalador (`ai-spec-harness install`) honra `$CODEX_HOME` e distribui:
- Skills resolvidas conforme `.codex/config.toml`
- Hooks de validação em `$CODEX_HOME/hooks/`

## Modo Legado (deprecado, será removido em vX): Codex CLI stateless

\`\`\`bash
ai-spec-harness task-loop --tool codex tasks/prd-minha-feature
\`\`\`

Este modo invoca `codex exec --yolo <prompt>` sem ACP. Não produz `events.jsonl`
nem `tool_calls.md`. Mantido por compatibilidade até versão vX (ver ADR-013
§"Consequências").

## ADRs Relevantes

- [ADR-013](tasks/adr/013-codex-cli-acp-native.md) — Codex via ACP nativo (binário `codex-acp`)
- [ADR-012](tasks/adr/012-copilot-cli-acp-native.md) — Copilot via ACP nativo (precedente)
- [ADR-009](tasks/adr/009-acp-protocol-adoption.md) — pinning de SDK ACP
- [ADR-008](docs/adr/008-parity-multi-tool-invariants.md) — invariantes de paridade multi-tool
```

### `.codex/config.toml` — template proposto F1-Codex

```toml
# .codex/config.toml — Configuração Codex-native para ai-spec-harness
# Lido pelo instalador (`ai-spec-harness install --tool codex`); NÃO consumido
# diretamente por codex-acp (overrides chegam via -c flags em runtime).

[runtime]
binary       = "codex-acp"
version_min  = "0.12.0"
npm_package  = "@zed-industries/codex-acp"
docs_url     = "https://github.com/zed-industries/codex-acp"

[model]
default          = "gpt-5.5"
reasoning_effort = "medium"   # low | medium | high

[access]
default = "restricted"        # restricted | full
# Em "full", o harness adiciona:
#   -c approval_policy="never"
#   -c sandbox_mode="danger-full-access"
#   -c web_search="live"

[skills]
# Path resolvido honrando $CODEX_HOME (F3-Codex); fallback ~/.codex/skills
path = "$CODEX_HOME/skills"
lock = "skills-lock.json"     # ADR-005 — pinning de skills via SHA-256

[telemetry]
governance_telemetry = "opt-in"   # GOVERNANCE_TELEMETRY=1 ativa
```

### Esboço de `internal/runtime/specs/codex.go` — F1-Codex

```go
package specs

import "strconv"

// Constantes do runtime Codex via ACP adapter (`@zed-industries/codex-acp`).
// Política de atualização (ADR-009 + ADR-013):
//   - CodexNpmVersion e CodexSDKVersion são constantes Go pinadas. Nunca usar @latest.
//   - CodexNpmVersion só é alterada via processo audit/ (tasks/templates/skill-upgrade-decision.md).
//   - CodexSDKVersion é mantida em sincronia com go.mod por scripts/sync-acp-sdk-version.sh.
const (
    CodexNpmPackage     = "@zed-industries/codex-acp"
    CodexNpmVersion     = "0.12.0"     // versão mínima que suporta -c overrides
    CodexSDKVersion     = "v0.13.0"    // mesma do Claude/Copilot (go.mod)
    DefaultCodexModel   = "gpt-5.5"
)

// Codex retorna a Spec do runtime Codex via ACP adapter.
// Binário canônico: "codex-acp" (não confundir com "codex" da OpenAI).
// Fallback: npx --yes @zed-industries/codex-acp@<CodexNpmVersion>.
// BootstrapArgs: codexBootstrapArgs (injeta -c model, -c reasoning, feature toggles, sandbox).
func Codex() Spec {
    return newSpecWithBootstrap(
        "codex",
        "Codex (ACP)",
        "codex-acp",
        nil,
        []FallbackLauncher{
            {
                Command:   "npx",
                FixedArgs: []string{"--yes", CodexNpmPackage + "@" + CodexNpmVersion},
            },
        },
        "", // AccessModeFlag vazio — Codex passa access via -c, não flag dedicada
        CodexSDKVersion,
        CodexNpmVersion,
        CodexNpmPackage,
        codexBootstrapArgs,
    )
}

// codexBootstrapArgs replica o comportamento de
// compozy/internal/core/agent/registry_specs.go:247-278.
func codexBootstrapArgs(model, reasoning string, _ []string, accessMode AccessMode) []string {
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
    if accessMode == AccessModeFull {
        args = appendCodexOverrides(args,
            `approval_policy="never"`,
            `sandbox_mode="danger-full-access"`,
            `web_search="live"`,
        )
    }
    return args
}

func appendCodexOverrides(args []string, overrides ...string) []string {
    for _, o := range overrides {
        args = append(args, "-c", o)
    }
    return args
}
```

---

## Especificação de Arquitetura (Draft)

Mudanças propostas para F1-Codex, agrupadas por arquivo (sem código final — pertence ao PRD/TechSpec):

| Arquivo | Mudança | Notas |
|---|---|---|
| `internal/runtime/specs/spec.go` | Adicionar `AccessMode string` (consts `AccessModeRestricted`, `AccessModeFull`); adicionar método `BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string` ao `Spec` com default `nil` | Mantém Claude/Copilot inalterados. `newSpecWithBootstrap` é variant constructor. |
| `internal/runtime/specs/codex.go` | **Novo arquivo** ~80 LoC | Constantes + `Codex()` + `codexBootstrapArgs` + helper |
| `internal/runtime/specs/codex_test.go` | **Novo arquivo** ~150 LoC | T-01..T-05 espelhando `copilot_test.go` |
| `internal/runtime/runner.go` | Em `Run()`, chamar `bootstrap := r.spec.BootstrapArgs(...)` e fazer `argv = append(bootstrap, r.spec.FixedArgs...)` | Order matters: `-c` flags vêm antes do que vier em FixedArgs |
| `internal/runtime/runner.go::Job` | Adicionar campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string` | Default `""`/`AccessModeRestricted`/`nil` |
| `internal/runtime/probe/probe.go:21-24` | `adrByID["codex"] = "tasks/adr/013-codex-cli-acp-native.md"` | Sem outras mudanças no probe |
| `cmd/ai_spec_harness/task_loop.go:21-24` | `runtimeACPCatalog["codex"] = specs.Codex` | + atualizar docstring linha 19 |
| `cmd/ai_spec_harness/task_loop.go:189-192` | Novas flags `--reasoning-effort` (default `medium`), `--access-mode` (default `restricted`) | Propagar via `taskloop.Options` |
| `cmd/ai_spec_harness/task_loop_test.go:48-52` | Inverter T-14: `wantErr: false`; adicionar T-15 cobrindo `--reasoning-effort=high --access-mode=full` | Garante regressão |
| `internal/taskloop/agent.go:333-351` | **Manter** `codexInvoker` legado por backward-compat | `--runtime=legacy` continua funcional; emitir warning de depreciação opcional |
| `CODEX.md` | Reescrever 30→80 linhas conforme §"Exemplos de Configuração 2026" acima | Substitui esqueleto atual |
| `tasks/adr/013-codex-cli-acp-native.md` | **Novo ADR** seguindo `tasks/adr/000-template.md` | Documenta decisões D-01..D-09 análogas a ADR-012 |
| `tasks/prd-codex-acp-spec/` | **Novo PRD** (folder com `prd.md`, `techspec.md`, `tasks.md`) | Estrutura espelha `tasks/prd-copilot-acp-spec/` |

### Riscos e Mitigações

1. **Extensão da interface `Spec` quebra Claude/Copilot** — mitigar com método default no-op (retorna `nil`) e teste de regressão T-01/T-02 em `claude_test.go` e `copilot_test.go` garantindo que `Spec.BootstrapArgs(...)` retorna `nil` para esses runtimes.
2. **`@zed-industries/codex-acp@0.12.0` ausente em ambientes air-gapped** — fallback `npx` resolve quando há acesso ao npm registry; em air-gapped, documentar pré-instalação manual em `CODEX.md` e ADR-013. Probe deve emitir mensagem explícita citando ADR-013.
3. **Tool name aliasing ausente confunde dashboards multi-tool** — F2-Codex resolve. Mitigação interina: documentar divergência na techspec; emitir warning no `execution_report.md` quando driver=codex e tools `search_query`/`image_query` aparecerem.
4. **Sandbox `danger-full-access` é destrutivo se acionado por engano** — `--access-mode full` exige consentimento explícito; default permanece `restricted`. Documentar em `CODEX.md` os riscos e quando usar.
5. **Confusão `codex` (CLI legado) vs `codex-acp` (adapter ACP)** — `CODEX.md` deve dedicar parágrafo a essa distinção; mensagem de erro do probe cita ambos.

---

## Continuidade — PRDs Futuros

> **Estado em 2026-05-21**: F1-Copilot entregue como PRD/TechSpec/Tasks em [`tasks/prd-copilot-acp-spec/`](../../tasks/prd-copilot-acp-spec/) + ADR-012. F1-Codex é o **próximo PRD candidato**; F2-Codex e F3-Codex listados aqui são PRDs futuros independentes. F2/F3/F4 do roadmap Copilot (memória, hooks, TUI) cobrem Codex automaticamente após F1-Codex.

### F1-Codex — Codex ACP Spec (próximo)

**Escopo bruto**: ver §"Roadmap" acima. PRD a criar em `tasks/prd-codex-acp-spec/`.

**Risco**: Médio (extensão da interface `Spec`). **Esforço**: Baixo–Médio (1 sprint).

**Dependências**: ADR-012 entregue. Sem outras dependências bloqueantes.

### F2-Codex — Tool name aliasing (opcional)

**Escopo bruto**: mapa `aliasesByDriver` em `events/convert.go`; entradas Codex; flag `--preserve-tool-names` para compat.

**Risco**: Baixo. **Esforço**: Baixo (~30 LoC + testes).

**Dependências**: F1-Codex entregue.

### F3-Codex — `$CODEX_HOME` resolver

**Escopo bruto**: helper em `internal/setup/codex.go`; integração com `copyToolValidationHooks`; documentação.

**Risco**: Baixo. **Esforço**: Baixo (~50 LoC).

**Dependências**: F1-Codex entregue.

### Herdadas do roadmap Copilot (cobrem Codex sem trabalho adicional)

- **F2-Copilot — Memória 2-níveis prompt-driven** — após entregue, sessões Codex consomem `internal/memory/store.go` automaticamente.
- **F3-Copilot — Hook System Go in-process (33 canônicos)** — dispatch points `agent.*` observam Codex via mesmo mecanismo.
- **F4-Copilot — TUI Bubble Tea + daemon** — renderização Codex paritária a Claude/Copilot.

Ver detalhes em [`compozy-adaptation-copilot-2026.md` §"Roadmap"](compozy-adaptation-copilot-2026.md).

---

## Referências Cruzadas

**Compozy (leitura via `gh` — SHA `7f38c44506`)**:

- `internal/core/agent/registry_specs.go:106-122` — Spec Codex
- `internal/core/agent/registry_specs.go:247-278` — `codexBootstrapArgs` + `appendCodexConfigOverrides`
- `internal/core/agent/registry_specs.go:247-250` — `codexManagedRuntimeConfigOverrides`
- `internal/core/agent/registry_specs.go:260-268` — full access mode overrides
- `internal/core/model/constants.go:15` — `DefaultCodexModel = "gpt-5.5"`
- `internal/core/agent/registry_compat.go::codexModelRequirements` — gating `codex-acp >= 0.12.0`
- `internal/core/agent/tool_call_name.go::driverToolTitleAlias` — aliasing Codex-único
- `internal/setup/agents.go:125-127` — `CodeXHome` resolver com fallback `~/.codex`
- `internal/setup/agents.go:219` — `codexPath` helper
- `internal/setup/agents.go:271` — registro `universalAgent("codex", ...)`
- `internal/core/agents/mcpserver/server.go` — MCP reservado `compozy`/`run_agent` (precedente, não usa skills)
- `internal/core/memory/store.go` — limites de memória (precedente para F2 herdada)
- `internal/core/extension/manifest.go` — 33 hooks (precedente para F3 herdada)

**ai-spec-harness (estado em `feat/acp-runtime-claude`)**:

- `internal/runtime/specs/spec.go:12-23` — value object `Spec` + `newSpec` (a estender)
- `internal/runtime/specs/claude.go:8-22` — constantes Claude (template para constantes Codex)
- `internal/runtime/specs/claude.go:28-45` — função `Claude()` (template estrutural)
- `internal/runtime/specs/copilot.go:8-27` — constantes Copilot (precedente de pinning)
- `internal/runtime/specs/copilot.go:33-50` — função `Copilot()` (template estrutural)
- `internal/runtime/specs/copilot_test.go` — pattern de testes para `codex_test.go`
- `internal/runtime/probe/probe.go:17-24` — `adrByID` (registrar `codex`)
- `internal/runtime/probe/probe.go:29-34` — `formatLauncherUnavailable` (já tool-agnóstico)
- `internal/runtime/runner.go` — runner generalizado (a estender com `BootstrapArgs`)
- `internal/runtime/client/client.go` — `acpClient` agnóstico de IDE (reusável)
- `internal/runtime/persistence/session.go` — forense (preservar)
- `internal/runtime/watchdog.go` — `ActivityWatchdog` (preservar)
- `internal/runtime/events/convert.go` — SDK→domínio (alvo de F2-Codex aliasing)
- `internal/taskloop/agent.go:333-351` — `codexInvoker` CLI legado (manter para `--runtime=legacy`)
- `cmd/ai_spec_harness/task_loop.go:21-24` — `runtimeACPCatalog` (registrar Codex)
- `cmd/ai_spec_harness/task_loop.go:82-97` — gating ACP (sem mudanças após registro)
- `cmd/ai_spec_harness/task_loop.go:189-192` — flags ACP atuais (adicionar `--reasoning-effort`, `--access-mode`)
- `cmd/ai_spec_harness/task_loop_test.go:48-52` — T-14 a inverter
- `CODEX.md` — esqueleto atual de 30 linhas (a reescrever em F1-Codex)
- `AGENTS.md` — referência canônica (sem mudanças)
- ADR-009 (`tasks/adr/009-acp-protocol-adoption.md`) — pinning SDK (precedente)
- ADR-008 (`docs/adr/008-parity-multi-tool-invariants.md`) — paridade multi-tool (precedente)
- ADR-010 (`tasks/prd-acp-runtime-claude/adr-010-event-tagged-union.md`) — tagged union de eventos
- ADR-011 (`tasks/adr/011-agent-registry-declarativo.md`) — Agent Registry F1 anterior
- ADR-012 (`tasks/adr/012-copilot-cli-acp-native.md`) — Copilot ACP nativo (precedente direto)
- ADR-013 (`tasks/adr/013-codex-cli-acp-native.md`) — **a criar em F1-Codex**

**Pesquisa correlata**:

- [`docs/research/compozy-adaptation-analysis.md`](compozy-adaptation-analysis.md) — análise genérica (10 dimensões, F1–F5 roadmap)
- [`docs/research/compozy-adaptation-copilot-2026.md`](compozy-adaptation-copilot-2026.md) — F1-Copilot entregue; F2/F3/F4 cobrem Codex post-F1
- [`docs/prompts/compozy-adaptation-research-codex.md`](../prompts/compozy-adaptation-research-codex.md) — prompt enriquecido de origem
- [`docs/prompts/compozy-adaptation-research-claude.md`](../prompts/compozy-adaptation-research-claude.md) — variante Claude
- [`docs/prompts/compozy-adaptation-research-copilot.md`](../prompts/compozy-adaptation-research-copilot.md) — variante Copilot

---

## Apêndice — Comparação Codex vs Copilot vs Claude (visão interna do compozy)

| Aspecto | Codex | Copilot | Claude |
|---|---|---|---|
| **Command** | `codex-acp` | `copilot` | `claude-agent-acp` |
| **FixedArgs** | (vazio; usa BootstrapArgs) | `["--acp"]` | (vazio) |
| **BootstrapArgs** | `codexBootstrapArgs` (model + reasoning + features + sandbox via `-c`) | `nil` | `nil` |
| **DefaultModel** | `gpt-5.5` | (runtime-determined) | `opus` |
| **UsesBootstrapModel** | `true` | `false` | `false` |
| **Reasoning Effort** | configurável (`low`/`medium`/`high`) via `-c model_reasoning_effort` | não exposto | não exposto |
| **Sandbox Controls** | `approval_policy`, `sandbox_mode`, `web_search` (em full access) | nenhum | `--bypass-permissions` |
| **Feature Toggles** | `features.code_mode=false`, `features.code_mode_only=false` (managed runs) | nenhum | nenhum |
| **Skills Directory** | `$CODEX_HOME/skills` (default `~/.codex/skills`) | `~/.copilot/skills` | `~/.claude/skills` |
| **Fallback Package** | `@zed-industries/codex-acp` | `@github/copilot` | `@agentclientprotocol/claude-agent-acp` |
| **Min Runtime Version** | `codex-acp >= 0.12.0` (gating por modelo) | sem gating | sem gating |
| **Tool Name Aliasing** | `search_query` → `web_search`; `image_query` → `image_search` | nenhum | nenhum |
| **Setup File comitado** | nenhum | nenhum | nenhum |
