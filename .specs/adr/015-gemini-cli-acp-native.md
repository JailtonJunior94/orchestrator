# ADR-015: Gemini CLI como runtime ACP nativo

**Status:** Proposta
**Data:** 2026-05-22
**Autores:** -

---

## Contexto

O harness suporta três runtimes ACP em producao: Claude (ADR-009), Copilot (ADR-012) e Codex (ADR-013). Gemini permanece como **unico runtime ACP-capable em modo wrapper legado**. Invocacao atual em `internal/wrapper/wrapper.go:91-95`:

```go
case "gemini":
    return fmt.Sprintf(
        "Invoke Gemini with skill %q in project %s:\n  gemini run --skill %s --project %s%s",
        skill, projectDir, skill, projectDir, extraArgs,
    )
```

Esse caminho produz comando opaco (`gemini run --skill ... --project ...`) consumido pela skill `execute-task` como bloco unico. Nao gera `events.jsonl`, `tool_calls.md` nem `execution_report.md` estruturados; nao e coberto por `ActivityWatchdog` (`internal/runtime/watchdog.go`), telemetria (ADR-006) ou normalizacao de tool-calls (F2-Claude). O `runtimeACPCatalog` em `cmd/ai_spec_harness/task_loop.go:27-31` registra apenas claude/codex/copilot — Gemini e ausencia declarada. Nao existe `internal/runtime/specs/gemini.go`.

Em 2026 o Google enviou suporte ACP nativo no `@google/gemini-cli` via flag `--acp`. Probe via `npx --yes @google/gemini-cli@0.43.0 --acp --help` (2026-05-22) confirma:

```
--acp                  Starts the agent in ACP mode  [boolean]
--experimental-acp     Starts the agent in ACP mode (deprecated, use --acp instead)  [boolean]
--approval-mode        Set the approval mode: default (prompt for approval),
                       auto_edit (auto-approve edit tools),
                       yolo (auto-approve all tools),
                       plan (read-only mode)
```

O `compozy/compozy` (`main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`) registra Gemini como runtime ACP de primeira classe em `internal/core/agent/registry_specs.go::model.IDEGemini`:

```go
model.IDEGemini: {
    ID:             model.IDEGemini,
    DisplayName:    "Gemini",
    SetupAgentName: "gemini-cli",
    DefaultModel:   model.DefaultGeminiModel,  // "gemini-2.5-pro"
    Command:        "gemini",
    FixedArgs:      []string{"--acp"},
    ProbeArgs:      []string{"--acp", "--help"},
    Fallbacks: []Launcher{{
        Command:   "npx",
        FixedArgs: []string{"--yes", "@google/gemini-cli", "--acp"},
    }},
    BootstrapArgs: func(_, _ string, _ []string, _ string) []string { return nil },
}
```

Compozy mantém `BootstrapArgs` no-op para Gemini — modelo, reasoning e approval mode são configurados via `gemini config` ou flags interativas, nao propagados em tempo de spawn. Esta ADR aproveita a infraestrutura `BootstrapArgsFunc` introduzida por ADR-013 (Codex) para ir um passo alem do Compozy: traduzir o `AccessMode` do harness (`AccessModeRestricted`/`AccessModeFull`) em flag `--approval-mode` correspondente, capturando uma capability exposta pela CLI 0.43.0 que Compozy ainda nao explora.

Adicionar Gemini ao runtime ACP do harness requer **apenas configuracao** — sem nova extensao estrutural da interface `Spec` (ja estendida por ADR-013 com `BootstrapArgsFunc` e `AccessMode`). Deixar Gemini em modo wrapper legado significa: (a) caixa-preta permanente sem forense; (b) divergencia observacional crescente com Claude/Codex/Copilot; (c) impossibilidade de cascatear toda a infraestrutura F2-F5 (MCP nested-agent, normalizacao, hooks Go in-process, memory 2-tier, metricas, auto-review) que ja existe ou esta planejada para os outros runtimes; (d) bloqueio do unico caminho realista de aproveitar a janela 1M+ do `gemini-2.5-pro` com a governanca `spec-hash`/`PRD-first` do harness.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Manter wrapper legado (`gemini run --skill`) | Zero codigo novo | Caixa-preta permanente; sem forense; sem watchdog; sem telemetria; sem cascata F2-F5; divergencia crescente com Claude/Codex/Copilot |
| Adotar `gemini --acp` via ACP nativo, paridade exata com Compozy (sem `--approval-mode`) | Mudanca minima (~50 LoC); paridade direta com referencia upstream | Descarta capability do `--approval-mode` exposta pela CLI 0.43.0; AccessMode do harness fica desconectado do que Gemini executa |
| Adotar `gemini --acp` + mapeamento `AccessMode` → `--approval-mode` via `BootstrapArgs` minimal | Aproveita capability nativa; mantem semantica do AccessMode coerente cross-runtime; reusa infra ADR-013 | Diverge do Compozy (que mantem BootstrapArgs no-op); precisa testes que validem o mapeamento |
| Construir bridge Go custom sobre Gemini SDK em `internal/runtime/gemini-adapter/` | Independencia da CLI Google | Reimplementa ACP; alta complexidade; duplica `@google/gemini-cli`; ≥2 sprints; bloqueia entrega |
| Aguardar SDK upstream `@google/gemini-agent-acp` em Go | Symmetry com Claude SDK | SDK nao existe; `@google/gemini-cli` ja e o caminho real em 2026 |

