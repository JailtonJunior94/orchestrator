# Tarefa 6.0: Flags --reasoning-effort + --access-mode + validação enum + warning full

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Registrar duas flags CLI novas em `cmd/ai_spec_harness/task_loop.go` para suportar parâmetros Codex-específicos:

- `--reasoning-effort` (string, default `"medium"`, valores aceitos: `low|medium|high`).
- `--access-mode` (string, default `"restricted"`, valores aceitos: `restricted|full`).

Implementar **validação enum** em `RunE`: valor inválido retorna `exit2` com mensagem clara listando enum aceito. Implementar **warning único via `sync.Once`** para `--access-mode=full` antes de prosseguir, alertando sobre `sandbox_mode=danger-full-access` no `codex-acp` (R-03 alto).

Inverter T-14 em `cmd/ai_spec_harness/task_loop_test.go:48-52` — `--tool codex --runtime acp` passa a ser aceito (não rejeitado). Adicionar T-15 novo cobrindo `--tool codex --runtime acp --reasoning-effort high --access-mode full`.

Para Claude/Copilot as flags são aceitas mas sem efeito (BootstrapArgs no-op da tarefa 1.0 retorna `nil`); documentar em help text.

<requirements>
- Flag --reasoning-effort com default "medium" + help text mencionando que só Codex consome.
- Flag --access-mode com default "restricted" + help text com warning sobre full.
- Validação enum no RunE antes de propagar; valor inválido → exit2 com mensagem.
- Warning único via sync.Once para --access-mode=full (mensagem explícita sobre sandbox).
- T-14 invertido: --tool codex --runtime acp passa a ser aceito.
- T-15 novo: combinação completa aceita e flags propagadas.
- T-24/T-25 cobrem casos de enum inválido.
- T-30 cobre warning único.
- task_loop_test.go suíte 100% verde após mudanças.
</requirements>

## Subtarefas

- [ ] 6.1 Registrar flag `--reasoning-effort` em `taskLoopCmd.Flags().String("reasoning-effort", "medium", "...")` com help text apropriado.
- [ ] 6.2 Registrar flag `--access-mode` similar com default `"restricted"` e help text mencionando warning de `full`.
- [ ] 6.3 No `RunE`, após `cmd.Flags().GetString(...)` dos dois valores, adicionar validação `validReasoning := map[string]bool{"low": true, "medium": true, "high": true}` (idem `validAccess`).
- [ ] 6.4 Retornar `fmt.Errorf("exit2")` com mensagem clara se valor não estiver no enum.
- [ ] 6.5 Adicionar `var accessModeFullWarnOnce sync.Once` em escopo apropriado (var package-level).
- [ ] 6.6 Se `accessMode == "full"`, invocar `accessModeFullWarnOnce.Do(func() { fmt.Fprintln(os.Stderr, "WARNING: ...") })` com mensagem do PRD HU-03/Q1.
- [ ] 6.7 Propagar `reasoningEffort` e `accessMode` para `taskloop.Options` (campo será consumido na tarefa 7.0).
- [ ] 6.8 **Inverter T-14**: linha 48-52 de `task_loop_test.go` — `--tool codex --runtime acp` passa a esperar `wantErr: false`.
- [ ] 6.9 **Adicionar T-15**: caso novo cobrindo `--reasoning-effort high --access-mode full` aceito, com asserção que valores chegam corretamente em `Options`.
- [ ] 6.10 Adicionar T-24 (reasoning inválido → exit2) e T-25 (access inválido → exit2).
- [ ] 6.11 Adicionar T-30 (warning único `full` emitido em stderr exatamente uma vez).
- [ ] 6.12 Rodar `go test ./cmd/ai_spec_harness/... ./internal/taskloop/...` → 100% verde.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `cmd/ai_spec_harness/task_loop.go — flags + catálogo` (esboço de código) e §"Sequenciamento de Desenvolvimento" → itens 6, 7, 8. Decisão registrada em ADR-013 D-08 (warning único via `sync.Once`).

Mensagem canônica do warning (Q1 do PRD): `"WARNING: --access-mode=full ativa sandbox_mode=danger-full-access no codex-acp. Pré-condição: consentimento operacional. Codex terá acesso pleno ao filesystem e à rede. Use somente em ambientes isolados. Ver CODEX.md."`

## Critérios de Sucesso

- Flags `--reasoning-effort` e `--access-mode` aparecem em `ai-spec task-loop --help`.
- Valor inválido em qualquer das duas flags retorna `exit2` com mensagem que lista enum aceito.
- `--access-mode=full` emite warning único em stderr antes de propagar.
- T-14 invertido: `--tool codex --runtime acp` (sem flags adicionais) é aceito.
- T-15 novo: combinação completa aceita e `Options.ReasoningEffort/AccessMode` corretos.
- `task_loop_test.go` suíte 100% verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-22 (Codex aceito): `--tool codex --runtime acp` → aceito; roteia para ACPRunner com defaults.
- [ ] T-23 (combinação completa): `--tool codex --runtime acp --reasoning-effort high --access-mode full` → aceito; Options.ReasoningEffort="high", Options.AccessMode=AccessModeFull; warning único emitido.
- [ ] T-24 (reasoning inválido): `--reasoning-effort invalid` → exit2 com mensagem listando `low|medium|high`.
- [ ] T-25 (access inválido): `--access-mode invalid` → exit2 com mensagem listando `restricted|full`.
- [ ] T-26 (Claude regressão): `--tool claude --reasoning-effort high --access-mode full --runtime acp` aceito; Options propagado mas BootstrapArgs no-op ignora.
- [ ] T-30 (warning único): duas invocações sucessivas de task-loop com `--access-mode=full` no mesmo processo emitem warning **apenas uma vez** (sync.Once).
- [ ] T-14 inversão validada (era rejeição, agora é aceitação).
- [ ] `go vet ./cmd/ai_spec_harness/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] Flags `--reasoning-effort` (default "medium") e `--access-mode` (default "restricted") registradas em `taskLoopCmd.Flags()`.
- [ ] Help text de cada flag menciona enum aceito; help de `--access-mode` menciona warning para `full`.
- [ ] Validação enum no `RunE` retorna `exit2` com mensagem clara em valor inválido.
- [ ] Variável `accessModeFullWarnOnce sync.Once` em escopo package-level.
- [ ] Warning para `--access-mode=full` emite mensagem do PRD HU-03/Q1 em stderr **uma vez por execução**.
- [ ] Valores propagados para `taskloop.Options.ReasoningEffort` e `Options.AccessMode` (campos a ser adicionados em Options — tarefa 7.0).
- [ ] T-14 em `task_loop_test.go:48-52` invertido (de `wantErr: true` para `wantErr: false`).
- [ ] T-15 novo cobre combinação completa com asserções de propagação.
- [ ] T-24, T-25, T-30 implementados.
- [ ] `go test ./cmd/ai_spec_harness/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.

## Arquivos Relevantes

- `cmd/ai_spec_harness/task_loop.go` (modificar: flags + validação + warning)
- `cmd/ai_spec_harness/task_loop_test.go` (inverter T-14, adicionar T-15/T-24/T-25/T-30)
- `internal/taskloop/taskloop.go` (Options ganha campos — completado na tarefa 7.0)
- ADR-013 §"Decisão" → D-08 (warning sync.Once)
- techspec.md §"Design de Implementação" → bloco `cmd/ai_spec_harness/task_loop.go`
- PRD HU-03, Q1 (mensagem do warning)
