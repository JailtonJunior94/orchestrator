# Especificações das Seções dos Dashboards

Especificações detalhadas de painéis para cada dashboard padrão. Cada seção define painéis com tipo de visualização, queries PromQL/LogQL/TraceQL e layout.

**IMPORTANTE:** Todas as queries PromQL devem usar os nomes de métricas exportados para o Prometheus (pontos → underscores, com sufixo de unidade). Consultar `references/otel-semantic-metrics.md` para o mapeamento.

---

## Seção 1: Dashboard Visão Geral do Serviço

Este dashboard fornece uma visão de saúde de alto nível. É o primeiro dashboard que um desenvolvedor ou SRE abre.

### Linha: Indicadores Chave (y=0)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Taxa de Requisições | stat | 6 | `sum(rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` |
| Taxa de Erros (%) | stat (threshold vermelho >1%) | 6 | `sum(rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment", http_response_status_code=~"5.."}[$__rate_interval])) / sum(rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])) * 100` |
| Latência P95 | stat | 6 | `histogram_quantile(0.95, sum by(le) (rate(http_server_request_duration_seconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` |
| Requisições Ativas | stat | 6 | `sum(http_server_active_requests{service_name=~"$service", deployment_environment=~"$environment"})` |

### Linha: Tendências de Requisições (y=9)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Taxa de Requisições ao Longo do Tempo | timeseries | 12 | `sum by(http_route) (rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` Legenda: `{{http_route}}` |
| Taxa de Erros ao Longo do Tempo | timeseries | 12 | `sum by(http_response_status_code) (rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment", http_response_status_code=~"[45].."}[$__rate_interval]))` Legenda: `{{http_response_status_code}}` |

### Linha: Distribuição de Latência (y=18)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Percentis de Latência | timeseries (multi-query) | 12 | P50: `histogram_quantile(0.50, sum by(le) (rate(http_server_request_duration_seconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` P95: mesmo com 0.95 P99: mesmo com 0.99 |
| Heatmap de Latência | heatmap | 12 | `sum by(le) (increase(http_server_request_duration_seconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` |

### Linha: Saúde do Runtime (y=27)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Goroutines | timeseries | 8 | `process_runtime_go_goroutines{service_name=~"$service", deployment_environment=~"$environment"}` |
| Memória Heap | timeseries | 8 | `process_runtime_go_mem_heap_alloc{service_name=~"$service", deployment_environment=~"$environment"}` Unidade: bytes |
| Duração de Pausa GC | timeseries | 8 | `rate(process_runtime_go_gc_pause_ns_sum{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]) / rate(process_runtime_go_gc_pause_ns_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])` Unidade: ns |

---

## Seção 2: Dashboard de Performance HTTP/gRPC

Análise detalhada de performance no nível de requisição.

### Linha: Visão Geral HTTP (y=0)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Requisições por Método | timeseries (empilhado) | 12 | `sum by(http_request_method) (rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` |
| Requisições por Código de Status | timeseries (empilhado) | 12 | `sum by(http_response_status_code) (rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` |

### Linha: Latência HTTP por Rota (y=9)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Latência P95 por Rota | timeseries | 12 | `histogram_quantile(0.95, sum by(le, http_route) (rate(http_server_request_duration_seconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` Legenda: `{{http_route}}` |
| Latência Média por Rota | timeseries | 12 | `sum by(http_route) (rate(http_server_request_duration_seconds_sum{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])) / sum by(http_route) (rate(http_server_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` Legenda: `{{http_route}}` |

### Linha: Tamanho de Requisição/Resposta HTTP (y=18)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Tamanho do Corpo da Requisição P95 | timeseries | 12 | `histogram_quantile(0.95, sum by(le) (rate(http_server_request_body_size_bytes_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` |
| Tamanho do Corpo da Resposta P95 | timeseries | 12 | `histogram_quantile(0.95, sum by(le) (rate(http_server_response_body_size_bytes_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` |

### Linha: Cliente HTTP (Saída) (y=27)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Taxa de Requisições de Saída | timeseries | 12 | `sum by(server_address) (rate(http_client_request_duration_seconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` Legenda: `{{server_address}}` |
| Latência P95 de Saída | timeseries | 12 | `histogram_quantile(0.95, sum by(le, server_address) (rate(http_client_request_duration_seconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` Legenda: `{{server_address}}` |

### Linha: Servidor gRPC (y=36) — Incluir apenas se instrumentação gRPC detectada
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Taxa de Requisições gRPC por Método | timeseries | 12 | `sum by(rpc_method) (rate(rpc_server_duration_milliseconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` Legenda: `{{rpc_method}}` |
| Latência gRPC por Método | timeseries | 12 | `histogram_quantile(0.95, sum by(le, rpc_method) (rate(rpc_server_duration_milliseconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` Legenda: `{{rpc_method}}` |

### Linha: Análise de Erros gRPC (y=45) — Incluir apenas se instrumentação gRPC detectada
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Erros gRPC por Código de Status | timeseries (empilhado) | 12 | `sum by(rpc_grpc_status_code) (rate(rpc_server_duration_milliseconds_count{service_name=~"$service", deployment_environment=~"$environment", rpc_grpc_status_code!="0"}[$__rate_interval]))` Legenda: `{{rpc_grpc_status_code}}` |
| Taxa de Erros gRPC (%) | timeseries | 12 | `sum(rate(rpc_server_duration_milliseconds_count{service_name=~"$service", deployment_environment=~"$environment", rpc_grpc_status_code!="0"}[$__rate_interval])) / sum(rate(rpc_server_duration_milliseconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])) * 100` |

