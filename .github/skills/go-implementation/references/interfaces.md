# Interfaces

<!-- TL;DR
Diretrizes para definição e uso de interfaces em Go: quando criar, quando evitar e como posicionar no pacote consumidor para reduzir acoplamento.
Keywords: interface, acoplamento, repository, substituição, consumidor, pacote
Load complete when: tarefa envolve criação, revisão ou posicionamento de interfaces em pacotes Go.
-->

## Quando usar
- Quando existir mais de uma implementação real ou um ponto claro de substituição.
- Quando um consumidor depender apenas de um comportamento pequeno e estável.
- Quando a interface reduzir acoplamento em uma fronteira real (repository, client externo, clock, ID generator).

## Quando evitar
- Para "facilitar testes" sem necessidade real de substituição.
- Antes de existir consumidor ou segunda implementação.
- Quando um tipo concreto simples resolve o problema.
- Quando a interface espelha 1:1 o tipo concreto sem abstrair nada.

## Diretrizes
- Definir a interface no lado consumidor (accept interfaces, return structs).
- Manter interfaces pequenas e focadas — 1 a 3 métodos é o ideal.
- Nomear pelo comportamento, não pela implementação: `Reader`, não `FileReader`.
- Compor interfaces pequenas em vez de criar uma grande: `ReadWriter` = `Reader` + `Writer`.
- Interface não exportada (minúscula) quando o consumidor for interno ao pacote.
- Exportar interface apenas quando consumidores externos precisarem implementá-la.

## Padrões de aplicação

### Fronteira de IO (repository, client)
```go
// application/order/service.go — interface no consumidor
type orderRepository interface {
    Save(ctx context.Context, order *domain.Order) error
    FindByID(ctx context.Context, id string) (*domain.Order, error)
}
```

### Abstração de infraestrutura (clock, ID)
```go
// application/order/service.go
type clock interface {
    Now() time.Time
}

type idGenerator interface {
    New() string
}
```

### Composição de interfaces
```go
type Reader interface {
    Read(ctx context.Context, id string) (*Entity, error)
}

type Writer interface {
    Save(ctx context.Context, entity *Entity) error
}

type ReadWriter interface {
    Reader
    Writer
}
```

## Regra 6 — Design e Contratos `[HARD]`

Complementa as diretrizes acima com o numeração das Regras Estritas (ver SKILL.md).

### R6.1 — `context.Context` obrigatório em fronteiras de I/O
Todo método que faz I/O (rede, banco, arquivo, subprocess, operação cancelável) DEVE receber
`context.Context` como **primeiro parâmetro**. Nunca armazenar `Context` em campo de struct.
Propagar o context recebido — `context.Background()`/`context.TODO()` apenas em `main()`,
inicialização de servidor e testes.

```go
// PROIBIDO — context em struct
type Repo struct { ctx context.Context; db *sql.DB }
// PROIBIDO — I/O sem context
func (r *Repo) FindByID(id int64) (*Entity, error) {}

// CORRETO — context como primeiro parâmetro
func (r *Repo) FindByID(ctx context.Context, id int64) (*Entity, error) {}
```

### R6.2 — Tipos concretos por padrão; interface sob demanda real
Introduzir interface apenas quando houver múltiplas implementações em produção, necessidade real de
substituição em teste, ou fronteira de pacote onde o consumidor não deve depender do concreto.
Ver "Quando usar" / "Quando evitar" acima.

### R6.3 — Interface definida no pacote consumidor
Declarar a interface no pacote que a **consome**, não no que a implementa (accept interfaces, return
structs). Exceção: interface compartilhada por múltiplos consumidores pode residir em `pkg/`
dedicado — nunca em `internal/` do produtor.

### R6.5 — Erros sentinel vs tipo customizado — decisão explícita
A escolha deve ser explícita e baseada nas necessidades do caller (complementa R5.10). Erros
exportados passam a fazer **parte da API pública** — documentá-los.

| Caller usa `errors.Is`? | Caller usa `errors.As`? | Mensagem | Use |
|---|---|---|---|
| Não | Não | Estática | `errors.New(...)` inline |
| Não | Não | Dinâmica | `fmt.Errorf("ctx: %v", ...)` |
| Sim | Não | Estática | `var ErrNome = errors.New(...)` exportado |
| Sim | Sim | Dinâmica | `type NomeError struct{ ... }` exportado |

## Riscos Comuns
- Interface com 5+ métodos que nenhum consumidor usa inteiramente.
- Interface definida no pacote do implementador em vez do consumidor.
- Interface prematura que precisa mudar a cada nova funcionalidade.

## Proibido
- Interface sem consumidor real.
- Interface que replica a struct pública método a método sem abstrair.
- `interface{}` / `any` como substituto de modelagem de domínio.
