# Relatorio de Bugfix

- Total de bugs no escopo: 13
- Corrigidos: 10
- Testes de regressao adicionados: 13
- Pendentes: BUG-909 (bloqueado externamente)
- Estado final: blocked

## Bugs
- ID: BUG-909
- Severidade: critical
- Origem: RF-28, tarefa 9.0; finding de review
- Estado: blocked
- Causa raiz: a prova live anterior aceitava qualquer saída contendo `GOVERNANCE`; ela não demonstrava que a ação solicitada foi negada e terminada pelo CLI nativo.
- Arquivos alterados: tests/integration/hooks_live/live_test.go; tests/integration/hooks_live/README.md; .specs/prd-harness-quatro-clis-loop-aprovacao/bugfix_report.md; .specs/prd-harness-quatro-clis-loop-aprovacao/9.0_execution_report.md
- Teste de regressao: TestHooksLiveMatrixDispatchesThroughRealCLIs
- Validacao: cada célula instala e remove seu próprio projeto efêmero. Claude usa `--setting-sources project,local --permission-mode acceptEdits`; Codex usa `--skip-git-repo-check --approve-for-me`, sem bypass de trust; Copilot preserva o HOME autenticado, usa `--add-dir <fixture> --allow-all-tools` e não grava configuração do usuário; OpenCode usa `--dir --auto` e sentinela de carga do plugin. Pré-tool solicita arquivo `.go` e prova ausência da mutação; pós-tool exige a mutação e o diagnóstico do validador; session-end exige o diagnóstico de tarefa ativa sem `APPROVED`. `AISPEC_HOOKS_LIVE=1 make test-hooks-live` falhou com 1/12 célula aprovada (`opencode/post-tool`): Claude permitiu a mutação de pré-tool e não emitiu diagnósticos de pós-tool ou encerramento; Codex chamou e concluiu `PreToolUse`/`PostToolUse`, mas não negou a mutação de pré-tool, e não emitiu `SessionEnd`; Copilot excedeu o limite de 90s em cada célula, sem saída ou prova de hook; OpenCode reconheceu `apply_patch`, mas não extraiu o arquivo da chamada nativa, permitiu a mutação de pré-tool e não emitiu `session.idle` sob `opencode run`. Nenhuma negação de permissão, sandbox ou execução direta do plugin foi aceita como prova.

- ID: BUG-910
- Severidade: major
- Origem: RF-19, RF-27, tarefa 8.0; finding de review
- Estado: skipped
- Causa raiz: este achado aplicou precedência incorreta ao contrato observacional da tarefa 8.0. RF-27 do PRD é autoritativo e exige bloquear o encerramento da sessão.
- Arquivos alterados: internal/embedded/assets/.opencode/plugin/governance.js; tests/integration/opencode_session_end_dispatch_test.go; internal/embedded/opencode_plugin_scan_test.go
- Teste de regressao: TestOpenCodeGovernancePluginToolExecuteAfterIsSafeNoOp; TestOpenCodeGovernancePluginSessionIdleObservesActiveTaskWithoutApprovedVerdict
- Validacao: substituído pelo BUG-903, que preserva apenas `tool.execute.after` como observacional e bloqueia `session.idle`.

- ID: BUG-911
- Severidade: major
- Origem: RF-48, tarefa 11.0; finding de review
- Estado: fixed
- Causa raiz: os 18 relatórios concluídos usavam prosa iniciada por `comprovado:` ou qualificações equivalentes após o separador `->`; nenhuma dessas formas é uma linha de evidência válida por RF-48, e o parser estrito corretamente as recusou.
- Arquivos alterados: .specs/prd-harness-quatro-clis-loop-aprovacao/{1.0,2.0,3.0,4.1,4.2,4.3,4.4,4.5,4.6,4.7,4.8,5.0,6.0,7.0,8.0,9.0,10.0,11.0}_execution_report.md; .specs/prd-harness-quatro-clis-loop-aprovacao/bugfix_report.md
- Teste de regressao: TestFullPRD_HarnessQuatroClisLoopAprovacao_ChainIsClosed (preexistente; o comportamento do parser já era coberto e não exigiu teste novo)
- Validacao: `go run . check-traceability .specs/prd-harness-quatro-clis-loop-aprovacao` -> `OK: cadeia de rastreabilidade verificada — 63 requisitos, 18 tarefas.`

