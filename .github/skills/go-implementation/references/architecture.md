# Arquitetura Go

<!-- TL;DR
Especificidades de arquitetura Go: DI manual via construtores, estrutura de diretórios, layouts recomendados e organização de pacotes internos.
Keywords: arquitetura, di, construtor, diretórios, pacotes, internal, layout
Load complete when: tarefa envolve estrutura de projeto, injeção de dependências ou organização de pacotes Go.
-->

Principios gerais de arquitetura, DI e sinais de excesso estao em `shared-architecture.md` (agent-governance). Este arquivo cobre apenas especificidades Go.

## DI em Go
- DI manual via construtores. Wire/fx apenas quando arvore de dependencias justificar.

## Estrutura de Diretorios

### Projeto novo — layouts recomendados

#### Serviço HTTP/gRPC
```
cmd/<service>/main.go
internal/
  domain/<aggregate>/         # entidades, value objects, regras
  application/<usecase>/      # orquestração, interfaces de porta
  infra/<adapter>/            # repositórios, clients, messaging
  handler/                    # HTTP/gRPC handlers, DTOs, middlewares
```

#### Worker / Consumer
```
cmd/<worker>/main.go
internal/
  domain/
  application/
  infra/
```

#### Monolito modular
```
cmd/server/main.go
internal/
  <module>/
    domain/
    application/
    infra/
    handler/
```

#### CLI
```
cmd/<cli>/main.go              # bootstrap, root command
internal/
  cmd/                         # subcommands (cada arquivo = um comando)
  config/                      # flags, env, config file parsing
  output/                      # formatação de saída (table, JSON, text)
  domain/                      # lógica de negócio quando houver
  infra/                       # clients, filesystem, IO
```
- Root command em `main.go` com wiring de subcommands.
- Cada subcommand em arquivo separado dentro de `internal/cmd/`.
- Flags e args validados no command, lógica delegada para camada interna.
- Saída formatada em camada própria — não misturar `fmt.Println` com lógica.
- Usar `cobra` ou stdlib `flag` conforme complexidade; não impor framework para CLI de 2 comandos.

### Regras Go
- `cmd/` apenas bootstrap. `internal/` como default. `pkg/` apenas se genuinamente reutilizavel.
- Profundidade maxima: `internal/<camada>/<pacote>/`.

## Regras Estritas — Estrutura e Funcoes

### R0 — `init()` PROIBIDA `[HARD]`
A funcao `init()` e terminantemente proibida em qualquer arquivo Go de producao, teste ou
biblioteca. Sem excecoes. Motivos: ordem de execucao implicita/nao-deterministica, acesso a estado
global e IO sem controle, codigo de inicializacao impossivel de testar, dependencia oculta entre
pacotes via efeitos colaterais, goroutines sem mecanismo de shutdown (leak garantido — ver R5.34).

```go
// PROIBIDO
var _db *sql.DB
func init() { _db, _ = sql.Open("postgres", os.Getenv("DATABASE_URL")) }

// CORRETO — injecao explicita via construtor
type UserRepository struct{ db *sql.DB }
func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

// CORRETO — default via factory explicita
func defaultConfig() Config { return Config{Timeout: 30 * time.Second} }
```

Criterio de aceitacao: `grep -rn "^func init()" --include="*.go" .` nao deve retornar nada.

### R1 — Toda funcao deve ser metodo de struct `[HARD]`
Funcoes de dominio/aplicacao/infraestrutura DEVEM ser metodos de struct. Funcoes standalone
(top-level `func foo(...)`) sao proibidas nessas camadas. Logica de negocio e validacao ficam
atachadas ao tipo (`func (p *Payment) Validate() error`, `func (uc *UseCase) Execute(...)`).

Unicas excecoes permitidas (exaustivas):

| Contexto | Justificativa |
|----------|--------------|
| `func main()` | Ponto de entrada do runtime |
| `func New*(deps...) (*T, error)` | Construtores/factories — apenas validam invariantes e montam a struct |
| `func TestXxx(t *testing.T)` | Registrador de suite (`suite.Run(...)`) |
| Funcoes de `pkg/` utilitario sem estado | Ex.: `pkg/uuid/New() string` — sem estado nem dependencias injetaveis |

Criterio: `grep -rn "^func [^(]" --include="*.go" .` (excluindo as excecoes acima) nao deve
retornar funcoes nao autorizadas.

### R5.16 / R5.33 — `os.Exit`/`log.Fatal` apenas em `main`, saida unica `[HARD]`
`os.Exit` e `log.Fatal*` so podem existir em `main()`. Fora dela impossibilitam teste e pulam
`defer`. Prefira uma unica saida em `main`, delegando toda a logica para `run() error`.

```go
// CORRETO
func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cfg, err := loadConfig()
    if err != nil {
        return fmt.Errorf("carregar configuracao: %w", err)
    }
    // ...
    return nil
}
```

### R6.4 — Zero values uteis `[HARD]`
Projete structs cujo zero-value seja funcional e seguro. Construtores (`New*`) sao obrigatorios
apenas quando ha **invariantes a validar** ou **dependencias obrigatorias a injetar**. Nao criar
`NewConfig() *Config { return &Config{} }` sem invariante — o zero-value ja basta (`var buf bytes.Buffer`).

### R6.6 — Injecao de dependencia via construtor, zero estado global `[HARD]`
Todo estado que nao e constante de dominio puro deve ser injetado via construtor. Proibido: estado
mutavel em variaveis globais de pacote, singletons com `sync.Once` em codigo de producao (apenas
`main`), inicializacao lazy de dependencias via campo opcional nao injetado.

```go
// CORRETO — tudo injetado, zero estado global
type UserService struct {
    repo UserRepository
    obs  observability.Observability
}

func NewUserService(repo UserRepository, obs observability.Observability) *UserService {
    return &UserService{repo: repo, obs: obs}
}
```