## Decisao

Decidimos adotar **Gemini via ACP nativo** usando `@google/gemini-cli` com flag `--acp`, **mais** mapeamento `AccessMode` → `--approval-mode` via `BootstrapArgs` minimal. O novo `internal/runtime/specs/gemini.go` declara:

- `Command: "gemini"` (binario padrao da CLI Google — NAO `gemini-cli`, que e o nome do pacote npm)
- `FixedArgs: []string{"--acp"}` (modo ACP nativo, estavel desde 2026-Q1; alias deprecated `--experimental-acp` mantido upstream)
- Fallback unico `npx --yes @google/gemini-cli@<GeminiNpmVersion> --acp` (constante Go pinada conforme ADR-009)
- `AccessModeFlag: ""` (Gemini nao tem flag dedicada estatica de sandbox como Claude `--bypass-permissions`; mapeamento dinamico vai via `BootstrapArgs`)
- `BootstrapArgs: geminiBootstrapArgs` — funcao local que emite `--approval-mode <value>` baseado em `AccessMode`

As cinco decisoes que compoe esta ADR sao:

### D-01 — Command canonico `gemini` + `FixedArgs: ["--acp"]`

Espelha a entrada `model.IDEGemini` em `compozy/internal/core/agent/registry_specs.go`. O binario `gemini` (NAO `gemini-cli`, que e o nome do pacote npm) e instalado via `npm install -g @google/gemini-cli` ou invocavel via `npx --yes @google/gemini-cli`. A flag `--acp` foi estavel desde 0.39.x (anteriormente `--experimental-acp`); a 0.43.0 mantem ambas com `--experimental-acp` marcada como deprecated.

### D-02 — Pinning npm via audit/

Constantes Go pinadas em `internal/runtime/specs/gemini.go`:

```go
const (
    GeminiNpmPackage   = "@google/gemini-cli"
    GeminiNpmVersion   = "0.43.0"  // dist-tag latest em 2026-05-22
    GeminiSDKVersion   = "v0.13.0"  // mesma de Claude/Codex/Copilot — sincronizada com go.mod
    DefaultGeminiModel = "gemini-2.5-pro"  // espelha compozy/internal/core/model/constants.go
)
```

`GeminiNpmVersion` atualizada **somente** via processo `audit/` (mesma politica de ADR-009 D-06 e ADR-013 D-06). Versao pinada validada via `npm view @google/gemini-cli version dist-tags`. Branches `preview` (0.44.x) e `nightly` ativas mas nao adotadas em producao.

### D-03 — `BootstrapArgs` minimal (mapeia AccessMode → --approval-mode)

Diferente do `BootstrapArgs: nil` que Compozy usa, o harness adota uma `BootstrapArgsFunc` que **so** emite `--approval-mode` baseado em `AccessMode`:

```go
func geminiBootstrapArgs(_, _ string, _ []string, mode AccessMode) []string {
    switch mode {
    case AccessModeFull:
        return []string{"--approval-mode", "yolo"}
    case AccessModeRestricted, "":
        return []string{"--approval-mode", "default"}
    default:
        return []string{"--approval-mode", "default"}
    }
}
```

`model` e `reasoning` sao **ignorados** intencionalmente (sublinhados `_`): Gemini propaga modelo via `--model` (flag separada que o `ACPRunner.Run()` ja adiciona quando `Job.Model != ""`); reasoning effort nao e exposto via CLI no Gemini 0.43.0 (`thoughts_tokens` aparecem na resposta sem controle prevenivel pelo cliente). `addDirs` e ignorado porque Gemini 0.43.0 nao expoe flag equivalente a Claude `--add-dir`.

### D-04 — `AccessModeFlag: ""` na `Spec`

`Spec.AccessModeFlag` permanece vazio porque Gemini **nao tem flag estatica dedicada** equivalente a Claude `--bypass-permissions`. O controle de access mode ocorre via `--approval-mode` que e dinamico (depende do `AccessMode` do `Job`), por isso vai via `BootstrapArgs` (D-03), nao via campo estatico da `Spec`. Isto preserva a interface `Spec` introduzida em ADR-013 sem alteracoes.

