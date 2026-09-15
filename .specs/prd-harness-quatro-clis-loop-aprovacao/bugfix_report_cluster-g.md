# Relatorio de Bugfix

- Total de bugs no escopo: 1
- Corrigidos: 1
- Testes de regressao adicionados: 1
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-013
- Severidade: major
- Origem: RF-23 (tarefa 8.0) — checagem de runtime do plugin de governanca do OpenCode (`ErrOpenCodePluginRuntimeMissing`, `internal/install/install.go`, `installOpenCode`) introduzida corretamente por RF-23, mas incompativel com fixtures pre-existentes de ADR-024 (RI-01/RI-02/RI-04) em `internal/install/install_integration_test.go`.
- Estado: fixed
- Causa raiz: `newFakeLookPatherInt()` sem argumentos simula "nenhum binario disponivel no PATH" para QUALQUER nome consultado, incluindo `"bash"`. A intencao original documentada nos comentarios dos dois testes afetados era simular apenas ausencia de BINARIOS DE AGENTE ACP (claude/codex/copilot/opencode-cli), nao do interpretador `bash` usado pelo plugin de governanca do OpenCode. Com a nova checagem RF-23 (`installOpenCode` chama `LookPath("bash")` e retorna `ErrOpenCodePluginRuntimeMissing` se falhar), os testes `TestIntegration_7_5_ScenarioG_Monorepo` e `TestIntegration_7_5_ProbeNonFatal_BinaryAbsent` — que incluem `skills.ToolOpenCode` no conjunto de `Tools` — passaram a falhar mesmo em maquinas reais onde `bash` sempre existe, porque o fake nunca declarava `"bash"` como disponivel.
- Arquivos alterados:
  - `internal/install/install_integration_test.go` (linhas ~486 e ~557): `newFakeLookPatherInt()` -> `newFakeLookPatherInt("bash")` nos dois testes que incluem `skills.ToolOpenCode` no conjunto de `Tools`, com comentario atualizado explicitando a intencao (nenhum binario de agente ACP disponivel, mas `bash` presente para o plugin do OpenCode).
  - Confirmado via grep que os outros dois usos de `newFakeLookPatherInt()` no mesmo arquivo (`TestIntegration_7_5_ScenarioP_EmptyRepo`, linha ~315, e `TestIntegration_7_5_ScenarioM_GoRepo`, linha ~388) usam `Tools: []skills.Tool{skills.ToolClaude}` — sem `ToolOpenCode` — portanto nao acionam a checagem RF-23 e nao precisam de ajuste.
  - Nenhuma alteracao em codigo de producao: `installOpenCode`, `ErrOpenCodePluginRuntimeMissing` e `openCodeRequiredTools` permanecem inalterados, preservando o gate RF-23.
- Teste de regressao: os proprios testes `TestIntegration_7_5_ScenarioG_Monorepo` e `TestIntegration_7_5_ProbeNonFatal_BinaryAbsent` (build tag `integration`) reproduzem a `reproduction` do bug (fake LookPather + `ToolOpenCode` no escopo) e validam o `expected` (install conclui sem erro, com `bash` disponivel e nenhum binario de agente ACP disponivel).
- Validacao:
  - `go build ./...` -> sem erros
  - `go vet ./...` -> sem erros
  - `go test -tags=integration ./internal/install/... -run "TestIntegration_7_5" -v -count=1` -> 4 passed
  - `go test -tags=integration ./internal/install/... -count=1` -> 84 passed
  - `go test -tags=integration ./... -count=1` -> 3205 passed, 1 failed (pre-existente, fora de escopo), 4 skipped

## Comandos Executados

- `go build ./...` -> sem erros
- `go vet ./...` -> sem erros
- `go test -tags=integration ./internal/install/... -run "TestIntegration_7_5" -v -count=1` -> 4 passed
- `go test -tags=integration ./internal/install/... -count=1` -> 84 passed
- `go test -tags=integration ./... -count=1` -> 3205 passed, 1 failed, 4 skipped; falha isolada em `internal/parity` (`E2EParitySuite` indefinido, `internal/parity/e2e_parity_test.go`), regressao de build pre-existente introduzida no commit `900d8af`, anterior a este PRD e sem relacao com RF-23/BUG-013 — fora de escopo desta correcao.

## Riscos Residuais

- Nenhum risco residual identificado para BUG-013: a correcao e restrita aos testes, preserva o comportamento de producao do gate RF-23 e a intencao original das fixtures de ADR-024.
- Falha pre-existente e nao relacionada em `internal/parity` (build quebrado por `E2EParitySuite` indefinido) permanece fora de escopo e deve ser tratada como bug separado.
