# Changelog

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
