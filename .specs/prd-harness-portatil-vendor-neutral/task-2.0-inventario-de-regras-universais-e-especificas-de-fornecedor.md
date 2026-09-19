# Tarefa 2.0: Inventario de regras universais e especificas de fornecedor

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Classificar cada regra do repositorio como universal ou especifica de fornecedor, fechando a lista de entrada da canonicalizacao.

<requirements>
- RF-01
- RF-14
- RF-54
</requirements>

## Subtarefas

- [ ] 2.1 Varrer `AGENTS.md`, `CLAUDE.md`, `CODEX.md`, `COPILOT.md`, `.claude/`, `.codex/`, `.opencode/` e `.github/`
- [ ] 2.2 Classificar cada regra como `universal`, `claude-specific`, `codex-specific`, `copilot-specific` ou `opencode-specific`
- [ ] 2.3 Registrar a decisao de NAO criar diretorio de workflows (RF-14) e responder o item correspondente do Definition of Done por ausencia de objeto (RF-54), com gatilho de reabertura declarado

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Inventario commitado e revisavel em diff, sem nenhum item classificado como indefinido
- O subconjunto `universal` esta fechado e e a lista de entrada da tarefa 4.0
- Nenhum arquivo foi movido nesta tarefa

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Ver `techspec.md`, seção *Arquivos Relevantes e Dependentes*.
