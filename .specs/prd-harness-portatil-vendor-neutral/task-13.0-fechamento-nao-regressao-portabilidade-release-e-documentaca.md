# Tarefa 13.0: Fechamento: nao-regressao, portabilidade, release e documentacao

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Provar que nada regrediu contra o baseline empirico e publicar a entrega como minor.

<requirements>
- RF-48
- RF-49
- RF-50
- RF-51
- RF-52
- RF-53
- RF-55
- RF-56
- RF-57
- RF-63
</requirements>

## Subtarefas

- [ ] 13.1 Rodar os 15 alvos do baseline e confrontar com os numeros empiricos de V-40
- [ ] 13.2 Teste de alternancia entre as quatro CLIs sem reinstalar nem converter o projeto (RF-53)
- [ ] 13.3 Teste de que nenhum artefato instalado exige processo permanente (RF-52)
- [ ] 13.4 Teste de que nenhum commit ou push ocorre implicitamente em nenhum fluxo (RF-55)
- [ ] 13.5 Fixture de projeto instalado na versao anterior contra o binario novo, sem acao do usuario (RF-50)
- [ ] 13.6 Politica de release verificada por teste de configuracao, nao por convencao (RF-56)
- [ ] 13.7 Documentacao reconciliada e changelog MINOR declarando a regra nova nominalmente com nota de migracao
- [ ] 13.8 RF-63: revisar que nenhuma afirmacao declara como herdada capacidade que V-33, V-34 ou V-37 provaram inexistente

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Todos os gates do baseline verdes, com os numeros de V-40 preservados
- Zero mudanca de default nao declarada no changelog
- Projeto instalado na versao anterior funciona sem nenhuma acao do usuario
- Evidencias persistidas por tarefa conforme o padrao vigente (RF-57)

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
