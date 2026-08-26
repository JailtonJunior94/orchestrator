# Decisão de Upgrade — bubbletea (remoção)

## Metadados

- **Skill:** bubbletea
- **Versão anterior (hash):** 4cf3387527d25a0ef4d07d0a6b01d041f18b91151f0528c5859040dbdb1f1a55
- **Versão nova (hash):** removida (sem entrada no lock)
- **Data:** 2026-05-30
- **Responsável:** JailtonJunior94

## Motivador

Remoção da skill bubbletea de todos os caminhos operacionais no commit `a34df86`
("fix(production-proof): corrige 14 achados cross-tool na causa raiz e remove bubbletea").
Registro criado retroativamente para restabelecer a conformidade com a regra de
`audit/` do AGENTS.md, violada quando o diretório foi removido e gitignored no mesmo ciclo.

## Critério de Aceitação

- `skills-lock.json` sem entrada `bubbletea` (verificado em `a34df86`).
- `.agents/skills/bubbletea/` removida (verificado em `a34df86`).
- Suite de testes verde após a remoção (evidência no corpo do commit `a34df86`).

## Riscos

Projetos que dependiam da skill bubbletea deixam de recebê-la em novas instalações.
Mitigado: a skill não era referenciada por nenhuma skill de governança restante.

## Resultado

- [x] `skills-lock.json` atualizado (entrada removida)
- [x] `make test && make integration` passam (evidência registrada no commit `a34df86`)
- [x] Registro salvo em `audit/skill-upgrade-bubbletea-2026-05-30.md`
