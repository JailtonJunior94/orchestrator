# Tarefa 2.0: Criar specs/codex.go com Codex() + codexBootstrapArgs + testes

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar `internal/runtime/specs/codex.go` declarando o construtor `Codex() Spec` para o runtime Codex via adapter `@zed-industries/codex-acp` (binário canônico `codex-acp`, distinto do CLI legacy `codex` da OpenAI).

Declarar constantes pinadas: `CodexNpmPackage = "@zed-industries/codex-acp"`, `CodexNpmVersion = "0.14.0"` (último stable em 2026-05-21), `CodexMinNpmVersion = "0.12.0"` (mínimo do compozy para `gpt-5.5`, informacional), `CodexSDKVersion = "v0.13.0"` (mesma do Claude/Copilot), `DefaultCodexModel = "gpt-5.5"`.

Implementar função local `codexBootstrapArgs(model, reasoning string, addDirs []string, mode AccessMode) []string` replicando fielmente `compozy/internal/core/agent/registry_specs.go:247-278`. Emite pares `-c key="value"`: model, model_reasoning_effort, features.code_mode=false, features.code_mode_only=false; em `AccessModeFull` adiciona approval_policy/sandbox_mode/web_search.

Criar `codex_test.go` com matriz: constructor defaults (T-01..T-04), bootstrap matrix model/reasoning/access (T-06..T-09), método delega (T-12).

<requirements>
- Codex() usa `newSpecWithBootstrap(...)` da tarefa 1.0 passando `codexBootstrapArgs`.
- Command = "codex-acp" (não "codex").
- FixedArgs = nil (toda config via BootstrapArgs).
- Fallback único `npx --yes @zed-industries/codex-acp@0.14.0`.
- AccessModeFlag = "" (Codex passa access via -c, não flag dedicada).
- codexBootstrapArgs replica fielmente o compozy: ordem e formato dos -c flags.
- strconv.Quote escapa values corretamente (segurança R-SEC-001).
- Constantes seguem política ADR-009 (pinadas, atualização via audit/).
- Diff zero em internal/runtime/persistence/, watchdog.go, client/.
</requirements>

## Subtarefas

