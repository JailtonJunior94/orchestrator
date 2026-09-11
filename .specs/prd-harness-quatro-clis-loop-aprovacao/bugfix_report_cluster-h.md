# Relatorio de Bugfix

- Total de bugs no escopo: 2
- Corrigidos: 2
- Testes de regressao adicionados: 2
- Pendentes: nenhum
- Estado final: done

## Bugs

- ID: BUG-014
- Severidade: major
- Origem: RF-31, RF-34 (tarefa 4.8, "Prova de paridade entre os três caminhos e fluxos E2E") — review adversarial encontrou que o critério de sucesso "mesmo cenário de entrada produz veredito, motivo canônico e estrutura de evidência idênticos" não era satisfeito: apenas `approved`/`reviewRounds`/`fixInvocations` eram comparados entre os três caminhos, sem o motivo canônico real do `approval.Cycle`.
- Estado: fixed
- Causa raiz: `ACPRunner` já expunha `summary.CycleStopReason` (populado a partir de `approval.CycleResult.Reason()` em `internal/runtime/runner.go`), mas `Service.Execute` (via `conductApprovalCycle`, `internal/taskloop/approval_cycle.go`) só usava `result.Reason()` para compor uma string livre em `ReviewResult.Note`, e `RunLoop` (`internal/taskloop/runloop.go`) obtinha `result approval.CycleResult` em `s.runRejectedCycle` mas nunca persistia o motivo canônico em `LoopReport`. `parityOutcome` (`internal/taskloop/parity_test.go`) não tinha nenhum campo para o motivo, então a comparação de paridade nunca provava a igualdade do motivo canônico — apenas de proxies indiretos (heurística de substring em `report.md` para `Service.Execute`; `Escalated`/`FinalReview.Verdict` para `RunLoop`).
- Arquivos alterados:
  - `internal/taskloop/report.go`: `ReviewResult` ganhou o campo `CycleStopReason string` (zero-value quando não há Cycle); `renderAvancado` passou a emitir a linha `- **Cycle Stop Reason:** <valor>` na subseção "Review Result" quando o campo não está vazio — superfície estruturada exposta ao chamador de teste via `report.md` (mesmo arquivo que o teste de paridade já lia para `Service.Execute`).
  - `internal/taskloop/approval_cycle.go`: em `conductApprovalCycle`, após `cycle.Run` retornar sem erro, `review.CycleStopReason = result.Reason().String()` é atribuído antes do `switch result.Approved()` — mesma convenção de string (`"approved"`, `"max_rounds"`, `"no_convergence"`, `"empty_diff"`, `"blocked_input"`) usada por `Summary.CycleStopReason` do ACPRunner. Permanece vazio quando `cycle.Run` retorna erro (nenhum resultado válido) ou quando o caminho legado (sem Cycle) é usado.
  - `internal/taskloop/runloop.go`: `LoopReport` ganhou o campo `CycleStopReason string` (`json:"cycle_stop_reason,omitempty"`); no branch onde `result, recorder, cycleErr = s.runRejectedCycle(...)` é bem-sucedido, `report.CycleStopReason = result.Reason().String()` é atribuído junto com `report.FinalReview`. Só populado quando o Cycle é de fato conduzido (`len(criteria) > 0`); permanece zero-value no caminho legado do `BugfixLoop` (RF-62 preservado).
  - `internal/taskloop/parity_test.go`: `parityOutcome` ganhou `stopReason string`; `driveServiceExecuteParity` extrai o valor via novo helper `extractCycleStopReason` (parse estrutural da linha "Cycle Stop Reason:" do `report.md`, sem heurística de substring em "blocked"/"escalated"); `driveRunLoopParity` lê `report.CycleStopReason` diretamente do `LoopReport`; `driveACPRunnerParity` lê `summary.CycleStopReason` (já existia). A comparação final `execOutcome != runLoopOutcome || runLoopOutcome != acpOutcome` (struct comparável) agora inclui `stopReason` automaticamente, e uma asserção dedicada `if outcome.stopReason != "approved"` foi adicionada ao loop de verificação por caminho.
- Teste de regressao: `TestApprovalCycleParityAcrossThreeProductionPaths` (mesmo teste, escopo estendido) passa a comparar o motivo canônico estruturado entre os três caminhos, satisfazendo literalmente o critério de aceite da tarefa 4.8 ("veredito, motivo canônico e estrutura de evidência idênticos").
- Validacao:
  - `go build ./...` -> sem erros
  - `go vet ./...` -> sem erros
  - `go test ./internal/taskloop/... -run TestApprovalCycleParityAcrossThreeProductionPaths -count=1 -v` -> 1 passed
  - `go test ./internal/taskloop/... ./internal/runtime/... ./internal/approval/... -count=1 -v` -> 1582 passed
  - `go test -tags=integration ./internal/taskloop/... ./internal/runtime/... -count=1` -> 1505 passed
  - `go test ./... -count=1` -> 2906 passed

