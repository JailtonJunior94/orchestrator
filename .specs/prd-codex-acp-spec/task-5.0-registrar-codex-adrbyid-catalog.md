# Tarefa 5.0: Registrar codex em adrByID e runtimeACPCatalog

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Registrar entrada Codex em **duas tabelas de roteamento** que controlam:

1. **`adrByID`** em `internal/runtime/probe/probe.go:21-24` — mapping `spec.ID → ADR path` consumido por `formatLauncherUnavailable(spec, adrPath)`. Adicionar `"codex": ".specs/adr/013-codex-cli-acp-native.md"` ao map. Mensagem de erro de probe Codex passa a referenciar ADR-013.

2. **`runtimeACPCatalog`** em `cmd/ai_spec_harness/task_loop.go:21-24` — mapping `tool name → func() Spec` consumido pelo gate `--runtime=acp`. Adicionar `"codex": specs.Codex` ao map. Linha 82-97 (gate) passa a aceitar Codex automaticamente sem nova lógica.

Mudança cirúrgica: 2 arquivos, ~4 linhas alteradas no total. Mas é **precondição cirúrgica** para tarefas 6.0/7.0/8.0 — sem ela, `task_loop.go:82-97` continua rejeitando `--tool codex --runtime acp` com `exit2` (teste T-14 atual continua passando como rejeição).

<requirements>
- adrByID["codex"] aponta para ADR-013 com path relativo correto.
- runtimeACPCatalog["codex"] aponta para função `specs.Codex` (não chamada — referência).
- Mensagem de erro de probe Codex segue padrão Claude/Copilot referenciando ADR.
- Gate em task_loop.go ainda valida `--reasoning-effort` e `--access-mode` na tarefa 6.0 (não aqui).
- Testes T-13/T-14/T-15/T-16/T-22 começam a validar Codex aceito.
- ATENÇÃO: T-14 atual em task_loop_test.go:48-52 valida rejeição de codex — será **invertido na tarefa 6.0** (não nesta).
</requirements>

## Subtarefas

- [ ] 5.1 Editar `internal/runtime/probe/probe.go:21-24` adicionando `"codex": ".specs/adr/013-codex-cli-acp-native.md"` ao map `adrByID`.
- [ ] 5.2 Editar `cmd/ai_spec_harness/task_loop.go:21-24` adicionando `"codex": specs.Codex` ao map `runtimeACPCatalog`.
- [ ] 5.3 Verificar que `cmd/ai_spec_harness/task_loop.go` já importa `internal/runtime/specs` (deveria, já que Claude/Copilot estão no catálogo). Se necessário, adicionar import.
- [ ] 5.4 Atualizar comentário do `runtimeACPCatalog` mencionando os 3 runtimes suportados nesta versão.
- [ ] 5.5 Rodar `go build ./...` para garantir compilação.
- [ ] 5.6 Rodar `go test ./internal/runtime/probe/...` e `./cmd/ai_spec_harness/...` (T-14 ainda esperado falhar — invertido na tarefa 6.0).
- [ ] 5.7 Verificar manualmente: `go run ./cmd/ai_spec_harness task-loop --tool codex --runtime acp .specs/prd-x` agora passa o gate (mas pode falhar em outro ponto enquanto tarefa 6.0 não tiver flags).

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `cmd/ai_spec_harness/task_loop.go — flags + catálogo` e §"Sequenciamento de Desenvolvimento" → item 5/6. Decisão registrada em ADR-013 D-04 (catálogo continua em `cmd/`, não em `specs/`) e §"Pontos de Integração" item probe/adrByID.

Anti-padrão: NÃO usar `specs.Codex()` (chamada) no map — usar referência `specs.Codex` (função sem parênteses).

## Critérios de Sucesso

- `adrByID["codex"]` retorna `".specs/adr/013-codex-cli-acp-native.md"`.
- `runtimeACPCatalog["codex"]` retorna função que invocada produz Spec Codex com `ID="codex"`.
- `go build ./...` compila sem erros.
- Mensagem de erro de probe Codex (testado em tarefa subsequente via T-13) referencia ADR-013.
- Mensagem do gate `--runtime acp` lista `[claude codex copilot]` em ordem lexicográfica (T-14 atual ainda rejeita codex; será invertido em 6.0).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-16: `adrByID["codex"] == ".specs/adr/013-codex-cli-acp-native.md"`.
- [ ] Probe error com Spec Codex referencia ADR-013 (T-13 cobre quando launcher unavailable).
- [ ] T-15 (Claude regressão): `runtimeACPCatalog["claude"]` continua resolvendo para Spec Claude (testar via factory).
- [ ] T-13 (Copilot regressão): `runtimeACPCatalog["copilot"]` continua resolvendo (sanity).
- [ ] `go build ./...` sem erros.
- [ ] `go vet ./...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `internal/runtime/probe/probe.go:21-24` contém linha `"codex": ".specs/adr/013-codex-cli-acp-native.md"`.
- [ ] `cmd/ai_spec_harness/task_loop.go:21-24` contém linha `"codex": specs.Codex`.
- [ ] Imports atualizados se necessário (sem ciclo).
- [ ] Comentário do `runtimeACPCatalog` reflete catálogo de 3 tools (Claude, Codex, Copilot).
- [ ] `go build ./...` → sem erros.
- [ ] `go test ./internal/runtime/probe/... ./cmd/ai_spec_harness/...` → testes existentes mantêm comportamento (T-14 ainda rejeita Codex porque inversão acontece em 6.0).
- [ ] `git diff` mostra apenas 2 arquivos modificados.

## Arquivos Relevantes

- `internal/runtime/probe/probe.go` (modificar linha 21-24)
- `cmd/ai_spec_harness/task_loop.go` (modificar linha 21-24)
- `internal/runtime/specs/codex.go` (consumir `specs.Codex` da tarefa 2.0)
- ADR-013 §"Decisão" → D-04 (catálogo em CLI, não em specs)
- techspec.md §"Pontos de Integração" → `probe.go`/`task_loop.go`
