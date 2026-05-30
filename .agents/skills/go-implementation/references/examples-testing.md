# Exemplos: Testes e Validacao

<!-- TL;DR
Exemplos de testes em Go: construtores com invariantes, table-driven tests, mocks, fakes e testes de integração com TempDir.
Keywords: exemplo, teste, table-driven, mock, fake, invariante, integração
Load complete when: tarefa requer exemplos concretos de testes unitários, de integração ou uso de fakes/mocks em Go.
-->

## Construtor com invariantes
```go
type Config struct {
    timeout time.Duration
}

func NewConfig(timeout time.Duration) (Config, error) {
    if timeout <= 0 {
        return Config{}, fmt.Errorf("timeout must be positive")
    }
    return Config{timeout: timeout}, nil
}
```

## Interface no consumidor
```go
type clock interface {
    Now() time.Time
}
```

## Fuzz test para parser/validador
```go
// domain/order/money_test.go
func FuzzParseMoney(f *testing.F) {
    f.Add("100.00")
    f.Add("0")
    f.Add("-1")
    f.Add("")
    f.Add("99999999.99")
    f.Add("not-a-number")

    f.Fuzz(func(t *testing.T, input string) {
        result, err := ParseMoney(input)
        if err != nil {
            return // input invalido e esperado — apenas nao deve panic
        }
        // round-trip: valor parseado deve ser re-serializavel
        assert.Equal(t, result.String(), ParseMoney(result.String()))
    })
}
```

## Table-driven test com testify
```go
func TestNormalize(t *testing.T) {
    tests := []struct {
        name string
        in   string
        want string
    }{
        {name: "trim", in: " a ", want: "a"},
        {name: "empty", in: "", want: ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Normalize(tt.in)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

## Esqueleto canonico testify/suite (R4 — use case / service / handler)
Padrao obrigatorio para testes de use case, service e handler: suite struct com mocks tipados,
registrador `suite.Run`, `SetupTest` reiniciando mocks e tabela `scenarios` com SUT instanciado
dentro do loop. Cenarios minimos: happy path, erro de validacao de dominio, erro de infraestrutura.

```go
package usecase // ou usecase_test para blackbox

import (
    "context"
    "errors"
    "testing"

    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/suite"

    "github.com/JailtonJunior94/devkit-go/pkg/observability"
    "github.com/JailtonJunior94/devkit-go/pkg/observability/fake"

    "github.com/seu-org/seu-projeto/internal/<dominio>/application/dtos"
    repositoryMock "github.com/seu-org/seu-projeto/internal/<dominio>/infrastructure/repositories/mocks"
)

// 1. Suite struct — um campo mock por dependencia
type CreateUserUseCaseSuite struct {
    suite.Suite

    ctx      context.Context
    obs      observability.Observability
    userMock *repositoryMock.UserRepository
}

// 2. Registrador — APENAS suite.Run
func TestCreateUserUseCaseSuite(t *testing.T) {
    suite.Run(t, new(CreateUserUseCaseSuite))
}

// 3. SetupTest — reinicia mocks a cada cenario
func (s *CreateUserUseCaseSuite) SetupTest() {
    s.obs = fake.NewProvider()
    s.ctx = context.Background()
    s.userMock = repositoryMock.NewUserRepository(s.T())
}

// 4. Metodo de teste principal — table-driven
func (s *CreateUserUseCaseSuite) TestExecute() {
    type args struct {
        input *dtos.UserInput
    }

    type dependencies struct {
        userMock *repositoryMock.UserRepository
    }

    scenarios := []struct {
        name         string
        args         args
        dependencies dependencies
        expect       func(output *dtos.UserOutput, err error)
    }{
        {
            name: "deve criar usuario com sucesso",
            args: args{input: &dtos.UserInput{Name: "Joao", Email: "joao@email.com"}},
            dependencies: dependencies{
                userMock: func() *repositoryMock.UserRepository {
                    s.userMock.
                        EXPECT().
                        Save(s.ctx, mock.AnythingOfType("*entities.User")).
                        Return(nil).
                        Once()
                    return s.userMock
                }(),
            },
            expect: func(output *dtos.UserOutput, err error) {
                s.NoError(err)
                s.NotNil(output)
            },
        },
        {
            name: "deve retornar erro ao validar input invalido",
            args: args{input: &dtos.UserInput{Name: "", Email: ""}},
            dependencies: dependencies{userMock: s.userMock},
            expect: func(output *dtos.UserOutput, err error) {
                s.Error(err)
                s.Nil(output)
            },
        },
        {
            name: "deve retornar erro ao salvar no repositorio",
            args: args{input: &dtos.UserInput{Name: "Joao", Email: "joao@email.com"}},
            dependencies: dependencies{
                userMock: func() *repositoryMock.UserRepository {
                    s.userMock.
                        EXPECT().
                        Save(s.ctx, mock.AnythingOfType("*entities.User")).
                        Return(errors.New("falha no banco")).
                        Once()
                    return s.userMock
                }(),
            },
            expect: func(output *dtos.UserOutput, err error) {
                s.Error(err)
                s.Nil(output)
            },
        },
    }

    for _, scenario := range scenarios {
        s.Run(scenario.name, func() {
            uc := NewCreateUserUseCase(s.obs, scenario.dependencies.userMock)
            output, err := uc.Execute(s.ctx, scenario.args.input)
            scenario.expect(output, err)
        })
    }
}
```
