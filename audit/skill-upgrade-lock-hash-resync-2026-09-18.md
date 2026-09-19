# Decisão de Upgrade — resincronização de `design-patterns-mandatory` e `domain-modeling-production`

## Metadados

- **Skill:** `design-patterns-mandatory`
- **Versão anterior (hash):** `23765fd14638cece494170ba1e957a44bd54d4c5ba464d01fc6239c8c534b2da` (lock `version` 1.0.0)
- **Versão nova (hash):** `f07361a8054e71f281ef1c555ddeeb9fb1c491fbf2f9056f58c0c8f0cb902846` (lock `version` 1.1.0)
- **Data:** 2026-09-18
- **Responsável:** execução da skill `execute-all-tasks` sobre o PRD `harness-portatil-vendor-neutral`, sob mandato explícito do dono do repositório após confirmação via pergunta direta ("Resincronizar o lock e registrar decisão em audit/")

- **Skill:** `domain-modeling-production`
- **Versão anterior (hash):** `86da99bee0b2c0d8e741c3f9ebc8517159a61223906b9966d904222dcd9aec02` (lock `version` 1.0.0)
- **Versão nova (hash):** `b4b067ee2d9fd055b43d059077b60ebee301c842d63ae63981db0caf9bdfdeb8` (lock `version` 2.0.0, breaking: major version bump)
- **Data:** 2026-09-18
- **Responsável:** idem acima

## Motivador

Ao iniciar a execução do PRD `harness-portatil-vendor-neutral` via `execute-all-tasks`, o gate
bloqueante `ai-spec skills --verify` (ADR-005) reprovou a pré-condição de integridade com:

```
[!!] design-patterns-mandatory: hash diverge do registrado em skills-lock.json
[!!] domain-modeling-production: breaking: major version bump
```

`ai-spec skills check . -v` confirmou que ambas as skills instaladas em `.agents/skills/` já
estavam à frente do `skills-lock.json`: `design-patterns-mandatory` em `v1.1.0` (bump compatível,
minor) e `domain-modeling-production` em `v2.0.0` (bump major, sinalizado como potencialmente
quebrador). Nenhuma das duas skills faz parte do diff desta sessão (`git status` não mostra
alteração local nos respectivos `SKILL.md`) — o drift é anterior a esta execução e não está
relacionado às tarefas do PRD em questão (que tratam de portabilidade multi-CLI do harness, não de
design patterns ou modelagem de domínio). O dono do repositório confirmou explicitamente resincronizar
o lock e registrar a decisão, em vez de investigar ou ignorar o gate, para destravar a execução das
13 tarefas do PRD sem alterar o conteúdo das skills.

## Critério de Aceitação

- `ai-spec skills --verify` retorna exit 0 com "Integridade das skills OK (versao + hash SHA-256)." — **atendido**.
- `computedHash` de cada skill no lock == `shasum -a 256 .agents/skills/<skill>/SKILL.md` — **atendido**.
- Demais entradas de `skills-lock.json` permanecem inalteradas — **atendido** (diff restrito às
  duas entradas acima).

## Riscos

`domain-modeling-production` teve bump de major version (1.0.0 → 2.0.0), sinalizado pelo próprio
`ai-spec skills check` como potencial breaking change de conteúdo/contrato da skill. Esta execução
não usa `domain-modeling-production` nem `design-patterns-mandatory` nas tarefas do PRD
`harness-portatil-vendor-neutral` (skills de linguagem/processo aplicáveis são as de Go e as
skills processuais declaradas em cada `tasks.md`), portanto o risco de regressão imediata nesta
execução é nulo. Fica registrado para quem depender dessas duas skills em outra sessão: revisar o
diff de conteúdo do `SKILL.md` correspondente antes de assumir compatibilidade total com uso
anterior.

## Resultado

- [x] `skills-lock.json` atualizado com novo hash e versão para as duas skills
- [x] `ai-spec skills --verify` passa (exit 0)
- [x] Registro salvo em `audit/skill-upgrade-lock-hash-resync-2026-09-18.md`
