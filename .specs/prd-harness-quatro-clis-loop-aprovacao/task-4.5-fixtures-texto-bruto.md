# Tarefa 4.5: Adequação das fixtures de revisão ao contrato de texto bruto (B2)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Terceira fatia do Bloco D — fatia própria de teste, sem código de produção. O `Translator` do
agregado deriva o veredito de `output.RawText()` fail-closed
(`internal/approval/cycle.go:188`, `internal/approval/translator.go:13-20`): texto sem linha
`Verdict:` canônica resolve para `VerdictBlocked`. Os stubs vivos de `internal/taskloop` devolvem
`FinalReviewResult{Verdict: ...}` com `RawOutput` vazio — sob o agregado fechariam como `blocked` na
rodada 1. Esta fatia anexa `RawOutput` conforme o veredito já declarado por cada fixture, isolando a
mudança mecânica antes da migração de `RunLoop` (4.6).

<requirements>
- RF-34: fixtures passam a refletir o contrato "a tradução lê a saída real do revisor" (RF-46/RF-40).
- Nenhum arquivo de produção é alterado. Apenas `internal/taskloop/*_test.go`.
- Cada fixture com veredito declarado ganha `RawOutput` contendo uma linha `Verdict: <token>` que
  casa o campo `.Verdict` — nenhuma inversão de expectativa de teste.
- `internal/taskloop/bugfix_test.go` permanece intocado — `BugfixLoop` não passa a conduzir o
  `Cycle` (techspec, D-B3).
- `internal/runtime/runner_autoreview_test.go` (T-REV-01..04) permanece intocado — já migrado em 4.1.
</requirements>

## Subtarefas

- [ ] 4.5.1 Anexar `RawOutput` ao `stubReviewer` default (`internal/taskloop/runloop_test.go:67-88`)
      e aos 16 literais `FinalReviewResult{}` do arquivo.
- [ ] 4.5.2 Anexar `RawOutput` aos 3 literais de `internal/taskloop/integration_test.go`
      (`:439`, `:483`, `:538`).
- [ ] 4.5.3 Adicionar teste de tabela que afirma
      `approval.NewTranslator().Translate(fixture.RawOutput) == fixture.Verdict` para toda fixture
      com veredito — trava do contrato.

## Detalhes de Implementação

Seguir a techspec desta pasta, subseção **"Integração do agregado nos três caminhos (Bloco D)"**
(D-B2, com a tabela de escopo por arquivo) e **"Sobre a ordem interna de F2b"**.

## Critérios de Sucesso

- `git diff --stat` mostra apenas `internal/taskloop/runloop_test.go`,
  `internal/taskloop/integration_test.go` e o novo teste de contrato.
- `grep -n 'FinalReviewResult{' internal/taskloop/runloop_test.go internal/taskloop/integration_test.go`
  — toda ocorrência com `Verdict:` tem `RawOutput:` no mesmo literal (ou herda do `stubReviewer`).
- Teste de tabela do contrato `Translate(RawOutput) == Verdict` verde.
- `go test ./internal/taskloop/... -count=1` verde; nenhuma asserção existente invertida.
- `internal/taskloop/bugfix_test.go` e `internal/runtime/runner_autoreview_test.go` sem diff.
- Estilo R-STYLE-001 no código de teste novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/taskloop/... -count=1` verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória:

- Contrato: `Translate(fixture.RawOutput)` igual a `fixture.Verdict` para toda fixture.
- Regressão: suíte `internal/taskloop` inteira verde, sem inversão de expectativa.
- Preservação: `bugfix_test.go` e T-REV-01..04 sem diff.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/taskloop/runloop_test.go:67-88` — `stubReviewer` + 16 literais.
- `internal/taskloop/integration_test.go:439,483,538` — 3 literais.
- `internal/approval/translator.go:13-20` — contrato fail-closed do `Translate`.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — D-B2.
