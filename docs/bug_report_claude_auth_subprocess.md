# Bug Report — Claude: autenticação não propagada para subprocesso

**Data:** 2026-04-22
**Ferramenta:** claude (2.1.104)
**Bundle:** tasks/test-cross-cli-validation (e qualquer bundle)

## Comando executado

```bash
./ai-spec task-loop --tool claude --max-iterations 1 --timeout 3m \
  --report-path /tmp/report-claude-cross-cli.md \
  tasks/test-cross-cli-validation
```

## Comportamento esperado

Claude CLI executa o prompt do task-loop usando a sessão autenticada disponível no ambiente.

## Comportamento observado

```
Not logged in · Please run /login
ERRO: erro de autenticacao detectado para claude — execute 'claude' em
um terminal separado e faca login com '/login', ou defina
ANTHROPIC_API_KEY no ambiente para uso nao-interativo
```

O harness detectou o erro corretamente e registrou o motivo no relatório. Task permaneceu `pending`.

## Evidência objetiva

- `ANTHROPIC_API_KEY`: não definida no ambiente do subprocesso
- `claude --version`: 2.1.104 instalado e acessível no PATH
- Ambiente atual: Claude Code (processo pai) está autenticado, mas o subprocesso `claude` invocado pelo harness usa `cleanEnv()` que preserva `ANTHROPIC_API_KEY` somente se já estiver no ambiente pai
- Relatório: `exit=1`, `falhadas: 1`, nota: `"erro de autenticacao"`

## Impacto

- Claude é completamente bloqueado em ambientes onde `ANTHROPIC_API_KEY` não está definida como variável de ambiente
- Em ambientes CI/CD, requer configuração explícita do secret `ANTHROPIC_API_KEY`
- Em uso local via Claude Code: o subprocesso `claude` não herda a autenticação do processo pai (Claude Code)

## Hipótese de causa raiz

O claude CLI em modo não-interativo (`--print --bare`) requer `ANTHROPIC_API_KEY` ou login interativo via `/login`. O harness invoca claude como subprocesso sem modo TTY, impossibilitando o login interativo. `cleanEnv()` em `internal/taskloop/agent.go:147` preserva o ambiente mas não injeta credenciais ausentes.

## Severidade

**crítica em CI** / **informativa em dev** — comportamento esperado para ambientes sem API key, mas bloqueia todo uso do harness com `--tool claude` nesses cenários.

## Próxima ação recomendada

1. Documentar no `README.md` e `docs/troubleshooting.md` que `ANTHROPIC_API_KEY` é obrigatória para uso não-interativo com `--tool claude`.
2. No harness, antes de invocar claude, verificar se `ANTHROPIC_API_KEY` está no ambiente e emitir aviso antecipado (pre-flight check), evitando a invocação desnecessária.
3. Considerar adicionar `--tool claude` ao checklist de configuração de CI do repositório.
