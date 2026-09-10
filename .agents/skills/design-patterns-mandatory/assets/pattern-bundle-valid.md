# Pattern Decision Bundle

## Contexto
Problema:
Politica de precificacao muda por canal.

Objetivo tecnico:
Trocar algoritmo sem ifs espalhados.

Restricoes inegociaveis:
Preservar contrato publico.

## Diagnostico do problema
Sintomas observados:
Branching por canal em multiplos pontos.

Causa estrutural provavel:
Politica variavel acoplada ao contexto.

## Evidencias
Evidencia de codigo (path:line ou greenfield):
src/pricing/service.ts:42

Evidencia de comportamento:
Mudanca de canal exige editar o mesmo fluxo.

Evidencia de restricao:
Contrato publico deve permanecer estavel.

## Alternativa mais simples rejeitada
Alternativa:
Tabela de if/else local.

Motivo da rejeicao:
Nao reduz churn recorrente.

## Padrao primario
Recomendar:
Strategy

Padrao complementar:
nenhum

## Padroes rejeitados
- State: nao ha transicoes de estado dirigindo o comportamento.

## Justificativa de economia
Custo evitado:
Reduz branching duplicado.

Retorno esperado:
Nova politica entra sem editar o contexto principal.

## Justificativa de eficiencia
Impacto em execucao:
Sem custo material relevante fora de uma indirecao controlada.

Impacto em manutencao:
Isola algoritmos variaveis.

## Justificativa de robustez
Falhas mitigadas:
Evita regressao ao trocar politica.

Invariantes protegidos:
Contrato publico do servico permanece estavel.

## Estrutura minima
Participantes:
Contexto e estrategias concretas.

Responsabilidades:
Contexto delega calculo; estrategia implementa politica.

## Fluxo
1. Contexto recebe a requisicao.
2. Contexto delega para a estrategia.
3. Estrategia devolve o valor calculado.

## Pseudocodigo canonico
```text
interface PricingStrategy
  calculate(input)

class PricingContext
  strategy
  quote(input) -> strategy.calculate(input)
```

## Mapeamento por paradigma
Paradigma alvo:
OO leve com interfaces pequenas.

Adaptacao:
Usar interface minima e composicao.

## Plano de implementacao ou refatoracao
Passos:
1. Extrair contrato da politica.
2. Mover cada algoritmo para uma estrategia.
3. Injetar a estrategia no contexto.

## Plano de testes
Teste positivo:
Cada politica calcula o valor esperado.

Teste negativo:
Politica desconhecida falha de forma controlada.

Teste de regressao:
Canal antigo continua produzindo o mesmo valor.

## Criterios de aceite
- Novo algoritmo entra sem alterar o contrato publico.

## Riscos e criterios de nao uso
Riscos:
Criar estrategias demais para variacoes triviais.

Nao usar quando:
Houver apenas uma politica estavel e sem perspectiva real de mudanca.
