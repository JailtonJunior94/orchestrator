---
name: otel-grafana-dashboards
description: Analisa codebases Go instrumentados com OpenTelemetry e gera arquivos JSON de dashboards Grafana prontos para produção, cobrindo métricas (Prometheus), traces (Tempo) e logs (Loki). Os dashboards são compatíveis com desenvolvimento local (grafana/otel-lgtm) e produção (Coralogix já configurado). Usar ao criar dashboards de observabilidade, adicionar painéis de monitoramento ou gerar JSON Grafana importável para serviços instrumentados com OTel. Não usar para instrumentação não-OTel, monitoramento apenas de infraestrutura ou configuração/setup do Coralogix.
---

# Gerador de Dashboards Grafana para OpenTelemetry

Gerar arquivos JSON de dashboards Grafana completos e importáveis, analisando a instrumentação OpenTelemetry do codebase (métricas, traces, logs) e produzindo painéis que funcionam tanto no ambiente local (otel-lgtm + Prometheus/Loki/Tempo) quanto em produção (Coralogix — ambiente já configurado e operacional).

## Passo 1: Descobrir Instrumentação OpenTelemetry

1. Buscar no codebase todos os registros de instrumentos de métricas OpenTelemetry. Fazer grep pelos padrões:
   - `meter.Int64Counter`, `meter.Float64Counter`
   - `meter.Int64Histogram`, `meter.Float64Histogram`
   - `meter.Int64Gauge`, `meter.Float64Gauge`
   - `meter.Int64UpDownCounter`, `meter.Float64UpDownCounter`
   - `meter.Int64ObservableCounter`, `meter.Float64ObservableCounter`
   - `meter.Int64ObservableGauge`, `meter.Float64ObservableGauge`
   - `meter.Int64ObservableUpDownCounter`, `meter.Float64ObservableUpDownCounter`
2. Para cada instrumento encontrado, extrair:
   - **Nome da métrica** (primeiro argumento string, ex.: `"http.server.request.duration"`)
   - **Tipo** (Counter, Histogram, Gauge, UpDownCounter, variantes Observable)
   - **Unidade** (de `metric.WithUnit(...)`)
   - **Descrição** (de `metric.WithDescription(...)`)
   - **Atributos** usados com a métrica (grep por `metric.WithAttributes` próximo aos locais de uso)
3. Buscar registros de tracer e criação de spans:
   - `otel.Tracer(`, `tracer.Start(`
   - Extrair nomes de spans, atributos (`attribute.String`, `attribute.Int`, etc.) e eventos (`span.AddEvent`)
4. Buscar integrações de bridge de logs:
   - `slog`, `logrus`, `zap` com pacotes bridge OTel
   - `otelslog`, `otellogrus`, `otelzap`
5. Buscar imports de convenções semânticas: uso de `semconv.` para identificar chaves de atributos padrão.
6. Buscar instrumentação de middleware HTTP/gRPC:
   - `otelhttp`, `otelgrpc`, `otelgin`, `otelecho`, `otelmux`, `otelfiber`, `otelchi`
   - Estes geram automaticamente métricas padrão como `http.server.request.duration`, `http.server.active_requests`, `rpc.server.duration`.
7. Buscar instrumentação de banco de dados:
   - `otelsql`, `otelgorm`, `otelmongo`, `otelredis`, `pgx` com tracing
8. Registrar todas as descobertas em um inventário estruturado. Ler `references/otel-semantic-metrics.md` para cruzar métricas descobertas com convenções semânticas conhecidas e identificar métricas padrão geradas automaticamente pelo middleware.

## Passo 2: Classificar e Agrupar Métricas em Seções do Dashboard

1. Agrupar métricas descobertas em seções lógicas do dashboard. Ler `references/dashboard-sections.md` para a taxonomia padrão de seções.
2. Para cada seção, determinar:
   - Quais métricas mapeiam para quais painéis
   - Tipo de visualização apropriado (time series, stat, gauge, bar gauge, table, heatmap, logs, traces)
   - Queries PromQL apropriadas para datasource Prometheus
   - Queries LogQL apropriadas para datasource Loki
   - Queries TraceQL apropriadas para datasource Tempo
3. Para cada tipo de métrica, aplicar o padrão de query correto:
   - **Counter** → `rate(metric_name_total[5m])` ou `increase(metric_name_total[$__range])`
   - **Histogram** → `histogram_quantile(0.95, rate(metric_name_bucket[5m]))` para percentis; `rate(metric_name_sum[5m]) / rate(metric_name_count[5m])` para média
   - **Gauge** → `metric_name` diretamente
   - **UpDownCounter** → `metric_name` diretamente (mostra valor atual)
4. Aplicar filtros de label usando atributos descobertos. Padrões comuns:
   - `{service_name="$service"}` para filtro por serviço
   - `{http_method="$method"}` para breakdown por método HTTP
   - `{http_status_code=~"5.."}` para filtro de erros

## Passo 3: Gerar JSON dos Dashboards

1. Ler `assets/dashboard-template.json` para a estrutura base do dashboard com variáveis de template.
2. Para cada seção do Passo 2, gerar painéis usando os padrões em `references/panel-patterns.md`.
3. Aplicar estas convenções obrigatórias de dashboard:
   - **Variáveis de datasource**: Usar variáveis de template `${DS_PROMETHEUS}`, `${DS_LOKI}`, `${DS_TEMPO}` para que os dashboards funcionem com qualquer nome de datasource.
   - **Filtro de serviço**: Incluir variável de template `$service` filtrando pelo label `service_name`.
   - **Filtro de ambiente**: Incluir variável de template `$environment` filtrando pelo label `deployment_environment`.
   - **Intervalo de tempo**: Todas as queries devem usar `$__rate_interval` para janelas de rate() (não intervalos hardcoded).
   - **IDs dos painéis**: Auto-incrementar começando de 1.
   - **Posições no grid**: Organizar painéis em grid de 24 colunas. Larguras padrão: stat=6, timeseries=12, table=24, heatmap=12.
   - **Painéis de linha**: Usar painéis de linha colapsáveis para separar seções.
