# ADR-012: GitHub Copilot CLI como runtime ACP nativo

**Status:** Aceita — substitui [ADR-007](../../docs/adr/007-copilot-cli-stateless-workaround.md)
**Data:** 2026-05-21
**Autores:** -

---

## Contexto

A ADR-007 (2026-04-20) declarou que o `gh copilot` CLI era stateless: cada invocação ignorava `.github/copilot-instructions.md`, `COPILOT.md` e qualquer artefato de governança, tornando enforcement automático impossível e justificando um workaround manual via injeção `#file:` no prompt do usuário.

Em 2026 o GitHub Copilot CLI ganhou um modo servidor ACP (`copilot --acp`) compatível com o protocolo `coder/acp-go-sdk` já consumido pelo harness para Claude. O `compozy/compozy` (`main` `7f38c445069bd83a8e96bcd925ee1f12fde74435`) registra o Copilot como runtime ACP em pé de igualdade com Claude/Codex/Gemini/Cursor/Droid/OpenCode/Pi:

```go
// internal/core/agent/registry_specs.go:222-242
model.IDECopilot: {
    ID: model.IDECopilot, DisplayName: "Copilot CLI",
    SetupAgentName: "github-copilot",
    Command:   "copilot",
    FixedArgs: []string{"--acp"},
    ProbeArgs: []string{"--acp", "--help"},
    Fallbacks: []Launcher{{Command: "npx",
        FixedArgs: []string{"--yes", "@github/copilot", "--acp"}, ...}},
    DocsURL: "https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server",
},
```

O pressuposto técnico de ADR-007 — "CLI stateless sem ponto de extensão" — deixou de ser verdadeiro. Manter o workaround manual significa: (a) Copilot continuar caixa-preta sem `events.jsonl`, `tool_calls.md`, `execution_report.md`, `ActivityWatchdog` e telemetria forense (ADR-006); (b) divergência observacional permanente com Claude; (c) acumular dívida em invariantes de paridade multi-tool (ADR-008).

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Manter ADR-007 (workaround manual `#file:`) | Zero código novo | Caixa-preta permanente, sem forense, sem watchdog, sem telemetria; divergência crescente com Claude |
| Adotar Copilot via ACP nativo (`copilot --acp`) com fallback `npx --yes @github/copilot --acp` | Paridade observacional total com Claude; reusa `ACPRunner`, `persistence`, `watchdog`, `events`; remove dívida de ADR-007 | Requer generalizar `runner.go` e `probe/probe.go` (hoje Claude-specific em versão/erro); requer documentar versão mínima do `copilot` CLI |
| Aguardar SDK upstream `@github/copilot-agent-acp` empacotado em Go | Symmetry total com Claude SDK pinning (ADR-009) | SDK não existe em 2026-05; bloqueia entrega indefinidamente |
| Implementar Copilot via plugin custom para `gh` | Controle total de governança | Alta complexidade, manutenção contínua, dependência de API instável da `gh` extension |

## Decisão

Decidimos adotar **Copilot via ACP nativo** seguindo o padrão do compozy. O novo `internal/runtime/specs/copilot.go` declara:

- `Command: "copilot"` (binário canônico)
- `FixedArgs: ["--acp"]`
- Fallback único `npx --yes @github/copilot --acp` (versão npm pinada em constante Go, alinhada ao padrão de ADR-009)
- `AccessModeFlag` apropriado quando o Copilot CLI documentar (sem `--bypass-permissions` análogo no v0)

Para suportar isso, o runtime deixa de ser Claude-centric:

1. `internal/runtime/runner.go:113-120` deixa de hardcodear `specs.ClaudeSDKVersion` / `specs.ClaudeNpmVersion` no payload de `runtime_init`. As versões passam a ser derivadas do `Spec` resolvido (via método `Spec.SDKVersion() string` e/ou metadados do `Launcher`).
2. `internal/runtime/probe/probe.go:70-82` deixa de assumir `specs.ClaudeNpmPackage`; o template de erro passa a usar metadata do `Spec` recebido.
3. `cmd/ai_spec_harness/task_loop.go:77` (gating `runtime acp suporta apenas --tool claude`) é generalizado para aceitar qualquer `Spec` registrado no catálogo. Modelo: tabela `runtimeACPCatalog map[string]specs.Spec`.

ADR-007 é formalmente **substituída**. `COPILOT.md` raiz é reescrito para documentar o caminho ACP. A flag `--tool copilot --runtime acp` torna-se o caminho recomendado; o `copilotInvoker` CLI legado em `internal/taskloop/agent.go:381-388` é mantido por uma versão como rota de compatibilidade, com aviso de depreciação no log.

## Consequências

### Positivas

- Paridade observacional Copilot ↔ Claude: `events.jsonl`, `tool_calls.md`, `execution_report.md`, `ActivityWatchdog` e telemetria (ADR-006) passam a cobrir Copilot
- Remove dívida arquitetural: ADR-007 deixa de ser obstáculo a ADR-008 (paridade multi-tool)
- Generalização do `runner.go`/`probe.go` desbloqueia futuros runtimes ACP (Codex, Gemini, Droid) quando seus SDKs estabilizarem
- Reutiliza 100% do stack já validado: `ACPRunner`, `acpClient` (`internal/runtime/client/client.go`), `persistence`, `events`, `watchdog` — sem nova camada
- Reaproveita ADR-009 (pinning SDK) para versionar `@github/copilot` com o mesmo rigor de `@agentclientprotocol/claude-agent-acp`

### Negativas / Riscos

- Requer documentar a versão mínima do Copilot CLI que expõe `--acp` (a ser confirmada na techspec da F1)
- Auth flow do Copilot pode exigir `gh auth refresh` ou token válido — o harness precisa surface erros desse caminho sem inventar `--bypass-permissions` análogo
- `runtime_init` evento ganha cardinalidade — invariantes ADR-008 (paridade) e ADR-010 (tagged union) precisam reconfirmar que o campo `tool=copilot` é aceito por todos os consumidores downstream
- `copilotInvoker` legado mantido por uma versão duplica caminho de execução — risco baixo se gateado por flag explícita

### Neutras / Observações

- Parity check (ADR-008) deve promover Copilot CLI de `BestEffort` para os mesmos tiers de Claude conforme F1 entregar testes
- `.github/copilot-instructions.md` segue funcional para Copilot Chat no editor; coexiste com o caminho ACP do CLI sem conflito
- ADR-007 permanece no histórico (não é deletado) marcado como **substituído por ADR-012** para preservar rastreabilidade
- Se o GitHub remover o modo `--acp` upstream, o caminho legado em `copilotInvoker` continua disponível como fallback hard

## Referencias

- ADR-007 (substituída) — `docs/adr/007-copilot-cli-stateless-workaround.md`
- ADR-009 — `.specs/adr/009-acp-protocol-adoption.md` (pinning SDK)
- ADR-008 — `docs/adr/008-parity-multi-tool-invariants.md` (paridade)
- ADR-010 — `.specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md`
- Compozy — `internal/core/agent/registry_specs.go:222-242` (referência canônica do Spec Copilot ACP)
- Pesquisa — `docs/research/compozy-adaptation-copilot-2026.md` (gap map e roadmap)
- Documentação upstream — https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server
