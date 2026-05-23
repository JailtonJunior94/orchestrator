# Tarefa 8.0: Auto-review opt-in via skill local + hook `session.post_review`

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar flag `--auto-review` em `cmd/ai_spec_harness/task_loop.go` que, após session end sem erro, spawna nova `ACPRunner` com prompt da skill `review` (`.agents/skills/review/SKILL.md`) + diff acumulado da task (via `git diff` no `workDir`). Parsear resultado procurando marcadores `[HARD]`/`BLOQUEADO`/`CRÍTICO` para setar `Summary.ReviewStatus="blocked"`. Adicionar hook canônico novo `PointSessionPostReview`.

<requirements>
- Flag `--auto-review` em `task_loop.go` (default `false`)
- Estender `Job` com campo `AutoReview bool`
- Em `runner.go::Run()`, após session end sem erro e antes de `EnrichReport`, se `j.AutoReview=true`:
  - Carregar skill `review` de `.agents/skills/review/SKILL.md`
  - Coletar diff: `git diff --staged + git diff` em `j.WorkDir`
  - Construir prompt via helper `buildReviewPrompt(skillBody, gitDiff) string`
  - Spawnar nova `ACPRunner` (mesmo `spec`, mesmo `factory`) com `Job{Prompt, WorkDir, EvidenceDir: filepath.Join(j.EvidenceDir, "review"), ActivityTimeout: 5*time.Minute, Quiet: true}`
  - Persistir resultado em `evidence/<task>/review/execution_report.md` + cópia em `evidence/<task>/review.md` (apontador conveniente)
  - Parsear: `parseReviewStatus(reviewSummary.Output) string` retorna `"blocked"` ou `"ok"`
  - Setar `Summary.ReviewStatus` e `Summary.ReviewPath`
- Adicionar constante `PointSessionPostReview` em `hooks/dispatcher.go`; estender Dispatcher para suportar
- Tipo evento `SessionPostReviewEvent` com `Summary *runtime.Summary`, `ReviewPath string`, `Blocked bool`
- **Recursão bloqueada**: sessão de review **não** dispara nova review session — flag `AutoReview` é forçada `false` no child Job
- `execution_report.md` do parent ganha seção "Review Block" quando `ReviewStatus="blocked"` (lista de issues hard extraídos do review output)
- T-REV-01..T-REV-04 em `runner_test.go`
- T-INT-05 em `tests/integration/claude_2026_e2e_test.go`: mock review session com `[HARD] eval() detected` → `ReviewStatus="blocked"`
- Sem regressão: sessões sem `--auto-review` rodam idêntico a task 7.0
- Cobertura ≥ 70% global mantida
</requirements>

## Subtarefas

