# Relatorio de Bugfix

- Total de bugs no escopo: 30
- Corrigidos: 30
- Testes de regressao adicionados: 30
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-101
- Severidade: major
- Origem: NFR-03; finding de review do gate de producao
- Estado: fixed
- Causa raiz: `chmodTreeWritable` e o helper de limpeza do teste usavam paths
  reavaliados dentro de callbacks de `filepath.Walk`, produzindo o alerta G122 de
  TOCTOU.
- Arquivos alterados: `internal/fs/fs.go`, `internal/fs/os_test.go`
- Teste de regressao: `TestOS_RemoveAll_doesNotFollowSymlinkOutsideTree` preserva
  um arquivo externo apontado por symlink durante a remocao de uma arvore somente
  leitura. O helper de cleanup tambem passou a percorrer via `os.Root`.
- Validacao: `go test ./internal/fs -count=1`, `make lint` (0 issues),
  `go test ./... -count=1 -race`, `go vet ./...` e `go build ./...` passaram.

- ID: BUG-102
- Severidade: major
- Origem: NFR-03; finding de review do gate de producao
- Estado: fixed
- Causa raiz: helpers de subprocesso dos testes passavam `os.Args[0]` diretamente
  a `exec.Command`, que a analise de taint classifica como entrada nao confiavel
  (G702), embora o objetivo seja executar o binario do proprio teste.
- Arquivos alterados: `internal/taskloop/agent_test.go`,
  `internal/wrapper/wrapper_test.go`
- Teste de regressao: `TestCodexInvokerDeprecationWarning` e
  `TestWrapperEmitsGeminiDeprecationWarningOnce` continuam executando seus helpers
  em subprocesso resolvido por `os.Executable`.
- Validacao: `go test ./internal/taskloop ./internal/wrapper -count=1`,
  `make lint` (0 issues) e `go test ./... -count=1 -race` passaram.

- ID: BUG-103
- Severidade: major
- Origem: NFR-03; finding de review do gate de producao
- Estado: fixed
- Causa raiz: mockery v2.53.4 nao carrega corretamente tipos da biblioteca padrao
  com Go 1.27, encerrando ao analisar `crypto/sha256` e `os/exec`.
- Arquivos alterados: `Makefile`, `scripts/check-mocks.sh`, `mockery.yml` e os 40
  mocks declarados na configuracao.
- Teste de regressao: `make check-mocks` regenera em memoria de trabalho e compara
  hashes; duas execucoes consecutivas passaram apos `make mocks`, comprovando a
  geracao deterministica da configuracao v3.7.4.
- Validacao: `make mocks`, `make check-mocks` (duas vezes), `make lint` (0 issues),
  `go test ./... -count=1 -race`, `go vet ./...` e `go build ./...` passaram.

- ID: BUG-104
- Severidade: major
- Origem: NFR-03; finding de review do gate de producao
- Estado: fixed
- Causa raiz: a validacao delegava a `filepath.IsAbs` e `filepath.VolumeName`,
  cuja interpretacao depende do sistema operacional do runner. Como consequencia,
  paths POSIX absolutos nao eram reconhecidos no Windows e paths Windows nao eram
  reconhecidos de forma equivalente em sistemas POSIX.
- Arquivos alterados: `internal/sdd/result_schema.go`,
  `internal/sdd/result_schema_test.go`
- Teste de regressao: `TestResultValidatorRejectsAbsoluteEvidencePathsOnEveryPlatform`
  cobre paths relativos POSIX e Windows aceitos e rejeita sintaxes absolutas POSIX,
  raiz Windows, volume, UNC e prefixo estendido do Windows.
- Validacao: `go test ./internal/sdd -count=1`, `make test-sdd-evals`,
  `go test ./... -count=1 -race`, `make vet`, `make build` e `make lint`
  passaram; o linter encerrou com `0 issues`.

- ID: BUG-105
- Severidade: major
- Origem: NFR-04; finding de review do gate de producao
- Estado: fixed
- Causa raiz: o smoke de adaptadores inseria arquivos no `FakeFileSystem` com
  paths POSIX literais, enquanto o filesystem normaliza consultas com as regras
  de `filepath` do sistema operacional. No Windows, as chaves do mapa usavam
  separadores diferentes e `tasks.md` nao era encontrado.
