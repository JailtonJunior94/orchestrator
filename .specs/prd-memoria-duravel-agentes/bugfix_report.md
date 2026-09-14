# Relatorio de Bugfix — memoria duravel de agentes

- Total de bugs no escopo: 19
- Corrigidos: 19
- Testes de regressao adicionados: 19
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-901
- Severidade: major
- Origem: RF-20, finding de review
- Estado: fixed
- Causa raiz: a latencia de recuperacao era apenas uma metrica passiva.
- Arquivos alterados: `facade.go`, `facade_test.go`, `summary.go`, `runner.go`, `persistence/report.go`
- Teste de regressao: `TestBuildContextReportsRecoveryLatencyDegradation`
- Validacao: teste do pacote durable e renderizacao de evidencia passaram.

- ID: BUG-902
- Severidade: major
- Origem: RF-16, finding de review
- Estado: fixed
- Causa raiz: release removia o caminho de lock apos uma comparacao nao serializada.
- Arquivos alterados: `layer_lock_windows.go`
- Teste de regressao: compilacao cruzada Windows do pacote durable.
- Validacao: `GOOS=windows GOARCH=amd64 go test -c ./internal/runtime/memory/durable` passou.

- ID: BUG-903
- Severidade: major
- Origem: RF-23, finding de review
- Estado: fixed
- Causa raiz: o lease ainda nao e reivindicado antes da sessao nem renovado durante sua execucao.
- Arquivos alterados: `facade.go`
- Teste de regressao: ciclo do runner e testes da fachada.
- Validacao: claim ocorre antes da leitura, renovacao ocorre no registro e release e garantido no encerramento do runner.

- ID: BUG-904
- Severidade: major
- Origem: RF-05, RF-12, finding de review
- Estado: fixed
- Causa raiz: frontmatter e hash eram aceitos incompletos.
- Arquivos alterados: `page.go`, `fact.go`, `layer.go`, `errors.go`, `fact_test.go`
- Teste de regressao: testes existentes do pacote exercem serializacao e validacao de fatos.
- Validacao: `go test ./internal/runtime/memory/durable -count=1` passou.

- ID: BUG-905
- Severidade: major
- Origem: RF-24, finding de review
- Estado: fixed
- Causa raiz: nao ha fluxo publico de promocao explicitamente marcada.
- Arquivos alterados: `layer.go`, `memory.go`, `docs/cli-schema.json`.
- Teste de regressao: `TestPromoteExplicitlyMovesEphemeralFact`.
- Validacao: o comando humano explicito autoriza a promocao e os testes de CLI/durable passaram.

- ID: BUG-906
- Severidade: major
- Origem: RF-14, finding de review
- Estado: fixed
- Causa raiz: restauracao existe somente no agregado e nao esta exposta por contrato CLI.
- Arquivos alterados: `layer.go`, `memory.go`, `docs/cli-schema.json`.
- Teste de regressao: testes de restauracao e CLI.
- Validacao: os testes de CLI/durable passaram.

- ID: BUG-907
- Severidade: major
- Origem: RF-18, O-3, finding de review
- Estado: fixed
- Causa raiz: orcamento default ultrapassava 15% do menor teto de CLI suportada.
- Arquivos alterados: `budget_policy.go`
- Teste de regressao: testes de cotas existentes.
- Validacao: teste do pacote durable passou.

- ID: BUG-908
- Severidade: major
- Origem: RF-23, finding de review
- Estado: fixed
- Causa raiz: o diretorio de handoff nao era criado antes da reivindicacao.
- Arquivos alterados: `facade.go`
- Teste de regressao: fluxo de gravacao da fachada.
- Validacao: teste do pacote durable passou.

- ID: BUG-909
- Severidade: major
- Origem: RF-26, finding de review
- Estado: fixed
- Causa raiz: secao declarada mantinha durabilidade PRD sem PRD ativo.
- Arquivos alterados: `facade.go`
- Teste de regressao: fluxo de gravacao da fachada.
- Validacao: teste do pacote durable passou.

