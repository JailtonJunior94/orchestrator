# Tarefa 2.0: Fallback launcher chain genérico (remove npx-only)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Generalizar a resolução de fallback em `internal/runtime/probe/probe.go` para tratar cada
`specs.FallbackLauncher` como launcher de comando genérico, preservando seus `FixedArgs`
literalmente, em vez do tratamento npx-only atual (`extractPackage`/`extractVersion`/`NewNpxLauncher`).
Toda a cadeia `spec.Fallbacks` é tentada em ordem; o primeiro `Command` presente no PATH vence.
Conforme [ADR-017](adr-017-fallback-launcher-chain.md).

<requirements>
- `probe.resolve` materializa fallback via `specs.NewBinaryLauncher(path, fb.FixedArgs...)` (sem semântica npx).
- Cadeia `spec.Fallbacks` tratada em ordem; canônico (`spec.Command`) sempre primeiro.
- Remover/depreciar `extractPackage`/`extractVersion`; npx vira caso particular de `FallbackLauncher`.
- `sdkVersion/npmVersion/npmPackage` permanecem apenas como metadado da mensagem de erro de indisponibilidade.
- Paridade byte-equivalente (RF-05): argv resolvido idêntico ao baseline para as 4 specs atuais.
- Cache por `spec.ID` em `probe` preservado.
</requirements>

## Subtarefas

- [ ] 2.1 Reescrever `probe.resolve` para iterar `spec.Fallbacks` com `NewBinaryLauncher`.
- [ ] 2.2 Converter os fallbacks npx das specs (claude/codex/gemini/copilot) para o formato genérico mantendo `FixedArgs` atuais.
- [ ] 2.3 Remover/depreciar `extractPackage`/`extractVersion` (e `NewNpxLauncher` se órfão).
- [ ] 2.4 Atualizar o comentário de cabeçalho do pacote `probe` (não mais claude-agent-acp/npx específico).
- [ ] 2.5 Atualizar testes de `probe` + adicionar teste de paridade de argv por spec.

## Detalhes de Implementação

Ver `techspec.md` §"Componentes Modificados" e [ADR-017](adr-017-fallback-launcher-chain.md).
Tocar `internal/runtime/probe/probe.go` (`resolve`) e `internal/runtime/specs/*.go` (forma dos
fallbacks). Manter `formatLauncherUnavailable` e o mapeamento `adrByID`.

## Critérios de Sucesso

- Binário canônico ausente + fallback presente → launcher == fallback com `FixedArgs` exatos.
- Múltiplos fallbacks: primeiro presente no PATH vence, na ordem declarada.
- Argv resolvido idêntico ao baseline para claude/codex/gemini/copilot (RF-05).
- `make test`/`make lint` verdes; cobertura ≥ 75% no pacote `probe`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: fallback genérico, cadeia múltipla, canônico-primeiro, paridade de argv por spec (LookPath fake).
- [ ] Testes de integração: não obrigatórios (LookPath mockável em unit).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/probe/probe.go`
- `internal/runtime/probe/probe_test.go`
- `internal/runtime/specs/spec.go`, `claude.go`, `codex.go`, `gemini.go`, `copilot.go`
- `internal/runtime/specs/launcher*.go` (`NewBinaryLauncher`, `NewNpxLauncher`)