- Arquivos alterados: `internal/taskloop/e2e_agent_test.go`
- Teste de regressao: `TestE2EAgent_PromptContainsAgentBlocks`, executado por
  `make smoke-adapters`, agora monta o workspace absoluto e todos os artefatos
  com `filepath.Join`, preservando a mesma fixture em sistemas POSIX e Windows.
- Validacao: `go test ./internal/taskloop/... -count=1`, `make smoke-adapters`,
  `go test ./... -race`, `make lint`, `make vet`, `make build` e
  `git diff --check` passaram.

- ID: BUG-106
- Severidade: major
- Origem: NFR-04; finding de review do gate de producao
- Estado: fixed
- Causa raiz: a correcao inicial do BUG-105 ainda construia o workspace falso a
  partir de uma raiz sem volume. No Windows, o servico transforma o caminho em
  absoluto com volume antes de consultar o `FakeFileSystem`, divergindo da chave
  usada na fixture.
- Arquivos alterados: `internal/taskloop/e2e_agent_test.go`
- Teste de regressao: o helper `e2eWorkspacePath(t)` usa `filepath.Abs` e os
  cenarios `TestE2EAgent_PromptContainsAgentBlocks` e
  `TestE2EAgent_ForensicArtifactStructureIdentical` cobrem os fluxos agent e
  legado no mesmo workspace absoluto que o servico resolve.
- Validacao: `GOTOOLCHAIN=go1.26.2 make smoke-adapters` passou, cobrindo o
  smoke de adaptadores com a versao declarada pelo modulo.

- ID: BUG-107
- Severidade: major
- Origem: NFR-03; finding de review do gate de producao
- Estado: fixed
- Causa raiz: o gate gerava mocks in-place, misturando diferenças de ambiente e
  alterações parciais do gerador ao worktree que estava sendo verificado; além
  disso, o stderr era suprimido e impedia diagnosticar a divergência Linux.
- Arquivos alterados: `scripts/check-mocks.sh`, `scripts/check-mocks_test.sh`,
  `Makefile` e `.github/workflows/test.yml`
- Teste de regressao: `scripts/check-mocks_test.sh` simula sucesso, drift e falha
  parcial do gerador e confirma que nenhum caminho altera o worktree original.
- Validacao: `bash scripts/check-mocks_test.sh` passou com geração integralmente
  isolada e diagnóstico preservado.

- ID: BUG-108
- Severidade: critical
- Origem: RF-08; finding de review do gate de producao
- Estado: fixed
- Causa raiz: `validate-sdd` validava somente a forma do digest persistido.
- Arquivos alterados: `internal/sdd/state.go`, `cmd/ai_spec_harness/sdd.go`
- Teste de regressao: `TestStoreValidateDirectoryRejectsApprovedArtifactDrift` altera `tasks.md` após aprovação e exige erro stale.
- Validacao: testes dirigidos SDD passaram; o estado real desatualizado agora falha fechado.

- ID: BUG-109
- Severidade: critical
- Origem: RF-06; finding de review do gate de producao
- Estado: fixed
- Causa raiz: `State` persistia somente artefatos e eventos.
- Arquivos alterados: `internal/sdd/state.go`
- Teste de regressao: `TestStorePersistsCompleteOperationalModel` prova requisitos RF/NFR, DAG e task state persistidos.
- Validacao: testes dirigidos SDD passaram.

- ID: BUG-110
- Severidade: critical
- Origem: RF-07, RF-11; finding de review do gate de producao
- Estado: fixed
- Causa raiz: o comando encerrava após três flags booleanas, sem iniciar tentativa.
- Arquivos alterados: `cmd/ai_spec_harness/sdd.go`, `internal/taskloop/orchestrator.go`
- Teste de regressao: testes de início idempotente, lock e persistência de tentativa cobrem o fluxo real sem runtime remoto.
- Validacao: testes dirigidos do CLI e taskloop passaram.

