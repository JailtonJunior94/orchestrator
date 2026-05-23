# Tarefa 2.0: Criar specs/copilot.go com construtor e testes

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar `internal/runtime/specs/copilot.go` com construtor `Copilot() Spec` e constantes pinadas (`CopilotNpmPackage`, `CopilotNpmVersion`, `CopilotSDKVersion`, `CopilotMinCLIVersion`) seguindo o padrão de `claude.go:24-42`. Criar `internal/runtime/specs/copilot_test.go` cobrindo defaults da Spec, fallback npx e política de pinning.

Pré-requisito de merge: confirmar `CopilotNpmVersion` via `npm view @github/copilot versions` e `CopilotMinCLIVersion` via release notes upstream antes de fechar a tarefa como `done` (Q1/Q4 do PRD).

<requirements>
- Copilot() retorna Spec com ID="copilot", Command="copilot", FixedArgs=["--acp"].
- Spec carrega metadata via newSpec (Spec.SDKVersion/NPMVersion/NPMPackage corretos).
- Fallback único: npx --yes @github/copilot@<pin> --acp.
- AccessModeFlag vazio (D-07 do techspec; Copilot v0 sem flag análoga a --bypass-permissions).
- Constantes pinadas em SemVer-like (jamais @latest).
- CopilotNpmVersion e CopilotMinCLIVersion confirmados antes do merge.
</requirements>

## Subtarefas

- [ ] 2.1 Criar `internal/runtime/specs/copilot.go` com bloco `const` declarando `CopilotNpmPackage`, `CopilotNpmVersion`, `CopilotSDKVersion`, `CopilotMinCLIVersion` (documentação inline mirroring `claude.go:8-22`).
- [ ] 2.2 Implementar `Copilot() Spec` invocando `newSpec("copilot", "GitHub Copilot CLI (ACP)", "copilot", []string{"--acp"}, []FallbackLauncher{{...}}, "", CopilotSDKVersion, CopilotNpmVersion, CopilotNpmPackage)`.
- [ ] 2.3 Criar `internal/runtime/specs/copilot_test.go` com casos T-01 (ID/Command/FixedArgs), T-02 (metadata acessores), T-03 (fallback único, contém `--acp`), T-04 (constantes não-vazias e não `@latest`).
- [ ] 2.4 Confirmar `CopilotNpmVersion` consultando `npm view @github/copilot versions` (ou registry equivalente) — registrar versão confirmada no commit.
- [ ] 2.5 Confirmar `CopilotMinCLIVersion` via release notes do GitHub Copilot CLI (https://docs.github.com/en/copilot/reference/copilot-cli-reference/acp-server).

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/runtime/specs/copilot.go — novo arquivo`. Decisão D-06 (pinning) registra a política em ADR-012; D-07 justifica `AccessModeFlag=""`.

Anti-padrão: NÃO usar `@latest` nas constantes; NÃO instanciar `Spec{...}` por literal — apenas via `newSpec`.

## Critérios de Sucesso

- `internal/runtime/specs/copilot.go` compila.
- `Copilot()` retorna Spec idêntica em forma à canônica do compozy `registry_specs.go:222-242`, exceto `BootstrapArgs` (não suportado nesta versão do harness).
- Testes T-01..T-04 verdes.
- Constantes `CopilotNpmVersion` e `CopilotMinCLIVersion` confirmadas (não `X.Y.Z` placeholder).
- `internal/runtime/specs/claude_test.go` permanece verde (regressão protegida pela Tarefa 1.0).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-01: `spec.ID == "copilot"`, `spec.Command == "copilot"`, `spec.FixedArgs == ["--acp"]`.
- [ ] T-02: `spec.SDKVersion() == CopilotSDKVersion`, `spec.NPMVersion() == CopilotNpmVersion`, `spec.NPMPackage() == CopilotNpmPackage`.
- [ ] T-03: `len(spec.Fallbacks) == 1`, `spec.Fallbacks[0].Command == "npx"`, `spec.Fallbacks[0].FixedArgs` contém `"--acp"` ao final.
- [ ] T-04: `CopilotNpmVersion != "latest"`, `CopilotNpmVersion != ""`, `CopilotMinCLIVersion != ""`.
- [ ] `go vet ./internal/runtime/specs/...` sem warnings.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `internal/runtime/specs/copilot.go` criado com 4 constantes e função `Copilot()`.
- [ ] `internal/runtime/specs/copilot_test.go` criado com T-01..T-04 verdes.
- [ ] `CopilotNpmVersion` confirmada via `npm view @github/copilot versions` e registrada no commit message.
- [ ] `CopilotMinCLIVersion` confirmada via release notes upstream e registrada no comentário inline do constante.
- [ ] `AccessModeFlag` declarado como `""` (D-07) com comentário explicando.
- [ ] `go test ./internal/runtime/specs/...` → 100% verde.
- [ ] `go vet ./...` → sem warnings.
- [ ] Diff zero em `internal/runtime/persistence/`, `internal/runtime/watchdog.go`, `internal/runtime/client/`.

## Arquivos Relevantes

- `internal/runtime/specs/copilot.go` (criar)
- `internal/runtime/specs/copilot_test.go` (criar)
- `internal/runtime/specs/spec.go` (referência — newSpec signature estendida pela Tarefa 1.0)
- `internal/runtime/specs/claude.go` (template canônico)
- ADR-012 §"Decisão D-06" (pinning) e §"D-07" (AccessModeFlag vazio)
- Compozy `internal/core/agent/registry_specs.go:222-242` (Spec Copilot referência canônica)
