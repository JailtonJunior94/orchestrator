# Tarefa 6.0: Política de janela CLI-aware (token-budget + memória)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Tornar o token-budget e a compactação de memória sensíveis à janela de contexto da CLI ativa, usando o VO `ContextWindow`/`WindowClass` (Tarefa 1.0) como sinal **estático por Spec**. Zero-value preserva F1; janelas grandes (Gemini ≥1M) deixam de compactar nos limites conservadores.

<requirements>
- Popular `ContextWindow` nos construtores de catálogo das 4 CLIs (`claude.go`, `codex.go`, `copilot.go`, `gemini.go`); valor estático versionado.
- `TokenBudgetHook` resolve o limite por `Spec`/`WindowClass` (não só por string `Tool`); estender `metrics.ToolBudgets` por driver; `WindowLarge` ⇒ teto generoso; driver sem entrada cai no default atual (sem regressão).
- `memory.WindowPolicy` (domain service stateless): `LimitsFor(class, base) Limits`; `WindowStandard` ⇒ limites F1 (150/12KB · 200/16KB); `WindowLarge` ⇒ limites ampliados.
- Propagar `WindowClass` Spec→Job→hooks/memória; sem leitura de runtime/handshake.
- Override por config pode sobrescrever o default da Spec.
</requirements>

## Subtarefas

- [ ] 6.1 Popular `ContextWindow` no catálogo das 4 specs.
- [ ] 6.2 `TokenBudgetHook` por Spec/classe + estender `metrics.ToolBudgets`.
- [ ] 6.3 `memory/window_policy.go`: `WindowPolicy` + wiring em `prepareMemoryStore` (`runner.go`).
- [ ] 6.4 Propagar `WindowClass` em `Job` (`types.go`) e no runner.
- [ ] 6.5 Testes table-driven por classe (standard/large) + regressão F1 (zero-value).

## Detalhes de Implementação

Ver techspec.md §"Modelagem de Domínio" (VOs de janela, `WindowPolicy` domain service) e ADR: [023](adr-023-window-policy-cli-aware.md). Reusar `metrics.CheckBudget` e `memory.DefaultLimits`.

## Critérios de Sucesso

- Sessão Gemini (large) não compacta nos limites F1; Claude/Codex/Copilot inalterados com zero-value.
- `ContextWindow{}` ⇒ `WindowStandard` ⇒ comportamento token-budget/memória idêntico ao F1.
- Override de janela por config sobrescreve o default da Spec.
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
- `internal/runtime/specs/{claude,codex,copilot,gemini}.go`
- `internal/runtime/hooks/token_budget.go` + `token_budget_test.go`
- `internal/metrics/metrics.go` (`ToolBudgets`)
- `internal/runtime/memory/window_policy.go` (novo) + `window_policy_test.go`
- `internal/runtime/memory/store.go`, `internal/runtime/types.go`, `internal/runtime/runner.go`
