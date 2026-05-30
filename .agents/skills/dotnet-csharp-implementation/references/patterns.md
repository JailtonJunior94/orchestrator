# Patterns .NET / C#

<!-- TL;DR
Patterns nao cobertos inline no SKILL.md: Specification, Decorator com Scrutor, Strategy com keyed services, MediatR pipeline behaviors e Result pattern.
Keywords: specification, decorator, scrutor, strategy, keyed services, pipeline behavior, mediatr, result pattern
Load complete when: a tarefa envolve Specification, Decorator, Strategy, pipeline behaviors ou Result pattern.
-->

## Objetivo

Padroes adicionais para .NET. Factory, Primary Constructor, Record como Value Object e Repository ja
estao definidos inline no SKILL.md e **nao sao duplicados aqui**.

## Diretrizes

### Specification Pattern
- `ISpecification<T>` expondo `Expression<Func<T, bool>>` para queries EF Core combinaveis.
- Usar quando regras de negocio forem reutilizadas em multiplos handlers/repositorios.
- Combinar com `.And(...)`/`.Or(...)` que compoem as expressions; aplicar no repository via `Where`.

### Decorator com Scrutor
```csharp
services.AddScoped<IOrderService, OrderService>();
services.Decorate<IOrderService, LoggingOrderService>();
services.Decorate<IOrderService, CachingOrderService>();
```
- Cada decorator adiciona responsabilidade transversal (logging, cache, metricas) sem modificar o
  servico original. A ordem de `Decorate` define o aninhamento (ultimo registrado e o mais externo).

### Strategy com keyed services
- Interface + multiplas implementacoes; selecionar por chave com keyed services (.NET 8+):
  ```csharp
  services.AddKeyedScoped<IPricingStrategy, BlackFridayPricing>("black-friday");
  services.AddKeyedScoped<IPricingStrategy, DefaultPricing>("default");
  // consumo: [FromKeyedServices("default")] IPricingStrategy strategy
  ```
- Alternativa: injetar `IEnumerable<IPricingStrategy>` e selecionar por propriedade discriminadora.

### Chain of Responsibility com MediatR Pipeline Behaviors
```csharp
services.AddTransient(typeof(IPipelineBehavior<,>), typeof(ValidationBehavior<,>));
services.AddTransient(typeof(IPipelineBehavior<,>), typeof(LoggingBehavior<,>));
```
- Validacao, logging, retry e commit de Unit of Work como behaviors ortogonais ao handler.

### Result Pattern
- Usar `Result<T>` (proprio ou `OneOf<T, Error>`) em fronteiras de Application/Domain para fluxos
  esperados (not found, validacao falhou) em vez de excecoes.
- Reservar excecoes para falhas inesperadas de infraestrutura.

## Riscos Comuns
- Aplicar pattern sem fronteira real adiciona indirecao sem reduzir acoplamento.
- Decorators com ordem trocada produzem comportamento transversal aninhado incorreto.

## Proibido
- Reimplementar Factory/Repository/Primary Constructor aqui (ja inline no SKILL.md).
- Excecao para fluxo esperado quando Result expressa melhor a intencao.
- Strategy via `if/switch` gigante quando keyed services ou polimorfismo resolvem.
