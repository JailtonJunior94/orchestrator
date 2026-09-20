# Relatorio de Bugfix

- Total de bugs no escopo: 1
- Corrigidos: 1
- Testes de regressao adicionados: 1
- Pendentes: nenhum
- Estado final: done

## Bugs
- ID: COVERAGE-1
- Severidade: major
- Origem: RF-74, task 1.0, finding de review
- Estado: fixed
- Causa raiz: o teste denominado P95 usava o pior valor de cinco amostras, medindo P100 e falhando sob ruído de agendamento e coleta de lixo; a execução com `-race` adicionava contenção do runner, não latência do caminho medido.
- Arquivos alterados: internal/runtime/memory/durable/facade_bench_test.go
- Teste de regressao: TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage calcula o P95 por nearest-rank de vinte amostras e preserva o orçamento de 200ms fora de `-race`.
- Validacao: `make coverage` -> pass; `make coverage-packages` -> pass, durable 88.1%; `go test ./... -count=1 -race` -> pass.

## Comandos Executados
- `go test ./internal/runtime/memory/durable -run '^TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage$' -count=20 -v` -> pass em 20 execuções.
- `go test -cover ./internal/runtime/memory/durable -run '^TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage$' -count=20 -v` -> pass em 20 execuções.
- `go test -bench '^BenchmarkRecoveryWith1000ActiveFactsInSinglePage$' -benchmem ./internal/runtime/memory/durable` -> pass, p95_ms 15.39.
- `make coverage` -> pass.
- `make coverage-packages` -> pass.
- `go test ./... -count=1 -race` -> pass.

## Riscos Residuais
- Nenhum para COVERAGE-1; o teste de desempenho permanece fora de `-race`, que valida correção sob instrumentação e não é uma medição de latência comparável.
