# Configuracao .NET / C#

<!-- TL;DR
Configuracao em .NET: IOptions vs IOptionsMonitor vs IOptionsSnapshot, validacao no startup com ValidateOnStart, user-secrets e gestao de ambientes.
Keywords: ioptions, optionsmonitor, optionssnapshot, BindConfiguration, ValidateOnStart, user-secrets, appsettings, ASPNETCORE_ENVIRONMENT
Load complete when: a tarefa envolve carregamento de configuracao, variaveis de ambiente, secrets ou inicializacao de opcoes.
-->

## Objetivo

Definir como modelar, validar e consumir configuracao tipada no .NET.

## Diretrizes

### IOptions vs IOptionsMonitor vs IOptionsSnapshot
- `IOptions<T>`: Singleton, lido uma vez na inicializacao — para config imutavel durante a vida do app.
- `IOptionsSnapshot<T>`: Scoped, reavaliado por request — para config que pode mudar entre requests.
- `IOptionsMonitor<T>`: Singleton com notificacao de mudanca (`OnChange`) — para config dinamica
  consumida por servicos Singleton.

### Validacao no startup (fail-fast)
```csharp
builder.Services.AddOptions<DatabaseOptions>()
    .BindConfiguration("Database")
    .ValidateDataAnnotations()
    .Validate(o => o.MaxPoolSize > 0, "MaxPoolSize deve ser positivo")
    .ValidateOnStart();
```
- `ValidateOnStart()` faz a aplicacao falhar na inicializacao se a config for invalida, evitando erro
  tardio em runtime.

### Secrets
- Desenvolvimento: `dotnet user-secrets set "ConnectionStrings:Default" "..."`.
- Producao: variaveis de ambiente ou Azure Key Vault / AWS Secrets Manager via configuration provider.
- Nunca commitar `appsettings.Production.json` com segredos reais.

### Ambientes
- `appsettings.json` (defaults) + `appsettings.{Environment}.json` (override por ambiente).
- `ASPNETCORE_ENVIRONMENT` (web) ou `DOTNET_ENVIRONMENT` (worker/CLI) controla o ambiente ativo.

## Riscos Comuns
- Injetar `IOptionsSnapshot<T>` (Scoped) em servico Singleton — captive dependency.
- Esquecer `ValidateOnStart` e descobrir config invalida apenas no primeiro uso em runtime.

## Proibido
- Segredo hardcoded em `.cs` ou `appsettings.json` commitado.
- Ler `IConfiguration` cru espalhado pelo codigo em vez de tipar via `IOptions<T>`.
- Commitar connection string de producao com credenciais.
