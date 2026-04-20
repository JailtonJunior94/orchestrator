Voce esta trabalhando no ai-spec-harness, uma CLI Go para governanca operacional de IA.

## Stack

- Go 1.26+, spf13/cobra, jsonschema/v6
- Testes: go test com FakeFileSystem (unit) e build tag `integration`
- Release: GoReleaser + GitHub Actions + Homebrew Cask

## Convencoes

- Comentarios e mensagens de erro em PT-BR
- Commits: Conventional Commits (tipo ingles, corpo portugues)
- Testes table-driven, DI via construtor, zero estado global
- Erros: `fmt.Errorf("contexto: %w", err)`

## Comandos

- `make test` — testes unitarios
- `make integration` — testes de integracao
- `make lint` — golangci-lint
- `make build` — compila binario

## Estrutura

- `cmd/ai_spec_harness/` — comandos Cobra
- `internal/` — logica de negocio (34 pacotes)
- `internal/fs/fake.go` — FakeFileSystem para testes
- `testdata/` — fixtures e snapshots
