# Documento de Requisitos do Produto (PRD)

<!-- spec-version: 2 -->

## Visão Geral

O `ai-spec-harness` invoca agentes de IA via `exec.Cmd` one-shot (`internal/taskloop/agent.go`): manda um prompt, captura `stdout`/`stderr`/exit code, encerra o processo. Esse modelo perde toda a granularidade do que o agente faz durante a execução (mensagens parciais, raciocínio, chamadas de tool), só permite cancelamento por timeout absoluto e impossibilita features futuras como TUI ao vivo, execução concorrente com ramp-down gracioso ou retomada de sessão.

Esta funcionalidade substitui o caminho de invocação para o agente **Claude** por um cliente que fala o **Agent Client Protocol (ACP)**, com streaming de eventos em tempo real e watchdog de atividade. O caminho legacy permanece como default; o novo é ativado por flag `--runtime=acp`. Outros runtimes (Codex, Gemini, Copilot, etc.) ficam de fora — serão tratados em PRDs subsequentes.

O resultado é a base técnica para evoluir o `task-loop` rumo à paridade comportamental com runtimes como o do Compozy, sem quebrar consumidores atuais.

## Objetivos

- Tornar a invocação do Claude observável evento a evento, com `events.jsonl` por task como artefato auditável.
- Reduzir o tempo entre travamento real do agente e detecção/cancelamento de "timeout absoluto da task" para "≤ 120s sem qualquer evento do agente" (default do activity watchdog).
- Validar o protocolo ACP no harness com **uma** integração de runtime, antes de generalizar para outros IDEs.
- Manter 100% de compatibilidade com o caminho legacy enquanto a flag `--runtime=acp` for opt-in: nenhuma task existente deve mudar de comportamento sem mudança explícita de flag.

**Métricas de sucesso (mensuráveis ao encerrar o PRD):**
- `ai-spec task-loop --tool claude --runtime acp` executa uma task real e grava `evidence/<task>/events.jsonl` com pelo menos os eventos `agent_message`, `tool_call_start`, `tool_call_update` e `session_end` quando aplicáveis.
- Quando o agente fica mudo por mais de `--activity-timeout` (default 120s), o processo é cancelado e o `execution_report.md` registra `cancel_reason=activity_timeout`.
- Suite de testes roda sem o binário `claude-agent-acp` instalado, usando fake ACP server embutido; teste live executa apenas com `AI_SPEC_ACP_LIVE=1`.
- Caminho legacy (`--runtime=legacy`, default) continua passando 100% dos testes existentes.

## Histórias de Usuário

- Como **mantenedor de um repositório instrumentado**, quero ver em tempo real o que o agente está fazendo (mensagens, raciocínio, tool calls) para depurar prompts e configuração sem esperar a task terminar.
- Como **operador de CI**, quero que uma task travada (agente vivo mas mudo) seja cancelada em até 2 minutos para liberar runner, em vez de esperar o timeout absoluto da pipeline.
- Como **auditor de governança**, quero ter, por task executada via ACP, um arquivo `events.jsonl` com a sequência crua de eventos do ACP (incluindo `tool_call_id`, timestamps e payloads) para reconstruir o que aconteceu sem depender do log humano.
- Como **mantenedor do `ai-spec-harness`**, quero validar a integração ACP em um runtime antes de pagar o custo de generalizar para sete runtimes, para reduzir retrabalho se o protocolo ou o SDK mudar.

## Funcionalidades Core

1. **Cliente ACP para Claude** — Implementação de cliente que abre sessão ACP com `claude-agent-acp` (binário canônico) ou fallback `npx --yes @agentclientprotocol/claude-agent-acp`, envia o prompt produzido pelo `executor_template.tmpl` e consome o stream de `SessionUpdate` até `session_end`.
2. **Tradução de eventos** — Conversão de `acp.SessionUpdate` para um tipo interno `runtime.Event` com discriminação de `AgentMessage`, `AgentThought`, `ToolCallStart`, `ToolCallUpdate`, `SessionEnd` (espelhando a camada `acp_convert.go` do Compozy). Eventos desconhecidos viram `runtime.Event{Kind: "unknown", Raw: ...}` em vez de erro.
3. **Activity watchdog** — Monitor que registra timestamp do último evento recebido e cancela o `context` quando ultrapassa `--activity-timeout` (default 120s). É independente do timeout absoluto da task, que continua existindo.
4. **Output dual** — Stream humano no stdout durante a execução (linhas como `[agent] ...`, `[tool] ...`) e `events.jsonl` persistido em `evidence/<task>/events.jsonl` com o evento serializado completo.
5. **Evidência enriquecida** — Geração adicional de `evidence/<task>/tool_calls.md` (sumário humano dos tool calls do agente) preservando o `execution_report.md` no formato atual, com campos novos opcionais (`runtime: acp`, `cancel_reason`, `events_count`).
6. **Flag de runtime** — Novo argumento `--runtime=acp|legacy` em `cmd/ai_spec_harness/task_loop.go` com default `legacy`. Quando `acp`, exige `--tool claude` e falha com mensagem clara se for outro tool.
7. **Teste sem dependência externa** — Fake ACP server embutido em `internal/runtime/acpfake/` simula o `claude-agent-acp` para unit e integration tests; teste live em `tests/runtime/live/` roda apenas com `AI_SPEC_ACP_LIVE=1`.
8. **ADR-009 documentado** — Decisão de adoção do `github.com/coder/acp-go-sdk` registrada em `.specs/adr/009-acp-protocol-adoption.md` cobrindo alternativas, riscos e plano de implementação.

