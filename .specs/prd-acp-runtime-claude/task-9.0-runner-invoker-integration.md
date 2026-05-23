# Tarefa 9.0: Application Runner + Invoker Adapter + Integration Tests

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o *application service* `ACPRunner` em `internal/runtime/runner.go` que orquestra todos os componentes entregues nas tasks 1.0–8.0; o adapter `acpInvoker` em `internal/taskloop/acpinvoker.go` que faz a ponte entre o `AgentInvoker` existente e o runner; e a suíte de testes de integração em `internal/runtime/acp_integration_test.go` usando `acpfake` para validar o fluxo ponta-a-ponta no processo Go (sem build tag, roda no `make test` padrão).

<requirements>
- `ACPRunner.Run(ctx, job) (Summary, error)` orquestra: probe → client.Open → consume events em fan-out (jsonl + render + counters + watchdog.Touch) → close → enrich report + write tool_calls.md → retorna Summary.
- Cancelamento via `context.CancelCause`; `Run` extrai razão via `errors.As` e popula `summary.CancelReason`.
- `acpInvoker` mantém a assinatura de `AgentInvoker.Invoke(ctx, prompt, workDir, model) (stdout, stderr, exitCode, error)`; agrega stdout do renderer humano consolidado em um `bytes.Buffer` interno.
- Integration tests rodam in-process via `acpfake`; sem build tag; sem rede.
- Cenários cobertos: happy path, activity timeout, permission denied (RF-16), unknown drift, launcher fallback (binary→npx).
- `RF-12` preservado: tests existentes do legacy continuam passando.
</requirements>

## Subtarefas

### Runner

- [ ] 9.1 Criar `internal/runtime/runner.go` com struct `ACPRunner` injetando `Prober`, `ClientFactory`, `Persistence`, `Renderer`, `Clock` e a `specs.Spec` ativa.
- [ ] 9.2 Implementar `NewACPRunner(opts ...Option) *ACPRunner` com Functional Options (`WithClock`, `WithProber`, `WithClientFactory`, `WithPersistence`, `WithRenderer`).
- [ ] 9.3 Implementar `(r *ACPRunner) Run(ctx context.Context, j Job) (Summary, error)`:
  - cria `ctx, cancelCause = context.WithCancelCause(ctx)`
  - `launcher, err := r.probe.EnsureAvailable(ctx, r.spec)`; se falha, retorna Summary{} com erro
  - cria `JSONLWriter` apontando para `j.EvidenceDir/events.jsonl`
  - emite `events.NewRuntimeInit(launcher, sdkVer, npmVer)` e persiste
  - `client := r.factory.New(j.WorkDir)`; `defer client.Close()`
  - `client.Open(ctx, launcher, j.Prompt)`
  - watchdog: `wd := NewActivityWatchdog(j.ActivityTimeout, cancelCause, r.clock)`; `wd.Start(ctx)`; `defer wd.Stop()`
  - loop: `for evt := range client.Updates() { wd.Touch(); counters.Record(evt); persist.AppendEvent(evt); if !j.Quiet { renderer.Render(evt) }; if evt.Kind() == KindUnknown { unknownKinds = append(...) }; if isPermissionRequest(evt) { cancelCause(ErrPermissionDenied); break } }`
  - pós-loop: cause = `context.Cause(ctx)`; mapear para `CancelReason`
  - `persist.WriteToolCalls(counters.ToolCalls())`
  - `persist.EnrichReport(j.EvidenceDir/execution_report.md, summary)`
  - imprimir warning agregado em stderr se `len(unknownKinds) > 0` (RF-05)
  - retornar `Summary{...}, err` mapeado por sentinel
- [ ] 9.4 Definir `Job` e `Summary` em `internal/runtime/types.go` (se ainda não em 7.0).

### Invoker

