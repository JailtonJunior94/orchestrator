# Prompt: Criar Skill `dotnet-csharp-implementation`

> **Tipo:** Prompt enriquecido — gerado via `prompt-enricher`
> **Data:** 2026-05-29
> **Status:** Pronto para execução
> **Destino:** `.agents/skills/dotnet-csharp-implementation/`

---

## Prompt Original

> "Analise TODAS as skills de go-implementation, node-implementation e python-implementation,
> e faça uma para uso do .NET C# na última versão, utilizando os recursos mais recentes.
> Quero robustez, production-ready/production-proof sem falso positivo."

---

## Análise do Prompt Original

| Dimensão           | Observação                                                                                        |
|--------------------|---------------------------------------------------------------------------------------------------|
| Intenção principal | Criar skill `dotnet-csharp-implementation` com estrutura idêntica às skills existentes           |
| Versão alvo        | .NET 9 (último LTS estável) + C# 13 — verificar se .NET 10 já é GA no momento da execução       |
| Ambiguidade 1      | "últimos recursos" — resolvido: C# 13 / .NET 9 estáveis; incluir preview apenas se GA            |
| Ambiguidade 2      | "sem falso positivo" — interpretado como: sem APIs fictícias, tudo validável contra docs oficiais |
| Restrição de saída | Estrutura espelhada: `SKILL.md` + `references/*.md` (mesma granularidade das skills existentes)  |
| Escopo             | Não implementar a skill aqui — este arquivo é apenas o prompt enriquecido                        |

---

## Prompt Enriquecido

### Contexto e Objetivo

Você é um agente especializado em Go, Node/TypeScript, Python e agora **.NET / C#**.

Sua tarefa é **criar a skill `dotnet-csharp-implementation`** dentro de `.agents/skills/` deste
repositório, seguindo exatamente a mesma estrutura e profundidade das skills irmãs:

- `.agents/skills/go-implementation/` (referência principal de padrões inline + lazy-load)
- `.agents/skills/node-implementation/` (referência de convenções e DI)
- `.agents/skills/python-implementation/` (referência de economy-of-context)

A skill deve ser **production-ready** e **production-proof**, o que significa:

- Nenhum trecho de código, API, namespace ou NuGet Package fictício.
- Todo exemplo deve ser verificável contra a documentação oficial do .NET 9 / C# 13.
- Padrões inline devem cobrir os casos de uso mais frequentes sem carregar arquivo extra.
- Referências devem ter threshold de carregamento claro (TL;DR + Keywords + Load condition).
- Sem duplicação entre o que está inline no SKILL.md e o que está nos references.

---

### Versão e Ferramentas Alvo

| Componente              | Versão mínima          | Notas                                              |
|-------------------------|------------------------|----------------------------------------------------|
| .NET SDK                | 9.0 (LTS)              | Verificar se .NET 10 é GA; adotar o mais recente   |
| C#                      | 13                     | Primary constructors, params collections, lock obj |
| xUnit                   | 2.9+                   | Framework de teste padrão                          |
| FluentAssertions        | 7.x                    | Asserções legíveis sem verbose                     |
| NSubstitute             | 5.x                    | Mocking — preferido sobre Moq por API mais limpa   |
| Bogus                   | 35.x                   | Geração de dados de teste (Faker<T>)               |
| Testcontainers.NET      | 3.x                    | Containers efêmeros em integration tests           |
| FluentValidation        | 12.x                   | Validação de input com regras fluentes             |
| Polly                   | 8.x (via Microsoft.Extensions.Resilience) | Retry, circuit breaker, hedging   |
| OpenTelemetry .NET      | 1.9+                   | Tracing, metrics, logging via OTLP                 |
| Serilog                 | 4.x                    | Logging estruturado — alternativa a ILogger direto |
| EF Core                 | 9.x                    | ORM padrão; Dapper como alternativa para queries   |
| MediatR                 | 12.x                   | CQRS — Commands/Queries/Notifications              |
| MassTransit             | 8.x                    | Messaging — RabbitMQ, Azure Service Bus, Kafka     |
| Carter / FastEndpoints  | Última estável         | Alternativas Minimal API para organização modular  |
| Scrutor                 | 5.x                    | Decoradores e scanning de assembly no DI           |

**Documentações oficiais a consultar antes de escrever qualquer referência:**
- https://learn.microsoft.com/dotnet/core/whats-new/dotnet-9/overview
- https://learn.microsoft.com/dotnet/csharp/whats-new/csharp-13
- https://learn.microsoft.com/aspnet/core/fundamentals/minimal-apis
- https://learn.microsoft.com/dotnet/core/extensions/generic-host
- https://learn.microsoft.com/dotnet/core/diagnostics/observability-with-otel
- https://learn.microsoft.com/ef/core/what-is-new/ef-core-9.0/whatsnew

---

### Arquivos a Criar

#### Estrutura de diretório esperada

```
.agents/skills/dotnet-csharp-implementation/
├── SKILL.md
└── references/
    ├── architecture.md
    ├── conventions.md
    ├── testing.md
    ├── api.md
    ├── persistence.md
    ├── configuration.md
    ├── resilience.md
    ├── observability.md
    ├── security.md
    ├── messaging.md
    ├── concurrency.md
    ├── graceful-lifecycle.md
    ├── patterns.md
    ├── build.md
    ├── examples-domain-flow.md
    ├── examples-testing.md
    └── examples-infrastructure.md
```

