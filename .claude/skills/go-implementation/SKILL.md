---
name: go-implementation
version: 1.3.0
category: language
description: Implementa alteracoes em codigo Go usando governanca base, arquitetura, estilo, testes e padroes recorrentes. Use quando a tarefa exigir adicionar, corrigir, refatorar ou validar codigo Go, incluindo interfaces, generics, concorrencia e validacao da stack. Nao use para tarefas sem codigo Go, documentacao geral ou triagem sem alteracao.
---

# Implementacao Go

## Procedimentos

**Etapa 1: Carregar base obrigatoria**
1. Confirmar que o contrato de carga base definido em `AGENTS.md` foi cumprido.
2. Ler `references/architecture.md`.
3. Executar `bash scripts/verify-go-mod.sh`.
4. Ler `go.mod` quando ele existir no contexto analisado.
5. Carregar as **Regras Estritas Obrigatorias (Regras 0-7)** desta skill. Sao `[HARD]`
   (bloqueantes de merge) salvo quando marcadas `[SOFT]`. Aplicam-se a todo codigo Go de
   dominio, aplicacao e infraestrutura produzido ou modificado, em qualquer camada.

## Regras Estritas Obrigatorias (Regras 0-7)

> Severidade padrao: toda violacao e `[HARD]` (bloqueante de merge) salvo marcacao explicita
> `[SOFT]`. As regras sao cumulativas — nao ha precedencia entre elas. Em conflito com qualquer
> outra orientacao desta skill, prevalece a **restricao mais restritiva**.
> Fonte: `docs/prompts/go-implementation-strict-rules.md` + [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md).

### Indice das regras e onde cada uma e detalhada

| Regra | Tema | Onde aplicar / detalhamento |
|-------|------|-----------------------------|
| R0 | `init()` PROIBIDA | `references/architecture.md` |
| R1 | Toda funcao deve ser metodo de struct (excecoes exaustivas) | `references/architecture.md` |
| R2 | Proibir atribuicao direta de campo sem transformacao | inline abaixo |
| R3 | Mocks via `mockery.yml` obrigatorio | `references/testing.md` |
| R4 | Padrao `testify/suite` table-driven nos testes | `references/testing.md`, `references/examples-testing.md` |
| R5 | Uber Go Style Guide (PT-BR) | inline abaixo + `interfaces.md`, `concurrency.md`, `persistence.md`, `api.md`, `configuration.md` |
| R6 | Design e contratos Go (context, interface no consumidor, DI) | `references/interfaces.md`, `references/architecture.md` |
| R7 | Sempre usar recursos modernos do Go (versao do `go.mod`) | inline abaixo + `generics.md`, `observability.md` |
| Checklist | Gates de validacao R0-R7 | `references/build.md` + Etapa 5 |

### R2 — Proibir atribuicao direta de campo sem transformacao `[HARD]`

E proibido extrair um campo de struct para variavel local que apenas o renomeia, sem
transformacao real (parse, conversao de tipo, sanitizacao, calculo, formatacao). Variaveis
locais que apenas "desacucaram" um campo sao ruido e geram inconsistencia quando o campo evolui.

```go
// PROIBIDO — alias de campo sem transformacao
nome := user.Name
email := user.Email
return &dtos.UserOutput{Name: nome, Email: email}, nil

// CORRETO — usar o campo diretamente
return &dtos.UserOutput{Name: user.Name, Email: user.Email}, nil
```

Permitido quando ha transformacao real: `userID, err := strconv.ParseInt(input.UserID, 10, 64)`,
`code := strings.ToUpper(input.Code)`, retorno de funcao (`user, err := repo.FindByID(...)`),
shadow de context com timeout. Em code review, apontar toda copia direta de campo como `[HARD]`.

### R5 — Catalogo Uber Go Style Guide (PT-BR) `[HARD]` salvo indicacao

Regras topicas estao nas referencias (ver indice). As regras transversais abaixo aplicam-se a
todo codigo Go e devem ser verificadas sempre:

