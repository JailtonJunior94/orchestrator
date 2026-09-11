# Relatório de Review — Tarefa 10.0

- Veredito: APPROVED_WITH_REMARKS (sem tags criticas [critical]/[security]/[blocker]/[high] — nao escala para blocked)
- Alvo revisado: diff cumulativo do worktree (sha256 do `git diff`: ver `diff.sha256`), focado nos 8 pontos de maior risco indicados ao revisor: RemovedAgentError/ResolveTool (RF-03), manifesto InstalledFiles + expectedInstalledPaths, uninstall.go reescrito, testes novos de uninstall, contextgen buildToolNotes/buildEnforcementMatrix + correcao Copilot, os 9 golden files, cli_contract_test.go/task_loop.go, e grep residual de "gemini".

## Findings

- [MEDIUM] `internal/contextgen/contextgen.go`: `var _toolNoteCatalog` violava R-STYLE-001.3 (prefixo `_` proibido em identificador Go). CORRIGIDO nesta tarefa: renomeado para `toolNoteCatalog` (verificado com `go build`/`go vet`/`go test ./internal/contextgen/...` apos o fix — todos verdes).
- [LOW] Constantes `_codexTrustRPCTimeout` (e outras com o mesmo padrao) em `internal/install/install.go` seguem convencao pre-existente nao-conforme, anterior a esta tarefa. Fora do escopo de remocao do Gemini — nao tocado (regra do repo: corrigir comentarios/estilo apenas nas linhas efetivamente tocadas pelo diff da tarefa).
- [LOW] Working tree contem trabalho de tarefas anteriores (4.6-9.0) ainda nao commitado, pre-existente ao inicio desta tarefa (confirmado via `git status` no início da sessão). Nao e resultado da tarefa 10.0; registrado como risco residual de rastreabilidade para quem for commitar.
- [LOW] Mensagem de erro para `--tool opencode --runtime legacy` pode soar como "opencode nao existe" quando na verdade OpenCode e ACP-only por design (RF-13). Comportamento pre-existente (ValidTools legado nunca incluiu opencode), fora do escopo estrito de RF-01..RF-09.

## Verificacoes que passaram sem ressalva

- RemovedAgentError/ResolveTool (RF-03): tipagem, errors.As, mensagem com conjunto suportado e guia de migracao — corretos e testados.
- Manifesto InstalledFiles/HasFileTracking + expectedInstalledPaths: campo aditivo, nil vs vazio distinguidos, gravado so fora de dry-run, paths batendo com o que o instalador realmente escreve.
- uninstall.go: manifesto-primeiro com fallback conservador anunciado, testado com preservacao de arquivo do usuario, idempotencia e anuncio textual do fallback; limpeza incondicional de `.gemini/`/`GEMINI.md` e superset seguro do comportamento anterior.
- contextgen.go: buildToolNotes/buildEnforcementMatrix filtram corretamente por tools selecionadas (confirmado no diff de codex-only.agents.md); afirmacao falsa sobre Copilot sem hooks nativos corrigida.
- Os 9 golden files: cada diff explicado apenas por remocao do Gemini das listas ou pela correcao dos dois bugs pre-existentes (10.12/10.13); nenhum golden ainda cita Copilot "sem hooks nativos" ou Gemini.
- cli_contract_test.go/task_loop.go: gate negativo `TestCLISchemaDoesNotContainRetiredAgents` correto; roteamento do erro tipado no modo `--runtime acp` prioriza RemovedAgentError via errors.As antes do erro generico.
- grep -rni gemini: todas as ocorrencias restantes justificaveis (nome de modelo Gemini via OpenCode, chave do mapa de guias de migracao, residuo legado documentado, comentarios historicos).

**Residual risks**: nenhum critico. Debt de estilo pre-existente em identificadores `_`-prefixados (nao ampliado por esta tarefa apos o fix aplicado); mistura de fatias nao commitadas na working tree (pre-existente, nao introduzida por esta tarefa).
