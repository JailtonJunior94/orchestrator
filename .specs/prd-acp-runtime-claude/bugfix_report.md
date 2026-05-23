# Relatorio de Bugfix

- Total de bugs no escopo: 5
- Corrigidos: 5
- Testes de regressao adicionados: 8
- Pendentes: nenhum
- Estado final: done

## Bugs
- ID: BUG-ACP-01
- Severidade: critical
- Estado: fixed
- Causa raiz: o modo ACP reutilizava o preflight legacy `CheckAgentBinary`, abortando antes de `probe.EnsureAvailable` e impedindo o fallback via `npx` exigido pelo RF-03.
- Arquivos alterados: `internal/taskloop/taskloop.go`, `internal/taskloop/taskloop_test.go`, `cmd/ai_spec_harness/task_loop.go`
- Teste de regressao: `TestExecuteACP_SkipsLegacyBinaryChecker`, `TestExecuteACP_ReturnsLauncherUnavailableImmediately`
- Validacao: `go test ./internal/taskloop ./cmd/ai_spec_harness` passou; fluxo ACP agora ignora o checker legacy e propaga `ErrLauncherUnavailable` para o mapeamento `exit2`.

- ID: BUG-ACP-02
- Severidade: critical
- Estado: fixed
- Causa raiz: o `client.readLoop` descartava o `PromptResponse` final do SDK e fechava o canal sem sintetizar `session_end`, quebrando RF-05/RF-08 e impedindo o registro do evento terminal.
- Arquivos alterados: `internal/runtime/client/client.go`, `internal/runtime/client/client_test.go`, `internal/runtime/acp_integration_test.go`
- Teste de regressao: `TestAcpClient_HappyPath`, `TestACPRunner_HappyPath`, `TestACPRunner_WithRealPersistence`
- Validacao: `go test ./internal/runtime/client ./internal/runtime` passou; `events.jsonl` agora termina com `session_end` e o client emite o evento terminal no caminho feliz.

- ID: BUG-ACP-03
- Severidade: critical
- Estado: fixed
- Causa raiz: `RequestPermission` respondia `Cancelled`, mas nao cancelava o prompt turn nem produzia `permission_denied`; o runner aceitava encerramento normal e os testes mascaravam o contrato do RF-16.
- Arquivos alterados: `internal/runtime/client/client.go`, `internal/runtime/client/client_test.go`, `internal/runtime/runner.go`, `internal/runtime/acp_integration_test.go`
- Teste de regressao: `TestAcpClient_RequestPermissionCancelsPrompt`, `TestACPRunner_PermissionDenied`
- Validacao: `go test ./internal/runtime/client ./internal/runtime` passou; o cliente agora cancela imediatamente, retorna `client.ErrPermissionDenied`, o runner publica `cancel_reason=permission_denied` e imprime a orientacao de RF-16.

- ID: BUG-ACP-04
- Severidade: major
- Estado: fixed
- Causa raiz: o `acpInvoker` gravava tudo em `workDir/evidence/acp`, sem derivar o diretório da task corrente a partir do prompt, misturando artefatos de execucoes diferentes.
- Arquivos alterados: `internal/taskloop/acpinvoker.go`, `internal/taskloop/acpinvoker_test.go`
- Teste de regressao: `TestACPInvoker_Invoke_DerivesTaskEvidenceDir`
- Validacao: `go test ./internal/taskloop` passou; o evidence dir passou a ser resolvido como `evidence/task-<id>` quando o prompt referencia uma task concreta.

- ID: BUG-ACP-05
- Severidade: major
- Estado: fixed
- Causa raiz: o runtime nao serializava o payload completo de `runtime_init`, contava unknowns por kind distinto em vez de por evento e nunca chamava a telemetria ACP em producao.
- Arquivos alterados: `internal/runtime/runner.go`, `internal/runtime/acp_integration_test.go`, `internal/taskloop/acpinvoker.go`, `internal/taskloop/acpinvoker_test.go`
- Teste de regressao: `TestACPRunner_UnknownDrift`, `TestACPRunner_WithRealPersistence`, `TestACPInvoker_Invoke_LogsTelemetry`
- Validacao: `go test ./internal/runtime ./internal/taskloop ./internal/telemetry` passou; `runtime_init` agora inclui `launcher`, `command`, `args`, `sdk_version`, `npm_version`, o warning segue RF-05 e a sessao ACP passa a ser logada em `.agents/telemetry.log`.

## Comandos Executados
- `bash scripts/verify-go-mod.sh` -> falhou: arquivo ausente no repositorio (`bash: scripts/verify-go-mod.sh: No such file or directory`); validacao de toolchain seguiu via `go.mod`.
- `gofmt -w cmd/ai_spec_harness/task_loop.go internal/runtime/client/client.go internal/runtime/client/client_test.go internal/runtime/runner.go internal/runtime/acp_integration_test.go internal/taskloop/acpinvoker.go internal/taskloop/acpinvoker_test.go internal/taskloop/taskloop.go internal/taskloop/taskloop_test.go` -> sucesso
- `go test ./internal/runtime/... ./internal/taskloop/... ./cmd/ai_spec_harness ./internal/telemetry/...` -> sucesso, todos os pacotes `ok`
- `go test ./...` -> sucesso; destaque: `ok github.com/JailtonJunior94/ai-spec-harness/internal/runtime`, `ok .../internal/taskloop`, `ok .../cmd/ai_spec_harness`
- `go vet ./...` -> sucesso, sem output
- `make lint` -> sucesso; `golangci-lint` executado com `0 issues.`

## Riscos Residuais
- Nao ha risco residual funcional aberto dentro do escopo corrigido.
- O repositorio continua sem `scripts/verify-go-mod.sh`; isso e uma lacuna de tooling preexistente, nao um defeito introduzido por este bugfix.
