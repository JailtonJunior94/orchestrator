# Arquitetura .NET / C#

<!-- TL;DR
Especificidades de arquitetura .NET: DI via Generic Host/IServiceCollection, lifetimes, layouts de projeto (Web API, Worker, gRPC, CLI) e regras de fronteira de camada.
Keywords: arquitetura, di, generic host, IServiceCollection, lifetime, scrutor, camadas, layout, clean architecture
Load complete when: a tarefa envolve estrutura de projeto, injecao de dependencias, lifetimes ou organizacao de camadas .NET.
-->

## Objetivo

Principios gerais de arquitetura, DI e sinais de excesso estao em `shared-architecture.md`
(agent-governance). Este arquivo cobre apenas especificidades .NET.

## Diretrizes

### DI no .NET
- `IServiceCollection` e o contêiner padrao; registrar no `Program.cs` ou em extensions
  `AddXxx(this IServiceCollection)`. Resolver via construtor — evitar o anti-pattern Service Locator
  (`IServiceProvider.GetService` em codigo de dominio/aplicacao).
- Lifetimes:
  - `Singleton` — sem estado mutavel por request; seguro para concorrencia. Ex: `IHttpClientFactory`,
    `TimeProvider`, caches imutaveis.
  - `Scoped` — uma instancia por request/escopo. Default para `DbContext`, repositories e handlers.
  - `Transient` — nova instancia por resolucao. Para servicos leves e sem estado.
  - Regra: nunca injetar um servico `Scoped` dentro de um `Singleton` (captive dependency).
- `Scrutor` para assembly scanning e registro por convencao:
  `services.Scan(s => s.FromAssemblyOf<IMarker>().AddClasses().AsImplementedInterfaces().WithScopedLifetime())`.
- Keyed services (`AddKeyedScoped`, `[FromKeyedServices]`) quando houver multiplas implementacoes
  selecionadas por chave.

### Layouts de projeto

**Web API (Clean Architecture)** — um projeto por camada:
```
src/
  Domain/          # entidades, value objects, domain events, interfaces de dominio. Zero NuGet externo.
  Application/     # use cases, Commands/Queries, interfaces de porta (IRepository, IUnitOfWork)
  Infrastructure/  # EF Core, clients HTTP, messaging, implementacoes das interfaces de Application
  Api/             # Minimal APIs/Controllers, DI wiring, Program.cs, filtros, middlewares
```

**Worker / Background Service** — Generic Host sem ASP.NET Core quando nao ha HTTP:
```
src/
  Worker/          # Program.cs com Host.CreateApplicationBuilder, BackgroundService
  Application/
  Infrastructure/
```

**gRPC Service** — protos versionados e geracao via `Grpc.Tools`:
```
src/
  Api/Protos/      # arquivos .proto; <Protobuf Include="Protos/*.proto" GrpcServices="Server" />
  Application/
  Infrastructure/
```

**Monolito Modular** — modulos como projetos separados ou features verticais, cada um com suas camadas;
modulos comunicam por contratos publicos, nunca por tipos internos.

**CLI** — `System.CommandLine` (parsing robusto) ou `Spectre.Console` (UI rica):
```
src/
  Cli/             # Program.cs, root command, subcommands (um arquivo por comando)
  Application/
  Infrastructure/
```

## Riscos Comuns
- Camadas cruzadas: `Domain` referenciando EF Core ou ASP.NET Core quebra a regra de dependencia.
- Captive dependency: `Scoped` capturado por `Singleton` gera estado compartilhado entre requests.
- `DbContext` registrado como `Singleton` — nao e thread-safe.

## Proibido
- `Domain` referenciando qualquer NuGet externo (EF Core, MediatR, ASP.NET Core).
- `Application` referenciando `Infrastructure` (a dependencia e invertida via interfaces).
- `using static` em codigo de producao quando obscurece a origem do simbolo.
- Service Locator (`GetService`/`GetRequiredService`) fora da raiz de composicao.
