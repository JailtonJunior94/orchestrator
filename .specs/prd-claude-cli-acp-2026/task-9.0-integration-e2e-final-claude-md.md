# Tarefa 9.0: Integration E2E cross-wave + CLAUDE.md final + smoke F5

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Gate cross-wave final. Validar que F2..F5 funcionam integrados em uma única sessão (MCP nested + normalize + memory + hooks + cache metrics + auto-review). Atualizar `CLAUDE.md` raiz com a §"Runtime Capabilities" completa (todos os 5 capabilities listados). Smoke E2E F5 com seção combinada. Confirmar cobertura ≥ 70% global, ≥ 80% em subpacotes novos.

<requirements>
- Adicionar T-INT-06 em `tests/integration/claude_2026_e2e_test.go`: cenário cross-wave que ativa **todas** as flags (`--mcp-nested --auto-review --memory-workflow-limit-lines 100`) em uma sessão; validar:
  - `events.jsonl` tem eventos `nested_agent` (F2)
  - `events.jsonl` tem `normalized_name` (F2)
  - `execution_report.md` cita "Memory Compaction Requested" se memória atingiu limite (F3)
  - `execution_report.md` tem seção "Métricas Claude-2026" (F4)
  - `execution_report.md` tem seção "Review Block" se review bloqueou (F5)
- Atualizar `CLAUDE.md` raiz com §"Runtime Capabilities" **completa** (todos os 5 capabilities; ver techspec §"Exemplos de Configuração 2026" para texto base)
- Atualizar `CLAUDE.md` raiz com nota de precedência: memória do harness vence sobre auto-memory de Claude Code quando `.specs/<prd>/memory/` existe
- Atualizar `CLAUDE.md` raiz com nota: shell hooks em `.claude/hooks/*.sh` continuam servindo modo interativo; Go hooks servem modo orquestrado (ACPRunner)
- Atualizar `CHANGELOG.md` com entries por wave (F2, F3, F4, F5) usando skill `semantic-commit` para compor mensagens
- Rodar `make test && make integration && make parity` — todos verdes
- Cobertura agregada ≥ 70%; `internal/runtime/mcpserver/`, `memory/`, `hooks/` ≥ 80%
- Smoke manual final documentado em `execution_report.md`:
  ```bash
  ai-spec task-loop --tool claude --runtime acp \
    --mcp-nested --auto-review \
    --memory-workflow-limit-lines 100 \
    .specs/<prd-smoke>
  ```
  + verificação dos artefatos gerados
- Validar 31 invariantes (29 originais ADR-008 + INV-30 + INV-31) verdes
</requirements>

## Subtarefas

- [ ] 9.1 Implementar T-INT-06 (cross-wave) em `tests/integration/claude_2026_e2e_test.go`
- [ ] 9.2 Atualizar `CLAUDE.md` raiz com §"Runtime Capabilities" completa (~25 linhas)
- [ ] 9.3 Adicionar §"Precedência de Memória" em `CLAUDE.md` (precedence harness vs auto-memory Claude Code)
- [ ] 9.4 Adicionar §"Hooks: Shell vs Go" em `CLAUDE.md` (precedência modo interativo vs ACP)
- [ ] 9.5 Atualizar `CHANGELOG.md` com entries `feat(claude-2026): F2 — MCP nested + normalize`, `feat(claude-2026): F3 — memory + hooks`, etc.
- [ ] 9.6 Rodar `make test && make integration && make parity` — capturar output em `execution_report.md`
- [ ] 9.7 Rodar `go test ./internal/runtime/mcpserver/... ./internal/runtime/memory/... ./internal/runtime/hooks/... -coverprofile=cov.out` e validar ≥ 80%
- [ ] 9.8 Smoke manual cross-wave: comando documentado nos requirements + checklist de artefatos verificados
- [ ] 9.9 Atualizar `tasks.md` deste PRD com `Status: done` em todas as 9 tasks (sequencial via execute-all-tasks ou manual ao final)
- [ ] 9.10 Confirmar `ai-spec check-spec-drift .specs/prd-claude-cli-acp-2026/tasks.md` retorna sem drift (PRD/TechSpec hashes batem)

## Detalhes de Implementação

Ver `techspec.md` §"Verificação Final (pré-merge de cada wave)" para checklist canônico. **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- **T-INT-06 é o invariante de integração** — se falhar, alguma task anterior tem regressão escondida. Investigar antes de tentar paliativo.
- **`CLAUDE.md` raiz é a documentação canônica** que agentes futuros vão consumir. Manter conciso (não duplicar techspec); apontar para arquivos relevantes.
- **`CHANGELOG.md` segue Conventional Commits** (ADR documentado em algum lugar do repo) — usar skill `semantic-commit` para gerar mensagens corretas.
- **Cobertura ≥ 80% em subpacotes novos é hard requirement** — se algum dos três (`mcpserver`, `memory`, `hooks`) ficar abaixo, identificar quais arquivos faltam testes e adicionar antes de fechar.
- **Não fechar como `done` se qualquer make falhar** — investigar root cause; reabrir task específica que introduziu a regressão.
- **Spec-drift check no final** garante que PRD/TechSpec não mudaram desde o início da implementação — caso tenham mudado, `ai-spec sync-spec-hash` atualiza os comentários no cabeçalho.

## Critérios de Sucesso

- T-INT-06 verde (validação cross-wave)
- `make test` verde com cobertura ≥ 70% global
- `make integration` verde
- `make parity` reporta 31 invariantes verdes
- Cobertura `mcpserver/`, `memory/`, `hooks/` ≥ 80%
- `CLAUDE.md` raiz tem §"Runtime Capabilities" completa cobrindo todos os 5 capabilities
- `CHANGELOG.md` tem 4 entries de wave (F2, F3, F4, F5)
- Smoke manual cross-wave: todos os 5 artefatos esperados presentes no `execution_report.md` da sessão
- `ai-spec check-spec-drift .specs/prd-claude-cli-acp-2026/tasks.md` retorna sem drift
- `tasks.md` deste PRD: todas as 9 tasks com Status `done`

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes de integração: T-INT-06 (cross-wave) em `tests/integration/claude_2026_e2e_test.go`
- [ ] Validação manual: `make test && make integration && make parity` outputs salvos em `execution_report.md`
- [ ] Validação manual: smoke cross-wave com `--mcp-nested --auto-review` + checklist de artefatos
- [ ] `ai-spec check-spec-drift` sem drift
- [ ] Cobertura agregada e por subpacote conforme requirements

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Modificado** `CLAUDE.md` (raiz) — §"Runtime Capabilities" completa + §"Precedência de Memória" + §"Hooks: Shell vs Go"
- **Modificado** `CHANGELOG.md` — 4 entries (F2, F3, F4, F5)
- **Modificado** `tests/integration/claude_2026_e2e_test.go` (+T-INT-06)
- **Modificado** `.specs/prd-claude-cli-acp-2026/tasks.md` — status `done` em todas as linhas
- **Leitor:** todas as tasks anteriores (1.0..8.0) — gate de integração não pode mudar internals
