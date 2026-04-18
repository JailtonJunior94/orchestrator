# Regras de Base Branch

Use estas regras apenas quando `scripts/resolve_pr_base.py` não devolver uma resposta forte o suficiente por conta própria.

## Ordem de Confiança
1. Base branch explicitamente pedida pelo usuário.
2. Base branch da PR já aberta para a mesma head branch.
3. Upstream ou configuração remota que aponte uma base clara.
4. Convenção do repositório observada no nome da branch.
5. Branch padrão do remoto como fallback conservador.

## Convenções Comuns

### Branches de feature
Padrões frequentes:
- `feat/*`
- `feature/*`
- `fix/*`
- `bugfix/*`
- `refactor/*`
- `chore/*`
- `docs/*`
- `test/*`

Base preferencial em fluxo Git Flow:
- `develop`

Base preferencial em fluxo trunk-based:
- `main`

### Branches de release
Padrões frequentes:
- `release/*`
- `release-candidate`

Base comum:
- `main`

### Branches de hotfix
Padrões frequentes:
- `hotfix/*`

Base comum:
- `main`

## Sinais para Não Confiar na Inferência
- o nome da branch não segue padrão conhecido
- o repositório não usa `develop`
- há múltiplas branches candidatas com merge-base plausível
- a PR existente aponta para uma base diferente da inferida por nome

## Regra de Segurança
Se dois candidatos parecerem igualmente plausíveis e nenhuma evidência objetiva desempatar, pedir confirmação do usuário.
