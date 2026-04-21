# Changelog

## [0.11.2] - 2026-04-21

### Fixed
- **taskloop:** `matchesTaskPrefix` agora reconhece a convenção `task-X.Y-descricao.md` além das convenções `X-` e `X.Y-`, evitando que tasks com esse padrão sejam ignoradas e bloqueiem toda a cadeia de dependências no task-loop

## 0.11.1 (2026-04-21)

### Bug Fixes
- **taskloop:** isolate Unix syscalls behind build tags for Windows cross-compilation (1a4cfec)

## [0.11.1] - 2026-04-21

### Fixed
- **taskloop:** isolate Unix-only syscall fields (`Setpgid`, `syscall.Kill`) behind `//go:build !windows` to fix cross-compilation failure for `windows_amd64` and `windows_arm64` targets

## 0.11.0 (2026-04-21)

### Features
- **taskloop:** add report summary, update agent flags and inject AGENTS.md in prompt (43d50d0)

### Documentation
- **readme:** add task execution alternatives without task-loop (1ff3f75)
- **readme:** explain task-loop as automatic equivalent of per-session discipline (005c373)
- **readme:** add table of contents with anchor links for all sections (ff32669)
- **readme:** add execute-task per-session discipline with context reset and fidelity rules (adcacaa)
- **readme:** add mandatory skills usage guide with contracts and checklists (e104cab)

## [0.11.0] - 2026-04-21

### Added
- **taskloop:** seção "Resumo" no relatório com contagem de tasks por status (sucesso, puladas, falhadas)
- **taskloop:** instrução explícita de leitura de AGENTS.md no prompt do agente (RF-04, contrato de carga base)
- **taskloop:** flag `--bare` no Claude para pular carregamento automático de CLAUDE.md
- **taskloop:** atualizar flags do Codex para `exec --dangerously-bypass-approvals-and-sandbox`
- **taskloop:** atualizar flags do Copilot para `--autopilot --yolo`

### Fixed
- **taskloop:** guard clause em `ReadTaskFileStatus` para campos de status vazios ou com whitespace

### Changed
- **taskloop:** substituir `os.ReadFile` e `os.Stat` por `fs.FileSystem` em parser e taskloop para testabilidade com FakeFileSystem

### Documentation
- **readme:** adicionar sumário, guia mandatório de skills, disciplina execute-task por sessão e alternativas de execução sem task-loop
- **prompts:** adicionar prompt de execução sequencial automatizada de tasks

## 0.10.4 (2026-04-20)

### Bug Fixes
- **taskloop:** kill process group on cancel and update codex flags (39505f4)

### Documentation
- **readme:** add execute-task and task-loop usage examples (0bb6688)
- **readme:** reorganize sections and expand usage guides (9b4016b)

## 0.10.3 (2026-04-20)

### Bug Fixes
- **brew:** use Ruby chmod method to satisfy Homebrew FormulaAudit (d500dfe)

## 0.10.2 (2026-04-20)

### Bug Fixes
- **brew:** add chmod around xattr to fix post_install permission denied (a52e58a)

## 0.10.1 (2026-04-20)

### Bug Fixes
- **ci:** corrigir smoke test com comando inexistente detect (2c9aeb6)

## 0.10.0 (2026-04-20)

### Features
- **cli:** adicionar skills check, telemetria trend e lint strict — v0.10.0 (3ef9792)

### Documentation
- **readme:** documentar todas as opcoes de correcao do Gatekeeper do macOS (30a3785)
- **readme:** documentar workaround do Gatekeeper do macOS para binario nao assinado (b9d1823)

## 0.10.0 (2026-04-20)

### Added

- **skills check:** novo comando para verificar versões de skills externas contra `skills-lock.json`; detecta upgrades compatíveis (minor/patch) e potenciais quebras de interface (major bump) (`cmd/ai_spec_harness/skills.go`, `internal/skillscheck/`)
- **inspect --brief / --complexity:** exibe referências carregadas por skill por nível de complexidade via `contextgen.Loader` (`cmd/ai_spec_harness/inspect.go`)
- **telemetry report --trend:** evolução semanal de invocações nas últimas 4 semanas em formato texto ou JSON (`internal/telemetry/trend.go`)
- **telemetry report --budget-check:** verifica budget de invocações por skill com saída em texto ou JSON
- **telemetry report --top-skills:** ranking de skills por volume de uso
- **lint --strict:** trata invariantes `BestEffort` de paridade como erros; sem a flag, exibe apenas avisos (`cmd/ai_spec_harness/lint.go`)
- **contextgen.Loader:** carrega referências por skill com suporte a `brief` e `complexity` (`internal/contextgen/loader.go`)
- **requireFlag:** validação de flags obrigatórias com mensagem amigável em PT-BR e exemplo de uso real (`cmd/ai_spec_harness/flags.go`)
- **CodeQL:** workflow de análise de segurança estática adicionado ao CI (`.github/workflows/codeql.yml`)
- **hook validate-token-budget:** verificação de budget de tokens integrada ao Claude Code (`.claude/hooks/validate-token-budget.sh`)
- **docs/troubleshooting.md:** guia de 12 problemas comuns com sintoma, causa, solução e verificação
- **docs/skill-schema.json:** schema JSON para validação de `SKILL.md`

