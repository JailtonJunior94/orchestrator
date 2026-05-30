# Seguranca .NET / C#

<!-- TL;DR
Seguranca em .NET: JWT Bearer com validacao explicita, authorization policies, data protection, validacao de input, CORS, rate limiting e security headers.
Keywords: jwt, bearer, authorization, policy, data protection, cors, rate limiting, hsts, https, input validation
Load complete when: a tarefa envolve autenticacao, autorizacao, validacao de input, rate limiting, CORS ou tratamento de segredos.
-->

## Objetivo

Definir as praticas minimas de seguranca para APIs e servicos .NET.

## Diretrizes

### Autenticacao JWT Bearer
- `AddJwtBearer()` com validacao explicita de `Issuer`, `Audience`, `IssuerSigningKey` e `ClockSkew`:
  ```csharp
  builder.Services.AddAuthentication().AddJwtBearer(o =>
  {
      o.TokenValidationParameters = new()
      {
          ValidateIssuer = true,
          ValidateAudience = true,
          ValidateLifetime = true,
          ValidateIssuerSigningKey = true,
          ClockSkew = TimeSpan.FromSeconds(30)
      };
  });
  ```
- Algoritmos assimetricos (RS256, ES256) quando multiplos servicos validam tokens; usar `kid` para
  rotacao de chave sem downtime.

### Autorizacao
- Policies explicitas com `AddAuthorizationBuilder()`:
  `builder.Services.AddAuthorizationBuilder().AddPolicy("admin", p => p.RequireRole("Admin"));`.
- `IAuthorizationService` para autorizacao imperativa dentro de use cases.
- Principio de menor privilegio — exigir autenticacao por default, `[AllowAnonymous]` como opt-out explicito.

### Data Protection
- `AddDataProtection()` com chave persistida externamente (Azure Blob, Redis) em producao.
- Usar para tokens internos e cookies — nao para criptografia de dados de negocio.

### Validacao e sanitizacao de input
- Nunca confiar em input do cliente para decisoes de autorizacao.
- Limitar tamanho de payload (`MaxRequestBodySize` / limites de Kestrel).
- Em APIs REST que devolvem HTML, codificar com `HtmlEncoder.Default`.

### CORS
- Origins explicitas em producao — nunca `AllowAnyOrigin()`.
- `AllowCredentials()` apenas com origins especificas.

### Rate Limiting (.NET 7+)
- `AddRateLimiter()` com politicas nomeadas por recurso (sliding window/token bucket para endpoints
  publicos; fixed window para login).

### HTTPS e headers
- `UseHsts()` em producao; `UseHttpsRedirection()`.
- Security headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`.

## Riscos Comuns
- `ClockSkew` no default (5 min) aceita tokens expirados por margem ampla.
- Decisao de autorizacao cacheada perde revogacao de permissao.

## Proibido
- Segredo hardcoded em `.cs` ou `appsettings.json` commitado.
- SQL por concatenacao de string com input externo.
- Ignorar erros de certificado TLS em producao.
- Expor stack trace, mensagem interna ou estrutura do banco em resposta de erro.
- `[AllowAnonymous]` como default global.
