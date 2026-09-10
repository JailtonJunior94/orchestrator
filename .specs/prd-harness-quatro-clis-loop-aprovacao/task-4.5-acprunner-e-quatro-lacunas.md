# Tarefa 4.5: Migração de ACPRunner e fiação das quatro lacunas nos três caminhos

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Terceira das quatro fatias que promovem o loop ao agregado. Migra o terceiro e último caminho —
`ACPRunner` (`internal/runtime/runner.go:216-228`), que hoje dispara `runAutoReview` uma única vez —
para conduzir o `Cycle`. Com os três caminhos já consumindo o agregado, **fia as quatro lacunas do
loop** (RF-31) em `Summary`, `LoopReport` e telemetria, de modo que cada caminho as observe de forma
consistente.

As quatro lacunas:

1. **Ressalvas realimentam o ciclo** — `APPROVED_WITH_REMARKS` gera achados que voltam à correção.
2. **Detecção de não-convergência** — fingerprint repetida em rodadas consecutivas aborta.
3. **Aborto por ausência de mudança** — correção que não produz diff aborta.
4. **Ponto de corte exportado por rodada** — já entregue por 4.2; aqui é observável nos três
   caminhos.

A regra de decisão dessas lacunas já vive em `internal/approval/cycle.go` (tarefa 2.0); esta fatia é
só a **fiação** por caminho.

<requirements>
- RF-31: o loop existente é promovido ao agregado, não substituído; as quatro lacunas fecham e são
  observáveis nos três caminhos.
- `ACPRunner` conduz o `Cycle`; `runAutoReviewRound` permanece como implementação da porta
  `Reviewer` (não é removida) — T-REV-01/02/04 continuam verdes e sem alteração de asserção.
- A virada do critério estrito **não** acontece aqui (é 5.0).
- Estados terminais do Ciclo (não convergir, esgotar rodadas, não produzir mudança) são
  **resultados**, não erro de infraestrutura, e não acionam o caminho de retry (RF-45, modelado em
  2.0; respeitado na fiação).
</requirements>

## Subtarefas

- [ ] 4.5.1 Migrar `runner.go:216-228` para conduzir o `Cycle` via os adaptadores de
      `internal/runtime`.
- [ ] 4.5.2 Fiar as quatro lacunas em `Summary`/`LoopReport`/telemetria nos três caminhos, de forma
      que a saída de cada rodada seja distinguível (número, veredito, contagem por severidade, motivo
      de parada) sem abrir os artefatos de evidência.
- [ ] 4.5.3 Garantir que estados terminais do Ciclo não acionem retry.

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`** (quatro lacunas), tabela **"Componentes
modificados"** (`internal/taskloop/bugfix.go`, `internal/runtime/runner.go`), e a ADR-001 "Plano de
Implementação" itens 4–7. `4.3-decomposition-rationale.md` delimita a fronteira.

## Critérios de Sucesso

- `grep -n 'approval\.' internal/runtime/runner.go` retorna ocorrências.
- `go test ./internal/runtime/ -run TestAutoReview -v -count=1` verde;
  `git diff --stat internal/runtime/runner_autoreview_test.go` sem alteração de asserção em
  T-REV-01/02/04.
- Teste: `APPROVED_WITH_REMARKS` realimenta a correção.
- Teste: fingerprint repetida aborta **sem gastar a rodada seguinte**.
- Teste: correção sem diff aborta.
- Teste: estado terminal do Ciclo não aciona retry.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync check-hooks-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/runtime/... ./internal/taskloop/... -count=1` verde.
- Golden files de governança byte-idênticos.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória:

- `ACPRunner` conduz o `Cycle`; T-REV-01/02/04 intocados.
- Quatro lacunas: ressalvas realimentam; fingerprint repetida aborta; ausência de mudança aborta;
  ponto de corte observável por rodada.
- Estado terminal do Ciclo não aciona retry (RF-45).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/runner.go:216-228` — ponto de chamada de `runAutoReview`.
- `internal/runtime/runner_autoreview.go` — `runAutoReviewRound` como porta `Reviewer`.
- `internal/runtime/summary.go`, `internal/runtime/types.go` — campos de resultado do Ciclo.
- `internal/taskloop/bugfix.go`, `internal/taskloop/runloop.go`, `internal/taskloop/taskloop.go` —
  fiação das lacunas nos três caminhos.
- `internal/approval/cycle.go` — regra de decisão das quatro lacunas (tarefa 2.0).
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Componentes
  modificados".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md`.
