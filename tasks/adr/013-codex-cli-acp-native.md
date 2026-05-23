# ADR-013: Codex CLI como runtime ACP nativo

**Status:** Proposta
**Data:** 2026-05-21
**Autores:** -

---

## Contexto

O harness suporta dois runtimes ACP em produção: Claude (ADR-009) e Copilot (ADR-012). Codex hoje é invocável apenas em modo legado stateless via `internal/taskloop/agent.go:335-351`:

```go
func (c *codexInvoker) Invoke(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
    args := make([]string, 0, 5)
    args = append(args, "exec")
    if model != "" { args = append(args, "--model", model) }
    args = append(args, "--yolo", prompt)
    return runCmd(ctx, workDir, c.liveOut, "codex", args...)
}
```

Esse caminho invoca o binário `codex` (CLI legacy da OpenAI, v0.132.0 atual) e não produz `events.jsonl`, `tool_calls.md`, `execution_report.md` nem é coberto por `ActivityWatchdog` ou telemetria opt-in. O gate em `cmd/ai_spec_harness/task_loop.go:82-97` bloqueia explicitamente `--tool codex --runtime acp` (validado por T-14 em `task_loop_test.go:48-52`).

Em 2026 o ecossistema ACP ganhou um adapter dedicado para Codex: o pacote npm `@zed-industries/codex-acp` (atualmente em v0.14.0 stable; v0.12.0 é a mínima para `gpt-5.5`). O `compozy/compozy` (`main` SHA `7f38c445069bd83a8e96bcd925ee1f12fde74435`) registra Codex como runtime ACP em pé de igualdade com Claude/Copilot, mas com três peculiaridades documentadas em `internal/core/agent/registry_specs.go:106-122`:

```go
model.IDECodex: {
    ID: model.IDECodex, DisplayName: "Codex",
    SetupAgentName:     "codex",
    DefaultModel:       model.DefaultCodexModel,  // "gpt-5.5"
    Command:            "codex-acp",
    SupportsAddDirs:    true,
    UsesBootstrapModel: true,
    Fallbacks:          []Launcher{{Command: "npx", FixedArgs: []string{"--yes", "@zed-industries/codex-acp"}}},
    DocsURL:            "https://github.com/zed-industries/codex-acp",
    BootstrapArgs:      codexBootstrapArgs,
},
```

Três divergências do padrão Copilot (`registry_specs.go:222-242`):

1. Binário canônico é **`codex-acp`** (adapter da Zed Industries), não `codex` (CLI da OpenAI). Confusão de nomenclatura comum.
2. `UsesBootstrapModel: true` e `BootstrapArgs: codexBootstrapArgs` — Codex injeta model/reasoning/sandbox via pares `-c key="value"` em tempo de spawn, não via `FixedArgs` estáticos.
3. A função `codexBootstrapArgs` (`registry_specs.go:247-278`) emite: `model="<name>"`, `model_reasoning_effort="<level>"`, `features.code_mode=false`, `features.code_mode_only=false`; em `AccessModeFull` adiciona `approval_policy="never"`, `sandbox_mode="danger-full-access"`, `web_search="live"`.

