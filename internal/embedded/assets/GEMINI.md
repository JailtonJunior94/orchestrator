# Gemini CLI

Use `AGENTS.md` como fonte canonica das regras deste repositorio.

## Instrucoes

1. Ler `AGENTS.md` no inicio da sessao.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. `.gemini/commands/` sao adaptadores finos que apontam para a habilidade correta.
4. Em tarefas de execucao, carregar apenas `AGENTS.md`, `agent-governance` e a skill operacional da linguagem ou atividade afetada.
5. Skills de planejamento (`analyze-project`, `create-prd`, `create-technical-specification`, `create-tasks`) entram apenas quando a tarefa pedir esse fluxo explicitamente.
6. Carregar referencias adicionais apenas quando a tarefa exigir.
7. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
8. Validar mudancas com comandos proporcionais ao risco.

## Contrato de Carga Base

Antes de editar codigo, confirmar que estes arquivos foram lidos na sessao:

1. `AGENTS.md` — regras de arquitetura, modo de trabalho e restricoes.
2. `.agents/skills/agent-governance/SKILL.md` — governanca, DDD, erros, seguranca e testes sob demanda.
3. A skill de linguagem correspondente quando a tarefa alterar codigo:
   - Go: `.agents/skills/go-implementation/SKILL.md`
   - Node/TypeScript: `.agents/skills/node-implementation/SKILL.md`
   - Python: `.agents/skills/python-implementation/SKILL.md`

## Validacao

Ao concluir uma alteracao:

1. Rodar formatter nos arquivos alterados quando a stack oferecer.
2. Rodar testes direcionados aos modulos afetados.
3. Rodar lint quando disponivel e proporcional ao risco.
4. Registrar comandos executados e resultados de validacao.

## Hooks de Governanca

O Gemini CLI nao suporta hooks automaticos (PreToolUse/PostToolUse) nativos. Os scripts em `.gemini/hooks/` sao utilitarios manuais invocados explicitamente ou via `--hook`.

| Script | Finalidade |
|--------|-----------|
| `.gemini/hooks/validate-preload.sh <file>` | Lembrete de carga base antes de editar codigo |
| `.gemini/hooks/validate-governance.sh <file>` | Aviso ao modificar arquivos de governanca |

```bash
# Invocar manualmente antes/apos edicao
bash .gemini/hooks/validate-preload.sh path/to/file.go
bash .gemini/hooks/validate-governance.sh .agents/skills/review/SKILL.md
```

## Orientacoes Especificas para Gemini

1. Ao iniciar uma tarefa, ler `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` como contexto base antes de editar codigo.
2. Usar `@<command>` para invocar o comando TOML correspondente a skill desejada em `.gemini/commands/`.
3. Seguir as etapas procedurais do SKILL.md carregado pelo comando como se fossem instrucoes sequenciais.
4. Ao final da tarefa, executar os comandos de validacao descritos na secao Validacao do `AGENTS.md`.
5. Nao confiar em enforcement automatico — a compliance depende de seguir as instrucoes procedurais manualmente.
