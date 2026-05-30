# Concorrencia .NET / C#

<!-- TL;DR
Concorrencia em .NET: regras de async/await, propagacao de CancellationToken, Channel<T> para producer-consumer, Parallel.ForEachAsync e IAsyncEnumerable.
Keywords: async, await, cancellationtoken, channel, valuetask, parallel.foreachasync, task.whenall, iasyncenumerable, configureawait
Load complete when: a tarefa usa async/await, CancellationToken, Channel, Parallel ou streaming assincrono.
-->

## Objetivo

Definir as regras de concorrencia e assincronismo seguro em .NET.

## Diretrizes

### async/await — regras fundamentais
- Sempre propagar `CancellationToken` ate a borda de IO — nao engolir cancelamento.
- Nunca usar `.Result` ou `.Wait()` em codigo assincrono — causa deadlock em contextos com
  `SynchronizationContext` e bloqueia o thread pool.
- `ConfigureAwait(false)` em bibliotecas; desnecessario em codigo de aplicacao ASP.NET Core (sem
  `SynchronizationContext`).
- `ValueTask<T>` apenas quando o caminho sincrono for o comum (ex: cache hit) — evitar em APIs gerais.

### CancellationToken em fronteiras
- Toda operacao de IO (EF Core, HttpClient, MassTransit) aceita `CancellationToken`.
- Passar o token recebido do framework — nao criar `CancellationTokenSource` sem justificativa.
- `cancellationToken.ThrowIfCancellationRequested()` como checkpoint em loops longos.

### Channel<T> (producer-consumer)
- `Channel.CreateBounded<T>(capacity)` para limitar memoria e aplicar backpressure.
- `Channel.CreateUnbounded<T>()` apenas quando o producer tiver taxa controlada comprovada.
- Completar o writer com `channel.Writer.Complete()` para sinalizar fim de producao.

### Parallel e Task.WhenAll
- `Parallel.ForEachAsync` (.NET 6+) com `MaxDegreeOfParallelism` explicito para fan-out controlado.
- `Task.WhenAll` quando todas as tarefas devem completar; `Task.WhenAny` para first-winner.
- Evitar `Task.Run` em ASP.NET Core — o thread pool ja e gerenciado pelo runtime.

### IAsyncEnumerable<T>
- Usar para streaming (queries paginadas, arquivos grandes) sem buffering completo em memoria.
- `await foreach (var item in source.WithCancellation(ct))`.

## Riscos Comuns
- `async void` (exceto event handlers) impede captura de excecao e await.
- Fan-out ilimitado satura o thread pool e dependencias externas.
- Esquecer `Writer.Complete()` deixa o consumidor aguardando indefinidamente.

## Proibido
- `.Result` ou `.Wait()` em codigo ASP.NET Core.
- `Thread.Sleep` para sincronizacao.
- `new Thread()` em aplicacoes ASP.NET Core.
- Fan-out ilimitado sem grau de paralelismo.
- Ignorar `OperationCanceledException` sem relançar ou tratar.
