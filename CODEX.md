# Codex — ai-spec-harness

Use `AGENTS.md` como fonte canonica das regras deste repositorio. Stack, comandos, convencoes, estrutura, CI e padroes estao documentados em `AGENTS.md` — nao duplicados aqui.

## Modo Recomendado (2026): Codex via ACP

Em 2026 o adapter `codex-acp` (`@zed-industries/codex-acp`) passou a expor o motor
Codex via Agent Client Protocol (ACP) nativo. O harness suporta esse modo via
`--runtime=acp --tool=codex`, atingindo paridade observacional total com Claude e Copilot:
persistencia forense, watchdog de inatividade e telemetria opt-in cobrem invocacoes Codex
com os mesmos campos e granularidade.

Ver: [ADR-013](.specs/adr/013-codex-cli-acp-native.md) — Codex CLI como runtime ACP nativo.

### Pre-requisitos

- `codex-acp` versao >= 0.12.0 (verifique com `codex-acp --version`)
  Versao minima documentada em `internal/runtime/specs/codex.go`.
- Variaveis Codex configuradas conforme docs do `@zed-industries/codex-acp`.
- Alternativa sem binario local: `npx --yes @zed-industries/codex-acp@0.12.0` e
  acionado automaticamente como fallback pelo harness.

> **Atencao — confusao de nomenclatura (2026)**: existem dois CLIs com nomes parecidos:
> - `codex` — CLI legado da OpenAI (stateless, invoca `codex exec --yolo`). **Nao e ACP**.
> - `codex-acp` — adapter ACP da Zed Industries (`@zed-industries/codex-acp`). **Este e o correto para `--runtime=acp`**.
>
> O harness legado (`--runtime=legacy`) invoca `codex`; o modo recomendado invoca `codex-acp`.
> Nao substituir um pelo outro sem atualizar a configuracao.

### Uso

```bash
ai-spec-harness task-loop \
  --tool codex \
  --runtime acp \
  --reasoning-effort medium \
  --access-mode restricted \
  .specs/prd-minha-feature
```

A sessao produz os mesmos artefatos forenses do modo Claude/Copilot:

- `events.jsonl` — stream linha-a-linha de eventos ACP
- `tool_calls.md` — agregado de tool calls da sessao
- `execution_report.md` — summary final com metadados

Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) registra invocacoes com `tool=codex`,
`launcher=binary|npx`, `npm_version` e `sdk_version` reais.

### Flags Codex-especificas

- `--reasoning-effort {low,medium,high}` (default: `medium`) — controla
  `model_reasoning_effort` injetado via `-c model_reasoning_effort="<level>"` no `codex-acp`.
- `--access-mode {restricted,full}` (default: `restricted`) — modo de acesso do sandbox.

> **AVISO — `--access-mode=full`**: ativa `sandbox_mode="danger-full-access"`,
> `approval_policy="never"` e `web_search="live"`. Use **somente em ambientes isolados**.
> Em ambientes compartilhados ou de producao, nunca use `full` sem revisar o que o agente
> pode executar. O default `restricted` e sempre a escolha segura. Ver ADR-013 D-08.

### Timeout de inatividade

O `ActivityWatchdog` cancela sessoes Codex inativas com o mesmo timeout configuravel
que cancela sessoes Claude/Copilot (default: `120s`):

```bash
ai-spec-harness task-loop \
  --tool codex \
  --runtime acp \
  --activity-timeout 90s \
  .specs/prd-minha-feature
```

### Erro de launcher indisponivel

Se nem `codex-acp` nem `npx` estiverem disponiveis, o harness retorna:

```
codex-acp nao encontrado.
  1. Instale o adapter ACP: npm install -g @zed-industries/codex-acp@0.12.0
  2. OU use npx (fallback automatico se npm estiver disponivel).
  3. OU use --runtime=legacy (modo legado; sem artefatos forenses).
  Ver: .specs/adr/013-codex-cli-acp-native.md
```

---

## Modo Legado (deprecado): Codex CLI stateless

> **Status:** Deprecado. Sera removido em 2 versoes minor apos esta — ver
> [ADR-013](.specs/adr/013-codex-cli-acp-native.md) §"Consequencias" (D-05).
> Migrar para `--runtime=acp` conforme secao acima.

```bash
ai-spec-harness task-loop \
  --tool codex \
  .specs/prd-minha-feature
```

Este modo invoca `codex exec --yolo <prompt>` (CLI legado da OpenAI, sem ACP).
Nao produz `events.jsonl` nem `tool_calls.md`. Mantido por compatibilidade
por 2 versoes minor enquanto o aviso de deprecacao esta ativo.

Na primeira invocacao do processo o harness emite:

```
WARNING: Codex CLI em modo legado (sem ACP). Migrar para --runtime=acp --tool=codex.
         O modo legado sera removido em vX.Y.Z. Ver ADR-013.
```

---

## Governanca

Regras transversais, precedencia e restricoes operacionais estao definidas em:

- `AGENTS.md` — instrucao de sessao e contrato de carga base
- `.agents/skills/agent-governance/SKILL.md` — governanca para analise e alteracao de codigo
- `.claude/rules/governance.md` — precedencia e politica de evidencia

Regras essenciais:

1. Ler `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` antes de editar codigo.
2. Toda alteracao deve ser justificavel pelo PRD, por regra explicita ou por necessidade tecnica demonstravel.
3. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
4. Validar mudancas com comandos proporcionais ao risco.
5. Nao inventar contexto ausente.
6. Nao executar acoes destrutivas sem pedido explicito.

---

## ADRs Relevantes

| ADR | Titulo | Status |
|-----|--------|--------|
| [ADR-013](.specs/adr/013-codex-cli-acp-native.md) | Codex CLI como runtime ACP nativo | Aceita |
| [ADR-012](.specs/adr/012-copilot-cli-acp-native.md) | Copilot CLI como runtime ACP nativo (precedente) | Aceita |
| [ADR-009](.specs/adr/009-acp-protocol-adoption.md) | Adocao do ACP via coder/acp-go-sdk | Aceita |
| [ADR-008](docs/adr/008-parity-multi-tool-invariants.md) | 29 invariantes 3 niveis (paridade multi-tool) | Aceita |

---

## Mecanismo de Carregamento de Contexto

### Codex CLI via ACP — Modo Recomendado 2026

Com `--runtime=acp`, o harness conecta ao servidor ACP do `codex-acp` e transmite
contexto via protocolo JSON-RPC sobre stdio. O template de prompt e
`internal/taskloop/executor_template.tmpl`. Governanca e injetada no prompt da tarefa.

### `.codex/config.toml` — skills habilitadas

O arquivo `.codex/config.toml` lista as skills habilitadas para resolucao e upgrade via harness.
O instalador (`ai-spec-harness install --tool codex`) distribui hooks de validacao em
`$CODEX_HOME/hooks/` (default `~/.codex/hooks/`) e skills em `$CODEX_HOME/skills/`.

---

## Limitacoes Conhecidas

| Capacidade | Claude Code | Codex ACP | Codex CLI legado |
|---|---|---|---|
| Persistencia forense (events/tools/report) | Sim | Sim | Nao |
| ActivityWatchdog | Sim | Sim | Nao |
| Telemetria opt-in | Sim | Sim | Nao |
| Contexto automatico | CLAUDE.md | via prompt ACP | Nao |
| Estado de sessao persistente | Sim | Sim | Nao — stateless |
| Tool name aliasing canonico | Sim | Adiado (F2-Codex) | N/A |
