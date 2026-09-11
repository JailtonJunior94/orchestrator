# Relatorio de Bugfix

- Total de bugs no escopo: 2
- Corrigidos: 2
- Testes de regressao adicionados: 2
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-001
- Severidade: major
- Origem: RF-19, RF-27, RF-62 (`prd.md`); requisito explicito da tarefa 8.0 ("Só o ponto de
  pré-ferramenta bloqueia — é o único que impede algo ainda não feito. Os outros dois pontos
  canônicos são cobertos, mas observacionais") e subtarefa 8.7 (`task-8.0-enforcement-opencode.md`);
  review adversarial da tarefa 8.0 (Achado 3, severidade high → major)
- Estado: fixed
- Causa raiz: o handler `"session.idle"` do plugin de governança (`governance.js`) reusava
  `throw new Error(...)` do mesmo jeito que o handler `"tool.execute.before"`, tratando os dois
  pontos canônicos como igualmente bloqueantes. O código nunca foi diferenciado depois que o
  requisito de "apenas pré-ferramenta bloqueia" foi fixado na tarefa 8.0 — regressão de
  design, não de digitação. RF-27 exige que o gate de encerramento *exista e seja registrado*
  (satisfeito pelo script canônico `.agents/scripts/validate-session-end.sh`, testado
  isoladamente em `tests/integration/session_end_gate_test.go` via código de saída), não que o
  *dispatch do plugin* aborte a sessão — essa segunda camada é proibida pelo requisito explícito
  da tarefa 8.0.
- Arquivos alterados:
  - `internal/embedded/assets/.opencode/plugin/governance.js` (linhas 71-74, 161-166): novo
    helper `sessionEndAdvisoryMessage(reason)` e troca de `throw new Error(...)` por
    `console.warn(...)` no handler `session.idle`, preservando o texto do motivo da reprovação
    (nenhuma perda de informação, apenas deixa de abortar a sessão).
  - `tests/integration/opencode_session_end_dispatch_test.go`: renomeado e reescrito
    `TestOpenCodeGovernancePluginSessionIdleBlocksActiveTaskWithoutApprovedVerdict` para
    `TestOpenCodeGovernancePluginSessionIdleRecordsActiveTaskWithoutApprovedVerdictWithoutBlocking`
    — agora asserta saída zero (`ALLOWED`) e a presença do texto de advertência
    `"GOVERNANCE ADVISORY at session.idle"` no output combinado, em vez de saída não-zero e
    `BLOCKED:`. O segundo teste (`SessionIdleAllowsWhenNoActiveTask`) ganhou asserção extra de
    ausência de advertência quando não há reprovação.
  - `internal/embedded/opencode_plugin_scan_test.go`: novo teste
    `TestOpenCodeGovernancePluginSessionIdleNeverThrows` — varre o corpo do handler
    `session.idle` embutido (`go:embed`) e falha se houver `throw` nele, travando a regressão
    sem depender de `node` no PATH.
  - Confirmado sem necessidade de alteração: `internal/runtime/acp_opencode_telemetry_test.go`
    (não exercita `session.idle`, apenas a rejeição por pré-condição de handshake — nenhum
    ajuste necessário) e `tests/integration/session_end_gate_test.go` (testa o script
    canônico `validate-session-end.sh` isoladamente por código de saída, que continua sendo o
    mecanismo real do gate RF-27 — nenhum ajuste necessário).
- Teste de regressao:
  - `go test -tags integration ./tests/integration/... -run TestOpenCodeGovernancePluginSessionIdleRecordsActiveTaskWithoutApprovedVerdictWithoutBlocking -v`
  - `go test -tags integration ./tests/integration/... -run TestOpenCodeGovernancePluginSessionIdleAllowsWhenNoActiveTask -v`
  - `go test ./internal/embedded/... -run TestOpenCodeGovernancePluginSessionIdleNeverThrows -v`
- Validacao: os três testes acima passam apos a correção; falhavam antes (o primeiro esperava
  `BLOCKED:`/exit≠0, que não existe mais; o novo teste de varredura falharia contra o código
  anterior por conter `throw`).

- ID: BUG-003
- Severidade: major
- Origem: RF-19, RF-20 (`prd.md`); subtarefa 8.7/8.4 e critério "custo em três camadas" da
  tarefa 8.0 (`techspec.md` §Enforcement do OpenCode); review adversarial da tarefa 8.0
  (Achado 2, severidade medium → major, dado o potencial de bypass total)
- Estado: fixed
- Causa raiz: `MUTATING_TOOLS` é uma lista fechada (`bash`, `write`, `edit`, `multiedit`,
  `patch`), inferida da evidência real da OpenCode SDK 1.18.x e já documentada em
  `8.0_execution_report.md` (linha 103) como uma suposição não exaustivamente confirmada.
  O caminho `if (!tool || !MUTATING_TOOLS.has(tool)) { return }` tratava "tool não reconhecida"
  exatamente igual a "tool conhecidamente não-mutante", sem nenhum sinal operacional — se o
  OpenCode adicionar uma nova tool nativa de mutação em versão futura, o bypass seria total e
  silencioso.
  Investigação: dentro do escopo desta tarefa não há acesso a uma sessão OpenCode real para
  enumerar exaustivamente as tools nativas da SDK; a suposição documentada em
  `8.0_execution_report.md` (`write`, `edit`, `multiedit`, `patch` como as únicas tools
  nativas de mutação, além de `bash`) é a base de evidência disponível. Optou-se por
  **ambas** as mitigações sugeridas no achado, por serem complementares e de custo baixo:
  (a) log de aviso operacional quando uma tool fora do conjunto conhecido passa sem validação
  e (b) teste que trava a composição exata de `MUTATING_TOOLS`, forçando revisão manual
  deliberada quando uma tool nova precisar ser adicionada.
- Arquivos alterados:
  - `internal/embedded/assets/.opencode/plugin/governance.js` (linhas 9-32, 136-144): novo
    `unrecognizedToolWarned` (Set de dedupe) e `warnUnrecognizedToolOnce(tool)`; o handler
    `tool.execute.before` agora separa o caso `!tool` (retorno silencioso, inalterado) do caso
    "tool presente mas fora de `MUTATING_TOOLS`" (emite aviso uma única vez por nome de tool via
    `console.warn`, depois retorna sem validar — comportamento de bypass preservado, mas agora
    visível). Nenhuma tool nova foi inventada nem adicionada ao conjunto de mutação — apenas a
    visibilidade do bypass existente.
  - `tests/integration/opencode_plugin_dispatch_test.go`: novo teste
    `TestOpenCodeGovernancePluginWarnsOnceForUnrecognizedTool` — dispara
    `tool.execute.before` duas vezes com uma tool hipotética fora da lista
    (`future_mutating_tool`, usada apenas como fixture de teste, não uma feature nova) e
    asserta exit 0 (bypass continua permitido, comportamento não mudou), presença da mensagem
    `"not in the known mutating-tools set"` e exatamente um aviso (dedupe por nome de tool).
  - `internal/embedded/opencode_plugin_scan_test.go`: novo teste
    `TestOpenCodeGovernancePluginMutatingToolsListIsLockedToInvestigatedSet` — trava a
    declaração textual de `MUTATING_TOOLS` no asset embutido contra a composição investigada
    e assert a presença da string de aviso, sem depender de `node`.
- Teste de regressao:
  - `go test -tags integration ./tests/integration/... -run TestOpenCodeGovernancePluginWarnsOnceForUnrecognizedTool -v`
  - `go test ./internal/embedded/... -run TestOpenCodeGovernancePluginMutatingToolsListIsLockedToInvestigatedSet -v`
  - `go test -tags integration ./tests/integration/... -run TestOpenCodeGovernancePluginIgnoresNonMutatingTools -v` (nao-regressao: tool
    conhecidamente não-mutante `read` continua permitida sem bloqueio)
- Validacao: os testes acima passam apos a correção. O teste de trava falharia se
  `MUTATING_TOOLS` for alterado sem tocar o teste (força revisão manual, conforme pedido no
  achado), e o teste de aviso falharia contra o código anterior (nenhum `console.warn` era
  emitido para tool desconhecida).

## Comandos Executados

- `bash -c 'source .agents/lib/check-invocation-depth.sh && echo OK'` -> depth 0→1, dentro do
  limite (max 2)
- `go build ./...` -> ok, sem erros
- `go vet ./...` -> ok, sem erros
- `go test ./internal/embedded/... -run OpenCode -v -count=1` -> 4 testes, todos PASS (inclui os
  2 novos testes de trava/varredura)
- `go test ./internal/runtime/... -run OpenCode -v -count=1` -> 41 testes, todos PASS
- `go test -tags integration ./tests/integration/... -run "OpenCode|SessionEnd" -v -count=1` ->
  26 testes, todos PASS (inclui os 3 testes de regressao novos/reescritos)
- `go test ./... -count=1` -> 2903 testes, todos PASS (suite completa sem a tag `integration`)
- `go test -tags integration ./... -count=1` -> 3199 PASS, 5 FAIL, 4 SKIP — as 5 falhas e a
  falha de build do pacote `internal/parity` são **pre-existentes e fora do escopo deste
  bugfix** (ver Riscos Residuais); nenhuma delas toca `governance.js`, os pontos canonicos do
  OpenCode ou os arquivos de teste alterados/criados nesta correção. Confirmado via
  `git stash`/`git stash pop` (apenas arquivos rastreados; os 4 arquivos desta correção são
  novos/não-rastreados e não foram afetados) que o pacote `internal/runtime/specs` já não
  compila no HEAD commitado sem o acumulo de trabalho não commitado desta branch — condicao
  pre-existente da branch de longa duracao, nao introduzida por esta correcao.
- `node --version` -> v26.8.2 (disponivel no PATH; os testes de integracao que exigem `node`
  rodaram de fato, nao foram pulados)

## Riscos Residuais

- `internal/parity` (`e2e_parity_test.go`) falha de build por símbolo `E2EParitySuite`
  indefinido — pré-existente, não relacionado ao plugin OpenCode; fora do escopo dos bugs
  BUG-001/BUG-003.
- `TestIntegration_7_5_ScenarioG_Monorepo` e `TestIntegration_7_5_ProbeNonFatal_BinaryAbsent`
  (`internal/install`) falham com `opencode governance plugin runtime "bash" not found on
  PATH` — comportamento de um cenário de teste com PATH restrito (não relacionado à mudança
  em `governance.js`); pré-existente.
- `TestClaudeCrossWaveSmokeE2E` e `TestClaudeAutoReviewBlocksOnHardIssueE2E`
  (`tests/integration/claude_2026_e2e_test.go`) falham por divergência em `ReviewStatus`;
  não tocam OpenCode nem os arquivos desta correção; pré-existente.
- A suposição de que `bash`, `write`, `edit`, `multiedit` e `patch` são as únicas tools
  nativas do OpenCode que mutam o repositório permanece uma suposição documentada (não
  exaustivamente confirmada contra uma sessão OpenCode real autenticada — já registrado como
  risco em `8.0_execution_report.md`). A mitigação desta correção (aviso operacional +
  teste de trava) não elimina esse risco de fundo, apenas garante que uma futura tool
  nativa de mutação fora da lista não passe **em silêncio total** e que qualquer alteração
  da lista force revisão humana deliberada via falha do teste de trava.
- Nenhuma alteração foi commitada nesta sessão (fora do escopo do bugfix); os arquivos
  alterados/criados permanecem no working tree para revisão e commit posterior pelo dono da
  branch.
