# Generics

<!-- TL;DR
Diretrizes para uso de generics em Go: quando usar para coleções tipadas, mappers e result types, e quando preferir implementações concretas.
Keywords: generics, tipo, constraint, coleção, mapper, result-type, reutilização
Load complete when: tarefa envolve criação ou revisão de código com generics, constraints ou tipos parametrizados em Go.
-->

## Quando usar
- Quando houver algoritmo ou estrutura reutilizável para múltiplos tipos com a mesma semântica.
- Quando a alternativa seria duplicação relevante ou uso inseguro de `any`.
- Coleções tipadas, mappers, filtros, result types.

## Quando evitar
- Quando uma implementação concreta é mais clara.
- Quando a constraint fica mais complexa que o ganho obtido.
- Quando generics apenas escondem falta de modelagem de domínio.
- Quando o tipo parametrizado tem apenas um uso real.

## Diretrizes
- Preferir constraints mínimas e explícitas.
- Usar constraints da stdlib (`comparable`, `any`) e de `golang.org/x/exp/constraints` quando disponíveis.
- Evitar APIs genéricas excessivamente abstratas.
- Validar se a versão de Go do projeto (`go.mod`) suporta a solução proposta (generics requer Go 1.18+).
- Nomear type parameters com letras descritivas quando o contexto exigir clareza: `[K comparable, V any]` em vez de `[T any, U any]`.

## Padrões úteis

### Map/Filter/Reduce
```go
func Map[T, R any](items []T, fn func(T) R) []R {
    result := make([]R, len(items))
    for i, item := range items {
        result[i] = fn(item)
    }
    return result
}
```

### Result type
```go
type Result[T any] struct {
    Value T
    Err   error
}

func OK[T any](value T) Result[T] {
    return Result[T]{Value: value}
}

func Fail[T any](err error) Result[T] {
    return Result[T]{Err: err}
}
```

### Constraint customizada
```go
type Identifiable interface {
    ID() string
}

func FindByID[T Identifiable](items []T, id string) (T, bool) {
    for _, item := range items {
        if item.ID() == id {
            return item, true
        }
    }
    var zero T
    return zero, false
}
```

## Riscos Comuns
- Função genérica com constraint `any` que na verdade precisa de `comparable` ou outra — erro em runtime que seria pego em compile time.
- Abstração genérica criada para um único tipo — overhead sem ganho.
- Type parameter propagado por múltiplas camadas tornando assinaturas ilegíveis.

## Proibido
- Generics como substituto de `interface{}` sem ganho real de type safety.
- Constraint que depende de reflect para funcionar.
- Função genérica que internamente faz type assertion — anula o propósito.
