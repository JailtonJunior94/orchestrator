# Tarefa 1.0: Estender Spec value object com metadata SDK/NPM

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Estender o value object `specs.Spec` (`internal/runtime/specs/spec.go`) com três campos privados (`sdkVersion`, `npmVersion`, `npmPackage`) e três acessores públicos (`SDKVersion()`, `NPMVersion()`, `NPMPackage()`). Atualizar a assinatura do construtor `newSpec` para receber esses três valores ao final. Atualizar `specs.Claude()` em `internal/runtime/specs/claude.go` para passar as constantes Claude existentes (`ClaudeSDKVersion`, `ClaudeNpmVersion`, `ClaudeNpmPackage`).

Esta tarefa é **gate de regressão** (R-01): nenhuma outra mudança em `internal/runtime/specs/` deve começar antes que `claude_test.go` esteja 100% verde.

<requirements>
- Spec ganha 3 campos privados (lowercase) e 3 acessores read-only (uppercase).
- newSpec é a única forma de construir Spec (R-DDD-001 preservado).
- claude.go atualizado; comportamento de Claude inalterado.
- claude_test.go 100% verde sem alteração de teste.
- Diff zero em internal/runtime/persistence/ e internal/runtime/watchdog.go.
</requirements>

## Subtarefas

- [ ] 1.1 Adicionar campos privados `sdkVersion`, `npmVersion`, `npmPackage` em `specs.Spec`.
- [ ] 1.2 Adicionar acessores públicos `SDKVersion()`, `NPMVersion()`, `NPMPackage()` (read-only, sem ponteiro).
- [ ] 1.3 Estender `newSpec(...)` recebendo os três parâmetros adicionais ao final (`sdkVersion, npmVersion, npmPackage string`).
- [ ] 1.4 Atualizar `Claude()` em `claude.go` para passar `ClaudeSDKVersion`, `ClaudeNpmVersion`, `ClaudeNpmPackage`.
- [ ] 1.5 Rodar `go test ./internal/runtime/specs/...` e confirmar 100% verde.
- [ ] 1.6 Confirmar diff zero em `internal/runtime/persistence/` e `internal/runtime/watchdog.go` via `git diff --stat`.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/runtime/specs/spec.go — assinatura estendida` e §"Sequenciamento de Desenvolvimento" → item 1. Decisão registrada em ADR-012 D-03 ("Spec ganha metadata privada + acessores públicos").

Anti-padrão (R-DDD-001): NÃO instanciar `Spec{...}` por literal fora do package. Construtor `newSpec` permanece o único caminho.

## Critérios de Sucesso

- `internal/runtime/specs/` package compila sem erros.
- Acessores retornam exatamente os valores passados em `newSpec`.
- Suíte `internal/runtime/specs/claude_test.go` permanece 100% verde sem alteração de teste.
- Nenhuma mudança em `internal/runtime/persistence/` ou `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-05 (regressão Claude): `go test ./internal/runtime/specs/ -run TestClaude` verde.
- [ ] T-19 (acessores): novo teste em `spec_test.go` (ou caso de teste integrado em `claude_test.go`) verifica que `SDKVersion()`/`NPMVersion()`/`NPMPackage()` retornam os valores passados em `newSpec`.
- [ ] `go vet ./internal/runtime/specs/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] Campos `sdkVersion`/`npmVersion`/`npmPackage` adicionados como **privados** (lowercase) em `Spec`.
- [ ] Acessores `SDKVersion()`/`NPMVersion()`/`NPMPackage()` retornam string (read-only).
- [ ] `newSpec` signature estendida com os três parâmetros ao final; assinatura antiga não permanece como helper.
- [ ] `Claude()` em `claude.go` passa `ClaudeSDKVersion`, `ClaudeNpmVersion`, `ClaudeNpmPackage` na ordem.
- [ ] `go test ./internal/runtime/specs/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings em `internal/runtime/specs/`.
- [ ] `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go` → vazio (0 arquivos modificados).

## Arquivos Relevantes

- `internal/runtime/specs/spec.go` (modificar)
- `internal/runtime/specs/claude.go` (modificar)
- `internal/runtime/specs/claude_test.go` (validar regressão)
- `internal/runtime/specs/spec_test.go` (criar se não existir, ou adicionar caso a `claude_test.go`)
- ADR-012 §"Decisão D-03" — racional do design
- Referência cruzada: `internal/core/agent/registry_specs.go` do compozy (formato de Spec referência)
