# Tarefa 7.0: Instalação universal transparente (stack-aware + probe + verify)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Eliminar o ajuste manual no `ai-spec install` para P/M/G: derivar a skill de linguagem da detecção de stack, reportar disponibilidade do binário ACP no install (warning não-fatal), garantir stubs por CLI idempotentes e ampliar `verify` para cobrir binário ACP além de skill. Independente do runtime.

<requirements>
- RI-01: quando `opts.Langs` vazio, derivar de `detect.DetectLangs(projectDir)` e instalar a skill de linguagem correta (`AllSkills`) sem flag.
- RI-02: após instalar adaptadores, probe por CLI (`probe.EnsureAvailable`, binário direto ou fallback npx); ausência = `printer.Warn` não-fatal (install não aborta).
- RI-03: stubs por CLI determinísticos e idempotentes (`.claude/`, `.codex/config.toml`, `.gemini/commands/*`, `.github/copilot-instructions.md`) conforme detecção.
- RI-04: `Verify`/`VerifyItem` ganha item `binary` por CLI (current/missing), reusando o probe; saída unificada.
- Bootstrap continua < 30s (RF-11): probe com timeout curto, sem download forçado.
</requirements>

## Subtarefas

- [ ] 7.1 Derivar `opts.Langs` de `DetectLangs` quando vazio (`install.Execute`).
- [ ] 7.2 Probe não-fatal por CLI pós-instalação de adaptadores (warnings).
- [ ] 7.3 Auditar/garantir stubs por CLI idempotentes.
- [ ] 7.4 Estender `Verify`/`VerifyItem` com item `binary` por CLI.
- [ ] 7.5 Testes de integração P (repo vazio), M (Go/Node/Python), G (monorepo) convergindo a 100% `current`.

## Detalhes de Implementação

Ver techspec.md §"Arquitetura do Sistema" (install) e ADR: [024](adr-024-instalacao-transparente-stack-aware.md). Reusar `detect.DetectLangs/DetectTools`, `skills.AllSkills`, `probe.EnsureAvailable` (já presentes). Seguir o padrão de auto-detecção de tools já em `Execute`.

## Critérios de Sucesso

- Repo Go/Node/Python → skill de linguagem correta instalada sem `--langs`.
- Binário ACP ausente → warning não-fatal no install e `missing` no `verify`.
- Reexecução de `install` → `verify` 100% `current` (idempotência) nos cenários P/M/G.
- `make test` + `make integration` verdes; bootstrap < 30s.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/install/install.go` + `install_test.go` + `install_integration_test.go`
- `internal/detect/detect.go` (reuso)
- `internal/runtime/probe/probe.go` (reuso)
- `internal/skills/*` (`AllSkills`)
