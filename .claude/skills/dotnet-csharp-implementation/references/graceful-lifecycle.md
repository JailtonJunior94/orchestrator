# Ciclo de Vida e Shutdown Gracioso .NET / C#

<!-- TL;DR
Ciclo de vida em .NET: IHostedService/BackgroundService, IHostApplicationLifetime, ShutdownTimeout, drain de conexoes HTTP e ordem de encerramento.
Keywords: generic host, ihostedservice, backgroundservice, executeasync, stoppingtoken, ihostapplicationlifetime, shutdowntimeout, sigterm, drain
Load complete when: a tarefa envolve background services, inicializacao ordenada, shutdown gracioso ou drain de conexoes.
-->

## Objetivo

Definir como modelar inicializacao, background services e encerramento gracioso no Generic Host.

## Diretrizes

### Generic Host e IHostedService
- `IHostedService.StartAsync` / `StopAsync` para servicos com controle explicito de ciclo de vida.
- `BackgroundService` como classe base para loops continuos — implementar
  `ExecuteAsync(CancellationToken stoppingToken)` e propagar `stoppingToken` em toda operacao de IO:
  ```csharp
  protected override async Task ExecuteAsync(CancellationToken stoppingToken)
  {
      using var timer = new PeriodicTimer(TimeSpan.FromSeconds(30));
      while (await timer.WaitForNextTickAsync(stoppingToken))
      {
          await ProcessBatchAsync(stoppingToken);
      }
  }
  ```

### Ordem de inicializacao
- `IHostApplicationLifetime.ApplicationStarted` para acoes pos-inicializacao (ex: ativar readiness probe).
- `WebApplication.RunAsync()` bloqueia ate SIGTERM/SIGINT — preferir sobre `Run()` quando o token importa.

### Shutdown gracioso
```csharp
builder.Services.Configure<HostOptions>(o =>
    o.ShutdownTimeout = TimeSpan.FromSeconds(15)); // default 5s e insuficiente para workers
```
- O `ShutdownTimeout` deve ser menor que `terminationGracePeriodSeconds` do Kubernetes.

### Drain de conexoes HTTP
- `IHostApplicationLifetime.ApplicationStopping` para sinalizar que o processo nao aceita novos requests.
- Kestrel `Limits.KeepAliveTimeout` ajuda a drenar conexoes em aberto.

### Ordem de encerramento
- Inversao da ordem de inicializacao: HTTP server -> consumers -> database -> flush de telemetria.
- `IAsyncDisposable` em servicos com recursos — garantir `await DisposeAsync()` no shutdown.

## Riscos Comuns
- `BackgroundService` que ignora `stoppingToken` nao encerra dentro do timeout.
- `ShutdownTimeout` no default (5s) corta processamento longo no meio.

## Proibido
- `Environment.Exit()` fora de `Main` sem flush de recursos.
- `BackgroundService` que ignora `stoppingToken`.
- `IHostedService` que nao completa `StopAsync` antes do timeout.
- Bloquear `StopAsync` com trabalho sincrono longo sem respeitar o token.
