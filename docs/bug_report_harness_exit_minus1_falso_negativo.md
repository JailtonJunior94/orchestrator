# Bug Report — Harness: exit=-1 com task done contabilizado como falha

**Data:** 2026-04-22
**Ferramenta:** gemini (0.38.2) e copilot (VS Code extension)
**Bundle:** tasks/test-cross-cli-validation

## Comando executado

```bash
./ai-spec task-loop --tool gemini --max-iterations 1 --timeout 3m \
  --report-path /tmp/report-gemini-cross-cli.md \
  tasks/test-cross-cli-validation

./ai-spec task-loop --tool copilot --max-iterations 1 --timeout 3m \
  --report-path /tmp/report-copilot-cross-cli.md \
  tasks/test-cross-cli-validation
```

## Comportamento esperado

Quando o agente marca a task como `done` antes do timeout ser atingido, a iteração deve ser contabilizada como **sucesso** no relatório final, independente do exit code do processo.

## Comportamento observado

Relatório final de ambas as ferramentas:

```
- **Executadas com sucesso:** 0
- **Puladas:** 0
- **Falhadas:** 1
```

Tabela de resultados:
```
| 1 | 1.0 | ... | pending | done | 3m0s | -1 |
```

A nota registrada: `"agente saiu com codigo -1"` — mesmo com `Post-Status: done`.

## Evidência objetiva

- `task-1.0-validar-leitura-agents.md`: `**Status:** done` após execução
- `tasks.md`: linha da task atualizada para `done`
- `1.0_execution_report.md`: gerado pelo agente (copilot), contendo evidências completas
- Relatório do harness: `exit=-1` → `"Falhadas: 1"` — classificação incorreta dado o estado final da task

## Impacto

- Relatórios cross-CLI de sessões com timeout aparecem como "0 sucesso" mesmo quando o agente completou o trabalho
- Métricas de telemetria (`ai-spec telemetry report`) contabilizariam essas execuções como falhas
- Dificulta comparação objetiva entre ferramentas: gemini e copilot aparentam falhar, claude aparenta ter bloqueio de auth — nenhuma aparece como "sucesso"

## Hipótese de causa raiz

O harness classifica sucesso/falha com base no exit code do processo (`exit == 0` → sucesso), sem verificar a transição de status da task (`Pre-Status → Post-Status`). Quando o agente completa o trabalho mas o processo é SIGKILL'd pelo timeout, o exit code é -1 mas o resultado real é `done`.

Localização provável: `internal/taskloop/taskloop.go` — lógica de decisão de "sucesso" pós-execução.

## Severidade

**média** — relatórios falsos-negativos afetam rastreabilidade; a task está corretamente marcada como `done` no filesystem.

## Próxima ação recomendada

Em `taskloop.go`, considerar a transição `Pre-Status → done` como critério primário de sucesso, com exit code como critério secundário. Exemplo de lógica:

```
if postStatus == "done" {
    contabilizar como sucesso
} else if exitCode == 0 && statusUnchanged {
    contabilizar como pulada (sem progresso)
} else {
    contabilizar como falha
}
```
