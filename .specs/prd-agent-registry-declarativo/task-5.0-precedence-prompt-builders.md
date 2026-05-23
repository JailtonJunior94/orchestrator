# Tarefa 5.0: Precedence runtime + builders de prompt do agente

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar dois componentes puros do package `internal/agents/`:
1. `applyRuntimePrecedence(cfg *RuntimeOverride, defaults RuntimeDefaults)` — aplica CLI flags > AGENT.md defaults > harness defaults (D-05 / RF-13).
2. `BuildAgentBlocks(agent *ResolvedAgent, catalog []ResolvedAgent) (metadata, catalogBlock string)` — produz dois blocos textuais para enriquecimento de prompt (RF-15, RF-16).

<requirements>
- `RuntimeOverride` carrega flags explícitas (`ExplicitIDE`, `ExplicitModel`, etc.) para distinguir "não setado" de "setado vazio".
- Quando `Explicit*` é `false`, o campo é preenchido pelo default do agente.
- `BuildAgentBlocks` retorna catálogo ordenado lexicograficamente por `name` (RF-16).
- Catálogo limita a 200 entradas; entradas excedentes são descartadas silenciosamente (RF-16).
- Agente ativo marcado com `[active]` no catálogo.
- Componentes são funções puras — sem IO, sem cache, sem estado.
</requirements>

## Subtarefas

- [ ] 5.1 Criar `internal/agents/precedence.go` com `RuntimeOverride` e `applyRuntimePrecedence`.
- [ ] 5.2 Criar `internal/agents/prompt.go` com `BuildAgentBlocks`.
- [ ] 5.3 Definir formato exato dos blocos (metadata em Markdown estruturado; catálogo em lista bullet).
- [ ] 5.4 Adicionar testes T-10, T-11, T-12 (precedence) e T-15, T-16 (prompt builders).

## Detalhes de Implementação

Ver techspec, seção **Design de Implementação → Interfaces Chave** (blocos `precedence.go` e `prompt.go`) e ADR-011 → Decisão D-05.

Formato sugerido do bloco metadata:
```markdown
### Agente Ativo

- **Nome**: claude-revisor-rigoroso
- **Versão**: 1.0.0
- **Descrição**: Revisor de PR com viés conservador e foco em invariantes
- **Runtime**: claude / claude-opus-4-7 / high / bypass-permissions
```

Formato sugerido do catálogo:
```markdown
### Agentes Disponíveis

- `claude-revisor-rigoroso` [active] — Revisor de PR com viés conservador
- `codex-refator-incremental` — Refator incremental com foco em paridade
- ...
```

## Critérios de Sucesso

- T-10: CLI `--model X` + AGENT.md `model: Y` → resolve para `X` (CLI vence).
- T-11: CLI sem `--model` + AGENT.md `model: Y` → resolve para `Y`.
- T-12: nenhum setado → `RuntimeOverride` permanece vazio (caller usa harness defaults).
- T-15: catálogo de 3 entradas → bloco ordenado lex; agente ativo marcado.
- T-16: catálogo de 250 entradas → truncado para 200.
- Funções puras: testes não usam `fs.FileSystem` nem outros doubles.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-10: precedência CLI vence AGENT.md.
- [ ] T-11: AGENT.md preenche quando CLI ausente.
- [ ] T-12: vazio quando nenhum setado.
- [ ] T-15: catálogo formatado e ordenado.
- [ ] T-16: truncamento a 200.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/agents/precedence.go` (novo)
- `internal/agents/prompt.go` (novo)
- `internal/agents/precedence_test.go` (novo)
- `internal/agents/prompt_test.go` (novo)
