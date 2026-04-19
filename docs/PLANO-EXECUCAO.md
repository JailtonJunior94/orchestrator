# Plano de Execucao — Evolucao do ai-spec-harness

**Baseado em:** `relatorio-maturidade-19-04-26-16:24.md`
**Data:** 2026-04-19
**Objetivo:** Elevar maturidade do harness nos eixos robustez, economia de tokens, eficiencia operacional e spec-driven.

---

## Visao Geral das Frentes

| # | Frente | Prioridade | Impacto | Tarefas |
|---|--------|-----------|---------|---------|
| 1 | Economia de Tokens | Alta | -25-40% tokens/sessao | [001](tasks/001-claude-md-raiz.md), [002](tasks/002-preload-md.md), [003](tasks/003-indice-referencias.md) |
| 2 | Interface Padrao de Build | Alta | Normaliza agentes | [004](tasks/004-makefile.md) |
| 3 | Robustez e Testes | Media-Alta | +5 pacotes cobertos | [005](tasks/005-smoke-test-binario.md), [006](tasks/006-testes-doctor-inspect.md), [007](tasks/007-testes-python.md), [008](tasks/008-e2e-integration.md) |
| 4 | Spec-Driven | Media | Formaliza contratos | [009](tasks/009-schema-cli.md) |
| 5 | Dogfooding | Media | Valida o proprio harness | [010](tasks/010-dogfooding.md) |
| 6 | Commits e WIP Pendente | Alta | Consolida trabalho atual | [011](tasks/011-consolidar-wip.md) |

---

## Ordem de Execucao Recomendada

### Fase 1 — Fundacao (sem dependencias)
Tarefas que podem ser executadas em paralelo:

- [ ] **001** — Criar CLAUDE.md na raiz
- [ ] **004** — Criar Makefile com targets padrao
- [ ] **011** — Consolidar WIP (commit do trabalho pendente)

### Fase 2 — Economia de Tokens
Depende de 001 para alinhar formato:

- [ ] **002** — Extrair bloco de carga base para PRELOAD.md
- [ ] **003** — Criar indice resumido de referencias em skills

### Fase 3 — Robustez
Depende de 004 (Makefile) para targets de teste:

- [ ] **005** — Smoke test do binario no CI
- [ ] **006** — Testes para doctor e inspect
- [ ] **007** — Testes para scripts Python embarcados
- [ ] **008** — Integrar suite E2E existente ao CI

### Fase 4 — Formalizacao
Depende de Fase 1-3 estabilizadas:

- [ ] **009** — Schema JSON / COMMANDS.md dos comandos CLI
- [ ] **010** — Dogfooding: aplicar harness ao proprio repositorio

---

## Metricas de Sucesso

| Metrica | Antes | Meta |
|---------|-------|------|
| Pacotes com testes | 28/33 | 33/33 |
| Tokens bootstrap (estimado) | ~3000-5000 | ~2000-3500 |
| Smoke test binario | Nao | Sim |
| Makefile/Taskfile | Nao | Sim |
| CLAUDE.md raiz | Nao | Sim |
| Harness aplicado a si mesmo | Nao | Sim |

---

## Arquivos de Tarefas

Cada tarefa tem seu proprio arquivo em `docs/tasks/` com:
- Descricao detalhada
- Criterios de aceitacao
- Subtarefas com checklist
- Dependencias
- Estimativa de complexidade
