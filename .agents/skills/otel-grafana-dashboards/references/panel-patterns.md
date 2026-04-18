# Padrões JSON de Painéis Grafana

Estruturas JSON reutilizáveis para gerar painéis de dashboard. Todos os painéis usam variáveis de template de datasource para portabilidade entre ambientes.

## Definições de Variáveis de Datasource

Incluir estas em `templating.list` de cada dashboard:

```json
{
  "name": "DS_PROMETHEUS",
  "type": "datasource",
  "query": "prometheus",
  "current": { "text": "Prometheus", "value": "Prometheus" },
  "hide": 0,
  "label": "Prometheus"
}
```

```json
{
  "name": "DS_LOKI",
  "type": "datasource",
  "query": "loki",
  "current": { "text": "Loki", "value": "Loki" },
  "hide": 0,
  "label": "Loki"
}
```

```json
{
  "name": "DS_TEMPO",
  "type": "datasource",
  "query": "tempo",
  "current": { "text": "Tempo", "value": "Tempo" },
  "hide": 0,
  "label": "Tempo"
}
```

## Definições de Variáveis de Serviço e Ambiente

```json
{
  "name": "service",
  "type": "query",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "query": "label_values(http_server_request_duration_seconds_count, service_name)",
  "refresh": 2,
  "includeAll": true,
  "multi": true,
  "label": "Serviço"
}
```

```json
{
  "name": "environment",
  "type": "query",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "query": "label_values(http_server_request_duration_seconds_count, deployment_environment)",
  "refresh": 2,
  "includeAll": true,
  "multi": true,
  "label": "Ambiente"
}
```

## Tipo de Painel: Stat

```json
{
  "id": null,
  "title": "TITULO",
  "type": "stat",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "gridPos": { "h": 8, "w": 6, "x": 0, "y": 0 },
  "targets": [
    {
      "expr": "EXPRESSAO_PROMQL",
      "legendFormat": "LEGENDA",
      "refId": "A"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "thresholds": {
        "mode": "absolute",
        "steps": [
          { "color": "green", "value": null },
          { "color": "yellow", "value": 80 },
          { "color": "red", "value": 100 }
        ]
      },
      "unit": "UNIDADE",
      "mappings": []
    },
    "overrides": []
  },
  "options": {
    "reduceOptions": { "calcs": ["lastNotNull"], "fields": "", "values": false },
    "orientation": "auto",
    "textMode": "auto",
    "colorMode": "value",
    "graphMode": "area",
    "justifyMode": "auto"
  }
}
```

### Unidades Comuns para Painéis Stat
- Taxa de requisições: `reqps` (requisições por segundo)
- Taxa de erros: `percent`
- Latência: `s` (segundos), `ms` (milissegundos)
- Contagem: `short`
- Bytes: `bytes`

## Tipo de Painel: Time Series

```json
{
  "id": null,
  "title": "TITULO",
  "type": "timeseries",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
  "targets": [
    {
      "expr": "EXPRESSAO_PROMQL",
      "legendFormat": "LEGENDA",
      "refId": "A"
    }
  ],
  "fieldConfig": {
    "defaults": {
      "custom": {
        "drawStyle": "line",
        "lineInterpolation": "smooth",
        "lineWidth": 1,
        "fillOpacity": 10,
        "gradientMode": "none",
        "spanNulls": false,
        "showPoints": "auto",
        "pointSize": 5,
        "stacking": { "mode": "none", "group": "A" },
        "axisPlacement": "auto",
        "scaleDistribution": { "type": "linear" },
        "thresholdsStyle": { "mode": "off" }
      },
      "unit": "UNIDADE",
      "thresholds": {
        "mode": "absolute",
        "steps": [{ "color": "green", "value": null }]
      }
    },
    "overrides": []
  },
  "options": {
    "tooltip": { "mode": "multi", "sort": "desc" },
    "legend": { "displayMode": "table", "placement": "bottom", "calcs": ["mean", "max", "last"] }
  }
}
```

