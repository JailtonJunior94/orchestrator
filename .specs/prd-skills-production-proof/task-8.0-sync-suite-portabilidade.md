# Tarefa 8.0: Sync mirrors + suíte + portabilidade cross-CLI + governança

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Consolidar a entrega: sincronizar todos os mirrors, rodar a suíte completa, validar comportamento
idêntico nos 4 CLIs e numa matriz de projetos (pequeno/médio/grande, novo/existente), e atualizar a
governança. Cobre RF-22, RF-23 + sync/validação/governança.

<requirements>
- Sync de skills/hooks/scripts; gates de sync verdes (`check-skills-sync`, `check-hooks-sync`, `check-scripts-sync`).
- `make test integration lint vet` verdes; cobertura ≥ 75%.
- Comportamento idêntico verificado nos 4 CLIs e na matriz de projetos (RF-23) sem acoplamento a paths do `orchestrator`.
- Governança atualizada (`AGENTS.md`/`CLAUDE.md`/`GEMINI.md`/`COPILOT.md`/`CODEX.md` na parte de hooks/validadores/estrutura) e índice de ADRs.
</requirements>

## Subtarefas

- [ ] 8.1 Rodar `scripts/sync-skills.sh` e `scripts/sync-hooks.sh`; corrigir drift.
- [ ] 8.2 `make check-skills-sync check-hooks-sync check-scripts-sync`.
- [ ] 8.3 `make test integration lint vet`; coverage ≥ 75%.
- [ ] 8.4 Teste de portabilidade: cadeia em repo só-`.agents/` + matriz de projetos (pequeno/médio/grande, novo/existente) convergindo idêntico nos 4 CLIs.
- [ ] 8.5 Atualizar governança e índice de ADRs.

## Detalhes de Implementação

Ver techspec "Testes E2E" (RF-22, RF-23). Usar `t.TempDir()` + build tag `integration` para
fixtures de projeto; descoberta sempre agnóstica (`ls .agents/skills/`, frontmatter, env exportadas).

## Critérios de Sucesso

- Os três gates de sync passam sem drift.
- `make test integration lint vet` verdes; cobertura ≥ 75%.
- Cadeia converge idêntica nos 4 CLIs e na matriz de projetos, sem dependência de paths do `orchestrator`.
- Governança e ADRs atualizados e consistentes.

## Skills Necessárias

- `analyze-project` — atualizar artefatos de governança (AGENTS/CLAUDE/GEMINI/COPILOT/CODEX) conforme a stack detectada.

## Testes da Tarefa

- [ ] `make test integration lint vet` + os três gates de sync.
- [ ] Evidência registrada da matriz de projetos e da convergência cross-CLI.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `scripts/sync-skills.sh`, `scripts/sync-hooks.sh`, `scripts/check-scripts-sync.sh`, `Makefile`
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `COPILOT.md`, `CODEX.md`
- `testdata/`, testes de integração
