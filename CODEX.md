# ai-spec-harness — Codex

Use `AGENTS.md` como fonte canonica das regras deste repositorio. Stack, comandos, convencoes, estrutura, CI e padroes estao documentados em `AGENTS.md` — nao duplicados aqui.

## Instrucoes

1. Ler `AGENTS.md` no inicio da sessao — e a instrucao principal de sessao para este repositorio.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. `.codex/config.toml` lista as skills habilitadas para resolucao e upgrade via harness.
4. Em tarefas de execucao, carregar apenas `AGENTS.md`, `agent-governance` e a skill operacional da linguagem ou atividade afetada.
5. Skills de planejamento (`analyze-project`, `create-prd`, `create-technical-specification`, `create-tasks`) entram apenas quando a tarefa pedir esse fluxo explicitamente.
6. Carregar referencias adicionais apenas quando a tarefa exigir.
7. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
8. Validar mudancas com comandos proporcionais ao risco.

## Hooks de Governanca

Hook de preload: `.codex/hooks/validate-preload.sh` (instalado via `ai-spec-harness install`).
Se o hook nao estiver presente ou falhar, consulte `.codex/docs/workaround-preload.md`.

## Orientacoes Especificas para Codex

O Codex le `AGENTS.md` como instrucao de sessao e `.codex/config.toml` como metadados de skills para resolucao pelo harness. Para manter compliance:

1. Ao iniciar uma sessao, confirmar que `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` foram lidos.
2. Descobrir skills disponiveis consultando os paths listados em `.codex/config.toml`.
3. Seguir as etapas procedurais do SKILL.md de cada skill invocada.
4. Ao final da tarefa, executar os comandos de validacao descritos na secao Validacao do `AGENTS.md`.
5. Enforcement depende do modelo seguir as instrucoes — nao ha bloqueio automatico.
