# Prompt Enriquecido — Regras Estritas para go-implementation

> **Versão:** 2.0.0
> **Gerado por:** skill `prompt-enricher`
> **Destino:** skill `go-implementation` — extensão de regras obrigatórias
> **Escopo:** todo código Go implementado ou revisado pela skill go-implementation
> **Idioma de saída do agente:** PT-BR (comentários, erros, mensagens de log)
> **Referência base:** [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

---

## Contexto

Este prompt define restrições obrigatórias e não negociáveis que **se sobrepõem** às regras
existentes da skill `.agents/skills/go-implementation/SKILL.md`. Em caso de conflito, prevalece
a restrição mais restritiva. Toda tarefa de implementação Go que viole qualquer regra deste
documento deve ser **bloqueada e corrigida** antes de ser considerada concluída.

As regras abaixo se aplicam a **qualquer código Go de domínio, aplicação e infraestrutura**
produzido ou modificado pela skill, independentemente da camada (entity, use case, repository,
handler, service, adapter).

**Severidade padrão:** toda violação é classificada como `[HARD]` — bloqueante de merge —
salvo quando explicitamente indicado como `[SOFT]` (melhoria recomendada, não bloqueante).

**Relação com SKILL.md:** os patterns inline do SKILL.md (Factory Function, Functional Options,
Adapter, Decorator, Facade) e as regras de carregamento de referências permanecem vigentes.
Este prompt **complementa**, não substitui, o SKILL.md.

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

> ⚠️ **`func init()` é PROIBIDA** — ver Regra 0.

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

### 5.23 Ordenação de imports (3 grupos — convenção deste projeto)

O Uber Go Style Guide define 2 grupos (stdlib + todo o resto). Este projeto adota **3 grupos**,
convenção comum em codebases com muitos pacotes internos, aplicada via `goimports -local`:

```go
import (
    // 1. stdlib
    "context"
    "fmt"

    // 2. dependências externas
    "github.com/stretchr/testify/suite"

    // 3. pacotes internos do projeto (prefixo do módulo em go.mod)
    "github.com/seu-org/projeto/internal/domain"
)
```

Nunca mescle os grupos. Use `goimports` (não apenas `gofmt`) para manter a ordem.

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

**Exceção:** em test tables com ≤ 3 campos, pode-se omitir os nomes.

### 5.28 Omitir campos zero-value ao inicializar structs

Ao inicializar structs com nomes de campos (5.27), omita campos que receberiam o zero-value do
tipo — a menos que o zero-value forneça contexto semântico importante (ex.: test tables).

```go
// ❌ PROIBIDO — ruído desnecessário
user := User{
    FirstName:  "João",
    LastName:   "Silva",
    MiddleName: "",    // zero-value implícito
    Admin:      false, // zero-value implícito
}

// ✅ CORRETO — apenas campos com valor significativo
user := User{
    FirstName: "João",
    LastName:  "Silva",
}
```

### 5.29 Usar `var` para structs zero-value

Quando todos os campos de uma struct são zero-value, usar `var` em vez de literal vazio.

```go
// ❌ PROIBIDO
user := User{}

// ✅ CORRETO — sinaliza explicitamente "valor inicial/zero"
var user User
```

### 5.30 Inicializar referências de struct com `&T{}`

Use `&T{}` em vez de `new(T)` para manter consistência com a inicialização de structs.

```go
// ❌ PROIBIDO — inconsistente, forma de inicialização separada do valor
sptr := new(T)
sptr.Name = "bar"

// ✅ CORRETO — consistente com struct literals
sptr := &T{Name: "bar"}
```

### 5.31 Inicializar maps com `make()` (exceto literals fixos)

Use `make(map[K]V)` para maps populados programaticamente. Use map literals para conjuntos
fixos de elementos conhecidos em tempo de compilação.

```go
// ❌ PROIBIDO — confunde declaração com inicialização
var m = map[string]Entity{}

// ✅ CORRETO — populado programaticamente
m := make(map[string]Entity, len(items))
for _, item := range items {
    m[item.ID] = item
}

// ✅ CORRETO — conjunto fixo em tempo de compilação
m := map[string]string{
    "pt": "português",
    "en": "inglês",
}
```

### 5.32 Zero-value de `sync.Mutex` é válido — nunca use `new(sync.Mutex)`

O zero-value de `sync.Mutex` e `sync.RWMutex` é válido e usável sem inicialização.

```go
// ❌ PROIBIDO — alocação desnecessária
mu := new(sync.Mutex)
mu.Lock()

// ✅ CORRETO
var mu sync.Mutex
mu.Lock()

// ✅ CORRETO — campo em struct (não embutido — ver 5.4)
type Cache struct {
    mu   sync.Mutex
    data map[string]string
}
```

### 5.33 Exit Once — uma única saída em `main()`

Além de `os.Exit` e `log.Fatal*` existirem **apenas em `main()`** (Regra 5.16), prefira
**uma única chamada** delegando toda lógica para uma função `run()` retornando `error`.
Isso garante que `defer` seja executado, simplifica testes e torna o fluxo previsível.

```go
// ❌ PROIBIDO — múltiplas saídas, defers pulados
func main() {
    args := os.Args[1:]
    if len(args) != 1 {
        log.Fatal("argumento obrigatório")
    }
    f, err := os.Open(args[0])
    if err != nil {
        log.Fatal(err) // defer em f.Close nunca executa
    }
    defer f.Close()
    // ...
}

// ✅ CORRETO — saída única, lógica testável
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    args := os.Args[1:]
    if len(args) != 1 {
        return errors.New("argumento obrigatório")
    }
    f, err := os.Open(args[0])
    if err != nil {
        return err
    }
    defer f.Close()
    // ...
    return nil
}
```

### 5.34 Goroutines em `init()` são duplamente proibidas

Além de `init()` ser proibida pela Regra 0, é especialmente grave iniciar goroutines em `init()`:
a goroutine roda sem ciclo de vida controlado, sem mecanismo de parada e impossibilita testes.
Se um pacote precisa de trabalho em background, exponha um objeto com `Close()`/`Shutdown()`.

```go
// ❌ DUPLAMENTE PROIBIDO — init + goroutine sem controle
func init() {
    go doWork() // goroutine órfã, sem shutdown
}

// ✅ CORRETO — objeto com ciclo de vida explícito
type Worker struct {
    stop chan struct{}
    done chan struct{}
}

func NewWorker() *Worker {
    w := &Worker{
        stop: make(chan struct{}),
        done: make(chan struct{}),
    }
    go w.run()
    return w
}

func (w *Worker) Shutdown() {
    close(w.stop)
    <-w.done
}
```

### 5.35 Usar `sync.WaitGroup` para múltiplas goroutines

Quando múltiplas goroutines precisam ser aguardadas, use `sync.WaitGroup`. Para uma única
goroutine, use canal `done chan struct{}`.

```go
// ✅ CORRETO — múltiplas goroutines
var wg sync.WaitGroup
for i := 0; i < n; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        // ...
    }()
}
wg.Wait()

// ✅ CORRETO — goroutine única
done := make(chan struct{})
go func() {
    defer close(done)
    // ...
}()
<-done
```

### 5.36 Evitar conversões repetidas de string para `[]byte`

Em hot paths (loops, handlers de alta frequência), converta `string → []byte` uma única vez
fora do loop.

```go
// ❌ PROIBIDO em hot path — nova alocação a cada iteração
for i := 0; i < n; i++ {
    w.Write([]byte("prefixo:"))
}

// ✅ CORRETO — converte uma vez, reutiliza
prefix := []byte("prefixo:")
for i := 0; i < n; i++ {
    w.Write(prefix)
}
```

### 5.37 Linhas com limite de 99 caracteres `[SOFT]`

Limite suave de **99 caracteres** por linha. Quebre linhas antes desse limite. Não é um limite
rígido — código que ultrapassa por necessidade é permitido, mas cadeias de chamadas longas
e literais extensos devem ser quebrados para legibilidade.

### 5.38 Ser consistente

> **"Código consistente é mais fácil de manter, raciocinar e migrar."** — Uber Go Style Guide

Quando múltiplas abordagens são válidas, escolha uma e **mantenha-a em todo o pacote (ou
módulo)**. Aplicar estilo diferente dentro do mesmo pacote gera overhead cognitivo e
code reviews dolorosas. Mudanças de estilo devem ser feitas no nível de pacote ou maior.

### 5.39 Agrupar declarações similares

Use blocos `const()`, `var()`, `type()` para declarações relacionadas. Não agrupe declarações
não relacionadas no mesmo bloco.

```go
// ❌ PROIBIDO — declarações soltas
const a = 1
const b = 2
var x = "foo"
var y = "bar"

// ✅ CORRETO — agrupadas
const (
    a = 1
    b = 2
)

var (
    x = "foo"
    y = "bar"
)

// ❌ PROIBIDO — mistura tipos não relacionados no mesmo bloco
const (
    Add Operation = iota + 1
    Subtract
    EnvVar = "MY_ENV" // não relacionado
)

// ✅ CORRETO — separado
const (
    Add      Operation = iota + 1
    Subtract
)
const EnvVar = "MY_ENV"
```

### 5.40 Nomes de funções em MixedCaps

```go
// ❌ PROIBIDO
func get_user_by_id() {}
func GetUser_ByID() {}

// ✅ CORRETO — MixedCaps
func getUserByID() {}
func GetUserByID() {}

// ✅ EXCEÇÃO — funções de teste podem usar underscore para agrupar
func TestGetUser_WhenIDNotFound(t *testing.T) {}
```

### 5.41 Alias de import apenas para conflitos de nome

Use alias somente quando: (a) o nome do pacote não coincide com o último elemento do caminho,
ou (b) há conflito direto entre dois imports.

```go
// ❌ PROIBIDO — alias sem necessidade
import (
    runtimetrace "runtime/trace"
    nettrace     "golang.net/x/trace"
)

// ✅ CORRETO — alias apenas para resolver conflito
import (
    "runtime/trace"
    nettrace "golang.net/x/trace"
)

// ✅ CORRETO — alias porque nome do pacote ≠ último segmento do path
import (
    client "example.com/client-go"
    trace  "example.com/trace/v2"
)
```

### 5.42 Declarações top-level: omitir tipo quando óbvio

No nível do pacote, use `var` sem especificar o tipo quando ele já é evidente pela expressão.
Especifique o tipo apenas quando diferente do retorno da expressão.

```go
// ❌ PROIBIDO — tipo redundante
var _s string = F()
func F() string { return "A" }

// ✅ CORRETO
var _s = F()

// ✅ CORRETO — tipo necessário: F() retorna myError, queremos error
var _e error = F()
func F() myError { return myError{} }
```

### 5.43 Declarações locais: `:=` para valores, `var` para zero-values

```go
// ❌ PROIBIDO — var para valor explícito
var s = "foo"

// ✅ CORRETO
s := "foo"

// ❌ PROIBIDO — literal vazio para slice que receberá append
filtered := []int{}

// ✅ CORRETO — var para zero-value de slice
var filtered []int
for _, v := range list {
    if v > 10 {
        filtered = append(filtered, v)
    }
}
```

### 5.44 `nil` é um slice válido

```go
// ❌ PROIBIDO — literal vazio quando resultado é "nenhum item"
if x == "" {
    return []int{}
}

// ✅ CORRETO
if x == "" {
    return nil
}

// ❌ PROIBIDO — checar nil para saber se está vazio
func isEmpty(s []string) bool {
    return s == nil
}

// ✅ CORRETO — sempre usar len()
func isEmpty(s []string) bool {
    return len(s) == 0
}
```

> ⚠️ `nil` e `[]T{}` não são idênticos em serialização JSON (nil → `null`, vazio → `[]`).
> Escolha conscientemente.

### 5.45 Reduzir escopo de variáveis

Declare variáveis no menor escopo possível. Não reduza o escopo se isso forçar aninhamento
extra (ver 5.21).

```go
// ❌ PROIBIDO — escopo maior que o necessário
err := os.WriteFile(name, data, 0644)
if err != nil {
    return err
}

// ✅ CORRETO — err declarada diretamente no if
if err := os.WriteFile(name, data, 0644); err != nil {
    return err
}

// ✅ CORRETO — escopo ampliado quando resultado é usado depois
data, err := os.ReadFile(name)
if err != nil {
    return err
}
if err := cfg.Decode(data); err != nil {
    return err
}
```

### 5.46 Evitar parâmetros naked (sem nome aparente)

Parâmetros booleanos ou inteiros sem contexto óbvio devem usar comentário C-style ou, melhor
ainda, tipos nomeados.

```go
// ❌ PROIBIDO — "true, true" sem contexto
printInfo("foo", true, true)

// ✅ MELHOR — comentário inline
printInfo("foo", true /* isLocal */, true /* done */)

// ✅ IDEAL — tipos nomeados (type-safe, extensível)
type Region int
const (
    UnknownRegion Region = iota
    Local
)

type Status int
const (
    StatusReady  Status = iota + 1
    StatusDone
)

func printInfo(name string, region Region, status Status) {}
```

### 5.47 Usar raw string literals para evitar escaping

```go
// ❌ DIFÍCIL DE LER — escaping manual
wantError := "unknown name:\"test\""

// ✅ CORRETO — backtick literal, sem escape
wantError := `unknown name:"test"`
```

### 5.48 Format strings fora de `Printf` devem ser `const`

Permite que `go vet` analise estaticamente a string de formato.

```go
// ❌ PROIBIDO — variável escapa análise estática
msg := "valores inesperados %v, %v\n"
fmt.Printf(msg, 1, 2)

// ✅ CORRETO
const msg = "valores inesperados %v, %v\n"
fmt.Printf(msg, 1, 2)
```

### 5.49 Nomes de funções estilo `Printf` terminam com `f`

Para que `go vet` valide strings de formato em funções customizadas Printf-style, o nome
deve terminar com `f`.

```go
// ❌ PROIBIDO — go vet não detecta
func Wrap(msg string, args ...any) {}

// ✅ CORRETO — go vet verifica com -printfuncs=Wrapf
func Wrapf(format string, args ...any) {}
```

### 5.50 Functional Options — padrão obrigatório para structs com muitos opcionais

Quando uma struct tem mais de 3 campos opcionais ou configuráveis, usar Functional Options
em vez de múltiplos construtores ou builder fluente.

```go
// ❌ PROIBIDO — múltiplos construtores ou struct exposta
func NewServer(addr string, timeout int, maxConn int, tls bool) *Server {}

// ✅ CORRETO — functional options
type ServerOption func(*Server)

func WithTimeout(d time.Duration) ServerOption {
    return func(s *Server) { s.timeout = d }
}

func WithMaxConns(n int) ServerOption {
    return func(s *Server) { s.maxConns = n }
}

func NewServer(addr string, opts ...ServerOption) *Server {
    s := &Server{addr: addr, timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(s)
    }
    return s
}
```

---

## Regra 6 — Design e Contratos Go

Princípios de design derivados do SKILL.md que têm impacto direto na production-readiness
e que complementam as regras de estilo do Uber Guide.

### 6.1 `context.Context` é obrigatório em fronteiras de I/O

Todo método que realiza I/O (rede, banco, arquivo, subprocess, operação cancelável) **DEVE**
receber `context.Context` como **primeiro parâmetro**. Nunca armazene `Context` em struct.

```go
// ❌ PROIBIDO — context em struct
type Repo struct {
    ctx context.Context
    db  *sql.DB
}

// ❌ PROIBIDO — operação de I/O sem context
func (r *Repo) FindByID(id int64) (*Entity, error) {}

// ✅ CORRETO — context como primeiro parâmetro
func (r *Repo) FindByID(ctx context.Context, id int64) (*Entity, error) {}
```

**Propagação obrigatória:** nunca passe `context.Background()` ou `context.TODO()` dentro
de handlers/use cases; propague o context recebido. `context.Background()` é permitido apenas
em `main()`, inicialização de servidor e testes.

### 6.2 Preferir tipos concretos por padrão — interface sob demanda real

Introduza interface apenas quando houver ao menos **uma** das seguintes condições:
1. Múltiplas implementações concretas em produção (não apenas em testes)
2. Necessidade de substituição em testes (mock/fake)
3. Fronteira entre pacotes onde o consumidor não deve depender do concreto

```go
// ❌ EVITAR — interface prematura sem consumidor real
type UserGetter interface {
    GetUser(ctx context.Context, id int64) (*User, error)
}

// ✅ CORRETO — concreto por padrão; interface introduzida no pacote consumidor
// internal/user/repository/postgres_user_repository.go
type postgresUserRepository struct { db *sql.DB }

// internal/user/usecase/get_user_usecase.go  ← consumidor define a interface
type UserRepository interface {
    FindByID(ctx context.Context, id int64) (*User, error)
}
```

### 6.3 Interfaces definidas no pacote consumidor

Interfaces devem ser declaradas no pacote que as **consome**, não no pacote que as implementa.
Isso minimiza acoplamento e permite que implementações evoluam independentemente.

```go
// ❌ PROIBIDO — interface no pacote produtor
// internal/repository/user_repository.go
package repository
type UserRepository interface { FindByID(...) }

// ✅ CORRETO — interface no pacote consumidor
// internal/usecase/get_user_usecase.go
package usecase
type userRepository interface { FindByID(ctx context.Context, id int64) (*User, error) }
```

**Exceção:** interfaces que precisam ser compartilhadas por múltiplos consumidores podem
residir em pacote `pkg/` dedicado — nunca em `internal/` do produtor.

### 6.4 Zero values úteis — projetar structs que funcionam sem construtor

Sempre que possível, projete structs cujo zero-value seja funcional e seguro. Construtores
(`New*`) são obrigatórios apenas quando há **invariantes a validar** ou **dependências
obrigatórias a injetar**.

```go
// ❌ EVITAR — construtor sem invariante real
func NewConfig() *Config {
    return &Config{} // zero-value já seria suficiente
}

// ✅ CORRETO — construtor com invariante
func NewPaymentService(repo PaymentRepository, obs observability.Observability) (*PaymentService, error) {
    if repo == nil {
        return nil, errors.New("repo é obrigatório")
    }
    return &PaymentService{repo: repo, obs: obs}, nil
}

// ✅ CORRETO — zero-value útil (sem construtor necessário)
var buf bytes.Buffer
buf.WriteString("olá")
```

### 6.5 Erros sentinel e tipos customizados — decisão explícita

A escolha entre sentinel error, tipo customizado e `fmt.Errorf` deve ser explícita e baseada
nas necessidades do caller. Esta regra complementa 5.10 com critério de decisão para o
**design de pacotes**.

| O caller vai usar `errors.Is`? | O caller vai usar `errors.As`? | Mensagem | Use |
|---|---|---|---|
| Não | Não | Estática | `errors.New(...)` inline |
| Não | Não | Dinâmica | `fmt.Errorf("ctx: %v", ...)` |
| Sim | Não | Estática | `var ErrNome = errors.New(...)` exportado |
| Sim | Sim | Dinâmica | `type NomeError struct{ ... }` exportado |

Erros exportados passam a ser **parte da API pública** do pacote — documente-os.

### 6.6 Injeção de dependência via construtor — zero estado global

Todo estado que não é constante de domínio puro deve ser injetado via construtor. É proibido:
- Estado mutável em variáveis globais de pacote
- Singletons com `sync.Once` em código de produção (apenas `main()`)
- Inicialização lazy de dependências via campo opcional não injetado

```go
// ❌ PROIBIDO — estado global mutável
var _db *sql.DB

// ❌ PROIBIDO — inicialização lazy
type Service struct {
    db   *sql.DB
    repo *UserRepo // inicializado lazy em Execute()
}

// ✅ CORRETO — tudo injetado, zero estado global
type UserService struct {
    repo UserRepository
    obs  observability.Observability
}

func NewUserService(repo UserRepository, obs observability.Observability) *UserService {
    return &UserService{repo: repo, obs: obs}
}
```

---

## Regra 7 — Sempre usar os recursos mais modernos da linguagem Go

O agente **DEVE** escrever código usando a versão de Go declarada em `go.mod` e **preferir
obrigatoriamente** as APIs, builtins e pacotes introduzidos nas versões recentes da linguagem.
Reescrever manualmente o que a stdlib já oferece é proibido.

> **Critério de prioridade:** "Existe um builtin, função de stdlib ou idioma da linguagem que
> faz isso nativamente na versão do `go.mod`?" → se sim, usá-lo é **obrigatório**.

---

### 7.1 `any` em vez de `interface{}`

Desde Go 1.18, `any` é o alias oficial de `interface{}`. Usar `interface{}` é proibido.

```go
// ❌ PROIBIDO
func Process(v interface{}) {}
var m map[string]interface{}

// ✅ CORRETO
func Process(v any) {}
var m map[string]any
```

### 7.2 `log/slog` para logging estruturado — nunca `log` ou `fmt.Println`

Desde Go 1.21, `log/slog` é o logger estruturado oficial da stdlib. É proibido usar `log`,
`log.Printf`, `fmt.Println` ou qualquer logger caseiro onde logging estruturado seja necessário.

```go
// ❌ PROIBIDO
log.Printf("usuário criado: id=%d", id)
fmt.Println("erro:", err)

// ✅ CORRETO — slog com atributos tipados
slog.InfoContext(ctx, "usuário criado", slog.Int64("id", id))
slog.ErrorContext(ctx, "falha ao processar", slog.String("erro", err.Error()))

// ✅ CORRETO — logger injetado via construtor
type Service struct {
    log *slog.Logger
}

func NewService(log *slog.Logger) *Service {
    return &Service{log: log.With("component", "service")}
}
```

### 7.3 Pacote `slices` — nunca loops manuais para operações de coleção

Desde Go 1.21, o pacote `slices` oferece operações seguras, genéricas e idiomáticas.

```go
// ❌ PROIBIDO — loop manual para operações que slices já oferece
func contains(items []string, target string) bool {
    for _, v := range items {
        if v == target {
            return true
        }
    }
    return false
}

// ✅ CORRETO
import "slices"

slices.Contains(items, target)           // busca
slices.Index(items, target)              // índice ou -1
slices.Sort(items)                       // ordenação in-place
slices.SortFunc(items, cmp.Compare)      // ordenação com comparador
slices.Reverse(items)                    // inversão in-place
slices.Compact(items)                    // remove consecutivos duplicados
slices.Clone(items)                      // cópia superficial
slices.DeleteFunc(items, pred)           // filtrar fora
slices.Collect(iter.Seq[T])              // coletar iterador (Go 1.23+)
```

### 7.4 Pacote `maps` — nunca loops manuais para operações de mapa

Desde Go 1.21, o pacote `maps` oferece operações idiomáticas sobre maps.

```go
// ❌ PROIBIDO — loop manual para clonar
func cloneMap(m map[string]int) map[string]int {
    out := make(map[string]int, len(m))
    for k, v := range m {
        out[k] = v
    }
    return out
}

// ✅ CORRETO
import "maps"

maps.Clone(m)              // cópia superficial
maps.Keys(m)               // slice de chaves
maps.Values(m)             // slice de valores
maps.DeleteFunc(m, pred)   // remover entradas que satisfazem predicate
maps.Equal(m1, m2)         // comparação elemento a elemento
maps.Collect(iter.Seq2[K,V]) // coletar iterador (Go 1.23+)
```

### 7.5 Builtins `min`, `max`, `clear` — nunca implementações manuais

Desde Go 1.21, `min`, `max` e `clear` são builtins nativos.

```go
// ❌ PROIBIDO
func min(a, b int) int { if a < b { return a }; return b }
for k := range m { delete(m, k) } // limpar map manualmente

// ✅ CORRETO
x := min(a, b)
y := max(a, b, c)   // aceita variádico
clear(m)            // limpa map ou zera slice in-place
```

### 7.6 `errors.Join` para agregar múltiplos erros — nunca concatenação manual

Desde Go 1.20, `errors.Join` cria um erro composto que suporta `errors.Is`/`errors.As`.

```go
// ❌ PROIBIDO — concatenação que quebra a cadeia de erros
msg := err1.Error() + "; " + err2.Error()
return fmt.Errorf("%v; %v", err1, err2)

// ✅ CORRETO — erros compostos e navegáveis
return errors.Join(err1, err2)

// ✅ CORRETO — com contexto adicional (wrapping + join)
var errs []error
for _, item := range items {
    if err := process(item); err != nil {
        errs = append(errs, fmt.Errorf("item %s: %w", item.ID, err))
    }
}
return errors.Join(errs...)
```

### 7.7 Range sobre inteiros — nunca `for i := 0; i < n; i++` sem necessidade

Desde Go 1.22, `range` aceita inteiros diretamente.

```go
// ❌ EVITAR quando o índice é o único elemento necessário
for i := 0; i < 10; i++ {
    fmt.Println(i)
}

// ✅ CORRETO
for i := range 10 {
    fmt.Println(i)
}
```

**Exceção:** quando o corpo do loop modifica a variável de iteração ou precisa de controle
preciso do incremento, o `for` clássico continua sendo a forma correta.

### 7.8 Generics — eliminar duplicação de código com type parameters

Desde Go 1.18, use generics para componentes reutilizáveis em vez de duplicar lógica por tipo
ou usar `any` com type assertions internas.

```go
// ❌ PROIBIDO — duplicação de lógica idêntica por tipo
func ContainsInt(s []int, v int) bool { ... }
func ContainsString(s []string, v string) bool { ... }

// ❌ PROIBIDO — any com switch de tipo
func Contains(s any, v any) bool {
    switch slice := s.(type) { ... }
}

// ✅ CORRETO — generic com constraint
func Contains[T comparable](s []T, v T) bool {
    for _, item := range s {
        if item == v {
            return true
        }
    }
    return false
}
```

**Regra:** generics são adequados quando a lógica é **idêntica** para vários tipos. Não use
generics para polimorfismo de comportamento — para isso, use interfaces.

### 7.9 Pacote `cmp` para comparações e ordenação

Desde Go 1.21, `cmp.Compare` e `cmp.Equal` oferecem comparação ordenada e de igualdade
type-safe para tipos `ordered`.

```go
// ❌ EVITAR — comparação manual inline
slices.SortFunc(items, func(a, b Item) int {
    if a.Price < b.Price { return -1 }
    if a.Price > b.Price { return 1 }
    return 0
})

// ✅ CORRETO
import "cmp"

slices.SortFunc(items, func(a, b Item) int {
    return cmp.Compare(a.Price, b.Price)
})

// multi-campo (lexicográfico)
slices.SortFunc(items, func(a, b Item) int {
    return cmp.Or(
        cmp.Compare(a.Category, b.Category),
        cmp.Compare(a.Price, b.Price),
    )
})
```

### 7.10 `sync.OnceValue` / `sync.OnceValues` para inicialização lazy segura

Desde Go 1.21, `sync.OnceValue` e `sync.OnceValues` encapsulam o padrão `sync.Once` com
retorno de valor de forma type-safe.

```go
// ❌ PROIBIDO — sync.Once manual verboso
var (
    _cfg     Config
    _cfgOnce sync.Once
)
func getConfig() Config {
    _cfgOnce.Do(func() { _cfg = loadConfig() })
    return _cfg
}

// ✅ CORRETO — apenas em main/inicialização (não em handlers/use cases)
var getConfig = sync.OnceValue(loadConfig)

// com erro
var getDB = sync.OnceValues(func() (*sql.DB, error) {
    return sql.Open("postgres", os.Getenv("DATABASE_URL"))
})
```

> ⚠️ `sync.OnceValue` em `main()` ou setup de servidor é permitido. Em handlers, use cases ou
> repositories: **proibido** — injetar via construtor (Regra 6.6).

### 7.11 Iteradores com `iter.Seq` / `iter.Seq2` (Go 1.23+)

Desde Go 1.23, o pacote `iter` e range-over-functions permitem iteradores lazy e componíveis
sem alocação de slice intermediário.

```go
// ❌ EVITAR — alocação de slice intermediário apenas para iterar
func AllUsers(ctx context.Context) ([]User, error) { ... }
users, _ := repo.AllUsers(ctx)
for _, u := range users { process(u) }

// ✅ CORRETO — iterador lazy quando o caller precisa apenas iterar
func (r *Repo) Users(ctx context.Context) iter.Seq2[User, error] {
    return func(yield func(User, error) bool) {
        rows, err := r.db.QueryContext(ctx, "SELECT ...")
        if err != nil { yield(User{}, err); return }
        defer rows.Close()
        for rows.Next() {
            var u User
            if err := rows.Scan(&u.ID, &u.Name); err != nil {
                if !yield(User{}, err) { return }
                continue
            }
            if !yield(u, nil) { return }
        }
    }
}

// Uso
for user, err := range repo.Users(ctx) {
    if err != nil { return err }
    process(user)
}
```

**Quando usar:** iteradores são adequados para coleções grandes ou potencialmente infinitas onde
materializar o slice seria custoso. Para coleções pequenas e fixas, slice continua correto.

### 7.12 Versão do Go em `go.mod` — nunca assumir; sempre verificar

O agente DEVE verificar a versão de Go no `go.mod` antes de usar qualquer recurso moderno.
A tabela abaixo lista os recursos por versão mínima:

| Recurso | Versão mínima |
|---------|--------------|
| `any`, generics, `comparable` | Go 1.18 |
| `errors.Join` | Go 1.20 |
| `slices`, `maps`, `cmp`, `log/slog`, `min`/`max`/`clear`, `sync.OnceValue` | Go 1.21 |
| Range sobre inteiros, loop variable per-iteration | Go 1.22 |
| `iter.Seq`, range-over-functions, `slices.Collect`, `maps.Collect` | Go 1.23 |
| `weak`, melhorias em `sync.Map` | Go 1.24 |

> Se `go.mod` declarar versão anterior ao recurso desejado, o agente **NÃO DEVE** usá-lo e
> deve registrar explicitamente: `"recurso X requer Go Y; go.mod declara Go Z — não aplicado"`.

---

## Checklist de Validação (obrigatório antes de finalizar)

O agente **DEVE** executar e reportar o resultado de cada item antes de declarar a tarefa concluída.
Qualquer item com resultado diferente do esperado é `[HARD]` — bloqueante.

```bash
# ── R0: init() inexistente ────────────────────────────────────────────────────
grep -rn "^func init()" --include="*.go" .
# Esperado: NENHUMA linha

# ── R1: funções standalone proibidas ─────────────────────────────────────────
grep -rn "^func [^(]" --include="*.go" . \
  | grep -v "_test.go" \
  | grep -v "func main()" \
  | grep -v "func New" \
  | grep -v "^cmd/"
# Esperado: NENHUMA linha (exceto pkg/ utilitários sem estado declarados)

# ── R2: atribuições diretas de campo sem transformação ────────────────────────
# Revisão manual obrigatória em cada método implementado.
# Critério: "Esta variável local existe apenas para renomear um campo?" → PROIBIDA

# ── R3: mockery.yml e mocks atualizados ──────────────────────────────────────
test -f mockery.yml \
  && echo "mockery.yml: OK" \
  || echo "[HARD] AUSENTE — criar mockery.yml antes de escrever testes"
mockery --config mockery.yml --dry-run 2>&1 | grep -i "error\|differ" \
  && echo "[HARD] MOCKS DESATUALIZADOS" \
  || echo "Mocks: OK"

# ── R4: padrão testify/suite nos testes de use case / service / handler ───────
find . -path "*/internal/*_test.go" | xargs grep -L "suite\.Suite" 2>/dev/null \
  && echo "[HARD] FALTAM SUITES"
find . -path "*/internal/*_test.go" | xargs grep -L "SetupTest" 2>/dev/null \
  && echo "[HARD] FALTAM SetupTest"
find . -path "*/internal/*_test.go" | xargs grep -L "suite\.Run" 2>/dev/null \
  && echo "[HARD] FALTAM suite.Run"

# ── R5/R6: os.Exit / log.Fatal fora de main ──────────────────────────────────
grep -rn "os\.Exit\|log\.Fatal" --include="*.go" . | grep -v "^cmd/"
# Esperado: NENHUMA linha

# ── R5: panic fora de inicialização ──────────────────────────────────────────
grep -rn "\bpanic(" --include="*.go" . \
  | grep -v "_test.go" \
  | grep -v "template\.Must\|regexp\.MustCompile"
# Esperado: NENHUMA linha (exceto template.Must / regexp.MustCompile em main)

# ── R5: goroutines fire-and-forget ───────────────────────────────────────────
# Revisão manual: toda `go func()` deve ter canal stop+done ou sync.WaitGroup

# ── R5: type assertion sem comma-ok ──────────────────────────────────────────
# Revisão manual: toda assertion i.(T) deve ter a forma t, ok := i.(T)

# ── R5: globals não exportados sem prefixo _ ─────────────────────────────────
grep -rn "^var [a-z][a-zA-Z]\|^const [a-z][a-zA-Z]" --include="*.go" . \
  | grep -v "_test.go" | grep -v "^.*var err"
# Revisar: globals sem _ que não sejam erros sentinel (var errX)

# ── R6: context.Context não armazenado em struct ─────────────────────────────
# Revisão manual: nenhum campo de struct deve ter tipo context.Context

# ── R7: interface{} proibido — usar any ──────────────────────────────────────
grep -rn "interface{}" --include="*.go" . | grep -v "_test.go" | grep -v "vendor/"
# Esperado: NENHUMA linha

# ── Gate de qualidade final ──────────────────────────────────────────────────
go build ./...
go vet ./...
go test ./... -count=1 -race
golangci-lint run --timeout=5m 2>/dev/null || echo "[SOFT] golangci-lint não disponível"
```

---

## Como Usar Este Prompt com a Skill go-implementation

1. Carregar este arquivo **e** o SKILL.md (`.agents/skills/go-implementation/SKILL.md`) antes
   de iniciar qualquer implementação Go. Este prompt complementa, não substitui, o SKILL.md.
2. Toda violação das Regras 0–7 é `[HARD]` — bloqueante de merge — salvo quando marcada `[SOFT]`.
3. Nunca declarar tarefa concluída sem executar **todo** o Checklist de Validação e reportar
   os resultados explicitamente.
4. Regra 2 (atribuições diretas): aplicar o critério **"Esta variável local existe apenas para
   renomear o campo?"** — se sim, é proibida.
5. As Regras 0–7 são cumulativas e complementares — não há precedência entre elas.
6. Em conflito entre uma regra deste prompt e o SKILL.md, prevalece a **restrição mais restritiva**.
7. Regra 7 (recursos modernos): **sempre** verificar a versão em `go.mod` antes de aplicar.
   Não usar recurso de versão superior à declarada; registrar explicitamente quando isso ocorrer.
8. Dúvidas sobre Uber Go Style Guide: consultar
   [github.com/uber-go/guide/blob/master/style.md](https://github.com/uber-go/guide/blob/master/style.md).