- ID: BUG-912
- Severidade: major
- Origem: RF-19, RF-27, tarefa 8.0; finding de review
- Estado: skipped
- Causa raiz: este achado também assumia que `session.idle` deveria ser advisory; isso conflita com RF-27.
- Arquivos alterados: internal/embedded/assets/.opencode/plugin/governance.js; tests/integration/opencode_session_end_dispatch_test.go
- Teste de regressao: TestOpenCodeGovernancePluginSessionIdleObservesActiveTaskWithoutApprovedVerdict
- Validacao: substituído pelo BUG-903 e pelo teste de regressão que exige bloqueio de encerramento.

- ID: BUG-901
- Severidade: critical
- Origem: RF-47, RF-48, RF-49, RF-51, RF-53, RF-56; finding de review
- Estado: fixed
- Causa raiz: fixtures de runtime e taskloop declaravam APPROVED sem o mapa de critérios exigido pelos adaptadores estritos.
- Arquivos alterados: internal/runtime/runner_cycle_test.go; internal/runtime/approval_cycle_e2e_test.go; internal/taskloop/approval_cycle_test.go; internal/taskloop/parity_test.go; internal/taskloop/approval_rawtext_contract_test.go
- Teste de regressao: TestACPRunnerConductsCycleApprovedFirstRound; TestE2EApprovalCycle_ApprovesFirstRound; TestExecuteConductsApprovalCycleFromTaskCriteria
- Validacao: go test ./internal/runtime ./internal/taskloop -count=1 passa.

- ID: BUG-902
- Severidade: major
- Origem: RF-27; finding de review
- Estado: fixed
- Causa raiz: a expressao aceitava APPROVED_WITH_REMARKS.
- Arquivos alterados: .agents/scripts/validate-session-end.sh; tests/integration/session_end_gate_test.go
- Teste de regressao: TestSessionEndGateBlocksApprovedWithRemarks
- Validacao: go test -tags=integration ./tests/integration/... passa.

- ID: BUG-903
- Severidade: major
- Origem: RF-27; finding de review
- Estado: fixed
- Causa raiz: session.idle registrava apenas aviso para rejeição do validador, violando RF-27.
- Arquivos alterados: internal/embedded/assets/.opencode/plugin/governance.js; tests/integration/opencode_session_end_dispatch_test.go; internal/embedded/opencode_plugin_scan_test.go
- Teste de regressao: TestOpenCodeGovernancePluginSessionIdleBlocksActiveTaskWithoutApprovedVerdict
- Validacao: o teste de integração exige `BLOCKED:GOVERNANCE BLOCKED`, saída não-zero e ausência de `ALLOWED` para tarefa ativa sem APPROVED.

- ID: BUG-904
- Severidade: major
- Origem: task-7.0; finding de review
- Estado: fixed
- Causa raiz: MergeOpenCodeConfig substituia o objeto permission existente.
- Arquivos alterados: internal/runtime/specs/opencode.go; internal/runtime/specs/opencode_config_test.go
- Teste de regressao: TestMergeOpenCodeConfigMergesExistingPermissionBlock
- Validacao: go test ./internal/runtime/specs passa.

- ID: BUG-905
- Severidade: major
- Origem: RF-42, task-4.2; finding de review
- Estado: fixed
- Causa raiz: ReviewerAdapter nao gravava a saida em endereco derivado da rodada.
- Arquivos alterados: internal/runtime/approval_adapters.go
- Teste de regressao: TestEvidenceRoundAddressesAreDistinctAndImmutable
- Validacao: go test ./internal/runtime ./internal/taskloop -count=1 passa com evidência de rodada preservada e fixtures completas.

- ID: BUG-906
- Severidade: critical
- Origem: RF-28, task-9.0; finding de review
- Estado: fixed
- Causa raiz: a matriz consultava somente o catálogo e `DispatchProven` retornava um mapa global de doze valores verdadeiros; nenhum resultado observado de uma entrada nativa instalada era exigido pelo gate de paridade.
- Arquivos alterados: tests/integration/hooks_live/live_test.go; internal/runtime/specs/dispatch_proof.go; internal/runtime/specs/parity_gate_test.go; Makefile; .github/workflows/hooks-live.yml
- Teste de regressao: TestHooksLiveMatrixDispatchesInstalledNativeEntrypoints; TestHooksLiveMatrixFailsWhenNativeValidatorIsMissing; TestHooksLiveMatrixFailsParityForNonBlockingInvocation
- Validacao: `make test-hooks-live` instala Claude, Codex, Copilot e OpenCode em diretório temporário, extrai cada comando da configuração nativa gerada e o executa com payload violador; para OpenCode, o shim carrega `opencode.json` e a entrada `.opencode/plugin/governance.js` instalada. As 12 células retornam bloqueio não-zero. A remoção do validador Claude falha a invocação nativa, e uma célula marcada como não bloqueante falha o gate de paridade. Não há CLI de agente, credencial ou rede no teste ou workflow.

