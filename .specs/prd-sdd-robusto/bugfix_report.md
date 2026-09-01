# Relatorio de Bugfix

- Total de bugs no escopo: 3
- Corrigidos: 3
- Testes de regressao adicionados: 3
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

## Comandos Executados

- `python3 .agents/skills/bugfix/scripts/validate-bug-input.py --input .specs/prd-sdd-robusto/production-gates-bugs.json` -> SUCCESS: 3 bugs validados no formato canonico.
- `go test ./internal/fs ./internal/taskloop ./internal/wrapper -count=1` -> passou.
- `make mocks` -> passou; 40 mocks normalizados por mockery v3.7.4.
- `make check-mocks` -> passou em duas execucoes consecutivas.
- `make lint` -> passou, 0 issues.
- `go test ./... -count=1 -race` -> passou.
- `go vet ./...` -> passou.
- `go build ./...` -> passou.
- `git diff --check` -> passou.
- Revisao manual do delta de BUG-101..103 -> APPROVED, sem achados critical/high/medium/low.

## Riscos Residuais

- `os.Root` e usado em Go 1.26.2+, ja declarado pelo modulo; os gates locais foram
  executados em Go 1.27.0. A matriz remota de CI permanece responsabilidade do
  pipeline de integracao.
- A migracao para mockery v3.7.4 altera a forma do codigo gerado, mas todos os
  mocks permanecem declarados em `mockery.yml`, compilam e sao estaveis em
  regeneracoes consecutivas.
