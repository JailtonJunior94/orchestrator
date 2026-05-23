# Tarefa 3.0: Paridade de normalização de tool-calls (4 CLIs)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar a paridade de normalização: dar a Gemini uma tabela de alias explícita (não só `inherit_common`) e definir `input_mappings` para Copilot e Gemini. Propagar `DriverID` (em vez de `string` crua) no caminho de normalização do runner.

<requirements>
- `aliases.gemini` explícito em `normalization-rules.yaml` (RP-04), materializando os aliases hoje herdados de `common_aliases`.
- `input_mappings.copilot` e `input_mappings.gemini` (RP-01/RIN-02); quando comprovadamente no-op, registrar entrada com comentário `# no-op verificado`.
- Propagar `DriverID` (Tarefa 1.0) em `BuildNormalizedToolCall`/`normalizeEventInline` (`runner.go`), resolvendo na fronteira; driver inválido falha cedo (`ErrUnknownDriver`).
- `RawName`/`RawInput` permanecem byte-identical; `--no-normalize` recupera comportamento pré-normalização.
</requirements>

## Subtarefas

- [ ] 3.1 Adicionar `aliases.gemini` e `input_mappings.copilot/gemini` em `events/normalization-rules.yaml`.
- [ ] 3.2 Propagar `DriverID` em `normalizeEventInline`/`BuildNormalizedToolCall`.
- [ ] 3.3 Golden tests por driver para `input_mappings` (campo canônico idêntico).
- [ ] 3.4 Teste de herança: tabela explícita Gemini vence `inherit_common`.

## Detalhes de Implementação

Ver techspec.md §"Modelos de Dados" (snippet YAML) e §"Design de Implementação". ADR: [020](adr-020-driverid-vo-normalizacao-paridade.md). O loader (`loadRules`/`resolveInherit`) já não sobrescreve entrada explícita — validar esse caminho.

## Critérios de Sucesso

- Mesma tool-call em Copilot/Gemini produz os mesmos campos canônicos de input que Claude/Codex.
- Gemini usa a tabela explícita; remover `inherit_common` não muda o resultado para Gemini.
- `RawInput` nunca mutado; `--no-normalize` byte-identical ao pré-normalização.
- `make test` verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/events/normalization-rules.yaml`
- `internal/runtime/events/normalize.go` + `normalize_test.go`
- `internal/runtime/runner.go` (`normalizeEventInline`)
