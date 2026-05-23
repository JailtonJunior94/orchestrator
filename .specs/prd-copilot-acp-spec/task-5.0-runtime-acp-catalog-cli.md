# Tarefa 5.0: Tabela runtimeACPCatalog em cmd/ai_spec_harness/task_loop.go

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Substituir o gating literal `"claude"` em `cmd/ai_spec_harness/task_loop.go:77` por uma tabela `runtimeACPCatalog map[string]func() specs.Spec`. A tabela é a fonte de verdade para quais tools podem usar `--runtime=acp` nesta versão do harness. Entradas iniciais: `"claude" → specs.Claude`, `"copilot" → specs.Copilot`.

Mensagem de erro quando tool fora do catálogo é usado com `--runtime=acp` lista as tools suportadas em ordem lexicográfica.

<requirements>
- Tabela map[string]func() specs.Spec em cmd/ai_spec_harness/ (responsabilidade do CLI, não de specs/).
- Validação --runtime=acp consulta a tabela em vez de comparar literal contra "claude".
- Mensagem de erro lista tools suportados ordenados quando rejeitado.
- Comportamento Claude preservado (T-15 regressão).
- Copilot ACP aceito (T-13).
- Outros tools rejeitados com mensagem clara (T-14).
</requirements>

## Subtarefas

- [ ] 5.1 Criar variável `runtimeACPCatalog = map[string]func() specs.Spec{"claude": specs.Claude, "copilot": specs.Copilot}` em `cmd/ai_spec_harness/task_loop.go` (ou em arquivo adjacente do mesmo package).
- [ ] 5.2 Substituir o branch atual (`if effectiveTool != "claude" { ... }`) por lookup em `runtimeACPCatalog`.
- [ ] 5.3 Construir mensagem de erro listando chaves ordenadas (`sort.Strings`) quando tool fora do catálogo.
- [ ] 5.4 Adicionar testes T-13 (Copilot ACP aceito), T-14 (Gemini ACP rejeitado com lista), T-15 (Claude ACP regressão).
- [ ] 5.5 Validar suíte completa de `cmd/ai_spec_harness/`.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `cmd/ai_spec_harness/task_loop.go — tabela runtimeACPCatalog`. Decisão D-04 (tabela em `cmd/` não em `specs/` — responsabilidade do CLI).

Anti-padrão: NÃO duplicar a tabela em `internal/taskloop/`; o catálogo é responsabilidade do CLI. `Service.Execute` (Tarefa 6.0) deve consumir o catálogo via parâmetro ou pacote exposto, não redeclará-lo.

## Critérios de Sucesso

- `ai-spec-harness task-loop --tool copilot --runtime acp <prd>` passa pela validação CLI (não falha em validação de flags).
- `ai-spec-harness task-loop --tool gemini --runtime acp <prd>` falha com mensagem `"runtime acp suporta apenas --tool em [claude copilot] nesta versão"` (ou equivalente com lista ordenada).
- `ai-spec-harness task-loop --tool claude --runtime acp <prd>` continua funcionando idêntico ao comportamento atual.
- Testes T-13/T-14/T-15 verdes em `task_loop_test.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-13: `--tool copilot --runtime acp` → validação CLI passa, RunE prossegue para `taskloop.Service.Execute`.
- [ ] T-14: `--tool gemini --runtime acp` → erro com `exit2`, stderr contém `[claude copilot]` ordenado.
- [ ] T-15: `--tool claude --runtime acp` → validação CLI passa (regressão).
- [ ] Validação inversa: `--tool copilot` (sem `--runtime acp`) → continua roteando para legacy (Tarefa 7.0 cobre warning).
- [ ] `go test ./cmd/ai_spec_harness/...` → 100% verde.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `runtimeACPCatalog` declarado como `var` em `cmd/ai_spec_harness/task_loop.go` (ou arquivo adjacente).
- [ ] Validação `--runtime=acp` consulta o catálogo via `_, ok := runtimeACPCatalog[effectiveTool]`.
- [ ] Mensagem de erro lista as chaves ordenadas (slices.Sorted ou sort.Strings).
- [ ] Catálogo não é redeclarado em `internal/taskloop/` (D-04).
- [ ] T-13/T-14/T-15 verdes em `task_loop_test.go`.
- [ ] `go test ./cmd/ai_spec_harness/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.
- [ ] Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`.

## Arquivos Relevantes

- `cmd/ai_spec_harness/task_loop.go:67-110` (modificar — bloco de validação `--runtime`)
- `cmd/ai_spec_harness/task_loop_test.go` (estender com T-13/T-14/T-15)
- `internal/runtime/specs/copilot.go` (consumido — Tarefa 2.0)
- `internal/runtime/specs/claude.go` (consumido)
- ADR-012 §"Decisão D-04" (catálogo em `cmd/`)
