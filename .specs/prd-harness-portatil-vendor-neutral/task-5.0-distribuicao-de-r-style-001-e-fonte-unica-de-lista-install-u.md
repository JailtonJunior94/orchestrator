# Tarefa 5.0: Distribuicao de R-STYLE-001 e fonte unica de lista install/uninstall

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fazer o consumidor receber o conjunto completo de regras universais, fechando o defeito V-01.

<requirements>
- RF-12
- RF-13
- RF-61
</requirements>

## Subtarefas

- [ ] 5.1 Embarcar `code-style.md` nos assets distribuidos
- [ ] 5.2 Tocar os QUATRO pontos hardcoded NO MESMO LOTE: `CopyFile` do install, `syncFileIfPresent` do upgrade, lista fixa do uninstall e a invariante CL05 da paridade
- [ ] 5.3 RF-61: substituir as duas listas fixas distantes por fonte unica compartilhada entre install e uninstall (V-35)
- [ ] 5.4 Atualizar os 26 call sites de `.claude/rules/`, comecando pelos Go que quebram teste
- [ ] 5.5 RF-13: escrever o teste que cobre o comportamento governado ANTES de remover qualquer duplicata

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Instalacao limpa entrega `governance.md` E `code-style.md`, verificado em install e em upgrade
- Round-trip instala/desinstala deixa a arvore limpa, sem arquivo orfao e sem impedir o prune do diretorio
- CL05 cobre as duas regras
- Nenhum call site aponta para `.claude/rules/` como origem editavel

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
