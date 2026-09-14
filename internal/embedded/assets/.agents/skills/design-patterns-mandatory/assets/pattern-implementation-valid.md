# Pattern Implementation Bundle

## Objetivo
Resultado esperado:
Isolar a politica de precificacao em estrategias substituiveis.

## Contratos preservados
- API publica:
Metodo de cotacao continua com a mesma assinatura.
- Invariantes:
Preco final nunca pode ser negativo.
- Comportamentos que nao podem mudar:
Canal atual continua retornando o mesmo valor para o mesmo input.

## Participantes concretos
- Papel do pattern:
Contexto de precificacao e estrategias concretas.
- Tipo ou modulo real:
`PricingService` e modulos `pricing-strategies`.

## Adaptacao para a linguagem
Paradigma:
OO leve com composicao.

Escolha estrutural:
Usar interface minima e injecao explicita da estrategia.

## Pseudocodigo adaptado
```text
interface PricingStrategy
  calculate(input)

class PricingService
  strategy
  quote(input) -> strategy.calculate(input)
```

## Plano de mudanca
1. Extrair interface da politica de precificacao.
2. Implementar estrategia atual preservando comportamento.
3. Injetar estrategia no servico principal.

## Testes obrigatorios
Teste positivo:
Cada estrategia calcula o valor esperado no canal correspondente.

Teste negativo:
Falhar de forma controlada quando a estrategia estiver ausente.

Teste de regressao:
Canal antigo continua retornando o mesmo valor de antes.

Teste de falha:
Erro interno da estrategia nao pode vazar sem contexto.

## Rollback mental
Sinal de que a implementacao ficou cara demais:
Mais classes do que politicas reais de negocio.

Acao corretiva:
Voltar para funcao parametrizada ou dispatch table.
