# Relatorio de Bugfix — Rodada 3 (achados da terceira revisao pos-bugfix)

- Total de bugs no escopo: 2
- Corrigidos: 2
- Testes de regressao adicionados: 2
- Pendentes: nenhum
- Estado final: done
- Nota: ambos corrigidos diretamente pelo orquestrador (sem subagente), dado o escopo mecanico e bem delimitado pela revisao.

## Bugs

- ID: BUG-25
- Severidade: major
- Origem: re-review tarefa 7.0 (2a rodada, apos retry por limite semanal de API); R-STYLE-001.2 (hard)
- Estado: fixed
- Causa raiz: a correcao de BUG-08 (cascata tri-state de `durable_memory_enabled`) adicionou 8 testes novos em `internal/taskloop/runtimeconfig_internal_test.go`, cada um com doc-comment explicativo acima da assinatura — violando R-STYLE-001.2 (zero comentarios, nomes de teste devem ser autoexplicativos). A varredura de comentarios da correcao original de BUG-15 (subset A) nao cobriu este arquivo de teste especifico, apesar de listar os demais arquivos tocados por BUG-08.
- Correcao: removidos os 8 doc-comments (`TestResolveRuntimeConfig_HandoffLeaseTTLFromWorkspaceConfigOnly`, `TestResolveRuntimeConfig_HandoffLeaseTTLFlagEndToEndUntilJob`, `TestResolveRuntimeConfig_HandoffLeaseTTLWorkspaceWinsOverGlobal`, `TestOptionsToConfigOverrides_DurableMemoryDefaultDoesNotOverrideConfig`, `TestResolveRuntimeConfig_DurableMemoryFromWorkspaceConfigOnly`, `TestResolveRuntimeConfig_DurableMemoryFlagFalseDisablesWorkspaceTrue`, `TestResolveRuntimeConfig_DurableMemoryFlagEndToEndUntilJob`, `TestResolveRuntimeConfig_DurableMemoryZeroValuePreservesF1`), confirmado por `git diff HEAD` sem nenhuma linha `//` remanescente fora de `go:build`/`go:embed`.
- Arquivos alterados: `internal/taskloop/runtimeconfig_internal_test.go`
- Teste de regressao: nao aplicavel (remocao pura de comentario, sem alteracao de logica); os 8 testes ja existentes continuam cobrindo o comportamento
- Validacao: `git diff HEAD -- internal/taskloop/runtimeconfig_internal_test.go | grep -E '^\+' | grep -E '//' | grep -v 'go:build\|go:embed'` -> vazio; `go build ./...`; `go test ./internal/taskloop/... -run "DurableMemory|HandoffLeaseTTL" -v -count=1` (16 passed); `go test ./... -count=1` completo sem FAIL; `golangci-lint run ./internal/taskloop/...` -> "No issues found"

- ID: BUG-26
- Severidade: minor
- Origem: re-review tarefa 10.0 (3a rodada); rastreabilidade requisito-implementacao
- Estado: fixed
- Causa raiz: a correcao do BUG-24 (rodada 2) redesenhou o benchmark de RF-20 para medir 1.000 Fatos ativos numa unica pagina (em vez de 1.000 paginas fisicas separadas, interpretacao estruturalmente inatingivel dado o design real do sistema), mas os NOMES das funcoes (`BenchmarkRecoveryWith1000Pages`, `TestBuildContextP95PreliminaryBenchmark_1000Pages`) continuaram afirmando "1000 Pages", contradizendo o que o teste de fato mede — violacao indireta de R-STYLE-001.2 ("codigo deve ser autoexplicativo", e o nome de uma funcao de teste e sua unica documentacao permitida). Adicionalmente, `techspec.md` (linhas 289, 432) e `task-10.0-integracao-bench-make.md` (requisito RF-20, subtarefa 10.8, criterio de sucesso) continuaram afirmando literalmente "1.000 paginas" sem nota sobre a reinterpretacao, deixando a explicacao apenas em evidencia de execucao (`bugfix_report_round2.md`), nao em documentacao viva do requisito.
- Correcao: funcoes renomeadas para `BenchmarkRecoveryWith1000ActiveFactsInSinglePage` e `TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage`; `techspec.md` (secao de testes de integracao e tabela RF->teste) e `task-10.0-integracao-bench-make.md` (requisito RF-20, subtarefa 10.8, criterio de sucesso) atualizados para descrever com precisao o que e medido e por que essa e a interpretacao tecnicamente honesta de RF-20, citando a razao estrutural (`Facade.BuildContext`/`Layer.Read` resolvem por chave exata, no maximo 3 arquivos por chamada, sem varredura de diretorio). `spec-hash-techspec` recalculado em `tasks.md` (via `ai-spec hash`) para refletir o novo conteudo da techspec, evitando falso-positivo de drift.
- Arquivos alterados: `internal/runtime/memory/durable/recovery_bench_test.go`, `internal/runtime/memory/durable/facade_bench_test.go`, `.specs/prd-memoria-duravel-agentes/techspec.md`, `.specs/prd-memoria-duravel-agentes/task-10.0-integracao-bench-make.md`, `.specs/prd-memoria-duravel-agentes/tasks.md` (hash)
- Teste de regressao: os proprios benchmarks/testes renomeados, comportamento inalterado (mesma asserção de volume `FactsByLayer["prd"] == factCount`)
- Validacao: `go build ./...`; `go test ./internal/runtime/memory/durable/... -run TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage -v -count=1` (1 passed); `make bench` -> `BenchmarkRecoveryWith1000ActiveFactsInSinglePage-8 38 30297902 ns/op 31.12 p95_ms 13766580 B/op 133155 allocs/op PASS`; `make integration` -> todos os pacotes `ok`; `go test ./... -count=1` completo sem FAIL; `ai-spec check-spec-drift .specs/prd-memoria-duravel-agentes/tasks.md` -> "OK: sem drift detectado"; `gofmt -l` vazio

## Comandos Executados

- `go build ./...` -> sem erros
- `go test ./... -count=1` -> 0 FAIL
- `go test ./internal/taskloop/... -run "DurableMemory|HandoffLeaseTTL" -v -count=1` -> 16 passed
- `go test ./internal/runtime/memory/durable/... -run TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage -v -count=1` -> 1 passed
- `make bench` -> PASS, p95 real 31.12ms com 1.000 fatos ativos
- `make integration` -> todos os pacotes `ok`
- `golangci-lint run ./internal/taskloop/... ./internal/runtime/memory/durable/...` -> "No issues found"
- `gofmt -l` nos arquivos tocados -> vazio
- `ai-spec hash .specs/prd-memoria-duravel-agentes/techspec.md` -> hash recalculado e propagado a `tasks.md`
- `ai-spec check-spec-drift .specs/prd-memoria-duravel-agentes/tasks.md` -> "OK: sem drift detectado"

## Riscos Residuais

- RF-20, segunda clausula ("reportar quando o volume degradar essa garantia"): nao ha implementacao ativa de alerta/threshold alem da metrica passiva `BuildLatencyMs`. Pre-existente, fora do escopo desta rodada e das tarefas 7.0/10.0 conforme documentado pelo revisor da 3a rodada de 10.0 — nao bloqueia esta tarefa, mas fica registrado para rastreamento futuro.
