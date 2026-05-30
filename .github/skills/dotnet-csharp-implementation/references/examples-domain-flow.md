# Exemplo: Fluxo de Dominio End-to-End (.NET / C#)

<!-- TL;DR
Esqueleto completo do fluxo CreateOrder em Clean Architecture: Domain -> Application (Command/Handler/Validator) -> Infrastructure (Repository) -> Api (Minimal API) com testes.
Keywords: clean architecture, createorder, command, handler, validator, repository, minimal api, end-to-end, exemplo
Load complete when: a tarefa precisa de um esqueleto concreto de fluxo end-to-end (Entity -> Command -> Handler -> Endpoint -> Teste).
-->

## Objetivo

Demonstrar um fluxo `CreateOrder` completo em Clean Architecture. Adaptar ao contexto real — nao
copiar literalmente.

## Estrutura
```
Domain/
  Order.cs            — entidade com factory static e domain events
  IOrderRepository.cs — interface de porta (sem EF Core)
Application/
  CreateOrder/
    CreateOrderCommand.cs   — record IRequest<OrderId>
    CreateOrderHandler.cs   — IRequestHandler com Unit of Work
    CreateOrderValidator.cs — FluentValidation AbstractValidator
Infrastructure/
  Persistence/
    OrderRepository.cs      — implementacao EF Core
    AppDbContext.cs
Api/
  Endpoints/
    OrderEndpoints.cs       — MapGroup + TypedResults
```

## Domain
```csharp
public sealed class Order
{
    private readonly List<OrderItem> _items = [];
    public Guid Id { get; private init; }
    public Guid CustomerId { get; private init; }
    public OrderStatus Status { get; private set; }
    public IReadOnlyList<OrderItem> Items => _items;

    private Order() { } // EF Core

    public static Order Create(Guid customerId)
    {
        if (customerId == Guid.Empty)
            throw new ArgumentException("CustomerId obrigatorio", nameof(customerId));

        return new Order { Id = Guid.NewGuid(), CustomerId = customerId, Status = OrderStatus.Draft };
    }
}

public interface IOrderRepository
{
    Task AddAsync(Order order, CancellationToken cancellationToken);
    Task<Order?> GetByIdAsync(Guid id, CancellationToken cancellationToken);
}
```

## Application
```csharp
public sealed record CreateOrderCommand(Guid CustomerId) : IRequest<Guid>;

public sealed class CreateOrderValidator : AbstractValidator<CreateOrderCommand>
{
    public CreateOrderValidator() => RuleFor(c => c.CustomerId).NotEmpty();
}

public sealed class CreateOrderHandler(IOrderRepository repository, IUnitOfWork unitOfWork)
    : IRequestHandler<CreateOrderCommand, Guid>
{
    public async Task<Guid> Handle(CreateOrderCommand command, CancellationToken cancellationToken)
    {
        var order = Order.Create(command.CustomerId);
        await repository.AddAsync(order, cancellationToken);
        await unitOfWork.SaveChangesAsync(cancellationToken);
        return order.Id;
    }
}
```
- A validacao roda via `ValidationBehavior` no pipeline do MediatR — nao duplicar no endpoint.

## Api
```csharp
public static class OrderEndpoints
{
    public static void MapOrders(this IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/orders").WithTags("Orders");

        group.MapPost("/", async (CreateOrderCommand command, ISender sender, CancellationToken ct) =>
        {
            var id = await sender.Send(command, ct);
            return TypedResults.Created($"/orders/{id}", new { id });
        });
    }
}
```

## Testes
- Unit do handler: NSubstitute para `IOrderRepository`/`IUnitOfWork` + FluentAssertions; verificar
  `Received(1).AddAsync(...)` e `SaveChangesAsync`.
- Integration do endpoint: `WebApplicationFactory<Program>` + Testcontainers (Postgres), enviando
  POST e validando 201 com Location header.

## Proibido
- Logica de negocio no endpoint (deve delegar ao handler via `ISender`).
- `SaveChangesAsync` dentro do repository.
- Retornar entidade de dominio crua no endpoint — mapear para DTO.
