# API .NET / C#

<!-- TL;DR
Construcao de APIs em .NET: Minimal APIs com RouteGroupBuilder e TypedResults, Problem Details RFC 9457, validacao com FluentValidation e versionamento com Asp.Versioning.
Keywords: minimal api, mapgroup, typedresults, problem details, IExceptionHandler, fluentvalidation, asp.versioning, openapi
Load complete when: a tarefa envolve endpoints HTTP, controllers, filtros, validacao de request ou versionamento de API.
-->

## Objetivo

Definir as praticas de construcao de APIs HTTP em ASP.NET Core 10, com preferencia por Minimal APIs.

## Diretrizes

### Minimal APIs (preferido em .NET 10)
- Agrupar endpoints com `RouteGroupBuilder` via `MapGroup()`:
  ```csharp
  var orders = app.MapGroup("/orders").WithTags("Orders");
  orders.MapPost("/", CreateOrderAsync);
  orders.MapGet("/{id:guid}", GetOrderAsync);
  ```
- Usar `TypedResults` para respostas tipadas (gera OpenAPI correto automaticamente):
  `TypedResults.Created($"/orders/{id}", dto)`, `TypedResults.NotFound()`,
  `Results<Created<OrderDto>, ValidationProblem>` como tipo de retorno.
- Filtros transversais com `IEndpointFilter` (validacao, logging, tratamento de erro).
- `.WithName()`, `.WithTags()` e suporte OpenAPI nativo (`Microsoft.AspNetCore.OpenApi`,
  `app.MapOpenApi()`) para documentacao automatica.

### Problem Details (RFC 9457)
- `builder.Services.AddProblemDetails()` + `IExceptionHandler` para mapear excecoes nao tratadas:
  ```csharp
  app.UseExceptionHandler();
  // registrar: builder.Services.AddExceptionHandler<GlobalExceptionHandler>();
  ```
- Respostas de erro consistentes com `ProblemDetails` / `ValidationProblemDetails`.
- Nunca expor stack trace, mensagem interna ou path fisico em producao.

### Validacao de Request
- FluentValidation como `IEndpointFilter` ou via `IValidator<T>` injetado no handler.
- Retornar 400 com `ValidationProblemDetails` para erros de validacao.
- Validar e rejeitar na borda — nao propagar input nao validado para Application.

### Versionamento de API
- `Asp.Versioning.Http` / `Asp.Versioning.Mvc` (pacotes oficiais) para versionamento por URL,
  header ou query.
- Deprecar versoes com `[ApiVersion("1.0", Deprecated = true)]` ou `.HasDeprecatedApiVersion(...)`.

### Compressao e Content Negotiation
- `ResponseCompression` para endpoints de alta frequencia.
- `Produces`/`Accepts` explicitos para controle de content type e geracao de OpenAPI.

## Riscos Comuns
- Duplicar validacao no endpoint e no handler — centralizar via behavior/filtro.
- Retornar `IResult` nao tipado perde a geracao automatica de schema OpenAPI.
- `UseHttpsRedirection` sem `UseHsts` em producao.

## Proibido
- Expor excecao bruta ou stack trace na resposta.
- Endpoint que aceita input nao validado e o propaga para a camada de Application.
- Logica de negocio dentro do endpoint (deve delegar para handler/use case).
