# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Wiring do config hierárquico (`RuntimeConfig` mesclado) no `ACPRunner` antes de `Run()`
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Time ai-spec-harness (owner: JailtonJunior94)
- **Relacionados:** PRD (RIN-01); techspec; ADR-016 (config hierárquico); ADR-018 (RuntimeConfig); `internal/config/resolver.go` (`Resolver.Resolve`); `internal/config/runtime.go`; `internal/runtime/types.go` (`Job.RuntimeConfig`, `ApplyDefaults`); `internal/taskloop/runloop.go`

## Contexto

O resolver hierárquico existe e é testado (ADR-016, `DefaultResolver.Resolve`): aplica `defaults < global < projeto < overrides` campo a campo. O `RuntimeConfig` unificado existe (ADR-018) embutido em `Job`. **Mas as duas peças não estão conectadas:** o `ACPRunner` consome `Job.RuntimeConfig` já preenchido, e hoje esses valores chegam de defaults hardcoded em `specs/` + campos propagados ad-hoc pelo `taskloop`, não da cascata do `Resolver`. Resultado: a precedência `flags > workspace > global > defaults` não é aplicada uniformemente antes de `runner.Run()`, e as 4 CLIs podem divergir em timeout/retry/concurrent/batch conforme o caminho de origem.

Há também um *type gap*: `config.Runtime` usa `Timeout string` + `MaxRetries int` + `RetryBackoffMultiplier float64` + `Concurrent/BatchSize int`; `runtime.RuntimeConfig` usa `Timeout events.ActivityTimeout` + os mesmos numéricos. É preciso um mapeamento explícito (parse de duração) entre as duas representações.

## Decisão

1. Introduzir uma função de fronteira **`BuildRuntimeConfig(resolved config.Runtime, overrides ...) runtime.RuntimeConfig`** na camada de aplicação (`internal/taskloop`, consumidor), que:
   - Recebe o `config.Runtime` já resolvido por `config.Resolver.Resolve(cwd, flagsOverrides)`.
   - Faz o mapeamento de tipos: `time.ParseDuration(resolved.Timeout)` → `events.ActivityTimeout` (string vazia ⇒ zero ⇒ F1); numéricos 1:1.
   - Chama `RuntimeConfig.ApplyDefaults()` (normaliza `Concurrent`/`BatchSize` ≤0 → 1).
2. O `taskloop` resolve a config **uma vez**, antes de montar os `Job`, e injeta o mesmo `RuntimeConfig` para as 4 CLIs — garantindo precedência idêntica.
3. **Flags CLI** entram como `overrides` no `Resolver.Resolve` (camada de maior precedência), preservando `flags > workspace > global > defaults`.
4. Zero-value em qualquer camada preserva F1 (já garantido por `ApplyDefaults` + `mergeInto` que só sobrescreve não-zero).

O `ACPRunner.Run` não muda sua assinatura: continua recebendo `Job` com `RuntimeConfig` pronto. A decisão é **onde** montar (fronteira de aplicação, não no runner) e **com qual fonte** (resolver, não hardcode).

## Alternativas Consideradas

- **Resolver dentro do `ACPRunner`.** Vantagem: encapsula. Desvantagem: acopla o runner ao filesystem/HOME (config global), prejudica testabilidade e fere a fronteira (runner é orquestrador de sessão, não dono de config). Rejeitada.
- **Manter defaults hardcoded em `specs`.** Vantagem: nada a fazer. Desvantagem: precedência não-uniforme entre CLIs (o gap RIN-01). Rejeitada.
- **Novo tipo único substituindo `config.Runtime` e `runtime.RuntimeConfig`.** Vantagem: elimina mapeamento. Desvantagem: `config.Runtime` carrega também chaves não-runtime (TasksRoot, PRDPrefix, Coverage) — fundir aumentaria acoplamento. Rejeitada; mapeamento explícito na fronteira é mais coeso.

## Consequências

### Benefícios Esperados

- Precedência determinística e **idêntica nas 4 CLIs** aplicada antes de `Run()` (RIN-01).
- Runner permanece testável e desacoplado de filesystem/HOME.
- Mapeamento de tipos centralizado e testável (parse de duração num só lugar).

### Trade-offs e Custos

- Uma função de mapeamento a manter; risco de drift entre os dois tipos. Mitigado por teste de mapeamento e por `ApplyDefaults` como ponto único de normalização.

### Riscos e Mitigações

- **Risco:** `Timeout` malformado em config (ex.: "10x"). **Mitigação:** `BuildRuntimeConfig` retorna erro descritivo (`fmt.Errorf("timeout inválido: %w")`), fail-fast na fronteira (R-ERR-001: validar na entrada).
- **Rollback:** ausência de config + flags ⇒ `DefaultRuntime` ⇒ F1 exato.

## Plano de Implementação

1. `BuildRuntimeConfig` (mapeamento + parse + `ApplyDefaults`) com testes table-driven (incl. timeout vazio/ inválido).
2. `taskloop` resolve config uma vez e injeta nos `Job` das 4 CLIs.
3. Flags CLI → overrides no `Resolve`.
4. Teste de paridade: mesma config → mesmo `RuntimeConfig` para os 4 drivers.

## Monitoramento e Validação

- Gate: `make test`; teste de precedência (flags>workspace>global>defaults) e de regressão F1 (zero-value).
- Sucesso: alterar `concurrent` em `.claude/config.yaml` reflete igual nas 4 CLIs; flag CLI vence config.

## Impacto em Documentação e Operação

- Atualizar `docs/config-hierarchy.md` e AGENTS.md (Configuração) deixando claro que o runtime consome a cascata.

## Revisão Futura

- Revisitar se `config.Runtime` e `runtime.RuntimeConfig` convergirem naturalmente ou ao adicionar nova chave operacional.