Adicionar Codex ao runtime ACP do harness requer extensão estrutural (não apenas configuração) porque o value object `Spec` atual em `internal/runtime/specs/spec.go:12-23` não comporta `BootstrapArgs` dinâmico — Claude e Copilot são puramente declarativos. Deixar Codex em modo legado significa: (a) caixa-preta permanente sem forense; (b) divergência observacional crescente com Claude/Copilot; (c) bloqueio de ADR-008 (paridade multi-tool) para o terceiro runtime; (d) impossibilidade de expor reasoning effort e sandbox controls que diferenciam Codex.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Manter `codexInvoker` legado (`codex exec --yolo`) | Zero código novo | Caixa-preta permanente; sem forense; sem watchdog; sem telemetria; divergência crescente com Claude/Copilot; reasoning effort e sandbox indisponíveis |
| Adotar `codex-acp` via ACP nativo com extensão de `Spec` para `BootstrapArgs` | Paridade observacional total; reusa `ACPRunner`, `persistence`, `watchdog`, `events`; expõe reasoning/access mode como CLI flags; remove dívida de paridade | Requer estender interface `Spec` (impacto retrocompatível com default no-op); requer flags CLI novas (`--reasoning-effort`, `--access-mode`); pin novo `@zed-industries/codex-acp@0.14.0` |
| Hardcodar `codex-acp` args em `runner.go` (não generalizar `Spec`) | Mudança mínima | Mantém Claude/Copilot-centrismo; cada novo runtime futuro com bootstrap dinâmico (Droid já tem `--model/--reasoning-effort` flat; Gemini pode ter outro padrão) exigirá outro hardcode |
| Aguardar SDK upstream `@openai/codex-agent-acp` em Go | Symmetry com Claude SDK | SDK não existe; o adapter `@zed-industries/codex-acp` é o caminho real em 2026; bloqueia entrega indefinidamente |
| Implementar wrapper Go interno que invoque `codex` CLI legado e traduza para JSON-RPC ACP | Reusa binário já instalado | Reimplementa protocolo ACP; alta complexidade; duplica `acpClient` |

## Decisão

Decidimos adotar **Codex via ACP nativo** usando o adapter `codex-acp` (`@zed-industries/codex-acp`), seguindo o padrão do compozy. O novo `internal/runtime/specs/codex.go` declara:

- `Command: "codex-acp"` (binário canônico do adapter da Zed Industries)
- `FixedArgs: nil` (não injeta flags estáticas; toda configuração vai via `BootstrapArgs`)
- Fallback único `npx --yes @zed-industries/codex-acp@<CodexNpmVersion>` (constante Go pinada, política ADR-009)
- `AccessModeFlag: ""` (Codex passa access mode via `-c approval_policy=...` etc., não flag dedicada)
- `BootstrapArgs: codexBootstrapArgs` — função local replicando `compozy/internal/core/agent/registry_specs.go:247-278`

Para suportar isso, a interface `Spec` é estendida de forma retrocompatível:

1. `internal/runtime/specs/spec.go:12-23` ganha tipo `AccessMode string` (consts `AccessModeRestricted = "restricted"`, `AccessModeFull = "full"`) e método `BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string` com default no-op (retorna `nil`). Claude e Copilot herdam o no-op sem mudança de comportamento.

2. `internal/runtime/runner.go` em `Run()` chama `r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)` e faz prepend ao argv do launcher antes das `FixedArgs`.

