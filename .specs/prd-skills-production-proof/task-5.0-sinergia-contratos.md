# Tarefa 5.0: Sinergia — review confronta aceite + tabela de severidade + bugfix path cascade

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar zonas cinzentas de handoff: `review` confronta cada critério de aceite da task ativa
sempre; tabela canônica de severidade `review`↔`bug-schema`; path do validador do `bugfix`
resolvido em cascata. Cobre RF-14, RF-15, RF-17.

<requirements>
- `review/SKILL.md`: ler a task ativa e confrontar cada critério de aceite SEMPRE (remover leitura condicional).
- Nova `agent-governance/references/severity-mapping.md` (`critical→critical`, `high→major`, `medium→minor`, `low→minor`).
- `review` referencia a tabela ao emitir bugs; `bugfix` ao consumir.
- `bugfix/SKILL.md` resolve o validador em cascata (já entregue conceitualmente na 1.0; garantir consistência aqui).
</requirements>

## Subtarefas

- [ ] 5.1 Atualizar `review/SKILL.md` para confronto incondicional de critérios de aceite.
- [ ] 5.2 Criar `severity-mapping.md` e referenciar em `review` e `bugfix`.
- [ ] 5.3 Garantir cascata de path no `bugfix` (consistente com 1.0).

## Detalhes de Implementação

Ver techspec "Tabela de severidade (RF-15)" e ADR-001. Confronto de aceite independe de o diff
tocar arquivos citados na task.

## Critérios de Sucesso

- `review` lê a task ativa e cita cada critério de aceite no veredito, mesmo sem o diff tocar arquivos citados.
- `severity-mapping.md` existe e é referenciada por `review` e `bugfix`.
- `bugfix` resolve o validador em repo só-`.agents/`.

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Verificação textual: `review` não tem mais leitura condicional de task para critérios de aceite.
- [ ] Mapeamento de severidade aplicado num caso `high`→`major` de exemplo.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.agents/skills/review/SKILL.md`, `.agents/skills/bugfix/SKILL.md`
- `.agents/skills/agent-governance/references/severity-mapping.md`
- `.agents/skills/agent-governance/references/bug-schema.json`
