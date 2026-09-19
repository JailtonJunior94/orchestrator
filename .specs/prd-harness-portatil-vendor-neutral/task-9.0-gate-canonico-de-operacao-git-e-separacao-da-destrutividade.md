# Tarefa 9.0: Gate canonico de operacao Git e separacao da destrutividade

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Converter em codigo executavel a proibicao de commit e push nao solicitados, que hoje existe apenas como prosa.

<requirements>
- RF-40.1
- RF-40.2
</requirements>

## Subtarefas

- [ ] 9.1 Criar o script canonico fail-closed com exit de bloqueio 2 e diagnostico proprio (V-23: o enforcement NAO existe hoje)
- [ ] 9.2 Delegar a partir do hook canonico compartilhado pelos quatro provedores, sem reimplementar logica por agente
- [ ] 9.3 Fazer o plugin OpenCode consultar o MESMO criterio; o conjunto de ferramentas mutantes ja inclui `bash` e nao muda
- [ ] 9.4 RF-40.2: separar o criterio de destrutividade do criterio de preload — hoje estao fundidos
- [ ] 9.5 Particionar o teste que hoje libera comando destrutivo sob preload confirmado
- [ ] 9.6 Escapes proprios auditados em log dedicado
- [ ] 9.7 Registrar o script nos DOIS gates de espelhamento e nas DUAS listas do guardiao

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Busca por `git commit` e `git push` nos scripts canonicos retorna o gate novo — hoje retorna vazio
- Com preload confirmado, operacao Git nao solicitada permanece BLOQUEADA
- Com preload confirmado, operacao destrutiva permanece BLOQUEADA
- Comando Git de leitura continua permitido — o gate nao pode virar bloqueio cego
- Os quatro registros de espelhamento e CI presentes

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