## Requisitos Funcionais

- **RF-01:** O CLI deve aceitar a flag `--runtime` com valores `legacy` (default) e `acp`.
- **RF-02:** Quando `--runtime=acp` for usado com `--tool` diferente de `claude`, o CLI deve falhar com exit code 2 e mensagem `runtime acp suporta apenas --tool claude nesta versão`.
- **RF-03:** Com `--runtime=acp --tool claude`, o CLI deve resolver o binário do agente nesta ordem em uma fase `EnsureAvailable` executada **antes** de gerar o prompt: (a) `claude-agent-acp` no `PATH`; (b) `npx --yes @agentclientprotocol/claude-agent-acp@<VERSAO_PINADA>` se `npx` existir, onde `<VERSAO_PINADA>` é uma constante Go definida em `internal/runtime/specs/claude.go` e atualizada por processo de `audit/`; (c) falhar com exit code 2 e mensagem contendo três remédios explícitos: `Install claude-agent-acp; OR install @agentclientprotocol/claude-agent-acp@<VERSAO_PINADA> via npm; OR use --runtime=legacy`. A mensagem deve referenciar o ADR-009.
- **RF-04:** O cliente ACP deve abrir uma sessão usando `coder/acp-go-sdk`, enviar o prompt gerado pelo `executor_template.tmpl` e consumir o stream até `session_end` ou erro.
- **RF-05:** Cada `acp.SessionUpdate` recebido deve ser convertido em um `runtime.Event` interno cobrindo no mínimo os kinds: `agent_message`, `agent_thought`, `tool_call_start`, `tool_call_update`, `session_end`, e o evento sintético `runtime_init` emitido pelo próprio harness antes da primeira mensagem. Tipos não mapeados devem virar `runtime.Event{Kind: "unknown", RawKind: <string>}`, ser persistidos no `events.jsonl` e contabilizados separadamente; nunca devem causar pânico nem interromper a sessão. Ao final da task, se houver pelo menos um evento `unknown`, o harness deve imprimir em stderr uma linha agregada no formato `N unknown ACP events skipped (kinds: a, b, c)`.
- **RF-06:** O activity watchdog deve cancelar o `context` quando o intervalo entre o último evento recebido e o instante atual exceder `--activity-timeout` (default `120s`, configurável). O cancelamento deve registrar `cancel_reason=activity_timeout` no `execution_report.md`.
- **RF-07:** A flag `--activity-timeout` deve aceitar valores no formato `time.Duration` do Go (ex.: `90s`, `2m`). Valor `0` desabilita o watchdog.
- **RF-08:** Para cada task executada via `--runtime=acp`, o sistema deve gravar `evidence/<task>/events.jsonl` contendo um evento por linha, em ordem de chegada, no envelope: `{ts: <RFC3339Nano>, kind: <string>, tool_call_id: <string|null>, launcher: <"binary"|"npx">, raw: <acp.SessionUpdate JSON inteiro>}`. O primeiro evento gravado deve ser sempre o sintético `runtime_init` carregando `{launcher, command, args, sdk_version, npm_version}` no `raw` para rastreabilidade stand-alone.
- **RF-09:** Para cada task executada via `--runtime=acp`, o sistema deve gravar `evidence/<task>/tool_calls.md` listando cada tool call com nome, status final, e referência ao `tool_call_id` no `events.jsonl`. Se a task não fez tool calls, o arquivo é criado com `Nenhum tool call registrado.`.
- **RF-10:** O `execution_report.md` gerado no modo ACP deve manter os campos atuais e adicionar: `runtime: acp`, `launcher: <binary|npx>`, `events_count: <int contando apenas eventos mapeados>`, `unknown_events_count: <int>`, `cancel_reason: <none|activity_timeout|context_canceled|tool_error|permission_denied>`.
- **RF-11:** O stream humano no stdout deve renderizar eventos com prefixos `[agent]`, `[thought]`, `[tool]`, `[tool:done]` e ser desabilitável com `--quiet` (apenas o `events.jsonl` é gravado).
- **RF-12:** A suíte de testes existente do `task-loop` em modo `--runtime=legacy` deve continuar passando sem alteração.
- **RF-13:** Testes unitários e de integração do caminho ACP devem usar `internal/runtime/acpfake/` e rodar sem o binário `claude-agent-acp` instalado.
- **RF-14:** O teste live deve residir em `tests/integration/acp_live/` protegido por build tag `//go:build acp_live`. A execução requer `go test -tags=acp_live ./tests/integration/acp_live` e, dentro do teste, `t.Skip` quando `claude-agent-acp` ausente do `PATH`. Sem a build tag, o pacote não é compilado.
- **RF-15:** O ADR-009 (`.specs/adr/009-acp-protocol-adoption.md`) deve existir no momento do merge da implementação, com status `Aceita` e referenciar este PRD. O ADR pode permanecer em status `Proposta` até a techspec ser aprovada; transição para `Aceita` é gate de merge da implementação.
- **RF-16:** Quando o agente ACP emitir `requestPermission`, o harness deve cancelar a sessão imediatamente, encerrar a task com `cancel_reason=permission_denied`, exit code 3, e mensagem em stderr: `agent requested permission; configure accessMode=bypassPermissions no claude-agent-acp ou execute em ambiente que pré-aprove. Veja ADR-009`. Não há auto-allow nem auto-deny nesta fase.

