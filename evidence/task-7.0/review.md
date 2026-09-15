# Relatório de Review (modo --auto-review)

- Veredito: APPROVED_WITH_REMARKS
- Alvo revisado: arquivos alterados da tarefa 7.0 (internal/skills/skills.go, internal/runtime/specs/spec.go, internal/runtime/specs/codex.go, internal/runtime/specs/gemini.go, internal/runtime/specs/opencode.go, internal/runtime/specs/registry.go, internal/runtime/specs/literal_scan_test.go, internal/runtime/specs/registry_test.go, internal/runtime/specs/codex_test.go, internal/runtime/specs/gemini_test.go, internal/runtime/specs/spec_test.go, internal/runtime/specs/opencode_test.go, internal/runtime/specs/opencode_config_test.go, internal/runtime/specs/pin_gate_test.go, internal/runtime/specs/format_gate_test.go, internal/detect/agent.go, internal/detect/detect.go, internal/detect/agent_test.go, internal/runtime/runner.go, internal/install/install.go, internal/install/install_opencode_test.go, internal/metrics/metrics.go, internal/metrics/budget_coverage_test.go, .agents/normalization-rules.yaml, internal/runtime/events/normalization-rules.yaml, internal/runtime/events/mirror_gate_test.go, internal/runtime/acp_opencode_test.go, internal/runtime/probe/probe_test.go, cmd/ai_spec_harness/task_loop.go, cmd/ai_spec_harness/task_loop_test.go, cmd/ai_spec_harness/flags.go, cmd/ai_spec_harness/install.go, cmd/ai_spec_harness/verify.go, docs/cli-schema.json)
- Refs carregadas: agent-governance (base), go-implementation (skill de linguagem)

## Mapa de Critérios de Aceite
- [atendido] `go build ./... && go vet ./... && go test ./... -count=1` verde ao final da tarefa -> evidence/task-7.0/build.log, evidence/task-7.0/vet.log, evidence/task-7.0/test.log (exit=0, 53 pacotes ok)
- [atendido] Teste prova que o argv gerado para o OpenCode é `opencode acp --cwd <repo> ...` — subcomando antes de qualquer argumento iniciado por hífen -> TestOpenCodeArgvSubcommandBeforeFlags (internal/runtime/specs/opencode_test.go) e TestEnsureAvailable_OpenCode_DirectBinaryArgvHasSubcommandFirst (internal/runtime/probe/probe_test.go) -> pass
- [atendido] Teste da validação de formato falha para uma spec artificial em que um item de FixedArgs aparece depois de um argumento com hífen -> TestValidateFixedArgsFormatRejectsPositionalAfterFlag e TestNewSpecPanicsOnInvalidFixedArgsFormat (internal/runtime/specs/format_gate_test.go) -> pass
- [atendido] Teste prova que o launcher de fallback usa versão pinada; gate de varredura reprova `@latest` em qualquer launcher do repositório -> TestOpenCodeFallbackPinnedVersion e TestNoLauncherUsesLatestVersion (internal/runtime/specs/pin_gate_test.go) -> pass
- [atendido] `ai-spec-harness install .` detecta o OpenCode sem flag `--tools`, por cada um dos três sinais isoladamente -> TestBinaryAgentDetectorDetect/deve_detectar_OpenCode_apenas_pelo_binario_no_PATH, TestDetectOpenCodeByHomeConfigDirOnly, TestBinaryAgentDetectorDetect/deve_detectar_OpenCode_apenas_pelo_sinal_de_projeto (internal/detect/agent_test.go) -> pass
- [atendido] Nenhuma política de detecção opt-in por agente permanece: gate de varredura verde -> internal/detect/agent.go:163-167 (bloco de filtro opt-in do Gemini removido); TestNoAgentListLiteralOutsideRegistry (internal/runtime/specs/literal_scan_test.go) -> pass
- [atendido] Após `install`, o `opencode.json` do projeto contém apenas o bloco `permission` acrescentado; `$schema` e campos preexistentes preservados byte a byte -> TestInstall_OpenCode_WritesOnlyPermissionBlock e TestMergeOpenCodeConfigPreservesSchemaAndExistingFields -> pass
- [atendido] Teste falha se o instalador escrever `skills.paths` ou `instructions` -> TestMergeOpenCodeConfigNeverWritesSkillsOrInstructions e TestInstall_OpenCode_WritesOnlyPermissionBlock (asserção explícita de ausência) -> pass
- [atendido] Reexecutar `install` produz `opencode.json` idêntico (idempotência) -> TestInstall_OpenCode_IdempotentReinstall e TestMergeOpenCodeConfigIsIdempotent -> pass
- [atendido] O plugin de governança aparece em `.opencode/plugin/` e nenhuma entrada de configuração o referencia -> TestInstall_OpenCode_DepositsPluginDir; internal/install/install.go:1161-1194 (installOpenCode não escreve referência de config para o plugin) -> pass
- [atendido] Um projeto somente-OpenCode e um projeto somente-Claude têm o mesmo conjunto de validadores canônicos instalados -> TestInstall_OpenCode_ShipsCanonicalValidators -> pass
- [atendido] Resolução de janela: casamento exato, maior prefixo, modelo desconhecido e modelo ausente, nunca maior que a entrada correspondente -> TestResolveOpenCodeWindowExactMatch, TestResolveOpenCodeWindowLongestPrefixMatch, TestResolveOpenCodeWindowUnknownModelFallsBackConservative, TestResolveOpenCodeWindowEmptyModelFallsBackConservative, TestResolveOpenCodeWindowNeverExceedsTableEntry -> pass
- [atendido] Gate de cobertura de orçamento fica vermelho quando agente de janela grande fica sem entrada e quando existe chave órfã -> TestToolBudgetsLargeCoverageFailsOnMissingAgent e TestToolBudgetsLargeCoverageFailsOnOrphanKey (internal/metrics/budget_coverage_test.go) -> pass
- [atendido] `inherit_common` contém o OpenCode nos dois arquivos de regras de normalização, gate de sincronia verde -> .agents/normalization-rules.yaml:20-22, internal/runtime/events/normalization-rules.yaml:20-22; TestInheritCommonMirrorsBetweenSourceAndEmbedded (internal/runtime/events/mirror_gate_test.go) -> pass
- [atendido] Paridade observacional: eventos, tool-calls e execution_report com a mesma estrutura dos demais agentes -> TestACPIntegration_OpenCode_ToolCallsAndReport e TestACPIntegration_OpenCode_ActivityWatchdog (internal/runtime/acp_opencode_test.go) -> pass
- [atendido] Não-regressão: para os três agentes atuais, argv e janela resolvida são byte-idênticos aos anteriores -> TestResolveWindowNilResolverIsByteIdenticalForExistingAgents (internal/runtime/specs/opencode_test.go); go test ./... -count=1 sem alteração de assertivas pré-existentes -> pass