### D-05 — Mapeamento explicito `AccessMode` → `--approval-mode`

Decisao concreta sobre quais valores de `--approval-mode` o harness usa:

| `specs.AccessMode` | `--approval-mode` | Justificativa |
|---|---|---|
| `AccessModeRestricted` (default) | `default` | Comportamento conservador: Gemini pede aprovacao antes de cada tool destrutivo. Equivalente semantico ao Claude sem `--bypass-permissions`. |
| `AccessModeFull` | `yolo` | Auto-aprova todas as ferramentas (incluindo destrutivas). Equivalente semantico ao Claude `--bypass-permissions` e ao Codex `approval_policy="never" + sandbox_mode="danger-full-access"`. |

Os outros valores expostos pela CLI Gemini sao **deliberadamente nao usados**:

- `auto_edit` (auto-aprova apenas edit tools) — semantica nao mapeavel ao binario `AccessMode` do harness; reintroduzir um terceiro estado quebraria simetria com Claude/Codex. Reservado para futura extensao se demanda concreta surgir.
- `plan` (read-only mode) — semantica de "leitura-apenas" colide com o modelo de execucao do harness (tasks precisam editar arquivos para fechar). Util para sessoes investigativas mas fora do escopo `task-loop`.

Esta tabela e a **fonte de verdade**: testes T-29 e T-30 (ver "Consequencias") validam o mapeamento literal. Mudanca futura deste mapeamento requer nova ADR.

## Consequencias

### Positivas

- Paridade observacional Gemini ↔ Claude ↔ Codex ↔ Copilot: `events.jsonl`, `tool_calls.md`, `execution_report.md`, `ActivityWatchdog` e telemetria opt-in (ADR-006) passam a cobrir Gemini
- Cascata automatica de toda a infraestrutura F2-F5 (MCP nested-agent, normalizacao tool-calls, hooks Go in-process, memory 2-tier, auto-review opt-in) ja existente ou planejada para Claude — Gemini herda sem codigo extra alem do `Spec`
- D-03/D-05 aproveitam capability `--approval-mode` que Compozy ainda nao explora — harness ganha controle granular de approval semantically aligned com `AccessMode`
- Reusa 100% do stack ACP validado por Claude/Codex/Copilot: `ACPRunner`, `acpClient`, `persistence`, `events`, `watchdog`
- Reaproveita ADR-009 (pinning SDK) e ADR-013 (interface `Spec` com `BootstrapArgsFunc`) sem mudanca estrutural — apenas mais um spec no catalogo
- Janela 1M+ tokens do `gemini-2.5-pro` fica acessivel via runtime forense (com `effective_context_tokens` futuro em F4-Gemini)

### Negativas / Riscos

- Diverge do Compozy em D-03/D-05 (Compozy mantem `BootstrapArgs: nil`). Risco de drift se Compozy futuramente decidir popular `--approval-mode` com semantica diferente. Mitigacao: documentar o mapeamento em `GEMINI.md` e em comentario de codigo no `geminiBootstrapArgs`; revisao quando `audit/` atualizar `GeminiNpmVersion`.
- Flag `--acp` da CLI Gemini ainda relativamente nova (estavel desde 0.39.x); risco de breaking change em 0.44.x ou 1.0.0. Mitigacao: `--experimental-acp` existe como alias deprecated, indicando que Google mantem ciclo de compatibilidade; pinning via D-02 limita exposure.
- Mapeamento `AccessModeFull → yolo` aciona auto-aprovacao total no Gemini — consentimento operacional do usuario e pre-condicao. Mitigacao: warning `accessModeFullWarnOnce` (mesma infra de ADR-013 em `cmd/ai_spec_harness/task_loop.go:19-21`) emite aviso na primeira invocacao por execucao.
- Tool name aliasing especifico Gemini **nao** e implementado nesta fase — Gemini herda tabela `common` (Compozy comprova suficiencia em `tool_call_name.go:84`). Decisao registrada como F2-Gemini no roadmap, nao bloqueante.
- Metricas Gemini-2026 (`cache_read_tokens`, `effective_context_tokens`, `prompt_tokens_billed`, `thoughts_tokens`) nao sao extraidas em F0/F1-Gemini — capability fica para F4-Gemini. Sem perda funcional, apenas latencia em telemetria opt-in.

### Neutras / Observacoes

