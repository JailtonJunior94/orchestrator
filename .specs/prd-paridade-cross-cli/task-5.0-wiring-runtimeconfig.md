# Tarefa 5.0: Wiring do config hierárquico no RuntimeConfig

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Conectar o resolver hierárquico (ADR-016) ao `RuntimeConfig` (ADR-018) na fronteira de aplicação, garantindo que a precedência `flags > workspace > global > defaults` seja aplicada de forma idêntica para as 4 CLIs antes de `runner.Run()`. Independente do runner (não toca `runner.go`).

<requirements>
- `BuildRuntimeConfig(resolved config.Runtime) (runtime.RuntimeConfig, error)` em `internal/taskloop`: mapeia tipos (`time.ParseDuration(Timeout)`→`events.ActivityTimeout`; numéricos 1:1) e chama `RuntimeConfig.ApplyDefaults()`.
- `taskloop` resolve a config uma vez (via `config.Resolver.Resolve(cwd, flagsOverrides)`) e injeta o mesmo `RuntimeConfig` nos `Job` das 4 CLIs.
- Flags CLI entram como `overrides` (camada de maior precedência).
- `Timeout` malformado ⇒ erro descritivo (`fmt.Errorf("timeout inválido: %w")`), fail-fast na fronteira.
- Zero-value em qualquer camada preserva F1 (`DefaultRuntime`).
</requirements>

## Subtarefas

- [ ] 5.1 `taskloop/runtimeconfig.go`: `BuildRuntimeConfig` (mapeamento + parse + `ApplyDefaults`).
- [ ] 5.2 Integrar `config.Resolver.Resolve` no `taskloop` (uma resolução, injeção nas 4 CLIs).
- [ ] 5.3 Flags CLI → overrides no `Resolve`.
- [ ] 5.4 Testes: precedência (flags>workspace>global>defaults), timeout vazio/válido/malformado, regressão F1, paridade entre os 4 drivers.

## Detalhes de Implementação

Ver techspec.md §"Design de Implementação" (`BuildRuntimeConfig`) e ADR: [025](adr-025-runtimeconfig-wiring-acprunner.md). Reusar `config.DefaultResolver.Resolve` e `runtime.RuntimeConfig.ApplyDefaults` (já testados). O `ACPRunner.Run` **não** muda de assinatura.

## Critérios de Sucesso

- Alterar `concurrent` em `.claude/config.yaml` reflete igual nas 4 CLIs; flag CLI vence config.
- Ausência de config + flags ⇒ `DefaultRuntime` ⇒ F1 exato.
- `Timeout` malformado falha na fronteira com mensagem clara.
- `make test` verde.

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
- `internal/taskloop/runtimeconfig.go` (novo) + `runtimeconfig_test.go`
- `internal/taskloop/runloop.go`
- `internal/config/resolver.go`, `internal/config/runtime.go` (reuso)
- `internal/runtime/types.go` (`RuntimeConfig.ApplyDefaults`)
