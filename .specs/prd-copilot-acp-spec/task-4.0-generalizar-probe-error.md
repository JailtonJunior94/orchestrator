# Tarefa 4.0: Generalizar probe error template + tabela adrByID

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Generalizar `internal/runtime/probe/probe.go:69-82` para deixar de assumir `specs.ClaudeNpmPackage` no template de erro `errMsgTemplate`. Template torna-se função parametrizada pelo `Spec` recebido. Referência ao ADR no remédio passa a ser metadata local ao package via tabela `adrByID map[string]string`.

Cache de probe (`sync.Map` keyed por `spec.ID`) permanece inalterado — apenas o caminho de erro muda.

<requirements>
- errMsgTemplate substituído por função parametrizada por spec.Command/NPMPackage/NPMVersion.
- Tabela adrByID local ao package probe mapeia spec.ID → path do ADR (`claude` → ADR-009, `copilot` → ADR-012).
- Cache permanece keyed por spec.ID; comportamento inalterado.
- Diff zero em internal/runtime/persistence/ e internal/runtime/watchdog.go.
</requirements>

## Subtarefas

- [ ] 4.1 Criar função `formatLauncherUnavailable(spec specs.Spec, adrPath string) string` em `internal/runtime/probe/probe.go` substituindo o uso de `errMsgTemplate`.
- [ ] 4.2 Criar tabela `adrByID = map[string]string{"claude": ".specs/adr/009-acp-protocol-adoption.md", "copilot": ".specs/adr/012-copilot-cli-acp-native.md"}`.
- [ ] 4.3 Lookup do ADR em `resolve()`: usar `adrByID[spec.ID]` com fallback para path raiz `.specs/adr/` quando ID desconhecido.
- [ ] 4.4 Mensagem de erro para Spec Copilot deve conter: nome do binário (`copilot`), pacote npm com versão (`@github/copilot@<pin>`), sugestão `--runtime=legacy`, path do ADR-012.
- [ ] 4.5 Estender `internal/runtime/probe/probe_test.go` com casos T-06 (Spec Copilot, nada disponível), T-07 (Spec Copilot, binary present), T-08 (Spec Copilot, fallback npx present), T-20 (lookup adrByID válido), T-21 (lookup adrByID inválido com fallback).
- [ ] 4.6 Confirmar diff zero em `internal/runtime/persistence/` e `internal/runtime/watchdog.go`.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/runtime/probe/probe.go — generalização do erro`. Decisão D-09 (`adrByID` local ao package).

Anti-padrão: NÃO acoplar `specs/` à estrutura de docs ADRs; mapping vive em `probe/`.

## Critérios de Sucesso

- Erro de Spec Copilot sem binary nem npx: contém `"copilot não encontrado"`, `"@github/copilot@..."`, `"--runtime=legacy"`, `"012-copilot-cli-acp-native.md"`.
- Erro de Spec Claude permanece idêntico ao atual (regressão).
- Cache continua keyed por `spec.ID` (validar via reuse).
- Testes T-06/T-07/T-08/T-20/T-21 verdes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-06: Spec Copilot, binary ausente, npx ausente → erro com mensagem completa (binário, npm, fallback, ADR-012).
- [ ] T-07: Spec Copilot, binary presente → retorna `BinaryLauncher("copilot", "--acp")`.
- [ ] T-08: Spec Copilot, binary ausente, npx presente → retorna `NpxLauncher("@github/copilot", CopilotNpmVersion)`.
- [ ] T-20: `adrByID["claude"]` aponta para ADR-009; `adrByID["copilot"]` para ADR-012.
- [ ] T-21: ID desconhecido → fallback gracioso (path raiz ou string vazia documentada).
- [ ] Regressão Claude: T-06 equivalente para Claude continua verde com mensagem ADR-009.
- [ ] `go test ./internal/runtime/probe/...` → 100% verde.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `errMsgTemplate` substituído por função `formatLauncherUnavailable(spec, adrPath)` ou equivalente.
- [ ] `adrByID` declarado como `var` no package `probe`; mapping inclui `claude` e `copilot`.
- [ ] `resolve()` usa `adrByID[spec.ID]` com fallback documentado.
- [ ] Mensagem de erro Copilot contém todos os componentes (binário, npm@pin, fallback legacy, path ADR-012).
- [ ] Mensagem de erro Claude continua idêntica em estrutura ao comportamento atual.
- [ ] Cache (`sync.Map`) inalterado em comportamento.
- [ ] T-06..T-08 + T-20/T-21 verdes em `probe_test.go`.
- [ ] `go test ./internal/runtime/probe/...` 100% verde.
- [ ] **Diff zero** em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`.

## Arquivos Relevantes

- `internal/runtime/probe/probe.go:69-82` (modificar)
- `internal/runtime/probe/probe_test.go` (estender)
- `internal/runtime/specs/copilot.go` (referência — Tarefa 2.0)
- ADR-012 §"Decisão D-09" (tabela `adrByID`)
- ADR-009 `.specs/adr/009-acp-protocol-adoption.md` (referenciado para Claude)
