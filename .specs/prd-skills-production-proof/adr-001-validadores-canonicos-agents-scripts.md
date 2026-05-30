# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Validadores canônicos em `.agents/scripts/` com resolução em cascata
- **Data:** 2026-05-30
- **Status:** Proposta
- **Decisores:** Mantenedor do harness
- **Relacionados:** `.specs/prd-skills-production-proof/prd.md` (RF-09, RF-13, RF-17),
  `.specs/prd-skills-production-proof/techspec.md`, ADR-002 (deste PRD)

## Contexto

Os validadores de evidência (`validate-task-evidence.sh`, `validate-bugfix-evidence.sh`)
vivem hoje apenas em `.claude/scripts/` (+ mirror embedded). Skills resolvem o path de forma
inconsistente: `execute-task` já tenta cascata `.claude/scripts/` → `.agents/scripts/` →
`scripts/`, mas `bugfix` usa path rígido `.claude/scripts/`. Em um repositório que copia
apenas `.agents/` (cenário-alvo de portabilidade), os validadores somem e o enforcement é
pulado silenciosamente. O padrão de libs (`.agents/lib/` canônico → `scripts/lib/` mirror) já
resolve esse problema para `check-invocation-depth.sh` e deve ser estendido aos validadores.

## Decisão

Tornar `.agents/scripts/` a **fonte de verdade** dos validadores de evidência, espelhada para
`.claude/scripts/` e `internal/embedded/assets/.claude/scripts/` via `sync-skills.sh`, com gate
de drift `check-scripts-sync.sh`. Todas as skills resolvem o validador em **cascata determinística**
`.agents/scripts/` → `.claude/scripts/` → `scripts/`, falhando explicitamente quando nenhum é
encontrado (nunca pular validação em silêncio).

## Alternativas Consideradas

- **Manter `.claude/scripts/` como canônico**: simples, mas quebra portabilidade em repos
  só-`.agents/` — rejeitada por contrariar o objetivo primário de funcionar em outros repos.
- **Duplicar validadores por-tool** (`.codex/scripts/`, `.gemini/scripts/`): redundância e
  risco de drift entre cópias — rejeitada; o ponto comum deve ser único.

## Consequências

### Benefícios Esperados

- Enforcement preservado em qualquer repositório que instale o harness.
- Um único validador por gate, chamado igualmente por todos os tools (base da ADR-002).
- Consistência com o padrão já validado de `.agents/lib/`.

### Trade-offs e Custos

- Mais um par canônico→mirror para manter sincronizado e cobrir por gate de CI.
- Callers que assumiam `.claude/scripts/` precisam migrar para a cascata.

### Riscos e Mitigações

- **Risco:** caller esquecido aponta para path antigo. **Mitigação:** manter `.claude/scripts/`
  como mirror funcional durante a transição + grep de callers no PR.
- **Rollback:** reverter a ordem de cascata e manter `.claude/scripts/` como canônico.

## Plano de Implementação

1. Mover validadores para `.agents/scripts/`; deixar `.claude/scripts/` como mirror gerado.
2. Estender `sync-skills.sh` para espelhar `.agents/scripts/`.
3. Criar `check-scripts-sync.sh` e alvo no `Makefile`.
4. Atualizar `bugfix`/`execute-task`/`execute-all-tasks` para a cascata.
5. Concluído quando o gate de sync passa e a cadeia resolve os validadores em repo só-`.agents/`.

## Monitoramento e Validação

- `make check-scripts-sync` verde; teste de portabilidade em `t.TempDir()` só-`.agents/`.

## Impacto em Documentação e Operação

- Atualizar `AGENTS.md`/`CLAUDE.md` (estrutura), `enforcement-matrix.md`, e os SKILL.md afetados.

## Revisão Futura

- Revisitar se um quinto tool exigir scripts específicos ou se o embedding mudar de layout.
