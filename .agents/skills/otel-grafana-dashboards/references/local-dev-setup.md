# Configuração do Ambiente Local com grafana/otel-lgtm

## Template Docker Compose

Gerar o seguinte `docker-compose.observability.yml`:

```yaml
version: "3.8"

services:
  otel-lgtm:
    image: grafana/otel-lgtm:0.7.5
    container_name: otel-lgtm
    ports:
      - "3000:3000"   # Interface Grafana
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "9090:9090"   # Prometheus
    volumes:
      - otel-lgtm-data:/data
      - ./provisioning/dashboards/dashboards.yaml:/otel-lgtm/grafana/conf/provisioning/dashboards/custom.yaml:ro
      - ./dashboards:/otel-lgtm/grafana/conf/provisioning/dashboards/custom:ro
    environment:
      - GF_DASHBOARDS_DEFAULT_HOME_DASHBOARD_PATH=/otel-lgtm/grafana/conf/provisioning/dashboards/custom/service-overview.json
    restart: unless-stopped

volumes:
  otel-lgtm-data:
    driver: local
```

## Configuração de Provisioning de Dashboards

Gerar `provisioning/dashboards/dashboards.yaml`:

```yaml
apiVersion: 1

providers:
  - name: "Dashboards Customizados"
    orgId: 1
    folder: "OpenTelemetry"
    type: file
    disableDeletion: false
    editable: true
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /otel-lgtm/grafana/conf/provisioning/dashboards/custom
      foldersFromFilesStructure: false
```

## Variáveis de Ambiente da Aplicação

Para a aplicação instrumentada enviar telemetria para a stack otel-lgtm local:

```bash
# Desenvolvimento local
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_SERVICE_NAME=meu-servico
OTEL_RESOURCE_ATTRIBUTES=deployment.environment=development,service.version=0.1.0

# Se usar exporter gRPC
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```

## Estrutura de Diretórios

O diretório de saída deve seguir esta estrutura:

```
observability/
├── docker-compose.observability.yml
├── dashboards/
│   ├── service-overview.json
│   ├── http-grpc-performance.json
│   ├── database-dependencies.json
│   ├── logs-explorer.json
│   ├── traces-explorer.json
│   └── custom-app-metrics.json         # Apenas se métricas customizadas encontradas
├── provisioning/
│   └── dashboards/
│       └── dashboards.yaml
```

## Acessando a Stack

Após `docker compose -f observability/docker-compose.observability.yml up -d`:

| Serviço | URL | Credenciais |
|---|---|---|
| Grafana | http://localhost:3000 | admin / admin |
| Prometheus | http://localhost:9090 | — |

## Datasources do Grafana (Pré-configurados no otel-lgtm)

A imagem otel-lgtm pré-configura estes datasources:
- **Prometheus**: Para queries de métricas PromQL
- **Loki**: Para queries de logs LogQL
- **Tempo**: Para queries de traces TraceQL

Nenhuma configuração manual de datasource é necessária. As variáveis de template dos dashboards (`${DS_PROMETHEUS}`, `${DS_LOKI}`, `${DS_TEMPO}`) serão resolvidas para os datasources pré-configurados.

## Persistência de Dados

O volume nomeado `otel-lgtm-data` persiste todos os dados (métricas, logs, traces, configurações do Grafana) entre reinicializações do container. Para resetar, executar:

```bash
docker compose -f observability/docker-compose.observability.yml down -v
```
