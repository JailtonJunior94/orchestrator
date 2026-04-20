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
- **Cobertura:** threshold minimo de 75% enforced no CI (cobertura atual: ~76.7%)
  - Gerar relatorio local: `make coverage`

## Padroes importantes

- `internal/fs/fake.go` — usar FakeFileSystem em testes unitarios, nunca OS real
- `internal/output.Printer` — injetar em todo Service, usar `io.Discard` em testes
- Build tag `integration` para testes que tocam o filesystem real
- Skills externas rastreadas em `skills-lock.json` com hash SHA-256; regras de upgrade em `AGENTS.md` seção "Upgrades de Skills Externas"
- Harness auto-instalado neste repo (self-dogfooding)
- Telemetria: ativar com `GOVERNANCE_TELEMETRY=1`; consultar ciclo completo em [`docs/telemetry-feedback-cycle.md`](docs/telemetry-feedback-cycle.md); gerar relatorio com `ai-spec-harness telemetry report`
- Relatorios de auditoria e execution_reports de ciclos completos devem ser salvos em `audit/` e indexados em [`audit/README.md`](audit/README.md)

## Decisoes Arquiteturais

ADRs documentam decisoes significativas que nao sao obvias no codigo. Consultar antes de propor mudancas estruturais.

- [`tasks/adr/001-go-embed-baseline.md`](tasks/adr/001-go-embed-baseline.md) — Por que assets de governanca sao embarcados no binario via `go:embed`
- [`tasks/adr/002-fake-filesystem-testes.md`](tasks/adr/002-fake-filesystem-testes.md) — Por que usar FakeFileSystem customizado em vez de afero ou mocks gerados
- [`tasks/adr/003-paridade-semantica.md`](tasks/adr/003-paridade-semantica.md) — Por que verificar paridade por invariantes semanticas e nao por diff textual
- [`tasks/adr/004-lazy-loading-referencias.md`](tasks/adr/004-lazy-loading-referencias.md) — Por que referencias de skills sao carregadas sob demanda
- [`tasks/adr/005-skills-lock-sha256.md`](tasks/adr/005-skills-lock-sha256.md) — Por que skills externas usam lock file com SHA-256
- [`docs/adr/006-telemetria-feedback-cycle.md`](docs/adr/006-telemetria-feedback-cycle.md) — Por que telemetria e opt-in com append-only log
- [`docs/adr/007-copilot-cli-stateless-workaround.md`](docs/adr/007-copilot-cli-stateless-workaround.md) — Por que Copilot CLI usa workaround de injecao manual
- [`docs/adr/008-parity-multi-tool-invariants.md`](docs/adr/008-parity-multi-tool-invariants.md) — Por que paridade usa 29 invariantes semanticas com 3 niveis

Template para novas ADRs: [`tasks/adr/000-template.md`](tasks/adr/000-template.md)
