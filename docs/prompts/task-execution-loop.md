# Prompt operacional do `Service.RunLoop`

Este documento descreve o fluxo esperado quando o `task-loop` executa o orquestrador consolidado de `internal/taskloop`.

## Ordem do fluxo

1. Selecionar a proxima task elegivel em `tasks.md`.
2. Executar a task com `execute-task`.
3. Validar criterios de aceite com `AcceptanceGate`.
4. Registrar evidencias no arquivo da task com `EvidenceRecorder`.
5. Repetir ate esgotar tasks elegiveis.
6. Rodar a `review` uma unica vez sobre o diff consolidado.
7. Se o veredito for `REJECTED`, entrar no `BugfixLoop` com limite de 3 iteracoes.
8. Se o veredito for `APPROVED_WITH_REMARKS`, abrir plano de acao por ressalva.

## Contexto minimo da revisao final

O payload da revisao consolidada deve sempre carregar:

- caminho do `prd.md`
- caminho do `techspec.md`
- caminho do `tasks.md`
- caminho da ultima task executada, quando existir
- lista de tasks concluidas no lote
- diff consolidado atual

Sem esse contexto, a skill `review` nao consegue validar RFs e criterios de aceite com confianca.

## Telemetria canonica

Quando `GOVERNANCE_TELEMETRY=1`, o fluxo deve emitir eventos em stderr no formato:

```text
[taskloop] event=<nome> value=<valor> ts=<RFC3339>
```

Eventos esperados:

- `task_completed`
- `acceptance_failed`
- `final_review_verdict`
- `bugfix_iteration`
- `implement_promoted`
- `escalated`

## Modo nao interativo

Quando `--non-interactive` estiver ativo, cada ressalva deve assumir `Document` como acao default e gerar follow-up em `tasks.md`.
