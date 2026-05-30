# Exemplo: Infraestrutura (.NET / C#)

<!-- TL;DR
Exemplos de infraestrutura em .NET: graceful shutdown com BackgroundService, cursor-based pagination com EF Core, versionamento de API e outbox processor com PeriodicTimer.
Keywords: graceful shutdown, backgroundservice, periodictimer, cursor pagination, ef core, api versioning, asp.versioning, outbox processor
Load complete when: a tarefa precisa de exemplo de graceful shutdown, cursor pagination, versionamento de API ou outbox processor.
-->

## Objetivo

Exemplos de codigo de infraestrutura recorrente. Adaptar ao contexto real.

## Graceful shutdown com BackgroundService
```csharp
public sealed class OutboxProcessor(IServiceScopeFactory scopeFactory, ILogger<OutboxProcessor> logger)
    : BackgroundService
{
    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        using var timer = new PeriodicTimer(TimeSpan.FromSeconds(5));
        while (await timer.WaitForNextTickAsync(stoppingToken))
        {
            try
            {
                await using var scope = scopeFactory.CreateAsyncScope();
                var dispatcher = scope.ServiceProvider.GetRequiredService<IOutboxDispatcher>();
                await dispatcher.DispatchPendingAsync(stoppingToken);
            }
            catch (OperationCanceledException) when (stoppingToken.IsCancellationRequested)
            {
                break; // shutdown solicitado — encerrar limpo
            }
            catch (Exception ex)
            {
                logger.LogError(ex, "Falha ao processar outbox; retentando no proximo tick");
            }
        }
    }
}
```
- `IServiceScopeFactory` cria escopo por iteracao para resolver dependencias `Scoped` (ex: `DbContext`).

## Cursor-based pagination com EF Core
```csharp
public async Task<IReadOnlyList<Order>> GetPageAsync(Guid? cursor, int pageSize, CancellationToken ct)
{
    var query = _context.Orders.AsNoTracking().OrderBy(o => o.Id).AsQueryable();
    if (cursor is { } last)
        query = query.Where(o => o.Id > last);

    return await query.Take(pageSize).ToListAsync(ct);
}
```
- Cursor estavel (chave ordenavel) evita os problemas de skip/take em paginas profundas.

## Versionamento de API (Asp.Versioning)
```csharp
var versionSet = app.NewApiVersionSet()
    .HasApiVersion(new ApiVersion(1.0))
    .HasApiVersion(new ApiVersion(2.0))
    .Build();

app.MapGroup("/v{version:apiVersion}/orders")
   .WithApiVersionSet(versionSet)
   .MapOrders();
```

## Outbox processor
- Implementado como `BackgroundService` com `PeriodicTimer` (exemplo acima), retry via Polly em falhas
  transitorias do broker, e DLQ para mensagens que excederem o limite de tentativas.

## Proibido
- `BackgroundService` que ignora `stoppingToken`.
- Engolir `Exception` sem log e sem politica de retry.
- Offset pagination (`Skip(n)`) em datasets grandes quando cursor e viavel.
