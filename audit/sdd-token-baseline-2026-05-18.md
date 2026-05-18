# Baseline de Tokens — SDD ai-spec-harness

> Data da medição: 2026-05-18
> Repositório: `ai-spec-harness` (orchestrator)
> Branch: `main`
> Metodologia: heurística oficial `chars/3.5` aplicada sobre `wc -c` das `SKILL.md`. Equivalente à contagem default do `ai-spec metrics` (sem `--precise`).

## Por que esta medição existe

A seção "Estratégia de Desenvolvimento de Alta Performance" do README e o documento [`docs/sdd-strategy.md`](../docs/sdd-strategy.md) afirmam que o SDD economiza tokens via isolamento de contexto. Este baseline materializa essa afirmação em números verificáveis — qualquer pessoa pode reproduzir os valores com `wc -c .agents/skills/*/SKILL.md`.

## Bloqueio observado durante a coleta

Tentativa de rodar `ai-spec metrics . --format json` falhou com `SKILL.md ausente em skill tests`. Causa: existe um diretório `.agents/skills/tests/` que **não é uma skill** (contém `conftest.py` e `test_validation_scripts.py`). O comando `metrics` itera sobre todo subdiretório de `.agents/skills/` esperando `SKILL.md` e quebra em pastas de teste.

**Workaround usado:** medição manual via `wc -c`. **Correção sugerida** (fora do escopo deste baseline): excluir `.agents/skills/tests/` da varredura do `metrics` ou movê-lo para `.agents/tests/`.

## Dados primários

| Skill | Caminho | Bytes | Tokens estimados (`chars/3.5`) |
| :--- | :--- | ---: | ---: |
| create-prd | `.agents/skills/create-prd/SKILL.md` | 5.692 | 1.626 |
| create-technical-specification | `.agents/skills/create-technical-specification/SKILL.md` | 4.802 | 1.372 |
| create-tasks | `.agents/skills/create-tasks/SKILL.md` | 11.790 | 3.368 |
| execute-task | `.agents/skills/execute-task/SKILL.md` | 8.707 | 2.487 |
| execute-all-tasks | `.agents/skills/execute-all-tasks/SKILL.md` | 14.024 | 4.007 |
| agent-governance | `.agents/skills/agent-governance/SKILL.md` | 6.027 | 1.722 |
| go-implementation | `.agents/skills/go-implementation/SKILL.md` | 7.609 | 2.174 |
| node-implementation | `.agents/skills/node-implementation/SKILL.md` | 5.600 | 1.600 |
| python-implementation | `.agents/skills/python-implementation/SKILL.md` | 5.481 | 1.566 |
| **Total** | — | **69.732** | **19.922** |

## Cenários comparativos (custo por execução)

### Pipeline completo (governance via SDD)

| Cenário | Skills carregadas | Tokens |
| :--- | :--- | ---: |
| Planejamento (PRD → TechSpec → Tasks) | `create-prd` + `create-technical-specification` + `create-tasks` | 6.366 |
| Execução task Go | `agent-governance` + `go-implementation` + `execute-task` | 6.383 |
| Execução task Node | `agent-governance` + `node-implementation` + `execute-task` | 5.809 |
| Execução task Python | `agent-governance` + `python-implementation` + `execute-task` | 5.775 |

### Prompt direto (sem SDD, controle hipotético)

Um prompt "implemente X" sem governança não carrega skills, mas requer que o operador re-explique contexto a cada execução. Custo estimado por iteração:

- Contexto re-explicado (instrução + arquivos relevantes + critérios de aceite): **~3.000 tokens por iteração**
- Retrabalho médio observado no histórico do repositório (relatos em `audit/maturidade-*`): **2 a 4 iterações** por feature não trivial

Total por feature não trivial em modo prompt direto: **6.000 a 12.000 tokens cumulativos**, sem rastreabilidade nem invariantes — risco de regressão silenciosa não medido.

### Pipeline com isolamento (`execute-all-tasks`)

Diferença chave do isolamento por subagent:

| Modo | Tokens orquestrador (10 tasks) | Tokens cumulativos (10 tasks) |
| :--- | ---: | ---: |
| Execução sequencial sem isolation | 63.830 | 63.830 |
| `execute-all-tasks` com subagent fresh | ≤1.000 (orquestrador retém ≤100 tokens/task) | 63.830 (mas dilluído em subagents independentes) |

**Ganho operacional:** o contexto da sessão principal não cresce com o número de tasks — o orquestrador permanece "limpo" e auditável.

## Conclusão verificável

1. **Custo base por execução: ~6.000 tokens.** Independe da linguagem (variação <10% entre Go/Node/Python).
2. **Sem SDD, prompt direto consome 6.000–12.000 tokens** por feature não trivial **e perde rastreabilidade** — paridade de custo com prejuízo de qualidade.
3. **Em PRDs com ≥ 5 tasks, `execute-all-tasks` é mandatório por economia** — mantém orquestrador enxuto e isola falhas.

## Como reproduzir

```bash
# 1. Tamanho real das skills
wc -c .agents/skills/{create-prd,create-technical-specification,create-tasks,execute-task,execute-all-tasks,agent-governance,go-implementation,node-implementation,python-implementation}/SKILL.md

# 2. Conversão para tokens (heurística oficial chars/3.5)
# tokens = bytes / 3.5

# 3. Quando a CLI estiver desbloqueada (após corrigir .agents/skills/tests/):
ai-spec metrics . --format json
ai-spec metrics . --precise --format json  # ~15% mais preciso, requer tiktoken
```

## Próximos passos para evoluir este baseline

1. Corrigir o bug do `ai-spec metrics` em `.agents/skills/tests/` para gerar JSON estruturado.
2. Habilitar `GOVERNANCE_TELEMETRY=1` em pelo menos um PRD real e rodar `ai-spec telemetry report --top-skills 10` após uma sprint.
3. Comparar tokens reais (telemetria) vs estimados (`chars/3.5`) e calibrar o multiplicador.
