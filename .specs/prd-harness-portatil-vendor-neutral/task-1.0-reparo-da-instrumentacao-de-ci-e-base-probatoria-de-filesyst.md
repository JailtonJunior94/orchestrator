# Tarefa 1.0: Reparo da instrumentacao de CI e base probatoria de filesystem

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fazer os gates que protegem toda a entrega efetivamente executarem, e dar ao filesystem um substrato de teste que pode falhar.

<requirements>
- RF-20.1
- RF-58
- RF-59
- RF-60
- RF-63
</requirements>

## Subtarefas

- [ ] 1.1 Incluir `./internal/parity/...` com `-tags=integration` no job `integration` de `.github/workflows/test.yml` (V-24: hoje nao roda em job algum)
- [ ] 1.2 Corrigir o filtro `-run` do passo de snapshots, que hoje filtra um nome de teste inexistente e passa com zero testes (V-33)
- [ ] 1.3 Instituir mutation test obrigatorio para todo gate novo, no molde de `cmd/ai_spec_harness/catalog_sync_test.go`
- [ ] 1.4 Criar alvo `make` e step de CI que recomputem os hashes de `skills-lock.json` (V-34: o mecanismo existe e ninguem o executa)
- [ ] 1.5 Resolver RF-58: decorador de falha local ao teste OU prova de transacao em filesystem real; `internal/fs/fake.go` nao e alterado por default (V-32)
- [ ] 1.6 Propagar os erros de hash hoje descartados e desambiguar `DirHash` para nao-diretorio (V-38)

## Detalhes de Implementação

Ver `techspec.md`, seções *Design de Implementação*, *Abordagem de Testes* e *Sequenciamento de
Desenvolvimento*. Os fatos verificados que sustentam esta tarefa estão em `prd.md`, seção
*Fatos Verificados* (V-01..V-41). Não duplicar conteúdo aqui.

## Critérios de Sucesso

- Os nomes dos testes de `internal/parity/e2e_parity_test.go` aparecem NOMINALMENTE no log do job de CI
- Cada gate reparado falha sob injecao deliberada de divergencia, provado por execucao e nao por inspecao visual
- Editar um `SKILL.md` sem regerar `skills-lock.json` derruba o CI
- Teste que injeta falha no meio de uma sequencia de escrita observa erro propagado
- Nenhum caminho produz sucesso a partir de duas leituras de hash falhas

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
