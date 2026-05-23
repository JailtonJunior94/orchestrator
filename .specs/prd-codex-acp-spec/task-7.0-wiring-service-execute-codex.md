# Tarefa 7.0: Service.Execute propaga ReasoningEffort/AccessMode/AddDirs

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Estender `internal/taskloop/taskloop.go::Options` com 3 campos novos (`ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string`) e modificar `Service.Execute(opts)` para propagar esses valores para `Job` (`internal/runtime/runner.go`) quando `Runtime == "acp"`.

Para Claude/Copilot, os valores propagados são ignorados pelo `BootstrapArgs` no-op da tarefa 1.0. Para Codex, são consumidos por `codexBootstrapArgs` e injetados como `-c` flags no spawn (tarefa 4.0).

Esta tarefa completa a cadeia de propagação CLI → taskloop.Options → runtime.Job → spec.BootstrapArgs → exec.Command.

<requirements>
- Options ganha 3 campos novos (mesmos tipos de Job da tarefa 3.0).
- Service.Execute popula Job com esses 3 campos quando Runtime=="acp".
- Claude/Copilot recebem os valores em Job mas Spec.BootstrapArgs no-op ignora.
- T-26 valida regressão Claude (flags Codex passadas sem efeito).
- T-27 valida que Runtime=="legacy" não consome esses campos (codexInvoker path inalterado).
- Diff zero em internal/runtime/persistence/, watchdog.go, client/.
- Service.Execute mantém comportamento de erro existente (ErrLauncherUnavailable, etc.).
</requirements>

## Subtarefas

- [ ] 7.1 Adicionar campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string` ao struct `Options` em `internal/taskloop/taskloop.go`.
- [ ] 7.2 No `Service.Execute`, no ramo `Runtime == "acp"` onde `Job` é construído (após resolução de Spec via `runtimeACPCatalog`), popular `Job.ReasoningEffort = opts.ReasoningEffort`, `Job.AccessMode = opts.AccessMode`, `Job.AddDirs = opts.AddDirs`.
- [ ] 7.3 Verificar tratamento de zero-value: `opts.AccessMode == ""` deve resultar em `Job.AccessMode == AccessModeRestricted` (ou ser tratado em consumo). Decisão alinhada com tarefa 3.0/4.0.
- [ ] 7.4 Confirmar que `cmd/ai_spec_harness/task_loop.go` (tarefa 6.0) já propaga `reasoningEffort` e `accessMode` para `Options`.
- [ ] 7.5 Atualizar/adicionar testes em `internal/taskloop/taskloop_test.go`: T-26 (Claude com flags ignoradas), T-27 (legacy não consome).
- [ ] 7.6 Rodar `go test ./internal/taskloop/... ./cmd/ai_spec_harness/...` → 100% verde.
- [ ] 7.7 **Gate R-08**: `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` → vazio.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `taskloop/taskloop.go::Service.Execute` (esboço) e §"Sequenciamento de Desenvolvimento" → item 9. Decisão registrada em ADR-013 D-02 (propagação opcional via Options).

Anti-padrão: NÃO consumir `opts.ReasoningEffort`/`opts.AccessMode` no ramo `Runtime == "legacy"` — `codexInvoker` legado ignora esses campos (caminho stateless).

## Critérios de Sucesso

- `Options` carrega 3 campos novos.
- `Service.Execute` propaga corretamente para `Job` quando Runtime ACP.
- Claude/Copilot fluem inalterados (BootstrapArgs no-op).
- Codex flui com `-c` flags corretas no spawn (validado em conjunto com integração 9.0).
- Legacy path (`Runtime=="legacy"`) inalterado.
- Diff zero em módulos invariantes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-26 (Claude regressão com flags Codex): `Options{Tool:"claude", Runtime:"acp", ReasoningEffort:"high", AccessMode:AccessModeFull}` flui; `Spec.BootstrapArgs()` no-op retorna `nil`; spawn args **NÃO** contêm `-c` flags. Comportamento Claude idêntico.
- [ ] T-27 (legacy path): `Options{Tool:"codex", Runtime:"legacy"}` (com ou sem flags Codex) → roteia para `codexInvoker.Invoke()`; novos campos de Options ignorados.
- [ ] T-32 (regressão taskloop): suíte completa de `internal/taskloop/...` 100% verde.
- [ ] Gate R-08: `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` → vazio.
- [ ] `go vet ./...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `internal/taskloop/taskloop.go::Options` tem campos `ReasoningEffort string`, `AccessMode specs.AccessMode`, `AddDirs []string` com go-doc.
- [ ] `Service.Execute` popula `Job` com esses 3 campos no ramo `Runtime == "acp"`.
- [ ] Ramo `Runtime == "legacy"` permanece inalterado em consumo dos novos campos.
- [ ] `cmd/ai_spec_harness/task_loop.go` propaga `reasoningEffort/accessMode` para `Options` (verificar — pode já estar na tarefa 6.0).
- [ ] T-26 valida que Claude com flags Codex compila e roda sem efeito.
- [ ] T-27 valida que legacy path Codex permanece igual.
- [ ] **Gate R-08**: diff zero em `persistence/`, `watchdog.go`, `client/`.
- [ ] `go test ./internal/taskloop/... ./cmd/ai_spec_harness/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.

## Arquivos Relevantes

- `internal/taskloop/taskloop.go` (modificar: Options + Service.Execute)
- `internal/taskloop/taskloop_test.go` (adicionar T-26, T-27)
- `cmd/ai_spec_harness/task_loop.go` (verificar: propagação para Options já feita na tarefa 6.0)
- `internal/runtime/runner.go` (consumir Job estendido da tarefa 3.0)
- `internal/runtime/specs/spec.go` (consumir AccessMode da tarefa 1.0)
- ADR-013 §"Decisão" → D-02
- techspec.md §"Design de Implementação" → bloco `taskloop.go`
