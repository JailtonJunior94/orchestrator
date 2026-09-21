@AGENTS.md

# Claude Code — ai-spec-harness

`AGENTS.md` (importado acima) e a fonte canonica: stack, comandos, convencoes, estrutura, CI e ADRs
vivem la. Este arquivo contem apenas o que e especifico do Claude Code.

## Carga de contexto

- Regras transversais em `.claude/rules/` carregam em toda sessao: `code-style.md` (R-STYLE-001, hard: codigo em ingles, zero comentarios) e `governance.md` (R-GOV-001).
- Em tarefa de execucao, carregar `.agents/skills/agent-governance/SKILL.md` e a skill da linguagem afetada. Skills de planejamento (`create-prd`, `create-technical-specification`, `create-tasks`) so quando a tarefa pedir.
- `references/` de cada skill sao carregadas sob demanda conforme o gatilho declarado no SKILL.md, nunca em bloco.

## Skills, subagentes e hooks

- `.claude/skills/` e espelho de `.agents/skills/` (gate `make check-skills-sync`); editar sempre em `.agents/skills/` e rodar `scripts/sync-skills.sh`.
- `.agents/policies/` e a origem canonica de `R-GOV-001`/`R-STYLE-001`; `.claude/rules/` (e equivalentes) sao espelhos derivados. Editar sempre em `.agents/policies/` e rodar `scripts/sync-policies.sh` (gate `make check-policies-sync`, ver tabela "Area tocada | Gate" em `AGENTS.md`).
- `.claude/agents/` define 8 subagentes (`task-executor`, `reviewer`, `bugfixer`, `refactorer`, `prd-writer`, `technical-specification-writer`, `task-planner`, `project-analyzer`). Use-os para execucao isolada; a sessao principal nao deve acumular contexto de implementacao.
- Hooks shell em `.claude/hooks/` **nao** estao ativos por default: `ai-spec install .` os registra em `.claude/settings.local.json` (nao versionado). Confirmar com `/hooks`.
- `validate-governance.sh` roda como hook de **pos-ferramenta** (`internal/runtime/specs/registry.go`, `scriptPostTool`): com hooks ativos ele **informa** edicao de `AGENTS.md` e de `SKILL.md` com `exit 1`, mas o `exit` ocorre depois que a edicao ja aconteceu — `AfterTool` nao bloqueia em nenhuma das quatro CLIs suportadas (Claude, Codex, Copilot, OpenCode). Exportar `GOVERNANCE_HOOK_MODE=warn` quando a tarefa exigir editar esses arquivos suprime o aviso pos-edicao, nao um bloqueio previo. Detalhes: [`docs/hooks-canonicos.md`](docs/hooks-canonicos.md).

## Runtime orquestrado (`ai-spec task-loop --tool claude --runtime acp`)

- Flags Claude-especificas: `--mcp-nested`, `--auto-review`, `--no-normalize`, `--disable-hooks`, `--durable-memory`, `--memory-workflow-limit-lines`. Defaults preservam o comportamento F1. Detalhes em [`docs/runtime-claude-capabilities.md`](docs/runtime-claude-capabilities.md).
- Hooks Go (`internal/runtime/hooks/`) servem o modo orquestrado; hooks shell servem o modo interativo. Nao se sobrepoem e o harness nunca altera `.claude/hooks/*.sh`.
- `runtime.pre_open` aborta a sessao se `AGENTS.md` nao existir no WorkDir.
- Quando `.specs/<prd>/memory/` existe, a memoria do harness vence a auto-memory do Claude Code.
- Telemetria opt-in: `GOVERNANCE_TELEMETRY=1`; relatorio com `ai-spec telemetry report`.

## Ao compactar

Preservar a lista de arquivos alterados, os comandos de validacao ja executados com resultado e o
PRD/tarefa ativa.
