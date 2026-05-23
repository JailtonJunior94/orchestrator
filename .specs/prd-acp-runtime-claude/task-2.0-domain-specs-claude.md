# Tarefa 2.0: Domain Specs (Claude) + Sync Script

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o catálogo de specs do novo pacote `internal/runtime/specs/`, contendo `Spec`, `FallbackLauncher`, `Launcher` (VO) e a única spec desta entrega: `Claude()`. Pinar as constantes `ClaudeSDKVersion` (alinhada com `go.mod`) e `ClaudeNpmVersion` (versão npm de `@agentclientprotocol/claude-agent-acp`). Entregar também `scripts/sync-acp-sdk-version.sh` para manter a constante Go sincronizada com `go.mod`.

<requirements>
- `Launcher` é VO imutável com `Kind()` retornando `"binary"` ou `"npx"`.
- `Spec` exposto apenas via construtor `Claude()` (sem struct literal pelo consumidor — R-DDD-001).
- `ClaudeNpmVersion` é uma constante Go (não env var, não config externa).
- `ClaudeSDKVersion` é uma constante Go atualizada por `scripts/sync-acp-sdk-version.sh` lendo `go.mod`.
- Sem importação de `coder/acp-go-sdk` (esta task antecede a adição da dependência).
- Snapshot test garante estabilidade da spec (campo a campo).
</requirements>

## Subtarefas

- [ ] 2.1 Criar `internal/runtime/specs/launcher.go`: VO `Launcher{ kind launcherKind; cmd string; args []string }`; type não-exportado `launcherKind` com constantes `launcherBinary`, `launcherNpx`; construtores `NewBinaryLauncher(cmd string, args ...string) Launcher`; `NewNpxLauncher(pkg, version string) Launcher`; getters `Kind() string`, `Command() (string, []string)`.
- [ ] 2.2 Criar `internal/runtime/specs/spec.go`: struct `Spec { ID, DisplayName, Command string; FixedArgs []string; Fallbacks []FallbackLauncher; AccessModeFlag string }`; struct `FallbackLauncher { Command string; FixedArgs []string }`. Construtor `NewSpec` interno ao pacote para impedir literal fora.
- [ ] 2.3 Criar `internal/runtime/specs/claude.go`: constantes `ClaudeNpmPackage = "@agentclientprotocol/claude-agent-acp"`; `ClaudeNpmVersion = "X.Y.Z"` (pinada conforme decisão do mantenedor no momento do merge — usar a última versão estável publicada); `ClaudeSDKVersion = "vX.Y.Z"` (sincronizada com `go.mod` por 2.5). Função pública `Claude() Spec` retornando spec com `Command: "claude-agent-acp"`, `Fallbacks` contendo `npx --yes <pkg>@<ver>`, `AccessModeFlag: "--bypass-permissions"`.
- [ ] 2.4 Criar `internal/runtime/specs/claude_test.go`: snapshot test que valida cada campo de `Claude()` e a forma exata do fallback npx; teste para `Launcher.Kind()` em ambos os modos.
- [ ] 2.5 Criar `scripts/sync-acp-sdk-version.sh`: lê `go.mod`, extrai a versão de `github.com/coder/acp-go-sdk`, atualiza a constante `ClaudeSDKVersion` em `internal/runtime/specs/claude.go` por substituição de texto idempotente. Saída em stderr quando atualizar; exit 0 sem alterações; exit 1 em erro de parse. Executável (`chmod +x`).
- [ ] 2.6 Atualizar `Makefile` adicionando target `sync-acp-sdk-version` que chama o script; rodar como pre-step de `make verify` apenas localmente (não em CI inicialmente).

## Detalhes de Implementação

Ver `techspec.md`:
- §"Modelagem de Domínio" → "Value Objects" (Launcher)
- §"Design de Implementação" → "Spec e Launcher (catálogo)"
- §"Pontos de Integração" → "claude-agent-acp (subprocesso)"
- ADR-009 §"Decisão" para o formato pinado

## Critérios de Sucesso

- `go test ./internal/runtime/specs/...` passa.
- `Claude()` retorna a mesma estrutura entre execuções (snapshot test verde).
- `scripts/sync-acp-sdk-version.sh` executado em repo sem mudança retorna exit 0 sem editar arquivo.
- `scripts/sync-acp-sdk-version.sh` em ambiente onde `go.mod` tem nova versão atualiza a constante e exit 0.
- Sem import de `coder/acp-go-sdk` no pacote (verificável por grep).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Snapshot test de `Claude()` (campo a campo)
- [ ] Tests table-driven para `Launcher.Kind()` e `Command()`
- [ ] Test do `sync-acp-sdk-version.sh`: cenários "sem mudança", "mudança aplicada", "go.mod sem o pacote" (erro)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/specs/launcher.go` + `launcher_test.go` (novo)
- `internal/runtime/specs/spec.go` (novo)
- `internal/runtime/specs/claude.go` + `claude_test.go` (novo)
- `scripts/sync-acp-sdk-version.sh` (novo, executável)
- `scripts/sync-acp-sdk-version_test.sh` (novo; smoke test bash, opcional)
- `Makefile` (modificado)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-2.0/execution_report.md`
- [ ] `go test ./internal/runtime/specs/... -count=1 -race -cover` cobre 100% das funções públicas
- [ ] `golangci-lint run ./internal/runtime/specs/...` sem violações
- [ ] `shellcheck scripts/sync-acp-sdk-version.sh` sem warnings
- [ ] Constante `ClaudeNpmVersion` documentada com comentário citando a decisão de pinning (PRD §"Restrições Técnicas" + ADR-009)
- [ ] Commit semântico `feat(runtime/specs): add Claude spec with launcher fallbacks and sync script`
