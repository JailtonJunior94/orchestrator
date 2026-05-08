# Equivalência CLI ↔ skill: `ai-spec task-loop` e `execute-task`

<!-- contrato canônico RF-08 — não editar sem ADR -->

## Propósito

Operar via `ai-spec task-loop` é semanticamente idêntico a operar via skill `execute-task`
em qualquer agente IA. Esta paridade é o **escape hatch** para ambientes onde o agente
não pode reter contexto entre passos (ex: Copilot stateless — ADR-007) e para pipelines
CI/CD sem acesso interativo ao agente.

## Mapeamento de etapas

| Etapa skill | Método `Service.RunLoop` | Flag CLI `task-loop` |
|---|---|---|
| validate eligibility | `s.preflight(ctx, workDir)` | automático (sem flag) |
| load context | `filepath.Abs` + `s.fsys.Exists` | `<prd-folder>` (arg posicional) |
| execute | `deps.Executor.Execute(ctx, ...)` | `--tool` / `--executor-tool` |
| validate | `deps.Gate.Verify(ctx, ...)` | automático (sem flag) |
| capture evidence | `deps.Recorder.Append(ctx, ...)` | `--report-path` |
| review | `deps.FinalReviewer.ReviewConsolidated(ctx, ...)` | `--reviewer-tool` / `--reviewer-model` |
| bugfix | `BugfixLoop.Run(ctx, ...)` | `--max-iterations` |
| close | `s.finalizeReport(report, opts, ...)` | exit code 0 |

## Quando usar CLI vs skill

- **Skill `execute-task`**: agente IA com contexto persistente (Claude Code, Gemini CLI); acesso
  direto ao filesystem e git. Preferido quando disponível.
- **CLI `ai-spec task-loop`**: ambientes restritos sem contexto persistente (Copilot stateless,
  CI/CD, scripts), ou quando for necessário rotacionar executor e reviewer com modelos distintos
  via `--executor-tool`/`--reviewer-tool`.

> **Nota:** as variáveis de ambiente `AI_TASKS_ROOT`, `AI_PRD_PREFIX`, `AI_TOOL` e
> `AI_INVOCATION_DEPTH` são exportadas por `scripts/lib/check-invocation-depth.sh` e
> consumidas por ambos os caminhos, garantindo paridade de configuração entre agentes.
