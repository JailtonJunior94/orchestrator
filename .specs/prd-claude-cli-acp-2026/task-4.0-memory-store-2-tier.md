# Tarefa 4.0: Memory store 2-tier (`internal/runtime/memory/`) com limites configuráveis

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar memory store 2-tier (workflow + task) replicando `compozy/internal/core/memory/store.go`. Workflow em `.specs/<prd>/memory/MEMORY.md` com defaults 150 linhas / 12 KB; task em `.specs/<prd>/memory/<taskFileName>` com defaults 200 linhas / 16 KB. `FileState.NeedsCompaction` sinalizado quando limite ultrapassado. **Sem compactação automática** — diretiva textual prompt-driven entregue na task 6.0.

<requirements>
- Criar `internal/runtime/memory/store.go` com interface `Store` + tipos `Limits`, `FileState`, `Document`, `WriteMode`
- Funções `New(tasksDir, limits) Store`, `ReadWorkflow(ctx)`, `ReadTask(ctx, taskFileName)`, `WriteWorkflow(ctx, content, mode)`, `WriteTask(ctx, taskFileName, content, mode)`
- Constantes default: `DefaultWorkflowLineLimit = 150`, `DefaultWorkflowByteLimit = 12 * 1024`, `DefaultTaskLineLimit = 200`, `DefaultTaskByteLimit = 16 * 1024`
- `FileState.NeedsCompaction` setado quando limite ultrapassado em **qualquer** dos dois eixos (linhas OU bytes)
- `ReadWorkflow`/`ReadTask` retornam `Document{Exists: false}` (sem erro) quando arquivo não existe
- `WriteMode`: `WriteModeReplace` (sobrescreve) ou `WriteModeAppend` (concatena sem trim)
- **Sem compactação automática** nesta task — `NeedsCompaction` é apenas sinal; consumidor (task 6.0) injeta diretiva no prompt
- Esta task **não integra com `runner.go`** — apenas entrega o subpacote standalone
- Cobertura ≥ 80% no subpacote `memory/`
</requirements>

## Subtarefas

- [ ] 4.1 Definir tipos `Limits`, `FileState`, `Document`, `WriteMode` + constantes default
- [ ] 4.2 Implementar `Store` interface + struct concreto
- [ ] 4.3 Implementar `ReadWorkflow`/`ReadTask` com `os.ReadFile` + tratamento de `os.IsNotExist`
- [ ] 4.4 Implementar `WriteWorkflow`/`WriteTask` com `os.WriteFile` (replace) ou `os.OpenFile` flag append
- [ ] 4.5 Calcular `LineCount`/`ByteCount` e setar `NeedsCompaction`
- [ ] 4.6 Helper para criar diretório `.specs/<prd>/memory/` se ausente
- [ ] 4.7 Escrever T-MEM-01..T-MEM-05 (ver techspec §"Abordagem de Testes")

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → "F3-Claude" → `internal/runtime/memory/store.go`. Stub completo no documento. **Não duplicar aqui** — `execute-task` carrega techspec automaticamente.

Pontos críticos:
- **Path resolution**: `Directory(tasksDir) = filepath.Join(tasksDir, "memory")`; `WorkflowPath = filepath.Join(Directory(tasksDir), "MEMORY.md")`; `TaskPath = filepath.Join(Directory(tasksDir), filepath.Base(taskFileName))`. Idêntico Compozy `internal/core/memory/store.go`.
- **`filepath.Base(taskFileName)` é defensivo**: previne traversal `../../../etc/passwd` quando `taskFileName` vem de input externo.
- **`LineCount`**: contar bytes `\n` no conteúdo (não usar `strings.Split` — custo de memória dobrado).
- **`ByteCount`**: `len(content)` — não fazer encoding/decoding intermediário.
- **`NeedsCompaction = LineCount > limits.WorkflowLines || ByteCount > limits.WorkflowBytes` (ou Task equivalentes)** — **OR** sobre os dois eixos.
- **Erro de leitura ≠ ausência**: `os.IsNotExist(err)` retorna `Document{Exists: false, FileState{Path: path}}, nil`. Outros erros propagam com `fmt.Errorf("memory: read %s: %w", path, err)`.
- **Não integrar com `runner.go` nesta task** — task 6.0 faz a integração + flags CLI. Esta task entrega standalone testável.

## Critérios de Sucesso

- T-MEM-01..T-MEM-05 verdes
- `go test ./internal/runtime/memory/... -coverprofile=cov.out` reporta ≥ 80% de cobertura
- T-MEM-02: 151 linhas → `NeedsCompaction=true`
- T-MEM-03: 50 linhas com 300 bytes cada (15000 bytes total > 12288 limite) → `NeedsCompaction=true`
- T-MEM-04: dir sem MEMORY.md → `Document{Exists:false}, err=nil`
- T-MEM-05: append mode preserva conteúdo prévio sem trim
- Sem dependências externas adicionadas em `go.mod`

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: T-MEM-01..T-MEM-05 em `internal/runtime/memory/store_test.go`
- [ ] Cobertura ≥ 80% no subpacote
- [ ] Sem integração com `runner.go` (testes E2E ficam em task 6.0)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **Novo** `internal/runtime/memory/store.go` (~150 LoC)
- **Novo** `internal/runtime/memory/store_test.go` (~200 LoC)
- **Referência:** `compozy/internal/core/memory/store.go` (padrão a replicar)
