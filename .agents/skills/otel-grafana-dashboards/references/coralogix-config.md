# Compatibilidade de Dashboards com Coralogix

O ambiente Coralogix já está configurado e operacional (collector, API keys, pipelines de ingestão). Este documento serve como referência para garantir que os dashboards gerados sejam importáveis sem ajustes.

## Por que os Dashboards São Compatíveis

Os dashboards gerados por esta skill funcionam no Coralogix sem modificação porque:

1. **Métricas**: O Coralogix armazena métricas OTLP e expõe uma API de query compatível com Prometheus. As mesmas queries PromQL funcionam.
2. **Logs**: O Coralogix suporta queries de logs. Queries LogQL dos painéis Loki são compatíveis com o datasource de logs do Coralogix.
3. **Traces**: O Coralogix suporta tracing distribuído com interface compatível com Jaeger. Queries TraceQL dos painéis Tempo mapeiam para a busca de traces do Coralogix.
4. **Labels**: Atributos de resource (`service.name`, `deployment.environment`) são preservados através do OTLP e disponíveis como labels em todos os motores de query.
5. **Nomes de métricas**: A conversão OTel → Prometheus (pontos → underscores, sufixos de unidade) é preservada pelo Coralogix. Os mesmos nomes usados no Prometheus local funcionam em produção.

## Mapeamento de Datasources

Ao importar no Grafana do Coralogix, mapear as variáveis de template:

| Variável do Dashboard | Datasource no Coralogix |
|---|---|
| `${DS_PROMETHEUS}` | Datasource de Métricas do Coralogix |
| `${DS_LOKI}` | Datasource de Logs do Coralogix |
| `${DS_TEMPO}` | Datasource de Traces/Spans do Coralogix |

## Importando Dashboards no Grafana do Coralogix

1. Ir para Coralogix → Dashboards → Grafana
2. Clicar + → Importar
3. Fazer upload do arquivo JSON ou colar seu conteúdo
4. Mapear as variáveis de datasource conforme tabela acima

## Variáveis de Filtro Adicionais para Coralogix

Para melhor integração com o Coralogix, os dashboards devem incluir estas variáveis de template opcionais (não quebram o ambiente local, pois são filtros adicionais):

```json
{
  "name": "cx_application_name",
  "type": "query",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "query": "label_values(up, cx_application_name)",
  "refresh": 2,
  "includeAll": true,
  "multi": true,
  "label": "Aplicação (Coralogix)",
  "hide": 2
}
```

```json
{
  "name": "cx_subsystem_name",
  "type": "query",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "query": "label_values(up, cx_subsystem_name)",
  "refresh": 2,
  "includeAll": true,
  "multi": true,
  "label": "Subsistema (Coralogix)",
  "hide": 2
}
```

**Nota:** `"hide": 2` oculta essas variáveis por padrão. No ambiente Coralogix, alterar para `"hide": 0` para exibí-las. No ambiente local elas são ignoradas pois o label não existe.

## Funcionalidades do Coralogix Aproveitáveis com os Dashboards

- **Enriquecimento automático**: O Coralogix enriquece logs com geo-IP, parsing de user-agent, etc. Painéis de logs já se beneficiam disso sem configuração adicional.
- **Alertas**: Qualquer painel de dashboard pode ser usado como fonte de alerta no sistema de alertas do Coralogix. Os painéis de Taxa de Erros e Latência P95 do dashboard Visão Geral são candidatos naturais.
- **SLO**: Métricas do dashboard Visão Geral do Serviço (taxa de erros, latência) podem alimentar definições de SLO do Coralogix diretamente.
- **Correlação**: Os links entre dashboards (Visão Geral → Traces → Logs) funcionam no Grafana hospedado do Coralogix, permitindo navegação fluida entre sinais.

## Regras de Compatibilidade para Geração de Queries

Ao gerar queries para os dashboards, respeitar estas regras para garantir compatibilidade entre ambientes:

1. **Usar apenas PromQL padrão** — sem extensões ou funções proprietárias
2. **Usar apenas LogQL padrão** — sem operadores específicos do Coralogix
3. **Usar apenas TraceQL padrão** — sem filtros específicos do Coralogix
4. **Nomes de métricas**: Sempre usar o formato Prometheus (underscores, sufixos de unidade)
5. **Labels**: Usar `service_name` (não `service.name`) pois o Prometheus converte pontos em underscores
6. **Variáveis**: Usar `$__rate_interval` e `$__auto` em vez de intervalos hardcoded
7. **Datasources**: Sempre referenciar via variáveis de template (`${DS_PROMETHEUS}`, etc.)