- ID: BUG-111
- Severidade: major
- Origem: RF-09, RF-10; finding de review do gate de producao
- Estado: fixed
- Causa raiz: o parser lia apenas RF e não era chamado pelo CLI; a tabela canônica no topo também não era reconhecida.
- Arquivos alterados: `internal/sdd/tasks/parser.go`, `cmd/ai_spec_harness/sdd.go`
- Teste de regressao: `TestParserParse` cobre NFR sem cobertura e `TestParserAcceptsCanonicalTasksTableBeforeSections` cobre o formato aprovado.
- Validacao: testes dirigidos SDD passaram.

- ID: BUG-112
- Severidade: critical
- Origem: RF-01, RF-03; finding de review do gate de producao
- Estado: fixed
- Causa raiz: o validador aceitava hashes e evidências mencionados apenas em Markdown.
- Arquivos alterados: `.agents/scripts/validate-task-evidence.sh`, mirrors e testes shell.
- Teste de regressao: a suíte rejeita result JSON ausente, patch divergente e exige digest de log físico contido.
- Validacao: oito cenários de `validate-task-evidence_test.sh` passaram.

- ID: BUG-113
- Severidade: critical
- Origem: NFR-04; incidente release v0.29.2 no SHA 8d946753
- Estado: fixed
- Causa raiz: `release.yml` disparava diretamente no push, em paralelo aos testes.
- Arquivos alterados: `.github/workflows/release.yml`
- Teste de regressao: contrato do workflow exige `workflow_run` de `Tests` concluído com sucesso e verifica o `head_sha` no checkout.
- Validacao: inspeção do YAML confirmou condição fail-closed e vínculo ao SHA testado.

- ID: BUG-114
- Severidade: critical
- Origem: RF-03, RF-14; finding de review do gate de producao
- Estado: fixed
- Causa raiz: `Finish` validava apenas presença/formato do resultado, sem snapshot ou arquivos.
- Arquivos alterados: `internal/taskloop/orchestrator.go`, `internal/taskloop/orchestrator_test.go`
- Teste de regressao: `TestOrchestratorFinishRejectsSnapshotAndPhysicalEvidenceDivergence` cobre patch artificial, arquivo ausente e digest falso.
- Validacao: testes dirigidos do taskloop passaram.

- ID: BUG-115
- Severidade: major
- Origem: RF-05, NFR-01; finding de review do gate de producao
- Estado: fixed
- Causa raiz: checkpoints históricos usavam YAML mínimo fora do schema v2.
- Arquivos alterados: `.specs/prd-sdd-robusto/.checkpoints/*.json`, `scripts/test-sdd-evals.sh`
- Teste de regressao: o gate rejeita qualquer YAML legado e valida todo checkpoint pelo `execution-result.schema.json`.
- Validacao: dez checkpoints v2 passaram no schema e permanecem `blocked`, sem promover prova histórica incompleta a `done`.

- ID: BUG-116
- Severidade: major
- Origem: RF-16; finding de review do gate de producao
- Estado: fixed
- Causa raiz: campos e totais eram procurados globalmente no relatório.
- Arquivos alterados: `.agents/scripts/validate-bugfix-evidence.sh`, `internal/evidence/evidence.go` e testes.
- Teste de regressao: relatório com segundo bloco incompleto e total incorreto é rejeitado pelo shell e pelo validador Go.
- Validacao: testes shell e `go test ./internal/evidence` passaram.

- ID: BUG-117
- Severidade: major
- Origem: RF-18; finding de review do gate de producao
- Estado: fixed
- Causa raiz: o corpus imprimia contagens e custo/tokens constantes, sem matriz de classificação ou latência observada.
- Arquivos alterados: `evals/sdd/manifest.json`, `scripts/test-sdd-evals.sh`
- Teste de regressao: o runner calcula quality, escape, falso positivo/negativo e latências e aplica thresholds do manifest.
- Validacao: 21 fixtures produziram quality=1, escapes/FP/FN=0 e latência total observada de 226.230 ms.

- ID: BUG-118
- Severidade: major
- Origem: NFR-03; finding de review do gate de producao
- Estado: fixed
- Causa raiz: geração in-place podia deixar mocks parciais e temporários em falha ou drift.
- Arquivos alterados: `scripts/check-mocks.sh`, `scripts/check-mocks_test.sh`
- Teste de regressao: geradores falsos cobrem sucesso, drift e falha após escrita parcial em árvore isolada.
- Validacao: `bash scripts/check-mocks_test.sh` passou e o fixture original permaneceu byte a byte intacto.

