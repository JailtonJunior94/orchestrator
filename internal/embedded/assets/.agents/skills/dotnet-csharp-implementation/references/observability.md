# Observabilidade .NET / C#

<!-- TL;DR
Observabilidade em .NET: logging estruturado com ILogger e LoggerMessage source generator, OpenTelemetry (tracing/metrics), Activity, Meter e health checks.
Keywords: ilogger, loggermessage, serilog, opentelemetry, otlp, activity, activitysource, meter, counter, histogram, health checks
Load complete when: a tarefa envolve logging, tracing, metricas ou health checks.
-->

## Objetivo

Definir as praticas de logging, tracing, metricas e health checks para servicos .NET.

## Diretrizes

### Logging estruturado
- `ILogger<T>` como abstracao padrao; Serilog como provider de producao com sinks OTLP ou Console JSON.
- Usar o source generator `[LoggerMessage]` para hot paths (evita boxing e alocacao):
  ```csharp
  internal static partial class Log
  {
      [LoggerMessage(Level = LogLevel.Information, Message = "Pedido {OrderId} confirmado")]
      public static partial void OrderConfirmed(this ILogger logger, Guid orderId);
  }
  ```
- Campos minimos: `level`, `message`, `TraceId`, `SpanId`, `exception`.
- Nao logar PII, tokens, senhas ou payloads completos de request.

### OpenTelemetry .NET
```csharp
builder.Services.AddOpenTelemetry()
    .WithTracing(tracing => tracing
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddEntityFrameworkCoreInstrumentation()
        .AddOtlpExporter())
    .WithMetrics(metrics => metrics
        .AddAspNetCoreInstrumentation()
        .AddHttpClientInstrumentation()
        .AddRuntimeInstrumentation()
        .AddOtlpExporter());
```

### Tracing manual (Activity)
- `ActivitySource` com nome versionado para spans customizados:
  `private static readonly ActivitySource Source = new("Orders", "1.0.0");`.
- `using var activity = Source.StartActivity("ConfirmOrder");` e enriquecer com `SetTag`.
- Nomear atividades pelo papel da operacao, nao pelo nome do metodo.

### Metricas customizadas
- `Meter` + `Counter<T>`, `Histogram<T>`, `ObservableGauge<T>`.
- Labels com cardinalidade controlada — nunca ID de usuario ou request ID como label.

### Health checks
```csharp
builder.Services.AddHealthChecks()
    .AddDbContextCheck<AppDbContext>("database");

app.MapHealthChecks("/health/live", new() { Predicate = _ => false });
app.MapHealthChecks("/health/ready");
```

## Riscos Comuns
- Label de metrica derivado de input de usuario explode cardinalidade.
- Logar excecao completa com stack trace na resposta ao cliente.
- Esquecer de propagar `Activity.Current` em chamadas assincronas quebra a correlacao de trace.

## Proibido
- `Console.WriteLine` ou `Debug.WriteLine` em codigo de producao.
- Logar PII, segredos ou payload completo.
- Metrica com label de alta cardinalidade sem sanitizacao.
