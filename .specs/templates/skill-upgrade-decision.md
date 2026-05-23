# Decisão de Upgrade — <nome-da-skill>

## Metadados

- **Skill:** <nome>
- **Versão anterior (hash):** <sha256-anterior>
- **Versão nova (hash):** <sha256-novo>
- **Data:** <YYYY-MM-DD>
- **Responsável:** <nome ou handle>

## Motivador

<!-- Por que atualizar agora? Bug fix, nova feature, breaking change? -->

## Critério de Aceitação

<!-- Como saber que o upgrade foi bem-sucedido? -->
<!-- Ex: make test passa, comportamento X preservado, novo comportamento Y disponível -->

## Riscos

<!-- Alguma quebra de compatibilidade? Dependência de nova convenção? -->

## Resultado

- [ ] `skills-lock.json` atualizado com novo hash
- [ ] `make test && make integration` passam
- [ ] Registro salvo em `audit/skill-upgrade-<nome>-<data>.md`
