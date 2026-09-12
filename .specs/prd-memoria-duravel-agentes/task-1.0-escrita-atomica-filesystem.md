# Tarefa 1.0: Escrita atômica na abstração de filesystem

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Promover a escrita atômica que hoje existe apenas em `internal/sdd/state.go:299` até `:321` para a abstração `internal/fs.FileSystem`, tornando-a reutilizável e testável. Hoje `internal/fs/fs.go:162` faz escrita direta: se o processo morrer no meio, o arquivo fica truncado. Nenhuma página de memória pode ser gravada por esse caminho.

Esta é a base de 5.0 e não depende de nenhuma outra tarefa.

<requirements>
- RF-16 (parte atômica): escrita que nunca deixa arquivo parcial visível.
- Mudança estritamente aditiva: nenhuma assinatura existente pode mudar.
- Implementação real e implementação falsa, para que 5.0 seja testável sem tocar disco.
- Erro de `Sync` e de `Close` verificado — `errcheck` só tolera a allowlist de `.golangci.yml:14`, e `Sync` não está nela.
</requirements>

## Subtarefas

- [x] 1.1 Adicionar o método de escrita atômica à interface `FileSystem` em `internal/fs/fs.go`.
- [x] 1.2 Implementar em `OSFileSystem` portando o padrão de `internal/sdd/state.go:299`: temporário no mesmo diretório, `Sync`, `Close`, `Rename`, com `defer` de limpeza do temporário.
- [x] 1.3 Implementar em `FakeFileSystem` (`internal/fs/fake.go`) com semântica equivalente.
- [x] 1.4 Rodar `make mocks` e confirmar `make check-mocks` verde.
- [x] 1.5 Confirmar que os dublês que embutem `*fs.FakeFileSystem` continuam compilando (ex.: `internal/runtime/persistence/jsonl_test.go:18`).

## Detalhes de Implementação

Ver techspec.md, seção "Interfaces Chave" (bloco de escrita atômica) e "Ordem de Build" (fatia T1). Ver MD-003 (`adr-003-escrita-atomica-lock-camada-lease.md`), seção Decisão, primeiro parágrafo.

## Critérios de Sucesso

- Teste prova que arquivo parcial nunca fica visível: interrupção entre escrita e rename não deixa conteúdo incompleto no caminho final.
- Nenhuma assinatura existente de `FileSystem` foi alterada; o compilador prova completude nas duas implementações.
- `make check-mocks` verde após regeneração.
- `make vet` e `make lint` mantêm a linha de base de zero problemas.
- Nenhum erro ignorado fora da allowlist do `.golangci.yml`.

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
- `internal/fs/fs.go` — interface e implementação real
- `internal/fs/fake.go` — implementação falsa
- `internal/fs/mocks/file_system.go` — regenerado
- `internal/sdd/state.go` — padrão de origem, apenas leitura
- `mockery.yml` — já declara `FileSystem`
- `.golangci.yml` — allowlist de `errcheck`
