# Tarefa 4.0: Sub-suite Gemini integration + smoke test + warning `--access-mode=full` + CompatibilityTable

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Entregar o gate de paridade observacional Gemini ↔ Claude/Codex/Copilot. Adiciona sub-suite Gemini em `internal/runtime/acp_integration_test.go` reusando o fake ACP server existente (cobertura ≥ 90% dos casos cobertos para Codex). Cria smoke test em `tests/integration/gemini_acp_smoke_test.go` que invoca `gemini --acp` real ou via fake (skipable via `-short`). Estende `accessModeFullWarnOnce` em `cmd/ai_spec_harness/task_loop.go` para cobrir Gemini (warning único em stderr antes de spawn quando `--access-mode=full`). Valida que `internal/taskloop/compatibility.go::CompatibilityTable::IsSupported("gemini", "gemini-2.5-pro")` retorna `true` (RF-28; sem mudança de código esperada).

<requirements>
- Sub-suite Gemini cobre: open OK, prompt, ≥ 2 tipos de tool call, agent message, completion, cancel por `ActivityWatchdog`, erro de launcher unavailable, fallback npx, validação de que `geminiBootstrapArgs` produz `--approval-mode <value>` no spawn (`default` por padrão; `yolo` em `--access-mode=full`).
- Smoke test (`tests/integration/gemini_acp_smoke_test.go`) usa build tag `//go:build integration`. Skip quando `gemini` não disponível no PATH e `-short` ativo.
- `accessModeFullWarnOnce` estendido: warning é único por execução do processo independente de quantas tasks são processadas. Mensagem específica para Gemini: `"WARNING: --access-mode=full ativa --approval-mode=yolo no gemini-cli. Pré-condição: consentimento operacional. Ver GEMINI.md."`
- `TestCompatibilityTableContainsGemini` valida que entrada Gemini já existente em `internal/taskloop/compatibility.go:34-43` está intacta.
- Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/client.go`.
</requirements>

## Subtarefas

- [ ] 4.1 Adicionar sub-suite Gemini em `internal/runtime/acp_integration_test.go` reusando fake ACP server (estrutura idêntica à sub-suite Codex/Copilot).
- [ ] 4.2 Criar `tests/integration/gemini_acp_smoke_test.go` com build tag `//go:build integration` e skip condicional (sem `gemini` no PATH OR `testing.Short()`).
- [ ] 4.3 Em `cmd/ai_spec_harness/task_loop.go`, estender lógica de `accessModeFullWarnOnce` (linhas ~19-21) para cobrir Gemini (mensagem específica conforme RF-33).
- [ ] 4.4 Estender `internal/taskloop/compatibility_test.go` (ou criar `TestCompatibilityTableContainsGemini` se ausente) validando `IsSupported("gemini", "gemini-2.5-pro")`, `IsSupported("gemini", "pro")`, `IsSupported("gemini", "flash")` etc.
- [ ] 4.5 Estender T-13/T-14/T-15/T-16/T-29/T-30/T-31 no `task_loop_test.go` confirmando Gemini agora aceito (gate inverter conforme padrão F1-Codex T-14).
- [ ] 4.6 Validar `events.jsonl`/`tool_calls.md`/`execution_report.md` produzidos via sub-suite Gemini tem mesma estrutura que Claude/Codex/Copilot.

## Detalhes de Implementação

Ver techspec.md:
- §"Abordagem de Testes / Testes de Integração" — lista exata de cenários.
- §"Mensagens de Erro e Warning Literais" RF-33 — texto exato do warning.
- §"Arquitetura do Sistema / F1-Gemini" — escopo de arquivos novos/modificados.
- §"Considerações Técnicas / TD-06" — sequenciamento (4.0 é gate antes da fan-out W2).

Precedente direto: `.specs/prd-codex-acp-spec/task-9.0-acp-integration-codex-suite.md` (mesma estrutura para Codex; sub-suite reusa fake server). `task-11.0-smoke-test-codex-acp.md` (smoke real).

## Critérios de Sucesso

- `go test -run TestACPIntegration_Gemini -v ./internal/runtime/...` retorna `PASS` com ≥ 8 sub-tests cobrindo cenários listados.
- `go test -tags integration ./tests/integration/... -run TestGeminiACPSmoke` passa quando `gemini` no PATH; skip limpo quando ausente.
- `go test -run TestAccessModeFullEmitsWarningForGemini ./cmd/ai_spec_harness/...` retorna `PASS`.
- `go test -run TestCompatibilityTableContainsGemini ./internal/taskloop/...` retorna `PASS` com ≥ 6 modelos Gemini válidos.
- Comando `ai-spec-harness task-loop --tool gemini --runtime acp --access-mode full --dry-run .specs/prd-fake` emite warning único em stderr na primeira invocação; silencioso na segunda.
- `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/client.go` retorna **zero linhas** modificadas (RF-32 parcial).
- Suite regressão Claude/Codex/Copilot integration permanece 100% verde.

### Definition of Done

1. Sub-suite Gemini cobre ≥ 90% dos casos cobertos para Codex (medir por número de sub-tests).
2. Smoke test rodável com flag `-tags integration`; skip explícito documentado.
3. Warning `--access-mode=full` emitido exatamente uma vez para Gemini (testar via mock stderr writer).
4. `CompatibilityTable` valida Gemini sem mudança de código (RF-28).
5. Diff zero em módulos protegidos (`persistence/`, `watchdog.go`, `client/`).
6. T-13..T-16 e T-29..T-31 estendidos para Gemini também verdes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Sub-suite `TestACPIntegration_Gemini_*` (≥ 8 sub-tests)
- [ ] `TestGeminiACPSmoke` (build tag integration; skipable)
- [ ] `TestAccessModeFullEmitsWarningForGemini` (sync.Once)
- [ ] `TestCompatibilityTableContainsGemini`
- [ ] T-13..T-16, T-29..T-31 estendidos para Gemini (já em 1.0; aqui valida via task_loop_test.go integrado)
- [ ] Regressão: suite Claude/Codex/Copilot integration verde

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **EDIÇÃO**: `internal/runtime/acp_integration_test.go` (sub-suite Gemini)
- **NOVO**: `tests/integration/gemini_acp_smoke_test.go`
- **EDIÇÃO**: `cmd/ai_spec_harness/task_loop.go` (extensão accessModeFullWarnOnce para Gemini)
- **EDIÇÃO**: `internal/taskloop/compatibility_test.go` ou novo `TestCompatibilityTableContainsGemini`
- **EDIÇÃO**: `cmd/ai_spec_harness/task_loop_test.go` (T-13..T-16, T-29..T-31 estendidos)
- **REFERÊNCIA (não modificar)**: `internal/taskloop/compatibility.go:34-43` (Gemini já catalogado)
