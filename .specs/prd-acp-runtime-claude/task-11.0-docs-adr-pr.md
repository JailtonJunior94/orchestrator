# Tarefa 11.0: Docs + ADR Transitions + Final PR

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar a entrega: (a) atualizar `README.md` com seção "Runtime ACP (experimental)"; (b) gerar entrada no `CHANGELOG.md` resumindo os 10 commits anteriores; (c) atualizar `AGENTS.md` com links para ADR-009 e ADR-010; (d) **transitar status** dos ADR-009 e ADR-010 de `Proposta` para `Aceita` (RF-15); (e) abrir o PR único agrupando os 11 commits do branch `feat/acp-runtime-claude` no `main`. Usa as skills `finalize-changelog-readme-push` e `pull-request`.

<requirements>
- `README.md`: nova seção "Runtime ACP (experimental)" explicando `--runtime=acp`, requisitos (binário ou npx), exemplo de comando, link para ADR-009.
- `CHANGELOG.md`: entrada conforme convenção do repo, agrupando commits 1.0–10.0.
- `AGENTS.md`: adicionar ADR-009 e ADR-010 à lista de ADRs.
- ADR-009 e ADR-010: status muda de `Proposta` para `Aceita`; campo `Data` atualizado.
- PR criado com título, descrição estruturada e checklist de teste; referencia PRD, techspec e ambos os ADRs.
- Branch `feat/acp-runtime-claude` empurrado para origin.
- `make verify` passa após todas as edições.
</requirements>

## Subtarefas

### README + AGENTS

- [ ] 11.1 Adicionar em `README.md`, após a seção "Núcleo operacional" (ou equivalente), uma seção `## Runtime ACP (experimental)` cobrindo: o que é, como ativar, requisitos de instalação, exemplo `ai-spec task-loop --tool claude --runtime acp`, link para `.specs/adr/009-acp-protocol-adoption.md`.
- [ ] 11.2 Atualizar `AGENTS.md` adicionando ADR-009 e ADR-010 à lista de ADRs (se houver tal lista; caso contrário, criar subseção "ADRs locais por PRD").

### CHANGELOG

- [ ] 11.3 Usar a skill `finalize-changelog-readme-push` (declarada na coluna Skills) para gerar a entrada do `CHANGELOG.md`. A skill consome `git log` desde a última tag e produz a entrada estruturada. Tipo de versão: `minor` (feature nova opt-in, sem breaking change).

### ADR transitions

- [ ] 11.4 Em `.specs/adr/009-acp-protocol-adoption.md`, alterar `Status: Proposta` para `Status: Aceita`; atualizar `Data` para a data do merge.
- [ ] 11.5 Em `.specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md`, mesma alteração de status e data.

### Commit + PR

- [ ] 11.6 Commit semântico final agrupando docs + ADR transitions: `docs(acp-runtime): add README section, CHANGELOG entry and accept ADR-009/010`.
- [ ] 11.7 Validação final: `make verify` deve passar com tudo aplicado.
- [ ] 11.8 Usar a skill `pull-request` (declarada na coluna Skills) para abrir o PR do branch `feat/acp-runtime-claude` contra `main`, com título `feat: ACP runtime for Claude (RF-01–RF-16, ADR-009, ADR-010)` e descrição estruturada incluindo:
  - resumo executivo (1 parágrafo)
  - links para PRD, techspec, ADR-009, ADR-010
  - lista dos 11 commits com 1 linha cada
  - checklist de teste (unit + integration + live opt-in)
  - notas para revisor (manter `--runtime=legacy` como default; não promover default nesta entrega)
- [ ] 11.9 Publicar URL do PR no `execution_report.md` desta task.

## Detalhes de Implementação

Ver:
- `techspec.md` §"Plano de Rollout" (iterações 3 e 5)
- `prd.md` RF-15
- ADR-009 §"Plano de Implementação" item 1 (critério de transição para `Aceita`)
- `.agents/skills/finalize-changelog-readme-push/SKILL.md` para o fluxo da skill
- `.agents/skills/pull-request/SKILL.md` para template de PR

## Critérios de Sucesso

- `README.md` tem a nova seção, validada por presença de string `## Runtime ACP` via grep.
- `CHANGELOG.md` tem entrada nova no topo cobrindo 11 commits.
- `AGENTS.md` referencia ADR-009 e ADR-010.
- ADR-009 e ADR-010 ambos com `Status: Aceita`.
- `make verify` passa.
- PR aberto no GitHub com URL registrada.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

- `finalize-changelog-readme-push` — gera entrada no CHANGELOG.md, revisa README.md, prepara staging, cria commit semântico e publica no remoto.
- `pull-request` — abre o PR final com título, descrição estruturada e checklist de teste agrupando os 11 commits da branch.

## Testes da Tarefa

- [ ] `grep -n "## Runtime ACP" README.md` retorna ao menos uma linha
- [ ] `grep -n "ADR-009\|ADR-010" AGENTS.md` retorna duas linhas
- [ ] `grep -n "Status: Aceita" .specs/adr/009-acp-protocol-adoption.md .specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md` retorna duas linhas
- [ ] CHANGELOG.md tem entrada datada do merge contendo "ACP runtime" no título
- [ ] PR existe no GitHub (`gh pr view --web` ou URL no execution_report)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `README.md` (modificado)
- `CHANGELOG.md` (modificado pela skill)
- `AGENTS.md` (modificado)
- `.specs/adr/009-acp-protocol-adoption.md` (modificado: status)
- `.specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md` (modificado: status)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-11.0/execution_report.md`
- [ ] PR URL registrado no execution_report
- [ ] Ambos os ADRs com `Status: Aceita` (verificável por grep)
- [ ] CHANGELOG.md atualizado e commitado
- [ ] README.md atualizado e commitado
- [ ] Branch `feat/acp-runtime-claude` no origin com 12 commits (tasks 1–11)
- [ ] Commit semântico desta task: `docs(acp-runtime): finalize README, CHANGELOG, ADR-009/010 acceptance`