- [ ] 2.1 Criar `internal/runtime/specs/codex.go` com header de package + imports (strconv).
- [ ] 2.2 Declarar bloco `const` com CodexNpmPackage, CodexNpmVersion, CodexMinNpmVersion, CodexSDKVersion, DefaultCodexModel + comentários de política (ADR-013 D-06).
- [ ] 2.3 Implementar função pública `Codex() Spec` invocando `newSpecWithBootstrap(...)`.
- [ ] 2.4 Implementar função local `codexBootstrapArgs(model, reasoning string, _ []string, mode AccessMode) []string` replicando compozy.
- [ ] 2.5 Implementar helper local `appendCodexOverrides(args []string, overrides ...string) []string` (emite pares `-c <override>`).
- [ ] 2.6 Criar `internal/runtime/specs/codex_test.go` com T-01 (defaults), T-02 (metadata), T-03 (fallback), T-04 (pinning não-latest).
- [ ] 2.7 Adicionar tests T-06 (model vazio + AccessModeRestricted), T-07 (model + reasoning + restricted), T-08 (full access), T-09 (reasoning low).
- [ ] 2.8 Adicionar test T-12 (`Spec.BootstrapArgs(...)` delega corretamente para `codexBootstrapArgs`).
- [ ] 2.9 Rodar `go test ./internal/runtime/specs/...` → 100% verde.
- [ ] 2.10 Confirmar comentário de header de `codex.go` documenta confusão `codex` vs `codex-acp` (D-01).

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/runtime/specs/codex.go — novo arquivo` (esboço completo com ~50 LoC) e §"Sequenciamento de Desenvolvimento" → item 2. Decisão registrada em ADR-013 D-01 (binário `codex-acp` vs `codex`), D-06 (pinning), D-07 (AccessModeFlag vazio).

Anti-padrão: NÃO usar `@latest` em `CodexNpmVersion`; NÃO usar literal `Spec{...}` (R-DDD-001).

## Critérios de Sucesso

- `internal/runtime/specs/codex.go` compila e satisfaz interface `Spec` via `newSpecWithBootstrap`.
- `Codex().ID == "codex"`, `Codex().Command == "codex-acp"`, `Codex().FixedArgs == nil`.
- `Codex().BootstrapArgs("gpt-5.5", "high", nil, AccessModeFull)` produz exatamente o argv esperado (incluindo sandbox/approval/web_search).
- `Codex().BootstrapArgs("gpt-5.5", "medium", nil, AccessModeRestricted)` **não** contém sandbox flags.
- `codex_test.go` cobre matriz completa T-01..T-04, T-06..T-09, T-12.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-01: `Codex().ID == "codex"`, `Codex().Command == "codex-acp"`, `Codex().FixedArgs == nil`.
- [ ] T-02: `Codex().SDKVersion() == CodexSDKVersion`, `NPMVersion() == CodexNpmVersion`, `NPMPackage() == CodexNpmPackage`.
- [ ] T-03: `len(Codex().Fallbacks) == 1`, `Fallbacks[0].Command == "npx"`, args contém `@zed-industries/codex-acp@0.14.0`.
- [ ] T-04: `CodexNpmVersion != "latest"` e não-vazia; `CodexMinNpmVersion <= CodexNpmVersion` (semver lex).
- [ ] T-06: `codexBootstrapArgs("", "", nil, AccessModeRestricted)` retorna `["-c", "features.code_mode=false", "-c", "features.code_mode_only=false"]`.
- [ ] T-07: `codexBootstrapArgs("gpt-5.5", "medium", nil, AccessModeRestricted)` inclui `-c model="gpt-5.5"` e `-c model_reasoning_effort="medium"` e **NÃO** inclui sandbox flags.
- [ ] T-08: `codexBootstrapArgs("gpt-5.5", "high", nil, AccessModeFull)` inclui todos os de T-07 + `-c approval_policy="never"`, `-c sandbox_mode="danger-full-access"`, `-c web_search="live"`.
- [ ] T-09: `codexBootstrapArgs("gpt-5.5", "low", nil, AccessModeRestricted)` inclui `-c model_reasoning_effort="low"`.
- [ ] T-12: `Codex().BootstrapArgs(...)` retorna mesmo resultado que chamada direta a `codexBootstrapArgs(...)`.
- [ ] `go vet ./internal/runtime/specs/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `internal/runtime/specs/codex.go` criado com constantes pinadas (5 valores) + função `Codex()` + função local `codexBootstrapArgs` + helper `appendCodexOverrides`.
- [ ] Comentário de header documenta distinção `codex` (CLI legacy) vs `codex-acp` (adapter Zed).
- [ ] `CodexNpmVersion = "0.14.0"` (validado via `npm view` em 2026-05-21).
- [ ] `Codex()` invoca `newSpecWithBootstrap(...)` (não literal `Spec{...}`).
- [ ] `codex_test.go` cobre T-01..T-04, T-06..T-09, T-12 com tabela de casos.
- [ ] `strconv.Quote` aplicado a values em `-c model="..."` e `-c model_reasoning_effort="..."` (proteção contra injeção).
- [ ] `go test ./internal/runtime/specs/...` → 100% verde incluindo testes Claude/Copilot/spec.
- [ ] `go vet ./...` → sem warnings.
- [ ] `git diff --stat internal/runtime/persistence/ internal/runtime/watchdog.go internal/runtime/client/` → vazio.

## Arquivos Relevantes

- `internal/runtime/specs/codex.go` (criar)
- `internal/runtime/specs/codex_test.go` (criar)
- `internal/runtime/specs/spec.go` (consumir tipos `AccessMode`, `BootstrapArgsFunc`, `newSpecWithBootstrap` da tarefa 1.0)
- ADR-013 §"Decisão" → D-01, D-06, D-07
- techspec.md §"Design de Implementação" → bloco `codex.go — novo arquivo`
- Referência cruzada: `compozy/internal/core/agent/registry_specs.go:106-122` (Codex Spec) e `:247-278` (`codexBootstrapArgs`)
