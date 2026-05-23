# Tarefa 8.0: Codex compat em taskloop — codexInvoker warning + CompatibilityTable

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar duas mudanças complementares no layer `internal/taskloop/` para completar awareness Codex:

1. **Aviso de depreciação no `codexInvoker`** (`internal/taskloop/agent.go:335-351`): adicionar variável package-level `codexLegacyWarnOnce sync.Once` e emitir WARNING em stderr na primeira invocação por execução do processo, anunciando depreciação e referenciando ADR-013. Mensagem: `"WARNING: Codex CLI legado (codex exec --yolo) em uso. Migrar para --runtime=acp (binário codex-acp, pacote @zed-industries/codex-acp). O modo legado será removido em 2 versões minor. Ver ADR-013."`

2. **Entrada Codex em `CompatibilityTable`** (`internal/taskloop/compatibility.go`): adicionar mínima `"codex": []string{"gpt-5.5"}` ao map. Outros modelos requerem `--allow-unknown-model` (semântica existente preservada).

Ambas as mudanças são pequenas (~10-30 LoC cada), ambas no package `internal/taskloop/`, ambas dependem da tarefa 6.0, ambas concorrem para "taskloop-layer awareness de Codex". Empacotadas em uma única task por delivery slice coerente.

<requirements>
- codexLegacyWarnOnce sync.Once em escopo package-level (não local à função).
- Warning emitido exatamente uma vez por execução do processo (não por task).
- Mensagem do warning referencia explicitamente ADR-013 e o binário codex-acp.
- CompatibilityTable["codex"] contém ["gpt-5.5"] (mínimo).
- --allow-unknown-model continua aceitando modelos arbitrários para Codex.
- T-28 (warning único) e T-29 (segunda invocação sem warning) cobrem sync.Once.
- T-34 (gpt-5.5 aceito) e T-35 (outro modelo rejeitado sem flag) cobrem CompatibilityTable.
- Diff zero em persistence/, watchdog.go, client/.
- Modo ACP (--runtime=acp) NÃO chama codexInvoker — warning aparece apenas no legacy path.
</requirements>

## Subtarefas

- [ ] 8.1 Declarar `var codexLegacyWarnOnce sync.Once` em escopo package em `internal/taskloop/agent.go`.
- [ ] 8.2 Modificar `codexInvoker.Invoke(...)` adicionando `codexLegacyWarnOnce.Do(func() { fmt.Fprintln(os.Stderr, "WARNING: ...") })` no início.
- [ ] 8.3 Mensagem canônica do warning conforme techspec §"Design de Implementação" → bloco `codexInvoker.Invoke`.
- [ ] 8.4 Editar `internal/taskloop/compatibility.go::CompatibilityTable` adicionando entrada `"codex": []string{"gpt-5.5"}` (ou estrutura equivalente conforme padrão atual).
- [ ] 8.5 Verificar que `validateModelForTool("codex", "gpt-5.5", allowUnknown=false)` aceita.
- [ ] 8.6 Verificar que `validateModelForTool("codex", "gpt-4", allowUnknown=false)` rejeita (T-35).
- [ ] 8.7 Adicionar testes T-28/T-29 em `internal/taskloop/agent_test.go`.
- [ ] 8.8 Adicionar testes T-34/T-35 em `internal/taskloop/compatibility_test.go`.
- [ ] 8.9 Rodar `go test ./internal/taskloop/...` → 100% verde.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/taskloop/agent.go — codexInvoker com sync.Once warning` (esboço completo) e §"Sequenciamento de Desenvolvimento" → item 10. Decisão registrada em ADR-013 D-05 (caminho legado mantido por 2 versões minor) e D-10 (CompatibilityTable mínima).

Anti-padrão: NÃO declarar `sync.Once` dentro da função (perde semântica entre invocações no mesmo processo); usar variável package-level.

## Critérios de Sucesso

- Primeira chamada a `codexInvoker.Invoke` no processo emite WARNING.
- Chamadas subsequentes no mesmo processo **não emitem** warnings adicionais.
- `--runtime=acp --tool=codex` (que NÃO invoca `codexInvoker`) não emite warning.
- `CompatibilityTable["codex"]` aceita `gpt-5.5` sem flag.
- `--allow-unknown-model` permite modelos arbitrários para Codex.
- T-28/T-29/T-34/T-35 verdes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-28 (warning primeira invocação): `codexInvoker.Invoke(...)` em fresh process → stderr contém `"WARNING: Codex CLI legado"` + `"ADR-013"`.
- [ ] T-29 (segunda invocação): segunda chamada a `codexInvoker.Invoke(...)` no mesmo processo → stderr **NÃO** contém WARNING adicional (sync.Once).
- [ ] T-34 (compat aceito): `validateModelForTool("codex", "gpt-5.5", false)` → nil (aceito).
- [ ] T-35 (compat rejeitado): `validateModelForTool("codex", "gpt-4", false)` → erro; com `allowUnknown=true` → aceito.
- [ ] `go test ./internal/taskloop/...` 100% verde.
- [ ] `go vet ./internal/taskloop/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `var codexLegacyWarnOnce sync.Once` declarado em escopo package-level em `internal/taskloop/agent.go`.
- [ ] `codexInvoker.Invoke(...)` invoca `codexLegacyWarnOnce.Do(...)` no início; mensagem do PRD HU-06 com `ADR-013` referenciado.
- [ ] `CompatibilityTable["codex"]` contém pelo menos `"gpt-5.5"`.
- [ ] `validateModelForTool` aceita `("codex", "gpt-5.5", false)`; rejeita outros modelos sem `--allow-unknown-model`.
- [ ] T-28, T-29, T-34, T-35 verdes.
- [ ] **Sub-test importante**: confirmar que `--runtime=acp` NÃO invoca `codexInvoker` (warning não aparece em path ACP — válido se ACP path foi corretamente roteado em 7.0).
- [ ] `go test ./internal/taskloop/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.
- [ ] `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` → vazio.

## Arquivos Relevantes

- `internal/taskloop/agent.go` (modificar: linhas 335-351 — codexInvoker + sync.Once)
- `internal/taskloop/agent_test.go` (adicionar T-28, T-29)
- `internal/taskloop/compatibility.go` (modificar: CompatibilityTable)
- `internal/taskloop/compatibility_test.go` (adicionar T-34, T-35)
- ADR-013 §"Decisão" → D-05, D-10
- techspec.md §"Design de Implementação" → bloco `codexInvoker com sync.Once warning`
- PRD HU-06 (mensagem do warning), Q4 (janela de depreciação)
