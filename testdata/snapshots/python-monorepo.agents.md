<!-- governance-schema: 1.0.0 -->
# Regras para Agentes de IA

Este diretorio centraliza regras para uso com agentes de IA em tarefas reais de analise, alteracao e validacao de codigo.

## Objetivo

Use estas instrucoes para manter consistencia, seguranca e qualidade ao trabalhar com codigo, configuracao, validacao e evolucao de sistemas.

## Arquitetura: monorepo

O projeto aparenta ser um monorepo, com multiplos componentes ou workspaces sob a mesma raiz. A governanca deve preservar fronteiras entre pacotes e validar apenas os workspaces afetados.

Stack detectada: stack principal nao detectada automaticamente.
Frameworks detectados: FastAPI.

## Estrutura de Pastas

```
.
README.md
packages
packages/shared
packages/shared/pyproject.toml
services
services/api
services/api/pyproject.toml
services/api/tests
services/api/tests/__init__.py
services/worker
services/worker/pyproject.toml
services/worker/tests
services/worker/tests/__init__.py
```

## Padrao Arquitetural

Predominio de arquitetura em camadas, com separacao entre transporte, servicos, persistencia e modelos.

### Fluxo de Dependencias

- Dependencias devem apontar de bordas externas para o nucleo do negocio.
- Detalhes de framework, IO e persistencia nao devem vazar para o centro do sistema.

## Modo de trabalho

1. Entender o contexto antes de editar qualquer arquivo.
2. Preferir a menor mudanca segura que resolva a causa raiz.
3. Preservar arquitetura, convencoes e fronteiras ja existentes no contexto analisado.
4. Nao introduzir abstracoes, camadas ou dependencias sem demanda concreta.
5. Atualizar ou adicionar testes quando houver mudanca de comportamento.
6. Rodar validacoes proporcionais a mudanca.
7. Registrar bloqueios e suposicoes explicitamente quando o contexto estiver incompleto.

## Diretrizes de Estrutura

1. Priorize entendimento do codigo e do contexto atual antes de propor refatoracoes.
2. Respeite padroes existentes de nomenclatura, organizacao e tratamento de erro.
3. Defina estrutura simples, evolutiva e com defaults explicitos.
4. Evite reescritas amplas quando uma alteracao localizada resolver o problema.
5. Estabeleca contratos, testes e comandos de validacao cedo quando eles ainda nao existirem.
6. Considere risco de regressao como restricao principal.
7. Evite overengineering disfarcado de arquitetura futura.

## Regras por Arquitetura

1. Limitar mudancas ao workspace, pacote ou servico afetado.
2. Nao criar dependencias internas cruzadas sem contrato explicito.
3. Validar primeiro apenas os workspaces impactados antes de ampliar o escopo.

## Regras por Linguagem

Para tarefas que alteram codigo, carregar a skill:

- `.agents/skills/agent-governance/SKILL.md`



Para tarefas de correcao de bugs com remediacao e teste de regressao, carregar tambem:

- `.agents/skills/bugfix/SKILL.md`

### Composicao Multi-Linguagem

Em projetos com mais de uma linguagem (ex: monorepo Go + Node), carregar apenas a skill da linguagem afetada pela mudanca. Se a tarefa cruzar linguagens, carregar ambas e aplicar a validacao de cada stack nos arquivos correspondentes. Nao misturar convencoes de uma linguagem em arquivos de outra.

## Referencias

Cada skill lista suas proprias referencias em `references/` com gatilhos de carregamento no respectivo `SKILL.md`. Nao duplicar a listagem aqui — consultar o SKILL.md da skill ativa para saber quais referencias carregar e em que condicao.

## Notas por Ferramenta

- **Claude Code**: skills pre-carregadas via `.claude/skills/`, hooks via `.claude/hooks/`, agents delegados via `.claude/agents/`.
- **Gemini CLI**: commands em `.gemini/commands/*.toml` apontam para skills canonicas. Sem hooks ou agents nativos — o modelo deve seguir as instrucoes procedurais do SKILL.md carregado.
- **Codex**: le `AGENTS.md` como instrucao de sessao. Entradas em `.codex/config.toml` sao metadados para `upgrade.sh`, nao spec oficial do Codex CLI. O agente deve seguir as instrucoes de `AGENTS.md` para descobrir e carregar skills.
- **Copilot**: `.github/copilot-instructions.md` como instrucao principal. `.github/agents/` sao wrappers. Sem hooks nativos — compliance depende do modelo seguir as instrucoes.

### Matrix de Enforcement

| Capacidade | Claude Code | Gemini CLI | Codex | Copilot |
|---|---|---|---|---|
| Carga base automatica | hook PreToolUse | procedural | procedural | procedural |
| Protecao de governanca | hook PostToolUse | procedural | procedural | procedural |
| Skills pre-carregadas | sim (symlinks) | sim (commands) | nao | sim (agents) |
| Enforcement programatico | sim (hooks) | nao | nao | nao |
| Validacao de evidencias | script | procedural | procedural | procedural |

Ferramentas sem enforcement programatico dependem do modelo seguir instrucoes procedurais. A compliance nessas ferramentas e best-effort.

## Economia de Contexto

Carregar o minimo necessario para a tarefa reduz custo de tokens em 35-50%:

| Complexidade | Criterio | O que carregar |
|---|---|---|
| `trivial` | Rename, typo, import, formatacao | Apenas AGENTS.md |
| `standard` | Bug fix, novo metodo, refactor local | AGENTS.md + TL;DR das references afetadas |
| `complex` | Nova feature, interface publica, migracao | AGENTS.md + referencias completas |

- Classificar a complexidade **antes** de carregar qualquer referencia.
- Quando a reference tiver bloco `<!-- TL;DR ... -->`, preferir o TL;DR ao documento completo em tarefas standard.
- Override explicito via `--complexity=<nivel>` prevalece sobre classificacao automatica.

## Validacao

Antes de concluir uma alteracao:

Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md` como base canonica.

Comandos detectados no projeto (Python):
1. Rodar fmt: `ruff format .`.
2. Rodar test: `pytest`.
3. Rodar lint: `ruff check .`.

## Restricoes

1. Nao inventar contexto ausente.
2. Nao assumir versao de linguagem, framework ou runtime sem verificar.
3. Nao alterar comportamento publico sem deixar isso explicito.
4. Nao usar exemplos como copia cega; adaptar ao contexto real.

5. Nao alterar contratos entre workspaces sem deixar o impacto explicito.

### Controle de profundidade de invocacao

- Skills que invocam outros skills (execute-task, refactor) devem verificar profundidade via `scripts/lib/check-invocation-depth.sh`.
- Limite padrao: 2 niveis. Configuravel via `AI_INVOCATION_MAX`.
- Variaveis de ambiente: `AI_INVOCATION_DEPTH` (corrente), `AI_INVOCATION_MAX` (limite).