- ID: BUG-119
- Severidade: major
- Origem: RF-03, RF-15; finding de review do gate final de producao
- Estado: fixed
- Causa raiz: o relatório 10.0 e a revisão referenciavam um snapshot anterior às
  correções pós-plano, sem vínculo físico com o candidato cumulativo atual.
- Arquivos alterados: `.specs/prd-sdd-robusto/10.0_execution_report.md`,
  `evidence/10.0/final-wave-bug-119-125.log`, snapshot físico final e resultado
  da revisão independente fresca.
- Teste de regressao: o pacote final é validado contra um artefato físico de patch
  com SHA-256 recomputável, logs frescos e revisão independente do mesmo digest.
- Validacao: gates locais amplos e dirigidos estão persistidos em
  `evidence/10.0/final-wave-bug-119-125.log`; revisão externa/remota não é alegada.

- ID: BUG-120
- Severidade: critical
- Origem: RF-11, RF-14; finding de review do gate final de producao
- Estado: fixed
- Causa raiz: `Finish` comparava o resultado final ao snapshot inicial imutável,
  tornando uma alteração legítima impossível e permitindo um resultado stale.
- Arquivos alterados: `internal/taskloop/orchestrator.go`,
  `internal/taskloop/orchestrator_test.go`.
- Teste de regressao: `TestOrchestratorFinishesAttemptIdempotently` modifica o
  workspace após `Start`, exige digests finais diferentes dos iniciais e conclui;
  `TestOrchestratorFinishRejectsSnapshotAndPhysicalEvidenceDivergence` rejeita
  patch e provas artificiais.
- Validacao: testes dirigidos de `internal/taskloop` e race global passaram.

- ID: BUG-121
- Severidade: critical
- Origem: RF-03, RF-05, RF-06; finding de review do gate final de producao
- Estado: fixed
- Causa raiz: o modelo operacional promovia `done` diretamente de `tasks.md` e
  aceitava um relatório pela mera existência do path.
- Arquivos alterados: `internal/sdd/state.go`, `internal/sdd/state_test.go`.
- Teste de regressao: `TestStoreImportsDoneOnlyFromValidPhysicalCheckpoint`
  cobre checkpoint done com provas físicas, checkpoint blocked e ausência do
  checkpoint, que agora falha fechada.
- Validacao: testes dirigidos de `internal/sdd` e a suíte global passaram.

- ID: BUG-122
- Severidade: critical
- Origem: RF-01, RF-03, RF-14; finding de review do gate final de producao
- Estado: fixed
- Causa raiz: o validador comparava dois campos textuais controlados pelo mesmo
  produtor e não exigia um artefato físico correspondente ao patch.
- Arquivos alterados: `.agents/scripts/validate-task-evidence.sh`, mirrors
  sincronizados e `tests/scripts/validate-task-evidence_test.sh`.
- Teste de regressao: `TC9-patch-fisico-divergente` mantém Markdown/JSON
  coerentes e adultera somente o patch físico; o gate rejeita pelo digest.
- Validacao: 9/9 cenários passaram e 24 mirrors ficaram em sync.

- ID: BUG-123
- Severidade: major
- Origem: RF-10; finding de review do gate final de producao
- Estado: fixed
- Causa raiz: dependências cross-PRD reconhecidas pelo parser eram ignoradas.
- Arquivos alterados: `internal/sdd/tasks/parser.go`,
  `internal/sdd/tasks/parser_test.go`.
- Teste de regressao: `TestParserParseAtResolvesCrossPRDFailClosed` cobre PRD e
  task ausentes, task não done, hash stale, ciclo cross-PRD e destino válido.
- Validacao: testes dirigidos do parser e suíte global passaram.

- ID: BUG-124
- Severidade: major
- Origem: RF-07, NFR-01 e seção Migração e Rollback da TechSpec; finding de review
- Estado: fixed
- Causa raiz: o CLI não expunha caminho operacional para migrar estado legado
  nem rollback limitado à identidade da migração.