- **5.8** Enums com `iota` comecam em 1 (`iota + 1`) — zero value reservado para "nao inicializado",
  salvo quando o zero value tiver semantica de default desejavel.
- **5.10** Erros: `errors.New` (estatico, sem comparacao), `fmt.Errorf("ctx: %v", ...)` (dinamico,
  sem comparacao), `var ErrNome = errors.New(...)` (caller usa `errors.Is`), tipo customizado
  `NomeError` (caller usa `errors.As`). Wrapping com `%w` quando o caller precisa de `errors.Is/As`;
  contexto sucinto em PT-BR (`"criar usuario: %w"`, nunca `"failed to..."`). Tratar erro **uma unica
  vez** — nao logar e retornar o mesmo erro.
- **5.11** Type assertion sempre com comma-ok (`t, ok := i.(T)`), nunca `t := i.(T)`.
- **5.12** Sem `panic` em codigo de producao — retornar erro. Excecao: `template.Must`/`regexp.MustCompile`
  apenas em inicializacao de `main`. Em testes usar `t.Fatal`/`t.FailNow`, nunca `panic`.
- **5.15** Nao usar nomes built-in (`error`, `string`, `len`, ...) como identificadores.
- **5.19** Preferir `strconv` a `fmt` para conversoes primitivas (`strconv.Itoa(n)`).
- **5.20** Especificar capacidade de slices/maps quando conhecida (`make([]T, 0, len(items))`).
- **5.21/5.22** Reduzir aninhamento com early return/continue; evitar `else` desnecessario.
- **5.23** Imports em 3 grupos (stdlib | externas | internas do modulo), via `goimports -local`.
- **5.24** Nomes de pacote: minusculo, sem underscore, singular, especifico (`payment`, nao `util`/`common`).
- **5.25** Ordenacao no arquivo: tipos -> var/const -> `New*` -> metodos exportados -> nao exportados.
- **5.26** Globais nao exportados com prefixo `_` (`const _defaultTimeout`); excecao: erros sentinel (`var errX`).
- **5.27-5.31** Inicializar struct com nomes de campos; omitir campos zero-value; `var x T` para
  struct zero-value; `&T{...}` em vez de `new(T)`; `make(map...)` para maps populados programaticamente.
- **5.36-5.45** Hot path: converter `string->[]byte` uma vez; `var` para zero-values e slices que
  recebem append; `nil` e slice valido (usar `len()` para vazio). Raw string literals para evitar escape.
- **5.47-5.49** Format strings fora de Printf devem ser `const`; funcoes Printf-style terminam com `f`.
- **5.50** Functional Options para structs com mais de 3 campos opcionais (ver "Patterns frequentes").
- **5.37** `[SOFT]` Limite suave de 99 caracteres por linha.

### R7 — Sempre usar recursos modernos do Go `[HARD]`

> Antes de aplicar qualquer recurso, **verificar a versao em `go.mod`**. Se a versao declarada for
> anterior ao recurso, NAO usa-lo e registrar: `"recurso X requer Go Y; go.mod declara Go Z — nao aplicado"`.

- **7.1** `any` em vez de `interface{}` (Go 1.18) — detalhe em `generics.md`.
- **7.2** `log/slog` para logging estruturado (Go 1.21) — detalhe em `observability.md`.
- **7.3** Pacote `slices` em vez de loops manuais de colecao (Go 1.21).
- **7.4** Pacote `maps` em vez de loops manuais de mapa (Go 1.21).
- **7.5** Builtins `min`, `max`, `clear` (Go 1.21).
- **7.6** `errors.Join` para agregar erros (Go 1.20) — nunca concatenacao de strings.
- **7.7** Range sobre inteiros `for i := range n` (Go 1.22) quando so o indice importa.
- **7.8** Generics para eliminar duplicacao identica (Go 1.18) — detalhe em `generics.md`.
- **7.9** Pacote `cmp` (`cmp.Compare`, `cmp.Or`) para comparacao/ordenacao type-safe (Go 1.21).
- **7.10** `sync.OnceValue`/`sync.OnceValues` apenas em `main`/inicializacao — detalhe em `configuration.md`.
- **7.11** Iteradores `iter.Seq`/`iter.Seq2` para colecoes grandes ou lazy (Go 1.23).


