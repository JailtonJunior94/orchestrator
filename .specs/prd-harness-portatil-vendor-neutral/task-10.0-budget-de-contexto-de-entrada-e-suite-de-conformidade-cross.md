# Tarefa 10.0: Budget de contexto de entrada e suite de conformidade cross-provider

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Medir o contexto declarado por provedor e provar equivalencia de invariantes nos quatro.

<requirements>
- RF-35
- RF-36
- RF-37
- RF-38
- RF-39
- RF-40
- RF-41
- RF-41.1
- RF-42
</requirements>

## Subtarefas

- [ ] 10.1 Budget com numerador DECLARADO POR PROVEDOR, nunca derivado do prompt do harness (V-30)
- [ ] 10.2 Teste que prova que o numerador de OpenCode e Codex inclui os arquivos carregados nativamente
- [ ] 10.3 Teto com margem de 10% sobre o medido, conforme a convencao vigente; o budget por skill permanece intocado
- [ ] 10.4 Runner UNICO abstraindo script shell e harness Node, generalizando o unico teste que hoje cobre os quatro provedores
- [ ] 10.5 Asssercao TRIPLA: exit code, ausencia do marcador de cadeia quebrada, e presenca do veredito do validador canonico
- [ ] 10.6 Manifesto da suite replicando a tabela normativa do PRD, com teste que falha se divergir
- [ ] 10.7 Caso live confirmatorio no fluxo nightly, aprovando apenas com disparo comprovado e sem mutacao
- [ ] 10.8 V-41: o pacote de teste novo precisa cair num dos tres pacotes que o CI roda, ou o workflow e corrigido NESTA tarefa

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- A suite fica VERMELHA quando o gate e removido, validado por remocao deliberada e nao por inspecao
- Os quatro provedores retornam o mesmo exit code para a mesma entrada
- Se o numerador dos quatro provedores for identico, a medicao esta derivando do prompt e a tarefa falhou
- Toda falha e atribuida a core, adapter ou provider, ou a impossibilidade de atribuir e declarada
- 100% dos cenarios classificados como deterministicos rodam em CI

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
