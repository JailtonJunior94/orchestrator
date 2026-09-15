# Decisão de Upgrade — re-ancoragem de 3 hashes do skills-lock.json

## Metadados

- **Skills:** `design-patterns-mandatory`, `domain-modeling-production`, `semantic-commit`
- **Versão anterior (hash):**
  - `design-patterns-mandatory`: `c9e562aa50d5dcc4b5b6a71dc9c1571c4d08d85bb25a03f5a572e1494ca64ced` (sem `version` no lock)
  - `domain-modeling-production`: `0a548091f281fcb1a290a25ed8c19f6f5289ec21688c234332d9d04a03dab0cc` (sem `version` no lock)
  - `semantic-commit`: `ea289b301d4afae4f798a8a394eb74aaa943fd5990307fc0c86da07036167280` (lock `version` 1.0.0)
- **Versão nova (hash):**
  - `design-patterns-mandatory`: `9540189e448c1ee5ebccb75128dbca61bd26a7ad71ec042e2bc40f4422d204cd`
  - `domain-modeling-production`: `66827dd6ae276129fd55961d5e07be1b356828b3dbcaf28509aefc801ad9dd5b`
  - `semantic-commit`: `967598b88fc05a2c7f63651696adf6c49735d8af15fde9a359c34e09ff5351bf` (lock `version` → 1.1.0, alinhado ao frontmatter instalado)
- **Data:** 2026-09-10
- **Responsável:** Stefany Lima Teixeira (decisão), execução assistida por Claude Code

## Motivador

O commit `ba80b56` ("feat(skills): adiciona skills de design patterns e domain modeling")
registrou `design-patterns-mandatory` e `domain-modeling-production` no `skills-lock.json`
com `computedHash` que não corresponde ao conteúdo commitado de `.agents/skills/<nome>/SKILL.md`
(drift de conteúdo sem re-hash). Em paralelo, `semantic-commit` foi atualizado para o
frontmatter `version: 1.1.0` (bump compatível, não-breaking) sem re-ancoragem do hash nem
sincronização do campo `version` no lock.

O gate `ai-spec skills --verify` (bloqueante, ADR-005) reprovava as 3 skills por
"hash diverge do registrado em skills-lock.json", impedindo a Etapa 1 de
`execute-all-tasks` para o PRD `harness-quatro-clis-loop-aprovacao`
(a skill `domain-modeling-production` é exigida pelas tarefas 2.0 e 6.0 desse PRD).

Verificação: working tree limpo e idêntico ao HEAD; nenhuma modificação local não
registrada. As divergências são de estado commitado. O conteúdo atual das 3 SKILL.md
é o conteúdo pretendido — a correção é re-ancorar o hash, não reverter o conteúdo.

## Critério de Aceitação

- `ai-spec skills --verify` retorna exit 0 com "Integridade das skills OK (versao + hash SHA-256)." — **verificado**.
- `computedHash` de cada skill == `sha256sum .agents/skills/<nome>/SKILL.md` — **verificado**.
- Demais 11 skills do lock permanecem inalteradas.

## Riscos

Nenhum risco novo além do já documentado para o gate: skills modificadas localmente sem
registro passam a falhar em `skills --verify`. Comportamento intencional
(docs/troubleshooting.md).

## Adendo — `version` no frontmatter das 2 skills novas

`internal/skills/skill-frontmatter.schema.json` marca `version` como `required`, e o teste de
integração `TestExternalSkills_FrontmatterAndReferences` (`-tags=integration`) já falhava desde
`ba80b56` para `design-patterns-mandatory` e `domain-modeling-production` (frontmatter sem
`version`). Correção: adicionado `version: 1.0.0` ao frontmatter de ambas (primeira versão
rastreada). Hashes re-ancorados:

- `design-patterns-mandatory`: `23765fd14638cece494170ba1e957a44bd54d4c5ba464d01fc6239c8c534b2da`
- `domain-modeling-production`: `bd043a8a4cdc1667ea936332c799f66aa7ae3090c6e733870f912107d8544151`

`go test -tags=integration -run TestExternalSkills_FrontmatterAndReferences ./internal/integration/`
passa após a correção.

## Resultado

- [x] `skills-lock.json` atualizado com os hashes + `version` de `semantic-commit` → 1.1.0
- [x] `version: 1.0.0` adicionado a `design-patterns-mandatory` e `domain-modeling-production`; hashes re-ancorados; `version` registrado no lock
- [x] `ai-spec skills --verify` exit 0
- [x] `TestExternalSkills_FrontmatterAndReferences` verde
- [x] Registro salvo em `audit/skill-upgrade-lock-hash-resync-2026-09-10.md`
