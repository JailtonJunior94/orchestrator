# Prompt Enriquecido

```text
Valide o `task-loop` deste repositorio de forma efetiva, rigorosa, economica e enxuta em `Claude Code` e `Codex`.

Objetivo:
- executar uma validacao real do fluxo do `task-loop`
- validar obrigatoriamente em `Claude Code` e `Codex`
- comecar por `Claude Code`
- manter o escopo minimo necessario para produzir evidencia util

Ordem obrigatoria:
1. Executar `make build` na raiz do repositorio e falhar imediatamente se o build falhar.
2. Executar primeiro um `dry-run` do `task-loop` com `Claude Code`.
3. Se o `dry-run` estiver coerente, executar um lote pequeno real.
4. Repetir o mesmo fluxo em `Codex`: `dry-run` seguido de lote pequeno real.
5. Registrar evidencias objetivas do que aconteceu nas duas ferramentas.

Contexto do repositorio:
- linguagem principal: Go
- CLI: `./ai-spec`
- referencia funcional do loop: `docs/task-loop-reference.md`
- o fluxo recomendado pede `dry-run` antes de execucao real e lote pequeno nas primeiras iteracoes

Comandos esperados:
```bash
make build
./ai-spec task-loop --tool claude --dry-run /Users/jailtonjunior/Git/financialcontrol-api/tasks/prd-refatoracao-monolito-modular
./ai-spec task-loop --tool claude --max-iterations 2 --report-path ./task-loop-report-claude.md /Users/jailtonjunior/Git/financialcontrol-api/tasks/prd-refatoracao-monolito-modular
./ai-spec task-loop --tool codex --dry-run /Users/jailtonjunior/Git/financialcontrol-api/tasks/prd-refatoracao-monolito-modular
./ai-spec task-loop --tool codex --max-iterations 2 --report-path ./task-loop-report-codex.md /Users/jailtonjunior/Git/financialcontrol-api/tasks/prd-refatoracao-monolito-modular
```

Se algum comando falhar:
- interrompa a execucao
- informe o comando exato que falhou
- informe a causa provavel com base na saida observada
- nao invente correcao sem evidencia

Criterios de aceitacao:
- `make build` concluido com sucesso antes do loop
- `dry-run` executado e analisado em `Claude Code`
- execucao real feita com `Claude Code` em lote pequeno
- `dry-run` executado e analisado em `Codex`
- execucao real feita com `Codex` em lote pequeno
- relatorio salvo em `./task-loop-report-claude.md`
- relatorio salvo em `./task-loop-report-codex.md`
- resposta final em Markdown, em PT-BR, com as secoes abaixo e sem prolixidade

Formato obrigatorio da resposta:
1. `Resumo`
2. `Comandos executados`
3. `Resultado Claude Code`
4. `Resultado Codex`
5. `Comparacao objetiva`
6. `Falhas ou bloqueios`

Restricoes:
- nao expandir escopo para planejamento, refatoracao ou correcoes de codigo fora da validacao
- nao pular o `dry-run`
- nao usar mais que `--max-iterations 2` nesta primeira rodada em nenhuma das ferramentas
- nao afirmar sucesso sem citar evidencias objetivas
- manter a resposta curta e factual
```

## Justificativas das adicoes

- Forcei `dry-run` antes da execucao real porque isso e o fluxo recomendado na documentacao do `task-loop`.
- Limitei a primeira rodada a `--max-iterations 2` para reduzir risco e custo.
- Fixei o formato de saida para garantir evidencia comparavel entre `Claude Code` e `Codex`.
