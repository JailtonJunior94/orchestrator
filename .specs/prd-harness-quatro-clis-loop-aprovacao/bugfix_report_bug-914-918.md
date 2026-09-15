# Relatorio de Bugfix

- Total de bugs no escopo: 5
- Corrigidos: 3
- Testes de regressao adicionados: 5
- Pendentes: BUG-915 e BUG-916 dependem de mecanismos externos de runtime ainda nao comprovados.
- Estado final: blocked

## Bugs
- ID: BUG-914
- Severidade: critical
- Origem: RF-27, RF-28, tarefa 9.0; finding de review
- Estado: fixed
- Causa raiz: o workflow nativo era agendado e explicitamente nao era gate de merge.
- Arquivos alterados: `.github/workflows/hooks-live.yml`; `Makefile`; `tests/integration/hooks_live/README.md`.
- Teste de regressao: `TestHooksLiveMatrixDispatchesThroughRealCLIs` continua fail-closed para ausencia de prerequisitos ou efeito nativo.
- Validacao: workflow agora executa em `pull_request` e `push` para `main`, usa o environment protegido `hooks-live`, exige os segredos antes de instalar os CLIs e nao pertence a `make test`.

- ID: BUG-915
- Severidade: major
- Origem: RF-28, tarefa 9.0; finding de review
- Estado: blocked
- Causa raiz: a configuracao global do Copilot era mutada no workflow para declarar trust; isso nao e uma prova do mecanismo nativo documentado.
- Arquivos alterados: `.github/workflows/hooks-live.yml`; `tests/integration/hooks_live/README.md`.
- Teste de regressao: `TestHooksLiveMatrixDispatchesThroughRealCLIs` usa somente `--add-dir <fixture>`, que e o mecanismo documentado para confiar e carregar configuracao do diretorio.
- Validacao: o workflow nao grava mais `~/.copilot/config.json`; a execucao local real de `copilot` 1.0.83 excedeu 90 segundos sem output nas tres celulas.

- ID: BUG-916
- Severidade: major
- Origem: RF-27, RF-28, tarefa 9.0; finding de review
- Estado: blocked
- Causa raiz: o payload real de `codex exec` usa `input.arguments.patchText`; o extrator anterior nao descia em `arguments` nem reconhecia `patchText`.
- Arquivos alterados: `.agents/lib/parse-hook-input.sh`; `scripts/lib/parse-hook-input.sh`; `internal/embedded/assets/.agents/lib/parse-hook-input.sh`; `tests/integration/portability_test.go`.
- Teste de regressao: `TestPortability_HookGateCrossCLI/codex-apply-patch-go-ok`.
- Validacao: `codex` 0.154.0 acionou `PreToolUse` e `PostToolUse`, mas concluiu ambos e permitiu a mutacao; `codex exec` nao emitiu `SessionEnd`. Nao foi usado bypass de trust.

- ID: BUG-917
- Severidade: major
- Origem: RF-27, tarefa 9.0; finding de review
- Estado: fixed
- Causa raiz: `session.idle` foi registrado como chave de hook, mas o contrato documentado do OpenCode 1.18.30 entrega eventos de sessao exclusivamente pelo handler `event`.
- Arquivos alterados: `internal/embedded/assets/.opencode/plugin/governance.js`; `internal/install/hooks_parity_matrix_test.go`; `tests/integration/opencode_session_end_dispatch_test.go`; `internal/embedded/opencode_plugin_scan_test.go`.
- Teste de regressao: `TestOpenCodeGovernancePluginSessionIdleBlocksActiveTaskWithoutApprovedVerdict`; `TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator`; `TestHooksLiveMatrixDispatchesThroughRealCLIs/opencode/session-end`.
- Validacao: `AISPEC_HOOKS_LIVE=1 go test -tags=hooks_live -run 'TestHooksLiveMatrixDispatchesThroughRealCLIs/opencode/session-end' -count=1 ./tests/integration/hooks_live` passa contra OpenCode 1.18.30; a matriz completa passa as tres celulas OpenCode.

- ID: BUG-918
- Severidade: major
- Origem: RF-27, RF-28, tarefa 9.0; finding de review
- Estado: fixed
- Causa raiz: a configuracao Claude local nao declarava `Stop`, e o exit 1 do `PreToolUse` e apenas reportado como erro pelo Claude 2.1.236, sem impedir a escrita.
- Arquivos alterados: `.claude/settings.local.json`; `.claude/hooks/validate-preload.sh`; `internal/embedded/assets/.claude/hooks/validate-preload.sh`; `tests/integration/hooks_live/live_test.go`.
- Teste de regressao: `TestHooksLiveMatrixDispatchesThroughRealCLIs/claude/(pre-tool|post-tool|session-end)`.
- Validacao: os tres subtestes Claude passam com `--include-hook-events --output-format stream-json`; o hook de pre-tool retorna exit 2, que impede a mutacao na CLI atual.

## Comandos Executados
- `go test -tags=integration ./tests/integration/... -run 'TestPortability_HookGateCrossCLI|TestOpenCodeGovernancePluginBlocksApplyPatchTextWhenPrerequisiteMissing|TestOpenCodeGovernancePluginSessionIdle' -count=1` -> sucesso.
- `go test ./internal/install/... -run TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator -count=1` -> sucesso.
- `make test` -> sucesso.
- `make integration` -> sucesso.
- `make build` -> sucesso.
- `make vet` -> sucesso.
- `make check-spec-paths check-scripts-sync check-hooks-sync` -> sucesso.
- `AISPEC_HOOKS_LIVE=1 make test-hooks-live` -> falhou com 5/12 celulas aprovadas antes da correcao Claude; apos a correcao, os tres subtestes Claude passaram isoladamente.
- `go test ./... -race -count=1` -> sucesso; a recuperacao de memoria com 1000 fatos ficou abaixo de 200ms apos substituir a busca de blocos por varredura linear.
- `make check-mocks && make test-check-mocks` -> sucesso com Mockery v3.8.0; o contrato suportado nao oferece `--dry-run`, portanto o gate compara uma geracao isolada com os mocks versionados.
- `AISPEC_HOOKS_LIVE=1 make test-hooks-live` -> 6/12 celulas aprovadas: Claude 3/3, OpenCode 3/3; Codex 0/3 e Copilot 0/3.

## Riscos Residuais
- A matriz nao tem 12/12 provas nativas. Permanecem bloqueadas as tres celulas Codex porque `hooks/list` nao expoe hashes de hooks de projeto antes da revisao interativa `/hooks`, unico mecanismo documentado para persistir o trust; o teste nao modifica configuracao permanente nem usa bypass. As tres celulas Copilot executam a mutacao sob `--add-dir`, mas a documentacao da CLI 1.0.83 limita esse mecanismo a `.github/skills` e `.github/agents`; nao ha contrato documentado de hooks de projeto para despachar `.github/hooks/governance.json`.
- Nenhuma celula bloqueada foi marcada como comprovada, e nenhum shim, negacao de permissao ou evidencia sintetica foi usado como prova nativa.