3. `internal/runtime/runner.go::Job` ganha campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string`. Defaults: `""`, `AccessModeRestricted`, `nil`.

4. `cmd/ai_spec_harness/task_loop.go:21-24` adiciona `"codex": specs.Codex` ao `runtimeACPCatalog`. Linha 82-97 (gate) passa a aceitar Codex automaticamente.

5. `cmd/ai_spec_harness/task_loop.go` ganha duas flags novas:
   - `--reasoning-effort` (default `"medium"`; valores aceitos: `low`, `medium`, `high`)
   - `--access-mode` (default `"restricted"`; valores aceitos: `restricted`, `full`)
   Propagadas via `taskloop.Options` até `Job.ReasoningEffort` / `Job.AccessMode`.

6. `internal/runtime/probe/probe.go:21-24` ganha `"codex": "tasks/adr/013-codex-cli-acp-native.md"` em `adrByID`.

7. `cmd/ai_spec_harness/task_loop_test.go:48-52` (T-14) é **invertido**: `--tool codex --runtime acp` passa a ser aceito. Novo caso T-15 cobre `--reasoning-effort high --access-mode full`.

O `codexInvoker` CLI legado em `internal/taskloop/agent.go:335-351` é **mantido** como rota de compatibilidade quando `--runtime=legacy` (default), com aviso de depreciação via `sync.Once` na primeira invocação por execução do processo. `CODEX.md` raiz é reescrito para documentar o caminho ACP como recomendado.

## Consequências

### Positivas

- Paridade observacional Codex ↔ Claude ↔ Copilot: `events.jsonl`, `tool_calls.md`, `execution_report.md`, `ActivityWatchdog` e telemetria (ADR-006) passam a cobrir Codex
- Remove dívida arquitetural: terceiro runtime sai do tier `BestEffort` em ADR-008
- Extensão da interface `Spec` (com `BootstrapArgs`) desbloqueia Droid e outros runtimes futuros que injetam config dinamicamente (Droid usa `--model X --reasoning-effort Y`; pode ser modelado como BootstrapArgs também)
- Expõe reasoning effort e access mode como CLI flags — capabilities únicas de Codex aproveitadas em vez de descartadas
- Reutiliza 100% do stack ACP já validado por Claude/Copilot: `ACPRunner`, `acpClient`, `persistence`, `events`, `watchdog`
- Reaproveita ADR-009 (pinning SDK) para versionar `@zed-industries/codex-acp` com o mesmo rigor

### Negativas / Riscos

- Interface `Spec` ganha método `BootstrapArgs(...)` — mudança retrocompatível via default no-op, mas amplia surface para revisão. Mitigado por T-05/T-19 (regressão Claude/Copilot).
- Duas flags CLI novas (`--reasoning-effort`, `--access-mode`) — risco de confusão se passadas com `--tool claude` ou `--tool copilot` (devem ser ignoradas pelo `BootstrapArgs` no-op). Documentado em help text.
- `codex-acp >= 0.12.0` é versão mínima para `gpt-5.5`; pin inicial será `0.14.0` (último stable em 2026-05-21 via `npm view @zed-industries/codex-acp versions`). Ambientes air-gapped sem npx exigem instalação manual.
- `AccessModeFull` aciona `sandbox_mode="danger-full-access"` no Codex — consentimento operacional do usuário é pré-condição. Documentado em `CODEX.md` com warning explícito.
- Tool name aliasing Codex (`search_query` → `web_search`, `image_query` → `image_search`) **não** é implementado nesta fase — telemetria Codex usará nomes nativos. Decisão registrada como follow-up (F2-Codex no roadmap), não bloqueante.

### Neutras / Observações

- Parity check (ADR-008) deve promover Codex de ausente para suportado conforme F1-Codex entregar testes
- `codexInvoker` legado (`codex exec --yolo`) permanece operacional via `--runtime=legacy` por 2 versões minor (mesma política de Q5 de ADR-012 para Copilot)
- `CODEX.md` raiz atual (esqueleto de 30 linhas) é reescrito para documentar `--runtime=acp` como recomendado
- Se o ecossistema `@zed-industries/codex-acp` for descontinuado upstream, o `codexInvoker` legado continua como fallback hard
- Confusão de nomenclatura `codex` (OpenAI CLI legado) vs `codex-acp` (Zed Industries adapter) é documentada explicitamente em `CODEX.md`

## Referencias

- ADR-009 — `tasks/adr/009-acp-protocol-adoption.md` (pinning SDK)
- ADR-008 — `docs/adr/008-parity-multi-tool-invariants.md` (paridade)
- ADR-010 — `tasks/prd-acp-runtime-claude/adr-010-event-tagged-union.md`
- ADR-011 — `tasks/adr/011-agent-registry-declarativo.md` (Agent Registry)
- ADR-012 — `tasks/adr/012-copilot-cli-acp-native.md` (precedente direto F1-Copilot)
- Compozy — `internal/core/agent/registry_specs.go:106-122` (Codex Spec) e `:247-278` (`codexBootstrapArgs`)
- Compozy — `internal/core/agent/registry_compat.go::codexModelRequirements` (gating `codex-acp >= 0.12.0`)
- Pesquisa — `docs/research/compozy-adaptation-codex-2026.md` (gap map e roadmap detalhados)
- Adapter upstream — https://github.com/zed-industries/codex-acp
- npm package — https://www.npmjs.com/package/@zed-industries/codex-acp (v0.14.0 stable em 2026-05-21)