- Preferir nomes curtos e precisos no escopo certo.
- Retornar erros como ultimo valor e usar wrapping com `%w` quando necessario.
- Manter funcoes pequenas quando isso melhorar leitura e teste, sem fragmentacao artificial.
- Preferir zero values uteis e construtores apenas quando houver invariantes ou dependencias obrigatorias.
- Usar `context.Context` em fronteiras de IO, rede, subprocesso ou operacoes cancelaveis.
- Usar sentinel errors (`var ErrNotFound = errors.New(...)`) para erros que o chamador compara com `errors.Is`.
- Usar tipos customizados (`type ValidationError struct{...}`) quando o chamador precisar extrair dados com `errors.As`.
- Formatar com `gofmt`. Usar `go test ./...` como gate minimo.

**Patterns frequentes (inline — evitar carregar patterns.md para estes; principios cross-linguagem em `agent-governance/references/shared-patterns.md`)**
- **Factory Function:** Usar `New*(deps...) (*T, error)` quando construcao exigir validacao de invariantes ou dependencias obrigatorias. Retornar `(T, error)` ou `*T`. Nao usar factory abstrata para um unico tipo concreto.
- **Functional Options:** Usar `func With*(v) Option` quando o objeto tiver muitos campos opcionais. Preferir sobre builder fluente: `func NewServer(addr string, opts ...ServerOption) *Server`. Cada option e uma `func(*T)` que modifica o alvo.
- **Adapter:** Usar struct que implementa interface do consumidor e delega para tipo externo quando integrar dependencia incompativel. Repository concreto e o exemplo mais comum.
- **Decorator:** Struct/funcao que wrapa interface e adiciona comportamento transversal (logging, metricas, retry). Ex: `type loggingRepo struct{ next Repo; log *slog.Logger }` — cada metodo loga e delega para `next`.
- **Facade:** Service/use case que orquestra multiplas dependencias em operacao de alto nivel. Ex: `func (s *Service) Checkout(ctx, id)` chama orders, payments, notify em sequencia com tratamento de erro.

**Etapa 2: Selecionar apenas o contexto necessario**
1. Ler `references/interfaces.md` quando a tarefa introduzir, remover ou remodelar interfaces, construtores ou fronteiras de dependencia.
2. Ler `references/generics.md` quando a tarefa introduzir ou alterar parametros de tipo, constraints ou componentes reutilizaveis com generics.
3. Ler `references/concurrency.md` quando a tarefa usar goroutines, channels, cancelamento, worker pools ou sincronizacao.
4. Ler `references/patterns-structural.md` **somente** quando a tarefa envolver Singleton, patterns raramente uteis (Abstract Factory, Prototype, Flyweight) ou codigo de exemplo completo dos patterns. Factory Function, Functional Options, Adapter, Decorator e Facade ja estao definidos na secao "Patterns frequentes" acima e NAO devem motivar o carregamento deste arquivo — isso evita ~960 tokens redundantes. O arquivo e mantido como referencia historica.
5. Ler `references/patterns-behavioral.md` quando a tarefa envolver strategy, chain of responsibility, observer/eventos, maquina de estado ou template method.
6. Ler `references/observability.md` quando a tarefa envolver logging, tracing, metricas ou health checks.
7. Ler `references/api.md` quando a tarefa envolver handlers HTTP/gRPC, middlewares, DTOs ou serializacao.
8. Ler `references/persistence.md` quando a tarefa envolver repositories, transactions, migrations, queries ou connection management.
9. Ler `references/configuration.md` quando a tarefa envolver carregamento de configuracao, variaveis de ambiente ou inicializacao de dependencias.
10. Ler `references/resilience.md` quando a tarefa envolver retries, circuit breakers, timeouts em chamadas externas, fallbacks ou protecao contra falhas transitorias.
11. Ler `references/messaging.md` quando a tarefa envolver producao ou consumo de mensagens, eventos, filas, topicos, outbox pattern ou idempotencia de consumidores.
12. Ler `references/security.md` quando a tarefa envolver autenticacao, autorizacao, validacao de input, rate limiting, CORS ou tratamento de segredos.
13. Ler `references/testing.md` quando a tarefa envolver estrategia de testes, integration tests, testcontainers, fixtures ou cobertura.
14. Ler `references/examples-domain-flow.md` quando a tarefa precisar de esqueleto concreto de fluxo end-to-end (dominio, service, handler, teste com suite e mockery). Para tarefas menores, usar o esqueleto inline: `Entity -> Service(deps) -> Handler(service) -> test com suite/mockery`, sem carregar o arquivo completo.
15. Ler `references/examples-testing.md` quando a tarefa precisar de exemplos de fuzz test, table-driven test, construtor com invariantes ou interface no consumidor.
16. Ler `references/examples-infrastructure.md` quando a tarefa precisar de exemplo de graceful shutdown, paginacao cursor-based ou versionamento de API.
17. Ler `references/build.md` quando a tarefa envolver Dockerfile, Makefile, pipeline de CI, build flags, imagem de container ou gates de qualidade.
18. Ler `references/graceful-lifecycle.md` quando a tarefa envolver inicializacao ordenada, shutdown gracioso, handler de sinais, drain de conexoes ou encerramento de goroutines de longa duracao.