- `internal/wrapper/wrapper.go::ValidTools["gemini"] = true` (`internal/wrapper/wrapper.go:14-18`) **permanece** durante transicao. Quando `--runtime` nao for `acp`, emite warning informativo de deprecation. Remocao planejada para release N+2, mesma politica de ADR-012 Q5 (Copilot) e ADR-013 (Codex).
- `internal/taskloop/compatibility.go:34-43` ja contem modelos Gemini (`gemini-2.5-pro`, `pro`, `flash`, `flash-lite`, `gemini-3-pro-preview`) — sem alteracao necessaria.
- `GEMINI.md` raiz e atualizado para documentar `--runtime=acp` como recomendado, com secao "Runtime Capabilities (F0-Gemini+)" listando `--approval-mode` mapping e flags relacionadas.
- Hooks shell em `.gemini/hooks/*.sh` (post-execute-task, validate-governance, etc.) coexistem sem conflito com hooks Go in-process do dispatcher (F3-Claude/F3-Gemini), exatamente como `.claude/hooks/*.sh` coexistem com Go hooks no caso Claude.
- Subcomandos nativos da CLI Gemini 0.43.0 (`gemini hooks migrate`, `gemini skills install`, `gemini mcp`) sao **complementares** ao harness e nao concorrentes; uso fica a criterio do usuario em modo interativo. Modo orquestrado (ACP) nao depende deles.
- A `Spec` Gemini nao seta `SupportsAddDirs` nem equivalente porque Compozy nao seta (default zero `false`) e a CLI Gemini 0.43.0 nao expoe flag de workspace extras. Se mudar upstream, ajuste vai via nova ADR ou audit/.

### Testes obrigatorios (F0+F1-Gemini)

A entrega da decisao desta ADR deve incluir os seguintes testes:

- **T-13 estendido** (`cmd/ai_spec_harness/task_loop_test.go`): `TestRuntimeACPCatalogIncludesGemini` — verifica que `runtimeACPCatalog["gemini"]` retorna `specs.Gemini()` valido.
- **T-14 estendido**: `TestGeminiSpecHasCorrectCommandAndFlags` — `Command == "gemini"`, `FixedArgs == ["--acp"]`, `Fallbacks[0].Command == "npx"`, `Fallbacks[0].FixedArgs == ["--yes", "@google/gemini-cli@0.43.0", "--acp"]`.
- **T-15 estendido**: `TestGeminiFallbackResolvesViaNpx` — quando binario `gemini` ausente do PATH, fallback npx e usado.
- **T-16 estendido**: `TestGeminiBootstrapArgsForRestricted` — `geminiBootstrapArgs("", "", nil, AccessModeRestricted) == ["--approval-mode", "default"]`.
- **T-29 novo**: `TestGeminiBootstrapArgsForFull` — `geminiBootstrapArgs("", "", nil, AccessModeFull) == ["--approval-mode", "yolo"]`.
- **T-30 novo**: `TestGeminiBootstrapArgsIgnoresModelAndReasoning` — `geminiBootstrapArgs("gemini-2.5-pro", "high", []string{"/tmp"}, AccessModeRestricted) == ["--approval-mode", "default"]` (model, reasoning, addDirs nao afetam o output).
- **T-31 novo**: `TestGeminiBootstrapArgsDefaultsToRestricted` — `geminiBootstrapArgs("", "", nil, "") == ["--approval-mode", "default"]` (zero-value AccessMode mapeia para default conservador).
- **T-32 novo**: smoke test em `tests/integration/gemini_acp_smoke_test.go` (ou paridade com `tests/integration/claude_2026_e2e_test.go`) — invoca `gemini --acp` real (skipable via `-short`) e verifica que `events.jsonl` e produzido para task simples.

## Referencias

- ADR-009 — `.specs/adr/009-acp-protocol-adoption.md` (pinning SDK)
- ADR-008 — `docs/adr/008-parity-multi-tool-invariants.md` (paridade)
- ADR-011 — `.specs/adr/011-agent-registry-declarativo.md` (Agent Registry)
- ADR-012 — `.specs/adr/012-copilot-cli-acp-native.md` (precedente: CLI principal + `--acp`)
- ADR-013 — `.specs/adr/013-codex-cli-acp-native.md` (precedente direto: `BootstrapArgsFunc`, `AccessMode`)
- ADR-014 — `.specs/adr/014-claude-cli-acp-native.md` (paralelo: cascata F2-F5)
- Compozy — `internal/core/agent/registry_specs.go::model.IDEGemini` (Spec Gemini)
- Compozy — `internal/core/model/constants.go::DefaultGeminiModel` (`"gemini-2.5-pro"`)
- Compozy — `internal/core/agent/tool_call_name.go:84` (Gemini herda tabela `commonToolTitleAliases`)
- Probe — `npx --yes @google/gemini-cli@0.43.0 --acp --help` validado em 2026-05-22
- Pesquisa — `docs/research/compozy-adaptation-gemini-2026.md` (gap map, roadmap F0..F5, e adendo de validacao 0.43.0)
- npm package — https://www.npmjs.com/package/@google/gemini-cli (dist-tag `latest = 0.43.0` em 2026-05-22)
- Docs upstream — https://geminicli.com
