# Exemplo: Testes (.NET / C#)

<!-- TL;DR
Exemplos de teste em .NET: TheoryData/MemberData (table-driven), builders com Bogus e verificacao de interacao com NSubstitute.
Keywords: theorydata, memberdata, inlinedata, bogus, faker, nsubstitute, received, fluentassertions, table-driven
Load complete when: a tarefa precisa de exemplos concretos de table-driven test, builders de dados ou verificacao de interacao.
-->

## Objetivo

Exemplos concretos de testes determinísticos. Adaptar nomes e tipos ao contexto real.

## TheoryData (table-driven)
```csharp
public static TheoryData<string, bool> EmailValidationCases => new()
{
    { "valid@example.com", true },
    { "invalid-email", false },
    { "", false },
};

[Theory, MemberData(nameof(EmailValidationCases))]
public void IsValid_Email_ReturnsExpectedResult(string email, bool expected)
{
    var result = new EmailValidator().IsValid(email);
    result.Should().Be(expected);
}
```

## Bogus para builders de teste
```csharp
private static readonly Faker<Order> OrderFaker = new Faker<Order>()
    .CustomInstantiator(f => Order.Create(customerId: f.Random.Guid()));

[Fact]
public void Create_ValidCustomer_StartsInDraft()
{
    var order = OrderFaker.Generate();
    order.Status.Should().Be(OrderStatus.Draft);
}
```
- `CustomInstantiator` respeita a factory de dominio (`Order.Create`) em vez de instanciar via reflexao.

## NSubstitute com verificacao de interacao
```csharp
[Fact]
public async Task Handle_NewOrder_PersistsAndCommits()
{
    var repository = Substitute.For<IOrderRepository>();
    var unitOfWork = Substitute.For<IUnitOfWork>();
    var handler = new CreateOrderHandler(repository, unitOfWork);

    var id = await handler.Handle(new CreateOrderCommand(Guid.NewGuid()), CancellationToken.None);

    id.Should().NotBeEmpty();
    await repository.Received(1).AddAsync(Arg.Any<Order>(), Arg.Any<CancellationToken>());
    await unitOfWork.Received(1).SaveChangesAsync(Arg.Any<CancellationToken>());
}
```
- `Received(1)` verifica a interacao quando o comportamento (persistir + commitar) importa.
- `Arg.Is<Order>(o => o.Status == OrderStatus.Confirmed)` para asserts sobre o argumento capturado.

## Proibido
- `Thread.Sleep`/`Task.Delay` para sincronizar.
- Dados literais magicos espalhados — preferir Bogus ou constantes nomeadas.
- Mock que retorna valor estatico sem verificar interacao quando ela e o objetivo do teste.
