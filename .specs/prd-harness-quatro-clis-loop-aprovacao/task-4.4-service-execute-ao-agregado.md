# Tarefa 4.4: Migração de Service.Execute ao agregado (caminho real de produção)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Segunda das quatro fatias que promovem o loop ao agregado. Migra `Service.Execute`
(`internal/taskloop/taskloop.go:244`) — **o caminho real de produção** — para o mesmo adaptador de
`internal/taskloop` criado em 4.3. Concentra o grosso do ajuste cirúrgico de asserções de veredito
leniente em `taskloop_test.go` e `reviewer_test.go`.

<requirements>
- RF-34: cada rodada executa em sessão/subagente novo, recebendo apenas os achados e o delta.
- `Service.Execute` conduz o `Cycle` via o adaptador de `internal/taskloop`; nenhuma lógica de
  decisão de veredito reimplementada.
- A virada do critério estrito **não** acontece aqui (é 5.0). Comportamento legado preservado,
  suíte existente verde — estágio de paridade.
- Ajuste de asserções de veredito **somente** onde codificam parsing leniente incompatível com o
  agregado fail-closed; cada ajuste justificado por requisito no relatório (seção "Conflitos de
  Regra"). Asserções que já usam veredito canônico permanecem intocadas. T-REV-01/02/04 não mudam.
</requirements>

## Subtarefas

- [ ] 4.4.1 Migrar `Service.Execute` (`taskloop.go:244`) para conduzir o `Cycle` via o adaptador de
      4.3.
- [ ] 4.4.2 Ajustar cirurgicamente as asserções incompatíveis em `taskloop_test.go` e
      `reviewer_test.go`, cada ajuste justificado por requisito.
- [ ] 4.4.3 Verificar que `RunLoop` (4.3) e `Service.Execute` produzem veredito e motivo canônico
      idênticos para a mesma entrada (pré-prova de paridade; a prova formal é 4.6).

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`**, tabela **"Riscos Conhecidos"** ("O caminho de
produção não é o que parecia" → "os três caminhos migram para o agregado"), e
`4.3-decomposition-rationale.md`.

## Critérios de Sucesso

- `grep -n 'approval\.' internal/taskloop/taskloop.go` retorna ocorrências.
- `go test ./internal/taskloop/... -count=1` verde; `git diff internal/taskloop/taskloop_test.go
  internal/taskloop/reviewer_test.go` mostra apenas ajustes justificados por requisito.
- T-REV-01/02/04 verdes e sem alteração de asserção.
- Estilo R-STYLE-001 no código novo/tocado.
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

- `Service.Execute` conduz o `Cycle`; rodada em sessão nova (RF-34).
- Paridade `RunLoop` ↔ `Service.Execute` para a mesma entrada.
- Preservação: T-REV-01/02/04 verdes e não modificados.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/taskloop/taskloop.go:244` — `Service.Execute`.
- `internal/taskloop/approval_adapters.go` — adaptador criado em 4.3.
- `internal/taskloop/taskloop_test.go`, `internal/taskloop/reviewer_test.go` — asserções a ajustar.
- `internal/approval/` — agregado e portas.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Riscos Conhecidos".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/4.3-decomposition-rationale.md`.
