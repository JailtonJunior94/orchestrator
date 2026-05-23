# Tarefa 10.0: Documentação F1-Codex cross-cutting

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Atualizar quatro arquivos de documentação para refletir F1-Codex:

1. **`CODEX.md` raiz** (esqueleto atual de ~30 linhas → reescrita ~80-100 linhas): seção "Modo Recomendado (2026): Codex via ACP" primeira; seção "Modo Legado" deprecada; documentação explícita da confusão `codex` vs `codex-acp`; descrição das flags `--reasoning-effort` e `--access-mode`; warning sobre `--access-mode=full`; exemplo de comando completo; referência a ADR-013.

2. **`AGENTS.md`** (raiz): adicionar linha na tabela de ADRs apontando para ADR-013 ("Codex CLI as native ACP runtime").

3. **`docs/cli-schema.json`**: adicionar enums `--reasoning-effort` (`low|medium|high`) e `--access-mode` (`restricted|full`) ao schema das flags do `task-loop`.

4. **`docs/telemetry-feedback-cycle.md`**: documentar que invariantes Codex ACP cobrem os mesmos kinds que Claude/Copilot, incluindo `runtime_init` com `tool=codex`, `npm_version=0.14.0`, `sdk_version=v0.13.0`.

Paralelizável com tarefa 9.0 (sub-suite tests) — arquivos disjuntos.

Conteúdo de `CODEX.md` reescrito segue exemplo em `docs/research/compozy-adaptation-codex-2026.md` §"Exemplos de Configuração 2026 — `CODEX.md` raiz".

<requirements>
- CODEX.md tem §"Modo Recomendado (2026)" primeira, com comando, pré-condições, validações.
- CODEX.md tem §"Modo Legado" marcada deprecated + janela de 2 versões minor (ADR-013 D-05).
- CODEX.md documenta confusão codex vs codex-acp em §"Pré-requisitos" ou seção dedicada.
- CODEX.md descreve --reasoning-effort, --access-mode + warning sobre full.
- AGENTS.md tabela de ADRs ganha linha ADR-013.
- docs/cli-schema.json valida JSON syntax após adição dos enums.
- docs/telemetry-feedback-cycle.md menciona tool=codex.
- Skills sync drift gate (make check-skills-sync) continua passando após edição (CODEX.md é raiz, não skill).
</requirements>

## Subtarefas

- [ ] 10.1 Reescrever `CODEX.md` raiz (substituir conteúdo atual): cabeçalho + Modo Recomendado + Modo Legado + ADRs Relevantes. Modelo: `COPILOT.md` (reescrito em F1-Copilot) + ajustes Codex-específicos da pesquisa.
- [ ] 10.2 Editar `AGENTS.md` na seção de ADRs adicionando linha para ADR-013.
- [ ] 10.3 Editar `docs/cli-schema.json` adicionando propriedades `reasoning-effort` (enum) e `access-mode` (enum) ao schema das flags. Validar JSON via `jq . docs/cli-schema.json` (ou `make lint` se existir gate).
- [ ] 10.4 Editar `docs/telemetry-feedback-cycle.md` documentando cobertura Codex e os campos `runtime_init` esperados.
- [ ] 10.5 Verificar gates: `make check-skills-sync` e `make check-hooks-sync` continuam passando.
- [ ] 10.6 Rodar `make lint` (golangci-lint) — não deve impactar pois mudanças são em .md/.json.

## Detalhes de Implementação

Ver `techspec.md` §"Sequenciamento de Desenvolvimento" → itens 13-16 e §"Arquivos Relevantes e Dependentes" → seção "Modificados". Conteúdo modelo para `CODEX.md` em `docs/research/compozy-adaptation-codex-2026.md` §"Exemplos de Configuração 2026 — `CODEX.md` raiz reescrita F1-Codex".

Anti-padrão: NÃO inventar novos campos no `cli-schema.json` sem validar contra o conjunto real de flags em `cmd/ai_spec_harness/task_loop.go`.

## Critérios de Sucesso

- `CODEX.md` reescrito conforme modelo da pesquisa; seção "Modo Recomendado" é a primeira após cabeçalho.
- `AGENTS.md` lista ADR-013 na tabela.
- `docs/cli-schema.json` válido (JSON syntax) e contém enums das duas flags novas.
- `docs/telemetry-feedback-cycle.md` menciona Codex.
- Gates `make check-skills-sync`, `make check-hooks-sync`, `make vet`, `make lint` continuam passando.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Validação manual: `CODEX.md` renderiza corretamente em Markdown viewer; estrutura segue modelo F1-Copilot.
- [ ] `jq . docs/cli-schema.json > /dev/null` (sem erro de sintaxe).
- [ ] `grep -c "reasoning-effort\|access-mode" docs/cli-schema.json` ≥ 2.
- [ ] `grep -c "ADR-013\|013-codex-cli-acp-native" AGENTS.md` ≥ 1.
- [ ] `grep -c "codex" docs/telemetry-feedback-cycle.md` ≥ 1.
- [ ] `make check-skills-sync` 0 drift.
- [ ] `make check-hooks-sync` 0 drift.
- [ ] `make lint` sem erros.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `CODEX.md` reescrito com §"Modo Recomendado (2026)" primeira, §"Modo Legado" deprecada, ≥ 80 linhas, documentando: comando, pré-condições, flags Codex-específicas (`--reasoning-effort`, `--access-mode`), warning sobre `--access-mode=full`, confusão `codex` vs `codex-acp`, referência a ADR-013.
- [ ] `AGENTS.md` tabela de ADRs contém linha ADR-013 com link relativo correto.
- [ ] `docs/cli-schema.json` válido em JSON e contém propriedades enum para `reasoning-effort` (`["low","medium","high"]`) e `access-mode` (`["restricted","full"]`).
- [ ] `docs/telemetry-feedback-cycle.md` documenta que F1-Codex cobre `tool=codex` nos mesmos kinds (`runtime_init`, etc.) que F1-Claude/F1-Copilot.
- [ ] Gates pass: `make check-skills-sync`, `make check-hooks-sync`, `make lint`, `make vet`.
- [ ] `jq . docs/cli-schema.json` retorna JSON válido.

## Arquivos Relevantes

- `CODEX.md` (raiz — reescrever)
- `AGENTS.md` (raiz — atualizar tabela ADRs)
- `docs/cli-schema.json` (atualizar)
- `docs/telemetry-feedback-cycle.md` (atualizar)
- ADR-013 §"Decisão" → D-05 (janela legacy), D-08 (warning full)
- techspec.md §"Sequenciamento de Desenvolvimento" → itens 13-16
- `docs/research/compozy-adaptation-codex-2026.md` §"Exemplos de Configuração 2026 — `CODEX.md` raiz" (modelo)
- `COPILOT.md` reescrita em F1-Copilot (referência de formato)
