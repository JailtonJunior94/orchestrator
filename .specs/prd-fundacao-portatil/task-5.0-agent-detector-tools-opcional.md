# Tarefa 5.0: AgentDetector + `--tools` opcional

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Habilitar bootstrap zero-fricção: novo `detect.AgentDetector` que detecta agentes presentes por
binário ACP no PATH (reusando `internal/runtime/specs`/probe) combinado com diretórios de config
conhecidos e arquivos de projeto (sinal já coberto por `FileDetector`). O flag `--tools` torna-se
opcional — ausente instala nos agentes detectados; presente é override explícito. Conforme
[ADR-019](adr-019-instalador-portatil-detect-verify.md).

<requirements>
- `detect.AgentDetector.Detect(ctx, opts)` combinando: binário ACP no PATH + dirs de config (`~/.claude`, `~/.codex`, `~/.gemini`, ...) + arquivos de projeto.
- Reusar nomes de comando das specs (`internal/runtime/specs`) — não duplicar literais.
- `LookPath` apenas (não executar binários); `os.UserHomeDir` para dirs de config (R-SEC-001).
- `InstallOptions.Tools` opcional: vazio ⇒ auto-detect; preenchido ⇒ override (precedência de flag, ADR-016).
- CLI `install`: `--tools` deixa de ser obrigatório; manter modo não-interativo como base (RF-12).
- Em repo vazio com binários presentes: detecção instala nos agentes corretos sem `--tools`.
</requirements>

## Subtarefas

- [ ] 5.1 Criar `internal/detect/agent.go` com `AgentDetector` e `DetectOptions`.
- [ ] 5.2 Implementar detecção por binário no PATH reusando `specs` + dirs de config via `os.UserHomeDir`.
- [ ] 5.3 Integrar com `FileDetector` existente (arquivos de projeto) como sinal adicional.
- [ ] 5.4 Tornar `InstallOptions.Tools` opcional; `install.Service.Execute` chama detecção quando vazio.
- [ ] 5.5 Ajustar `cmd/ai_spec_harness/install.go`: `--tools` opcional (default = detectados; flag = override).
- [ ] 5.6 Testes: LookPath fake detecta; só arquivo de projeto; repo vazio sem binários ⇒ vazio; flag override.

## Detalhes de Implementação

Ver `techspec.md` §"Interfaces Chave" (`AgentDetector`) e [ADR-019](adr-019-instalador-portatil-detect-verify.md).
Tocar `internal/detect/detect.go` (reuso `FileDetector`), novo `internal/detect/agent.go`,
`internal/install/install.go` (validação de `Tools`), `internal/config/config.go` (`InstallOptions`),
`cmd/ai_spec_harness/install.go`. Depende da precedência de flag/config da Tarefa 1.0.

## Critérios de Sucesso

- `install` sem `--tools` num ambiente com binários ⇒ instala nos agentes detectados.
- `--tools` presente sobrescreve a detecção (override explícito).
- Detecção usa `LookPath` (sem executar agentes) e `os.UserHomeDir` (sem hardcode).
- `make test`/`make lint` verdes; cobertura ≥ 75% nos pacotes alterados.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: detecção por binário (LookPath fake), por arquivo, vazio sem binários, override por flag.
- [ ] Testes de integração: opcional — detecção em `t.TempDir()` com PATH controlado (consolidável na Tarefa 6.0).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/detect/agent.go` (novo)
- `internal/detect/detect.go` (`FileDetector`)
- `internal/install/install.go` (validação de `Tools`)
- `internal/config/config.go` (`InstallOptions.Tools` opcional)
- `cmd/ai_spec_harness/install.go`
- `internal/runtime/specs/*.go` (nomes de comando reusados)
