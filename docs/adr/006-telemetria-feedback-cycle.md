# ADR-006: Telemetria opt-in com feedback cycle

**Status:** Aceita  
**Data:** 2026-04-20  
**Autores:** -

---

## Contexto

Necessidade de medir uso de skills e referencias para otimizar token budget. Sem dados de uso, decisoes de otimizacao sao baseadas em estimativas sem evidencia. O ciclo de feedback depende de metricas acionaveis para identificar quais referencias sao carregadas com frequencia, quais sao ignoradas e onde o budget de tokens esta sendo desperdicado.

## Alternativas Consideradas

| Alternativa | Vantagens | Desvantagens |
|-------------|-----------|--------------|
| Telemetria always-on | Dados completos de uso, sem lacunas | Impacto em privacidade, overhead em toda invocacao, resistencia do usuario |
| Sampling (amostragem) | Menor overhead, coleta parcial representativa | Dados incompletos, complexidade de configuracao de taxa |
| Nenhuma telemetria | Zero overhead, zero complexidade | Sem visibilidade de uso, otimizacoes baseadas em suposicoes |

## Decisao

Decidimos implementar telemetria opt-in via variavel de ambiente `GOVERNANCE_TELEMETRY=1`, com append-only log em `.agents/telemetry.log` e geracao de relatorio via CLI (`ai-spec-harness telemetry report`). O usuario escolhe ativar explicitamente; quando ativa, cada carregamento de skill/referencia registra um evento no log. O relatorio agrega os dados para decisoes de otimizacao.

## Consequencias

### Positivas
- Sem impacto em privacidade (opt-in explicito)
- Overhead minimo (<1ms por append em arquivo local)
- Ciclo fechado com metricas acionaveis para otimizar token budget
- Formato append-only simplifica implementacao e evita corrupcao

### Negativas / Riscos
- Dados incompletos quando telemetria nao esta ativada
- Dependencia de disciplina do usuario para manter telemetria ligada
- Log pode crescer indefinidamente sem rotacao automatica

### Neutras / Observacoes
- Relatorio pode ser gerado a qualquer momento sem impacto no fluxo normal
- Formato do log e interno e pode evoluir sem quebrar compatibilidade externa

## Referencias

- `docs/telemetry-feedback-cycle.md` — documentacao do ciclo completo
- `internal/telemetry/` — implementacao do coletor e reporter
