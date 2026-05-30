# Testes .NET / C#

<!-- TL;DR
Estrategia de testes em .NET: xUnit (Fact/Theory), FluentAssertions, NSubstitute, Bogus, Testcontainers e WebApplicationFactory para integration tests.
Keywords: xunit, theory, inlinedata, memberdata, fluentassertions, nsubstitute, bogus, testcontainers, WebApplicationFactory, cobertura
Load complete when: a tarefa envolve escrever ou alterar testes unitarios ou de integracao em .NET.
-->

## Objetivo

Definir a estrategia de testes determinísticos e isolados para codigo .NET, do unitario ao
integration test com containers efemeros.

## Diretrizes

### Unit Tests
- xUnit como framework padrao: `[Fact]` para caso unico; `[Theory]` + `[InlineData]` / `[MemberData]`
  / `[ClassData]` para variacoes — equivalente ao table-driven.
- FluentAssertions para asserções legiveis: `result.Should().Be(expected)`,
  `act.Should().Throw<DomainException>()`.
- NSubstitute para mocks: `Substitute.For<IOrderRepository>()`. Preferir sobre Moq pela API mais limpa.
- Bogus (`Faker<T>`) para gerar dados de teste com builders tipados — eliminar dados literais magicos.
- Nomenclatura: `MethodName_Scenario_ExpectedResult`
  (ex: `Confirm_OrderAlreadyShipped_ThrowsDomainException`).
- Testes determinísticos: sem `Thread.Sleep`/`Task.Delay` para sincronizar, sem estado global, sem
  dependencia de ordem de execucao. Injetar `TimeProvider` (ou `FakeTimeProvider` de
  `Microsoft.Extensions.TimeProvider.Testing`) para tempo controlado.
- Isolar dependencias com `Substitute.For<>` (NSubstitute) ou fakes escritos a mao — nunca usar banco
  real em unit test.

### Integration Tests
- Testcontainers.NET para containers efemeros (Postgres, Redis, RabbitMQ, Kafka).
- `WebApplicationFactory<TEntryPoint>` para exercitar a pipeline HTTP completa.
- `IAsyncLifetime` para iniciar/parar containers; `IClassFixture<T>` para compartilhar entre testes
  da mesma classe.
- Separar por projeto (`*.IntegrationTests.csproj`) ou categoria via `[Trait("Category","Integration")]`.

Exemplo com Testcontainers:
```csharp
public sealed class OrderRepositoryTests : IAsyncLifetime
{
    private readonly PostgreSqlContainer _postgres = new PostgreSqlBuilder()
        .WithImage("postgres:17-alpine")
        .Build();

    public Task InitializeAsync() => _postgres.StartAsync();
    public Task DisposeAsync() => _postgres.DisposeAsync().AsTask();

    [Fact]
    public async Task SaveAsync_ValidOrder_PersistsToDatabase()
    {
        await using var context = CreateDbContext(_postgres.GetConnectionString());
        var repository = new OrderRepository(context);

        var order = Order.Create(customerId: Guid.NewGuid());
        await repository.AddAsync(order, TestContext.Current.CancellationToken);
        await context.SaveChangesAsync(TestContext.Current.CancellationToken);

        var loaded = await repository.GetByIdAsync(order.Id, TestContext.Current.CancellationToken);
        loaded.Should().NotBeNull();
    }
}
```

### Cobertura
- `dotnet test --collect:"XPlat Code Coverage"` gera relatorio Cobertura via Coverlet.
- Focar cobertura em regras de dominio e use cases; nao perseguir 100% em wiring de DI.

## Riscos Comuns
- Esquecer de propagar `CancellationToken` em metodos assincronos de teste.
- Container compartilhado entre classes sem isolamento de dados gera flakiness.
- Mock que retorna valor estatico sem verificar interacao quando o comportamento importa.

## Proibido
- `Thread.Sleep` ou `Task.Delay` para sincronizacao em testes.
- Teste dependendo de servico externo real (rede, banco compartilhado).
- Banco de dados real em unit test.
- Ignorar `CancellationToken` em testes assincronos.
