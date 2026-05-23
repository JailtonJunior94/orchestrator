# Tarefa 1.0: Criar `specs/gemini.go` + `geminiBootstrapArgs` + testes unit

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar o construtor `specs.Gemini()` no catálogo de runtimes ACP do harness, espelhando estruturalmente o padrão `claude.go`/`codex.go`/`copilot.go`. A Spec aponta para o binário `gemini` com `FixedArgs:["--acp"]` e fallback `npx --yes @google/gemini-cli@0.43.0 --acp`. Adiciona função local `geminiBootstrapArgs` que mapeia `AccessMode` em flag `--approval-mode` conforme D-05 de ADR-015 (`Restricted → "default"`, `Full → "yolo"`; demais valores → `"default"`; `model`/`reasoning`/`addDirs` ignorados intencionalmente).

<requirements>
- Reusar `BootstrapArgsFunc` e `AccessMode` já introduzidos por ADR-013 (F1-Codex) — diff zero em `internal/runtime/specs/spec.go`.
- Constantes pinadas: `GeminiNpmPackage = "@google/gemini-cli"`, `GeminiNpmVersion = "0.43.0"`, `GeminiSDKVersion = "v0.13.0"`, `DefaultGeminiModel = "gemini-2.5-pro"`.
- `geminiBootstrapArgs` segue mapeamento literal D-05 sem emitir `auto_edit` nem `plan`.
- Comentário de cabeçalho do arquivo documenta divergência intencional do Compozy (D-05 em ADR-015).
- Suite de testes 100% verde antes de avançar para 2.0.
</requirements>

## Subtarefas

- [ ] 1.1 Criar `internal/runtime/specs/gemini.go` com pacote, imports, comentário de cabeçalho referenciando ADR-015 D-01..D-05.
- [ ] 1.2 Declarar constantes pinadas conforme techspec §"Interface Pública — `specs.Gemini()`".
- [ ] 1.3 Implementar `Gemini() Spec` usando `newSpecWithBootstrap(...)` existente em `specs/spec.go`.
- [ ] 1.4 Implementar `geminiBootstrapArgs(_, _ string, _ []string, mode AccessMode) []string` com switch literal D-05.
- [ ] 1.5 Criar `internal/runtime/specs/gemini_test.go` com testes T-14, T-15, T-16, T-29, T-30, T-31 conforme techspec §"Testes Unitários".
- [ ] 1.6 Validar regressão Claude/Codex/Copilot: `go test ./internal/runtime/specs/...` 100% verde.

## Detalhes de Implementação

Ver techspec.md:
- §"Interface Pública — `specs.Gemini()`" (linhas ~117-170) — código fonte exato do `Gemini()` e `geminiBootstrapArgs`.
- §"Mensagens de Erro e Warning Literais" — não aplicável a esta task.
- §"Considerações Técnicas / TD-01" — justificativa de reuso da `BootstrapArgsFunc` de F1-Codex.
- §"Considerações Técnicas / TD-03" — divergência Compozy em D-05 documentada em comentário do `geminiBootstrapArgs`.

ADR-015 D-01..D-05 (em `.specs/adr/015-gemini-cli-acp-native.md`) é a fonte normativa do mapeamento.

## Critérios de Sucesso

- `internal/runtime/specs/gemini.go` existe com `Gemini() Spec` e `geminiBootstrapArgs` exportados/locais conforme padrão.
- `internal/runtime/specs/gemini_test.go` existe e roda em < 1s.
- `go test ./internal/runtime/specs/... -v` retorna `PASS` para `TestGeminiSpecHasCorrectCommandAndFlags`, `TestGeminiFallbackResolvesViaNpx`, `TestGeminiBootstrapArgsForRestricted`, `TestGeminiBootstrapArgsForFull`, `TestGeminiBootstrapArgsIgnoresModelAndReasoning`, `TestGeminiBootstrapArgsDefaultsToRestricted`.
- `go test ./internal/runtime/specs/claude_test.go ./internal/runtime/specs/codex_test.go ./internal/runtime/specs/copilot_test.go ./internal/runtime/specs/spec_test.go` permanece 100% verde (regressão).
- `git diff --stat internal/runtime/specs/spec.go internal/runtime/specs/claude.go internal/runtime/specs/codex.go internal/runtime/specs/copilot.go` retorna **zero linhas** modificadas.
- `golangci-lint run ./internal/runtime/specs/...` sem warnings novos.

### Definition of Done

1. Suite de testes Gemini (6 testes T-14..T-31) verde.
2. Regressão de specs existentes (Claude/Codex/Copilot/spec.go) verde.
3. Diff zero em arquivos protegidos (RF-32 parcial — escopo: `internal/runtime/specs/spec.go` e siblings).
4. `geminiBootstrapArgs` documenta D-05 em comentário Go (godoc-style).
5. `golangci-lint` sem regressão.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-14 `TestGeminiSpecHasCorrectCommandAndFlags`
- [ ] T-15 `TestGeminiFallbackResolvesViaNpx`
- [ ] T-16 `TestGeminiBootstrapArgsForRestricted`
- [ ] T-29 `TestGeminiBootstrapArgsForFull`
- [ ] T-30 `TestGeminiBootstrapArgsIgnoresModelAndReasoning`
- [ ] T-31 `TestGeminiBootstrapArgsDefaultsToRestricted`
- [ ] Regressão: suite `internal/runtime/specs/{claude,codex,copilot,spec}_test.go` 100% verde

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- **NOVO**: `internal/runtime/specs/gemini.go`
- **NOVO**: `internal/runtime/specs/gemini_test.go`
- **REFERÊNCIA (não modificar)**: `internal/runtime/specs/spec.go`, `claude.go`, `codex.go`, `copilot.go`, `.specs/adr/015-gemini-cli-acp-native.md`
