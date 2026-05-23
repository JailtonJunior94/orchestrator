# Tarefa 10.0: CLI Flags + Telemetry + Live Tests + CI Workflow

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Entregar a camada de apresentação e o gating de qualidade do PRD: (a) flags `--runtime`, `--activity-timeout`, `--quiet` em `cmd/ai_spec_harness/task_loop.go` com validações (RF-01, RF-02, RF-07, RF-11); (b) campos novos na telemetria opt-in (`runtime`, `launcher`, `events_count`, `unknown_events_count`, `cancel_reason`); (c) live test em `tests/integration/acp_live/` protegido por build tag `//go:build acp_live` (RF-14); (d) job opcional `acp_live` no workflow do GitHub Actions (nightly).

<requirements>
- `--runtime` aceita `legacy` (default) e `acp`; combinação `--runtime=acp --tool != claude` falha com exit 2 e mensagem exata do RF-02.
- `--activity-timeout` aceita `time.Duration` (`90s`, `2m`); valor `0` desabilita watchdog (RF-07).
- `--quiet` suprime stream humano; jsonl + warning unknown agregado continuam (decisão #17).
- Telemetria opt-in com 5 campos novos; sem alteração no caminho legacy (não envia campos vazios).
- Live test em `tests/integration/acp_live/` compila **apenas** com build tag `acp_live`; sem a tag, `go test ./...` não tenta compilar o arquivo.
- Dentro do live test: `t.Skip` se `claude-agent-acp` ausente do PATH **e** `npx` ausente; sem `ANTHROPIC_API_KEY` válido, valida só handshake até primeiro evento.
- Workflow CI: novo job `acp-live` em `.github/workflows/ci.yml` (ou arquivo novo `acp-live.yml`) executando em schedule nightly; `continue-on-error: true` para não bloquear merges.
</requirements>

## Subtarefas

### CLI Flags

- [ ] 10.1 Em `cmd/ai_spec_harness/flags.go` adicionar definição das flags `--runtime`, `--activity-timeout` (`time.Duration` flag), `--quiet`.
- [ ] 10.2 Em `cmd/ai_spec_harness/task_loop.go` validar:
  - `--runtime` ∈ {legacy, acp}; senão exit 2 com `runtime inválido: %q — valores aceitos: legacy, acp`.
  - se `--runtime=acp`, exigir `--tool=claude`; senão exit 2 com mensagem RF-02.
  - `--activity-timeout >= 0`; senão exit 2.
- [ ] 10.3 Propagar `runtime`, `activity-timeout`, `quiet` para `taskloop.NewAgentInvoker(tool, runtime, ...)` (assinatura estendida em 9.0).
- [ ] 10.4 Testes table-driven em `cmd/ai_spec_harness/task_loop_test.go` cobrindo cada validação.

### Telemetria

- [ ] 10.5 Em `internal/telemetry/...` adicionar campos opcionais `runtime`, `launcher`, `events_count`, `unknown_events_count`, `cancel_reason` ao evento existente do `task-loop`. Manter compat com leitores atuais (campos ausentes = sem alteração).
- [ ] 10.6 No `acpInvoker` (task 9.0, repassar Summary) ou no caller, popular esses campos quando `runtime=acp`; caminho legacy permanece sem esses campos.
- [ ] 10.7 Testes em `internal/telemetry/...` validando que o caminho legacy não emite os campos novos e o caminho ACP emite os 5.

### Live Tests

- [ ] 10.8 Criar `tests/integration/acp_live/live_test.go` com primeira linha `//go:build acp_live`.
- [ ] 10.9 Implementar `TestACPLive_Handshake`: `t.Skip` se `exec.LookPath("claude-agent-acp")` e `exec.LookPath("npx")` ambos falharem; cria `Job` com prompt mínimo (`"echo OK"`) e `activity_timeout=30s`; valida que recebe ao menos `runtime_init` + uma mensagem (ou erro tipado `ErrPermissionDenied` se ambiente não pré-aprovou).
- [ ] 10.10 Criar `tests/integration/acp_live/README.md` documentando: pré-requisitos (`claude-agent-acp` instalado OU `npx` + `node`; `ANTHROPIC_API_KEY` ou config local `~/.claude/`), comando `go test -tags=acp_live ./tests/integration/acp_live`, custo esperado (aprox tokens).

### CI Workflow

- [ ] 10.11 Adicionar Makefile target `test-acp-live` que executa `go test -tags=acp_live ./tests/integration/acp_live`.
- [ ] 10.12 Criar `.github/workflows/acp-live.yml` com:
  - trigger: `schedule: - cron: "0 6 * * *"` (nightly 06:00 UTC); `workflow_dispatch` para gatilho manual.
  - jobs: instalar Go 1.26.2 + `npm install -g @agentclientprotocol/claude-agent-acp@<VERSAO_PINADA>`; rodar `make test-acp-live`; `continue-on-error: true`; upload de `evidence/` como artefato.
  - secrets: usar `secrets.ANTHROPIC_API_KEY` se configurado; senão `t.Skip` interno cobre.
- [ ] 10.13 Não alterar `.github/workflows/ci.yml` principal (o live test continua opt-in).

## Detalhes de Implementação

Ver `techspec.md`:
- §"Endpoints de API" (tabela de flags)
- §"Monitoramento e Observabilidade" (campos de telemetria)
- §"Abordagem de Testes" → "Testes Live"
- §"Plano de Rollout" item 4
- PRD RF-01, RF-02, RF-07, RF-11, RF-14

## Critérios de Sucesso

- `go build ./cmd/ai_spec_harness/...` passa.
- `go test ./cmd/ai_spec_harness/...` cobre todas as validações de flag.
- `go test ./...` sem `-tags=acp_live` **não compila** os arquivos do live test (validar com `go list -f '{{.GoFiles}}' ./tests/integration/acp_live` retornando vazio).
- `go test -tags=acp_live ./tests/integration/acp_live` com binário ausente `t.Skip` corretamente.
- Workflow YAML valida com `actionlint` (ou GitHub Actions UI no PR final).
- Telemetria nova só aparece quando `runtime=acp` (validado em test).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] `TestTaskLoopFlags_Runtime`: tabela cobrindo `legacy`, `acp`, valor inválido, combo `acp + tool != claude`
- [ ] `TestTaskLoopFlags_ActivityTimeout`: tabela cobrindo `120s`, `0`, valor negativo, valor inválido
- [ ] `TestTaskLoopFlags_Quiet`: validar propagação até `acpInvoker`
- [ ] `TestTelemetry_LegacyUnchanged`: caminho legacy não emite campos novos
- [ ] `TestTelemetry_ACPFields`: caminho ACP emite 5 campos com valores corretos
- [ ] `TestACPLive_Handshake`: roda apenas com build tag; `t.Skip` se sem binário
- [ ] `actionlint .github/workflows/acp-live.yml` sem erros (ou validação manual)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `cmd/ai_spec_harness/task_loop.go` (modificado)
- `cmd/ai_spec_harness/flags.go` (modificado)
- `cmd/ai_spec_harness/task_loop_test.go` (modificado)
- `internal/telemetry/*.go` (modificado)
- `tests/integration/acp_live/live_test.go` (novo, build tag `acp_live`)
- `tests/integration/acp_live/README.md` (novo)
- `Makefile` (modificado: target `test-acp-live`)
- `.github/workflows/acp-live.yml` (novo)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-10.0/execution_report.md`
- [ ] `go test ./cmd/ai_spec_harness/... ./internal/telemetry/... -count=1 -race -cover` ≥ 85%
- [ ] `go test ./...` (sem tag) não compila live test
- [ ] `go test -tags=acp_live ./tests/integration/acp_live` `t.Skip` corretamente em ambiente sem binário
- [ ] `actionlint .github/workflows/acp-live.yml` limpo (ou inspeção manual documentada)
- [ ] Commit semântico `feat(cli): add --runtime/--activity-timeout/--quiet flags, telemetry fields and acp_live CI job`