- ID: BUG-015
- Severidade: minor
- Origem: RF-45 (tarefa 4.6, achado de review "low") — o mapeamento homogêneo de `CycleResult.Reason() ∈ {ReasonMaxRounds, ReasonNoConvergence, ReasonEmptyDiff, ReasonBlockedInput}` para `ErrBugfixExhausted`/`Escalated=true` no `RunLoop` (`internal/taskloop/runloop.go`, bloco `if !result.Approved() { report.Escalated = true; exhausted = true }`) tinha testes dedicados apenas para os três primeiros motivos (`TestRunLoopRejectedEscalated` para `ReasonMaxRounds`, `TestRunLoopRejectedNoConvergence` para `ReasonNoConvergence`, e cobertura de `ReasonEmptyDiff` no `ACPRunner`), mas nenhum teste forçava explicitamente `ReasonBlockedInput` no caminho `RunLoop`.
- Estado: fixed
- Causa raiz: ausência de teste, não de comportamento incorreto — `internal/approval/policy.go` (`ApprovalPolicy.Decide`) já retorna `ReasonBlockedInput` fail-closed sempre que `Translator.Translate` (`internal/approval/translator.go`) não encontra uma linha com prefixo canônico de veredito (`verdict:`/`veredicto:`/`veredito:`) no output bruto do reviewer, produzindo `VerdictBlocked`; o `RunLoop` já tratava esse caso pelo mesmo branch genérico de "não aprovado", mas sem prova de teste isolada.
- Arquivos alterados:
  - `internal/taskloop/runloop_test.go`: novo teste `TestRunLoopRejectedBlockedInputEscalated`, inserido logo após `TestRunLoopRejectedNoConvergence`. Usa `stubReviewer` com `RawOutput` sem nenhuma linha de veredito canônico ("revisao sem veredito canonico declarado\n"), forçando `Translator.Translate` a retornar `VerdictBlocked` já na primeira rodada, o que aciona `ApprovalPolicy.Decide` a interromper o ciclo com `ReasonBlockedInput` antes de qualquer chamada ao `Fixer` (bloqueio ocorre no branch `if stop { return c.closeBlocked(reason) }` de `internal/approval/cycle.go`, anterior à extração de `findings` e à invocação de correção).
- Teste de regressao: `TestRunLoopRejectedBlockedInputEscalated` confirma `Escalated=true`, `errors.Is(err, ErrBugfixExhausted)`, `deps.BugfixInvoker.(*runloopBugfixInvoker).calls == 0` (nenhuma rodada de correção é gasta — fail-closed conservador) e `report.CycleStopReason == "blocked_input"` (campo introduzido pela correção de BUG-014, reaproveitado aqui como prova estruturada).
- Validacao:
  - `go build ./...` -> sem erros
  - `go vet ./...` -> sem erros
  - `go test ./internal/taskloop/... -run TestRunLoopRejectedBlockedInputEscalated -count=1 -v` -> 1 passed
  - `go test ./internal/taskloop/... ./internal/runtime/... ./internal/approval/... -count=1 -v` -> 1582 passed
  - `go test -tags=integration ./internal/taskloop/... ./internal/runtime/... -count=1` -> 1505 passed
  - `go test ./... -count=1` -> 2906 passed

## Comandos Executados

- `go build ./...` -> sem erros
- `go vet ./...` -> sem erros
- `go test ./internal/taskloop/... ./internal/runtime/... ./internal/approval/... -count=1 -v` -> 1582 passed
- `go test -tags=integration ./internal/taskloop/... ./internal/runtime/... -count=1` -> 1505 passed
- `go test ./... -count=1` -> 2906 passed
- `go test -tags=integration ./internal/parity/... -count=1` -> build failed (`E2EParitySuite` indefinido em `internal/parity/e2e_parity_test.go`), regressão de build pré-existente introduzida no commit `900d8af`, muito anterior a este PRD e sem relação com BUG-014/BUG-015 — confirmado fora de escopo desta correção.

## Riscos Residuais

- Nenhum risco residual identificado para BUG-014/BUG-015: as mudanças são aditivas (novos campos com zero-value preservado nos caminhos legados sem Cycle), nenhuma asserção pré-existente foi alterada, e nenhum veredito, contagem de rodadas ou estrutura de evidência de produção mudou de comportamento observável.
- A exposição de `CycleStopReason` em `Service.Execute` depende do parse textual de uma linha específica do `report.md` renderizado (`extractCycleStopReason`), já que `Service.Execute` não retorna a struct `Report` ao chamador — é a mesma superfície (arquivo) que o teste de paridade já usava para inferir `approved` por heurística; se o formato de `renderAvancado` mudar no futuro sem atualizar o parser do teste, a paridade deixaria de comparar o motivo canônico silenciosamente. Risco considerado aceitável neste escopo (aditivo, não upgrade arquitetural de `Service.Execute` para retornar `*Report`).
- A regressão de build pré-existente em `internal/parity` (`-tags=integration`) permanece não corrigida — fora de escopo desta tarefa, já registrada como "fora de escopo" desde o cluster-g.
