# Convencoes .NET / C#

<!-- TL;DR
Convencoes de codigo C#: nomeacao, Nullable Reference Types, records vs classes, primary constructors, organizacao de arquivos e using declarations.
Keywords: nomeacao, nullable, record, class, primary constructor, required, file-scoped namespace, using
Load complete when: a tarefa envolve nomeacao, escolha entre record/class, nulidade ou organizacao de arquivos.
-->

## Objetivo

Definir as convencoes de estilo e linguagem adotadas em codigo C# moderno (.NET 10 / C# 14).

## Diretrizes

### Nomeacao
- `PascalCase` para tipos, metodos, propriedades e membros publicos.
- `camelCase` para variaveis locais e parametros.
- `_camelCase` para campos privados de instancia.
- Prefixo `I` apenas para interfaces (`IOrderRepository`).
- Sem prefixo hungaro; sem sufixo redundante (`OrderManagerManager`).

### Nullable Reference Types
- `<Nullable>enable</Nullable>` em todos os projetos novos.
- Tratar avisos de nulidade (CS8600–CS8625) como erros em producao:
  `<TreatWarningsAsErrors>true</TreatWarningsAsErrors>` ou `<WarningsAsErrors>nullable</WarningsAsErrors>`.
- Usar `required` em propriedades que devem ser definidas na inicializacao do objeto:
  `public required string Name { get; init; }`.
- Evitar `!` (null-forgiving) — usar apenas quando uma invariante garante o valor e documentar o motivo.

### Records vs Classes
- `record` (ou `record struct`) para value objects, DTOs de request/response e domain events.
- `class` para entidades de dominio com identidade e mutabilidade controlada.
- Nao usar `record` para entidades com ciclo de vida gerenciado por ORM (igualdade estrutural
  conflita com identidade por chave).
- `readonly record struct` para value objects pequenos e imutaveis sem alocacao no heap.

### Primary Constructors (C# 12+)
- Usar em servicos, handlers e repositorios para injecao limpa:
  `public sealed class OrderService(IOrderRepository repo) { ... }`.
- Nao usar quando houver logica no corpo do construtor alem de atribuicao (validacao, normalizacao).

### Recursos de C# 14 relevantes
- `field` keyword: acessor de propriedade sem declarar backing field explicito —
  `public string Name { get => field; set => field = value.Trim(); }`.
- Null-conditional assignment: `customer?.Order = newOrder;`.
- Extension members (propriedades e membros estaticos de extensao) quando estender tipos de forma
  expressiva sem poluir a API original.

### Organizacao de arquivos
- Um tipo publico por arquivo; arquivo nomeado igual ao tipo principal.
- Excecao: tipos privados auxiliares pequenos podem coexistir no mesmo arquivo.
- File-scoped namespaces (`namespace Foo.Bar;`) para reduzir indentacao.
- `global using` em `GlobalUsings.cs` para namespaces ubiquos do projeto.

### using declarations
- Preferir `using var resource = ...;` sobre `using (...) {}` em escopos simples.
- `await using` para `IAsyncDisposable` (ex: `DbContext`, streams assincronos).

## Riscos Comuns
- `record` para entidade ORM gera comportamento de igualdade inesperado em change tracking.
- Desabilitar nullable em um projeto e habilitar em outro cria fronteiras de anotacao inconsistentes.

## Proibido
- Suprimir avisos de nulidade globalmente com `#nullable disable` em codigo novo.
- `var` quando o tipo nao for obvio pelo lado direito e prejudicar a leitura.
- Mais de um tipo publico de topo por arquivo.
