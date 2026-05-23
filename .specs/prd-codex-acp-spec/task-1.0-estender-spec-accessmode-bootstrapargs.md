# Tarefa 1.0: Estender Spec com AccessMode + BootstrapArgs default no-op

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Estender a interface `specs.Spec` (`internal/runtime/specs/spec.go`) de forma **retrocompatível** para comportar `BootstrapArgs` dinâmico que Codex requer. Três adições estruturais:

1. Tipo `AccessMode string` com consts `AccessModeRestricted = "restricted"`, `AccessModeFull = "full"`.
2. Tipo de função `BootstrapArgsFunc func(model, reasoning string, addDirs []string, mode AccessMode) []string` + campo privado `bootstrapArgs BootstrapArgsFunc` em `Spec` + método público `Spec.BootstrapArgs(...)` que delega para a função ou retorna `nil` se `bootstrapArgs == nil`.
3. Variant constructor `newSpecWithBootstrap(...)` que aceita `BootstrapArgsFunc` ao final; `newSpec(...)` original permanece inalterado em assinatura.

Atualizar `Claude()` e `Copilot()` para usar `newSpec(...)` puro (sem `BootstrapArgsFunc`) — comportamento inalterado, `bootstrapArgs == nil` → `BootstrapArgs(...)` retorna `nil`.

Esta tarefa é **gate de regressão crítico (R-01)**: nenhuma outra mudança em `internal/runtime/specs/` pode começar antes que `claude_test.go`, `copilot_test.go` e `spec_test.go` estejam 100% verdes sem alteração.

<requirements>
- Tipo `AccessMode` exportado com 2 consts (`AccessModeRestricted`, `AccessModeFull`).
- Tipo `BootstrapArgsFunc` exportado para uso por specs externos.
- Campo `bootstrapArgs` privado (lowercase) em `Spec`; método `BootstrapArgs(...)` público.
- Method `BootstrapArgs(...)` retorna `nil` quando `bootstrapArgs == nil` (no-op default).
- `newSpec(...)` original permanece com mesma assinatura; chamadores Claude/Copilot inalterados.
- `newSpecWithBootstrap(...)` é variant constructor para specs que injetam bootstrap.
- R-DDD-001 preservado: Spec continua value object imutável; bootstrapArgs é campo privado set apenas via constructors.
- Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/`.
</requirements>

## Subtarefas

- [ ] 1.1 Adicionar tipo `AccessMode string` + consts `AccessModeRestricted`, `AccessModeFull` em `internal/runtime/specs/spec.go`.
- [ ] 1.2 Adicionar tipo `BootstrapArgsFunc func(model, reasoning string, addDirs []string, mode AccessMode) []string`.
- [ ] 1.3 Adicionar campo privado `bootstrapArgs BootstrapArgsFunc` em `Spec` struct.
- [ ] 1.4 Adicionar método público `Spec.BootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string` com guard `if s.bootstrapArgs == nil { return nil }`.
- [ ] 1.5 Criar variant constructor `newSpecWithBootstrap(...)` que delega para `newSpec(...)` e atribui `bootstrapArgs`.
- [ ] 1.6 Verificar que `Claude()` em `claude.go` continua usando `newSpec(...)` original (sem mudança).
- [ ] 1.7 Verificar que `Copilot()` em `copilot.go` continua usando `newSpec(...)` original (sem mudança).
- [ ] 1.8 Adicionar/estender `spec_test.go` com testes T-10/T-11 (no-op de Claude/Copilot retorna `nil`) e T-19 (acessor).
- [ ] 1.9 Rodar `go test ./internal/runtime/specs/...` e confirmar 100% verde.
- [ ] 1.10 Confirmar diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/` via `git diff --stat`.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/runtime/specs/spec.go — extensão de Spec` e §"Sequenciamento de Desenvolvimento" → item 1. Decisão registrada em ADR-013 D-02 ("Extensão da interface `Spec` com `BootstrapArgs`") e D-03 (variant constructor `newSpecWithBootstrap`).

Anti-padrão (R-DDD-001): NÃO instanciar `Spec{...}` por literal fora do package. Construtores (`newSpec`/`newSpecWithBootstrap`) são o único caminho.

## Critérios de Sucesso

- `internal/runtime/specs/` package compila sem erros após as adições.
- `Spec.BootstrapArgs(...)` retorna `nil` para Specs criadas via `newSpec(...)` (Claude/Copilot).
- Suítes `claude_test.go` e `copilot_test.go` permanecem 100% verdes **sem alteração de teste**.
- `spec_test.go` ganha casos T-10, T-11 (no-op Claude/Copilot) e cobertura do acessor.
- Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-05 (regressão Claude): `go test ./internal/runtime/specs/ -run TestClaude` 100% verde.
- [ ] T-10 (no-op Claude): novo caso em `spec_test.go` valida `specs.Claude().BootstrapArgs("any", "any", nil, AccessModeFull) == nil`.
- [ ] T-11 (no-op Copilot): novo caso valida `specs.Copilot().BootstrapArgs("any", "any", nil, AccessModeFull) == nil`.
- [ ] T-19 (acessores/regressão): `go test ./internal/runtime/specs/...` cobre todos os testes existentes + novos sem regressão.
- [ ] `go vet ./internal/runtime/specs/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] Tipo `AccessMode string` exportado em `spec.go` com consts `AccessModeRestricted = "restricted"` e `AccessModeFull = "full"`.
- [ ] Tipo `BootstrapArgsFunc` exportado com assinatura `func(model, reasoning string, addDirs []string, mode AccessMode) []string`.
- [ ] Campo `bootstrapArgs BootstrapArgsFunc` **privado** (lowercase) em `Spec`.
- [ ] Método público `Spec.BootstrapArgs(...)` com guard que retorna `nil` quando `bootstrapArgs == nil`.
- [ ] Variant constructor `newSpecWithBootstrap(...)` cria Spec com bootstrapArgs preenchido.
- [ ] `Claude()` e `Copilot()` permanecem usando `newSpec(...)` original sem mudança de comportamento.
- [ ] `claude_test.go` e `copilot_test.go` passam 100% sem alteração.
- [ ] `spec_test.go` contém casos T-10 e T-11 (no-op explicitamente validado para Claude/Copilot).
- [ ] `go test ./internal/runtime/specs/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings em `internal/runtime/specs/`.
- [ ] `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` → vazio (0 arquivos modificados).

## Arquivos Relevantes

- `internal/runtime/specs/spec.go` (modificar: tipos, campo, método, variant constructor)
- `internal/runtime/specs/spec_test.go` (modificar: adicionar T-10, T-11)
- `internal/runtime/specs/claude.go` (verificar: sem mudança)
- `internal/runtime/specs/claude_test.go` (validar regressão)
- `internal/runtime/specs/copilot.go` (verificar: sem mudança)
- `internal/runtime/specs/copilot_test.go` (validar regressão)
- ADR-013 §"Decisão" → D-02, D-03 (racional do design)
- techspec.md §"Design de Implementação" → bloco `spec.go — extensão de Spec`
