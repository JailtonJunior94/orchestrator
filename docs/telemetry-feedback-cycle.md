# Ciclo de Feedback de Telemetria

Este documento descreve como ativar, coletar, agregar e interpretar os dados de telemetria do harness para fechar o ciclo coleta → relatório → decisão.

---

## Ativação

A telemetria é opt-in. Para ativar, defina a variável de ambiente antes de usar o harness:

```bash
export GOVERNANCE_TELEMETRY=1
```

Sem essa variável, nenhum dado é escrito. Para ativar permanentemente no projeto, adicione ao `.envrc` ou ao script de setup do repositório.

---

## Coleta

Cada invocação de skill que passa pela skill `agent-governance` registra uma linha em `.agents/telemetry.log`:

```
2026-04-20T10:30:00Z skill=bugfix ref=bug-schema.json
2026-04-20T10:31:00Z skill=review ref=security.md
2026-04-20T10:32:00Z skill=bugfix ref=testing.md
```

Formato: `<RFC3339> skill=<nome> ref=<nome>` — a linha `ref=` é omitida quando nenhuma referência foi carregada.

O arquivo é append-only e nunca truncado automaticamente.

---

## Agregação

Use o subcomando `telemetry report` para agregar os dados:

```bash
# Relatório completo (todo o histórico)
ai-spec-harness telemetry report

# Últimas 24 horas
ai-spec-harness telemetry report --since 24h

# Última semana
ai-spec-harness telemetry report --since 168h

# JSON estruturado (integrável com jq ou dashboards)
ai-spec-harness telemetry report --format json

# Filtrar e exportar
ai-spec-harness telemetry report --since 24h --format json | jq '.skills'
```

O subcomando legado `telemetry summary` ainda está disponível para contagens brutas.

---

## Relatório

Exemplo de saída do `telemetry report`:

```
Relatório de Telemetria
Período: últimas 24h
Total de invocações: 12

Skills Mais Usadas (top 5):
  1. bugfix                         5  (41.7%)
  2. review                         4  (33.3%)
  3. go-implementation              3  (25.0%)

Referências Mais Carregadas (top 5):
  1. testing.md                     6
  2. security.md                    4
  3. error-handling.md              2

Métricas:
  Refs por invocação (média): 1.0
  Tokens estimados:           6840 (12 refs × 570 tok/ref)

Alertas:
  ⚠ skill 'foo' invocada 2 vez(es) sem carregar nenhuma referência — possível bypass de governança
```

---

## Interpretação das Métricas

| Métrica | O que significa | Ação sugerida |
|---|---|---|
| **Skills mais usadas** | Quais skills geram mais valor operacional | Priorizar melhorias e documentação nas skills com maior adoção |
| **Refs mais carregadas** | Quais referências são mais consultadas | Verificar se referências muito acessadas estão atualizadas e bem estruturadas |
| **Taxa refs/invocação** | Média de arquivos carregados por chamada | Taxa alta (> 3) pode indicar skills "gordas" — candidatas a otimização de tokens |
| **Tokens estimados** | Custo operacional aproximado no período | Comparar entre períodos para avaliar tendência de consumo |
| **Alertas de bypass** | Skills invocadas sem carregar nenhuma referência | Investigar se o carregamento de governança está sendo pulado intencionalmente |

---

## Loop de Decisão

```
GOVERNANCE_TELEMETRY=1
        ↓
  uso normal de skills
        ↓
  .agents/telemetry.log acumula entradas
        ↓
  ai-spec-harness telemetry report --since 168h
        ↓
  análise de métricas:
    - skill mais usada → priorizar melhoria
    - taxa refs alta → candidata a otimização
    - alerta bypass → investigar governança
        ↓
  decisão de produto ou ajuste de skill
        ↓
  (novo ciclo)
```

---

## Limpeza do Log

O log não é rotacionado automaticamente. Para limpar manualmente:

```bash
# Ver tamanho
ls -lh .agents/telemetry.log

# Arquivar e reiniciar
mv .agents/telemetry.log .agents/telemetry-$(date +%Y%m%d).log.bak
```
