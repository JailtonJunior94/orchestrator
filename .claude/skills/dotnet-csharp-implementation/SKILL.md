---
name: dotnet-csharp-implementation
version: 1.4.0
category: language
prerequisites: [agent-governance]
description: >
  Implementa alteracoes em codigo .NET/C# usando governanca base, arquitetura, estilo,
  testes e padroes recorrentes. Use quando a tarefa exigir adicionar, corrigir, refatorar
  ou validar codigo C# incluindo Minimal APIs, Generic Host, EF Core, MediatR e validacao
  da stack. Nao use para tarefas sem codigo .NET/C#, documentacao geral ou triagem sem
  alteracao.
---

# Implementacao .NET / C#

> Alvo: **.NET 10 (LTS)** + **C# 14**. Adaptar a versao real declarada no `.csproj` /
> `Directory.Build.props` do projeto analisado — nao impor 10 quando o projeto fixar outra LTS.

## Procedimentos

**Etapa 1: Carregar base obrigatoria**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Ler `references/architecture.md`.
3. Ler o `.csproj` (ou `Directory.Build.props` / `Directory.Packages.props`) para identificar
   `<TargetFramework>`, `<LangVersion>`, `<Nullable>` e as dependencias centralizadas.
4. Executar `dotnet --version` para confirmar o SDK disponivel antes de assumir recursos de linguagem.
5. Carregar as **Regras Estritas Obrigatorias (R0-R6)** desta skill. Sao `[HARD]` (bloqueantes de
   merge) salvo quando marcadas `[SOFT]`. Aplicam-se a todo codigo C# de dominio, aplicacao e
   infraestrutura produzido ou modificado, em qualquer camada.

## Regras Estritas Obrigatorias (R0-R6)

> Severidade padrao: toda violacao e `[HARD]` (bloqueante de merge) salvo marcacao explicita `[SOFT]`.
> As regras sao cumulativas e nao tem precedencia entre si. Em conflito com outra orientacao desta
> skill, prevalece a **restricao mais restritiva**. Verificar `<TargetFramework>` / `<LangVersion>`
> no `.csproj` antes de aplicar recursos de linguagem; se a versao for anterior ao recurso, NAO
> usa-lo e registrar a omissao.

- **R0 — Nullable como erro `[HARD]`:** habilitar Nullable Reference Types (`<Nullable>enable</Nullable>`)
  e tratar warnings de nulidade (CS86xx) como erro em projetos novos. Usar `required` em propriedades
  obrigatorias na inicializacao. Proibido `!` (null-forgiving) sem comentario justificando.
- **R1 — Async sem bloqueio `[HARD]`:** propagar `CancellationToken` ate a borda de IO em toda operacao
  assincrona. Proibido `.Result`, `.Wait()` e `.GetAwaiter().GetResult()` — causam deadlock e bloqueiam
  o thread pool. Nao usar `async void` exceto em event handlers.
- **R2 — Imutabilidade por tipo correto `[HARD]`:** usar `record` (ou `record struct`) para DTOs, value
  objects e domain events imutaveis; `class` apenas para entidades de dominio com identidade e
  mutabilidade controlada. Nao usar `record` para entidades com ciclo de vida gerenciado por ORM.
- **R3 — Result em fronteiras esperadas `[HARD]`:** retornar `Result<T>` (ou equivalente) em fronteiras
  de Application/Domain para fluxos esperados (not found, validacao). Reservar exceptions para falhas
  inesperadas de infraestrutura. Nao usar exception como fluxo de controle previsivel.
- **R4 — Dependencias testaveis `[HARD]`:** injetar `TimeProvider` para tempo testavel em vez de
  `DateTime.UtcNow`/`DateTime.Now` hardcoded. Resolver `HttpClient` sempre via `IHttpClientFactory` —
  nunca instanciar com `new HttpClient()`.
- **R5 — Repository com fronteira limpa `[HARD]`:** interface do repository no lado consumidor
  (Application); implementacao concreta em Infrastructure. Proibido expor `IQueryable<T>` fora de
  Infrastructure — materializar e mapear para entidades de dominio.
- **R6 — Testes para todo comportamento `[HARD]`:** toda mudanca de comportamento exige teste xUnit novo
  ou atualizado; mockar por interface com NSubstitute; `TheoryData`/`MemberData` para casos
  parametrizados; sem `DateTime.Now`/`Guid.NewGuid()` nao injetados em assercoes deterministicas.

