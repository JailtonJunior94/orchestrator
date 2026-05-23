# Copilot — ai-spec-harness

Use `AGENTS.md` como fonte canonica das regras deste repositorio. Stack, comandos, convencoes, estrutura, CI e padroes estao documentados em `AGENTS.md` — nao duplicados aqui.

## Modo Recomendado (2026): Copilot via ACP

Em 2026 o GitHub Copilot CLI passou a expor servidor ACP nativo (`copilot --acp`).
O harness suporta esse modo via `--runtime=acp --tool=copilot`, atingindo paridade
observacional total com Claude: persistencia forense, watchdog de inatividade e
telemetria opt-in cobrem invocacoes Copilot com os mesmos campos e granularidade.

Ver: [ADR-012](.specs/adr/012-copilot-cli-acp-native.md) — Copilot CLI como runtime ACP nativo (substitui [ADR-007](docs/adr/007-copilot-cli-stateless-workaround.md)).

### Pre-requisitos

- `copilot` CLI versao >= `CopilotMinCLIVersion` (verifique com `copilot --version`).
  Versao minima documentada em `internal/runtime/specs/copilot.go`.
- `gh auth status` deve mostrar token Copilot valido.
- Alternativa sem binario local: `npx --yes @github/copilot@<pin> --acp` e
  acionado automaticamente como fallback pelo harness.

### Uso

```bash
ai-spec-harness task-loop \
  --tool copilot \
  --runtime acp \
  .specs/prd-minha-feature
```

A sessao produz os mesmos artefatos forenses do modo Claude:

- `events.jsonl` — stream linha-a-linha de eventos ACP
- `tool_calls.md` — agregado de tool calls da sessao
- `execution_report.md` — summary final com metadados

Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) registra invocacoes com `tool=copilot`,
`launcher=binary|npx`, `npm_version` e `sdk_version` reais.

### Timeout de inatividade

O `ActivityWatchdog` cancela sessoes Copilot inativas com o mesmo timeout configuravel
que cancela sessoes Claude (default: `120s`):

```bash
ai-spec-harness task-loop \
  --tool copilot \
  --runtime acp \
  --activity-timeout 90s \
  .specs/prd-minha-feature
```

### Erro de launcher indisponivel

Se nem `copilot` nem `npx` estiverem disponiveis, o harness retorna:

```
copilot nao encontrado.
  1. Instale copilot CLI:  https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server
  2. OU instale @github/copilot@<pin> via npm.
  3. OU use --runtime=legacy.
  Ver: .specs/adr/012-copilot-cli-acp-native.md
```

---

## Modo Legado (deprecado): Copilot CLI stateless

> **Status:** Deprecado. Sera removido em versao futura — ver [ADR-012](.specs/adr/012-copilot-cli-acp-native.md) §"Consequencias".
> Migrar para `--runtime=acp` conforme secao acima.

```bash
ai-spec-harness task-loop \
  --tool copilot \
  .specs/prd-minha-feature
```

Este modo invoca `copilot --autopilot --yolo -p <prompt>` sem ACP.
Nao produz `events.jsonl` nem `tool_calls.md`. Mantido por compatibilidade
por uma versao minor enquanto o aviso de deprecacao esta ativo.

Na primeira invocacao do processo o harness emite:

```
WARNING: Copilot CLI em modo legado (sem ACP). Migrar para --runtime=acp.
         O modo legado sera removido em vX.Y.Z. Ver ADR-012.
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
| [ADR-012](.specs/adr/012-copilot-cli-acp-native.md) | Copilot CLI como runtime ACP nativo | Aceita |
| [ADR-007](docs/adr/007-copilot-cli-stateless-workaround.md) | Copilot CLI stateless workaround | Substituida por ADR-012 |
| [ADR-009](.specs/adr/009-acp-protocol-adoption.md) | Adocao do ACP via coder/acp-go-sdk | Aceita |
| [ADR-008](docs/adr/008-parity-multi-tool-invariants.md) | 29 invariantes 3 niveis (paridade multi-tool) | Aceita |

---

## Mecanismo de Carregamento de Contexto

### Copilot Chat (VS Code / GitHub.com) — Suporte Nativo

O arquivo `.github/copilot-instructions.md` e carregado automaticamente como contexto
de repositorio pelo GitHub Copilot Chat na extensao VS Code (v1.143+) e no GitHub.com.

- **Arquivo de contexto automatico:** `.github/copilot-instructions.md`
- **Escopo:** instruido automaticamente em todas as conversas do Copilot Chat no repositorio
- **Este arquivo (`COPILOT.md`):** documentacao de governanca; referenciavel via `#file:COPILOT.md`

### Copilot CLI via ACP — Modo Recomendado 2026

Com `--runtime=acp`, o harness conecta ao servidor ACP do `copilot` CLI e transmite
contexto via protocolo JSON-RPC sobre stdio. O template de prompt e
`internal/taskloop/executor_template.tmpl`. Governanca e injetada no prompt da tarefa.

---

## Limitacoes Conhecidas (historico — modo legado)

| Capacidade | Claude Code | Copilot ACP | Copilot CLI legado |
|---|---|---|---|
| Persistencia forense (events/tools/report) | Sim | Sim | Nao |
| ActivityWatchdog | Sim | Sim | Nao |
| Telemetria opt-in | Sim | Sim | Nao |
| Contexto automatico | CLAUDE.md | via prompt ACP | Nao |
| Estado de sessao persistente | Sim | Sim | Nao — stateless |
