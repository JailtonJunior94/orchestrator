# Tarefa 4.0: Hooks nativos por-tool + sandbox/approval Codex + matriz + sync gate

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Transformar paridade "por instrução" em "por enforcement": configurar hooks nativos no formato de
cada CLI (Claude, Codex, Copilot, Gemini), todos invocando os mesmos validadores compartilhados
(cascata da 1.0); suplementar o Codex com sandbox/approval; reescrever a matriz de enforcement;
criar gate de sync. Cobre RF-05, RF-10..RF-13. Ver ADR-002.

<requirements>
- `.claude/settings.json` (`hooks.PreToolUse`/`Stop`/`SubagentStop`).
- `.codex/hooks.json` + `[[hooks.PreToolUse]]` em `.codex/config.toml` + `sandbox_mode`/`approval_policy`.
- `.github/hooks/governance.json` (`version:1`, `preToolUse`/`agentStop`).
- `.gemini/settings.json` (`hooks.BeforeTool`/`AfterAgent`).
- Todos os hooks chamam os validadores via cascata `.agents/scripts/`/`.agents/hooks/`.
- `enforcement-matrix.md` reescrita para a realidade 2026 (incl. caveat Codex).
- `scripts/check-scripts-sync.sh` + alvo `Makefile`; `execute-task`/`execute-all-tasks` falham
  de forma bloqueante quando o tool é capaz e o `.sh` não roda (sem "modo legado" silencioso).
</requirements>

## Subtarefas

- [ ] 4.1 Criar os 4 configs de hook por-tool apontando para os validadores compartilhados.
- [ ] 4.2 Adicionar `sandbox_mode`/`approval_policy` ao Codex com comentário sobre a lacuna.
- [ ] 4.3 Reescrever `enforcement-matrix.md` (hooks nativos nos 4 tools; caveat Codex).
- [ ] 4.4 Criar `scripts/check-scripts-sync.sh` + alvo no `Makefile` + sync nos mirrors.
- [ ] 4.5 Remover degradação silenciosa em `execute-task`/`execute-all-tasks` (falha bloqueante).

## Detalhes de Implementação

Ver techspec "Hooks nativos por-tool" e "Comportamento de hook ausente" + ADR-002. Formatos das
docs oficiais 2026 (Claude/Codex/Copilot/Gemini).

## Critérios de Sucesso

- Cada um dos 4 tools tem config de hook válido apontando para o validador compartilhado.
- Codex possui `sandbox_mode` + `approval_policy` configurados.
- `enforcement-matrix.md` reflete hooks nativos 2026 e o caveat do Codex.
- `make check-scripts-sync` falha em drift e passa quando sincronizado.
- Hook ausente em tool capaz → estado `failed`/`blocked` (não "modo legado" silencioso).

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Fixtures por-tool validam o formato de cada config de hook.
- [ ] Teste: report sem critério de aceite comprovado é bloqueado pelo caminho de hook.
- [ ] `make check-scripts-sync` verde.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.claude/settings.json`, `.codex/hooks.json`, `.codex/config.toml`, `.github/hooks/governance.json`, `.gemini/settings.json`
- `.agents/skills/agent-governance/references/enforcement-matrix.md`
- `scripts/check-scripts-sync.sh`, `Makefile`
- `.agents/skills/execute-task/SKILL.md`, `.agents/skills/execute-all-tasks/SKILL.md`
