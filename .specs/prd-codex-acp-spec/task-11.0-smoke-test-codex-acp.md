# Tarefa 11.0: Smoke test real codex-acp + captura audit/

Status: blocked

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Executar **smoke test E2E manual** com o binário real `codex-acp` (não fake server) para validar paridade observacional Codex ↔ Claude ↔ Copilot. Capturar evidência forense em `audit/<timestamp>-codex-acp-smoke/` contendo `events.jsonl`, `tool_calls.md`, `execution_report.md`.

Esta é a **gate final de F1-Codex** (RF-19 + OB-02 + OB-06). Sem ela, paridade observacional não está validada fora do fake ACP server. O smoke não é automatizado em CI nesta fase (D-08 do techspec — assume disponibilidade quando `LookPath` resolve).

Pré-condição: `codex-acp >= 0.12.0` instalado OU fallback `npx --yes @zed-industries/codex-acp@0.14.0` operacional. Em 2026-05-21 a máquina local não tem o binário (`which codex-acp` not found); operador deve `npm install -g @zed-industries/codex-acp@0.14.0` antes de iniciar.

<requirements>
- codex-acp ou npx disponíveis para fallback.
- Auth Codex/Zed válido (pré-condição operacional fora do escopo do harness).
- Modo restricted: smoke produz events.jsonl com runtime_init carregando tool=codex, npm_version=0.14.0, sdk_version=v0.13.0.
- Modo full: warning único emitido em stderr antes da invocação.
- tool_calls.md e execution_report.md gerados com mesma estrutura que Claude/Copilot.
- ActivityWatchdog opera (ou time-out apropriado se sessão concluir).
- Evidência completa em audit/<timestamp>-codex-acp-smoke/ — versionada (git add -f) para PR record.
- Suíte completa Go (T-31, T-32, T-33) 100% verde antes do smoke.
</requirements>

## Subtarefas

- [ ] 11.1 Confirmar pré-condições: `codex-acp --version` retorna `>= 0.12.0` OU `npx --yes @zed-industries/codex-acp@0.14.0 --version` funcional.
- [ ] 11.2 Rodar suíte completa pré-smoke: `go test ./...` 100% verde; `make lint`, `make vet` sem erros; `make check-skills-sync`/`check-hooks-sync` 0 drift.
- [ ] 11.3 Construir binário: `make build` produzir `./ai-spec` atualizado.
- [ ] 11.4 Smoke modo restricted (default):
  ```bash
  GOVERNANCE_TELEMETRY=1 ./ai-spec task-loop \
    --tool codex \
    --runtime acp \
    --reasoning-effort medium \
    --activity-timeout 120s \
    .specs/prd-codex-acp-spec
  ```
- [ ] 11.5 Smoke modo full (com warning):
  ```bash
  ./ai-spec task-loop \
    --tool codex --runtime acp \
    --reasoning-effort high --access-mode full \
    .specs/prd-codex-acp-spec
  ```
  Verificar que warning `--access-mode=full` é emitido em stderr **antes** da invocação.
- [ ] 11.6 Capturar artefatos: copiar `audit/<run>/events.jsonl`, `tool_calls.md`, `execution_report.md` para `audit/<timestamp>-codex-acp-smoke/`.
- [ ] 11.7 Verificar `events.jsonl` contém eventos: `runtime_init`, `agent_message`, `tool_call_start`, `tool_call_end`, `completion` (no mínimo).
- [ ] 11.8 Verificar `runtime_init` carrega: `tool=codex`, `npm_version="0.14.0"`, `sdk_version="v0.13.0"`, `launcher="binary"` ou `launcher="npx"`.
- [ ] 11.9 Verificar `.agents/telemetry.log` (se `GOVERNANCE_TELEMETRY=1`) contém entrada `skill=runtime_init tool=codex launcher=...`.
- [ ] 11.10 Comparar estrutura JSON de `events.jsonl` Codex vs Claude/Copilot existentes em `evidence/task-10.0/` (de F1-Claude). Diferenças aceitáveis: nomes de tool (`search_query` em Codex vs `web_search` em outros — adiado para F2-Codex per D-09); cardinalidade `tool` no `runtime_init`.
- [ ] 11.11 Versionar a evidência: `git add -f audit/<timestamp>-codex-acp-smoke/` (audit/ é gitignored — usar -f).

## Detalhes de Implementação

Ver `techspec.md` §"Abordagem de Testes" → "Testes E2E" e §"Sequenciamento de Desenvolvimento" → item 17. Decisão registrada em ADR-013 §"Decisão" D-08 (smoke manual fora de CI nesta fase) e §"Consequências/Negativas/Riscos" R-04 (ambientes air-gapped requerem instalação manual).

Anti-padrão: NÃO automatizar este smoke em CI nesta fase — requer auth Codex/Zed válida no runner que não é garantida.

