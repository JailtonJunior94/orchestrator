# Tarefa 1.0: Validadores canônicos em `.agents/scripts/` + resolução em cascata

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Tornar `.agents/scripts/` a fonte de verdade dos validadores de evidência, com mirror em
`.claude/scripts/` e embedded, e fazer todas as skills resolverem o validador em cascata
`.agents/scripts/` → `.claude/scripts/` → `scripts/`. Base para o gate de aceite (2.0) e os
hooks (4.0). Ver ADR-001.

<requirements>
- `.agents/scripts/{validate-task-evidence.sh,validate-bugfix-evidence.sh,validate-refactor-evidence.sh}` como canônico.
- `.claude/scripts/` e `internal/embedded/assets/.claude/scripts/` viram mirrors gerados.
- `scripts/sync-skills.sh` espelha `.agents/scripts/` para os mirrors.
- `bugfix/SKILL.md` deixa de usar path rígido `.claude/scripts/` e passa a resolver em cascata.
- Nenhum caller pode pular validação em silêncio — falha explícita se nenhum path resolver.
</requirements>

## Subtarefas

- [ ] 1.1 Mover validadores para `.agents/scripts/`; manter `.claude/scripts/` como mirror.
- [ ] 1.2 Estender `scripts/sync-skills.sh` para sincronizar `.agents/scripts/` → mirrors.
- [ ] 1.3 Atualizar `bugfix/SKILL.md` Etapa 5.5 para cascata `.agents/scripts/` → `.claude/scripts/` → `scripts/`.
- [ ] 1.4 Verificar/uniformizar a cascata em `execute-task`/`execute-all-tasks`.

## Detalhes de Implementação

Ver techspec.md seções "Cascata de validadores (RF-09, RF-17)" e ADR-001. Helper `_resolve_validator`
espelha o padrão de `check-invocation-depth.sh` (`.agents/lib/` → `scripts/lib/`).

## Critérios de Sucesso

- Os 3 validadores existem em `.agents/scripts/` e são byte-idênticos aos mirrors.
- `scripts/sync-skills.sh` sincroniza `.agents/scripts/` sem drift.
- `bugfix` resolve o validador em repo que contém apenas `.agents/` (sem `.claude/`).
- Nenhuma skill referencia path rígido `.claude/scripts/` para validadores.

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Teste shell: cascata resolve em repo só-`.agents/` (fixture em `t.TempDir()`/testdata).
- [ ] `make check-skills-sync` (e novo gate quando criado em 4.0) verde.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.agents/scripts/validate-task-evidence.sh`, `.agents/scripts/validate-bugfix-evidence.sh`, `.agents/scripts/validate-refactor-evidence.sh`
- `.claude/scripts/*` (mirror), `internal/embedded/assets/.claude/scripts/*`
- `scripts/sync-skills.sh`, `.agents/skills/bugfix/SKILL.md`, `.agents/skills/execute-task/SKILL.md`
