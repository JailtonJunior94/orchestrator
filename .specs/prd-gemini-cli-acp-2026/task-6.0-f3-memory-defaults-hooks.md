# Tarefa 6.0: F3-Gemini — switch tool-aware memory defaults 250/400 + hooks integration test

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar switch tool-aware em `cmd/ai_spec_harness/task_loop.go` que aplica defaults Gemini-generosos para memory 2-tier (workflow 250 linhas / 20 KiB; task 400 linhas / 32 KiB) **somente quando** `--tool gemini` e as flags `--memory-*-limit-*` **não foram setadas explicitamente**. Aproveita janela 1M+ tokens do `gemini-2.5-pro` (TD-04 da techspec). Override via flag preservado. Criar integration test (`tests/integration/gemini_hooks_test.go`) validando que hooks dispatcher (F3-Claude infra) despara hooks tool-agnósticos com driver Gemini.

**Dependência inter-PRD**: requer `.specs/prd-claude-cli-acp-2026/` F3-Claude entregue — provê `internal/runtime/hooks/dispatcher.go` e `internal/runtime/memory/store.go`. Se ausente, marcar como `blocked`.

<requirements>
- Switch tool-aware usa `cmd.Flags().Changed("memory-*-limit-*")` para detectar override explícito. Sem override + tool=gemini → defaults Gemini-generosos.
- Defaults atuais Claude/Codex/Copilot (150 workflow / 200 task / 12 KiB / 16 KiB) **preservados** quando `tool != "gemini"` — RF-30.
- Diff zero em `internal/runtime/memory/store.go` (apenas defaults variam via parâmetros).
- Integration test hooks: verifica hook `runtime.pre_open` despachado antes de `c.Open` mesmo quando driver é Gemini.
- Override CLI explícito (`--memory-task-limit-lines 600`) prevalece sobre default Gemini.
</requirements>

## Subtarefas

- [ ] 6.1 Validar pré-requisito: F3-Claude entregue (`internal/runtime/hooks/dispatcher.go` e `internal/runtime/memory/store.go` existem com `cmd.Flags().Changed`-style detection). Se ausentes, marcar `blocked`.
- [ ] 6.2 Adicionar switch tool-aware em `cmd/ai_spec_harness/task_loop.go` na lógica de resolução de defaults de memory, conforme techspec §"Switch tool-aware para defaults de memory (F3-Gemini)" (linhas ~225-250).
- [ ] 6.3 Adicionar teste T-34: `TestGeminiDefaultsMemoryLimitsAreGenerous` — CLI sem flag, `--tool gemini` resolve para 250/400.
- [ ] 6.4 Adicionar teste T-35: `TestGeminiMemoryLimitOverrideByCliFlag` — `--memory-task-limit-lines 600` prevalece; workflow ainda default Gemini.
- [ ] 6.5 Adicionar teste regressão: `TestGeminiDefaultsDoNotAffectClaudeCodexCopilot` — `--tool claude` resolve para 150/200 (preserva comportamento atual).
- [ ] 6.6 Criar `tests/integration/gemini_hooks_test.go` (build tag `//go:build integration`): hook `runtime.pre_open` é despachado antes de `c.Open` para sessão Gemini.

## Detalhes de Implementação

Ver techspec.md:
- §"Design de Implementação / Switch tool-aware para defaults de memory" (linhas ~225-250) — código exato do switch.
- §"Considerações Técnicas / TD-04" — justificativa de defaults Gemini-generosos.
- §"Mapeamento RF → Componente → Teste" — RF-16, RF-17.

Precedente: `.specs/prd-claude-cli-acp-2026/` Wave F3-Claude (memory + hooks).

## Critérios de Sucesso

- `cmd/ai_spec_harness/task_loop.go` contém switch tool-aware para defaults de memory aplicado **somente** quando `tool == "gemini"`.
- `go test -run TestGeminiDefaultsMemoryLimitsAreGenerous ./cmd/ai_spec_harness/...` retorna `PASS`.
- `go test -run TestGeminiMemoryLimitOverrideByCliFlag ./cmd/ai_spec_harness/...` retorna `PASS`.
- `go test -run TestGeminiDefaultsDoNotAffectClaudeCodexCopilot ./cmd/ai_spec_harness/...` retorna `PASS`.
- `go test -tags integration -run TestGeminiHooksDispatch ./tests/integration/...` retorna `PASS`.
- `git diff --stat internal/runtime/memory/store.go internal/runtime/hooks/dispatcher.go` retorna **zero linhas** modificadas.
- Suite regressão Claude/Codex/Copilot memory + hooks verde.

### Definition of Done

1. Switch tool-aware implementado e cirúrgico (apenas em `task_loop.go`).
2. Defaults Gemini 250/400 aplicados sem flag; override CLI prevalece.
3. Claude/Codex/Copilot continuam com defaults 150/200 (RF-30 regressão).
4. Hook dispatch validado com driver Gemini.
5. Diff zero em `internal/runtime/memory/store.go` e `internal/runtime/hooks/dispatcher.go`.
6. Dependência inter-PRD F3-Claude validada antes do início.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-34 `TestGeminiDefaultsMemoryLimitsAreGenerous` (unit)
- [ ] T-35 `TestGeminiMemoryLimitOverrideByCliFlag` (unit)
- [ ] `TestGeminiDefaultsDoNotAffectClaudeCodexCopilot` (unit, regressão)
- [ ] `TestGeminiHooksDispatch` em integration suite (hook runtime.pre_open com driver Gemini)
- [ ] Regressão: suite memory + hooks Claude/Codex/Copilot verde

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **EDIÇÃO**: `cmd/ai_spec_harness/task_loop.go` (switch tool-aware defaults memory)
- **EDIÇÃO**: `cmd/ai_spec_harness/task_loop_test.go` (T-34, T-35, regressão Claude/Codex/Copilot)
- **NOVO**: `tests/integration/gemini_hooks_test.go`
- **REFERÊNCIA (não modificar)**: `internal/runtime/memory/store.go`, `internal/runtime/hooks/dispatcher.go`
