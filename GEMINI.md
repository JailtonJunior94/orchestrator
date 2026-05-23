# ai-spec-harness — Gemini CLI

> Use `AGENTS.md` como fonte canonica das regras deste repositorio. Stack, comandos, convencoes, estrutura, CI e padroes estao documentados em `AGENTS.md` — nao duplicados aqui.

## Instrucoes

1. Ler `AGENTS.md` no inicio da sessao.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. `.gemini/commands/` sao adaptadores finos que apontam para a habilidade correta
   (apenas no modo wrapper legado; no modo ACP, skills sao invocadas pelo runner).
4. Em tarefas de execucao, carregar apenas `AGENTS.md`, `agent-governance` e a skill operacional da linguagem ou atividade afetada.
5. Skills de planejamento (`analyze-project`, `create-prd`, `create-technical-specification`, `create-tasks`) entram apenas quando a tarefa pedir esse fluxo explicitamente.
6. Carregar referencias adicionais apenas quando a tarefa exigir.
7. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
8. Validar mudancas com comandos proporcionais ao risco.

## Hooks de Governanca

Hook de preload: `.gemini/hooks/validate-preload.sh` (instalado via `ai-spec-harness install`).
Se o hook nao estiver presente ou falhar, consulte `.gemini/docs/workaround-preload.md`.

Shell hooks em `.gemini/hooks/*.sh` servem o **modo interativo** (uso direto do `gemini` CLI pelo usuario).
Go hooks em `internal/runtime/hooks/` servem o **modo orquestrado** (ACPRunner via `--runtime acp`).
Os dois conjuntos coexistem sem conflito. Para portar hooks interativos: `gemini hooks migrate`.

## Runtime Capabilities (F0-Gemini+)

Quando o Orchestrator invoca Gemini via ACP (`--runtime acp --tool gemini`), o runtime expoe:

```bash
ai-spec task-loop --tool gemini --runtime acp .specs/prd-X
```

- **ACP nativo** via `gemini --acp`. Fallback: `npx --yes @google/gemini-cli@0.43.0 --acp`.
  Modelo default: `gemini-2.5-pro`. Pinning em `internal/runtime/specs/gemini.go`.
- **Spec**: `Command="gemini"`, `FixedArgs=["--acp"]`, `BootstrapArgs=nil` — modelo, reasoning
  e sandbox sao configurados pelo gemini upstream via gemini config; nao propagamos via flags `-c`.
- **Mapeamento D-05** (ADR-015): `AccessModeRestricted → --approval-mode=default`;
  `AccessModeFull → --approval-mode=yolo`. Divergencia intencional do Compozy documentada em ADR-015.
  **Warning**: `--access-mode=full` equivale a `yolo` — sem confirmacao de tool calls. Usar com cautela.

### Modo Legado Wrapper (deprecated)

```bash
# Wrapper legado — sem events.jsonl, sem metricas, sem cascata F2-F5
gemini run --skill <name> --project <dir>
```

`internal/wrapper/wrapper.go` preserva o wrapper legado durante a transicao. Emite warning de
deprecation via `sync.Once` quando invocado. **Remocao planejada para release N+2** apos ADR-015.
Para migrar: substituir por `--runtime acp --tool gemini`.

## Runtime Capabilities (F2-Gemini+)

```bash
ai-spec task-loop --tool gemini --runtime acp --mcp-nested .specs/prd-X
```

- **MCP nested-agent** (`--mcp-nested`): expoe tool `run_agent(agent_name, prompt, model?, timeout?)`
  via protocolo MCP stdio. Profundidade maxima: `AISPEC_MAX_AGENT_DEPTH` (default 3). Child sessions
  produzem `events.jsonl` e `execution_report.md` em sub-dir proprio. Eventos espelhados no parent
  com kind `nested_agent`. Implementado por `internal/runtime/mcpserver/` (tool-agnostico; cascata
  automatica apos F0/F1-Gemini).
- **Tool-call normalization** (sempre ativa por default a partir de F2-Gemini): nomes de tool
  canonicalizados via `.agents/normalization-rules.yaml`. Gemini herda tabela `common`
  (`inherit: common`) — sem overrides especificos (Compozy confirma que Gemini usa nomes proximos
  ao schema canonico). `events.jsonl` ganha `normalized_name` e `raw_name` lado a lado.
  `--no-normalize` desabilita (debug). Implementado em `internal/runtime/events/normalize.go`.

