# Índice de Relatórios de Auditoria

| data | tipo | escopo | arquivo | veredito |
|------|------|--------|---------|---------|
| 2026-04-20 | maturidade | repositório completo | [maturidade-ai-spec-harness-20-04-26-11-18.md](maturidade-ai-spec-harness-20-04-26-11-18.md) | 8/10 — pronto para agentes |
| 2026-04-20 | execution_report | telemetry feedback loop | [execution-report-telemetry-20-04-20.md](execution-report-telemetry-20-04-20.md) | APPROVED_WITH_REMARKS |
| 2026-05-18 | token-baseline | SDD strategy tokens | [sdd-token-baseline-2026-05-18.md](sdd-token-baseline-2026-05-18.md) | medição manual via `wc -c` (CLI bloqueada por `.agents/skills/tests/`) |
| 2026-05-21 | smoke-e2e | copilot --acp ACP runtime (task 10.0) | [2026-05-21T150838-copilot-acp-smoke/](2026-05-21T150838-copilot-acp-smoke/) | Smoke executado: `events.jsonl` 55 linhas, `runtime_init` tool=copilot, `EventsCount=51`, `UnknownEventsCount=3` (available_commands_update/config_option_update — Copilot-específicos), `cancel_reason=permission_denied`. Bugfix aplicado: `probe.resolve` não passava `spec.FixedArgs` ao `NewBinaryLauncher`, resultando em copilot sem `--acp`. Fix e teste adicionados em `internal/runtime/probe/`. |
