# Tarefa 6.0: Wiring taskloop.Service.Execute para roteamento ACP por tool

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Em `internal/taskloop/taskloop.go`, `Service.Execute` ganha branch ACP que consulta `runtimeACPCatalog` (exposto pela Tarefa 5.0) quando `opts.Runtime == "acp"`. Resolve `Spec` correspondente ao `opts.Tool` e instancia `ACPRunner` reusando 100% do stack existente. Quando `opts.Runtime == "legacy"`, comportamento atual preservado.

Esta é a **tarefa crítica** do PRD (R-08): diff zero obrigatório em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/`. Revisão manual do diff é gate de merge.

<requirements>
- Service.Execute ganha branch ACP-routing baseado em opts.Runtime + opts.Tool.
- Quando opts.Runtime == "acp": resolve Spec via runtimeACPCatalog, instancia ACPRunner.
- Quando opts.Runtime == "legacy": comportamento atual inalterado (copilotInvoker para Copilot).
- DIFF ZERO em internal/runtime/persistence/, internal/runtime/watchdog.go, internal/runtime/client/.
- Suítes internal/runtime/, internal/taskloop/, cmd/ai_spec_harness/ 100% verdes.
</requirements>

## Subtarefas

- [ ] 6.1 Adicionar branch em `Service.Execute` (ou função adjacente) que detecta `opts.Runtime == "acp"`.
- [ ] 6.2 Resolver `Spec` consumindo `runtimeACPCatalog[opts.Tool]` (importado do package `cmd/ai_spec_harness/` ou exposto via pacote intermediário — decisão de design no momento).
- [ ] 6.3 Instanciar `airuntime.NewACPRunner(spec, opts...)` e invocar `Run(ctx, job)` reusando padrão do Claude.
- [ ] 6.4 Garantir que o fluxo legado (`opts.Runtime == "legacy"`) permanece **byte-idêntico** ao comportamento atual.
- [ ] 6.5 Adicionar caso T-16 em `taskloop_test.go` validando que `Runtime == "legacy"` continua usando `copilotInvoker`.
- [ ] 6.6 Rodar suítes completas: `go test ./internal/runtime/...`, `go test ./internal/taskloop/...`, `go test ./cmd/ai_spec_harness/...` (T-22, T-23, T-24).
- [ ] 6.7 **Revisão manual obrigatória do diff**: `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` deve retornar vazio.

## Detalhes de Implementação

Ver `techspec.md` §"Relacionamentos e Fluxo de Dados" e §"Design de Implementação" → bloco `internal/taskloop/taskloop.go`. Decisões D-02 (reuso total) e D-04 (catálogo no `cmd/`).

Risco R-08 (crítico): mudança no caminho de `Service.Execute` pode acidentalmente afetar lógica de persistência ou watchdog se feita sem disciplina. A regra de "diff zero" nesses três módulos é guarda-rail obrigatório.

Anti-padrão: NÃO modificar `internal/runtime/client/client.go` para suportar Copilot — o `acpClient` é agnóstico de IDE por design. Se precisar, é sinal de erro de design — pausar e revisar.

## Critérios de Sucesso

- `Service.Execute` roteia Copilot para `ACPRunner` quando `opts.Runtime == "acp"`.
- Fluxo legado (`opts.Runtime == "legacy"`) permanece byte-idêntico.
- T-16 verde.
- Suítes completas T-22 (`internal/runtime/...`), T-23 (`internal/taskloop/...`), T-24 (`cmd/ai_spec_harness/...`) verdes.
- **Diff zero** em três módulos críticos.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-16: `opts.Runtime == "legacy" && opts.Tool == "copilot"` → roteia para `copilotInvoker` (caminho atual).
- [ ] Novo caso (decorrência de RF-08): `opts.Runtime == "acp" && opts.Tool == "copilot"` → instancia `ACPRunner` com `specs.Copilot()`.
- [ ] T-22: `go test ./internal/runtime/...` → 100% verde.
- [ ] T-23: `go test ./internal/taskloop/...` → 100% verde.
- [ ] T-24: `go test ./cmd/ai_spec_harness/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] Branch ACP-routing implementado em `Service.Execute`.
- [ ] Catálogo consumido sem duplicação (consume `runtimeACPCatalog` da Tarefa 5.0).
- [ ] Fluxo legado byte-idêntico (T-16 verde).
- [ ] T-22/T-23/T-24 verdes.
- [ ] **Diff zero verificado**: `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` retorna 0 arquivos. Evidência registrada no PR/commit message.
- [ ] Sem mudança em `acpClient` (`internal/runtime/client/client.go`).
- [ ] `go vet ./...` → sem warnings.

## Arquivos Relevantes

- `internal/taskloop/taskloop.go` (modificar — branch ACP-routing)
- `internal/taskloop/taskloop_test.go` (estender — T-16 + novo caso ACP)
- `cmd/ai_spec_harness/task_loop.go` (consultar — catálogo da Tarefa 5.0)
- `internal/runtime/runner.go` (consumido — ACPRunner)
- `internal/runtime/persistence/` (**não modificar**)
- `internal/runtime/watchdog.go` (**não modificar**)
- `internal/runtime/client/` (**não modificar**)
- ADR-012 §"Decisão D-02" (reuso total) e §"Risco R-08"