- Arquivos alterados: `cmd/ai_spec_harness/root.go`,
  `cmd/ai_spec_harness/sdd.go`, `internal/sdd/state.go` e testes associados.
- Teste de regressao: `TestStoreMigrationDryRunAndRunIDScopedRollback` e
  `TestSDDMigrationCommandsRequireConfirmationAndScopeRollback` provam dry-run
  sem escrita, confirmação obrigatória e rejeição de outro `run_id`.
- Validacao: testes unitários e de integração dirigidos passaram.

- ID: BUG-125
- Severidade: minor
- Origem: NFR-04; finding de actionlint no gate final de producao
- Estado: fixed
- Causa raiz: o summary fazia múltiplos redirecionamentos consecutivos para o
  mesmo arquivo, disparando ShellCheck SC2129.
- Arquivos alterados: `.github/workflows/release.yml`.
- Teste de regressao: `actionlint` analisa `test.yml` e `release.yml` sem findings.
- Validacao: `actionlint .github/workflows/test.yml .github/workflows/release.yml`
  passou sem output.

- ID: BUG-126
- Severidade: major
- Origem: RF-06; gate `validate-sdd` do estado operacional de produção
- Estado: fixed
- Causa raiz: o estado aprovado foi criado antes da persistência do modelo
  operacional v2 e permaneceu sem requisitos, DAG, tarefas e evidências,
  apesar dos artefatos e checkpoints já estarem no contrato atual.
- Arquivos alterados: `sdd-state.json`, checkpoints JSON v2,
  `scripts/test-sdd-evals.sh` e evidências finais.
- Teste de regressao: `scripts/test-sdd-evals.sh` agora valida o
  `sdd-state.json` do PRD após validar todos os checkpoints; a suíte falha
  fechada se o modelo operacional aprovado estiver incompleto.
- Validacao: `go run . migrate-sdd .specs/prd-sdd-robusto --run-id
  production-proof-20260903-r2 --confirm`, `go run . validate-sdd
  .specs/prd-sdd-robusto` e o resultado físico 10.0 passaram.

- ID: BUG-127
- Severidade: critical
- Origem: RF-01, RF-03; `make test-validators` (caso a) falhando em `main`
- Estado: fixed
- Causa raiz: o gate de criterios de aceite usava a classe de bracket multibyte
  `Crit[eé]rios` em `awk`. Em `awk` byte-oriented (mawk, padrao em Debian/Ubuntu
  e nos runners Linux da CI) a classe nunca casa "Criterios" acentuado, entao
  `criteria_count` era 0, o gate era silenciosamente desligado com o aviso de
  legado e um relatorio com criterios nao comprovados passava com exit 0
  (fail-open exatamente no invariante que o PRD exige fail-closed).
- Arquivos alterados: `.agents/scripts/validate-task-evidence.sh`,
  `.agents/scripts/validate-review-evidence.sh` e os tres mirrors
  (`.claude/scripts/`, `internal/embedded/assets/.claude/scripts/`,
  `internal/embedded/assets/.agents/scripts/`).
- Correcao: classes de bracket multibyte substituidas por alternacao
  (`Crit(e|é)rios`, `cr(i|í)tico`, `m(e|é)dia`), que e byte-safe em mawk, gawk e
  grep POSIX.
- Teste de regressao: `scripts/test-validators.sh` caso "a2" executa o mesmo
  cenario sob `LC_ALL=C` e falha se o gate voltar a emitir "gate de aceite
  ignorado" — a assercao e independente do awk instalado no runner.
- Validacao: `make test-validators` -> 10 asserts OK, 0 FAIL.

- ID: BUG-128
- Severidade: major
- Origem: RF-02, RF-05; `make test-hooks` falhando em `main`
- Estado: fixed
- Causa raiz: as fixtures F02 e F02b de `scripts/test-hooks.sh` foram escritas
  antes de o schema `execution-result.schema.json` passar a exigir `patch_ref`
  para `status: done`. O gate de hooks ficou vermelho em `main` (2 asserts FAIL),
  mascarando regressoes reais de contrato.
