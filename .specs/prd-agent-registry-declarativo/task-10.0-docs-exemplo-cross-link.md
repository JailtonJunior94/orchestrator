# Tarefa 10.0: Documentação AGENTS.md + exemplo AGENT.md + cross-link ADR-011

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Tarefa de documentação. Atualizar `AGENTS.md` (raiz do repo) com nova seção "Agent Registry" referenciando o ADR-011 e o PRD/TechSpec. Criar exemplo canônico de `AGENT.md` em `docs/agents/example/AGENT.md` (path final a definir na implementação). Atualizar `docs/research/compozy-adaptation-analysis.md` com nota de "Fase 1 concluída" se aplicável.

<requirements>
- `AGENTS.md` ganha seção "Agent Registry" com:
  - Breve descrição (1 parágrafo).
  - Link para `.specs/adr/011-agent-registry-declarativo.md`.
  - Link para `.specs/prd-agent-registry-declarativo/prd.md` e `techspec.md`.
  - Caminho do exemplo canônico de `AGENT.md`.
- Exemplo `AGENT.md` autossuficiente: frontmatter válido + corpo de prompt minimal + comentários explicativos no Markdown (não no YAML).
- ADR-011 já criado durante `create-technical-specification` — esta tarefa apenas garante cross-link.
- `docs/research/compozy-adaptation-analysis.md` — opcional: atualizar seção "Próximos PRDs" sinalizando Fase 1 entregue.
</requirements>

## Subtarefas

- [ ] 10.1 Adicionar seção "Agent Registry" em `AGENTS.md`.
- [ ] 10.2 Criar exemplo canônico em `docs/agents/example/AGENT.md` (ou path consistente com o repo).
- [ ] 10.3 Verificar que ADR-011 está referenciado em índice de ADRs (se houver) ou em `AGENTS.md`.
- [ ] 10.4 Verificar que CLAUDE.md (se houver entrada de ADRs) inclui ADR-011.
- [ ] 10.5 Atualizar `docs/research/compozy-adaptation-analysis.md` se Fase 1 estiver completa (delegado à tarefa 5 do plano superior).

## Detalhes de Implementação

CLAUDE.md atual lista ADRs em uma tabela em `## ADRs` — adicionar entrada `011 — .specs/adr/011-agent-registry-declarativo.md — Agent Registry declarativo`.

Exemplo canônico de `AGENT.md` (referência inicial; pode ser refinado):

```markdown
---
name: example-claude-reviewer
description: Exemplo de agente declarativo (revisor Claude com viés conservador)
version: 1.0.0
runtime:
  ide: claude
  model: claude-opus-4-7
  reasoning_effort: medium
  access_mode: bypass-permissions
---

# Exemplo: Revisor Claude

Você é um revisor cuidadoso. Sua função é:
- Validar invariantes ACP/governança
- Priorizar segurança e tratamento de erros
- Garantir paridade cross-CLI

Este arquivo é descoberto automaticamente pelo Agent Registry quando colocado em
`~/.ai-harness/agents/example-claude-reviewer/AGENT.md` (global) ou
`.ai-harness/agents/example-claude-reviewer/AGENT.md` (workspace).
```

## Critérios de Sucesso

- `AGENTS.md` tem nova seção "Agent Registry" com todos os links válidos (relativos ao repo).
- `docs/agents/example/AGENT.md` (ou path equivalente) existe e parseia corretamente quando consumido pelo registry (sanity-check via teste manual).
- `.specs/adr/011-agent-registry-declarativo.md` referenciado em `AGENTS.md` e/ou índice de ADRs.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Verificação manual: `AGENTS.md` renderiza corretamente em GitHub e todos os links resolvem.
- [ ] Verificação manual: exemplo `AGENT.md` é parseado pelo registry sem erro (sanity-check com teste manual ou caso de teste extra adicionado).
- [ ] Lint Markdown opcional (se houver no projeto).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `AGENTS.md` (modificado)
- `CLAUDE.md` (modificado, se houver tabela de ADRs)
- `docs/agents/example/AGENT.md` (novo)
- `docs/research/compozy-adaptation-analysis.md` (opcionalmente modificado — seção "Próximos PRDs")
- `.specs/adr/011-agent-registry-declarativo.md` (referência; sem modificação)