**Economia de contexto**
Se mais de 4 referencias forem necessarias para a mesma tarefa, priorizar as 3 mais criticas para o escopo da mudanca e registrar as demais como contexto nao carregado. Carregar referencias adicionais apenas se a implementacao revelar necessidade concreta.

**Etapa 3: Modelar a alteracao**
1. Identificar o menor conjunto seguro de mudancas que satisfaz a solicitacao.
2. Mapear o comportamento afetado, as dependencias envolvidas e o risco de regressao.
3. Preferir tipos concretos por padrao.
4. Introduzir interface apenas quando existir fronteira consumidora real, necessidade de substituicao ou ponto claro de teste.
5. Aplicar pattern apenas quando ele reduzir acoplamento, branching recorrente ou ambiguidade arquitetural.

**Etapa 4: Implementar**
1. Editar o codigo seguindo a versao Go declarada em `go.mod` e as convencoes do contexto analisado.
2. Manter comentarios curtos e apenas quando agregarem contexto real.
3. Atualizar ou adicionar testes para toda mudanca de comportamento.
4. Adaptar exemplos ao contexto real em vez de replica-los literalmente.

**Etapa 5: Validar**
1. Seguir Etapa 4 de `.agents/skills/agent-governance/SKILL.md`.
2. Em Go, preferir `gofmt` como formatter e `golangci-lint run` como lint quando disponiveis.
3. Executar o **Checklist de Validacao (R0-R7)** documentado em `references/build.md` e reportar
   o resultado de cada gate. Qualquer item com resultado diferente do esperado e `[HARD]` —
   bloqueante. Incluir os greps de R0 (`init()`), R1 (funcoes standalone), R5 (`os.Exit`/`log.Fatal`
   fora de `main`, `panic` fora de inicializacao) e R7 (`interface{}`), alem de
   `go build ./...`, `go vet ./...`, `go test ./... -count=1 -race` e `mockery --config mockery.yml --dry-run`.

## Tratamento de Erros
* Se `go.mod` estiver ausente, parar antes de assumir versao de Go ou dependencias.
* Se o contexto nao fornecer comando de teste, lint ou formatter, registrar a ausencia explicitamente em vez de inventar substitutos.
* Se mais de uma abordagem parecer plausivel, preferir a alternativa com menos tipos, menos indirecao e menor custo de teste.
* Se houver conflito entre esta skill e a governanca base, seguir a restricao mais segura e registrar a suposicao.
