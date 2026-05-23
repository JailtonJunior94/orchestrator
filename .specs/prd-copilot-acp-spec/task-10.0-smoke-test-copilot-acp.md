# Tarefa 10.0: Smoke test real copilot --acp + captura audit/

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Executar smoke test end-to-end usando o binário real `copilot` (Copilot CLI) em modo `--acp` contra um PRD de tamanho representativo. Capturar evidência forense em `audit/<timestamp>-copilot-acp-smoke/` contendo `events.jsonl`, `tool_calls.md`, `execution_report.md` e snippet de `telemetry.log`. Atualizar `audit/README.md` indexando a evidência.

Esta é a tarefa de **gate final E2E**: prova fora do fake server que paridade observacional Copilot ↔ Claude é real. Sem isso, OB-02 e OB-05 do PRD não são validados.

<requirements>
- Pré-requisito: copilot --version >= CopilotMinCLIVersion (confirmado na Tarefa 2.0).
- Pré-requisito: gh auth status válido com token Copilot.
- Smoke produz events.jsonl, tool_calls.md, execution_report.md em audit/<timestamp>-copilot-acp-smoke/.
- runtime_init carrega tool=copilot, Launcher=binary|npx, EventsCount > 0, UnknownEventsCount == 0.
- audit/README.md atualizado indexando a evidência.
</requirements>

## Subtarefas

- [ ] 10.1 Verificar pré-condições: `copilot --version` ≥ `CopilotMinCLIVersion`; `gh auth status` mostra token Copilot válido; binário `copilot` no PATH.
- [ ] 10.2 Escolher PRD-alvo para smoke (sugestão: `.specs/prd-copilot-acp-spec/` em modo dry-run controlado, ou um PRD menor de uso interno).
- [ ] 10.3 Executar:
   ```bash
   GOVERNANCE_TELEMETRY=1 ai-spec-harness task-loop \
     --tool copilot \
     --runtime acp \
     --activity-timeout 120s \
     --quiet \
     .specs/prd-copilot-acp-spec
   ```
   (ou similar com PRD-alvo escolhido em 10.2).
- [ ] 10.4 Capturar os artefatos forenses gerados em `audit/<timestamp>-copilot-acp-smoke/`:
   - `events.jsonl`
   - `tool_calls.md`
   - `execution_report.md`
   - snippet relevante de `.agents/telemetry.log` (linhas com `tool=copilot`)
- [ ] 10.5 Validar manualmente:
   - `runtime_init` event no `events.jsonl` carrega `sdk_version == CopilotSDKVersion`, `npm_version == CopilotNpmVersion`, `launcher` em `{binary, npx}`.
   - `execution_report.md` registra `EventsCount > 0`, `UnknownEventsCount == 0`, `CancelReason` apropriado.
   - `tool_calls.md` agrega counts coerentes com a execução.
- [ ] 10.6 Atualizar `audit/README.md` indexando a evidência com timestamp, comando executado e resultado.
- [ ] 10.7 Confirmar com `go test ./...` que a suíte completa permanece verde (regressão final).

## Detalhes de Implementação

Ver `techspec.md` §"Testes E2E". Critério de aceitação alinhado com OB-02 (paridade observacional) e OB-05 (preservação de invariantes forenses + watchdog).

Anti-padrão: NÃO automatizar este smoke em CI nesta fase (ambiente de runner pode não ter `copilot` CLI). Smoke é manual; CI cobre apenas matriz com fake server (Tarefa 8.0).

Decisão operacional: se `copilot --acp` falhar por motivos de auth ou de versão durante o smoke, **não marcar como done** — abrir status `needs_input` documentando o bloqueio e o que está faltando para reexecutar.

## Critérios de Sucesso

- Smoke executa sem erros de pré-validação (CLI versão suficiente, auth válida, binário no PATH).
- `events.jsonl` capturado contém ≥ 1 `runtime_init`, ≥ 1 `agent_message`, e (se PRD tiver tasks reais) ≥ 1 `tool_call_start`/`tool_call_end`.
- `runtime_init` carrega versões Copilot reais.
- `execution_report.md` válido com `UnknownEventsCount == 0`.
- `audit/README.md` indexa a evidência.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Smoke manual: comando executado sem erro de validação de flags.
- [ ] Inspeção manual: `events.jsonl` contém os kinds esperados.
- [ ] Inspeção manual: `runtime_init` payload tem campos Copilot corretos.
- [ ] Inspeção manual: `execution_report.md` válido (campos preenchidos, `UnknownEventsCount == 0`).
- [ ] Inspeção manual: `.agents/telemetry.log` contém linhas `tool=copilot` se `GOVERNANCE_TELEMETRY=1`.
- [ ] Regressão final: `go test ./...` → 100% verde.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] Pré-condições verificadas (`copilot --version`, `gh auth status`).
- [ ] Comando smoke executado e completado.
- [ ] `audit/<timestamp>-copilot-acp-smoke/` contém os 4 artefatos: `events.jsonl`, `tool_calls.md`, `execution_report.md`, snippet telemetria.
- [ ] `runtime_init` no `events.jsonl` carrega versões Copilot reais (não constantes Claude).
- [ ] `execution_report.md` registra `EventsCount > 0`, `UnknownEventsCount == 0`.
- [ ] `audit/README.md` indexa a evidência com timestamp + comando + resultado.
- [ ] Regressão final `go test ./...` 100% verde.
- [ ] Se houver falha não recuperável (auth, versão CLI, ambiente), status muda para `needs_input` com bloqueio documentado.

## Arquivos Relevantes

- `audit/<timestamp>-copilot-acp-smoke/` (criar — diretório de evidência)
- `audit/README.md` (atualizar — índice de auditorias)
- `.specs/prd-copilot-acp-spec/` ou PRD-alvo escolhido (insumo do task-loop)
- `internal/runtime/specs/copilot.go` (consumido — Tarefa 2.0)
- Binário externo `copilot` (pré-requisito operacional)
- `gh auth status` (pré-requisito operacional)
