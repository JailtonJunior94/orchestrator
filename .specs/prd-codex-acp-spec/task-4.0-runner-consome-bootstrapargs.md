# Tarefa 4.0: runner.go::Run consome spec.BootstrapArgs e prepend ao argv

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Modificar `ACPRunner.Run(ctx, job)` em `internal/runtime/runner.go` para chamar `r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)` e fazer **prepend** ao argv do launcher antes das `FixedArgs`. Garantir comportamento idempotente para Claude/Copilot (no-op retorna `nil`, prepend de `nil` mantém argv original).

**Esta tarefa carrega o gate de invariante forense crítico (R-08)**: nenhuma mudança em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/` é permitida. Diff zero validado via `git diff --stat` antes do merge.

Ordem do argv resultante: `launcherArgs (do probe) + bootstrap (do Spec) + FixedArgs (do Spec)`. Para Codex: bootstrap contém `-c` flags; FixedArgs é vazio. Para Claude/Copilot: bootstrap é `nil`; FixedArgs continua sendo `["--bypass-permissions"]` ou `["--acp"]`.

<requirements>
- spec.BootstrapArgs(...) invocado uma vez por Run() antes do spawn.
- Prepend ao argv preserva ordem: launcherArgs + bootstrap + FixedArgs.
- Comportamento idempotente: nil bootstrap não muda argv.
- Diff zero em internal/runtime/persistence/, watchdog.go, client/.
- runtime_init event registra argv completo (já generalizado em F1-Copilot; sem mudança aqui).
- Cancel/erro paths inalterados.
- Tests T-17/T-18 (Codex spawn args) e T-19 (Claude regressão) cobrem.
</requirements>

## Subtarefas

- [ ] 4.1 Localizar ponto em `ACPRunner.Run` onde launcher é resolvido e argv montado (após `probe.EnsureAvailable`).
- [ ] 4.2 Chamar `bootstrap := r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)`.
- [ ] 4.3 Montar argv: `argv := append([]string{}, launcherArgs...); argv = append(argv, bootstrap...); argv = append(argv, r.spec.FixedArgs...)` (slice safety — não mutar slice original).
- [ ] 4.4 Passar argv ao `exec.Command` (ou método equivalente do `acpClient`).
- [ ] 4.5 Verificar que `buildRuntimeInitRaw(...)` recebe argv final completo (não apenas FixedArgs).
- [ ] 4.6 Confirmar que tratamento de `AccessMode == ""` (zero-value) é consistente com a tarefa 3.0 (default `AccessModeRestricted`).
- [ ] 4.7 Rodar `go test ./internal/runtime/...` (incluindo integration_test.go pré-existente) → 100% verde.
- [ ] 4.8 **Gate crítico R-08**: rodar `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` → deve retornar **vazio (0 arquivos)**.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `runner.go — Job estendido + ACPRunner.Run` (esboço do código) e §"Sequenciamento de Desenvolvimento" → item 4. Decisão registrada em ADR-013 D-02 (extensão da interface) e em §"Riscos Conhecidos" R-08 (diff zero obrigatório).

Anti-padrão: NÃO mutar slices recebidos do probe (`launcherArgs`); sempre criar slice novo via `append([]string{}, ...)`.

## Critérios de Sucesso

- `ACPRunner.Run` chama `BootstrapArgs(...)` exatamente uma vez por execução.
- Argv final para Codex inclui `-c model="..."`, `-c model_reasoning_effort="..."`, `-c features.code_mode=false`, etc. — validado por T-17/T-18.
- Argv final para Claude inclui `--bypass-permissions` (FixedArgs) mas **nenhum** `-c` flag — validado por T-19.
- Argv final para Copilot inclui `--acp` (FixedArgs) mas **nenhum** `-c` flag.
- `runtime_init` event registra o argv montado (visível em `events.jsonl` durante testes integration).
- Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-17 (Codex restricted): integration test confirma spawn args contêm `-c model=..., -c model_reasoning_effort=..., -c features.code_mode=false, -c features.code_mode_only=false` e **NÃO** contêm sandbox.
- [ ] T-18 (Codex full): integration test confirma spawn args contêm todos os de T-17 + `sandbox_mode="danger-full-access"`, `approval_policy="never"`, `web_search="live"`.
- [ ] T-19 (Claude regressão): integration test confirma spawn args **NÃO** contêm `-c` flags; preservam comportamento atual.
- [ ] T-31 (regressão `internal/runtime/`): suíte completa 100% verde.
- [ ] Gate R-08: `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` → **vazio**.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `ACPRunner.Run` invoca `r.spec.BootstrapArgs(job.Model, job.ReasoningEffort, job.AddDirs, job.AccessMode)` exatamente uma vez.
- [ ] Argv montado na ordem: `launcherArgs + bootstrap + FixedArgs` via `append([]string{}, ...)` sem mutação.
- [ ] Para Claude/Copilot: bootstrap é `nil`, argv resultante idêntico ao comportamento pré-F1-Codex.
- [ ] Para Codex: argv contém os `-c` overrides corretos conforme `AccessMode`.
- [ ] `runtime_init` event registra o argv completo (validado em integration test).
- [ ] **Gate R-08 (crítico)**: `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` retorna **vazio**.
- [ ] `go test ./internal/runtime/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.

## Arquivos Relevantes

- `internal/runtime/runner.go` (modificar: `ACPRunner.Run`)
- `internal/runtime/runner_test.go` (estender; adicionar/atualizar fixtures)
- `internal/runtime/specs/spec.go` (consumir método `BootstrapArgs` da tarefa 1.0)
- `internal/runtime/persistence/` (verificar diff zero)
- `internal/runtime/watchdog.go` (verificar diff zero)
- `internal/runtime/client/` (verificar diff zero)
- ADR-013 §"Decisão" → D-02; §"Riscos" → R-08
- techspec.md §"Design de Implementação" → bloco `runner.go — Job estendido`