- ID: BUG-907
- Severidade: major
- Origem: task-4.8; finding de review
- Estado: fixed
- Causa raiz: E2EParitySuite e seu registrador estavam ausentes.
- Arquivos alterados: internal/parity/e2e_parity_test.go
- Teste de regressao: TestE2EParitySuite
- Validacao: go test -tags=integration ./internal/parity passa.

- ID: BUG-908
- Severidade: major
- Origem: RF-48, task-11.0; finding de review
- Estado: fixed
- Causa raiz: Criterion.HasEvidence aceitava qualquer nome com espaços antes de `->` como comando.
- Arquivos alterados: internal/traceability/traceability.go; internal/traceability/traceability_test.go; cmd/ai_spec_harness/check_traceability_test.go
- Teste de regressao: TestValidate_CriterionRejectsArbitraryEvidenceProse; TestCriterionHasEvidenceAcceptsOnlyRF48Forms
- Validacao: go test ./internal/traceability ./cmd/ai_spec_harness -count=1 passa.

- ID: BUG-913
- Severidade: major
- Origem: RF-19, RF-27, tarefa 8.0; finding de `make test`
- Estado: fixed
- Causa raiz: o handler OpenCode `tool.execute.after` somente emitia um aviso e nao invocava o hook canonico declarado `validate-governance.sh`; a matriz de instalacao corretamente recusava a celula por falta de invocacao estrutural.
- Arquivos alterados: internal/embedded/assets/.opencode/plugin/governance.js; tests/integration/opencode_session_end_dispatch_test.go; .specs/prd-harness-quatro-clis-loop-aprovacao/bugfix_report.md
- Teste de regressao: TestOpenCodeGovernancePluginToolExecuteAfterObservesValidatorWithoutBlocking
- Validacao: `go test ./internal/install/... -run TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator -v -count=1` e `make test` passam. O teste de integracao usa validador que registra marcador e retorna 1, provando invocacao sem bloquear o fluxo.

