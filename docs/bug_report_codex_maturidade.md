# Bug Report: codex — Zero Output Durante Execução Real

- **Data:** 2026-04-22
- **Ferramenta:** codex-cli 0.122.0
- **Bundle:** tasks/maturidade
- **Severidade:** Média

## Comando executado

```bash
./ai-spec task-loop --tool codex --max-iterations 1 --timeout 3m \
  --report-path /tmp/report_codex_maturidade_real.md tasks/maturidade
```

Internamente o harness invoca:

```bash
codex exec --dangerously-bypass-approvals-and-sandbox "<prompt>"
```

## Comportamento esperado

O agente codex deve processar o prompt e produzir output visível no terminal (streaming) e/ou no relatório, similar ao que gemini e copilot produzem.

## Comportamento observado

```
task-loop iniciado: folder=tasks/maturidade tool=codex max=1 timeout=3m0s
-> iteracao 1: executando task 1.0 (Quick-Reference Mode no Contextgen)
  resultado: pending -> pending (exit=-1, duracao=3m0s)
task-loop finalizado: limite de iteracoes atingido (1)
relatorio salvo em: /tmp/report_codex_maturidade_real.md
```

- Exit code: -1 (timeout kill via SIGKILL)
- Duração: exatos 3m0s (timeout esgotado)
- Output streaming: **nenhum** — campo `Agent Output` do relatório está vazio
- Status da task: `pending` → `pending` (inalterado)

## Evidência objetiva

Relatório gerado em `/tmp/report_codex_maturidade_real.md`:

```
### Iteration 1: Task 1.0 — Quick-Reference Mode no Contextgen
- Duration: 3m0s
- Exit Code: -1
- Status Change: pending -> pending
- Note: agente saiu com codigo -1; status inalterado apos execucao; pulando
- Agent Output: (vazio)
```

Contraste com gemini (mesmo bundle, mesmo timeout):

```
- Agent Output:
  MCP issues detected. Run /mcp list for status.
  I will start by reading AGENTS.md...
  I will list the files...
  [10 passos de raciocínio capturados]
```

## Impacto

Impossível diagnosticar o que o codex está fazendo durante execução. Sem output, qualquer falha é opaca — não é possível distinguir entre: (a) codex travado aguardando input, (b) codex trabalhando silenciosamente, (c) codex falhando internamente.

## Hipótese de causa raiz

O `codex exec --dangerously-bypass-approvals-and-sandbox` pode:

1. **Requer TTY interativo**: O codex pode redirecionar seu output para o terminal diretamente (usando `/dev/tty` ou similar) em vez de `stdout`, ignorando o `io.MultiWriter` configurado pelo harness.
2. **Modo silencioso por padrão**: A flag `exec` pode suprimir streaming por design quando não há TTY.
3. **Pipe fechado**: O codex pode detectar que `stdout` é um pipe e mudar o comportamento de output.

Nota: a invocação `codex exec` usa `args = append(args, prompt)` como argumento posicional — diferente de claude/gemini/copilot que usam `-p`. Verificar se o prompt está sendo recebido corretamente.

## Próxima ação recomendada

1. Executar `codex exec --dangerously-bypass-approvals-and-sandbox "test prompt" 2>&1` diretamente no terminal para verificar se produz output.
2. Verificar se codex suporta output para pipe (ex: `codex exec ... | cat`).
3. Se codex requer TTY, considerar invocação via `script -q /dev/null codex exec ...` ou similar para simular terminal no harness.
4. Verificar se o argumento posicional do prompt está sendo passado corretamente (comparar com `-p` usado pelas outras CLIs).
