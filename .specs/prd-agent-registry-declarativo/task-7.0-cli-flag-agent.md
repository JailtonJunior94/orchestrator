# Tarefa 7.0: Adicionar flag CLI --agent com validação de exclusividade

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Expor a nova flag `--agent <name>` em `cmd/ai_spec_harness/task_loop.go`. Validar mutuamente exclusividade com `--tool` (modo simples) e com flags do modo avançado (`--executor-tool`, `--reviewer-tool`). Resolve Q1 do PRD via decisão D-06 do ADR-011.

<requirements>
- Flag `--agent <name>` adicionada ao parser de flags do task-loop CLI.
- `--agent` + `--tool` → erro `ErrFlagsConflitantes` (ou erro novo semanticamente equivalente).
- `--agent` + `--executor-tool` (ou `--reviewer-tool`) → erro de conflito.
- `--agent` + `--model` / `--reasoning-effort` é **permitido** — flags compõem override sobre defaults do AGENT.md (RF-13).
- Help text descritivo: explica a relação com `--tool` e como o agente é resolvido (workspace > global).
</requirements>

## Subtarefas

- [ ] 7.1 Adicionar flag `--agent` em `cmd/ai_spec_harness/task_loop.go`.
- [ ] 7.2 Adicionar validação de exclusividade antes de chamar `ResolveProfiles` ou `ResolveProfileFromAgent`.
- [ ] 7.3 Documentar comportamento no help text (`flag.StringVar(..., "...")`).
- [ ] 7.4 Adicionar testes T-20 e T-21.

## Detalhes de Implementação

Ver techspec, seção **Design de Implementação → Arquivos Modificados** (`cmd/ai_spec_harness/task_loop.go`) e ADR-011 → Decisão D-06.

Padrão de erro existente: `internal/taskloop/profile.go:13-18` define `ErrFlagsConflitantes`. Reutilizar ou criar erro adjacente semanticamente equivalente.

Help text sugerido:
```
--agent <name>   Resolve agente declarativo (AGENT.md) em workspace ou global.
                 Mutuamente exclusivo com --tool, --executor-tool, --reviewer-tool.
                 Flags --model e --reasoning-effort sobrescrevem defaults do AGENT.md.
```

## Critérios de Sucesso

- T-20: `--agent foo --tool claude` → erro `ErrFlagsConflitantes`-like.
- T-21: `--agent foo --executor-tool codex` → erro de conflito.
- `--agent foo --model X` → executa com agent + override de model (não é conflito).
- `--help` mostra documentação clara da flag.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-20: `--agent` + `--tool` → erro.
- [ ] T-21: `--agent` + `--executor-tool` → erro.
- [ ] Caso positivo: `--agent` + `--model` é aceito.
- [ ] Snapshot ou regex sobre `--help` confirma presença da flag.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `cmd/ai_spec_harness/task_loop.go` (modificado)
- `cmd/ai_spec_harness/task_loop_test.go` (modificado)
