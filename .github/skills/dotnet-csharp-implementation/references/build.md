# Build e CI .NET / C#

<!-- TL;DR
Build em .NET: Dockerfile multistage com imagem chiseled e USER nao-root, pipeline GitHub Actions, analyzers Roslyn e central package management.
Keywords: dockerfile, chiseled, multistage, app_uid, github actions, dotnet publish, analyzers, TreatWarningsAsErrors, central package management
Load complete when: a tarefa envolve Dockerfile, pipeline de CI, imagem de container, analyzers ou gates de qualidade.
-->

## Objetivo

Definir build reprodutivel, empacotamento seguro em container e gates de qualidade para .NET.

## Diretrizes

### Dockerfile multistage com .NET 10
```dockerfile
FROM mcr.microsoft.com/dotnet/sdk:10.0 AS build
WORKDIR /src
COPY ["src/Api/Api.csproj", "src/Api/"]
RUN dotnet restore "src/Api/Api.csproj"
COPY . .
RUN dotnet publish "src/Api/Api.csproj" -c Release -o /app/publish

FROM mcr.microsoft.com/dotnet/aspnet:10.0-noble-chiseled AS runtime
WORKDIR /app
COPY --from=build /app/publish .
USER $APP_UID
ENTRYPOINT ["dotnet", "Api.dll"]
```
- Imagem **chiseled** (Ubuntu minimal) reduz a superficie de ataque e o tamanho.
- `USER $APP_UID` executa como nao-root.
- `.dockerignore` excluindo `bin/`, `obj/`, `.git/`.

### GitHub Actions — pipeline minima
```yaml
- run: dotnet restore
- run: dotnet build --no-restore -c Release
- run: dotnet test --no-build -c Release --collect:"XPlat Code Coverage"
- run: dotnet publish --no-build -c Release -o publish/
```

### Analise estatica e qualidade
- `dotnet format --verify-no-changes` como gate de CI.
- `Microsoft.CodeAnalysis.NetAnalyzers` habilitado (`<EnableNETAnalyzers>true</EnableNETAnalyzers>`).
- `<TreatWarningsAsErrors>true</TreatWarningsAsErrors>` em projetos de producao.
- `dotnet-csharpier --check` como formatacao deterministica opinionada (alternativa).

### NuGet e dependencias
- Central Package Management: `<ManagePackageVersionsCentrally>true</ManagePackageVersionsCentrally>`
  com `Directory.Packages.props` (`<PackageVersion>` por pacote).
- `Directory.Build.props` para propriedades comuns (TargetFramework, Nullable, LangVersion).
- `dotnet list package --vulnerable` em CI para detectar CVEs conhecidos.

## Riscos Comuns
- Copiar `bin/`/`obj/` para a imagem (sem `.dockerignore`) infla o contexto e vaza artefatos.
- Rodar como root no container amplia o impacto de uma vulnerabilidade.

## Proibido
- Executar container como root quando `$APP_UID` esta disponivel.
- Imagem de runtime com SDK completo (usar `aspnet`/`runtime` chiseled).
- Suprimir warnings de analyzer globalmente em vez de corrigir a causa.
