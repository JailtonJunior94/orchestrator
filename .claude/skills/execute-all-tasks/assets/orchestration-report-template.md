# Relatório de Orquestração de PRD

## PRD
- Slug: <slug>
- Diretório: tasks/prd-<slug>/
- PRD: tasks/prd-<slug>/prd.md
- TechSpec: tasks/prd-<slug>/techspec.md
- Tasks: tasks/prd-<slug>/tasks.md

## Resultado Final
- Status do orquestrador: done | partial | failed | needs_input
- Total de tarefas no PRD: <N>
- Tarefas done: <N>
- Tarefas pending: <N>
- Tarefas blocked: <N>
- Tarefas failed: <N>
- Tarefas needs_input: <N>

## Snapshot Inicial vs Final
| # | Título | Status inicial | Status final |
|---|--------|----------------|--------------|
| <id> | <título> | <status_inicial> | <status_final> |

## Tarefas Executadas Nesta Sessão
| # | Título | Status | Report Path | Summary |
|---|--------|--------|-------------|---------|
| <id> | <título> | <status> | tasks/prd-<slug>/<id>_execution_report.md | <1-line summary do subagent> |

## Tarefas Puladas (já estavam done)
- <id>: <título>

## Waves Executadas
| # | Modo | Tarefas | Início (UTC) | Fim (UTC) |
|---|------|---------|--------------|-----------|
| 1 | sequencial \| paralelo | <ids> | <ts> | <ts> |

## Próximos Passos
- [recomendação acionável: ex.: "Tarefa 2.0 retornou blocked por X — revisar techspec seção Y"]

## Suposições
- [suposição feita pelo orquestrador, ex.: "Tarefa 4.0 foi inferida do padrão task-4.0-*.md"]

## Riscos Residuais
- [risco, ex.: "Subagent paralelo da tarefa 3.0 ainda em voo quando halt foi disparado"]
