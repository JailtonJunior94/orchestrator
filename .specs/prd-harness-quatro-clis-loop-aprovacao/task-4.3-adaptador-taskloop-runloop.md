# Tarefa 4.3: Adaptador de taskloop e migração de RunLoop ao agregado

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Primeira das quatro fatias que promovem o loop ao agregado (segunda decomposição da tarefa 4.0 —
ver `4.3-decomposition-rationale.md` e `## Riscos de Integração` em `tasks.md`).

Os adaptadores das portas entregues em 4.1 vivem em `internal/runtime`, mas `Service.Execute` e
`RunLoop` estão em `internal/taskloop` e hoje conduzem review por `FinalReviewer`/`BugfixInvoker`,
sem depender de `internal/runtime`. Esta fatia cria o adaptador de `internal/taskloop` sobre as
portas `approval.Reviewer`/`approval.Fixer`/`approval.Repository` (portas declaradas no consumidor,
coerente com a techspec) e migra o **primeiro** dos três caminhos — `RunLoop` — para conduzir o
`Cycle`, delegando a `BugfixLoop` apenas evidência e telemetria.

<requirements>
- RF-34: cada rodada de revisão e de correção executa em sessão/subagente novo, recebendo apenas os
  achados e o delta. Contexto acumulado através de rodadas é proibido.
- O adaptador de `internal/taskloop` implementa as três portas do agregado sem reimplementar a
  lógica de decisão — ela permanece em `internal/approval`.
- `RunLoop` (`internal/taskloop/runloop.go`) passa a conduzir o `Cycle`; `BugfixLoop`
  (`internal/taskloop/bugfix.go:83`) recebe apenas o que o agregado não absorve.
- A virada do critério estrito **não** acontece aqui (é 5.0). Comportamento legado preservado com a
  suíte existente verde — estágio de paridade (ADR-001, item 4).
- Ajuste de asserções de veredito **somente** em `internal/taskloop/runloop_test.go` e apenas onde
  codificam parsing leniente incompatível com o agregado fail-closed; cada ajuste justificado por
  requisito no relatório. T-REV-01/02/04 de `runner_autoreview_test.go` não mudam.
</requirements>

## Subtarefas

- [ ] 4.3.1 Criar `internal/taskloop/approval_adapters.go`: adaptadores sobre `FinalReviewer`
      (texto bruto, nunca `Verdict`), `BugfixInvoker` e o capturador de diff/ponto de corte, mais
      testes de contrato dos adaptadores.
- [ ] 4.3.2 Migrar `RunLoop` para conduzir o `Cycle`, delegando a `BugfixLoop` só evidência e
      telemetria.
- [ ] 4.3.3 Ajustar cirurgicamente as asserções incompatíveis em `runloop_test.go`.

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`** e a subseção **"Interfaces Chave"** (três
portas declaradas no consumidor; `Reviewer` devolve texto bruto). `4.3-decomposition-rationale.md`
detalha o raio de explosão e a fronteira desta fatia.

## Critérios de Sucesso

- `grep -n 'approval\.' internal/taskloop/runloop.go internal/taskloop/approval_adapters.go` retorna
  ocorrências.
- `internal/taskloop/approval_adapters.go` não contém lógica de decisão de veredito — só tradução de
  chamada.
- `go test ./internal/taskloop/... -count=1` verde; `git diff internal/taskloop/runloop_test.go`
  mostra apenas ajustes justificados por requisito.
- T-REV-01/02/04 verdes e sem alteração de asserção.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/taskloop/... -count=1` verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória:

- Adaptadores de `internal/taskloop`: cada porta traduz a chamada e nada mais.
- `RunLoop` conduz o `Cycle`; rodada em sessão nova (RF-34).
- Preservação: T-REV-01/02/04 verdes e não modificados.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/taskloop/runloop.go` — `RunLoop`, primeiro caminho a migrar.
- `internal/taskloop/bugfix.go:83` — `BugfixLoop.Run`.
- `internal/taskloop/reviewer.go`, `internal/taskloop/acpinvoker.go`,
  `internal/taskloop/acceptance.go` — fontes dos adaptadores.
- `internal/taskloop/runloop_test.go` — asserções a ajustar cirurgicamente.
- `internal/approval/` — agregado e portas.
- `internal/runtime/approval_adapters.go` — referência dos adaptadores do caminho runtime.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Interfaces Chave".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/4.3-decomposition-rationale.md`.