## Critérios de Sucesso

- Smoke restricted produz `events.jsonl` com kinds esperados; runtime_init com metadata Codex correta.
- Smoke full emite warning único antes da invocação.
- `tool_calls.md` e `execution_report.md` gerados com mesma estrutura que evidências Claude/Copilot pré-existentes.
- Evidência versionada em `audit/<timestamp>-codex-acp-smoke/` (via git add -f).
- Suíte Go completa pré-smoke 100% verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Pré-smoke: `go test ./...` → 100% verde.
- [ ] Pré-smoke: `make lint`, `make vet` → sem erros.
- [ ] Pré-smoke: `make check-skills-sync` + `make check-hooks-sync` → 0 drift.
- [ ] Smoke restricted: comando do 11.4 termina com exit 0; `audit/<run>/` contém os 3 artefatos forenses.
- [ ] Smoke full: warning `--access-mode=full` emitido em stderr **uma vez** antes da invocação.
- [ ] `events.jsonl` parse OK: cada linha JSON válido; kinds esperados presentes.
- [ ] `runtime_init` event tem `tool=codex`, `npm_version=0.14.0`, `sdk_version=v0.13.0`.
- [ ] `tool_calls.md` listing of tool calls aggregated (não vazio se Codex usou tools).
- [ ] `execution_report.md` tem `Launcher: binary` ou `Launcher: npx`, `EventsCount > 0`, `UnknownEventsCount == 0`.
- [ ] T-31, T-32, T-33 (suítes regressão) 100% verde em isolamento.
- [ ] Evidência adicionada ao repo via `git add -f audit/<timestamp>-codex-acp-smoke/`.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] Pré-condições verificadas: `codex-acp --version` ≥ 0.12.0 OU `npx` operacional.
- [ ] Suíte completa Go 100% verde antes do smoke.
- [ ] Smoke restricted executado; `events.jsonl` produzido; campos `runtime_init` corretos (`tool=codex`, versões).
- [ ] Smoke full executado; warning único emitido em stderr antes da invocação.
- [ ] `tool_calls.md` e `execution_report.md` gerados em `audit/<run>/`.
- [ ] Evidência copiada para `audit/<timestamp>-codex-acp-smoke/` (timestamp ISO 8601 compacto, ex: `20260521-1530`).
- [ ] Estrutura forense paritária com Claude/Copilot pré-existentes (comparação manual aceitável; aliasing `search_query`/`image_query` é divergência conhecida adiada — D-09).
- [ ] Evidência versionada via `git add -f audit/<timestamp>-codex-acp-smoke/` para PR record.
- [ ] PRD §"Métricas mensuráveis" validadas: probe < 200ms p95 (binary) ou < 2s p95 (npx); EventsCount > 0; UnknownEventsCount == 0.

## Arquivos Relevantes

- `audit/<timestamp>-codex-acp-smoke/events.jsonl` (criar)
- `audit/<timestamp>-codex-acp-smoke/tool_calls.md` (criar)
- `audit/<timestamp>-codex-acp-smoke/execution_report.md` (criar)
- `evidence/task-10.0/` (referência para comparação estrutural — F1-Claude evidência)
- `cmd/ai_spec_harness/task_loop.go` (binário de teste)
- `CODEX.md` (consultar para pré-condições documentadas — tarefa 10.0)
- ADR-013 §"Decisão" → D-08
- techspec.md §"Abordagem de Testes" → Testes E2E
- PRD §"Métricas mensuráveis" (thresholds a validar)

## Evidência de Execução

- `ai-spec skills check`
  - Resultado: verificação concluída sem bloqueio de skills externas; 13 skills verificadas.
- `ai-spec check-spec-drift .specs/prd-codex-acp-spec/tasks.md`
  - Resultado: `DRIFT: prd.md → tasks.md: IDs faltantes: RF-25`
  - Impacto: bloqueio de governança da skill `execute-task`; o smoke 11.0 não pode prosseguir até o PRD voltar a estar 100% coberto em `tasks.md`.
- `command -v codex-acp`
  - Resultado: binário resolvido via cache local de `npx` em `/Users/jailtonjunior/.npm/_npx/5395aca3a8ed6b4b/node_modules/.bin/codex-acp`.
- `command -v npx`
  - Resultado: `/Users/jailtonjunior/.nvm/versions/node/v24.15.0/bin/npx`.
- `find evidence/task-11.0 -maxdepth 1 -type f -print | sort`
  - Resultado: existe apenas `evidence/task-11.0/events.jsonl`.
  - Impacto: a tentativa anterior não atende o DoD; faltam `tool_calls.md`, `execution_report.md` e a cópia forense final em `audit/<timestamp>-codex-acp-smoke/`.
