# Tarefa 3.0: Roteamento `taskloop.Service.Execute` + warning wrapper deprecation `sync.Once`

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Habilitar roteamento Gemini via `ACPRunner` em `internal/taskloop/taskloop.go::Service.Execute` quando `Runtime == "acp"`. Adicionar warning único por execução do processo (via `sync.Once`) em `internal/wrapper/wrapper.go::buildInstruction` quando wrapper legado (`gemini run --skill ...`) é invocado — mensagem referencia ADR-015 e sugere migração. Wrapper continua operacional (TD-05): warning é informativo, não bloqueante.

<requirements>
- Padrão idêntico ao roteamento Claude/Codex/Copilot: case `"gemini"` em `Service.Execute` propaga `Options.AccessMode` para `Job.AccessMode`; demais campos opcionais preservados.
- Warning único via `sync.Once` no escopo do `buildInstruction` (pacote `wrapper`). Aciona somente quando `tool == "gemini"` e wrapper é invocado.
- Mensagem exata conforme techspec §"Mensagens de Erro e Warning Literais" RF-08.
- Wrapper `internal/wrapper/wrapper.go::ValidTools["gemini"] = true` **permanece** — não remover.
- Coexistência wrapper ↔ ACP: ambos os caminhos funcionam após esta task (escolha via `--runtime`).
</requirements>

## Subtarefas

- [ ] 3.1 Adicionar case `"gemini"` em `internal/taskloop/taskloop.go::Service.Execute` no bloco de roteamento ACP, espelhando estrutura existente para `"codex"`/`"copilot"`.
- [ ] 3.2 Propagar `Options.AccessMode` para `Job.AccessMode` (já implementado para Codex; reusar).
- [ ] 3.3 Adicionar variável `geminiWrapperWarnOnce sync.Once` no escopo do pacote `wrapper`.
- [ ] 3.4 Em `buildInstruction("gemini", ...)`, chamar `geminiWrapperWarnOnce.Do(...)` que emite warning conforme RF-08.
- [ ] 3.5 Mensagem do warning (RF-08): `"WARNING: Gemini wrapper legado (gemini run --skill) em uso. Migrar para --runtime=acp (binário gemini com --acp). Ver ADR-015."`
- [ ] 3.6 Adicionar/estender testes: `TestServiceRoutesGeminiToACPRunner`, `TestWrapperEmitsGeminiDeprecationWarningOnce`.

## Detalhes de Implementação

Ver techspec.md:
- §"Arquitetura do Sistema / F1-Gemini" (linhas ~38-44) — lista exata de arquivos a editar.
- §"Mensagens de Erro e Warning Literais" — texto exato do warning.
- §"Considerações Técnicas / TD-05" — política de coexistência wrapper ↔ ACP.

Precedente direto: F1-Codex em `.specs/prd-codex-acp-spec/task-7.0-wiring-service-execute-codex.md` (mesmo padrão de roteamento) e `task-8.0-codex-compat-taskloop.md` (sync.Once warning idêntico).

## Critérios de Sucesso

- `internal/taskloop/taskloop.go::Service.Execute` aceita `Tool == "gemini" && Runtime == "acp"` e roteia via `ACPRunner`.
- `internal/wrapper/wrapper.go` contém `var geminiWrapperWarnOnce sync.Once` e o chama em `buildInstruction("gemini", ...)`.
- Warning é emitido **uma única vez** por execução do processo (testar via duas invocações consecutivas no mesmo processo de teste).
- `go test ./internal/taskloop/... -run TestServiceRoutesGeminiToACPRunner` retorna `PASS`.
- `go test ./internal/wrapper/... -run TestWrapperEmitsGeminiDeprecationWarningOnce` retorna `PASS`.
- Comando `ai-spec-harness task-loop --tool gemini .specs/prd-fake` (sem `--runtime`) emite warning único em stderr.
- Comando `ai-spec-harness task-loop --tool gemini --runtime acp --dry-run .specs/prd-fake` **não** emite warning de deprecation (modo ACP é o recomendado).

### Definition of Done

1. Roteamento ACP Gemini operacional via `Service.Execute`.
2. Warning único `sync.Once` emitido apenas no modo wrapper.
3. Wrapper preservado funcional (não bloqueante).
4. Mensagem do warning literal conforme RF-08.
5. Testes da task verdes; regressão Claude/Codex/Copilot verde.
6. Diff zero em `internal/runtime/runner.go`, `internal/runtime/client/client.go`, `internal/runtime/persistence/`, `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] `TestServiceRoutesGeminiToACPRunner` — Service.Execute roteia Gemini via ACPRunner quando Runtime=acp
- [ ] `TestServicePropagatesAccessModeForGemini` — Options.AccessMode propagado a Job.AccessMode
- [ ] `TestWrapperEmitsGeminiDeprecationWarningOnce` — warning emitido apenas 1x em N chamadas consecutivas no mesmo processo
- [ ] `TestWrapperGeminiLegacyStillFunctional` — buildInstruction("gemini", ...) ainda retorna instrução válida mesmo com warning

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **EDIÇÃO**: `internal/taskloop/taskloop.go` (case "gemini" em Service.Execute)
- **EDIÇÃO**: `internal/wrapper/wrapper.go` (sync.Once warning em buildInstruction)
- **EDIÇÃO** (testes): `internal/taskloop/taskloop_test.go`, `internal/wrapper/wrapper_test.go`
- **REFERÊNCIA**: `.specs/prd-codex-acp-spec/task-7.0-*.md`, `task-8.0-*.md`
