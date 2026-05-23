# Tarefa 2.0: Wiring — catalog + probe adrByID + cli-schema enum

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Registrar `"gemini"` no `runtimeACPCatalog` (`cmd/ai_spec_harness/task_loop.go:27-31`), na tabela `adrByID` (`internal/runtime/probe/probe.go`), e adicionar `gemini` ao enum de `--tool` em `docs/cli-schema.json` quando `--runtime=acp`. Após esta task, o gate em `task_loop.go` aceita `--tool gemini --runtime acp` mas o roteamento ainda não está completo (vem em 3.0).

<requirements>
- Diff cirúrgico: apenas as três edições listadas; nenhuma mudança comportamental adicional.
- `runtimeACPCatalog` recebe entrada `"gemini": specs.Gemini` na mesma posição alfabética relativa.
- `adrByID` em `probe.go` ganha `"gemini": ".specs/adr/015-gemini-cli-acp-native.md"`.
- `docs/cli-schema.json` enum de `--tool` inclui `gemini` (já listado em outros contextos; validar consistência).
- Cache de probe (`spec.ID`-keyed) continua funcionando para Gemini sem mudança.
</requirements>

## Subtarefas

- [ ] 2.1 Adicionar entrada `"gemini": specs.Gemini` em `cmd/ai_spec_harness/task_loop.go:27-31` (`runtimeACPCatalog`).
- [ ] 2.2 Adicionar entrada `"gemini": ".specs/adr/015-gemini-cli-acp-native.md"` em `internal/runtime/probe/probe.go` (`adrByID`).
- [ ] 2.3 Atualizar `docs/cli-schema.json` adicionando `gemini` em enum de `--tool` quando `--runtime=acp` (RF-25).
- [ ] 2.4 Estender testes existentes para cobrir Gemini: `TestRuntimeACPCatalogIncludesGemini` (T-13), extensão de `TestProbeReferencesADR`, extensão de `TestCLISchemaContainsAllTools`, extensão de `TestProbeCacheKey`.
- [ ] 2.5 Validar que gate em `task_loop.go` (mensagem de erro lista catálogo dinamicamente) agora aceita Gemini.

## Detalhes de Implementação

Ver techspec.md:
- §"Arquitetura do Sistema / F0-Gemini — Spec registration" (linhas ~33-37) — lista exata de arquivos a editar.
- §"Considerações Técnicas / Arquivos Relevantes e Dependentes" — confirma arquivos modificados (apenas 3).
- §"Mapeamento RF → Componente → Teste" — RF-05, RF-06, RF-25, RF-29.

Precedente direto: `.specs/prd-codex-acp-spec/task-5.0-registrar-codex-adrbyid-catalog.md` (cobre o mesmo padrão para Codex).

## Critérios de Sucesso

- `cmd/ai_spec_harness/task_loop.go:27-31` contém entrada Gemini no `runtimeACPCatalog`.
- `internal/runtime/probe/probe.go` contém entrada Gemini em `adrByID`.
- `docs/cli-schema.json` valida com `gemini` no enum de `--tool` (verificar via `jq` ou JSON schema validator).
- `go test ./cmd/ai_spec_harness/... -run TestRuntimeACPCatalogIncludesGemini` retorna `PASS`.
- `go test ./internal/runtime/probe/...` 100% verde.
- `ai-spec-harness task-loop --tool gemini --runtime acp --dry-run .specs/prd-fake` **não retorna erro de gate** (pode retornar erro de spec inválido se Spec não estiver registrada — ok). Validar que mensagem de erro não inclui "unsupported tool: gemini".

### Definition of Done

1. Três edições aplicadas (catalog, probe, cli-schema).
2. Testes T-13 e extensões verdes.
3. Comando `--tool gemini --runtime acp --dry-run` passa do gate.
4. Diff em arquivos não-listados (especialmente `taskloop.go`, `wrapper.go`) **zero**.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-13 estendido: `TestRuntimeACPCatalogIncludesGemini`
- [ ] Extensão de `TestProbeReferencesADR` cobrindo gemini → ADR-015
- [ ] Extensão de `TestCLISchemaContainsAllTools` cobrindo `gemini` no enum
- [ ] Extensão de `TestProbeCacheKey` validando cache key `gemini`

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **EDIÇÃO**: `cmd/ai_spec_harness/task_loop.go:27-31` (adicionar entry)
- **EDIÇÃO**: `internal/runtime/probe/probe.go` (adrByID)
- **EDIÇÃO**: `docs/cli-schema.json` (enum --tool quando --runtime=acp)
- **EDIÇÃO** (testes): `cmd/ai_spec_harness/task_loop_test.go`, `internal/runtime/probe/probe_test.go`
- **REFERÊNCIA**: `.specs/adr/015-gemini-cli-acp-native.md`