---

### Especificação de Cada Arquivo

#### `SKILL.md`

Seguir exatamente o layout do `go-implementation/SKILL.md`:

```yaml
---
name: dotnet-csharp-implementation
version: 1.0.0
description: >
  Implementa alterações em código .NET/C# usando governança base, arquitetura, estilo,
  testes e padrões recorrentes. Use quando a tarefa exigir adicionar, corrigir, refatorar
  ou validar código C# incluindo Minimal APIs, Generic Host, EF Core, MediatR e validação
  da stack. Não use para tarefas sem código .NET/C#, documentação geral ou triagem sem
  alteração.
---
```

**Etapa 1 — Carregar base obrigatória:**
1. Confirmar contrato de carga base do `AGENTS.md`.
2. Ler `references/architecture.md`.
3. Ler o arquivo `.csproj` ou `Directory.Build.props` para identificar `<TargetFramework>` e dependências.
4. Executar `dotnet --version` para confirmar SDK disponível.

**Padrões inline obrigatórios (evitar carregar `patterns.md` para estes):**
- **Factory / Static Factory:** Usar `static T Create(args)` em records e classes de domínio quando
  a construção exigir invariantes. Retornar `Result<T>` em vez de lançar exceção em fronteiras públicas.
- **Primary Constructor (C# 12+):** Usar para injeção de dependência em serviços, repositories e
  handlers quando não houver lógica no construtor além de atribuição.
  Exemplo: `public class OrderService(IOrderRepository repo, IUnitOfWork uow) { ... }`
- **Record como Value Object:** Usar `record` ou `record struct` para value objects e DTOs imutáveis.
  Invariantes em construtor estático ou método factory. Não usar record para entidades mutáveis.
- **Repository:** Interface no lado consumidor (Application). Repository concreto em Infrastructure.
  Não expor `IQueryable<T>` fora de Infrastructure — mapear para entidades de domínio.
- **MediatR Command/Query:** `IRequest<TResponse>` para Commands e Queries. Handler em arquivo
  separado do Command/Query. Notificações com `INotification` para domain events desacoplados.

**Etapa 2 — Selecionar contexto (lazy-load):**
Listar os 17 gatilhos de carregamento (espelhar estrutura do go-implementation), adaptados ao ecossistema .NET:
- `references/conventions.md` — estrutura de projeto, namespaces, nomeação, nullable
- `references/testing.md` — xUnit, NSubstitute, Testcontainers, table-driven (TheoryData), cobertura
- `references/api.md` — Minimal APIs, Controllers, Filters, Problem Details, versioning
- `references/patterns.md` — strategy, composite, specification, decorator, chain-of-responsibility
- `references/concurrency.md` — async/await, CancellationToken, Channel<T>, Parallel, IAsyncEnumerable
- `references/resilience.md` — Polly v8, Microsoft.Extensions.Resilience, circuit breaker, hedging
- `references/build.md` — Dockerfile multistage, .NET 9 chiseled images, GitHub Actions, dotnet publish
- `references/graceful-lifecycle.md` — IHostedService, BackgroundService, IHostApplicationLifetime, SIGTERM
- `references/examples-domain-flow.md` — esqueleto completo: Entity → Command → Handler → Endpoint → Teste
- `references/examples-testing.md` — TheoryData, InlineData, ClassData, fakes com Bogus, NSubstitute
- `references/examples-infrastructure.md` — graceful shutdown, cursor pagination, API versioning
- `references/configuration.md` — IOptions<T>, IOptionsMonitor<T>, IOptionsSnapshot<T>, secrets
- `references/persistence.md` — EF Core 9, Dapper, migrations, Unit of Work, shadow properties
- `references/observability.md` — ILogger<T>, OpenTelemetry, Activity, Meter, structured logging
- `references/security.md` — JWT Bearer, IAuthorizationService, data protection, CORS, rate limiting
- `references/messaging.md` — MassTransit, outbox pattern, idempotent consumers, saga

**Economia de contexto:** mesma regra — se mais de 4 referências, priorizar 3 mais críticas.

**Etapa 3 — Modelar:**
- Menor conjunto seguro de mudanças.
- Preferir `record` para DTOs, `class` para entidades mutáveis.
- Nullable Reference Types habilitado: tratar warns como erros em projetos novos.
- Respeitar convenções do projeto (Controllers vs Minimal APIs, MediatR vs direto).

**Etapa 4 — Implementar:**
- Seguir `<TargetFramework>` do `.csproj`.
- Manter XML doc apenas em membros públicos de biblioteca; evitar em código de aplicação.
- Atualizar ou adicionar testes para toda mudança de comportamento.

**Etapa 5 — Validar:**
- `dotnet build --no-restore`
- `dotnet test --no-build`
- `dotnet format --verify-no-changes` (se disponível no projeto)
- `dotnet-csharpier --check` (se disponível)

**Tratamento de erros:**
- Se `.csproj` ausente, parar antes de assumir framework ou dependências.
- Se projeto usar solução multi-projeto (`.sln`), validar apenas os projetos afetados.
- Se houver conflito entre esta skill e a governança base, seguir restrição mais segura e registrar.

---

#### `references/architecture.md`

**Escopo:** Especificidades de arquitetura .NET — DI via Generic Host, estruturas de diretório para
Web API, Worker, gRPC e CLI. Princípios gerais já estão em `agent-governance/references/shared-architecture.md`.

**Conteúdo mínimo exigido:**

1. **DI no .NET:** `IServiceCollection` como contêiner padrão. Lifetimes: Singleton, Scoped, Transient
   com regras de uso explícitas. Scrutor para scanning. Evitar `ServiceLocator` anti-pattern.

2. **Layouts de projeto** com estruturas de diretório para:
   - Web API (Clean Architecture): `Domain/`, `Application/`, `Infrastructure/`, `Api/`
   - Worker / Background Service: Generic Host sem ASP.NET Core quando não há HTTP
   - gRPC Service: Protos em `Protos/`, geração via `Grpc.Tools`
   - Monolito Modular: módulos como projetos separados ou features verticais
   - CLI: `System.CommandLine` ou `Spectre.Console` para interfaces ricas

3. **Regras .NET:**
   - Um projeto por camada no mínimo; não cruzar dependências de camada.
   - `Domain` não referencia nada externo (zero NuGet packages externos).
   - `Application` referencia apenas `Domain` e interfaces.
   - `Infrastructure` implementa interfaces de `Application`.
   - Evitar `using static` em código de produção.

---

#### `references/conventions.md`

**Conteúdo mínimo exigido:**

1. **Nomenclatura:** PascalCase para tipos/membros públicos, camelCase para variáveis locais e
   parâmetros, `_camelCase` para campos privados, `I`-prefixo apenas para interfaces.

2. **Nullable Reference Types:** `<Nullable>enable</Nullable>` em projetos novos. Tratar aviso
   CS8600–CS8625 como erros (`<TreatWarningsAsErrors>true</TreatWarningsAsErrors>`). Usar
   `required` em propriedades que devem ser definidas na inicialização.

3. **Records vs Classes:**
   - `record` (ou `record struct`) para value objects, DTOs de request/response e domain events.
   - `class` para entidades de domínio com identidade e mutabilidade controlada.
   - Não usar `record` para entidades com ciclo de vida gerenciado por ORM.

4. **Primary Constructors (C# 12+):** Usar em serviços, handlers e repositórios para injeção limpa.
   Não usar quando houver lógica no corpo do construtor além de atribuição.

5. **Organização de arquivos:** um tipo por arquivo; arquivo nomeado igual ao tipo principal.
   Exceção: tipos privados auxiliares podem coexistir no mesmo arquivo.

6. **`using` declarations:** preferir `using var` sobre `using(){}` para recursos em escopos simples.

---

#### `references/testing.md`

**Conteúdo mínimo exigido:**

1. **Unit Tests:**
   - xUnit como framework padrão: `[Fact]` para caso único, `[Theory]` + `[InlineData]` / `[MemberData]`
     / `[ClassData]` para variações — equivalente ao table-driven do Go.
   - FluentAssertions para asserções legíveis: `result.Should().Be(expected)`.
   - NSubstitute para mocks: `Substitute.For<IRepository>()`. Preferir sobre Moq por API mais limpa.
   - Bogus (`Faker<T>`) para geração de dados de teste com builders tipados — eliminar dados literais.
   - Nomenclatura: `MethodName_Scenario_ExpectedResult` (ex: `Confirm_OrderAlreadyShipped_ThrowsDomainException`).
   - Testes determinísticos: sem `Thread.Sleep`, sem estado global, sem dependência de ordem de execução.
   - Isolar com `FakeXxx` (hand-written) ou `Substitute.For<>` (NSubstitute) — nunca usar banco real em unit test.

2. **Integration Tests:**
   - Testcontainers.NET para containers efêmeros (Postgres, Redis, RabbitMQ, Kafka).
   - `WebApplicationFactory<TEntryPoint>` para testar a pipeline HTTP completa.
   - `IClassFixture<T>` para compartilhar container entre testes da mesma classe.
   - Build tag de separação: projeto separado `*.IntegrationTests.csproj` ou categoria via `[Trait]`.

3. **Exemplo de Integration Test com Testcontainers:**
   ```csharp
   public class OrderRepositoryTests : IAsyncLifetime
   {
       private readonly PostgreSqlContainer _postgres = new PostgreSqlBuilder()
           .WithImage("postgres:16-alpine")
           .Build();

       public async Task InitializeAsync() => await _postgres.StartAsync();
       public async Task DisposeAsync() => await _postgres.DisposeAsync();

       [Fact]
       public async Task Save_ValidOrder_PersistsToDatabase()
       {
           // arrange
           await using var context = CreateDbContext(_postgres.GetConnectionString());
           var repo = new OrderRepository(context);

           // act + assert
           ...
       }
   }
   ```

4. **Proibido:**
   - `Thread.Sleep` ou `Task.Delay` para sincronização.
   - Teste dependendo de serviço externo real.
   - Mock que retorna valor estático sem verificar interação quando o comportamento importa.
   - Ignorar `CancellationToken` em testes assíncronos.

---

#### `references/api.md`

**Conteúdo mínimo exigido:**

1. **Minimal APIs (preferido em .NET 9):**
   - Agrupar endpoints com `RouteGroupBuilder` e `MapGroup()`.
   - Usar `TypedResults` para respostas tipadas e gerar OpenAPI correto automaticamente.
   - Filtros com `IEndpointFilter` para validação, logging e tratamento de erros transversais.
   - `WithOpenApi()` e `WithTags()` para documentação automática.

2. **Problem Details (RFC 9457):**
   - `IProblemDetailsService` injetado para respostas de erro consistentes.
   - `IExceptionHandler` para capturar exceções não tratadas e mapear para `ProblemDetails`.
   - Nunca expor stack trace, mensagem interna ou path em produção.

3. **Validação de Request:**
   - FluentValidation integrado como `IEndpointFilter` ou via `IValidator<T>` injetado no handler.
   - Retornar 400 com `ValidationProblemDetails` para erros de validação.
   - Validar e rejeitar na borda — não propagar input não validado para Application.

4. **Versionamento de API:**
   - `Asp.Versioning` (pacote oficial) para versionamento por URL ou header.
   - Deprecar versões com `[ApiVersion(..., Deprecated = true)]`.

5. **Compressão e Content Negotiation:**
   - Habilitar `ResponseCompression` para endpoints de alta frequência.
   - Usar `[Produces("application/json")]` para controle explícito de content type.

---

#### `references/persistence.md`

**Conteúdo mínimo exigido:**

1. **Repository Pattern com EF Core 9:**
   - Interface no projeto `Application` — sem referência a EF Core no domínio.
   - Implementação em `Infrastructure` com `DbContext` injetado.
   - Não expor `IQueryable<T>` fora de Infrastructure.
   - Mapear entidades EF Core para entidades de domínio no repository (nunca retornar `DbSet<T>` diretamente).

2. **Unit of Work:**
   - `IUnitOfWork` com `CommitAsync(CancellationToken)` gerenciado na camada de Application.
   - Não chamar `SaveChangesAsync()` dentro do repository — responsabilidade do handler/use case.

3. **EF Core 9 — recursos relevantes:**
   - `ExecuteUpdateAsync` / `ExecuteDeleteAsync` para bulk operations sem carregar entidades.
   - Shadow properties para auditoria (`CreatedAt`, `UpdatedAt`) sem poluir o domínio.
   - `ComplexType` (EF Core 8+) para value objects mapeados como owned types sem tabela separada.
   - `TimeProvider` injetado nos interceptors para datas testáveis (sem `DateTime.UtcNow` hardcoded).

4. **Migrations:**
   - `dotnet ef migrations add <Name> --project Infrastructure --startup-project Api`
   - `dotnet ef database update` apenas em desenvolvimento; scripts SQL em produção.
   - Migrations destrutivas devem ter script de rollback revisado antes de deploy.
   - Separar migrations de schema (DDL) de migrations de dados (DML) quando possível.

5. **Dapper (alternativa para queries complexas):**
   - Usar `IDbConnectionFactory` injetável para testabilidade.
   - Sempre parametrizar queries — nunca concatenar input.
   - Usar `QueryAsync<T>` com `CancellationToken` via extensão custom quando disponível.

6. **Proibido:**
   - SQL por concatenação de string com input externo.
   - `DbContext` com vida Singleton.
   - Domínio importando `Microsoft.EntityFrameworkCore`.
   - `SaveChangesAsync()` dentro de repository individual (quebra Unit of Work).

---

#### `references/configuration.md`

**Conteúdo mínimo exigido:**

1. **`IOptions<T>` vs `IOptionsMonitor<T>` vs `IOptionsSnapshot<T>`:**
   - `IOptions<T>`: Singleton, lido uma vez na inicialização — para config imutável.
   - `IOptionsSnapshot<T>`: Scoped, reavaliado por request — para config que pode mudar em dev.
   - `IOptionsMonitor<T>`: Singleton com notificação de mudança — para config dinâmica em produção.

2. **Validação de configuração no startup:**
   ```csharp
   services.AddOptions<DatabaseOptions>()
       .BindConfiguration("Database")
       .ValidateDataAnnotations()
       .ValidateOnStart(); // falha rápido se config inválida
   ```

3. **Secrets:**
   - Desenvolvimento: `dotnet user-secrets set "ConnectionStrings:Default" "..."`.
   - Produção: variáveis de ambiente ou Azure Key Vault / AWS Secrets Manager via provider.
   - Nunca commitar `appsettings.Production.json` com segredos reais.

4. **Ambientes:**
   - `appsettings.json` (defaults) + `appsettings.{Environment}.json` (override).
   - `ASPNETCORE_ENVIRONMENT` ou `DOTNET_ENVIRONMENT` para controlar o ambiente ativo.

---

#### `references/resilience.md`

**Conteúdo mínimo exigido:**

1. **Polly v8 via `Microsoft.Extensions.Resilience`:**
   ```csharp
   services.AddResiliencePipeline("external-api", builder =>
   {
       builder
           .AddRetry(new RetryStrategyOptions
           {
               MaxRetryAttempts = 3,
               BackoffType = DelayBackoffType.Exponential,
               UseJitter = true,
               ShouldHandle = new PredicateBuilder().Handle<HttpRequestException>()
                                                    .HandleResult<HttpResponseMessage>(r => !r.IsSuccessStatusCode)
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

2. **`HttpClient` com Polly:**
   - Usar `AddStandardResilienceHandler()` do `Microsoft.Extensions.Http.Resilience` como baseline.
   - Named clients via `IHttpClientFactory` — nunca instanciar `HttpClient` diretamente.

3. **Hedging:** Disponível em Polly v8 para requests paralelos com first-winner.

4. **`TimeProvider`:** Usar abstração `TimeProvider` (System.TimeProvider) para timeouts testáveis
   em vez de `DateTime.UtcNow` ou `DateTimeOffset.UtcNow` hardcoded.

5. **Regras:**
   - Retry apenas para erros transitórios (5xx, timeout, network) — nunca para 4xx.
   - Backoff exponencial com jitter obrigatório.
   - Respeitar `CancellationToken` em todo pipeline de resiliência.

6. **Proibido:**
   - `HttpClient` instanciado com `new` — causa socket exhaustion.
   - Retry infinito ou sem limite de tentativas.
   - Ignorar `CancellationToken` em retry loops.

---

#### `references/observability.md`

**Conteúdo mínimo exigido:**

1. **Logging Estruturado:**
   - `ILogger<T>` como abstração padrão; Serilog como provider de produção com sinks OTLP ou Console JSON.
   - Usar `LoggerMessage.Define` ou source generator `[LoggerMessage]` para hot paths (evita boxing).
   - Campos mínimos: `level`, `message`, `TraceId`, `SpanId`, `exception`.
   - Não logar PII, tokens, senhas ou payloads completos de request.

2. **OpenTelemetry .NET:**
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

3. **Activity (Tracing Manual):**
   - `ActivitySource` com nome versionado para criar spans customizados.
   - Propagar `Activity.Current` via `CancellationToken` em chamadas assíncronas.
   - Nomear atividades pelo papel da operação, não pelo nome do método.

4. **Métricas customizadas:**
   - `Meter` + `Counter<T>`, `Histogram<T>`, `ObservableGauge<T>`.
   - Labels com cardinalidade controlada — nunca ID de usuário ou request ID como label.

5. **Health Checks:**
   ```csharp
   builder.Services.AddHealthChecks()
       .AddDbContextCheck<AppDbContext>("database")
       .AddCheck<RedisHealthCheck>("redis");

   app.MapHealthChecks("/health/live", new() { Predicate = _ => false });
   app.MapHealthChecks("/health/ready");
   ```

6. **Proibido:**
   - `Console.WriteLine` ou `Debug.WriteLine` em código de produção.
   - Logar exceção completa com stack trace para o cliente.
   - Métrica com label derivado de input de usuário sem sanitização.

---

#### `references/security.md`

**Conteúdo mínimo exigido:**

1. **Autenticação JWT Bearer:**
   - `AddJwtBearer()` com validação explícita de `Issuer`, `Audience`, `IssuerSigningKey` e `ClockSkew`.
   - Verificar expiração e claims obrigatórios em cada request — não cachear decisão.
   - `kid` header para rotação de chave sem downtime.
   - Algoritmos assimétricos (RS256, ES256) quando múltiplos serviços validam tokens.

2. **Autorização:**
   - Policies explícitas com `AddAuthorization()` + `builder.Services.AddAuthorizationBuilder()`.
   - `IAuthorizationService` para autorização imperativa dentro de use cases.
   - Aplicar princípio de menor privilégio — não usar `[AllowAnonymous]` como default.

3. **Data Protection:**
   - `AddDataProtection()` com chave persistida externamente (Azure Blob, Redis) em produção.
   - Não usar para criptografia de dados de negócio — usar para tokens internos, cookies de sessão.

4. **Input Validation e Sanitização:**
   - Nunca confiar em input do cliente para decisões de autorização.
   - Limitar tamanho de payload com `MaxRequestBodySize` ou `LengthLimit` em Minimal APIs.
   - HTML encoding automático via Razor/Blazor — verificar em APIs REST com `HtmlEncoder.Default`.

5. **CORS:**
   - Origins explícitas em produção — não usar `AllowAnyOrigin()`.
   - `AllowCredentials()` apenas com origins específicas — nunca com wildcard.

6. **Rate Limiting (.NET 7+):**
   - `AddRateLimiter()` com políticas nomeadas por recurso.
   - Sliding window ou token bucket para endpoints públicos; fixed window para login.

7. **HTTPS e Headers:**
   - `UseHsts()` em produção. `UseHttpsRedirection()` em desenvolvimento.
   - Security headers via middleware ou `IHeaderDictionary`: `X-Content-Type-Options`, `X-Frame-Options`.

8. **Proibido:**
   - Segredo hardcoded em `.cs` ou `appsettings.json` commitado.
   - SQL por concatenação de string com input externo.
   - `InsecureSkipVerify` equivalente — ignorar erros de certificado TLS em produção.
   - Expor stack trace, mensagem interna ou estrutura do banco em resposta de erro.

---

#### `references/messaging.md`

**Conteúdo mínimo exigido:**

1. **MassTransit v8:**
   - `AddMassTransit()` com transporte configurável (RabbitMQ, Azure Service Bus, Kafka, In-Memory para testes).
   - Consumers implementam `IConsumer<TMessage>` com `CancellationToken` via `ConsumeContext`.
   - Consumers devem ser **idempotentes** — at-least-once delivery é a garantia padrão.

2. **Outbox Pattern:**
   - MassTransit Entity Framework Outbox: `AddEntityFrameworkOutbox<TDbContext>()`.
   - Publicar mensagens dentro da transação do banco — commit atômico de dados + mensagem.
   - Processor de outbox como `IHostedService` separado com retry e DLQ.

3. **Dead-Letter Queue:**
   - Configurar fault consumers: `IConsumer<Fault<TMessage>>` para mensagens que falharam após N tentativas.
   - Logar contexto suficiente para diagnóstico sem reprocessar manualmente.

4. **Domain Events com MediatR:**
   - `INotification` + `INotificationHandler<T>` para eventos de domínio in-process.
   - Publicar `INotification` dentro do handler de Command após commit.
   - Não confundir domain events in-process com integration events via broker.

5. **Schema e Contratos:**
   - Definir contratos de mensagem em projeto compartilhado (`Contracts/`) ou pacote NuGet.
   - Usar backward-compatible changes: adicionar campos opcionais, nunca remover ou renomear.
   - `MessageUrn` ou namespace explícito para evitar colisão de tipo em topicos/filas.

6. **Proibido:**
   - Publicar evento antes do commit sem outbox pattern.
   - Consumer que ignora exceção e confirma mensagem (ack sem processar).
   - Mensagem sem `CorrelationId` ou trace context.

---

#### `references/concurrency.md`

**Conteúdo mínimo exigido:**

1. **`async`/`await` — Regras Fundamentais:**
   - Sempre propagar `CancellationToken` até a borda de IO — não swallow cancellation.
   - Nunca usar `.Result` ou `.Wait()` em código assíncrono — causa deadlock em ASP.NET Core.
   - `ConfigureAwait(false)` em bibliotecas; desnecessário em código de aplicação ASP.NET.
   - Usar `ValueTask<T>` apenas quando o caminho síncrono for o comum (ex: cache hit) — evitar em APIs gerais.

2. **`CancellationToken` em fronteiras:**
   - Toda operação de IO (EF Core, HttpClient, MassTransit) aceita `CancellationToken`.
   - Passar token recebido do framework — não criar `CancellationTokenSource` sem justificativa.
   - `CancellationToken.ThrowIfCancellationRequested()` para checkpoints em loops longos.

3. **`Channel<T>` (producer-consumer):**
   - `Channel.CreateBounded<T>(capacity)` para limitar memória e aplicar backpressure.
   - `Channel.CreateUnbounded<T>()` apenas quando o producer tiver taxa controlada comprovada.
   - Completar o writer com `channel.Writer.Complete()` para sinalizar fim de produção.

4. **`Parallel` e `Task.WhenAll`:**
   - `Parallel.ForEachAsync` (.NET 6+) com `DegreeOfParallelism` explícito para fan-out controlado.
   - `Task.WhenAll` para fan-out quando todas as tarefas devem completar; `Task.WhenAny` para first-winner.
   - Evitar `Task.Run` em código ASP.NET Core — thread pool já gerenciado pelo runtime.

5. **`IAsyncEnumerable<T>`:**
   - Usar para streaming de dados (queries paginadas, arquivos grandes) sem buffering completo em memória.
   - `await foreach` com `CancellationToken` via `WithCancellation()`.

6. **Proibido:**
   - `.Result` ou `.Wait()` em código ASP.NET Core.
   - `Thread.Sleep` para sincronização.
   - `new Thread()` em aplicações ASP.NET Core.
   - Fan-out ilimitado sem controle de grau de paralelismo.
   - Ignorar `OperationCanceledException` sem relançar ou tratar.

---

#### `references/graceful-lifecycle.md`

**Conteúdo mínimo exigido:**

1. **Generic Host e `IHostedService`:**
   - `IHostedService.StartAsync` / `StopAsync` para serviços de background com controle explícito.
   - `BackgroundService` como classe base para loops contínuos: implementar `ExecuteAsync(CancellationToken)`.
   - `stoppingToken` propagado para toda operação de IO dentro do `ExecuteAsync`.

2. **Ordem de inicialização:**
   - `IHostApplicationLifetime.ApplicationStarted` para ações pós-inicialização (ex: readiness probe ativa).
   - `WebApplication.RunAsync()` bloqueia até SIGTERM/SIGINT — não usar `Run()` quando o token importa.

3. **Shutdown gracioso:**
   ```csharp
   builder.Services.Configure<HostOptions>(options =>
   {
       options.ShutdownTimeout = TimeSpan.FromSeconds(15); // default 5s — insuficiente para workers
   });
   ```
   - Timeout de shutdown deve ser menor que `terminationGracePeriodSeconds` do Kubernetes.

4. **Drenagem de conexões HTTP:**
   - `UseShutdownTimeout()` + Kestrel `Limits.KeepAliveTimeout` para drenagem de conexões HTTP em aberto.
   - `IHostApplicationLifetime.ApplicationStopping` para sinalizar que o processo não aceita novos requests.

5. **Ordem de encerramento:**
   - Inversão da ordem de inicialização: HTTP server → consumers → database → telemetry flush.
   - `IAsyncDisposable` em serviços com recursos — garantir `await DisposeAsync()` no shutdown.

6. **Proibido:**
   - `Environment.Exit()` fora de `Main` sem flush de recursos.
   - `BackgroundService` que ignora `stoppingToken`.
   - `ShutdownTimeout` deixado no default (5s) para workers com processamento longo.
   - `IHostedService` que não completa `StopAsync` antes do timeout.

---

#### `references/build.md`

**Conteúdo mínimo exigido:**

1. **Dockerfile multistage com .NET 9:**
   ```dockerfile
   FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
   WORKDIR /src
   COPY ["src/Api/Api.csproj", "src/Api/"]
   RUN dotnet restore "src/Api/Api.csproj"
   COPY . .
   RUN dotnet publish "src/Api/Api.csproj" -c Release -o /app/publish

   FROM mcr.microsoft.com/dotnet/aspnet:9.0-noble-chiseled AS runtime
   WORKDIR /app
   COPY --from=build /app/publish .
   USER $APP_UID
   ENTRYPOINT ["dotnet", "Api.dll"]
   ```
   - Imagem **chiseled** (Ubuntu minimal) para surface de ataque reduzida e menor tamanho.
   - `USER $APP_UID` para executar como não-root.
   - `.dockerignore` excluindo `bin/`, `obj/`, `.git/`.

2. **GitHub Actions — pipeline mínima:**
   ```yaml
   - run: dotnet restore
   - run: dotnet build --no-restore -c Release
   - run: dotnet test --no-build -c Release --collect:"XPlat Code Coverage"
   - run: dotnet publish --no-build -c Release -o publish/
   ```

3. **Análise estática e qualidade:**
   - `dotnet format --verify-no-changes` como gate de CI.
   - Roslyn Analyzers: `Microsoft.CodeAnalysis.NetAnalyzers` habilitado no `.csproj`.
   - `<TreatWarningsAsErrors>true</TreatWarningsAsErrors>` em projetos de produção.
   - `dotnet-csharpier` para formatação determinística (alternativa opinionada).

4. **NuGet e dependências:**
   - `Directory.Build.props` para versões centralizadas de pacotes.
   - `<ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>` com `Directory.Packages.props`.
   - `dotnet list package --vulnerable` em CI para detectar CVEs conhecidos.

---

#### `references/patterns.md`

**Escopo:** Padrões não cobertos inline no SKILL.md. Factory, Primary Constructor, Record e Repository
já estão inline — **não duplicar aqui**.

**Conteúdo mínimo exigido:**

1. **Specification Pattern:** `ISpecification<T>` com `Expression<Func<T, bool>>` para queries EF Core
   combináveis. Usar quando regras de negócio forem reutilizadas em múltiplos handlers/repositórios.

2. **Decorator com Scrutor:**
   ```csharp
   services.AddScoped<IOrderService, OrderService>();
   services.Decorate<IOrderService, LoggingOrderService>();
   services.Decorate<IOrderService, CachingOrderService>();
   ```
   - Cada decorator adiciona responsabilidade transversal (logging, cache, métricas) sem modificar o original.

3. **Strategy:** Interface + múltiplas implementações registradas como `IEnumerable<IStrategy>`.
   Seleção por chave com `IKeyedServiceProvider` (.NET 8+) ou dictionary de factories.

4. **Chain of Responsibility com MediatR Pipeline Behaviors:**
   ```csharp
   services.AddTransient(typeof(IPipelineBehavior<,>), typeof(ValidationBehavior<,>));
   services.AddTransient(typeof(IPipelineBehavior<,>), typeof(LoggingBehavior<,>));
   ```
   - Validação, logging, retry e UoW commit como behaviors ortogonais ao handler.

5. **Result Pattern:**
   - Usar `Result<T>` (ou `OneOf<T, Error>`) em fronteiras de Application/Domain em vez de exceções
     para fluxos esperados (not found, validation failed).
   - Exceções apenas para falhas inesperadas de infraestrutura.

---

#### `references/examples-domain-flow.md`

**Conteúdo mínimo exigido:**

Esqueleto completo e funcional de um fluxo `CreateOrder` end-to-end em Clean Architecture:

```
Domain/
  Order.cs           — entidade com factory static e domain events
  IOrderRepository.cs — interface de porta (sem EF Core)

Application/
  CreateOrder/
    CreateOrderCommand.cs   — record IRequest<OrderId>
    CreateOrderHandler.cs   — IRequestHandler com Unit of Work
    CreateOrderValidator.cs — FluentValidation AbstractValidator<CreateOrderCommand>

Infrastructure/
  Persistence/
    OrderRepository.cs      — implementação EF Core
    AppDbContext.cs

Api/
  Endpoints/
    OrderEndpoints.cs       — MapGroup + TypedResults
```

- Handler deve validar via MediatR ValidationBehavior (não duplicar no endpoint).
- Endpoint retorna `TypedResults.Created(...)` com Location header correto.
- Teste unitário do handler com NSubstitute + FluentAssertions.
- Teste de integração do endpoint com `WebApplicationFactory` + Testcontainers.

---

#### `references/examples-testing.md`

**Conteúdo mínimo exigido:**

1. **TheoryData (table-driven equivalente):**
   ```csharp
   public static TheoryData<string, bool> EmailValidationCases => new()
   {
       { "valid@example.com", true },
       { "invalid-email", false },
       { "", false },
   };

   [Theory, MemberData(nameof(EmailValidationCases))]
   public void Validate_Email_ReturnsExpectedResult(string email, bool expected)
   {
       var result = new EmailValidator().IsValid(email);
       result.Should().Be(expected);
   }
   ```

2. **Bogus para builders de teste:**
   ```csharp
   private static readonly Faker<Order> _orderFaker = new Faker<Order>()
       .CustomInstantiator(f => Order.Create(
           customerId: f.Random.Guid(),
           items: f.Make(3, () => new OrderItem(f.Commerce.ProductName(), f.Random.Int(1, 10))),
           currency: f.PickRandom("BRL", "USD", "EUR")
       ));
   ```

3. **NSubstitute com verificação de interação:**
   ```csharp
   var repo = Substitute.For<IOrderRepository>();
   repo.GetByIdAsync(Arg.Any<Guid>(), Arg.Any<CancellationToken>())
       .Returns(Task.FromResult<Order?>(null));

   // ...exercitar o SUT...

   await repo.Received(1).SaveAsync(Arg.Is<Order>(o => o.Status == OrderStatus.Confirmed),
                                    Arg.Any<CancellationToken>());
   ```

---

#### `references/examples-infrastructure.md`

**Conteúdo mínimo exigido:**

1. **Graceful shutdown completo** com `BackgroundService`, `IHostApplicationLifetime` e drenagem HTTP.
2. **Cursor-based pagination** com EF Core: `Where(x => x.Id > cursor).Take(pageSize).OrderBy(x => x.Id)`.
3. **API versioning** com `Asp.Versioning.Http`: grupos de rota versionados com deprecação explícita.
4. **Outbox processor** como `BackgroundService` com `PeriodicTimer` e retry via Polly.

---

### Critérios de Aceitação da Skill (Production-Ready)

A skill é considerada production-ready quando atende **todos** os critérios abaixo:

| # | Critério                                                                                     | Como verificar                                              |
|---|----------------------------------------------------------------------------------------------|-------------------------------------------------------------|
| 1 | Nenhum namespace, tipo, método ou pacote NuGet fictício em qualquer referência               | Verificar cada API contra docs.microsoft.com                |
| 2 | Todo exemplo de código compila com .NET 9 SDK e C# 13 sem warnings de compilação            | Testar em projeto scratch antes de incluir                  |
| 3 | Padrões inline no SKILL.md não são repetidos nas referências (sem duplicação)                | Grep por termos inline nas references                       |
| 4 | Cada `references/*.md` tem TL;DR com Keywords e condição de carregamento                    | Verificar presença do bloco `<!-- TL;DR ... -->`            |
| 5 | A skill cobre os mesmos tópicos que `go-implementation` (17 referências)                    | Comparar lista de arquivos                                   |
| 6 | Exemplos de teste não usam `.Result`, `.Wait()`, `Thread.Sleep` ou estado global            | Revisão manual dos exemplos                                 |
| 7 | `references/security.md` cobre autenticação JWT, CORS, rate limiting e SQL injection        | Checklist de seções                                         |
| 8 | `references/resilience.md` usa Polly v8 com API `Microsoft.Extensions.Resilience`           | Verificar namespace correto: `Microsoft.Extensions.Resilience` |
| 9 | Dockerfile usa imagem `noble-chiseled` e `USER $APP_UID`                                     | Revisar o exemplo de Dockerfile                             |
| 10 | Toda operação assíncrona propaga `CancellationToken` nos exemplos                           | Revisão de assinaturas nos exemplos                         |

---

### Anti-Padrões a Evitar Explicitamente na Skill

| Anti-padrão                                                      | Correção esperada na skill                                       |
|------------------------------------------------------------------|------------------------------------------------------------------|
| Usar `new HttpClient()` diretamente                              | Sempre via `IHttpClientFactory`                                  |
| `DbContext` com lifetime Singleton                               | Scoped por padrão; Transient apenas em cenários especiais        |
| `.Result` ou `.Wait()` em código async                           | `await` em todo ponto de IO                                      |
| `Console.WriteLine` em produção                                  | `ILogger<T>` com structured logging                             |
| `DateTime.UtcNow` hardcoded em serviços                          | `TimeProvider` injetado via DI                                   |
| Exceção para fluxos esperados (not found, validation)            | `Result<T>` ou `Problem Details` com status code adequado       |
| `[AllowAnonymous]` como default                                  | Require authentication por default, opt-out explícito            |
| `IQueryable<T>` exposto fora de Infrastructure                   | Materializar no repositório, retornar entidades de domínio       |
| Migrations executadas automaticamente em produção                | Script SQL gerado por `dotnet ef migrations script`              |
| Mock sem verificação de interação quando comportamento importa   | `Received(n)` com `NSubstitute` ou `Verify` explícito           |

---

### Instruções de Execução para o Agente

1. **Ler antes de criar:**
   - `.agents/skills/go-implementation/SKILL.md` (padrão de referência principal)
   - `.agents/skills/node-implementation/references/conventions.md`
   - `AGENTS.md` (contrato de carga base)

2. **Não criar nada sem verificar a API:**
   - Cada tipo, namespace e método deve ser verificado nas docs oficiais antes de incluir no arquivo.
   - Links de referência aceitos: `learn.microsoft.com`, `github.com/dotnet`, `nuget.org`.

3. **Estrutura de cada `references/*.md`:**
   ```markdown
   # Título

   <!-- TL;DR
   Resumo de 1-2 linhas do propósito do arquivo.
   Keywords: palavra1, palavra2, palavra3
   Load complete when: condição objetiva que justifica carregar este arquivo.
   -->

   ## Objetivo
   ...

   ## Diretrizes
   ...

   ## Riscos Comuns
   ...

   ## Proibido
   - item
   ```

4. **Após criar todos os arquivos:**
   - Verificar que `SKILL.md` lista exatamente os mesmos 17 gatilhos de carregamento especificados.
   - Verificar que nenhum exemplo de código usa APIs que não existem em .NET 9 / C# 13.
   - Verificar que `references/architecture.md` tem estrutura de diretório para todos os tipos de projeto (.Web API, Worker, gRPC, CLI).

5. **Este é um prompt de criação de skill — não implementar código de aplicação.**
   - O output deve ser apenas os arquivos `.md` da skill.
   - Nenhum `.csproj`, `.cs` ou `Dockerfile` deve ser criado no repositório (apenas como exemplos dentro dos `.md`).
