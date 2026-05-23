# Tarefa 5.0: Launcher Probe

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar `internal/runtime/probe/`, responsável pela fase `EnsureAvailable` exigida pelo RF-03: resolver o launcher do `claude-agent-acp` na ordem (a) binário canônico no PATH; (b) `npx --yes @agentclientprotocol/claude-agent-acp@<VER>`; (c) falhar com mensagem contendo três remédios. Cache em memória por processo (`sync.OnceValue`-style) para evitar re-probing por task. Sem importação de `coder/acp-go-sdk`.

<requirements>
- `Prober` interface no caller (a definir na task 9.0); esta task entrega a implementação concreta `defaultProber`.
- Resolução **antes** de qualquer escrita em `evidence/`; falha rápida (R-DDD-001 "fail fast").
- Cache em memória por processo: primeira resolução fica disponível para tasks subsequentes na mesma invocação CLI.
- Mensagem de erro literal: `claude-agent-acp não encontrado. Install claude-agent-acp; OR install @agentclientprotocol/claude-agent-acp@<VER> via npm; OR use --runtime=legacy. Veja .specs/adr/009-acp-protocol-adoption.md`.
- `LookPather` interface injetável para testes (não chama `exec.LookPath` direto em testes).
- Sentinel `ErrLauncherUnavailable` definido no pacote `runtime` (criado em 6.0) ou local.
</requirements>

## Subtarefas

- [ ] 5.1 Criar `internal/runtime/probe/lookpath.go` com interface `LookPather { LookPath(name string) (string, error) }` e implementação default `osLookPather struct{}` que delega para `exec.LookPath`.
- [ ] 5.2 Criar `internal/runtime/probe/probe.go` com `func EnsureAvailable(ctx context.Context, spec specs.Spec, look LookPather) (specs.Launcher, error)`.
- [ ] 5.3 Implementar resolução ordenada: `look.LookPath(spec.Command)` → sucesso retorna `specs.NewBinaryLauncher(...)`; senão tenta cada `spec.Fallbacks` (provavelmente apenas `npx`) verificando se `npx` está no PATH; se ainda falhar, retorna `ErrLauncherUnavailable` wrappado com mensagem dos 3 remédios.
- [ ] 5.4 Adicionar cache via `sync.OnceValues[specs.Launcher, error]` por par `(spec.ID)`: armazenar em `sync.Map` interno do pacote; primeira chamada de cada spec resolve, subsequentes reusam.
- [ ] 5.5 Definir/coordenar com task 6.0 o sentinel `ErrLauncherUnavailable` (criar em `internal/runtime/errors.go` se 6.0 ainda não tiver entrado; caso contrário, importar de lá).
- [ ] 5.6 Criar `probe_test.go` com cenários: (a) binário canônico encontrado; (b) só npx disponível; (c) nem binário nem npx; (d) cache: segunda chamada não invoca `LookPath` novamente.

## Detalhes de Implementação

Ver `techspec.md`:
- §"Design de Implementação" → "Interfaces Chave" (`Prober`)
- §"Pontos de Integração" → "claude-agent-acp (subprocesso)" (decisão #19: cache em memória)
- §"Estratégia de Erros" → linha `ErrLauncherUnavailable`
- PRD RF-03 para a forma exata da mensagem e ordem de fallback

## Critérios de Sucesso

- `go test ./internal/runtime/probe/...` ≥ 90% cobertura.
- Mensagem de erro literal idêntica ao especificado no RF-03.
- Cache validado em teste: duas chamadas com a mesma spec invocam `LookPath` apenas uma vez.
- Sem chamadas diretas a `exec.LookPath` em código de produção (todas via `LookPather`).
- Sem import de `coder/acp-go-sdk`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Tabela `TestEnsureAvailable` cobrindo: binary OK; só npx; nenhum (erro com mensagem exata)
- [ ] `TestEnsureAvailable_Cache`: contador de invocações de `LookPath` mockado; assertar uma única chamada por spec
- [ ] `TestEnsureAvailable_ContextCanceled`: contexto cancelado antes da chamada retorna erro de contexto, não tenta probe

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/probe/lookpath.go` (novo)
- `internal/runtime/probe/probe.go` + `probe_test.go` (novo)
- `internal/runtime/errors.go` (novo ou modificado, depende da ordem com 6.0)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-5.0/execution_report.md`
- [ ] `go test ./internal/runtime/probe/... -count=1 -race -cover` ≥ 90%
- [ ] `golangci-lint run ./internal/runtime/probe/...` sem violações
- [ ] Mensagem de erro confere com RF-03 (validado por test golden de string)
- [ ] Commit semântico `feat(runtime/probe): resolve claude-agent-acp launcher with npx fallback and cache`