4. Gerar os seguintes dashboards padrão (um arquivo JSON cada):

### Dashboard 1: Visão Geral do Serviço (`service-overview.json`)
Ler `references/dashboard-sections.md` Seção 1 para especificações dos painéis.

### Dashboard 2: Performance HTTP/gRPC (`http-grpc-performance.json`)
Ler `references/dashboard-sections.md` Seção 2 para especificações dos painéis.

### Dashboard 3: Banco de Dados e Dependências Externas (`database-dependencies.json`)
Ler `references/dashboard-sections.md` Seção 3 para especificações dos painéis.

### Dashboard 4: Explorador de Logs (`logs-explorer.json`)
Ler `references/dashboard-sections.md` Seção 4 para especificações dos painéis.

### Dashboard 5: Explorador de Traces (`traces-explorer.json`)
Ler `references/dashboard-sections.md` Seção 5 para especificações dos painéis.

### Dashboard 6: Métricas Customizadas da Aplicação (`custom-app-metrics.json`)
- Gerar painéis para cada métrica customizada (fora das convenções semânticas) descoberta no Passo 1.
- Usar visualização apropriada por tipo de métrica.

## Passo 4: Gerar docker-compose e Configuração de Provisioning

1. Ler `references/local-dev-setup.md` para o template docker-compose com grafana/otel-lgtm.
2. Gerar um `docker-compose.observability.yml` com:
   - Container `grafana/otel-lgtm:0.7.5`
   - Volume mounts para arquivos JSON de dashboard e configuração de provisioning
   - Mapeamento correto de portas (3000, 4317, 4318, 9090)
   - Volume de persistência de dados
3. Gerar um `provisioning/dashboards/dashboards.yaml` para auto-provisioning do Grafana.
4. Gerar a estrutura de diretórios `provisioning/dashboards/` contendo todos os arquivos JSON de dashboard.

## Passo 5: Garantir Compatibilidade com Coralogix

O ambiente Coralogix já está configurado e operacional (collector, API keys, pipelines). Esta etapa garante apenas que os dashboards gerados sejam importáveis sem ajustes.

1. Ler `references/coralogix-config.md` para entender o mapeamento de datasources do Coralogix.
2. Verificar que todas as queries usam apenas PromQL/LogQL/TraceQL padrão — sem sintaxe proprietária. O Coralogix suporta queries compatíveis com Prometheus em métricas ingeridas via OTLP.
3. Adicionar variáveis de template opcionais `$cx_application_name` e `$cx_subsystem_name` nos dashboards para permitir filtro adicional no ambiente Coralogix, sem quebrar o ambiente local.
4. Verificar que os nomes de métricas seguem o mapeamento OTel → Prometheus (pontos → underscores, sufixos de unidade) conforme `references/otel-semantic-metrics.md`, pois o Coralogix preserva essa conversão.
5. Garantir que os links entre dashboards usam caminhos relativos (`/d/uid/slug`) que funcionam tanto no Grafana local quanto no Grafana hospedado do Coralogix.

## Passo 6: Validar JSON dos Dashboards

1. Executar `python3 scripts/validate-dashboard.py --file <caminho>` para cada arquivo JSON de dashboard gerado.
2. Se a validação falhar, ler a saída stderr, corrigir o JSON e revalidar.
3. Verificar que todas as referências de datasource nos painéis usam variáveis de template (`${DS_PROMETHEUS}`, etc.).

## Passo 7: Escrever Arquivos de Saída

1. Escrever todos os arquivos JSON de dashboard em `observability/dashboards/`.
2. Escrever arquivo docker-compose em `observability/docker-compose.observability.yml`.
3. Escrever configurações de provisioning em `observability/provisioning/`.
4. Fornecer um resumo com:
   - Total de dashboards gerados
   - Total de painéis por dashboard
   - Métricas cobertas vs. descobertas
   - Quaisquer métricas sem cobertura de dashboard (avisar)

## Tratamento de Erros

* **Nenhuma instrumentação OTel encontrada:** Avisar o usuário que nenhuma instrumentação OpenTelemetry foi detectada. Sugerir verificar `go.opentelemetry.io/otel` no `go.mod`. Oferecer gerar dashboards baseados apenas em métricas HTTP/runtime padrão.
* **Tipo de métrica desconhecido:** Se um padrão de registro de métrica não for reconhecido, registrar como aviso e pular. Incluir a descoberta no resumo.
* **Falha na validação do dashboard:** Ler stderr de `scripts/validate-dashboard.py`, corrigir o erro JSON específico e revalidar. Problemas comuns: IDs de painéis duplicados, variáveis de datasource ausentes, posições de grid inválidas.
* **Nenhuma convenção semântica encontrada:** Gerar dashboards usando nomes brutos das métricas. Avisar que nomes de métricas podem diferir entre ambientes se convenções semânticas não forem usadas.
* **Bibliotecas de instrumentação mistas:** Se tanto OTel manual quanto middleware de auto-instrumentação forem encontrados, mesclar métricas e deduplicar. Preferir nomes de convenção semântica.