## Achados
- Severidade: medium
- Arquivo: internal/runtime/specs/registry.go
- Linha: 166-174
- Impacto: o `adrPath` do OpenCode aponta para `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-003-opencode-acp-subcomando.md` (escopo do PRD), enquanto os três agentes existentes apontam para `.specs/adr/0NN-*.md` (escopo canônico do repositório). O arquivo existe e a referência é válida; a inconsistência é apenas de convenção de caminho.
- Dica de correção: promover a ADR-003 para `.specs/adr/` em um passo de fechamento de PRD (fora do escopo desta tarefa) e atualizar `adrPath` correspondentemente.

- Severidade: low
- Arquivo: internal/runtime/specs/opencode.go
- Linha: 23-31
- Impacto: `DefaultOpenCodePermission()` declara um placeholder mínimo e defensável (nega apenas padrões finos categoricamente perigosos em `bash`, sem "deny" total de ferramenta nem "ask"), mas o conteúdo definitivo do bloco de permissão e do plugin de bloqueio pertence à tarefa 8.0.
- Dica de correção: revisar/estender `DefaultOpenCodePermission()` na tarefa 8.0 junto com o plugin de enforcement, mantendo as duas proibições já testadas (RF-20): sem "deny" total de ferramenta necessária, sem "ask".

## Arquivos Revisados
- internal/skills/skills.go
- internal/runtime/specs/spec.go
- internal/runtime/specs/codex.go
- internal/runtime/specs/gemini.go
- internal/runtime/specs/opencode.go
- internal/runtime/specs/registry.go
- internal/runtime/specs/literal_scan_test.go
- internal/runtime/specs/registry_test.go
- internal/runtime/specs/codex_test.go
- internal/runtime/specs/gemini_test.go
- internal/runtime/specs/spec_test.go
- internal/runtime/specs/opencode_test.go
- internal/runtime/specs/opencode_config_test.go
- internal/runtime/specs/pin_gate_test.go
- internal/runtime/specs/format_gate_test.go
- internal/detect/agent.go
- internal/detect/detect.go
- internal/detect/agent_test.go
- internal/runtime/runner.go
- internal/install/install.go
- internal/install/install_opencode_test.go
- internal/metrics/metrics.go
- internal/metrics/budget_coverage_test.go
- .agents/normalization-rules.yaml
- internal/runtime/events/normalization-rules.yaml
- internal/runtime/events/mirror_gate_test.go
- internal/runtime/acp_opencode_test.go
- internal/runtime/probe/probe_test.go
- cmd/ai_spec_harness/task_loop.go
- cmd/ai_spec_harness/task_loop_test.go
- cmd/ai_spec_harness/flags.go
- cmd/ai_spec_harness/install.go
- cmd/ai_spec_harness/verify.go
- docs/cli-schema.json

## Riscos Residuais
- ADR do OpenCode ainda reside em `.specs/prd-harness-quatro-clis-loop-aprovacao/` (ver achado medium acima).
- Conteúdo definitivo do bloco `permission` e do plugin de enforcement é da tarefa 8.0; o placeholder atual é testado e seguro, mas não é o desenho final.
- Nomes de hooks nativos do OpenCode (`tool.execute.before`, `tool.execute.after`, `session.idle`) são baseados na evidência disponível (V-04, discovery) para pré-ferramenta; os nomes de pós-ferramenta e encerramento de sessão não têm prova de disparo real registrada nesta tarefa (fica para a tarefa 8.0, que implementa o handshake e o plugin).

## Validações Executadas
- `go build ./...` -> pass (evidence/task-7.0/build.log)
- `go vet ./...` -> pass (evidence/task-7.0/vet.log)
- `go test ./... -count=1` -> pass, 53 pacotes ok (evidence/task-7.0/test.log)
- `go test ./internal/runtime/specs/... -count=1 -v` -> pass (specs package incluindo opencode_test.go, opencode_config_test.go, pin_gate_test.go, format_gate_test.go)
- `go test ./internal/detect/... -count=1 -v` -> pass (três sinais de detecção do OpenCode)
- `go test ./internal/install/... -run OpenCode -count=1` -> pass
- `go test ./internal/metrics/... -run ToolBudgetsLarge -count=1 -v` -> pass
- `go test ./internal/runtime/events/... -run OpenCode -v -count=1` e `-run InheritCommon` -> pass
- `go test ./internal/runtime -run OpenCode -v -count=1` -> pass
- `go test ./internal/runtime/probe/... -run OpenCode -v -count=1` -> pass
