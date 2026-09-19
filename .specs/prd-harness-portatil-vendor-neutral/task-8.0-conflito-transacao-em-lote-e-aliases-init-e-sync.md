# Tarefa 8.0: Conflito, transacao em lote e aliases init e sync

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fazer a aplicacao ser tudo-ou-nada, com conflito como categoria de primeira classe.

<requirements>
- RF-22
- RF-23
- RF-23.1
- RF-24
- RF-25
- RF-26
- RF-27
- RF-28
- RF-29
</requirements>

## Subtarefas

- [x] 8.1 Terceira categoria de conflito no rastreador de escrita, com precedencia EXPLICITA em codigo
- [x] 8.2 Relatorio estruturado com cinco categorias (criado, atualizado, preservado, mesclado, conflito), substituindo saida textual como fonte de decisao (RF-23.1)
- [x] 8.3 Transacao por staging, promocao e journal de reversao, com a ordem registrada ANTES de qualquer escrita destrutiva
- [x] 8.4 Eliminar `RemoveAll` seguido de `CopyDir` como operacao irreversivel, e remover o atalho que pula a escrita do manifesto
- [x] 8.5 Abort-on-conflict por default, com flag explicita de sobrescrita que nomeia cada arquivo
- [x] 8.6 Registrar `init` e `sync` como aliases, cobertos pelo teste de contrato de CLI e pelo schema de CLI
- [x] 8.7 A prova de RF-28 usa filesystem REAL em diretorio temporario, nunca o duble (V-32)

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Injecao de falha no meio do lote deixa a arvore BYTE-IDENTICA a inicial
- Variar qual arquivo falha entre primeiro, meio e ultimo — falhar so no ultimo e o caso que mais passa por acaso
- Nenhum arquivo temporario remanescente apos a falha
- Arquivo nao gerenciado nunca e tocado nem removido
- Conflito aborta o lote e a mensagem aponta a origem canonica
- `sync` e deterministico e nenhuma flag ou saida existente foi alterada

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

- `design-patterns-mandatory` — staging e rollback exigem escolha justificada entre padroes do catalogo (Command, Memento, Unit of Work), com pseudocodigo canonico e plano de testes que sustentem a garantia de atomicidade.

## Testes da Tarefa

- [x] Testes unitários
- [x] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Ver `techspec.md`, seção *Arquivos Relevantes e Dependentes*.
