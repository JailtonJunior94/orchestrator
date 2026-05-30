# Resiliencia .NET / C#

<!-- TL;DR
Resiliencia em .NET: Polly v8 via Microsoft.Extensions.Resilience, pipelines de retry/circuit breaker/timeout, AddStandardResilienceHandler para HttpClient, hedging e TimeProvider.
Keywords: polly, resilience, retry, circuit breaker, timeout, hedging, AddResiliencePipeline, AddStandardResilienceHandler, IHttpClientFactory, jitter
Load complete when: a tarefa envolve retries, circuit breakers, timeouts, hedging ou protecao contra falhas transitorias.
-->

## Objetivo

Definir como aplicar resiliencia a chamadas externas usando Polly v8 integrado ao
`Microsoft.Extensions.Resilience`.

## Diretrizes

### Pipeline de resiliencia (Polly v8)
```csharp
builder.Services.AddResiliencePipeline("external-api", builder =>
{
    builder
        .AddRetry(new RetryStrategyOptions
        {
            MaxRetryAttempts = 3,
            BackoffType = DelayBackoffType.Exponential,
            UseJitter = true,
            ShouldHandle = new PredicateBuilder().Handle<HttpRequestException>()
        })
        .AddCircuitBreaker(new CircuitBreakerStrategyOptions
        {
            SamplingDuration = TimeSpan.FromSeconds(10),
            MinimumThroughput = 10,
            FailureRatio = 0.5,
            BreakDuration = TimeSpan.FromSeconds(30)
        })
        .AddTimeout(TimeSpan.FromSeconds(5));
});
```
Consumir via `ResiliencePipelineProvider<string>.GetPipeline("external-api")` e
`pipeline.ExecuteAsync(async ct => ..., cancellationToken)`.

### HttpClient com resiliencia
- Usar `AddStandardResilienceHandler()` de `Microsoft.Extensions.Http.Resilience` como baseline:
  ```csharp
  builder.Services.AddHttpClient<IPaymentClient, PaymentClient>()
      .AddStandardResilienceHandler();
  ```
- Named/typed clients via `IHttpClientFactory` — nunca instanciar `HttpClient` diretamente.

### Hedging
- `AddHedging` (Polly v8) para disparar requests paralelos e usar o primeiro a responder em endpoints
  idempotentes e sensiveis a latencia.

### TimeProvider
- Usar a abstracao `TimeProvider` para timeouts e backoff testaveis, em vez de `DateTime.UtcNow`
  ou `Stopwatch` hardcoded.

## Riscos Comuns
- Retentar erros 4xx (nao transitorios) amplifica falha sem resolver a causa.
- Backoff sem jitter gera thundering herd quando muitos clientes retentam em sincronia.
- Circuit breaker sem `MinimumThroughput` adequado abre cedo demais sob baixo trafego.

## Proibido
- `HttpClient` instanciado com `new` — causa socket exhaustion.
- Retry infinito ou sem limite de tentativas.
- Retry para erros 4xx.
- Ignorar `CancellationToken` no pipeline de resiliencia.
