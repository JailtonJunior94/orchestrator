# Prompt Enriquecido — Regras Estritas para go-implementation

> **Gerado por:** skill `prompt-enricher`
> **Destino:** skill `go-implementation` — extensão de regras obrigatórias
> **Escopo:** todo código Go implementado ou revisado pela skill go-implementation
> **Idioma de saída do agente:** PT-BR (comentários, mensagens de erro, logs)
> **Referência base:** [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

---

## Contexto

Este prompt define restrições obrigatórias e não negociáveis que se sobrepõem às regras existentes
da skill `go-implementation`. Toda tarefa de implementação Go que as viole deve ser bloqueada e
corrigida antes de ser considerada concluída.

As regras abaixo se aplicam a **qualquer código Go de domínio, aplicação e infraestrutura**
produzido ou modificado pela skill, independentemente da camada (entity, use case, repository,
handler, service, adapter).

---

## Regra 0 — `init()` é PROIBIDA

### Definição

A função `init()` é **terminantemente proibida** em qualquer arquivo Go de produção, teste ou
biblioteca deste projeto. Sem exceções.

### Por que proibir

Conforme o [Uber Go Style Guide — Avoid `init()`](https://github.com/uber-go/guide/blob/master/style.md#avoid-init):

- `init()` introduz ordem de execução implícita e não determinística entre pacotes
- Acessa estado global, variáveis de ambiente ou realiza I/O sem controle explícito
- Impossibilita testes unitários do código de inicialização
- Cria dependência oculta entre pacotes via efeitos colaterais
- Goroutines em `init()` não têm mecanismo de shutdown → goroutine leak garantido

### Como substituir

```go
// ❌ PROIBIDO
var _db *sql.DB

func init() {
    _db, _ = sql.Open("postgres", os.Getenv("DATABASE_URL"))
}

// ✅ CORRETO — injeção explícita via construtor
type UserRepository struct {
    db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
    return &UserRepository{db: db}
}

// ❌ PROIBIDO — default global via init
var _defaultConfig Config

func init() {
    _defaultConfig = Config{Timeout: 30 * time.Second}
}

// ✅ CORRETO — função de factory explícita (exceção R1 permitida)
func defaultConfig() Config {
    return Config{Timeout: 30 * time.Second}
}
```

> **Critério de aceitação:** `grep -rn "^func init()" --include="*.go" .` não deve retornar
> nenhum resultado. O agente DEVE executar este comando e bloquear qualquer ocorrência.

---

## Regra 1 — Toda função deve estar atachada a uma struct (método)

### Definição

Funções Go de domínio/aplicação/infraestrutura **DEVEM** ser métodos de struct. Funções standalone
(top-level `func foo(...)`) são proibidas em código de produção nas camadas acima.

### Exceções explícitas e exaustivas

As únicas funções standalone permitidas são:

| Contexto | Justificativa |
|----------|--------------|
| `func main()` | Ponto de entrada obrigatório do runtime Go |
| `func New*(deps...) (*T, error)` | Construtores/factories — retornam instância da struct; o corpo deve apenas validar invariantes e montar a struct |
| `func TestXxx(t *testing.T)` | Registrador de suite exigido pelo `go test`; contém apenas `suite.Run(...)` |
| Funções de interface pública de pacotes `pkg/` utilitários sem estado | Ex.: `pkg/uuid/New() string` — apenas quando não há estado nem dependências injetáveis |

> ⚠️ **`func init()` é PROIBIDA** — ver Regra 0 abaixo.

> **Critério de aceitação:** `grep -rn "^func [^(]" --include="*.go" .` não deve retornar nenhuma
> função fora das exceções listadas acima. O agente DEVE executar este comando e corrigir toda
> ocorrência não autorizada antes de finalizar a tarefa.

### Exemplos

```go
// ❌ PROIBIDO — função standalone em camada de aplicação
func processPayment(p *Payment) error {
    // lógica de negócio solta
}

// ❌ PROIBIDO — helper standalone em camada de domínio
func validateAmount(amount float64) error {
    return nil
}

// ✅ CORRETO — método de struct
func (uc *ProcessPaymentUseCase) Execute(ctx context.Context, input *dtos.PaymentInput) (*dtos.PaymentOutput, error) {
    // lógica de negócio atachada à struct
}

// ✅ CORRETO — validação como método do próprio tipo de domínio
func (p *Payment) Validate() error {
    return nil
}

// ✅ CORRETO — construtor (exceção permitida)
func NewProcessPaymentUseCase(obs observability.Observability, repo PaymentRepository) *ProcessPaymentUseCase {
    return &ProcessPaymentUseCase{obs: obs, repo: repo}
}
```

---

## Regra 2 — Proibição de atribuições diretas de campos de struct

### Definição

É **PROIBIDO** extrair campos de uma struct para variáveis locais intermediárias sem transformação
real. Variáveis locais criadas apenas para nomear ou "desaçucarar" um campo de struct são ruído,
adicionam indireção desnecessária e geram inconsistência quando o campo evolui.

### O que é "atribuição direta proibida"

Uma atribuição é proibida quando:
1. A variável local recebe **diretamente** um campo de struct (`x := struct.Field`) **E**
2. A variável local não sofre nenhuma **transformação real** antes de ser usada (parse, conversão
   de tipo, sanitização, cálculo, formatação, ou ser passada como argumento com semântica distinta)

### O que NÃO é proibido

```go
// ✅ PERMITIDO — transformação real: parse de string para int
userID, err := strconv.ParseInt(input.UserID, 10, 64)

// ✅ PERMITIDO — transformação real: conversão de tipo de domínio
amount := money.FromCents(input.AmountCents)

// ✅ PERMITIDO — desestruturação com operação: uppercase normalizado
code := strings.ToUpper(input.Code)

// ✅ PERMITIDO — variável de erro do retorno de função (não é campo de struct)
user, err := uc.userRepo.FindByID(ctx, id)

// ✅ PERMITIDO — shadow de contexto com timeout (operação real)
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
```

### Exemplos proibidos

```go
// ❌ PROIBIDO — extração de campo sem transformação
func (uc *GetUserUseCase) Execute(ctx context.Context, id int64) (*dtos.UserOutput, error) {
    user, err := uc.userRepo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("buscar usuário: %w", err)
    }

    nome := user.Name   // ← PROIBIDO: sem transformação
    email := user.Email // ← PROIBIDO: sem transformação

    return &dtos.UserOutput{Name: nome, Email: email}, nil
}

// ✅ CORRETO — usar o campo diretamente
func (uc *GetUserUseCase) Execute(ctx context.Context, id int64) (*dtos.UserOutput, error) {
    user, err := uc.userRepo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("buscar usuário: %w", err)
    }

    return &dtos.UserOutput{
        Name:  user.Name,
        Email: user.Email,
    }, nil
}

// ❌ PROIBIDO — alias de campo de input sem transformação
func (s *CreateOrderService) Execute(ctx context.Context, input *dtos.OrderInput) (*dtos.OrderOutput, error) {
    customerID := input.CustomerID // ← PROIBIDO: só renomeia
    productID  := input.ProductID  // ← PROIBIDO: só renomeia
    _ = customerID
    _ = productID
}

// ✅ CORRETO
func (s *CreateOrderService) Execute(ctx context.Context, input *dtos.OrderInput) (*dtos.OrderOutput, error) {
    order, err := entities.NewOrder(input.CustomerID, input.ProductID, input.Quantity)
    if err != nil {
        return nil, fmt.Errorf("criar pedido: %w", err)
    }
    // ...
}
```

> **Critério de aceitação:** o agente deve revisar todo bloco de função implementado e verificar
> que nenhuma variável local seja cópia direta de campo sem transformação. Em code review, apontar
> toda ocorrência como `[HARD]` — bloqueante de merge.

---

## Regra 3 — Geração de mocks obrigatória via mockery.yml

### Definição

Todo mock de interface utilizado em testes **DEVE** ser gerado via [mockery](https://vektra.github.io/mockery/)
com configuração declarada em `mockery.yml` na raiz do módulo (ou do sub-módulo Go). É proibido:
- Escrever mocks à mão
- Usar `go generate` com diretivas inline sem o `mockery.yml` correspondente
- Commitar mocks com divergência em relação à interface vigente

### Configuração mínima obrigatória do mockery.yml

```yaml
# mockery.yml — raiz do módulo
version: "2"
quiet: false
disable-version-string: false
with-expecter: true       # habilita EXPECT() fluente (testify/mock)
mockname: "{{.InterfaceName}}"
outpkg: "mocks"
filename: "{{.InterfaceName | snakecase}}.go"
dir: "{{.InterfaceDir}}/mocks"
packages:
  # Declarar cada pacote com interfaces a mockar
  # Exemplo:
  github.com/seu-org/seu-projeto/internal/payment_method/domain/repositories:
    interfaces:
      PaymentMethodRepository:
```

### Regras de geração

| Regra | Detalhe |
|-------|---------|
| `with-expecter: true` | Obrigatório — habilita `.EXPECT()` fluente e type-safe |
| `outpkg: "mocks"` | Mocks sempre em sub-pacote `mocks/` relativo ao pacote da interface |
| `filename` | snake_case do nome da interface |
| Regeneração | `mockery --config mockery.yml` deve ser executado após qualquer alteração de interface |
| CI gate | O CI deve falhar se mocks estiverem desatualizados (`mockery --config mockery.yml --dry-run`) |

> **Critério de aceitação:** o agente DEVE verificar que `mockery.yml` existe e contém a interface
> sendo testada antes de escrever qualquer teste. Se a interface não estiver declarada no
> `mockery.yml`, adicioná-la e executar `mockery --config mockery.yml` antes de prosseguir.

---

## Regra 4 — Padrão obrigatório de arquivo `_test.go`

### Definição

Todo arquivo de teste Go que cubra use cases, services e handlers **DEVE** seguir o padrão
`testify/suite` com table-driven scenarios. Testes avulsos com `t.Run` direto são permitidos
apenas para funções utilitárias simples sem dependências injetáveis.

### Estrutura obrigatória

```
1. package <pacote>            — mesmo pacote do código testado (whitebox) ou <pacote>_test (blackbox)
2. imports organizados         — stdlib | testify | mocks | pacotes internos | externos
3. Suite struct                — embute suite.Suite + todos os mocks como campos tipados
4. TestXxx(t)                  — registrador: apenas suite.Run(t, new(XxxSuite))
5. SetupTest()                 — reinicia todos os mocks (chamado antes de cada cenário)
6. Método de teste principal   — tabela scenarios com args, dependencies, expect func
7. Loop for _, scenario        — s.Run + instanciação real do SUT + assertions via scenario.expect
```

### Estrutura canônica completa

```go
package usecase // ou usecase_test para blackbox

import (
    "context"
    "errors"
    "testing"

    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"

    // observability e infraestrutura de apoio
    "github.com/JailtonJunior94/devkit-go/pkg/observability"
    "github.com/JailtonJunior94/devkit-go/pkg/observability/fake"

    // DTOs e mocks do pacote testado
    "github.com/seu-org/seu-projeto/internal/<dominio>/application/dtos"
    repositoryMock "github.com/seu-org/seu-projeto/internal/<dominio>/infrastructure/repositories/mocks"
)

// 1. Suite struct — um campo mock por dependência
type <NomeCaso>UseCaseSuite struct {
    suite.Suite

    ctx                context.Context
    obs                observability.Observability
    <dependencia>Mock  *repositoryMock.<Interface>
}

// 2. Registrador — APENAS suite.Run
func Test<NomeCaso>UseCaseSuite(t *testing.T) {
    suite.Run(t, new(<NomeCaso>UseCaseSuite))
}

// 3. SetupTest — reinicia mocks a cada cenário (evita vazamento de estado)
func (s *<NomeCaso>UseCaseSuite) SetupTest() {
    s.obs = fake.NewProvider()
    s.ctx = context.Background()
    s.<dependencia>Mock = repositoryMock.New<Interface>(s.T())
}

// 4. Método de teste principal — table-driven
func (s *<NomeCaso>UseCaseSuite) Test<Acao>() {
    type args struct {
        // input do use case
    }

    type dependencies struct {
        // mocks tipados; um campo por dependência
        <dependencia>Mock *repositoryMock.<Interface>
    }

    scenarios := []struct {
        name         string
        args         args
        dependencies dependencies
        expect       func(output <Tipo>, err error)
    }{
        {
            name: "deve <comportamento esperado>",
            args: args{ /* ... */ },
            dependencies: dependencies{
                <dependencia>Mock: func() *repositoryMock.<Interface> {
                    s.<dependencia>Mock.
                        EXPECT().
                        <Metodo>(s.ctx, mock.AnythingOfType("<tipo>")).
                        Return(<valor>).
                        Once()
                    return s.<dependencia>Mock
                }(),
            },
            expect: func(output <Tipo>, err error) {
                s.NoError(err)
                s.NotNil(output)
                // assertions específicas do cenário
            },
        },
        // cenários adicionais: happy path, edge cases, erros de domínio, erros de infra
    }

    for _, scenario := range scenarios {
        s.Run(scenario.name, func() {
            // Arrange: instanciar SUT com as dependências do cenário
            uc := New<NomeCaso>UseCase(
                s.obs,
                scenario.dependencies.<dependencia>Mock,
            )

            // Act
            output, err := uc.Execute(s.ctx, scenario.args.input)

            // Assert
            scenario.expect(output, err)
        })
    }
}
```

### Cobertura mínima obrigatória de cenários

Todo método `Test<Acao>()` **DEVE** conter ao menos:

| Cenário | Obrigatório |
|---------|------------|
| Happy path — sucesso nominal | ✅ |
| Erro de validação de domínio (input inválido) | ✅ |
| Erro de infraestrutura (repositório/serviço externo falha) | ✅ |
| Edge case específico do negócio (ex.: normalização, idempotência) | ✅ quando aplicável |

### Regras de nomenclatura de cenários

- Usar PT-BR: `"deve criar método de pagamento com sucesso"`
- Prefixo de erro: `"deve retornar erro ao <ação> com <condição>"`
- Sem abreviações: `"deve retornar erro ao salvar no repositório"` (não `"err save"`)

> **Critério de aceitação:** o agente DEVE verificar que todo `_test.go` de use case/service/handler
> contém: (1) struct de suite com `suite.Suite` embutido, (2) `SetupTest` que reinicia mocks,
> (3) tabela `scenarios` com no mínimo os 3 tipos de cenário obrigatórios, (4) loop `s.Run` com
> instanciação real do SUT dentro do loop.

---

## Regra 5 — Uber Go Style Guide (PT-BR)

As regras abaixo são derivadas diretamente do
[Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) e são **obrigatórias**
neste projeto. O agente deve verificar compliance em todo código produzido ou revisado.

### 5.1 Ponteiros para interfaces

Quase nunca se usa ponteiro para interface. Passe interfaces como valores — os dados subjacentes
já podem ser ponteiros.

```go
// ❌ PROIBIDO
func Process(r *io.Reader) {}

// ✅ CORRETO
func Process(r io.Reader) {}
```

### 5.2 Verificação de conformidade de interface em tempo de compilação

Tipos exportados que devem implementar uma interface **DEVEM** ser verificados em compile-time:

```go
// ✅ OBRIGATÓRIO para tipos exportados que implementam interfaces públicas
var _ http.Handler = (*Handler)(nil)
var _ PaymentRepository = (*postgresPaymentRepository)(nil)
```

### 5.3 Receptores e interfaces

- Métodos com **receptor de valor** podem ser chamados em ponteiros e valores.
- Métodos com **receptor de ponteiro** só podem ser chamados em ponteiros.
- Prefira receptor de ponteiro (`*T`) quando o método modifica o estado ou a struct é grande.
- Use receptor de valor apenas para tipos imutáveis pequenos.
- **Seja consistente**: se um método usa `*T`, todos os métodos do tipo devem usar `*T`.

### 5.4 Mutex como campo não-embutido

```go
// ❌ PROIBIDO — mutex embutido vaza Lock/Unlock na API pública
type Cache struct {
    sync.Mutex
    data map[string]string
}

// ✅ CORRETO — mutex como campo privado
type Cache struct {
    mu   sync.Mutex
    data map[string]string
}
```

### 5.5 Copiar slices e maps nas fronteiras

```go
// ❌ PROIBIDO — caller pode modificar estado interno
func (s *Service) SetItems(items []Item) {
    s.items = items
}

// ✅ CORRETO — cópia defensiva na entrada
func (s *Service) SetItems(items []Item) {
    s.items = make([]Item, len(items))
    copy(s.items, items)
}

// ✅ CORRETO — cópia defensiva na saída
func (s *Service) Items() []Item {
    result := make([]Item, len(s.items))
    copy(result, s.items)
    return result
}
```

### 5.6 Defer para limpeza de recursos

Use `defer` para fechar arquivos, liberar locks e limpar recursos:

```go
// ✅ CORRETO
func (r *repo) FindByID(ctx context.Context, id int64) (*Entity, error) {
    rows, err := r.db.QueryContext(ctx, query, id)
    if err != nil {
        return nil, fmt.Errorf("consultar entidade: %w", err)
    }
    defer rows.Close()
    // ...
}
```

### 5.7 Tamanho de canal: um ou zero

```go
// ❌ PROIBIDO — tamanho arbitrário sem justificativa
c := make(chan int, 64)

// ✅ CORRETO
c := make(chan int)      // sem buffer
c := make(chan int, 1)   // buffer de um
```

### 5.8 Enums começam em 1 (com iota)

```go
// ❌ PROIBIDO — zero value ambíguo
type Status int
const (
    Active Status = iota  // Active=0, confunde com zero value
    Inactive
)

// ✅ CORRETO — zero value reservado para "não inicializado"
type Status int
const (
    StatusActive   Status = iota + 1
    StatusInactive
    StatusArchived
)
```

**Exceção:** quando zero value tem semântica de default desejável (ex.: `LogToStdout = 0`).

### 5.9 Use `time.Time` e `time.Duration` — nunca `int` para tempo

```go
// ❌ PROIBIDO
func poll(delayMs int) {}
type Config struct { IntervalSeconds int }

// ✅ CORRETO
func poll(delay time.Duration) {}
type Config struct { Interval time.Duration }

// ✅ CORRETO — quando forçado a usar int por serialização, incluir unidade no nome
type Config struct { IntervalMillis int `json:"intervalMillis"` }
```

### 5.10 Tratamento de erros

#### Tipos de erro

| Situação | Abordagem |
|----------|-----------|
| Caller não precisa comparar, mensagem estática | `errors.New("mensagem")` |
| Caller não precisa comparar, mensagem dinâmica | `fmt.Errorf("contexto: %v", val)` |
| Caller precisa comparar, mensagem estática | `var ErrNome = errors.New(...)` |
| Caller precisa comparar, mensagem dinâmica | tipo customizado `type NomeError struct{}` |

#### Wrapping de erros

```go
// ❌ PROIBIDO — contexto redundante "failed to"
return fmt.Errorf("failed to create user: %w", err)

// ✅ CORRETO — contexto sucinto
return fmt.Errorf("criar usuário: %w", err)

// Use %w quando o caller precisa de errors.Is/As
// Use %v para ocultar o erro subjacente (sem unwrap)
```

#### Nomenclatura de erros

```go
// Erros exportados: prefixo Err
var ErrNotFound = errors.New("não encontrado")
var ErrInvalidInput = errors.New("input inválido")

// Erros não exportados: prefixo err
var errInternal = errors.New("erro interno")

// Tipos de erro customizados: sufixo Error
type ValidationError struct { Field string; Message string }
func (e *ValidationError) Error() string { ... }
```

#### Tratar erro apenas uma vez

```go
// ❌ PROIBIDO — log + return duplica o tratamento
u, err := uc.repo.FindByID(ctx, id)
if err != nil {
    log.Printf("erro ao buscar usuário: %v", err)
    return nil, err  // caller também vai logar
}

// ✅ CORRETO — wrapping + return (caller decide o que fazer)
u, err := uc.repo.FindByID(ctx, id)
if err != nil {
    return nil, fmt.Errorf("buscar usuário %d: %w", id, err)
}
```

### 5.11 Type assertions com comma-ok

```go
// ❌ PROIBIDO — panic em tipo incorreto
t := i.(string)

// ✅ CORRETO — sempre comma-ok
t, ok := i.(string)
if !ok {
    return fmt.Errorf("tipo inesperado: %T", i)
}
```

### 5.12 Não usar `panic` em código de produção

```go
// ❌ PROIBIDO em qualquer camada de produção
func (uc *UseCase) Execute(...) {
    if input == nil {
        panic("input não pode ser nil")
    }
}

// ✅ CORRETO — retornar erro
func (uc *UseCase) Execute(...) (*Output, error) {
    if input == nil {
        return nil, ErrInvalidInput
    }
}
```

**Exceções permitidas:** `template.Must(...)` e similares apenas em inicialização de `main`,
nunca em handlers, use cases ou repositories.

Em testes: usar `t.Fatal` ou `t.FailNow`, nunca `panic`.

### 5.13 Evitar globais mutáveis — usar injeção de dependência

```go
// ❌ PROIBIDO — global mutável
var _timeNow = time.Now

func sign(msg string) string {
    now := _timeNow()
    return signWithTime(msg, now)
}

// ✅ CORRETO — injeção via construtor
type Signer struct {
    now func() time.Time
}

func NewSigner() *Signer {
    return &Signer{now: time.Now}
}

func (s *Signer) Sign(msg string) string {
    now := s.now()
    return signWithTime(msg, now)
}
```

### 5.14 Não embutir tipos em structs públicas

```go
// ❌ PROIBIDO — vaza implementação, dificulta evolução
type ConcreteList struct {
    *AbstractList
}

// ✅ CORRETO — campo privado + métodos delegadores explícitos
type ConcreteList struct {
    list *AbstractList
}

func (l *ConcreteList) Add(e Entity) { l.list.Add(e) }
func (l *ConcreteList) Remove(e Entity) { l.list.Remove(e) }
```

**Regra de ouro:** "todos os métodos/campos expostos pelo tipo embutido fariam sentido na API
do tipo externo?" → se "alguns" ou "não", use campo em vez de embedding.

### 5.15 Não usar nomes built-in

```go
// ❌ PROIBIDO — shadowing de built-ins
var error string
func handle(error string) {}

// ✅ CORRETO
var errorMessage string
func handle(msg string) {}
```

### 5.16 Exit apenas em main

```go
// ❌ PROIBIDO — os.Exit ou log.Fatal fora de main
func readConfig() Config {
    raw, err := os.ReadFile("config.yaml")
    if err != nil {
        log.Fatal(err)  // impossibilita teste, pula defers
    }
}

// ✅ CORRETO — retornar erro; main decide
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("carregar configuração: %w", err)
    }
    // ...
    return nil
}
```

### 5.17 Tags em structs serializadas

```go
// ❌ PROIBIDO — sem tag, nome do campo vira contrato frágil
type Payment struct {
    Amount int
    Method string
}

// ✅ CORRETO — tag explícita = contrato estável
type Payment struct {
    Amount int    `json:"amount"`
    Method string `json:"method"`
}
```

### 5.18 Goroutines devem ter ciclo de vida controlado

```go
// ❌ PROIBIDO — goroutine sem mecanismo de parada
go func() {
    for { flush(); time.Sleep(delay) }
}()

// ✅ CORRETO — ciclo de vida explícito via struct
type Flusher struct {
    stop chan struct{}
    done chan struct{}
}

func NewFlusher(delay time.Duration) *Flusher {
    f := &Flusher{
        stop: make(chan struct{}),
        done: make(chan struct{}),
    }
    go f.run(delay)
    return f
}

func (f *Flusher) run(delay time.Duration) {
    defer close(f.done)
    ticker := time.NewTicker(delay)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            flush()
        case <-f.stop:
            return
        }
    }
}

func (f *Flusher) Shutdown() {
    close(f.stop)
    <-f.done
}
```

### 5.19 Preferir `strconv` em vez de `fmt` para conversões primitivas

```go
// ❌ LENTO
s := fmt.Sprintf("%d", n)

// ✅ CORRETO — 2x mais rápido, metade das alocações
s := strconv.Itoa(n)
```

### 5.20 Especificar capacidade de slices e maps

```go
// ❌ PROIBIDO em hot path — múltiplas realocações
m := make(map[string]Entity)
s := make([]Entity, 0)

// ✅ CORRETO — pré-alocação com capacidade conhecida
m := make(map[string]Entity, len(items))
s := make([]Entity, 0, len(items))
```

### 5.21 Reduzir aninhamento com early return

```go
// ❌ PROIBIDO — aninhamento profundo
for _, v := range items {
    if v.IsValid() {
        if err := v.Process(); err == nil {
            v.Send()
        } else {
            return err
        }
    }
}

// ✅ CORRETO — early return / early continue
for _, v := range items {
    if !v.IsValid() {
        continue
    }
    if err := v.Process(); err != nil {
        return err
    }
    v.Send()
}
```

### 5.22 Evitar `else` desnecessário

```go
// ❌ PROIBIDO
var status string
if isActive {
    status = "ativo"
} else {
    status = "inativo"
}

// ✅ CORRETO
status := "inativo"
if isActive {
    status = "ativo"
}
```

### 5.23 Ordenação de imports (3 grupos)

```go
import (
    // 1. stdlib
    "context"
    "fmt"

    // 2. dependências externas
    "github.com/stretchr/testify/suite"

    // 3. pacotes internos do projeto
    "github.com/seu-org/projeto/internal/domain"
)
```

### 5.24 Nomes de pacotes

| Regra | Exemplo |
|-------|---------|
| Tudo minúsculo, sem underscore | `usecase`, não `UseCase` nem `use_case` |
| Não plural | `entity`, não `entities` |
| Não genérico | `payment`, não `util`, `common`, `shared`, `lib` |
| Curto e descritivo | `auth`, `order`, `metric` |

### 5.25 Ordenação de funções por receptor

```
// Ordem obrigatória dentro de um arquivo .go:
// 1. type declaration
// 2. var/const declarations
// 3. New*()/constructor
// 4. métodos exportados (ordem de chamada aproximada)
// 5. métodos não exportados
// 6. funções utilitárias standalone (apenas as exceções permitidas)
```

### 5.26 Prefixar globals não exportados com `_`

```go
// ❌ PROIBIDO
const defaultTimeout = 30 * time.Second

// ✅ CORRETO
const _defaultTimeout = 30 * time.Second

// Exceção: erros sentinel sem underscore
var errNotFound = errors.New("não encontrado")
```

### 5.27 Inicializar structs com nomes de campos

```go
// ❌ PROIBIDO — frágil, quebra ao adicionar campo
u := User{"João", "joao@email.com", true}

// ✅ CORRETO — explícito e resiliente a mudanças
u := User{
    Name:   "João",
    Email:  "joao@email.com",
    Active: true,
}
```

---

O agente DEVE executar e reportar o resultado de cada item:

```bash
# R1 — Detectar funções standalone proibidas (exceto exceções)
grep -rn "^func [^(]" --include="*.go" . \
  | grep -v "_test.go" \
  | grep -v "func main()" \
  | grep -v "func init()" \
  | grep -v "func New"

# R2 — Detectar atribuições diretas de campos (heurística; revisar manualmente)
# Não há linter automático — revisão manual obrigatória em code review

# R3 — Verificar mockery.yml existe e está atualizado
test -f mockery.yml && echo "OK" || echo "AUSENTE — criar mockery.yml"
mockery --config mockery.yml --dry-run 2>&1 | grep -i "error\|differ" || echo "Mocks atualizados"

# R4 — Verificar padrão de suite nos arquivos de teste de use case
grep -rn "suite\.Suite" --include="*_test.go" internal/
grep -rn "SetupTest" --include="*_test.go" internal/
grep -rn "suite\.Run" --include="*_test.go" internal/

# Gate mínimo de testes
go test ./... -count=1 -race
```

---

## Restrições Adicionais

| Restrição | Detalhe |
|-----------|---------|
| `context.Context` | Sempre o primeiro parâmetro em métodos com IO/rede/cancelamento |
| Erros | `fmt.Errorf("contexto da operação: %w", err)` — wrapping obrigatório |
| Sentinel errors | `var Err<Nome> = errors.New("...")` para erros comparados por `errors.Is` |
| Estado global | Zero tolerância — nenhum `var` global mutável fora de `init` |
| `panic` | Proibido em código de produção; permitido apenas em `main` para falha de bootstrap |
| Mocks manuais | Proibidos — usar apenas mocks gerados por mockery |

---

## Como Usar Este Prompt com a Skill go-implementation

1. Carregar este arquivo como contexto adicional antes de iniciar qualquer implementação Go.
2. Toda violação das Regras 1–4 é classificada como `[HARD]` — bloqueante de merge.
3. O agente não deve finalizar a tarefa sem executar o Checklist de Validação e reportar os resultados.
4. Em caso de dúvida sobre se uma atribuição é "direta" (Regra 2), aplicar o critério:
   **"Esta variável local existe apenas para renomear o campo?"** → se sim, é proibida.
