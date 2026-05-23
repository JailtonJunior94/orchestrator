# Tarefa 9.0: Documentação consolidada — `GEMINI.md` + `AGENTS.md` + `CHANGELOG.md` + telemetry-feedback-cycle

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Consolidar toda a documentação cross-cutting de F0..F5-Gemini em uma única task convergente. Reescreve `GEMINI.md` raiz (28 linhas → ~100 linhas) com seções "Runtime Capabilities (F0+/F2+/F3+/F4+/F5+)" como primárias, "Modo Legado Wrapper" deprecada, documentação do mapeamento D-05, warnings de `--access-mode=full` e `--auto-review`, defaults memory generosos e métricas Gemini-2026. Atualiza `AGENTS.md` (linha tabela ADRs adicionando ADR-015). Atualiza `CHANGELOG.md` com entradas conventional commits para F0..F5-Gemini. Atualiza `docs/telemetry-feedback-cycle.md` cobrindo invariantes Gemini ACP.

<requirements>
- `GEMINI.md` segue estrutura do `CLAUDE.md` (referência) e do exemplo em `docs/research/compozy-adaptation-gemini-2026.md` §"Exemplos de Configuração 2026".
- `AGENTS.md` recebe **uma única linha nova** na tabela de ADRs (ADR-015), sem reescrita do arquivo.
- `CHANGELOG.md` recebe 6 entradas conventional commits: `feat(gemini)` para F0..F5, mais 1 entrada `chore(deps)` para pin de `@google/gemini-cli@0.43.0`.
- `docs/telemetry-feedback-cycle.md` ganha seção curta (~10-15 linhas) sobre Gemini-2026 metrics aditivos a Claude-2026.
- Documentação só faz sentido após F2/F3/F4/F5 estarem em estado conhecido (5.0/6.0/7.0/8.0 done).
- Nenhuma mudança de código nesta task.
</requirements>

## Subtarefas

- [ ] 9.1 Validar que 5.0/6.0/7.0/8.0 estão `done` — se não, marcar 9.0 como `blocked`.
- [ ] 9.2 Reescrever `GEMINI.md` raiz: começar com mantida estrutura "Instrucoes" + "Hooks de Governanca" atuais; adicionar 5 seções "Runtime Capabilities (F0+/F2+/F3+/F4+/F5+)" conforme techspec §"Stub `GEMINI.md` raiz" (referência em research doc); seção "Modo Legado Wrapper" deprecada explicitamente.
- [ ] 9.3 Atualizar `AGENTS.md` tabela de ADRs — adicionar linha `| ADR-015 | .specs/adr/015-gemini-cli-acp-native.md | Gemini CLI ACP nativo (Proposta) |` ou equivalente conforme padrão local.
- [ ] 9.4 Atualizar `CHANGELOG.md` com entradas conventional commits:
  - `feat(gemini): F0 spec registration via gemini --acp (ADR-015)`
  - `feat(gemini): F1 paridade ACP E2E + wrapper deprecation`
  - `feat(gemini): F2 normalization cascata + MCP nested-agent`
  - `feat(gemini): F3 memory defaults generosos (250/400) + hooks cascata`
  - `feat(gemini): F4 métricas Gemini-2026 (cache_read, effective_context, prompt_billed, thoughts)`
  - `feat(gemini): F5 auto-review opt-in cascata`
  - `chore(deps): pin @google/gemini-cli@0.43.0`
- [ ] 9.5 Atualizar `docs/telemetry-feedback-cycle.md`: adicionar seção/linha sobre métricas Gemini-2026 (entries `gemini.cache_read`, `gemini.thoughts`, etc.) e invariantes Gemini ACP (mesmos kinds que Claude/Codex/Copilot, ADR-010 preservado).
- [ ] 9.6 Lint markdown via `markdownlint` (se disponível); validar TOC consistente.

## Detalhes de Implementação

Ver techspec.md:
- §"Considerações Técnicas / Arquivos Relevantes / Modificados (edição cirúrgica)" — lista exata de arquivos.
- §"Mapeamento RF → Componente → Teste" — RF-23, RF-24, RF-26, RF-27.

Ver pesquisa em `docs/research/compozy-adaptation-gemini-2026.md` §"Exemplos de Configuração 2026" — template literal do `GEMINI.md` reescrito.

Precedente direto: `.specs/prd-codex-acp-spec/task-10.0-documentacao-f1-codex.md` (mesmo padrão de docs consolidadas).

## Critérios de Sucesso

- `GEMINI.md` raiz contém todas as 5 seções "Runtime Capabilities" (F0+, F2+, F3+, F4+, F5+) com texto literal próximo ao da pesquisa.
- `AGENTS.md` tabela de ADRs inclui ADR-015 com link funcional.
- `CHANGELOG.md` contém 6 entradas `feat(gemini)` + 1 entrada `chore(deps)`, ordenadas cronologicamente.
- `docs/telemetry-feedback-cycle.md` documenta Gemini metrics adicionados a invariantes existentes.
- `git diff --stat GEMINI.md AGENTS.md CHANGELOG.md docs/telemetry-feedback-cycle.md` mostra **apenas** estes 4 arquivos modificados.
- Lint markdown sem warnings (ou warnings pré-existentes apenas).
- Nenhuma mudança em código Go (`go test ./...` continua verde).

### Definition of Done

1. `GEMINI.md` reescrito com 5 seções Runtime Capabilities + seção "Modo Legado" deprecada.
2. `AGENTS.md` linha ADR-015 adicionada.
3. `CHANGELOG.md` com 7 entradas (6 feat + 1 chore).
4. `docs/telemetry-feedback-cycle.md` cobre Gemini metrics.
5. Nenhuma alteração em código Go.
6. 5.0/6.0/7.0/8.0 confirmados `done` antes do início.
7. Lint markdown limpo.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Lint markdown em `GEMINI.md`, `AGENTS.md`, `CHANGELOG.md`, `docs/telemetry-feedback-cycle.md`
- [ ] `git grep "ADR-015" AGENTS.md` retorna ≥ 1 match
- [ ] `git grep "F0+" GEMINI.md` retorna ≥ 1 match (presença das seções Runtime Capabilities)
- [ ] `git grep "feat(gemini)" CHANGELOG.md` retorna 6 matches
- [ ] `git grep "@google/gemini-cli" CHANGELOG.md` retorna 1 match
- [ ] `go test ./...` continua verde (regressão Go)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **EDIÇÃO (reescrita)**: `GEMINI.md` (~28 → ~100 linhas)
- **EDIÇÃO (1 linha)**: `AGENTS.md` (tabela de ADRs)
- **EDIÇÃO (7 entradas)**: `CHANGELOG.md`
- **EDIÇÃO (~10-15 linhas)**: `docs/telemetry-feedback-cycle.md`
- **REFERÊNCIA**: `docs/research/compozy-adaptation-gemini-2026.md` §"Exemplos de Configuração 2026"
- **REFERÊNCIA**: `CLAUDE.md` (estrutura template), `.specs/prd-codex-acp-spec/task-10.0-*.md`
