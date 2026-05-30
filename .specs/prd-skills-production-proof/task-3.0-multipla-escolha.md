# Tarefa 3.0: Protocolo de múltipla escolha — referência + integração nas skills

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Introduzir um protocolo canônico de múltipla escolha (2–5 opções, "(Recomendado)", uma pergunta
por turno) e integrá-lo às skills de planejamento e à revisão de borda. Cobre RF-06..RF-08.

<requirements>
- Nova referência `agent-governance/references/multiple-choice-protocol.md`.
- `create-prd` e `create-technical-specification` aplicam o protocolo em ambiguidade material.
- `create-tasks` (fatiamento) e `review` (severidade de borda) aplicam o protocolo.
- Gatilho explícito: ambiguidade material — não usar em decisões triviais.
</requirements>

## Subtarefas

- [ ] 3.1 Escrever `multiple-choice-protocol.md` com TL;DR e gatilho de carregamento.
- [ ] 3.2 Integrar referência na etapa de esclarecimento de `create-prd` e `create-technical-specification`.
- [ ] 3.3 Integrar em `create-tasks` (decisão de fatiamento) e `review` (severidade de borda).

## Detalhes de Implementação

Ver techspec "Protocolo de múltipla escolha (RF-06..RF-08)". Formato: opções numeradas, primeira
marcada "(Recomendado)", uma pergunta por turno.

## Critérios de Sucesso

- A referência existe e segue o padrão de TL;DR/keywords/"Load complete when" das demais references.
- As 4 skills citam a referência no ponto de decisão correto.
- Nenhuma skill aplica múltipla escolha fora de ambiguidade material (sem ruído).

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Verificação textual: grep confirma referência citada nas 4 skills.
- [ ] `make check-skills-sync` verde após sync dos mirrors.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.agents/skills/agent-governance/references/multiple-choice-protocol.md`
- `.agents/skills/create-prd/SKILL.md`, `.agents/skills/create-technical-specification/SKILL.md`
- `.agents/skills/create-tasks/SKILL.md`, `.agents/skills/review/SKILL.md`
