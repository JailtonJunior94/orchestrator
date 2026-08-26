# Decisão de Upgrade — re-ancoragem dos 12 hashes do skills-lock.json

## Metadados

- **Skill:** todas as 12 skills externas rastreadas (confluence-changelog-publisher,
  finalize-changelog-readme-push, github-diff-changelog-publisher, github-pr-comment-triage,
  github-release-publication-flow, jira-tasks, otel-grafana-dashboards,
  postman-collection-generator, prompt-enricher, pull-request, semantic-commit, us-to-prd)
- **Versão anterior (hash):** hashes defasados (12/12 divergiam do conteúdo atual)
- **Versão nova (hash):** SHA-256 recalculado de cada `.agents/skills/<nome>/SKILL.md`
- **Data:** 2026-08-26
- **Responsável:** JailtonJunior94

## Motivador

Auditoria AI-ready identificou que os 12 `computedHash` do `skills-lock.json` estavam
defasados em relação ao conteúdo das skills (drift de conteúdo sem bump de versão) e que
nenhum código verificava o hash — a âncora de integridade da ADR-005 era cosmética.
Esta alteração re-ancora os hashes ao conteúdo atual e introduz o gate
`ai-spec skills --verify` que passa a verificar versão + hash (exit ≠ 0 em divergência).

## Critério de Aceitação

- `ai-spec skills --verify .` retorna exit 0 com "Integridade das skills OK".
- Adulteração manual de um SKILL.md é detectada com exit 1 e mensagem
  "hash diverge do registrado em skills-lock.json" (verificado em /tmp/verify-neg).
- `make test` verde, incluindo novos testes de hash em `internal/skillscheck`.

## Riscos

Projetos com skills modificadas localmente sem registro passam a falhar no gate
`skills --verify` invocado por execute-task/execute-all-tasks — comportamento
intencional do gate (documentado em docs/troubleshooting.md).

## Resultado

- [x] `skills-lock.json` atualizado com novos hashes
- [x] `make test` passa
- [x] Registro salvo em `audit/skill-upgrade-lock-hash-resync-2026-08-26.md`