## Comandos Executados
- gofmt -w arquivos Go alterados -> sucesso
- go test ./internal/runtime ./internal/taskloop ./internal/traceability ./internal/install -count=1 -> sucesso
- go test -tags=integration ./tests/integration/... -count=1 -> sucesso
- go test -tags=hooks_live ./tests/integration/hooks_live -count=1 -> bloqueado externamente: requer autenticação dos quatro CLIs, trust Codex e pasta confiável Copilot
- bash scripts/sync-skills.sh -> sucesso
- bash scripts/sync-hooks.sh -> sucesso
- bash scripts/check-scripts-sync.sh -> sucesso
- bash scripts/check-hooks-sync.sh -> sucesso
- bash .agents/scripts/validate-bugfix-evidence.sh --rf RF-47 --rf RF-48 --rf RF-27 .specs/prd-harness-quatro-clis-loop-aprovacao/bugfix_report.md -> sucesso
- make test -> sucesso
- make integration -> sucesso
- make build -> sucesso
- make vet -> sucesso
- make lint -> falhou por três achados preexistentes fora deste escopo: `internal/runtime/acp_opencode_telemetry_test.go:27` (S1016), `internal/specdrift/specdrift_test.go:227` (S1039) e `internal/adapters/adapters.go:234` (unused).
- bash scripts/check-skills-sync.sh && bash scripts/check-hooks-sync.sh && bash scripts/check-scripts-sync.sh -> sucesso
- AISPEC_HOOKS_LIVE=1 make test-hooks-live -> falhou com 1/12 células reais aprovadas: `claude/pre-tool` permitiu a mutação; `claude/post-tool` não emitiu diagnóstico; `claude/session-end` não emitiu bloqueio; `codex/pre-tool` chamou o hook mas permitiu a mutação; `codex/post-tool` não chegou à mutação por falha de patch do modelo, sem diagnóstico de hook; `codex/session-end` não emitiu o evento; `copilot/{pre-tool,post-tool,session-end}` excederam 90s sem saída; `opencode/pre-tool` permitiu a mutação; `opencode/post-tool` aprovou; `opencode/session-end` não emitiu bloqueio.
- make integration -> sucesso
- make test -> sucesso
- AISPEC_HOOKS_LIVE=1 make test-hooks-live -> falhou: 1/12 célula aprovada (`opencode/post-tool`). Claude 2.1.236 executou as mutações com `acceptEdits`, mas não disparou os diagnósticos nativos (3); Codex 0.154.0 executou em `workspace-write`, mostrou `hook: PreToolUse`/`PostToolUse` e permitiu a mutação sem diagnóstico (3); Copilot 1.0.83 autenticado no HOME real executou as mutações sem diagnóstico (3); OpenCode 1.18.30 carregou o plugin, mas pre-tool usou `apply_patch` não reconhecido e session-end não chamou o validador (2). Não há resultado nativo fabricado nem evidência baseada em negação de ferramenta, sandbox ou permissão.
- make check-spec-paths -> sucesso; 9 OK, 0 FAIL
- ai-spec check-spec-drift .specs/prd-harness-quatro-clis-loop-aprovacao/tasks.md -> sucesso; sem drift detectado
- bash scripts/sync-skills.sh && bash scripts/sync-hooks.sh -> sucesso
- bash scripts/check-scripts-sync.sh && bash scripts/check-hooks-sync.sh -> sucesso
- go run . check-traceability .specs/prd-harness-quatro-clis-loop-aprovacao -> OK: cadeia de rastreabilidade verificada — 63 requisitos, 18 tarefas.
- go test ./internal/traceability ./cmd/ai_spec_harness -count=1 -> sucesso
- go run . check-spec-drift .specs/prd-harness-quatro-clis-loop-aprovacao -> OK: sem drift detectado.
- go test ./internal/install/... -run TestMandatoryMatrixWrittenConfigContainsNativeKeyAndValidator -v -count=1 -> sucesso
- make test -> sucesso
- bash .agents/scripts/validate-bugfix-evidence.sh --rf RF-48 .specs/prd-harness-quatro-clis-loop-aprovacao/bugfix_report.md -> sucesso
- bash .agents/scripts/validate-task-evidence.sh .specs/prd-harness-quatro-clis-loop-aprovacao/9.0_execution_report.md -> passa quando o relatório está blocked; o validador exige prova física somente para Estado: done.
- go test ./tests/integration/hooks_live -count=1 -> sucesso.
- bash .agents/scripts/validate-bugfix-evidence.sh --rf RF-27 --rf RF-28 .specs/prd-harness-quatro-clis-loop-aprovacao/bugfix_report.md -> sucesso.

## Riscos Residuais
- A prova nativa continua pendente apesar dos quatro CLIs autenticados. Claude 2.1.236 e Copilot 1.0.83 executam as ferramentas no fixture, mas não produzem os diagnósticos dos hooks instalados; Codex 0.154.0 carrega ambos os formatos de hooks, porém os conclui sem bloquear ou diagnosticar a mutação; OpenCode 1.18.30 só documenta `--auto` como autorização não interativa e a célula pre-tool usa `apply_patch`, ausente de `MUTATING_TOOLS`, enquanto `session.idle` não dispara no encerramento de `opencode run`. Não existe outra flag não interativa documentada nesta versão do OpenCode para forçar a execução dessas ferramentas/evento. O workflow noturno não pode aprovar essas células até que os adapters/configurações nativos sejam corrigidos.
- Os validadores de tarefa permanecem bloqueados pela divergência de `final_state_sha256` dos resultados históricos contra a árvore de trabalho acumulada e não commitada; nenhum hash de resultado foi inventado ou regravado.
- BUG-909 permanece bloqueado porque 11 das 12 células nativas não produziram o efeito exigido. Codex não recebeu bypass de trust, Copilot não teve sua configuração de usuário alterada e os quatro CLIs receberam uma tentativa real com permissão suficiente quando a sessão iniciou; isso bloqueia o fechamento de RF-27/RF-28, não os testes determinísticos.
