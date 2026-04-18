# Referência de Métricas de Convenção Semântica OpenTelemetry

Métricas padrão geradas automaticamente pelas bibliotecas de instrumentação OTel. Usar para cruzar métricas descobertas e identificar as auto-geradas.

## Métricas de Servidor HTTP (otelhttp / otelgin / otelecho / otelfiber / otelchi / otelmux)

| Nome da Métrica | Tipo | Unidade | Descrição |
|---|---|---|---|
| `http.server.request.duration` | Histogram | `s` | Duração das requisições do servidor HTTP |
| `http.server.active_requests` | UpDownCounter | `{request}` | Número de requisições HTTP ativas no servidor |
| `http.server.request.body.size` | Histogram | `By` | Tamanho do corpo das requisições do servidor HTTP |
| `http.server.response.body.size` | Histogram | `By` | Tamanho do corpo das respostas do servidor HTTP |

### Atributos Comuns
- `http.request.method` — Método HTTP (GET, POST, etc.)
- `http.response.status_code` — Código de status HTTP (200, 404, 500, etc.)
- `http.route` — Padrão de rota correspondente (ex.: `/api/users/:id`)
- `url.scheme` — Esquema da URL (http, https)
- `server.address` — Hostname do servidor
- `server.port` — Porta do servidor
- `network.protocol.version` — Versão do HTTP (1.1, 2)
- `error.type` — Classificação do erro

## Métricas de Cliente HTTP (otelhttp client)

| Nome da Métrica | Tipo | Unidade | Descrição |
|---|---|---|---|
| `http.client.request.duration` | Histogram | `s` | Duração das requisições do cliente HTTP |
| `http.client.request.body.size` | Histogram | `By` | Tamanho do corpo das requisições do cliente HTTP |
| `http.client.response.body.size` | Histogram | `By` | Tamanho do corpo das respostas do cliente HTTP |

### Atributos Comuns
- `http.request.method` — Método HTTP
- `http.response.status_code` — Código de status da resposta
- `server.address` — Hostname do servidor destino
- `server.port` — Porta do servidor destino
- `error.type` — Classificação do erro

## Métricas gRPC (otelgrpc)

| Nome da Métrica | Tipo | Unidade | Descrição |
|---|---|---|---|
| `rpc.server.duration` | Histogram | `ms` | Duração das chamadas do servidor gRPC |
| `rpc.server.request.size` | Histogram | `By` | Tamanho das mensagens de requisição do servidor gRPC |
| `rpc.server.response.size` | Histogram | `By` | Tamanho das mensagens de resposta do servidor gRPC |
| `rpc.server.requests_per_rpc` | Histogram | `{count}` | Mensagens recebidas por RPC |
| `rpc.server.responses_per_rpc` | Histogram | `{count}` | Mensagens enviadas por RPC |
| `rpc.client.duration` | Histogram | `ms` | Duração das chamadas do cliente gRPC |
| `rpc.client.request.size` | Histogram | `By` | Tamanho das mensagens de requisição do cliente gRPC |
| `rpc.client.response.size` | Histogram | `By` | Tamanho das mensagens de resposta do cliente gRPC |

### Atributos Comuns
- `rpc.system` — Sistema RPC (grpc)
- `rpc.service` — Nome completo do serviço gRPC
- `rpc.method` — Nome do método gRPC
- `rpc.grpc.status_code` — Código de status gRPC (0=OK, 1=CANCELLED, etc.)
- `server.address` — Hostname do servidor
- `network.transport` — Protocolo de transporte (tcp)

## Métricas de Banco de Dados (otelsql / otelgorm)

| Nome da Métrica | Tipo | Unidade | Descrição |
|---|---|---|---|
| `db.client.connections.usage` | UpDownCounter | `{connection}` | Uso atual do pool de conexões |
| `db.client.connections.idle.max` | UpDownCounter | `{connection}` | Máximo de conexões ociosas configuradas |
| `db.client.connections.idle.min` | UpDownCounter | `{connection}` | Mínimo de conexões ociosas configuradas |
| `db.client.connections.max` | UpDownCounter | `{connection}` | Máximo de conexões abertas configuradas |
| `db.client.connections.wait_time` | Histogram | `ms` | Tempo de espera por uma conexão |
| `db.client.connections.create_time` | Histogram | `ms` | Tempo para criar uma conexão |
| `db.client.operation.duration` | Histogram | `ms` | Duração das operações de banco de dados |

### Atributos Comuns
- `db.system` — Sistema de banco de dados (postgresql, mysql, mongodb, redis)
- `db.name` — Nome do banco de dados
- `db.operation` — Tipo de operação (SELECT, INSERT, UPDATE, DELETE)
- `db.statement` — Statement do banco (pode estar sanitizada)
- `db.connection.state` — Estado da conexão (idle, used)
- `server.address` — Hostname do servidor de banco de dados
- `server.port` — Porta do servidor de banco de dados