### Changed

- **lint:** detecta tools e langs automaticamente e verifica invariantes `BestEffort` de paridade em toda execução
- **inspect:** integra `contextgen.Loader` para exibir referências por skill nos modos `--brief` e `--complexity`
- **telemetry report:** expandido com três novos modos de saída (`--trend`, `--budget-check`, `--top-skills`)
- **test.yml:** pipeline de CI expandido com novos targets, cobertura por pacote e testes de integração
- **Makefile:** novos targets para fuzz, cobertura e validação de schema
- **skills references:** atualizações em `agent-governance`, `go-implementation` e `object-calisthenics-go`

### Tests

- Fuzz tests adicionados em `internal/config`, `internal/detect` e `internal/manifest`
- Testes de integração para budget de tokens e skills externas (`internal/integration/token_budget_skill_test.go`)
- Contrato CLI expandido com novos subcomandos (`cmd/ai_spec_harness/cli_contract_test.go`)
- Novos testes: `flags_test.go`, `validation_test.go`, `skillscheck_test.go`, `trend_test.go`, `loader_test.go`

---

## 0.9.2 (2026-04-20)

### Bug Fixes
- **brew:** remover quarentena do Gatekeeper via post_install na Formula (3e21aab)

## 0.9.1 (2026-04-20)

### Bug Fixes
- **ci:** corrigir dirty state do GoReleaser por semver_output.txt no workspace (7540c7f)

### Documentation
- **readme:** atualizar instalacao para Formula (brew install) (32206f2)

## [Unreleased]

### Fixed

- **ci:** corrigir dirty state do GoReleaser causado por `semver_output.txt` não rastreado no workspace git

### Tests

- Expandir cobertura de testes unitários em `detect`, `install`, `metrics`, `scaffold`, `uninstall`, `upgrade` e `wrapper`
- Adicionar testes de integração para skills externas e orçamento de tokens
- Adicionar benchmarks para `metrics`, `parity` e `skills/schema`
- Adicionar contrato CLI em `cmd/ai_spec_harness/cli_contract_test.go`

### CI

- Atualizar `test.yml` com melhorias no pipeline de testes
- Adicionar script `scripts/check-package-coverage.sh` para verificação de cobertura por pacote

### Docs

- Adicionar ADR-006: telemetria opt-in com append-only log
- Adicionar ADR-007: workaround stateless para Copilot CLI
- Adicionar ADR-008: paridade multi-tool com 29 invariantes semânticas em 3 níveis
- Adicionar `.aiignore` e `.claudeignore` para controle de contexto dos agentes
- Atualizar governança operacional em `AGENTS.md`, `CLAUDE.md`, `CODEX.md`, `COPILOT.md` e `GEMINI.md`
- Expandir `docs/cli-schema.json` com novos comandos
- Atualizar `Makefile` com novos targets

## 0.9.0 (2026-04-20)

### Features
- **release:** migrar Homebrew de Cask para Formula (c77f61a)

### CI
- **release:** adicionar step para corrigir ordem de stanzas no Homebrew Cask após GoReleaser (9143e49)

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.8.0] - 2026-04-20

### Breaking Changes

- Resetar repositório para catálogo de skills (`65f25ae`)

### Added

- Implementar CLI Go de governança de IA (`7083197`)
- Adicionar distribuição via Homebrew e expandir CLI de governança (`937c15b`)
- Adicionar assets embutidos e validações de spec (`aa62ac9`)
- Separar modelo de custo em três eixos e adicionar gate de regressão — `metrics` (`c004882`)
- Adicionar workflow de dry-run e comandos semver-next e changelog — `release` (`1cd2b1a`)
- Adicionar pacote para resolver git refs em diretório temporário — `gitref` (`e8b29dc`)
- Adicionar flag `--ref` para install e upgrade a partir de git ref (`45e0ce3`)
- Adicionar scoring por focus-paths e suporte a monorepo Python — `detect` (`786d145`)
- Adicionar wrapper e verificação de pré-requisitos de skills (`61b18ae`)
- Expand skills baseline and document task loop flow — `governance` (`6237f5f`)
- Adicionar feedback loop de telemetria, spec-driven e governança multi-agente (`4d7a780`)
- Adicionar Codex, Copilot e parser de telemetria (`66dd041`)

### Fixed

- Usar /tmp para semver_output e ajustar validação de working tree — `release-dry-run` (`e5c3529`)
- Alinhar flags de autonomia total para todas as ferramentas — `taskloop` (`9517323`)
- Corrigir bad substitution ao interpolar mensagem de commit no bash — `ci` (`3c3b0d9`)