**Patterns frequentes (inline — evitar carregar `patterns.md` para estes; principios cross-linguagem em `agent-governance/references/shared-patterns.md`)**
- **Factory / Static Factory:** Usar `static T Create(args)` ou `static Result<T> Create(args)` em
  records e classes de dominio quando a construcao exigir invariantes. Retornar `Result<T>` em vez de
  lancar excecao em fronteiras publicas. Nao usar factory abstrata para um unico tipo concreto.
- **Primary Constructor (C# 12+):** Usar para injecao de dependencia em servicos, repositories e
  handlers quando nao houver logica no construtor alem de atribuicao.
  Ex: `public class OrderService(IOrderRepository repo, IUnitOfWork uow) { ... }`.
- **Record como Value Object:** Usar `record`/`record struct` para value objects e DTOs imutaveis.
  Invariantes em construtor estatico ou metodo factory. Nao usar `record` para entidades com ciclo
  de vida gerenciado por ORM.
- **Repository:** Interface no lado consumidor (Application). Implementacao concreta em Infrastructure.
  Nao expor `IQueryable<T>` fora de Infrastructure — materializar e mapear para entidades de dominio.
- **MediatR Command/Query:** `IRequest<TResponse>` para Commands e Queries; handler em arquivo separado
  do Command/Query. Domain events desacoplados via `INotification` + `INotificationHandler<T>`.

**Etapa 2: Selecionar apenas o contexto necessario**
1. Ler `references/conventions.md` quando a tarefa envolver nomeacao, nullable, organizacao de
   arquivos, escolha entre `record`/`class` ou primary constructors.
2. Ler `references/testing.md` quando a tarefa envolver estrategia de testes, xUnit, NSubstitute,
   Bogus, Testcontainers, `WebApplicationFactory` ou cobertura.
3. Ler `references/api.md` quando a tarefa envolver Minimal APIs, Controllers, filtros, Problem
   Details, validacao de request ou versionamento de API.
4. Ler `references/patterns.md` quando a tarefa envolver Specification, Decorator, Strategy,
   pipeline behaviors ou Result pattern. Factory, Primary Constructor, Record e Repository ja estao
   inline acima e NAO devem motivar o carregamento deste arquivo.
5. Ler `references/concurrency.md` quando a tarefa usar `async`/`await`, `CancellationToken`,
   `Channel<T>`, `Parallel`, `Task.WhenAll` ou `IAsyncEnumerable<T>`.
6. Ler `references/resilience.md` quando a tarefa envolver retries, circuit breakers, timeouts,
   hedging ou protecao contra falhas transitorias com Polly v8 / `Microsoft.Extensions.Resilience`.
7. Ler `references/build.md` quando a tarefa envolver Dockerfile, pipeline de CI, analyzers,
   imagem de container, central package management ou gates de qualidade.
8. Ler `references/graceful-lifecycle.md` quando a tarefa envolver `IHostedService`,
   `BackgroundService`, shutdown gracioso, `IHostApplicationLifetime` ou drain de conexoes.
9. Ler `references/examples-domain-flow.md` quando a tarefa precisar de esqueleto concreto de fluxo
   end-to-end (Entity -> Command -> Handler -> Endpoint -> Teste). Para tarefas menores, usar o
   esqueleto inline sem carregar o arquivo completo.
10. Ler `references/examples-testing.md` quando a tarefa precisar de exemplos de `TheoryData`,
    `MemberData`, builders com Bogus ou verificacao de interacao com NSubstitute.
11. Ler `references/examples-infrastructure.md` quando a tarefa precisar de exemplo de graceful
    shutdown, cursor-based pagination, versionamento de API ou outbox processor.
12. Ler `references/configuration.md` quando a tarefa envolver `IOptions<T>`, `IOptionsMonitor<T>`,
    `IOptionsSnapshot<T>`, validacao de config no startup ou tratamento de secrets.
13. Ler `references/persistence.md` quando a tarefa envolver EF Core, Dapper, migrations, Unit of
    Work, queries ou connection management.
14. Ler `references/observability.md` quando a tarefa envolver logging estruturado, OpenTelemetry,
    `Activity`, `Meter` ou health checks.
15. Ler `references/security.md` quando a tarefa envolver autenticacao, autorizacao, validacao de
    input, rate limiting, CORS, data protection ou tratamento de segredos.
16. Ler `references/messaging.md` quando a tarefa envolver producao ou consumo de mensagens, eventos,
    filas, outbox pattern, idempotencia de consumidores ou sagas.
17. Ler `references/architecture.md` (alem da Etapa 1) quando a tarefa exigir decisao de layout de
    projeto (Web API, Worker, gRPC, CLI) ou fronteiras de camada.

**Economia de contexto**
Classificar a complexidade da tarefa (trivial / standard / complex) conforme
`agent-governance/SKILL.md` antes de carregar referencias, e respeitar o teto correspondente:
- **trivial** (rename, typo, import, formatacao): nenhuma referencia — apenas esta SKILL.md.
- **standard** (metodo novo, fix local, refactor local): no maximo o TL;DR das 1-2 referencias
  diretamente ligadas a superficie alterada (o bloco `<!-- TL;DR -->` no topo de cada referencia).
- **complex** (feature, interface publica, migracao): carregar referencias completas sob demanda.
Se mais de 4 referencias forem necessarias para a mesma tarefa, priorizar as 3 mais criticas para o
escopo da mudanca e registrar as demais como contexto nao carregado. Carregar referencias adicionais
apenas se a implementacao revelar necessidade concreta.

**Etapa 3: Modelar a alteracao**
1. Identificar o menor conjunto seguro de mudancas que satisfaz a solicitacao.
2. Mapear o comportamento afetado, as dependencias envolvidas e o risco de regressao.
3. Preferir `record` para DTOs/value objects e `class` para entidades mutaveis.
4. Introduzir interface apenas quando existir fronteira consumidora real, necessidade de substituicao
   ou ponto claro de teste.
5. Respeitar as convencoes do projeto (Controllers vs Minimal APIs, MediatR vs chamada direta) antes
   de propor uma alternativa.

**Etapa 4: Implementar**
1. Editar o codigo seguindo o `<TargetFramework>` / `<LangVersion>` declarados e as convencoes do contexto.
2. Manter XML doc apenas em membros publicos de biblioteca; evitar em codigo de aplicacao.
3. Atualizar ou adicionar testes para toda mudanca de comportamento.
4. Adaptar exemplos ao contexto real em vez de replica-los literalmente.

**Etapa 5: Validar**
1. Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md`.
2. Em .NET, preferir nesta ordem quando disponiveis no projeto:
   - `dotnet build --no-restore`
   - `dotnet test --no-build`
   - `dotnet format --verify-no-changes`
   - `dotnet-csharpier --check` (alternativa opinionada de formatacao)
3. Executar o **Checklist de Validacao (R0-R6)** e reportar o resultado de cada gate. Qualquer item com
   resultado diferente do esperado e `[HARD]` — bloqueante de merge:
   - **Build (R0):** `dotnet build --no-restore` sem warning de nulidade (CS86xx) em projeto novo
     (`TreatWarningsAsErrors`/`<Nullable>enable</Nullable>`).
   - **Async (R1):** grep por regressao: `grep -rnE '\.Result|\.Wait\(\)|\.GetAwaiter\(\)\.GetResult\(\)|new HttpClient\(' --include='*.cs'` deve vir vazio ou justificado.
   - **Testes (R6):** `dotnet test --no-build` verde; em solucao multi-projeto, apenas os projetos afetados.
   - **Format:** `dotnet format --verify-no-changes` (ou `dotnet-csharpier --check`) sem diferencas.
   Se um comando nao existir no projeto, registrar a ausencia explicitamente em vez de inventar substituto.

## Tratamento de Erros
* Se o `.csproj` estiver ausente, parar antes de assumir framework ou dependencias.
* Em solucao multi-projeto (`.sln`), validar apenas os projetos afetados.
* Se o contexto nao fornecer comando de teste, lint ou formatter, registrar a ausencia explicitamente
  em vez de inventar substitutos.
* Se mais de uma abordagem parecer plausivel, preferir a alternativa com menos tipos, menos indirecao
  e menor custo de teste.
* Se houver conflito entre esta skill e a governanca base, seguir a restricao mais segura e registrar
  a suposicao.