## Métricas de Runtime Go (instrumentação de runtime)

| Nome da Métrica | Tipo | Unidade | Descrição |
|---|---|---|---|
| `process.runtime.go.goroutines` | UpDownCounter | `{goroutine}` | Número de goroutines ativas |
| `process.runtime.go.mem.heap_alloc` | UpDownCounter | `By` | Memória heap alocada |
| `process.runtime.go.mem.heap_idle` | UpDownCounter | `By` | Memória heap ociosa |
| `process.runtime.go.mem.heap_inuse` | UpDownCounter | `By` | Memória heap em uso |
| `process.runtime.go.mem.heap_objects` | UpDownCounter | `{object}` | Número de objetos no heap |
| `process.runtime.go.mem.heap_released` | UpDownCounter | `By` | Memória heap liberada para o SO |
| `process.runtime.go.mem.heap_sys` | UpDownCounter | `By` | Memória heap obtida do SO |
| `process.runtime.go.mem.live_objects` | UpDownCounter | `{object}` | Objetos vivos (alloc - frees) |
| `process.runtime.go.gc.count` | Counter | `{gc_cycle}` | Número de ciclos GC completados |
| `process.runtime.go.gc.pause_ns` | Histogram | `ns` | Duração da pausa stop-the-world do GC |
| `process.runtime.go.mem.lookups` | Counter | `{lookup}` | Número de lookups de ponteiro |
| `runtime.uptime` | Counter | `s` | Tempo de atividade do processo |

## Métricas de Mensageria (Kafka, RabbitMQ, NATS)

| Nome da Métrica | Tipo | Unidade | Descrição |
|---|---|---|---|
| `messaging.publish.duration` | Histogram | `ms` | Duração da publicação de mensagem |
| `messaging.receive.duration` | Histogram | `ms` | Duração do recebimento de mensagem |
| `messaging.process.duration` | Histogram | `ms` | Duração do processamento de mensagem |
| `messaging.publish.messages` | Counter | `{message}` | Número de mensagens publicadas |
| `messaging.receive.messages` | Counter | `{message}` | Número de mensagens recebidas |

### Atributos Comuns
- `messaging.system` — Sistema de mensageria (kafka, rabbitmq, nats)
- `messaging.destination.name` — Nome da fila/tópico
- `messaging.operation` — Operação (publish, receive, process)
- `messaging.message.id` — Identificador da mensagem

## Atributos de Resource (Aplicados a Todos os Sinais)

Estes atributos são definidos no Resource OTel e aparecem em todas as métricas, traces e logs:

| Atributo | Descrição |
|---|---|
| `service.name` | Nome lógico do serviço |
| `service.version` | Versão do serviço |
| `service.namespace` | Namespace do serviço |
| `service.instance.id` | Identificador único da instância |
| `deployment.environment` | Ambiente de deploy (dev, staging, production) |
| `host.name` | Hostname |
| `host.arch` | Arquitetura do host |
| `os.type` | Tipo do sistema operacional |
| `process.pid` | ID do processo |
| `process.runtime.name` | Nome do runtime (go) |
| `process.runtime.version` | Versão do Go |
| `telemetry.sdk.name` | Nome do SDK (opentelemetry) |
| `telemetry.sdk.language` | Linguagem do SDK (go) |
| `telemetry.sdk.version` | Versão do SDK |

## Mapeamento de Nome de Métrica OTel para Prometheus

Métricas OpenTelemetry são exportadas para o Prometheus com estas transformações:
1. Pontos (`.`) substituídos por underscores (`_`)
2. Métricas Counter recebem sufixo `_total`
3. Métricas Histogram geram séries `_bucket`, `_sum`, `_count`
4. Unidade é adicionada: ex.: `http.server.request.duration` (unidade: `s`) → `http_server_request_duration_seconds`
5. Unidade bytes: `By` → `bytes`
6. Unidade segundos: `s` → `seconds`
7. Unidade milissegundos: `ms` → `milliseconds`

### Exemplos
| Nome OTel | Nome Prometheus |
|---|---|
| `http.server.request.duration` (s) | `http_server_request_duration_seconds_bucket` / `_sum` / `_count` |
| `http.server.active_requests` | `http_server_active_requests` |
| `db.client.operation.duration` (ms) | `db_client_operation_duration_milliseconds_bucket` / `_sum` / `_count` |
| `process.runtime.go.goroutines` | `process_runtime_go_goroutines` |
| `api.counter` ({call}) | `api_counter_calls_total` |
