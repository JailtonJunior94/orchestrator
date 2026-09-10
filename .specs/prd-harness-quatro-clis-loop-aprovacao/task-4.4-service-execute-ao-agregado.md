# Tarefa 4.4: `Service.Execute` conduz o `Cycle` (caminho de produção, critérios por task)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Segunda fatia do Bloco D. Migra o sub-passo de revisão de `Service.Execute`
(`internal/taskloop/taskloop.go:654-721`, invocado quando `opts.Profiles.Reviewer != nil` e
`outcome.RunReviewer`) para conduzir o `approval.Cycle` via o adaptador de `internal/taskloop`
criado em 4.3. Este é o caminho com critérios de aceite por task — o task file da task corrente já
está em escopo (`taskFile`, `taskloop.go:480`) — e **estabelece o padrão** de construção do `Cycle`
que `RunLoop` e `ACPRunner` seguem depois.

<requirements>
- RF-34: cada rodada de revisão e de correção executa em sessão/subagente novo, recebendo apenas os
  achados e o delta.
- `Service.Execute` conduz o `Cycle` via o adaptador de `internal/taskloop`; nenhuma lógica de
  decisão de veredito reimplementada. Critérios de aceite = `taskcriteria.Extract` do task file da
  task corrente.
- A virada do critério estrito **não** acontece aqui (é 5.0). Comportamento legado preservado, suíte
  existente verde — estágio de paridade (ADR-001, item 4). A evidência-sentinela de paridade de 4.3
  mantém o `CriteriaMap` completo.
- Ajuste de asserções de veredito **somente** onde codificam parsing leniente incompatível com o
  agregado fail-closed; cada ajuste justificado por requisito no relatório (seção "Conflitos de
  Regra"). Asserções baseadas em exit code do reviewer permanecem intocadas. T-REV-01/02/04 não mudam.
</requirements>

## Subtarefas

- [x] 4.4.1 Construir o `Cycle` no ponto de revisão de `Service.Execute` com identidade da task,
      identidade do agente, política a partir de `opts`, critérios do task file e as três portas do
      adaptador de 4.3.
- [x] 4.4.2 Traduzir `CycleResult` para `IterationResult.ReviewResult` / `BugfixResult` preservando
      os campos observáveis hoje (status, saída, notas).
- [x] 4.4.3 Ajustar cirurgicamente asserções incompatíveis em `taskloop_test.go` /
      `reviewer_test.go`, cada ajuste justificado por requisito.

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`** (ordem interna, item 4), subseção
**"Integração do agregado nos três caminhos (Bloco D)"** (D-B1, linha `Service.Execute`), e a tabela
**"Riscos Conhecidos"** ("O caminho de produção não é o que parecia").

## Critérios de Sucesso

- `grep -n 'approval\.' internal/taskloop/taskloop.go` retorna ocorrências.
- `go test ./internal/taskloop/... -count=1` verde; `git diff internal/taskloop/taskloop_test.go
  internal/taskloop/reviewer_test.go` mostra apenas ajustes justificados por requisito.
- Teste: `Service.Execute` conduz o `Cycle`; rodada de revisão e de correção em sessão nova (RF-34).
- T-REV-01/02/04 verdes e sem alteração de asserção.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/taskloop/... -count=1` verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória:

- `Service.Execute` conduz o `Cycle`; rodada em sessão nova (RF-34).
- Critérios de aceite vêm do task file da task corrente.
- Preservação: T-REV-01/02/04 verdes e não modificados; asserções por exit code intactas.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/taskloop/taskloop.go:244,480,654-721,757-811` — `Service.Execute`, sub-passo de revisão.
- `internal/taskloop/approval_adapters.go` — adaptador criado em 4.3.
- `internal/taskcriteria/` — extrator de critérios (4.3).
- `internal/taskloop/taskloop_test.go`, `internal/taskloop/reviewer_test.go` — asserções a ajustar.
- `internal/approval/` — agregado e portas.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Integração do agregado
  nos três caminhos (Bloco D)".
