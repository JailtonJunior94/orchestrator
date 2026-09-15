# Decisão de Upgrade — re-ancoragem do hash de `domain-modeling-production`

## Metadados

- **Skill:** `domain-modeling-production`
- **Versão anterior (hash):** `bd043a8a4cdc1667ea936332c799f66aa7ae3090c6e733870f912107d8544151` (lock `version` 1.0.0)
- **Versão nova (hash):** `86da99bee0b2c0d8e741c3f9ebc8517159a61223906b9966d904222dcd9aec02` (lock `version` 1.0.0, sem mudança de versão — conteúdo compatível)
- **Data:** 2026-09-11
- **Responsável:** execução do PRD `harness-quatro-clis-loop-aprovacao` (ciclo review → bugfix → review), sob mandato explícito do dono do repositório para conformidade total do PRD

## Motivador

A execução das tarefas 7.0 ("OpenCode como agente oficial de primeira classe") e 10.0
("Remoção total do Gemini e desinstalação fiel") deste PRD exige, por RF-01 e RF-06, que toda
superfície do repositório que enumera o conjunto de agentes/CLIs oficiais reflita exatamente
`{claude, codex, copilot, opencode}`, sem `gemini`. `.agents/skills/domain-modeling-production/SKILL.md`
continha, numa diretiva `<critical>`, a enumeração antiga `Claude Code, Codex, Gemini e Copilot`.
Essa linha foi corrigida para `Claude Code, Codex, Copilot e OpenCode` durante a varredura de remoção
do Gemini, alterando o conteúdo do arquivo em 1 linha e, consequentemente, seu SHA-256 — sem que o
`computedHash` de `skills-lock.json` fosse re-ancorado no mesmo passo.

O gate `ai-spec skills --verify` (bloqueante, ADR-005) passou a reprovar a skill com "hash diverge do
registrado em skills-lock.json", detectado durante a validação final do ciclo de revisão do PRD.

Verificação: `git diff HEAD -- .agents/skills/domain-modeling-production/SKILL.md` mostra exatamente
1 linha alterada (a enumeração de agentes); nenhuma outra mudança de conteúdo. O conteúdo atual é o
conteúdo pretendido pelo PRD (RF-01/RF-06) — a correção é re-ancorar o hash, não reverter o conteúdo.

## Critério de Aceitação

- `ai-spec skills --verify` retorna exit 0 com "Integridade das skills OK (versao + hash SHA-256)."
- `computedHash` de `domain-modeling-production` no lock == `shasum -a 256 .agents/skills/domain-modeling-production/SKILL.md`.
- Demais skills do lock permanecem inalteradas.

## Riscos

Nenhum risco novo. A mudança de conteúdo é estritamente textual (nome de agente numa lista
enumerativa dentro de uma diretiva `<critical>`), sem alteração de comportamento procedural da
skill. Comportamento do gate é intencional e documentado (ADR-005): qualquer modificação de skill
requer re-ancoragem explícita e registrada.