- Arquivos alterados: `scripts/test-hooks.sh`
- Correcao: as duas fixtures declaram `patch_ref` como caminho relativo contido.
- Teste de regressao: os proprios asserts F02 e F02b, agora verdes.
- Validacao: `bash scripts/test-hooks.sh` -> 44 asserts OK, 0 FAIL.

- ID: BUG-129
- Severidade: major
- Origem: NFR-03, NFR-04; leitura do gate `test-validators`
- Estado: fixed
- Causa raiz: `tests/scripts/validate-task-evidence_test.sh` tinha `rtk cat`
  versionado em duas linhas (vazamento de uma ferramenta local de proxy). Sob
  `set -euo pipefail`, em qualquer ambiente sem `rtk` — incluindo os runners de
  CI — a substituicao de comando retorna 127 e a suite inteira aborta antes do
  primeiro caso.
- Arquivos alterados: `tests/scripts/validate-task-evidence_test.sh`
- Correcao: uso de `cat` do sistema.
- Teste de regressao: a propria suite (11 casos) passa a depender apenas de
  utilitarios POSIX; `git grep 'rtk '` nao retorna ocorrencias em `*.sh`.
- Validacao: `bash tests/scripts/validate-task-evidence_test.sh` -> 11 passaram.

- ID: BUG-130
- Severidade: critical
- Origem: RF-10, NFR-03; `make fuzz` (`FuzzParseTaskFile`)
- Estado: fixed
- Causa raiz: dois defeitos combinados em `Catalog.ParseTasksFile`.
  (1) `detectColumnIndices` gravava `depsIdx` mesmo quando nao reconhecia a
  linha como header (retorno `false`), deixando os indices parcialmente
  corrompidos por qualquer linha que citasse "Dependencias".
  (2) O guarda de limites checava apenas `len(cols) <= depsIdx`, ignorando
  `statusIdx` e a coluna de titulo. Com `depsIdx=1` e `statusIdx=3`, a linha
  `|0.0|` produzia `cols` de tamanho 3 e `cols[statusIdx]` estourava o slice:
  `panic: runtime error: index out of range [3] with length 3`.
- Arquivos alterados: `internal/taskloop/parser.go`
- Correcao: atribuicao atomica dos indices (so quando ha coluna Status) e guarda
  por `max(2, statusIdx, depsIdx)`.
- Teste de regressao: `TestParseTasksFile_LinhaSemHeaderNaoCorrompeIndices` e
  `TestParseTasksFile_HeaderComDepsAntesDeStatus` em
  `internal/taskloop/parser_test.go`, mais o corpus versionado
  `internal/taskloop/testdata/fuzz/FuzzParseTaskFile/be0b4c6bfbfa4441` com o
  input minimo que reproduzia o panic.
- Validacao: `make fuzz` -> exit 0 (9 alvos, 30s cada);
  `go test ./internal/taskloop -count=1` -> passou.

## Comandos Executados

- `python3 .agents/skills/bugfix/scripts/validate-bug-input.py --input .specs/prd-sdd-robusto/production-gates-bugs.json` -> SUCCESS: 26 bugs validados no formato canonico.
- Gates dirigidos BUG-120..125, `make test`, `make integration`, `make vet`,
  `make build`, `make lint`, race global, validators, evals, sync, actionlint e
  `git diff --check` -> passaram; saída consolidada em
  `evidence/10.0/final-wave-bug-119-125.log`.
