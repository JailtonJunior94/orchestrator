# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Guard de governança em runtime — spec-hash/drift + PRD-first no hook `runtime.pre_open`
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Time ai-spec-harness (owner: JailtonJunior94)
- **Relacionados:** PRD (RG-01, RG-02, RG-03, RG-04); techspec; AGENTS.md (Invariantes de Governança 1 e 2); `internal/specdrift/specdrift.go` (`CheckHash`/`CheckDrift`); `internal/runtime/hooks/governance.go`; `internal/runtime/hooks/dispatcher.go`

## Contexto

**Confirmado por grep:** `internal/runtime/` não referencia `specdrift`, `spec-hash`, `CheckHash` nem `CheckDrift`. O `ACPRunner` valida apenas a presença de `AGENTS.md` (`GovernanceHook` no ponto `runtime.pre_open`), mas **não** valida que o PRD/techspec consumido bate com o hash registrado em `tasks.md`. A doutrina PRD-first (AGENTS.md, invariantes 1 e 2) é hoje enforce manual via comentários `<!-- spec-hash-prd -->` e via CLI (`ai-spec check-spec-drift`) — não há fail-fast em runtime, e nada impede o `ACPRunner` de executar uma task cujo PRD divergiu da spec.

Existe infra pronta e testada (`internal/specdrift`): `CheckHash(specContent, tasksContent, label)` e `CheckDrift(dir)`. A lacuna é puramente de wiring no runtime, idêntico para as 4 CLIs.

## Decisão

1. Criar **`SpecDriftHook`** em `internal/runtime/hooks`, registrado no ponto `PointRuntimePreOpen` (ao lado do `GovernanceHook`), que:
   - Recebe o diretório do PRD ativo (`Job.TasksDir`) via `RuntimePreOpenEvent`.
   - Chama `specdrift.CheckDrift(tasksDir)`; se `report.Pass == false` (hash divergente OU cobertura RF faltante), retorna erro tipado `ErrSpecDrift` que **aborta a sessão antes de `c.Open`** (fail-fast, abort-on-first-error já existente no dispatcher).
   - **PRD-first (RG-02):** quando `tasks.md` não tem hash registrado para o PRD (`NoHashFound`), o hook recusa execução com `ErrPRDUntracked` ("task sem PRD rastreável").
2. O guard é **idêntico para os 4 drivers** (registrado incondicionalmente no `prepareHooksDispatcher`, exceto `--disable-hooks`).
3. **Opt-out controlado:** `Job.SkipDriftGuard` (default `false`) permite desabilitar só este hook para fluxos legados/exploratórios, registrando a suposição. `--disable-hooks` continua desabilitando tudo (debug).
4. **RG-03:** a suíte de invariantes ADR-008 (`internal/parity`) vira gate de CI obrigatório por CLI — promovendo paridade de *documentada* a *provável*.
5. **RG-04:** telemetria opt-in (ADR-006) inalterada; o guard não emite dados sem consentimento.

`Job.TasksDir` já existe (usado pelo memory store) — reaproveitado, sem novo campo de transporte de PRD.

## Alternativas Consideradas

- **Manter enforce só na CLI (`check-spec-drift`).** Vantagem: zero mudança no runtime. Desvantagem: não cobre invocação direta do `ACPRunner` (ex.: nested agent, orquestração); o gap RG-01 permanece. Rejeitada.
- **Validar dentro de `Run()` fora do dispatcher.** Vantagem: simples. Desvantagem: quebra o padrão de hooks canônicos e dificulta opt-out/teste isolado. Rejeitada — o ponto `runtime.pre_open` existe exatamente para isso.
- **Warning não-fatal em vez de abort.** Rejeitada: drift de spec é exatamente a classe de erro que a Âncora de Confiança (AGENTS.md inv. 2) exige falhar cedo.

## Consequências

### Benefícios Esperados

- Fecha RG-01: nenhuma sessão ACP executa com PRD/techspec divergente, nas 4 CLIs.
- PRD-first deixa de ser convenção e vira invariante de runtime (RG-02).
- Reuso de `internal/specdrift` testado — sem reimplementar hash.

### Trade-offs e Custos

- Leitura extra de `prd.md`/`techspec.md`/`tasks.md` no `pre_open` (custo de IO marginal, uma vez por sessão).
- Pode bloquear fluxos legados que rodavam com spec dessincronizada — mitigado por `SkipDriftGuard` documentado.

### Riscos e Mitigações

- **Risco:** falso-positivo em repositórios sem `tasks.md` (ex.: uso interativo fora de PRD). **Mitigação:** quando `Job.TasksDir == ""`, o hook é no-op (sem PRD ativo, nada a validar) — preserva F1 e uso ad-hoc.
- **Rollback:** `Job.SkipDriftGuard=true` ou `--disable-hooks`.

## Plano de Implementação

1. Estender `RuntimePreOpenEvent` com `TasksDir` (ou ler de `Job` no registro do hook).
2. `SpecDriftHook` + `ErrSpecDrift`/`ErrPRDUntracked` + testes (drift, no-hash, no-op).
3. Registrar no `prepareHooksDispatcher` respeitando `SkipDriftGuard`/`DisableHooks`.
4. Promover suíte `internal/parity` a gate de CI (test.yml) por CLI.

## Monitoramento e Validação

- Gate: `make test` + CI `internal/parity` por driver.
- Sucesso: sessão com hash divergente aborta com `ErrSpecDrift` antes de `c.Open`; sessão sem `TasksDir` roda normal.
- Critério de revisão: mudança no formato de `spec-hash` ou no algoritmo (`specdrift`).

## Impacto em Documentação e Operação

- Atualizar AGENTS.md (invariante 2) e `docs/troubleshooting.md` com o novo erro de runtime e o opt-out.
- Runbook: como diagnosticar `ErrSpecDrift` (`ai-spec check-spec-drift`, `ai-spec sync-spec-hash`).

## Revisão Futura

- Revisitar se o custo de IO no `pre_open` se tornar relevante (cache por sessão) ou ao unificar com o validador de evidence.
