# Tarefa 6.0: Escopo global + Verify file-first + idempotência

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Completar o instalador portátil: escopo de instalação `global` (sob `~/.aispec`/dirs globais
por-agente) além do `project` default; comando `Verify` file-first que reusa o comparador de
checksum do `internal/upgrade` para reportar `current/missing/drifted` por skill/agente; e
formalização da idempotência (install 2× → verify 100% `current`). symlink/copy (RF-08) preservados;
meta de bootstrap < 30s (RF-11). Conforme [ADR-019](adr-019-instalador-portatil-detect-verify.md).

<requirements>
- `InstallScope` (`project` default | `global`) em `InstallOptions`; global resolve destinos via `os.UserHomeDir` (R-SEC-001, paths normalizados/validados).
- `install.Verify(opts)` retornando `[]VerifyItem{Tool, Skill, State}` com `State ∈ {current, missing, drifted}`.
- Reuso do comparador de checksum de `internal/upgrade`: `StatusOK→current`, `StatusMissing→missing`, `StatusOutdated|ContentDivergent→drifted`.
- Idempotência: reexecutar `Install` converge; `Verify` subsequente reporta 100% `current`.
- CLI: subcomando `verify` (ou estender `inspect`) com `--global`; `--dry-run` mostra plano no escopo global.
- `$HOME` ausente: escopo global degrada com erro explícito; projeto permanece default.
- Bootstrap em repo vazio (com agentes detectados, Tarefa 5.0) conclui < 30s.
</requirements>

## Subtarefas

- [ ] 6.1 Adicionar `InstallScope` a `InstallOptions` e resolução de destinos globais (`os.UserHomeDir`).
- [ ] 6.2 Implementar `install.Verify` reusando o comparador de checksum do `internal/upgrade` e o manifesto `.ai_spec_harness.json`.
- [ ] 6.3 Mapear estados do upgrade para `current/missing/drifted`.
- [ ] 6.4 Garantir idempotência de `Install` (convergência) e cobrir com teste install→install→verify.
- [ ] 6.5 CLI: subcomando `verify` + flag `--global` no `install`/`verify`; `--dry-run` no escopo global.
- [ ] 6.6 Testes de integração (`t.TempDir()`, build tag `integration`): bootstrap < 30s, idempotência, escopo global com `HOME` temporário.

## Detalhes de Implementação

Ver `techspec.md` §"Interfaces Chave" (`Verify`, `VerifyState/VerifyItem`), §"Pontos de Integração"
(reuso `upgrade`/manifesto) e [ADR-019](adr-019-instalador-portatil-detect-verify.md). Tocar
`internal/install/install.go`, `internal/config/config.go`, `internal/upgrade/*` (reuso),
`cmd/ai_spec_harness/install.go`/`inspect.go`. Depende da detecção da Tarefa 5.0.

## Critérios de Sucesso

- `install --global` materializa assets em `~/.aispec`/dirs globais (via `HOME` controlado em teste).
- install→install→verify ⇒ 100% `current`; mutar arquivo ⇒ `drifted`; remover ⇒ `missing`.
- Bootstrap em `t.TempDir()` vazio com binários fake conclui < 30s (RF-11).
- symlink default em Unix, copy opcional preservados (RF-08).
- `make test`/`make integration`/`make lint` verdes; cobertura ≥ 75% nos pacotes alterados.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: mapeamento de estados (current/missing/drifted), resolução de destino por escopo, `$HOME` ausente.
- [ ] Testes de integração (`integration`, `t.TempDir()`): bootstrap < 30s, idempotência (install 2×→verify current), escopo global.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/install/install.go` (`Scope`, `Verify`, idempotência)
- `internal/config/config.go` (`InstallScope`, `InstallOptions`)
- `internal/upgrade/*` (comparador de checksum reusado)
- `cmd/ai_spec_harness/install.go`, `cmd/ai_spec_harness/inspect.go`
- testes de integração correspondentes (build tag `integration`)