- `go test ./internal/fs ./internal/taskloop ./internal/wrapper -count=1` -> passou.
- `make mocks` -> passou; 40 mocks normalizados por mockery v3.7.4.
- `make check-mocks` -> passou em duas execucoes consecutivas.
- `make lint` -> passou, 0 issues.
- `go test ./... -count=1 -race` -> passou.
- `go vet ./...` -> passou.
- `go build ./...` -> passou.
- `git diff --check` -> passou.
- `GOTOOLCHAIN=go1.26.2 go test ./internal/evidence ./internal/sdd/... ./internal/taskloop ./cmd/ai_spec_harness -count=1` -> passou, 1007 testes.
- `GOTOOLCHAIN=go1.26.2 bash scripts/test-sdd-evals.sh` -> passou, métricas calculadas e thresholds atendidos.
- `bash scripts/test-validators.sh` -> passou, 8 cenários.
- `bash tests/scripts/validate-task-evidence_test.sh .agents/scripts/validate-task-evidence.sh` -> passou, 8 cenários.
- `bash tests/scripts/validate-bugfix-evidence_test.sh .agents/scripts/validate-bugfix-evidence.sh` -> passou.
- `bash scripts/check-mocks_test.sh` -> passou para sucesso, drift e falha parcial.
- `bash scripts/check-scripts-sync.sh` -> passou, 24 comparações em sync.
- `GOTOOLCHAIN=go1.26.2 make check-mocks` -> passou em duas execucoes consecutivas.
- `GOTOOLCHAIN=go1.26.2 make smoke-adapters` -> passou.
- `bash -n scripts/check-mocks.sh` -> passou.
- `make test-check-mocks` -> passou; simulacao hermetica confirmou status nao-zero
  e diagnostico integral quando o gerador falha.
- `GOTOOLCHAIN=go1.26.2 make test` -> passou.
- `GOTOOLCHAIN=go1.26.2 make integration` -> passou.
- `GOTOOLCHAIN=go1.26.2 make vet` -> passou.
- `GOTOOLCHAIN=go1.26.2 make build` -> passou.
- `GOTOOLCHAIN=go1.26.2 make lint` -> passou, 0 issues.
- `GOTOOLCHAIN=go1.26.2 go test ./... -count=1 -race` -> passou.
- `GOTOOLCHAIN=go1.26.2 make test-sdd-evals` -> passou (21 fixtures; 1 aceita e 20 rejeitadas).
- `GOTOOLCHAIN=go1.26.2 make test-validators` -> passou.
- `ai-spec validate-sdd .specs/prd-sdd-robusto` -> passou.
- `ai-spec check-spec-drift .specs/prd-sdd-robusto/tasks.md` -> passou.
- `go test ./internal/taskloop/... -count=1` -> passou.
- `make smoke-adapters` -> passou.
- `go test ./... -race` -> passou.
- `make lint` -> passou (0 issues).
- `make vet` -> passou.
- `make build` -> passou.
- `bash .agents/scripts/validate-bugfix-evidence.sh --rf NFR-04 .specs/prd-sdd-robusto/bugfix_report.md` -> passou.
- Revisao manual do delta de BUG-101..103 -> APPROVED, sem achados critical/high/medium/low.
- `go test ./internal/sdd -count=1` -> passou (35 testes).
- `make test-sdd-evals` -> passou (21 fixtures; 1 aceita e 20 rejeitadas).
- `go test ./... -count=1 -race` -> passou (2604 testes em 67 pacotes).
- `make vet` -> passou.
- `make build` -> passou.
- `make lint` -> passou (0 issues).
- `git diff --check` -> passou.

## Riscos Residuais

- `os.Root` e usado em Go 1.26.2+, ja declarado pelo modulo; os gates locais foram
  executados em Go 1.27.0. A matriz remota de CI permanece responsabilidade do
  pipeline de integracao.
- A migracao para mockery v3.7.4 altera a forma do codigo gerado, mas todos os
  mocks permanecem declarados em `mockery.yml`, compilam e sao estaveis em
  regeneracoes consecutivas.
- A deteccao de paths do BUG-104 e lexical e independente do runner; a matriz
  remota em Windows continua sendo a evidencia complementar para o ambiente de CI.
- BUG-105 foi validado no runner local com paths nativos; a execucao remota da
  matriz Windows permanece a prova complementar do ambiente de CI.
- BUG-106 e BUG-107 foram executados localmente com o toolchain Go 1.26.2
  declarado no modulo; a matriz remota continua sendo a evidencia complementar
  dos runners Ubuntu, macOS e Windows.

## Rodada de Prontidao para Producao — 2026-09-04

Execucao completa dos gates em ambiente limpo (toolchain Go 1.26.8 instalado do
zero; nenhum Go presente antes). Quatro defeitos novos foram encontrados por
gates que estavam falhando ou fail-open em `main` e foram corrigidos.

