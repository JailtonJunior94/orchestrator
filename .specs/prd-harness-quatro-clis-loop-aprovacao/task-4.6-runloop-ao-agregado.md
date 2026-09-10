# Tarefa 4.6: `RunLoop` conduz o `Cycle`; `BugfixLoop` reduzido a projetor de evidência (B3)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Quarta fatia do Bloco D. Migra o **último** dos três caminhos — `RunLoop`
(`internal/taskloop/runloop.go:207`, `case VerdictRejected`) — para conduzir o `approval.Cycle`.
`RunLoop` faz revisão consolidada multi-task sem critérios em escopo: a fonte é a **união** dos
critérios dos task files de `report.TasksCompleted`, deduplicada por descrição. O `BugfixLoop` sai do
ramo `VerdictRejected` e permanece apenas como **projetor de evidência/telemetria** a partir dos
eventos e rodadas do `Cycle`.

<requirements>
- RF-34: cada rodada de revisão e de correção executa em sessão/subagente novo.
- `RunLoop` conduz o `Cycle` via o adaptador de 4.3; critérios = união dos task files do lote.
  União vazia → o lote mantém o caminho legado `FinalReviewer` sem `Cycle` (defensivo).
- **G1 (techspec D-B3-G1):** `repositoryPort.Checkpoint` ganha fallback quando `git rev-parse HEAD`
  falha → `approval.NewCheckpoint(hex(sha256(diff capturado)))`. `Service.Execute` continua obtendo a
  SHA git quando há repositório (zero regressão em 4.4).
- **G2 (techspec D-B3-G2):** as fixtures de escalonamento (`TestRunLoopRejectedEscalated`,
  `TestRunLoopIntegrationEscalonamento`) passam a **variar findings por rodada** (exercita
  `ReasonMaxRounds`, `BugfixCycles == 3` preservado); teste **novo** dedicado a `ReasonNoConvergence`
  (findings idênticos → aborto na rodada 2, `Escalated`, sem retry). As asserções "findings idênticos
  → 3 ciclos" são atualizadas por conflito direto com RF-37, cada uma justificada por requisito.
  Isso toca `integration_test.go` por motivo RF-37.
- `internal/taskloop/bugfix.go` **entra no escopo**: nova função
  `bugfixAttemptsFromCycle(result approval.CycleResult) []BugfixIteration` reconstrói
  `BugfixIteration` (`Sequence`, `Origin`, `RootCause`, `FailBefore`, `PassAfter`, `ReviewVerdict`,
  `CriticalFindings`) das rodadas/eventos do `Cycle`. `LoopReport.BugfixAttempts` / `BugfixCycles` /
  `Escalated` / `FinalReview` permanecem populados; asserções nominais de `runloop_test.go` verdes.
- `CycleResult.Reason() ∈ {ReasonMaxRounds, ReasonNoConvergence, ReasonEmptyDiff}` → mapeado para
  `ErrBugfixExhausted` + `report.Escalated = true` + stop reason "escalonamento humano" existente.
  Estados terminais RF-45 — **não** disparam retry.
- `BugfixLoop.Run` e `bugfix_test.go` permanecem (ainda chamados por `applyImplementDecisions`,
  `runloop.go:301`). Testes aditivos para `bugfixAttemptsFromCycle`; nenhuma asserção existente muda.
- A virada do critério estrito **não** acontece aqui (é 5.0).
</requirements>

## Subtarefas

- [ ] 4.6.1 Coletar a união de critérios dos task files de `report.TasksCompleted` e construir o
      `Cycle`.
- [ ] 4.6.2 Substituir `NewBugfixLoop(...).Run(...)` no `case VerdictRejected` pela construção e
      `Run` do `Cycle`; ler o side-channel `*bugfixEvidenceRecorder` do adaptador após `Cycle.Run`.
- [ ] 4.6.3 Implementar `bugfixAttemptsFromCycle` em `bugfix.go` e mapear estados terminais para
      `Escalated` / `ErrBugfixExhausted`.
- [ ] 4.6.4 Ajustar cirurgicamente asserções incompatíveis em `runloop_test.go`; cada ajuste
      justificado por requisito.

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`** (ordem interna, item 6) e a subseção
**"Integração do agregado nos três caminhos (Bloco D)"** (D-B1 linha `RunLoop`, D-B3 completo).

## Critérios de Sucesso

- `grep -n 'approval\.' internal/taskloop/runloop.go internal/taskloop/bugfix.go` retorna ocorrências.
- `go test ./internal/taskloop/... -count=1` verde; asserções sobre `BugfixAttempts` / `Origin` /
  `FailBefore` / `PassAfter` / `Escalated` de `runloop_test.go` passam sem inversão.
- Teste: `RunLoop` conduz o `Cycle`; rodada em sessão nova (RF-34).
- Teste: estado terminal do `Cycle` (`ReasonMaxRounds` / `ReasonNoConvergence` / `ReasonEmptyDiff`)
  → `Escalated=true`, sem retry (RF-45).
- Teste aditivo: `bugfixAttemptsFromCycle` reconstrói `BugfixIteration` das rodadas do `Cycle`.
- `git diff internal/taskloop/runloop_test.go internal/taskloop/integration_test.go` — apenas ajustes
  justificados por requisito (RF-37 / D-B3-G2); `bugfix_test.go` sem inversão de asserção.
- `repositoryPort.Checkpoint` retorna checkpoint válido sob `FakeFileSystem` sem `.git` (fallback
  sha256 do diff — D-B3-G1); teste dedicado.
- T-REV-01/02/04 verdes e sem alteração de asserção.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/taskloop/... -count=1` verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória:

- `RunLoop` conduz o `Cycle`; rodada em sessão nova (RF-34).
- União de critérios do lote; união vazia cai no caminho legado.
- `bugfixAttemptsFromCycle` preenche `LoopReport.BugfixAttempts` com a mesma evidência de hoje.
- Estado terminal do `Cycle` não aciona retry (RF-45).
- Preservação: T-REV-01/02/04 e `bugfix_test.go` sem inversão de asserção.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/taskloop/runloop.go:177-263,301` — `RunLoop`, `case VerdictRejected`, `applyImplementDecisions`.
- `internal/taskloop/bugfix.go:83,159,186` — `BugfixLoop.Run`, `extractBugfixEvidence`,
  `formatBugfixOrigin`; nova `bugfixAttemptsFromCycle`.
- `internal/taskloop/approval_adapters.go` — adaptador e side-channel de 4.3.
- `internal/approval/cycle.go:69-134`, `internal/approval/round.go:56`, `internal/approval/result.go`.
- `internal/taskloop/runloop_test.go` — asserções a ajustar cirurgicamente.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — D-B1, D-B3.