## Runtime Capabilities (F3-Gemini+)

```bash
ai-spec task-loop --tool gemini --runtime acp \
  --memory-workflow-limit-lines 250 .specs/prd-X
```

- **Hooks in-process Go**: pontos canonicos `runtime.pre_open`, `prompt.pre_build`,
  `prompt.post_build`, `tool_call.pre_dispatch`, `tool_call.post_complete`, `session.post_end`.
  Compartilhados com Claude/Codex/Copilot via `internal/runtime/hooks/dispatcher.go` (tool-agnostico).
  `--disable-hooks` desabilita todos (debug).
- **Memoria 2-tier** com **defaults Gemini-generosos** (aproveitando janela 1M+):
  workflow 250 linhas / 20 KiB; task 400 linhas / 32 KiB (vs 150/12 KiB e 200/16 KiB defaults Claude).
  Override via `--memory-workflow-limit-lines`, `--memory-task-limit-lines`, etc.
  `NeedsCompaction=true` anexa diretiva textual de compactacao ao prompt.
  Implementado em `internal/runtime/memory/store.go` (tool-agnostico).
  **Trade-off**: janela 1M+ barateia re-carga vs Claude, mas latencia inicial do prompt-build e
  custo de cache lookup sobem com prompt maior.

## Runtime Capabilities (F4-Gemini+)

- **Evidence Gemini-2026**: `execution_report.md` ganha secao "Metricas Gemini-2026" com:
  - `cache_read_tokens` — tokens lidos do context cache Gemini (TTL configuravel; diferente do Claude)
  - `effective_context_tokens` — tamanho real do contexto carregado na sessao
  - `prompt_tokens_billed` — tokens efetivamente cobrados apos cache hit
  - `thoughts_tokens` — tokens de reasoning interno Gemini 2.5 (opt-in; pode ser zero por default)
- Captura via `internal/runtime/events/gemini_metrics.go` (extracao defensiva — campo ausente nao bloqueia).
- Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`): entries `gemini.cache_read`, `gemini.thoughts`,
  `gemini.effective_context`, `gemini.prompt_billed` aparecem no relatorio final.
- **Caveat**: `thoughts_tokens` pode ser sempre zero em Gemini 2.5 quando reasoning nao e exposto
  por default — valor zero e semanticamente valido, nao e erro.

## Runtime Capabilities (F5-Gemini+)

```bash
ai-spec task-loop --tool gemini --runtime acp --auto-review .specs/prd-X
```

- **Auto-review** (opt-in `--auto-review`, default-off): apos sessao principal spawna nova ACPRunner
  com skill `review` + git diff como prompt. Resultado em `evidence/<task>/review.md`.
  Issues com tag `[HARD]` → `Summary.ReviewStatus="blocked"`. Recursao hard-bloqueada (child Job
  tem `AutoReview=false` forcado). Hook `session.post_review` disparado apos review.
- **Warning de custo amplificado**: auto-review em Gemini com diff grande pode ultrapassar quota
  de tokens da org — Gemini 2.5 Pro com janela 1M+ potencialmente preenchida amplifica custo vs
  Claude/Codex (~200K). Usar seletivamente em tasks de alto risco.

## Orientacoes Especificas para Gemini

1. Ao iniciar uma tarefa, ler `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` como contexto base antes de editar codigo.
2. No modo ACP (`--runtime acp`), skills sao invocadas pelo runner diretamente — sem necessidade de `@workspace.<command>`.
3. No modo wrapper legado, usar `@workspace.<command>` para invocar o wrapper TOML correspondente em `.gemini/commands/` e evitar colisao com comandos nativos das skills.
4. Seguir as etapas procedurais do SKILL.md carregado pelo comando como se fossem instrucoes sequenciais.
5. Ao final da tarefa, executar os comandos de validacao descritos na secao Validacao do `AGENTS.md`.
6. Nao confiar em enforcement automatico — a compliance depende de seguir as instrucoes procedurais manualmente.
