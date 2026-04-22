# Bug Report — Gemini: quota throttling causa timeout em tarefas simples

**Data:** 2026-04-22
**Ferramenta:** gemini (0.38.2)
**Bundle:** tasks/test-cross-cli-validation

## Comando executado

```bash
./ai-spec task-loop --tool gemini --max-iterations 1 --timeout 3m \
  --report-path /tmp/report-gemini-cross-cli.md \
  tasks/test-cross-cli-validation
```

## Comportamento esperado

Gemini executa a task 1.0 (leitura de AGENTS.md, resposta em 3 linhas, atualização de status) dentro de 3 minutos.

## Comportamento observado

Múltiplos erros de quota ao longo de toda a execução:

```
Attempt 1 failed: You have exhausted your capacity on this model.
Your quota will reset after 4s.. Retrying after 5159ms...
Attempt 1 failed: You have exhausted your capacity on this model.
Your quota will reset after 5s.. Retrying after 5089ms...
[... 14 ocorrências adicionais ...]
```

A execução total durou exatamente 3m0s (timeout atingido). O agente completou a task antes do kill, mas saiu com `exit=-1`.

## Evidência objetiva

- 14+ mensagens de retry com delays de 5-8s cada
- Duration: `3m0s` (limite exato)
- Exit code: `-1` (SIGKILL por timeout)
- Task marcada como `done` (trabalho concluído antes do kill)
- Saída inclui: `MCP issues detected. Run /mcp list for status.`

## Impacto

- Tarefas simples (read-only, sem alteração de código) já consomem todo o timeout de 3m
- Tasks reais de implementação (que exigem escrever código, rodar testes, revisar) quase certamente expirarão com gemini
- O timeout padrão de 30m (`--timeout 30m0s`) pode ser suficiente, mas não foi testado nesta sessão

## Hipótese de causa raiz

Rate limit por volume de chamadas ao modelo no nível da conta. Gemini CLI usa retry automático com backoff linear (5-8s), mas o overhead total de 14+ retries ultrapassa o timeout de 3m configurado no teste. O timeout de 3m foi escolhido para o teste; o padrão do harness é 30m.

## Severidade

**média** — O timeout de teste (3m) é artificialmente baixo. Com o timeout padrão de 30m o trabalho seria concluído, conforme evidenciado pela task marcada como `done` antes do SIGKILL. Porém, o falso-negativo no relatório (Bug #harness-exit-minus1) mascara o sucesso real.

## Próxima ação recomendada

1. Usar o timeout padrão (30m) para validações com gemini, não reduzir para testes.
2. Investigar se o modelo configurado para gemini está atingindo limits de quota específicos da conta (modelo padrão: verificar com `gemini --version` e configuração de conta).
3. Correlacionar com o bug de `exit=-1 falso-negativo` para não subestimar capacidade real do gemini.