## Restrições Técnicas de Alto Nível

- **Dependência nova obrigatória:** `github.com/coder/acp-go-sdk`. A versão deve ser pinada à **última versão stable com tag semântica** disponível no momento do merge (formato `require github.com/coder/acp-go-sdk vX.Y.Z` — sem pseudo-version de commit, sem `replace`). Qualquer upgrade subsequente exige decisão em `audit/` seguindo `.specs/templates/skill-upgrade-decision.md`. Não habilitar Renovate/Dependabot automático para esta dependência enquanto o SDK não atingir 1.0 estável.
- **Compatibilidade Go:** Manter compatibilidade com `go.mod` atual (`Go 1.26.2`).
- **Governança PRD-First e Spec-Hash:** A implementação subsequente deve gerar techspec, tasks e ADR-009 ligados pelo spec-hash via `ai-spec sync-spec-hash`. Não é permitido implementar sem PRD/techspec/tasks rastreáveis.
- **Backward compatibility obrigatória:** Default `--runtime=legacy`; toda task existente deve passar sem mudanças.
- **Segurança:** O fallback `npx --yes @agentclientprotocol/claude-agent-acp@<VERSAO_PINADA>` executa código de pacote npm. A versão npm é uma constante Go pinada em `internal/runtime/specs/claude.go`, atualizável apenas via processo `audit/`. Sem versionamento implícito (`@latest`). O launcher efetivamente usado fica registrado no `execution_report.md` (campo `launcher`) e como primeiro evento `runtime_init` no `events.jsonl`.
- **Sem rede além do ACP:** O cliente ACP fala com o subprocesso por stdio; nenhuma chamada de rede direta ao provider LLM deve ser feita pelo `ai-spec-harness` (essa é responsabilidade do `claude-agent-acp`).
- **Observabilidade existente:** Telemetria opt-in (`GOVERNANCE_TELEMETRY=1`) deve continuar funcionando e ganhar dois campos novos: `runtime` e `events_count`.

## Fora de Escopo

- **Outros runtimes ACP** (Codex, Cursor, Droid, OpenCode, Pi, Gemini, Copilot) — ficam para PRDs subsequentes.
- **TUI ao vivo** (bubbletea, sidebar, blocks) — somente stdout linear nesta fase.
- **Execução concorrente / batch / DAG de dependências entre tasks** — `task-loop` continua sequencial.
- **Daemon, `runs attach`/`watch`, registry de workspaces, snapshots persistentes** — sem daemon nesta fase.
- **Retries com backoff** — falhas continuam sendo classificadas pelo caminho atual; sem retry específico do ACP.
- **Migração do formato de config** (`.ai_spec_harness.json` → TOML hierárquico) — fora de escopo.
- **Memory entre runs** (two-tier markdown memory) — fora de escopo.
- **Reusable agents** (`.ai-spec/agents/<name>/`) — fora de escopo.
- **Substituir o `execution_report.md` por um formato novo** — ele permanece como contrato; só ganha campos opcionais.
- **Auto-allow / auto-deny de `requestPermission`** — modo ACP nesta fase falha imediatamente conforme RF-16; flags como `--on-permission=allow|deny` ficam para PRD futuro.

## Suposições e Questões em Aberto

**Suposições assumidas (validadas em rodada de perguntas de produto):**
- O `github.com/coder/acp-go-sdk` está em estado utilizável para Claude (compozy o usa em produção). A versão pinada exata é decidida na techspec, mas a política é "última stable tagged + audit/ manual" (não auto-upgrade).
- O `claude-agent-acp` instalado pelo usuário expõe as flags equivalentes às usadas pelo Compozy (sem `BootstrapArgs` customizados, conforme `registry_specs.go:claude` mostra). Se vier a exigir argumentos extras, vira RF adicional no techspec.
- O usuário aceita o trade-off de adicionar uma dependência Go externa (`coder/acp-go-sdk`) no `go.mod` em troca do streaming.
- Air-gapped sem `claude-agent-acp` nem `npx` é caso esperado e tratado por `EnsureAvailable` (RF-03); cobertura por teste com `acpfake`.

**Questões em aberto:** nenhuma. Todas as decisões pendentes para o techspec foram fechadas em rodada de perguntas de produto antes do `spec-version: 2`. A techspec deve detalhar **como** implementar, não **se** ou **o que**.
