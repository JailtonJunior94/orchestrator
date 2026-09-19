# Tarefa 4.0: Canonicalizacao de policies com par dedicado de espelhamento

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Estabelecer origem canonica unica das regras transversais, com o espelho verificado por gate dedicado.

<requirements>
- RF-09
- RF-10
- RF-11
- RF-16
</requirements>

## Subtarefas

- [ ] 4.1 Mover `governance.md` e `code-style.md` para a origem canonica BYTE A BYTE
- [ ] 4.2 Tornar `.claude/rules/` espelho gerado, preservando caminho e conteudo identicos
- [ ] 4.3 Criar o par dedicado de scripts de sync e de verificacao, com lista DECLARADA nos dois (nunca glob) e ausencia do canonico tratada como drift
- [ ] 4.4 NAO generalizar nem alterar os tres pares existentes de skills, hooks e scripts (D-13)
- [ ] 4.5 REGISTRO ANTI-GATE-ORFAO, indivisivel desta entrega: alvo no `Makefile`, step em `.github/workflows/test.yml`, e as DUAS listas hardcoded em `tests/integration/sync_gates_guard_test.go`

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- O gate falha ao editar o espelho E ao apagar o canonico
- Conteudo carregado pelo Claude Code e byte-identico ao anterior a migracao
- Os quatro registros anti-gate-orfao estao presentes e o gate roda em todo push e PR
- Os tres gates de sincronia existentes permanecem verdes, evidencia de que a decisao de nao generalizar foi respeitada

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
