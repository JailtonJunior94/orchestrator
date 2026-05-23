# Tarefa 3.0: Generalizar runtime_init em runner.go

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Generalizar `internal/runtime/runner.go:113-120` para deixar de hardcodar `specs.ClaudeSDKVersion` e `specs.ClaudeNpmVersion` no evento `runtime_init`. A origem das versões passa a ser o `Spec` resolvido (`r.spec.SDKVersion()` / `r.spec.NPMVersion()`).

Esta mudança é cirúrgica e aditiva: payload do evento mantém os mesmos campos (`launcher`, `command`, `args`, `sdk_version`, `npm_version`) — apenas a origem dos valores muda. ADR-010 (tagged union) preservada; nenhum kind novo. Sob o aspecto Claude, o payload final é idêntico em valores reais (regressão T-05).

<requirements>
- buildRuntimeInitRaw e events.NewRuntimeInit recebem versões via spec.SDKVersion()/NPMVersion(), não constantes Claude.
- Payload runtime_init para Claude permanece idêntico em campos e valores reais (T-05 regressão).
- Payload runtime_init para Copilot carrega versões Copilot (T-09).
- ADR-010 preservado: nenhum kind novo; campos existentes.
- Diff zero em internal/runtime/persistence/ e internal/runtime/watchdog.go.
</requirements>

## Subtarefas

- [ ] 3.1 Em `ACPRunner.Run`, substituir `specs.ClaudeSDKVersion` por `r.spec.SDKVersion()` na chamada a `buildRuntimeInitRaw`.
- [ ] 3.2 Substituir `specs.ClaudeNpmVersion` por `r.spec.NPMVersion()` na mesma chamada.
- [ ] 3.3 Substituir o mesmo par na chamada a `events.NewRuntimeInit`.
- [ ] 3.4 Validar que `buildRuntimeInitRaw` (helper privado em `runner.go`) não precisa mudar de assinatura — apenas o caller muda.
- [ ] 3.5 Rodar `go test ./internal/runtime/... -run TestRuntimeInit -v` ou suíte completa do package.
- [ ] 3.6 Confirmar diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/` via `git diff --stat`.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/runtime/runner.go — generalização de buildRuntimeInitRaw`. Decisão D-02 (reuso total do stack ACP). Risco R-02 (downstream consumers) mitigado por T-05/T-09.

Anti-padrão: NÃO mudar campos do payload (`launcher`, `command`, `args`, `sdk_version`, `npm_version`); NÃO adicionar campos novos (preservaria ADR-010 invariante).

## Critérios de Sucesso

- Payload `runtime_init` para Claude permanece byte-idêntico em valores reais (mesmas constantes resultantes).
- Payload `runtime_init` para Copilot, quando `r.spec` for `specs.Copilot()`, carrega `sdk_version == CopilotSDKVersion` e `npm_version == CopilotNpmVersion`.
- Suíte completa de `internal/runtime/` permanece verde.
- Nenhuma mudança em `internal/runtime/persistence/` ou `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-05 (regressão Claude): teste de `runtime_init` payload em `runner_test.go` (ou em `acp_integration_test.go`) com Spec Claude continua verde com mesmos valores reais.
- [ ] T-09 (Copilot): novo caso de teste em `runner_test.go` instanciando `ACPRunner` com Spec Copilot mockada (ou real, dependendo de fixture) e verificando que payload carrega versões Copilot.
- [ ] `go test ./internal/runtime/...` → 100% verde.
- [ ] `go vet ./internal/runtime/...` → sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `ACPRunner.Run` em `runner.go` não referencia mais `specs.Claude*` constants para versões.
- [ ] Versões em `runtime_init` vêm de `r.spec.SDKVersion()` e `r.spec.NPMVersion()` exclusivamente.
- [ ] Payload `runtime_init` para Claude permanece byte-idêntico em valores (T-05).
- [ ] Novo caso T-09 cobre Copilot Spec.
- [ ] `go test ./internal/runtime/...` → 100% verde.
- [ ] **Diff zero** em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/` (verificado via `git diff --stat`).
- [ ] ADR-010 invariante mantida: nenhum kind novo de evento, nenhum campo novo no payload `runtime_init`.

## Arquivos Relevantes

- `internal/runtime/runner.go:113-120` (modificar)
- `internal/runtime/runner_test.go` (estender ou criar caso T-09)
- `internal/runtime/events/event.go` (consultar — `NewRuntimeInit` signature)
- ADR-010 (preservar invariante)
- ADR-012 §"Decisão D-02" (reuso total do stack ACP)
