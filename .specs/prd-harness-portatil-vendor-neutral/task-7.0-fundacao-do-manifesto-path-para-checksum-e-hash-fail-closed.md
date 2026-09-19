# Tarefa 7.0: Fundacao do manifesto: path para checksum e hash fail-closed

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Tornar conflito uma proposicao decidivel, que e pre-requisito duro de todo o bloco de distribuicao.

<requirements>
- RF-24
</requirements>

## Subtarefas

- [ ] 7.1 Campo aditivo `omitempty` mapeando cada arquivo gerenciado ao seu hash (V-08: `checksums` e por-skill e write-only)
- [ ] 7.2 Install E upgrade passam a gravar o checksum por path; o upgrade passa a usar o rastreador de escrita que hoje e exclusivo do install
- [ ] 7.3 Existe leitura de PRODUCAO do campo novo — nao repetir o padrao write-only
- [ ] 7.4 Declarar as interfaces novas em `mockery.yml` no mesmo commit

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Manifesto legado e novo convivem sem regressao, pelo caminho de `HasFileTracking()`
- Manifesto legado sem o campo significa ausencia de rastreio e NUNCA divergencia — senao todo projeto instalado antes desta entrega classificaria tudo como conflito
- Nenhuma falha de leitura de hash produz resultado de sucesso
- Benchmark antes e depois sem regressao relevante

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
