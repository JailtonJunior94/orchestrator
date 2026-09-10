# Regras de Economia, Eficiencia e Robustez

Estas regras sao gates obrigatorios. Se qualquer gate critico falhar, a resposta correta e `nao aplicar padrao`.

## Gate 1: Solucao mais simples primeiro
- Preferir funcao, modulo, tabela de dispatch, extracao de metodo, extracao de tipo ou composicao direta antes de introduzir pattern formal.
- Rejeitar pattern quando houver apenas uma variante concreta e nenhuma pressao real de mudanca.
- Rejeitar pattern quando o ganho estiver apenas em "elegancia", sem reduzir custo ou risco.

## Gate 2: Economia total
- Medir economia em custo de mudanca, nao em beleza estrutural.
- Exigir evidencia de duplicacao recorrente, variacao recorrente, branching recorrente ou acoplamento recorrente.
- Rejeitar pattern se ele adicionar mais tipos, indirecao e pontos de falha do que remove.
- Penalizar fortemente patterns que criem hierarquias, wrappers ou factories sem necessidade operacional.

## Gate 3: Eficiencia
- Separar eficiencia de execucao e eficiencia de manutencao.
- So vender ganho de execucao quando houver impacto plausivel em hot path, memoria, I/O, concorrencia ou isolamento de recurso.
- So vender ganho de manutencao quando o pattern reduzir branching, duplicacao, acoplamento, dependencia de concrete classes ou churn de mudanca.
- Nao alegar performance sem mecanismo claro.

## Gate 4: Robustez
- Exigir descricao explicita de quais falhas o pattern ajuda a evitar.
- Preservar contratos, invariantes e testabilidade.
- Rejeitar pattern que torne falha mais opaca, debugging mais caro ou sequencia de execucao mais dificil de rastrear.

## Gate 5: Barra alta para overengineering
Patterns abaixo exigem no minimo dois sinais fortes e ganho operacional claro:
- Abstract Factory
- Builder
- Bridge
- Flyweight
- Mediator
- Visitor
- Singleton

## Regras de recusa imediata
- `Singleton`: recusar por padrao quando o objetivo real for conveniencia, acesso facil, cache improvisado ou compartilhamento global sem governanca de concorrencia e teste.
- `Abstract Factory`: recusar quando houver apenas um produto ou uma familia unica.
- `Builder`: recusar quando um construtor simples, record, named arguments, literal de objeto ou factory pura resolver.
- `Visitor`: recusar quando a estrutura muda com frequencia ou quando bastar um metodo no proprio tipo.
- `Flyweight`: recusar quando nao houver pressao real de memoria ou cardinalidade alta de objetos equivalentes.

## Preferencias obrigatorias
- Preferir composicao a heranca quando ambos resolverem o problema.
- Preferir composicao a `Template Method` quando ambos resolverem o problema.
- Preferir desacoplamento local a framework estrutural grande.
- Preferir API explicita a magia indireta.
- Preferir tipos e modulos estaveis a arvores profundas de subclasses.
