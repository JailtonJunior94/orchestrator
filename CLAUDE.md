# ai-spec-harness

CLI para instalar, validar, inspecionar e atualizar governanca operacional de IA em repositorios de software. Suporta Claude, Gemini, Codex e GitHub Copilot.

## Stack

- **Linguagem:** Go 1.26+
- **CLI framework:** spf13/cobra
- **Testes:** go test (unit com FakeFileSystem, integration com build tag `integration`)
- **Release:** GoReleaser + GitHub Actions + Homebrew Cask
- **Dependencias diretas:** cobra, jsonschema/v6

## Comandos

```bash
make test            # testes unitarios
make integration     # testes de integracao
make lint            # golangci-lint
make build           # compila binario
make vet             # go vet
```

## Convencoes

- **Idioma do codigo:** comentarios, erros e mensagens em portugues (PT-BR)
- **Commits:** Conventional Commits com tipo em ingles e corpo em portugues
- **Testes:** table-driven, FakeFileSystem para unit, t.TempDir() para integration
- **DI:** toda dependencia injetada via construtor, zero estado global
- **Erros:** sempre envolver com `fmt.Errorf("contexto: %w", err)`
- **Pacotes internos:** cada pacote em `internal/` com responsabilidade unica
- **Interfaces:** definir no pacote consumidor quando possivel

## Estrutura

```
cmd/ai_spec_harness/   # comandos cobra (CLI)
internal/              # logica de negocio (34 pacotes)
  fs/                  # abstracao de filesystem (OSFileSystem + FakeFileSystem)
  output/              # printer estruturado (Info, Step, Warn, Error, Debug)
  config/              # DTOs de configuracao
  skills/              # tipos de dominio (Tool, Lang, LinkMode)
  install/             # servico de instalacao
  upgrade/             # servico de upgrade
  detect/              # deteccao de linguagens e ferramentas
  metrics/             # estimativa de tokens e custo
  parity/              # paridade semantica multi-agente
  specdrift/           # deteccao de drift entre specs
  evidence/            # validacao de relatorios
  taskloop/            # orquestracao de execucao de tasks
testdata/              # fixtures, snapshots, projetos-exemplo
.agents/skills/        # skills externas (gerenciadas via skills-lock.json)
internal/embedded/     # assets embarcados (go:embed) - baseline de governanca
```

## CI

- **test.yml:** unit + integration + golangci-lint em ubuntu-24.04 e macos-15
- **release.yml:** semver-next automatico, GoReleaser multi-plataforma
- **release-dry-run.yml:** validacao de release sem side-effects

## Padroes importantes

- `internal/fs/fake.go` — usar FakeFileSystem em testes unitarios, nunca OS real
- `internal/output.Printer` — injetar em todo Service, usar `io.Discard` em testes
- Build tag `integration` para testes que tocam o filesystem real
- Skills externas rastreadas em `skills-lock.json` com hash SHA-256
- Harness auto-instalado neste repo (self-dogfooding)