- Bugs encontrados nesta rodada: 4 (BUG-127..BUG-130), todos corrigidos e com teste de
  regressao; os blocos individuais estao na secao `## Bugs` acima.

### Melhorias de Gate (nao-bug)

Lacunas de cobertura de CI observadas ao diagnosticar BUG-128 e BUG-130, corrigidas
em `.github/workflows/test.yml`:

- O passo de fuzz executava 7 dos 9 alvos declarados em `make fuzz`;
  `FuzzValidateFrontmatter` e `FuzzReadTaskFileStatus` foram adicionados para
  restaurar a paridade com o Makefile.
- `make test-portable-skills` existia no Makefile mas nao era executado por
  nenhum workflow; passou a rodar no job `lint`.

O passo de fuzz permanece `continue-on-error: true` (descoberta time-boxed). A
regressao dos crashers ja encontrados nao depende disso: o corpus versionado em
`internal/taskloop/testdata/fuzz/` e executado por `go test ./...`, que e
bloqueante. Tornar a descoberta bloqueante e decisao de politica do mantenedor.

### Auditoria de Prontidao (pos-correcao)

Com o gate de aceite finalmente ativo (BUG-127), os 10 relatorios de execucao do PRD foram
re-submetidos a `validate-task-evidence.sh`. Nenhum deles produz `FALTANDO` de contrato,
criterios, rastreabilidade, veredito ou cobertura — nem no modo default nem com
`AI_SDD_STRICT_EVIDENCE=1`. A evidencia historica sobrevive ao gate corrigido.

A unica falha ao reprocessar relatorios antigos e `snapshot fisico canonico invalido`, e ela
e estrutural, nao um defeito: `captureFinalSnapshot` recomputa o diff vivo contra `HEAD`, entao
a prova fisica so e verificavel no instante em que a tarefa fecha. Depois que o trabalho e
commitado, nenhum auditor consegue re-verificar a evidencia de uma tarefa passada, porque
RF-14 torna o commit SHA opcional. Isso limita auditoria retroativa e esta registrado como
risco aberto, nao como bug.

Gap fechado nesta rodada: NFR-01 exige "fail-closed em modo estrito", mas o validador nao tinha
modo estrito algum — os dois escapes de legado (`Arquivo:` nao resolvivel e task sem secao de
criterios) eram incondicionais e desligavam o gate de aceite inteiro. `AI_SDD_STRICT_EVIDENCE=1`
os fecha; o default permanece warning-only (casos d2/d3/d4 em `scripts/test-validators.sh`
travam os dois modos).

### Validacao Consolidada da Rodada

- `go build ./...` -> passou.
- `go vet ./...` -> passou.
- `go test ./... -count=1 -race` -> passou (2633 testes em 67 pacotes).
- `make integration` -> passou.
- `make fuzz` -> passou.
- `make lint` -> passou (0 issues).
- `make check-mocks` -> passou.
- `make coverage-packages` -> passou (8 pacotes acima do piso de 70%).
- `make check-skills-sync` / `check-hooks-sync` / `check-scripts-sync` -> passaram.
- `make test-hooks` -> 44 asserts OK, 0 FAIL.
- `make test-validators` -> 10 + 11 asserts OK, 0 FAIL.
- `make test-sdd-evals` -> 21 fixtures, 20 rejeitadas, 1 aceita, escape_rate 0.0.
- `make smoke-adapters` -> passou.
- `make test-portable-skills` -> passou.
- `./ai-spec validate-sdd .specs/prd-sdd-robusto` -> OK.
- `./ai-spec check-spec-drift .specs/prd-sdd-robusto/tasks.md` -> sem drift.

### Riscos Residuais da Rodada

- Toda a evidencia acima e de runner Linux com Go 1.26.8. A matriz remota
  Ubuntu/macOS/Windows continua sendo o gate externo pendente, como ja registrado
  no `_orchestration_report.md`.
- BUG-127 demonstra que gates escritos em shell podem falhar de forma aberta
  conforme o `awk` do runner. A guarda `LC_ALL=C` cobre a classe de defeito nos
  validadores de evidencia; outros scripts com regex acentuada continuam sendo
  risco aberto ate terem cobertura equivalente.