### Linha: Cliente gRPC (y=54) — Incluir apenas se instrumentação de cliente gRPC detectada
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Taxa de Requisições do Cliente gRPC | timeseries | 12 | `sum by(rpc_service, rpc_method) (rate(rpc_client_duration_milliseconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` Legenda: `{{rpc_service}}/{{rpc_method}}` |
| Latência P95 do Cliente gRPC | timeseries | 12 | `histogram_quantile(0.95, sum by(le, rpc_service) (rate(rpc_client_duration_milliseconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` Legenda: `{{rpc_service}}` |

---

## Seção 3: Dashboard de Banco de Dados e Dependências Externas

### Linha: Pool de Conexões do Banco (y=0)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Conexões Ativas | timeseries | 8 | `db_client_connections_usage{service_name=~"$service", deployment_environment=~"$environment", db_connection_state="used"}` |
| Conexões Ociosas | timeseries | 8 | `db_client_connections_usage{service_name=~"$service", deployment_environment=~"$environment", db_connection_state="idle"}` |
| Conexões Máximas | stat | 8 | `db_client_connections_max{service_name=~"$service", deployment_environment=~"$environment"}` |

### Linha: Performance de Operações do Banco (y=9)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Duração de Query P95 | timeseries | 12 | `histogram_quantile(0.95, sum by(le, db_operation) (rate(db_client_operation_duration_milliseconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` Legenda: `{{db_operation}}` |
| Taxa de Queries por Operação | timeseries (empilhado) | 12 | `sum by(db_operation) (rate(db_client_operation_duration_milliseconds_count{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` Legenda: `{{db_operation}}` |

### Linha: Queries Lentas do Banco (y=18)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Tempo de Espera de Conexão P95 | timeseries | 12 | `histogram_quantile(0.95, sum by(le) (rate(db_client_connections_wait_time_milliseconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` |
| Tempo de Criação de Conexão P95 | timeseries | 12 | `histogram_quantile(0.95, sum by(le) (rate(db_client_connections_create_time_milliseconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` |

### Linha: Mensageria — Incluir apenas se instrumentação de mensageria detectada (y=27)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Taxa de Mensagens Publicadas | timeseries | 8 | `sum by(messaging_destination_name) (rate(messaging_publish_messages_total{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval]))` Legenda: `{{messaging_destination_name}}` |
| Duração de Publicação P95 | timeseries | 8 | `histogram_quantile(0.95, sum by(le, messaging_destination_name) (rate(messaging_publish_duration_milliseconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` |
| Duração de Processamento P95 | timeseries | 8 | `histogram_quantile(0.95, sum by(le, messaging_destination_name) (rate(messaging_process_duration_milliseconds_bucket{service_name=~"$service", deployment_environment=~"$environment"}[$__rate_interval])))` |

---

## Seção 4: Dashboard Explorador de Logs

Usa datasource Loki (`${DS_LOKI}`).

### Linha: Volume de Logs (y=0)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Volume de Logs por Nível | timeseries (empilhado, barra) | 24 | `sum by(level) (count_over_time({service_name=~"$service", deployment_environment=~"$environment"} [$__auto]))` Legenda: `{{level}}` Cores: error=vermelho, warn=amarelo, info=azul, debug=cinza |

### Linha: Logs de Erro (y=9)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Stream de Logs de Erro | logs | 24 | `{service_name=~"$service", deployment_environment=~"$environment"} |= "" \| json \| level=~"error\|ERROR\|fatal\|FATAL"` |

### Linha: Busca de Logs (y=27)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Stream Completo de Logs | logs | 24 | `{service_name=~"$service", deployment_environment=~"$environment"} |= "$search"` Adicionar variável de template `$search` (tipo texto, label "Buscar"). |

### Linha: Métricas de Logs (y=45)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Taxa de Contagem de Erros | timeseries | 12 | `sum(count_over_time({service_name=~"$service", deployment_environment=~"$environment"} \| json \| level=~"error\|ERROR" [$__auto]))` |
| Throughput de Logs | stat | 12 | `sum(count_over_time({service_name=~"$service", deployment_environment=~"$environment"} [$__auto]))` |

---

## Seção 5: Dashboard Explorador de Traces

Usa datasource Tempo (`${DS_TEMPO}`).

### Linha: Busca de Traces (y=0)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Lista de Traces | traces | 24 | TraceQL: `{resource.service.name=~"$service" && resource.deployment.environment=~"$environment" && status=error}` Usar o tipo de painel de busca do Tempo. Incluir variável de template `$trace_id` (tipo texto) para busca direta de trace. |

### Linha: Mapa de Serviços (y=18)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Grafo de Serviços | node-graph | 24 | Usar recurso de grafo de serviços do Tempo. Datasource: `${DS_TEMPO}`, tipo de query: "Service Graph". |

### Linha: Métricas de Span (y=36) — Requer conector de métricas de span no OTel Collector
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Duração P95 de Span por Operação | timeseries | 12 | `histogram_quantile(0.95, sum by(le, span_name) (rate(traces_spanmetrics_latency_bucket{service_name=~"$service"}[$__rate_interval])))` Legenda: `{{span_name}}` |
| Taxa de Erros de Span por Operação | timeseries | 12 | `sum by(span_name) (rate(traces_spanmetrics_calls_total{service_name=~"$service", status_code="STATUS_CODE_ERROR"}[$__rate_interval])) / sum by(span_name) (rate(traces_spanmetrics_calls_total{service_name=~"$service"}[$__rate_interval])) * 100` Legenda: `{{span_name}}` |

### Linha: Correlação Trace-para-Logs (y=45)
| Painel | Tipo | Largura | Query |
|---|---|---|---|
| Logs do Trace | logs | 24 | `{service_name=~"$service"} \| json \| trace_id=~"$trace_id"` Link: Configurar datasource Tempo com vinculação trace-para-logs ao Loki. |
