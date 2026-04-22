# Bug Report: claude — Falha de Autenticação no Subprocesso

- **Data:** 2026-04-22
- **Ferramenta:** claude 2.1.104
- **Bundle:** tasks/maturidade
- **Severidade:** Alta

## Comando executado

```bash
./ai-spec task-loop --tool claude --max-iterations 1 --timeout 3m \
  --report-path /tmp/report_claude_maturidade_real.md tasks/maturidade
```

Internamente o harness invoca:

```bash
claude --dangerously-skip-permissions --print --bare -p "<prompt>"
```

## Comportamento esperado

O agente claude deve receber o prompt via `-p`, executar a tarefa 1.0, e retornar exit code 0 com o output da execução.

## Comportamento observado

```
task-loop iniciado: folder=tasks/maturidade tool=claude max=1 timeout=3m0s
-> iteracao 1: executando task 1.0 (Quick-Reference Mode no Contextgen)
Not logged in · Please run /login
ERRO:   erro de autenticacao detectado para claude — execute 'claude' em um terminal
        separado e faca login com '/login'
task-loop finalizado: limite de iteracoes atingido (1)
relatorio salvo em: /tmp/report_claude_maturidade_real.md
```

- Exit code: 1
- Duração: 0s
- Status da task: `pending` → `pending` (inalterado)

## Evidência objetiva

Relatório gerado em `/tmp/report_claude_maturidade_real.md`:

```
### Iteration 1: Task 1.0 — Quick-Reference Mode no Contextgen
- Duration: 0s
- Exit Code: 1
- Status Change: pending -> pending
- Note: erro de autenticacao: claude nao esta autenticado — execute 'claude' em
        um terminal separado e faca login com '/login'
- Agent Output:
  Not logged in · Please run /login
```

## Impacto

Todas as execuções reais com `--tool claude` falham imediatamente. Nenhuma task é processada. O harness reporta o erro corretamente, mas o agente nunca é invocado com sucesso.

## Hipótese de causa raiz

O Claude Code armazena autenticação em `~/.claude/` (session files), não em variáveis de ambiente. A função `cleanEnv()` em `internal/taskloop/agent.go:124` preserva todas as vars de ambiente mas a autenticação do claude é baseada em sessão local. O subprocesso (`claude --print --bare -p`) pode não encontrar a sessão ativa porque:

1. A CLI detecta que não está em modo interativo (sem TTY) e tenta revalidar a sessão
2. Ou o subprocesso tem uma visão diferente do estado de sessão do processo pai

O fato de `dry-run` funcionar (não invoca o agente) confirma que o problema está especificamente na invocação real do subprocesso `claude`.

## Próxima ação recomendada

1. Verificar se `claude --print --bare -p "test"` funciona diretamente no terminal sem ser via harness.
2. Se funcionar diretamente, investigar se há diferença no ambiente de processo (TTY, variáveis, cwd) entre execução direta e via harness.
3. Considerar adicionar um pré-check no harness que valide autenticação antes de iniciar o loop (ex: `claude --version` ou `claude auth status`).
