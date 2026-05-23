# Tarefa 6.0: Integrar Options.AgentName e BuildPromptContext no taskloop

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Acoplar o package `internal/agents/` ao `internal/taskloop/`. Adicionar campo `AgentName string` em `taskloop.Options`; quando preenchido, `ProfileConfig` é derivado do `ResolvedAgent` via mapping `runtime.ide → specs.<Tool>()`. Alterar `BuildPromptContext` para aceitar `*agents.ResolvedAgent` opcional e injetar blocos metadata + catálogo no prompt final. Garantir que `AgentName == ""` preserva 100% o fluxo legado.

<requirements>
- Campo `AgentName string` adicionado em `taskloop.Options` sem quebrar callers existentes.
- Quando `AgentName != ""`: registry resolve agente; `ResolveProfileFromAgent(agent, cliOverrides)` produz `ProfileConfig`.
- Mapping `runtime.ide → specs.<Tool>()`: nesta fase, suportar apenas `claude → specs.Claude()`; demais IDEs continuam roteadas via CLI invokers existentes (mantendo retrocompatibilidade dos invokers atuais).
- `BuildPromptContext(prdFolder, workDir, fsys, agent, catalog)` enriquece prompt na ordem: template base → metadata → catálogo → corpo do AGENT.md.
- Quando `agent == nil`, `BuildPromptContext` comporta-se exatamente como hoje (RF-14).
- Diff em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go` deve ser **zero**.
</requirements>

## Subtarefas

- [ ] 6.1 Adicionar `AgentName string` em `taskloop.Options` (`internal/taskloop/taskloop.go:20-42`).
- [ ] 6.2 Adicionar função `ResolveProfileFromAgent(agent agents.ResolvedAgent, override agents.RuntimeOverride) (*ProfileConfig, error)` em `internal/taskloop/profile.go`.
- [ ] 6.3 Alterar `BuildPromptContext` (`internal/taskloop/agent.go:82-96`) para aceitar `*agents.ResolvedAgent` e `[]agents.ResolvedAgent` (catálogo) opcionais.
- [ ] 6.4 Atualizar `taskloop.Service.Run` para resolver agente quando `opts.AgentName != ""` antes de derivar `ProfileConfig`.
- [ ] 6.5 Adicionar testes T-17 (legacy intact), T-18 (agent flow), T-19 (agent not found).

## Detalhes de Implementação

Ver techspec, seção **Arquitetura → Relacionamentos e Fluxo de Dados** e ADR-011 → Decisões D-03 (coexistência) e D-05 (precedência).

Mapping mínimo:
```go
func mapIDEToSpec(ide string) (specs.Spec, error) {
    switch ide {
    case "claude":
        return specs.Claude(), nil
    default:
        return specs.Spec{}, fmt.Errorf("ide %q ainda não suportado em ACP — use CLI invoker", ide)
    }
}
```

Para `codex/gemini/copilot` declarados em `AGENT.md`, nesta fase a tarefa deve documentar comportamento: caminho ACP retorna erro acionável e usuário deve usar `--tool` legado. Não é regressão — multi-IDE via ACP é Fase 5.

## Critérios de Sucesso

- T-17: `Options.AgentName == ""` → zero chamadas a registry (verificável por mock); fluxo legado intacto.
- T-18: `Options.AgentName == "foo"` válido → registry resolve → produz `Spec` → executa ACPRunner com prompt enriquecido.
- T-19: `Options.AgentName == "foo"` não encontrado → erro acionável citando candidatos.
- Suíte atual de `internal/taskloop/...` permanece verde.
- **Diff zero** em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go` (auditado manualmente).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-17: fluxo legado intacto sem `AgentName`.
- [ ] T-18: fluxo de agent resolve → executa.
- [ ] T-19: erro de agent não encontrado.
- [ ] Suíte completa `go test ./internal/taskloop/...` permanece verde.
- [ ] Auditoria manual: `git diff internal/runtime/persistence/ internal/runtime/watchdog.go` retorna vazio.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/taskloop/taskloop.go` (modificado — campo `AgentName`)
- `internal/taskloop/profile.go` (modificado — `ResolveProfileFromAgent`)
- `internal/taskloop/agent.go` (modificado — `BuildPromptContext` assinatura)
- `internal/taskloop/taskloop_test.go` (modificado)
- `internal/taskloop/profile_test.go` (modificado)
- `internal/taskloop/agent_test.go` (modificado)
