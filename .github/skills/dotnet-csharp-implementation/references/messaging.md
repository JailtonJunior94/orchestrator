# Mensageria .NET / C#

<!-- TL;DR
Mensageria em .NET: MassTransit v8 com consumers idempotentes, EF Outbox pattern, dead-letter queues, domain events in-process via MediatR e contratos versionados.
Keywords: masstransit, IConsumer, outbox, dead-letter, fault consumer, INotification, mediatr, idempotente, saga, correlationid
Load complete when: a tarefa envolve producao/consumo de mensagens, eventos, filas, outbox ou idempotencia de consumidores.
-->

## Objetivo

Definir as praticas de mensageria assincrona e eventos de dominio em servicos .NET.

## Diretrizes

### MassTransit v8
- `AddMassTransit()` com transporte configuravel (RabbitMQ, Azure Service Bus, Kafka; In-Memory em testes).
- Consumers implementam `IConsumer<TMessage>` e recebem `ConsumeContext<TMessage>` com
  `context.CancellationToken`.
- Consumers devem ser **idempotentes** — a garantia padrao e at-least-once delivery.

### Outbox Pattern
- `AddEntityFrameworkOutbox<TDbContext>()` para publicar mensagens dentro da transacao do banco
  (commit atomico de dados + mensagem).
- O delivery service da MassTransit entrega ao broker apos o commit, com retry e DLQ.

### Dead-Letter Queue
- `IConsumer<Fault<TMessage>>` para mensagens que falharam apos N tentativas.
- Logar contexto suficiente (CorrelationId, payload resumido) para diagnostico sem reprocessamento manual.

### Domain Events com MediatR
- `INotification` + `INotificationHandler<T>` para eventos de dominio in-process.
- Publicar `INotification` dentro do handler de Command apos o commit.
- Nao confundir domain events in-process com integration events via broker.

### Schema e contratos
- Contratos de mensagem em projeto compartilhado (`Contracts/`) ou pacote NuGet versionado.
- Mudancas backward-compatible: adicionar campos opcionais; nunca remover ou renomear.
- Namespace/`MessageUrn` explicito para evitar colisao de tipo em filas/topicos.

## Riscos Comuns
- Publicar evento antes do commit gera mensagem para estado que nao persistiu (resolver com outbox).
- Consumer sem idempotencia processa a mesma mensagem duas vezes sob at-least-once.

## Proibido
- Publicar evento antes do commit sem outbox pattern.
- Consumer que engole excecao e confirma a mensagem (ack sem processar).
- Mensagem sem `CorrelationId` ou contexto de trace.
