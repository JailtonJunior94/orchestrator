# Tarefa 8.0: F5-Gemini — auto-review integration test + INFO custo amplificado

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Validar via integration test que o auto-review opt-in (`--auto-review`) funciona com driver Gemini reusando 100% da infra F5-Claude (`internal/runtime/runner_autoreview.go`). Adicionar mensagem INFO única por session avisando sobre custo amplificado em janelas 1M+ tokens quando `--auto-review` + `--tool gemini` são combinados (RF-22 — RF do PRD aborda auto-review como cascata sem código novo na lógica de runner_autoreview).

**Dependência inter-PRD**: requer `.specs/prd-claude-cli-acp-2026/` F5-Claude entregue — provê `internal/runtime/runner_autoreview.go`. Se ausente, marcar como `blocked`.

<requirements>
- Zero código novo na lógica de `internal/runtime/runner_autoreview.go` (cascata F5-Claude é tool-agnóstica).
- INFO message única por session via mecanismo similar a `sync.Once` mas escopado por session: emite quando primeira combinação `--auto-review` + `--tool=gemini` é detectada. Mensagem literal: `"INFO: --auto-review com Gemini pode amplificar custo de tokens em janelas 1M+. Ver GEMINI.md §F5."`
- Integration test cobre fluxo: session principal Gemini completa → autoreview spawna child session → `evidence/<task>/review.md` é gravado.
- Hard issues no review marcam `Summary.ReviewStatus = "blocked"` (já implementado em F5-Claude; validar com driver Gemini).
- Diff zero em `internal/runtime/runner_autoreview.go`.
</requirements>

## Subtarefas

- [ ] 8.1 Validar pré-requisito: F5-Claude entregue (`internal/runtime/runner_autoreview.go` existe com `--auto-review` flag funcional). Se ausente, marcar `blocked`.
- [ ] 8.2 Adicionar INFO message única por session quando `--auto-review` + `--tool=gemini` detectados. Local sugerido: `cmd/ai_spec_harness/task_loop.go` próximo ao parsing de flags, ou `internal/taskloop/taskloop.go::Service.Execute` antes do spawn.
- [ ] 8.3 Criar `tests/integration/gemini_autoreview_test.go` (build tag `//go:build integration`) implementando T-39: `TestAutoReviewWithGeminiDriver` — session Gemini principal completa → review spawnado → `evidence/<task>/review.md` gravado com conteúdo válido.
- [ ] 8.4 Adicionar teste `TestAutoReviewBlocksOnHardIssuesWithGemini` — review com tag `[HARD]` marca `Summary.ReviewStatus = "blocked"` para sessões Gemini.
- [ ] 8.5 Adicionar teste `TestAutoReviewInfoMessageEmittedOncePerSession` — INFO emitido na primeira combinação `--auto-review` + `--tool=gemini` em uma session.

## Detalhes de Implementação

Ver techspec.md:
- §"Arquitetura do Sistema / F5-Gemini" — escopo (zero código novo no autoreview).
- §"Mensagens de Erro e Warning Literais" — texto exato do INFO.
- §"Considerações Técnicas / Riscos / R-D" — trade-off custo amplificado.
- §"Mapeamento RF → Componente → Teste" — RF-22.

Precedente: `.specs/prd-claude-cli-acp-2026/` Wave F5-Claude tasks (auto-review opt-in).

## Critérios de Sucesso

- INFO message emitido exatamente uma vez por session quando flags ativadas (testar via mock stderr).
- `go test -tags integration -run TestAutoReviewWithGeminiDriver ./tests/integration/...` retorna `PASS`.
- `go test -tags integration -run TestAutoReviewBlocksOnHardIssuesWithGemini ./tests/integration/...` retorna `PASS`.
- `go test -run TestAutoReviewInfoMessageEmittedOncePerSession ./...` retorna `PASS`.
- `git diff --stat internal/runtime/runner_autoreview.go` retorna **zero linhas** modificadas (RF-32 — autoreview core preservado).
- Suite regressão Claude autoreview verde.

### Definition of Done

1. INFO message implementada e emitida exatamente uma vez por session.
2. Integration test T-39 validando spawn de review child session com driver Gemini.
3. Hard issues bloqueando transição da task validados com Gemini driver.
4. Diff zero em `internal/runtime/runner_autoreview.go`.
5. Dependência inter-PRD F5-Claude validada antes do início.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-39 `TestAutoReviewWithGeminiDriver` (integration)
- [ ] `TestAutoReviewBlocksOnHardIssuesWithGemini` (integration)
- [ ] `TestAutoReviewInfoMessageEmittedOncePerSession` (unit)
- [ ] Regressão F5-Claude autoreview verde

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **EDIÇÃO MENOR**: `cmd/ai_spec_harness/task_loop.go` OU `internal/taskloop/taskloop.go` (INFO message, ~5 LoC)
- **NOVO**: `tests/integration/gemini_autoreview_test.go`
- **REFERÊNCIA (não modificar)**: `internal/runtime/runner_autoreview.go`
