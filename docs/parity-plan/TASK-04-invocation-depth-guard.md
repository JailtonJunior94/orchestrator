# TASK-04: Controle de profundidade de invocação

## Contexto

O script remoto `scripts/lib/check-invocation-depth.sh` protege contra encadeamento infinito
de skills (ex: execute-task → review → bugfix → ...). Funciona via variáveis de ambiente:

- `AI_INVOCATION_DEPTH` (default: 0) — nível atual
- `AI_INVOCATION_MAX` (default: 2) — limite máximo

Quando `depth >= max`, o script aborta com erro. Caso contrário, incrementa o depth e exporta.

No CLI Go, não existe mecanismo equivalente.

## Objetivo

Implementar guard de profundidade de invocação no CLI Go.

## Análise de design

O controle de profundidade no bash é relevante porque os hooks e skills podem chamar uns aos outros
em cascata. No CLI Go, o cenário de encadeamento é diferente:

- **Se o CLI Go é usado como ferramenta standalone** (install, upgrade, lint), o encadeamento
  não é um risco direto.
- **Se o CLI Go é invocado por hooks de agentes de IA** (ex: validate-governance.sh chama
  `ai-spec lint`), existe risco de loop.

A implementação deve cobrir o segundo caso: ler env vars, validar limite, incrementar e
propagar para subprocessos.

## Subtarefas

- [x] **4.1** Criar pacote `internal/invocation/`
  - Função `CheckDepth() error` — lê `AI_INVOCATION_DEPTH` e `AI_INVOCATION_MAX` do env
  - Retorna erro se `depth >= max`
  - Função `IncrementDepth()` — seta `AI_INVOCATION_DEPTH = current + 1` no env do processo

- [x] **4.2** Integrar no PersistentPreRun do root command
  - No `rootCmd.PersistentPreRunE`, chamar `invocation.CheckDepth()`
  - Se falhar, retornar erro com mensagem clara: "limite de profundidade de invocação atingido (depth=%d, max=%d)"
  - Chamar `invocation.IncrementDepth()` se passar

- [x] **4.3** Testes unitários
  - Testar `CheckDepth` com depth=0, max=2 → ok
  - Testar `CheckDepth` com depth=2, max=2 → erro
  - Testar `CheckDepth` sem env vars → ok (defaults)
  - Testar `IncrementDepth` incrementa corretamente
  - Testar com `AI_INVOCATION_MAX=0` → sempre bloqueia

- [x] **4.4** Teste de integração
  - Executar `AI_INVOCATION_DEPTH=5 AI_INVOCATION_MAX=3 ai-spec version` e verificar que retorna erro

## Arquivos afetados

- `internal/invocation/invocation.go` (novo)
- `internal/invocation/invocation_test.go` (novo)
- `cmd/ai_spec_harness/root.go` (PersistentPreRunE)

## Critério de conclusão

1. `AI_INVOCATION_DEPTH=2 AI_INVOCATION_MAX=2 ai-spec version` retorna exit code 1
2. `AI_INVOCATION_DEPTH=0 ai-spec version` funciona normalmente
3. `go test ./internal/invocation/...` passa
