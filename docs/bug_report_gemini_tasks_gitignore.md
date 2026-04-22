# Bug Report — Gemini: read_file bloqueado por tasks/ no .gitignore

**Data:** 2026-04-22
**Ferramenta:** gemini (0.38.2)
**Bundle:** todos (tasks/maturidade, tasks/prd-interactive-task-loop, etc.)

## Comando executado

```bash
./ai-spec task-loop --tool gemini --max-iterations 1 --timeout 3m \
  --report-path /tmp/report-gemini-cross-cli.md \
  tasks/test-cross-cli-validation
```

## Comportamento esperado

Gemini lê `tasks/test-cross-cli-validation/task-1.0-validar-leitura-agents.md` via `read_file` sem erros.

## Comportamento observado

```
Error executing tool read_file: File path
'/Users/jailtonjunior/Git/orchestrator/tasks/test-cross-cli-validation/task-1.0-validar-leitura-agents.md'
is ignored by configured ignore patterns.
```

Gemini tenta múltiplas abordagens (`list_directory`, `glob`, `read_file`) antes de cair em `run_shell_command` para ler o arquivo via shell.

## Evidência objetiva

- `.gitignore` linha 19: `tasks/`
- Gemini CLI respeita `.gitignore` como padrão de ignore para `read_file`
- Claude e Copilot não respeitam `.gitignore` para leitura de arquivos — divergência de comportamento cross-CLI
- Copilot leu o mesmo arquivo com `Read` sem nenhum erro
- Workaround automático do gemini: `run_shell_command` com `cat`/`sed` — funcional mas frágil

## Impacto

- Todos os bundles em `tasks/` ficam com acesso de leitura degradado para gemini
- Gemini só consegue operar via fallback shell, aumentando latência e risco de falha em ambientes sem shell compatível
- Divergência observável: claude e copilot acessam os arquivos normalmente; gemini não

## Hipótese de causa raiz

Gemini CLI usa `.gitignore` como lista de ignore para a ferramenta `read_file`, provavelmente por herdar o mecanismo de contexto do repositório que exclui arquivos ignorados pelo git. Claude e Copilot ignoram esta restrição para `read_file`.

## Severidade

**alta** — afeta todos os bundles de validação; workaround existe mas é frágil.

## Próxima ação recomendada

**Opção A (preferida):** Remover `tasks/` do `.gitignore` ou criar um arquivo `.geminiignore` separado que não inclua `tasks/`.

**Opção B:** Adicionar em `GEMINI.md` uma orientação explícita para usar `run_shell_command` como fallback quando `read_file` falhar em `tasks/`.

**Opção C:** Investigar se o Gemini CLI oferece flag `--no-ignore` equivalente ao `--no-gitignore` do ripgrep.
