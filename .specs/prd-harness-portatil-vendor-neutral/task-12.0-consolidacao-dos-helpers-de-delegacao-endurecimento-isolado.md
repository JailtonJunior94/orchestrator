# Tarefa 12.0: Consolidacao dos helpers de delegacao — endurecimento isolado

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Promover para producao o endurecimento que hoje so existe no teste, na direcao que fortalece o gate.

<requirements>
- RF-62
</requirements>

## Subtarefas

- [x] 12.1 Promover a logica do teste para producao: remocao de comentarios, resolucao de variaveis e exigencia de POSICAO DE EXECUCAO (V-36)
- [x] 12.2 Fazer o teste de matriz de paridade virar consumidor da producao, nunca o contrario
- [x] 12.3 Tratar ou registrar com decisao explicita cada violacao hoje mascarada que o endurecimento revelar

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Mencao textual a validador deixa de ser aceita como prova de delegacao
- Toda violacao revelada foi corrigida ou registrada com decisao explicita antes do merge
- ESPERADO: o gate fica mais vermelho ANTES de ficar verde — se consolidar e nada falhar, a promocao provavelmente nao aconteceu

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
- [x] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Ver `techspec.md`, seção *Arquivos Relevantes e Dependentes*.
