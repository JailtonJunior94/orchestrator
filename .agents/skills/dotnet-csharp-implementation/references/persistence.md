# Persistencia .NET / C#

<!-- TL;DR
Persistencia em .NET: Repository com EF Core 10, Unit of Work, ExecuteUpdate/Delete, ComplexType, TimeProvider em interceptors, migrations e Dapper para queries complexas.
Keywords: ef core, dbcontext, repository, unit of work, migrations, executeupdate, complextype, shadow property, dapper, timeprovider
Load complete when: a tarefa envolve repositories, transactions, migrations, queries ou connection management.
-->

## Objetivo

Definir as praticas de acesso a dados com EF Core 10 e Dapper, preservando as fronteiras de camada.

## Diretrizes

### Repository Pattern com EF Core 10
- Interface no projeto `Application` — sem referencia a EF Core no dominio.
- Implementacao em `Infrastructure` com `DbContext` injetado (lifetime `Scoped`).
- Nao expor `IQueryable<T>` fora de Infrastructure; materializar e retornar entidades de dominio.
- Nunca retornar `DbSet<T>` diretamente.

### Unit of Work
- `IUnitOfWork` com `SaveChangesAsync(CancellationToken)` orquestrado na camada de Application
  (handler/use case), nao dentro de cada repository.
- O repository adiciona/remove no change tracker; o commit e responsabilidade do use case.

### EF Core 10 — recursos relevantes
- `ExecuteUpdateAsync` / `ExecuteDeleteAsync` para operacoes em massa sem carregar entidades.
- Shadow properties para auditoria (`CreatedAt`, `UpdatedAt`) sem poluir o dominio.
- `ComplexType` (owned-like sem identidade) para value objects mapeados sem tabela separada.
- `TimeProvider` injetado em interceptors (`SaveChangesInterceptor`) para datas testaveis em vez de
  `DateTime.UtcNow` hardcoded.
- `AsNoTracking()` em queries read-only para reduzir overhead de change tracking.

### Migrations
- `dotnet ef migrations add <Name> --project Infrastructure --startup-project Api`.
- `dotnet ef database update` apenas em desenvolvimento; em producao, aplicar scripts SQL gerados por
  `dotnet ef migrations script --idempotent`.
- Migrations destrutivas exigem script de rollback revisado antes do deploy.
- Separar migrations de schema (DDL) de migrations de dados (DML) quando possivel.

### Dapper (alternativa para queries complexas)
- `IDbConnectionFactory` injetavel para testabilidade.
- Sempre parametrizar — nunca concatenar input em SQL.
- `QueryAsync<T>` com `CommandDefinition` carregando `CancellationToken`.

## Riscos Comuns
- `SaveChangesAsync` chamado dentro do repository quebra o Unit of Work.
- Query sem `AsNoTracking` em caminho read-heavy gera overhead desnecessario.
- N+1 por lazy loading implicito — preferir `Include`/projecao explicita.

## Proibido
- SQL por concatenacao de string com input externo.
- `DbContext` com lifetime `Singleton`.
- Dominio importando `Microsoft.EntityFrameworkCore`.
- `SaveChangesAsync()` dentro de repository individual.
