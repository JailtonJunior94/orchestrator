# Tarefa 7.0: Paridade 4×4 + cross-project + invariante de fallback

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Transformar "paridade" em garantia verificável: estender o framework `internal/parity` (hoje testa
tools individuais e pares) para uma matriz **4×4 cross-CLI** table-driven, adicionar um teste
**cross-project** (instalar em repo temporário e validar invariantes) e um **invariante de fallback
launcher** (binário direto ausente → fallback → resultado idêntico). Conforme PRD RF-18/RF-19 e
[ADR-017](adr-017-fallback-launcher-chain.md). Reusa `parity.Invariants()`/`Snapshot`.

<requirements>
- Matriz 4×4: gerar combinações table-driven das 4 tools (e subconjuntos) validando invariantes em cada uma.
- Cross-project (RF-18): `internal/parity/e2e_parity_test.go` — instalar num repo temporário (`t.TempDir()`) e validar via `parity.Invariants()`.
- Invariante de fallback (RF-19): binário direto ausente → launcher fallback → resultado idêntico ao do binário direto.
- Reusar `parity.Invariant`/`Snapshot`/`Invariants()`; não duplicar a lógica de checagem.
- Sem regressão: invariantes existentes continuam passando.
</requirements>

## Subtarefas

- [ ] 7.1 Refatorar `parity_test.go` para gerar combinações 4×4 table-driven (em vez de funções fixas por tool/par).
- [ ] 7.2 Adicionar invariante de fallback launcher em `parity.go` (`Invariants()`), cobrindo o caminho da Tarefa 2.0.
- [ ] 7.3 Criar `internal/parity/e2e_parity_test.go` (build tag `integration`): instalar em `t.TempDir()` e validar invariantes (usa instalador da Tarefa 6.0).
- [ ] 7.4 Validar que o argv/resultado é idêntico via binário direto vs fallback para as 4 specs.
- [ ] 7.5 Garantir verde de toda a suíte de paridade (existentes + novos).

## Detalhes de Implementação

Ver `techspec.md` §"Testes E2E" e §"Componentes Modificados" (`internal/parity`). Depende de: Tarefa
2.0 (fallback genérico), Tarefa 4.0 (runtime estável com retry/concorrência) e Tarefa 6.0
(instalador portátil para o cross-project). Reusar `internal/parity/parity.go` (`Invariant`,
`Snapshot`, `Invariants`, `Generate`).

## Critérios de Sucesso

- Matriz 4×4 cobre as 4 CLIs e subconjuntos; todas as invariantes aplicáveis passam.
- Teste cross-project instala em repo temporário e valida invariantes com sucesso.
- Invariante de fallback: com binário direto ausente, o fallback produz resultado idêntico (RF-19).
- `make test`/`make integration`/`make lint` verdes; cobertura ≥ 75% no pacote `parity`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários: matriz 4×4 table-driven; invariante de fallback (LookPath fake).
- [ ] Testes de integração (`integration`, `t.TempDir()`): cross-project install + validação de invariantes.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/parity/parity.go` (`Invariants()`, invariante de fallback)
- `internal/parity/parity_test.go` (matriz 4×4)
- `internal/parity/e2e_parity_test.go` (novo, cross-project, build tag `integration`)
- `internal/install/*` (reuso para cross-project)
- `internal/runtime/probe/*`, `internal/runtime/specs/*` (fallback)