- [ ] 8.1 Adicionar `AutoReview bool` a `Job` em `runner.go`
- [ ] 8.2 Adicionar constante `PointSessionPostReview` em `hooks/dispatcher.go` (5.0 ainda não tinha — adicionar agora)
- [ ] 8.3 Implementar helper `buildReviewPrompt(skillBody, gitDiff) string` em `runner.go` (ou helper file)
- [ ] 8.4 Implementar `runAutoReview(ctx, j, parentSummary) (ReviewResult, error)` em `runner.go`
- [ ] 8.5 Implementar `parseReviewStatus(reviewOutput) string` — procurar `[HARD]`/`BLOQUEADO`/`CRÍTICO`
- [ ] 8.6 Em `Run()` após `PointSessionPostEnd` (task 6.0), se `j.AutoReview` chamar `runAutoReview` + dispatch `PointSessionPostReview`
- [ ] 8.7 Helper `renderReviewBlockSection(reviewResult) string` em `evidence.go`; integrar em `EnrichReport`
- [ ] 8.8 Adicionar flag `--auto-review` em `task_loop.go` + propagação
- [ ] 8.9 Adicionar T-REV-01..T-REV-04 em `runner_test.go`
- [ ] 8.10 Adicionar T-INT-05 em `tests/integration/claude_2026_e2e_test.go`
- [ ] 8.11 Smoke manual: rodar `--auto-review` em task que produz código com `eval()`; checar `Summary.ReviewStatus="blocked"`

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "F5-Claude" → `buildReviewPrompt` + `parseReviewStatus`. **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- **Recursão hard-bloqueada**: ao construir o child Job para review, fazer `childJob.AutoReview = false` **explicitamente**. T-REV-04 valida.
- **Skill review tem corpo grande** — não duplicar inline; ler via `os.ReadFile(".agents/skills/review/SKILL.md")` em runtime. Falha de leitura → erro claro, **não** spawnar review.
- **`git diff` pode ser enorme**: limitar a primeiros N MB (decisão: 5 MB default; configurável via env `AISPEC_REVIEW_DIFF_MAX`). Truncar com warning prefixado no diff.
- **`parseReviewStatus` é simples**: `strings.Contains(out, "[HARD]") || strings.Contains(out, "BLOQUEADO") || strings.Contains(out, "CRÍTICO")` → `"blocked"`. Senão `"ok"`. Documentar regras no comentário.
- **Review session reusa `ACPRunner`**: mesmo `spec` (Claude), mesmo `factory`, mesmo `clock`. Apenas `Job` é diferente.
- **`ActivityTimeout` reduzido para 5 min**: review é tipicamente rápido; timeout largo demais polui evidence em casos de hang.
- **`evidence/<task>/review.md` é um apontador**: arquivo pequeno (~3 linhas) citando `evidence/<task>/review/execution_report.md` real, mais um resumo `ReviewStatus: blocked|ok`.
- **Não dispatcher mock**: T-REV-* mockam `runAutoReview` para retornar resultado canned; testes de integração T-INT-05 usam mock ACP client retornando output específico.

## Critérios de Sucesso

- T-REV-01..T-REV-04 verdes
- T-REV-04: child Job tem `AutoReview=false` mesmo quando parent tem `true` (recursão bloqueada)
- T-INT-05: review com `[HARD]` no output → `Summary.ReviewStatus="blocked"` + `execution_report.md` cita "Review Block"
- `make test` verde (cobertura ≥ 70%)
- `make integration` verde (sem regressão F2/F3/F4)
- 31 invariantes parity verdes
- Smoke manual: sessão com `--auto-review` em task pequena documenta `ReviewStatus` em `execution_report.md`
- Defaults preservam comportamento: `--auto-review` ausente → idêntico a task 7.0

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: T-REV-01..T-REV-04 em `internal/runtime/runner_test.go`
- [ ] Teste `parseReviewStatus_test.go`: cobre `[HARD]`, `BLOQUEADO`, `CRÍTICO`, ausência (retorna "ok")
- [ ] Teste `buildReviewPrompt_test.go`: snapshot do prompt esperado com skill body + diff mock
- [ ] Testes de integração: T-INT-05 em `tests/integration/claude_2026_e2e_test.go`
- [ ] Smoke manual documentado em `execution_report.md` da própria task
- [ ] Cobertura ≥ 70% global mantida

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Modificado** `internal/runtime/runner.go` (+`runAutoReview`, helpers; +campo `AutoReview` em `Job`; +campos `ReviewStatus`/`ReviewPath` em `Summary`)
- **Modificado** `internal/runtime/hooks/dispatcher.go` (+constante `PointSessionPostReview`; +tipo evento)
- **Modificado** `cmd/ai_spec_harness/task_loop.go` (+flag `--auto-review` + propagação)
- **Modificado** `internal/evidence/evidence.go` (+`renderReviewBlockSection`; integração em `EnrichReport`)
- **Modificado** `internal/runtime/runner_test.go` (+T-REV-01..T-REV-04)
- **Modificado** `tests/integration/claude_2026_e2e_test.go` (+T-INT-05)
- **Leitor:** `.agents/skills/review/SKILL.md`
- **Leitor:** `internal/runtime/specs/claude.go` (spec reusado)
