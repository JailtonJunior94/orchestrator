# Guia de Migração: Legacy → `--runtime acp`

> Relacionado: [ADR-012](../.specs/adr/012-copilot-cli-acp-native.md) (Copilot ACP),
> [ADR-013](../.specs/adr/013-codex-cli-acp-native.md) (Codex ACP),
> [ADR-015](../.specs/adr/015-gemini-cli-acp-native.md) (Gemini ACP),
> ADR-022 (guard de governança) e ADR-026 (sunset do legacy mode) — ambos na pasta do PRD
> `.specs/prd-paridade-cross-cli/`.

Os entrypoints legados (`codex exec`, Copilot sem ACP, wrapper `gemini run --skill`) coexistem com os
runtimes ACP nativos apenas durante a janela de depreciação. Eles **dobram a superfície de divergência**
e não recebem novas features de paridade — só o runtime ACP recebe. Este guia descreve como migrar.

## Por que migrar

- **Paridade absoluta cross-CLI:** normalização de tool-calls, métricas unificadas, memória 2-tier e
  guard de governança só são garantidos no runtime ACP (`--runtime acp`).
- **Comportamento idêntico entre as 4 CLIs:** mesma task → mesmos eventos/normalização/métricas
  (validado pela suíte `internal/parity`, RP-03, gate de CI obrigatório).
- **Manutenção única:** o legacy será removido (ver critério abaixo).

## Como migrar por CLI

O modo ACP é ativado com a flag `--runtime acp` no `task-loop`. O harness auto-detecta o binário ACP
e cai para o fallback `npx` quando necessário (ADR-017).

| CLI | Legacy (descontinuado) | ACP nativo (recomendado) | Binário / fallback |
|---|---|---|---|
| Codex | `codex exec --yolo` | `task-loop --tool codex --runtime acp` | `codex-acp` · `npx @zed-industries/codex-acp` |
| Copilot | Copilot sem ACP | `task-loop --tool copilot --runtime acp` | `copilot --acp` · `npx @github/copilot --acp` |
| Gemini | `gemini run --skill` (wrapper) | `task-loop --tool gemini --runtime acp` | `gemini --acp` · `npx @google/gemini-cli --acp` |
| Claude | — (já é ACP nativo) | `task-loop --tool claude --runtime acp` | `claude-agent-acp` · `npx @agentclientprotocol/claude-agent-acp` |

### Exemplo

```bash
# Antes (legacy)
ai-spec task-loop --tool codex .specs/prd-minha-feature

# Depois (ACP nativo — paridade garantida)
ai-spec task-loop --tool codex --runtime acp .specs/prd-minha-feature
```

### Notas de comportamento ao migrar

- **Guard de governança (ADR-022):** no runtime ACP, sessões de execução de task com `tasks.md`
  rastreável validam spec-hash/PRD-first (RG-01/RG-02). Se um PRD existente tiver hash defasado,
  rode `ai-spec sync-spec-hash` ou use `--skip-drift-guard` (desabilita apenas esse guard).
- **Watchdog:** o timeout de inatividade default permanece 120s; configure via `--activity-timeout`
  ou `config.yaml` (`runtime.timeout`). `--activity-timeout=0` desabilita.
- **Memória / janela:** CLIs de janela grande (ex.: Gemini ≥ 1M) usam limites ampliados
  automaticamente; flags `--memory-*` explícitas continuam prevalecendo.

## Critério de remoção do legacy (ADR-026)

A remoção do legacy mode é uma **tarefa futura** (não executada neste ciclo). Pré-condições objetivas,
todas verificáveis, conforme ADR-026:

1. Paridade RP-01..RP-04 verde nas 4 CLIs via runtime ACP (suíte `internal/parity` no CI — RG-03). ✅
2. Guard de governança (ADR-022) ativo e estável. ✅
3. Guia de migração publicado em `docs/` (este documento). ✅
4. Nenhum consumidor interno (self-dogfooding) dependente do legacy.

**Release alvo:** próxima minor após a estabilização das fases de paridade deste PRD (a versão será
fixada na tarefa de remoção, não antecipada aqui — ADR-026).

## Tarefa futura de remoção (registro)

> Registrada aqui por durabilidade (a pasta `.specs/` é gitignored). Mover para o tracker oficial quando
> as pré-condições 1–4 estiverem satisfeitas.

- **Título:** Remover legacy mode (`copilotInvoker`/`codexInvoker` em `internal/taskloop/agent.go` +
  `internal/wrapper/wrapper.go`).
- **Pré-condição (gate):** itens 1–4 do critério de remoção acima, com ênfase no item 4 (nenhum
  consumidor interno dependente do legacy).
- **Escopo:** remover os invokers legados e o wrapper; remover suas flags/mensagens de depreciação;
  manter apenas o caminho `--runtime acp`. Atualizar testes e docs.
- **Fora de escopo deste ciclo:** nenhum código legado é removido agora (ADR-026 §1).
