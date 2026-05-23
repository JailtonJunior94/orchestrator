# Tarefa 9.0: Documentação F1 cross-cutting

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Atualizar documentação cross-cutting da Fase 1: reescrever `COPILOT.md` raiz com a seção "Modo Recomendado (2026): Copilot via ACP" primeiro; atualizar `AGENTS.md` com linha ADR-012 e marcando ADR-007 como substituída; ajustar header de `docs/adr/007-copilot-cli-stateless-workaround.md`; documentar cobertura Copilot em `docs/telemetry-feedback-cycle.md`; validar enum em `docs/cli-schema.json`.

Esta tarefa é apenas documentação — sem código Go. Cobertura: RF-13, RF-14, RF-15, RF-16, RF-19, RF-20 do PRD.

<requirements>
- COPILOT.md reescrito: "Modo Recomendado (2026): Copilot via ACP" como primeira seção; "Modo Legado" marcada deprecated com timeline.
- AGENTS.md: linha ADR-012 na tabela; ADR-007 marcada como substituída por ADR-012.
- docs/adr/007-*.md: header ganha "**Status:** Substituída por ADR-012".
- docs/telemetry-feedback-cycle.md: documenta que invariantes cobrem tool=copilot.
- docs/cli-schema.json: validar enum --tool/--runtime; atualizar se necessário.
- Conteúdo histórico de ADR-007 preservado (não deletado).
</requirements>

## Subtarefas

- [ ] 9.1 Reescrever `COPILOT.md` raiz seguindo o template em `docs/research/compozy-adaptation-copilot-2026.md` §"Exemplos de Configuração 2026" — `COPILOT.md` raiz — reescrita F1.
- [ ] 9.2 Atualizar `AGENTS.md`: adicionar linha ADR-012 na tabela de ADRs; marcar ADR-007 como substituída na mesma tabela (ou em coluna de status).
- [ ] 9.3 Atualizar header de `docs/adr/007-copilot-cli-stateless-workaround.md`: trocar `**Status:** Aceita` por `**Status:** Substituída por ADR-012`. Conteúdo do corpo permanece intacto (preservação histórica).
- [ ] 9.4 Atualizar `docs/telemetry-feedback-cycle.md`: adicionar nota documentando que `runtime_init` ganha cardinalidade `tool=copilot` quando `--runtime=acp --tool=copilot`.
- [ ] 9.5 Validar `docs/cli-schema.json`: se enum `--tool` ou `--runtime` precisa atualização, ajustar. Caso o schema atual já cubra "copilot" sob "acp" implicitamente (sem enum estrito), nenhuma mudança é necessária — documentar a decisão no commit.
- [ ] 9.6 Lint manual de links: rodar `grep -r "ADR-012\|ADR-007" COPILOT.md AGENTS.md docs/` e validar que todos os paths apontam para arquivos existentes.

## Detalhes de Implementação

Ver `techspec.md` §"Arquivos Relevantes e Dependentes" → seção `Modificados` (documentação). Insumo principal: `docs/research/compozy-adaptation-copilot-2026.md` §"Exemplos de Configuração 2026" tem o texto base do `COPILOT.md` reescrito.

Anti-padrão: NÃO deletar ADR-007. Marcar como substituída e preservar conteúdo. NÃO duplicar conteúdo de ADR-012 em `COPILOT.md` — apenas referenciar.

## Critérios de Sucesso

- `COPILOT.md` primeira seção é "Modo Recomendado (2026): Copilot via ACP" com exemplo `--tool copilot --runtime acp`.
- `COPILOT.md` documenta pré-requisitos: `copilot --version` >= `CopilotMinCLIVersion`, `gh auth status` válido.
- `COPILOT.md` referencia ADR-012, ADR-007 (substituída), ADR-009, ADR-008.
- `AGENTS.md` tabela de ADRs contém ADR-012; ADR-007 marcada substituída.
- `docs/adr/007-*.md` header `**Status:**` atualizado.
- `docs/telemetry-feedback-cycle.md` menciona cobertura Copilot via ACP.
- `docs/cli-schema.json` validado e/ou atualizado conforme decisão registrada no commit.
- Todos os links cruzados resolvem para arquivos existentes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Lint manual: `grep -r "012-copilot-cli-acp-native.md\|007-copilot-cli-stateless-workaround.md" COPILOT.md AGENTS.md docs/` deve mostrar referências válidas em pelo menos `COPILOT.md` e `AGENTS.md`.
- [ ] Verificar via shell que cada link relativo aponta para arquivo existente (`ls` em cada path mencionado).
- [ ] `docs/cli-schema.json` continua sendo JSON válido (`jq . docs/cli-schema.json > /dev/null`).
- [ ] Se for usado `markdownlint` no projeto, rodar e confirmar conformidade.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `COPILOT.md` reescrito com "Modo Recomendado (2026): Copilot via ACP" primeiro.
- [ ] `COPILOT.md` referencia ADR-012, ADR-007 (substituída), ADR-009, ADR-008.
- [ ] `AGENTS.md` tabela de ADRs atualizada com ADR-012; ADR-007 marcada substituída.
- [ ] `docs/adr/007-copilot-cli-stateless-workaround.md` ganhou `**Status:** Substituída por ADR-012` no header.
- [ ] Conteúdo histórico de ADR-007 preservado (corpo intacto).
- [ ] `docs/telemetry-feedback-cycle.md` documenta cobertura Copilot.
- [ ] `docs/cli-schema.json` validado/atualizado; decisão registrada no commit.
- [ ] Todos os links cruzados validam contra arquivos existentes.
- [ ] Nenhum código Go tocado nesta tarefa.

## Arquivos Relevantes

- `COPILOT.md` (reescrever)
- `AGENTS.md` (modificar — tabela de ADRs)
- `docs/adr/007-copilot-cli-stateless-workaround.md` (atualizar header `**Status:**`)
- `docs/telemetry-feedback-cycle.md` (estender)
- `docs/cli-schema.json` (validar/atualizar)
- `docs/research/compozy-adaptation-copilot-2026.md` (insumo — texto base do `COPILOT.md`)
- `.specs/adr/012-copilot-cli-acp-native.md` (referência principal)
