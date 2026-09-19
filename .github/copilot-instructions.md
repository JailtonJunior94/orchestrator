Voce esta trabalhando no ai-spec-harness, uma CLI Go para governanca operacional de IA.

## Stack

- Go 1.27, spf13/cobra, coder/acp-go-sdk, jsonschema/v6, yaml.v3
- Testes: go test com FakeFileSystem (unit) e build tag `integration`
- Release: GoReleaser + GitHub Actions + Homebrew Cask

## Convencoes

- Codigo em ingles e sem comentarios (R-STYLE-001, hard — ver `.claude/rules/code-style.md`); mensagens de erro em ingles
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
- `internal/` — logica de negocio, um pacote por responsabilidade
- `internal/fs/fake.go` — FakeFileSystem para testes
- `testdata/` — fixtures e snapshots
