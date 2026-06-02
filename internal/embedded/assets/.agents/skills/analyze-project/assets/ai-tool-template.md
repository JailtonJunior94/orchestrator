# {{TOOL_NAME}}

Use `AGENTS.md` como {{TOOL_INSTRUCTION}} deste repositorio.

## Instrucoes

1. Ler `AGENTS.md` no inicio da sessao.
2. {{CONFIG_LINE_2}}
3. {{CONFIG_LINE_3}}
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
   - Go: `.agents/skills/go-implementation/SKILL.md` (+ `references/INDEX.yaml`)
   - Node/TypeScript: `.agents/skills/node-implementation/SKILL.md` (+ `references/INDEX.yaml`)
   - Python: `.agents/skills/python-implementation/SKILL.md`
   - .NET/C#: `.agents/skills/dotnet-csharp-implementation/SKILL.md` (+ `references/INDEX.yaml`)

### Carga surgical de references (economia de tokens)

Sempre leia o `references/INDEX.yaml` da skill carregada — ele mapeia cada reference para `file_patterns` e `diff_signals`. **Carregue references/*.md apenas quando o escopo da tarefa casar com a entrada do INDEX** (refs marcadas `always: true` sempre entram). Atalho: `bash .agents/scripts/resolve-references.sh <skill> <arquivos...>` lista exatamente o que carregar (le diff em stdin). Gate bloqueante quando a skill obrigatoria nao esta acessivel: `bash .agents/scripts/validate-skill-prerequisites.sh <arquivos...>`.

## Validacao

Ao concluir uma alteracao:

1. Rodar formatter nos arquivos alterados quando a stack oferecer.
2. Rodar testes direcionados aos modulos afetados.
3. Rodar lint quando disponivel e proporcional ao risco.
4. Registrar comandos executados e resultados de validacao.

{{SECAO_STACK}}