- [ ] 9.5 Criar `internal/taskloop/acpinvoker.go` com struct `acpInvoker { runner *runtime.ACPRunner; humanBuffer *bytes.Buffer; quiet bool; activityTimeout time.Duration }`.
- [ ] 9.6 Implementar `(c *acpInvoker) BinaryName() string` retornando `"claude-agent-acp"` (ou o resolvido pelo probe).
- [ ] 9.7 Implementar `(c *acpInvoker) Invoke(ctx, prompt, workDir, model string) (string, string, int, error)`: monta `Job{Prompt, WorkDir, EvidenceDir: workDir + "/evidence/<task>", ActivityTimeout: c.activityTimeout, Quiet: c.quiet}`; chama `runner.Run(ctx, job)`; mapeia `Summary.CancelReason` para exit code (RF-10): `none`→0, `activity_timeout`→1, `permission_denied`→3, demais→1; retorna `(humanBuffer.String(), "", exitCode, err)`.
- [ ] 9.8 Implementar `SetLiveOutput(w io.Writer)` para compat com `LiveOutputSetter` existente; o writer é repassado ao `HumanRenderer`.
- [ ] 9.9 Registrar `acpInvoker` em `NewAgentInvoker(tool string, runtime string)` em `internal/taskloop/agent.go`: assinatura **estendida** com parâmetro `runtime`; quando `runtime=="acp"`, retorna `acpInvoker`; default `runtime=="legacy"`, mantém comportamento atual.

### Integration Tests

- [ ] 9.10 Criar `internal/runtime/acp_integration_test.go` (sem build tag) com testes:
  - `TestACPRunner_HappyPath`: script com 3 mensagens + 2 tool calls + session_end; assertar Summary{events_count=8, unknown=0, cancel_reason=none}, `events.jsonl` com 9 linhas (incluindo runtime_init), `tool_calls.md` válido.
  - `TestACPRunner_ActivityTimeout`: script fica mudo 200ms; `activity_timeout=50ms`; assertar cancel_reason=activity_timeout, exit code mapeado.
  - `TestACPRunner_PermissionDenied`: script emite `requestPermission`; assertar cancel_reason=permission_denied, exit code 3.
  - `TestACPRunner_UnknownDrift`: script com kind desconhecido; assertar unknown_events_count>0, warning em stderr.
  - `TestACPRunner_NpxFallback`: probe simula binary ausente; assertar runtime_init.launcher=npx.
- [ ] 9.11 Criar testes do `acpInvoker` em `internal/taskloop/acpinvoker_test.go` validando mapeamento Summary→exit code e contrato `AgentInvoker`.

### Sanidade legacy

- [ ] 9.12 Garantir que toda a suíte `go test ./internal/taskloop/...` continua verde sem flag nova ativada (RF-12).

## Detalhes de Implementação

Ver `techspec.md`:
- §"Modelagem de Domínio" → "Application Service: ACPRunner"
- §"Design de Implementação" → "Interfaces Chave"
- §"Abordagem de Testes" → "Testes de Integração" (lista de cenários)
- §"Estratégia de Erros" → "Apresentação" (tabela de exit codes)

## Critérios de Sucesso

- `go test ./internal/runtime/... ./internal/taskloop/... -count=1 -race -cover` ≥ 85% agregado.
- Cinco cenários de integration verdes (lista 9.10).
- Suíte legacy intocada (RF-12).
- Sem leak de goroutines.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Os 5 cenários de integration listados em 9.10
- [ ] Testes unitários do `acpInvoker` (mapeamento + contrato)
- [ ] Sanidade da suíte legacy (RF-12)
- [ ] Race detector limpo (`-race`)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/runner.go` + `runner_test.go` (novo)
- `internal/runtime/types.go` (Job + Summary)
- `internal/runtime/options.go` (Functional Options)
- `internal/runtime/acp_integration_test.go` (novo)
- `internal/taskloop/acpinvoker.go` + `acpinvoker_test.go` (novo)
- `internal/taskloop/agent.go` (modificado: `NewAgentInvoker` aceita `runtime`)
- `internal/taskloop/runloop.go` (modificado se necessário para propagar `runtime`)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-9.0/execution_report.md`
- [ ] `go test ./internal/runtime/... ./internal/taskloop/... -count=1 -race -cover` ≥ 85%
- [ ] `golangci-lint run ./internal/runtime/... ./internal/taskloop/...` sem violações
- [ ] Suíte de testes legacy passa (não há regressão)
- [ ] Os 5 cenários de integration listados em 9.10 todos verdes
- [ ] Commit semântico `feat(runtime): add ACPRunner application service with integration tests`