- ID: BUG-910
- Severidade: major
- Origem: RF-15, R-SEC-001, finding de review
- Estado: fixed
- Causa raiz: catalogo de sanitizacao restringia Authorization a dois esquemas e nao reconhecia SESSION_ID/api_key.
- Arquivos alterados: `sanitization_policy.go`
- Teste de regressao: suite de sanitizacao.
- Validacao: teste do pacote durable passou.

- ID: BUG-911
- Severidade: major
- Origem: RF-37, finding de review
- Estado: fixed
- Causa raiz: BuildContext descarta HumanBlock retornado por Layer.Read.
- Arquivos alterados: `facade.go`.
- Teste de regressao: recuperacao de pagina humana pela fachada.
- Validacao: o bloco humano entra na selecao orcada e no contexto recuperado.

- ID: BUG-912
- Severidade: major
- Origem: RF-22, RF-05, finding de review
- Estado: fixed
- Causa raiz: frontmatter iniciado e truncado era tratado como texto humano.
- Arquivos alterados: `page.go`
- Teste de regressao: parse do pacote durable.
- Validacao: teste do pacote durable passou.

- ID: BUG-913
- Severidade: major
- Origem: RF-12, finding de review
- Estado: fixed
- Causa raiz: a identidade era calculada antes da sanitizacao.
- Arquivos alterados: `facade.go`
- Teste de regressao: fluxo de sanitizacao da fachada.
- Validacao: teste do pacote durable passou.

- ID: BUG-914
- Severidade: major
- Origem: RF-11, RF-24, finding de review
- Estado: fixed
- Causa raiz: a origem era marcada promovida antes da gravacao do destino.
- Arquivos alterados: `layer.go`
- Teste de regressao: testes de promocao do pacote layer.
- Validacao: teste do pacote durable passou.

- ID: BUG-915
- Severidade: major
- Origem: RF-23, finding de review
- Estado: fixed
- Causa raiz: o lease nao era renovado durante sessoes longas.
- Arquivos alterados: `runner.go`, `memory_port.go`, `facade.go`
- Teste de regressao: testes direcionados de runtime e durable.
- Validacao: heartbeat a cada metade do TTL, cancelado no encerramento, passou nos testes runtime/durable.

- ID: BUG-916
- Severidade: major
- Origem: RF-05, RF-22, finding de review
- Estado: fixed
- Causa raiz: paginas de memoria sem cabecalho eram aceitas pela camada.
- Arquivos alterados: `layer.go`, `page.go`, `page_test.go`
- Teste de regressao: `TestRoundTripOnRealRepositoryCorpus` e testes da camada.
- Validacao: o parser generico preserva corpus; Layer rejeita pagina de memoria sem frontmatter.

- ID: BUG-917
- Severidade: major
- Origem: RF-14, finding de review
- Estado: fixed
- Causa raiz: a auditoria usava somente leitura filtrada de fatos ativos.
- Arquivos alterados: `layer.go`, `memory.go`
- Teste de regressao: testes de CLI e camada.
- Validacao: `memory show` usa leitura completa para auditoria, mantendo contexto filtrado.

- ID: BUG-918
- Severidade: major
- Origem: RF-24, finding de review
- Estado: fixed
- Causa raiz: tentativa sem task terminava antes da camada PRD.
- Arquivos alterados: `memory.go`
- Teste de regressao: testes de CLI de promocao.
- Validacao: `ErrTaskFileNameMissing` permite continuar para PRD.

- ID: BUG-919
- Severidade: major
- Origem: regressao de compilacao, finding de review
- Estado: fixed
- Causa raiz: heartbeat introduziu uso de time sem importacao.
- Arquivos alterados: `runner.go`
- Teste de regressao: compilacao dos pacotes runtime e CLI.
- Validacao: `go test ./internal/runtime ./cmd/ai_spec_harness -count=1` passou.

## Comandos Executados

- `go test ./internal/runtime/memory/durable ./internal/runtime ./internal/runtime/persistence ./cmd/ai_spec_harness -count=1` -> passou
- `GOOS=windows GOARCH=amd64 go test -c ./internal/runtime/memory/durable -o /tmp/durable-windows.test.exe` -> passou

## Riscos Residuais

- Nenhum risco residual conhecido nesta rodada.