### Para Time Series Empilhado
Definir `"stacking": { "mode": "normal", "group": "A" }` e `"fillOpacity": 50`.

## Tipo de Painel: Heatmap

```json
{
  "id": null,
  "title": "TITULO",
  "type": "heatmap",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
  "targets": [
    {
      "expr": "EXPRESSAO_PROMQL",
      "format": "heatmap",
      "legendFormat": "{{le}}",
      "refId": "A"
    }
  ],
  "options": {
    "calculate": false,
    "yAxis": { "unit": "s" },
    "color": { "scheme": "Oranges", "mode": "scheme" },
    "cellGap": 1,
    "tooltip": { "show": true, "yHistogram": true }
  }
}
```

## Tipo de Painel: Logs

```json
{
  "id": null,
  "title": "TITULO",
  "type": "logs",
  "datasource": { "uid": "${DS_LOKI}" },
  "gridPos": { "h": 16, "w": 24, "x": 0, "y": 0 },
  "targets": [
    {
      "expr": "EXPRESSAO_LOGQL",
      "refId": "A"
    }
  ],
  "options": {
    "showTime": true,
    "showLabels": true,
    "showCommonLabels": false,
    "wrapLogMessage": true,
    "prettifyLogMessage": true,
    "enableLogDetails": true,
    "sortOrder": "Descending",
    "dedupStrategy": "none"
  }
}
```

## Tipo de Painel: Linha (Seção Colapsável)

```json
{
  "id": null,
  "title": "TITULO_DA_LINHA",
  "type": "row",
  "gridPos": { "h": 1, "w": 24, "x": 0, "y": 0 },
  "collapsed": false,
  "panels": []
}
```

## Tipo de Painel: Tabela

```json
{
  "id": null,
  "title": "TITULO",
  "type": "table",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "gridPos": { "h": 8, "w": 24, "x": 0, "y": 0 },
  "targets": [
    {
      "expr": "EXPRESSAO_PROMQL",
      "format": "table",
      "instant": true,
      "refId": "A"
    }
  ],
  "transformations": [
    {
      "id": "organize",
      "options": {
        "excludeByName": { "Time": true },
        "renameByName": {}
      }
    }
  ],
  "fieldConfig": {
    "defaults": { "unit": "UNIDADE" },
    "overrides": []
  },
  "options": {
    "showHeader": true,
    "sortBy": [{ "displayName": "Value", "desc": true }]
  }
}
```

## Tipo de Painel: Bar Gauge

```json
{
  "id": null,
  "title": "TITULO",
  "type": "bargauge",
  "datasource": { "uid": "${DS_PROMETHEUS}" },
  "gridPos": { "h": 8, "w": 12, "x": 0, "y": 0 },
  "targets": [
    {
      "expr": "EXPRESSAO_PROMQL",
      "legendFormat": "LEGENDA",
      "instant": true,
      "refId": "A"
    }
  ],
  "options": {
    "reduceOptions": { "calcs": ["lastNotNull"], "fields": "", "values": false },
    "orientation": "horizontal",
    "displayMode": "gradient",
    "showUnfilled": true
  },
  "fieldConfig": {
    "defaults": { "unit": "UNIDADE", "thresholds": { "mode": "absolute", "steps": [{ "color": "green", "value": null }] } },
    "overrides": []
  }
}
```

## Configuração de Links Entre Dashboards

Adicionar links de navegação entre dashboards:

```json
{
  "links": [
    {
      "title": "Visão Geral do Serviço",
      "url": "/d/service-overview/service-overview?var-service=$service&var-environment=$environment",
      "type": "link",
      "icon": "dashboard",
      "targetBlank": false
    },
    {
      "title": "Explorador de Traces",
      "url": "/d/traces-explorer/traces-explorer?var-service=$service&var-environment=$environment",
      "type": "link",
      "icon": "dashboard",
      "targetBlank": false
    }
  ]
}
```
