# Regras de Mapeamento por Paradigma

Traduzir o pseudocodigo do pattern para a forma mais simples que preserve a intencao. Nao copiar mecanicamente diagramas de UML para linguagens que nao se beneficiam disso.

## OO classico
- Usar interface ou classe abstrata quando houver polimorfismo real.
- Concentrar variacao em tipos pequenos e coesos.
- Evitar base classes com muitos hooks.
- `Template Method` so e aceitavel quando a heranca ja e natural no dominio.

## OO leve ou linguagens com tipos estruturais
- Preferir interfaces pequenas, objetos literais, records e composicao.
- `Strategy`, `Observer`, `Adapter`, `Decorator` e `Proxy` podem ser modelados com funcoes, closures ou objetos simples.
- Evitar fabricar factories ou builders verbosos se a linguagem ja oferece construcao declarativa barata.

## Funcional
- Substituir classes por funcoes puras, algebraic data types, tabelas de dispatch ou closures quando isso mantiver a semantica.
- `Strategy` vira selecao explicita de funcao.
- `Command` vira payload imutavel + handler deterministico.
- `State` vira maquina de estados explicita com transicoes puras.
- `Observer` vira stream, signal ou pub/sub somente quando o runtime ja suporta isso com rastreabilidade.

## Hibrido
- Misturar objetos e funcoes apenas quando cada parte tiver papel claro.
- Isolar efeito colateral nas bordas.
- Evitar wrappers multiplos se uma funcao de alto nivel resolver.

## Mapeamento de estruturas comuns
- `Abstract Factory`: modulo fabricante ou objeto factory; evitar classes de fabrica quando funcoes nomeadas bastarem.
- `Builder`: objeto acumulador ou pipeline imutavel; evitar builder mutavel se a linguagem tiver bons valores default.
- `Bridge`: separar dimensao de abstracao e implementacao apenas se ambas variarem independentemente.
- `Composite`: usar arvore com interface minima; nao criar composite se a estrutura nao for realmente recursiva.
- `Visitor`: usar somente quando a estrutura e estavel e as operacoes mudam bastante.

## Regras de robustez
- Preservar mensagens de erro e contratos existentes.
- Garantir que o mapeamento nao esconda ordem de execucao, ownership de recurso ou limites de transacao.
- Escolher a adaptacao com menor surpresa para a equipe que mantera o codigo.
