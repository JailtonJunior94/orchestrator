# Tarefa 4.0: Registry com cache por instância

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Expor a interface `Registry` (`Discover`, `Resolve`) e sua implementação default `defaultRegistry`, com cache stateful por instância (decisão D-04 do ADR-011 — diverge intencionalmente do cache global de `probe.cache`). Erros de resolução listam candidatos descobertos para serem acionáveis (RF-17).

<requirements>
- `Registry.Discover(ctx) ([]ResolvedAgent, error)` retorna catálogo completo (global + workspace, sem shadowed).
- `Registry.Resolve(name string) (ResolvedAgent, error)` retorna agente único; workspace prevalece.
- Cache válido por tempo de vida da instância; sem TTL; sem reset global (Q2 resolvido em D-04).
- Erro `ErrAgentNotFound` com mensagem listando candidatos descobertos (RF-17).
- Construtor: `NewDefaultRegistry(fsys fs.FileSystem, workdir, home string) Registry`.
- Concorrência segura para `Discover` paralelo (uso de `sync.Once` para inicializar cache).
</requirements>

## Subtarefas

- [ ] 4.1 Criar `internal/agents/registry.go` com interface `Registry` e impl `defaultRegistry`.
- [ ] 4.2 Implementar cache via `sync.Once` (carrega uma vez) — sem `sync.Map` global.
- [ ] 4.3 Implementar `Resolve` consumindo cache; erro acionável quando nome não encontrado.
- [ ] 4.4 Adicionar erro sentinela `ErrAgentNotFound` em `internal/agents/errors.go`.
- [ ] 4.5 Adicionar testes T-14 (cache: 2 chamadas → disco lido 1 vez) e T-19 (resolve não encontrado).

## Detalhes de Implementação

Ver techspec, seção **Design de Implementação → Interfaces Chave** (bloco `internal/agents/registry.go`) e ADR-011 → Decisão D-04.

Estrutura interna sugerida:
```go
type defaultRegistry struct {
    fsys    fs.FileSystem
    workdir string
    home    string
    once    sync.Once
    cached  []ResolvedAgent
    cachedErr error
}
```

`Discover` faz uso de `sync.Once.Do` para popular `cached`/`cachedErr` na primeira chamada.

## Critérios de Sucesso

- T-14: FakeFileSystem com contador comprova que 2 chamadas a `Resolve` lêem disco 1 vez.
- T-19: erro `ErrAgentNotFound` lista candidatos descobertos no formato citado em RF-17.
- Concorrência: 100 goroutines chamando `Discover` em paralelo em um teste produzem o mesmo resultado (sem race detectado por `go test -race`).
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-14: cache de processo é consultado em chamadas subsequentes.
- [ ] T-19: erro `ErrAgentNotFound` lista candidatos descobertos.
- [ ] Teste de concorrência: `go test -race` passa com 100 goroutines.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/agents/registry.go` (novo)
- `internal/agents/registry_test.go` (novo)
- `internal/agents/errors.go` (modificado — adicionar `ErrAgentNotFound`)
