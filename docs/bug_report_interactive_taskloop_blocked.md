# Bug Report: prd-interactive-task-loop — Tasks sem Dependência Marcadas como blocked

- **Data:** 2026-04-22
- **Ferramenta:** todas (claude, copilot, gemini, codex)
- **Bundle:** tasks/prd-interactive-task-loop
- **Severidade:** Média

## Comando executado

```bash
./ai-spec task-loop --tool <tool> --dry-run tasks/prd-interactive-task-loop
./ai-spec task-loop --tool <tool> --max-iterations 1 tasks/prd-interactive-task-loop
```

## Comportamento esperado

Tasks 1.0 e 2.0 têm dependências declaradas como `—` (nenhuma) em `tasks.md`. Com status `pending`, devem ser elegíveis para execução pelo harness. O loop deve iniciar pelo menos 1 iteração.

## Comportamento observado

```
task-loop iniciado: folder=tasks/prd-interactive-task-loop tool=<tool> max=20 timeout=30m0s
task-loop finalizado: nenhuma task elegivel (restantes estao bloqueadas, falharam ou aguardam input)
relatorio salvo em: task-loop-report-<timestamp>.md
```

- Iterações: **0** para todas as 4 ferramentas
- Status no `tasks.md`:
  ```
  | 1.0 | Criar utilitários de stream | blocked | — | Com 2.0 |
  | 2.0 | Evoluir interface AgentInvoker com out io.Writer | blocked | — | Com 1.0 |
  | 3.0 | Integrar streaming em Execute | pending | 1.0, 2.0 | Não |
  | 4.0 | Adicionar flag --no-stream na CLI | pending | 3.0 | Não |
  ```
- Status nos arquivos de task individuais:
  - `1.0-stream-utils.md`: `**Status:** blocked`
  - `2.0-agent-invoker.md`: `**Status:** blocked`

## Evidência objetiva

Relatório gerado:

```
## Final Task Status
| 1.0 | Criar utilitários de stream | blocked |
| 2.0 | Evoluir interface AgentInvoker | blocked |
| 3.0 | Integrar streaming em Execute | pending |
| 4.0 | Adicionar flag --no-stream | pending |
```

Comportamento idêntico em claude, copilot, gemini e codex — confirma que não é divergência entre ferramentas, mas estado dos dados.

## Impacto

O bundle `prd-interactive-task-loop` está permanentemente bloqueado. Nenhuma ferramenta consegue executar qualquer iteração. Tasks 3.0 e 4.0 dependem de 1.0/2.0, então também são inelegíveis em cadeia.

## Hipótese de causa raiz

Tasks 1.0 e 2.0 foram marcadas como `blocked` manualmente (possivelmente aguardando decisão de design ou input externo), mas sem registrar o motivo do bloqueio. O harness trata `blocked` como estado não elegível — correto conforme o contrato. O problema é que o bloqueio não tem causa documentada nem mecanismo de desbloqueio.

Não é bug do harness — é anomalia de dados nos arquivos de task.

## Próxima ação recomendada

1. Verificar por que tasks 1.0 e 2.0 foram marcadas como `blocked` — consultar git log dos arquivos.
2. Se o bloqueio foi intencional (ex: aguardando decisão de arquitetura), documentar a razão em campo `blocked-reason` ou similar no arquivo de task.
3. Se o bloqueio foi acidental, alterar status para `pending` em `tasks.md` e nos arquivos individuais.
4. Considerar adicionar ao harness um aviso explícito quando um bundle inicia com 0 iterações e todas as tasks em estado `blocked` sem dependências (pode indicar bloqueio acidental).
