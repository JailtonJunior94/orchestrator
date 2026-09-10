# Evidence Collection

Usar este roteiro quando a decisao estiver ambigua ou com sinais incompletos.

## Rodada 1: Problema Observavel
- Qual comportamento atual esta errado, caro ou rigido?
- O problema aparece em criacao, composicao, algoritmo, estado, travessia, acesso a recurso ou coordenacao?
- O caso e greenfield ou codebase existente?

## Rodada 2: Alternativa Simples
- Qual seria a menor mudanca local possivel?
- Essa mudanca simples falha por custo recorrente, duplicacao, acoplamento ou risco?
- Ha mais de uma variante real ou apenas hipotese?

## Rodada 3: Sinais de Conflito
- `Strategy vs State`: ha transicoes de estado governando comportamento ou so troca de algoritmo?
- `Adapter vs Facade`: o problema e contrato incompatível ou excesso de complexidade operacional?
- `Adapter vs Proxy`: o problema e traducao de interface ou governanca de acesso?
- `Facade vs Proxy`: o problema e simplificar subsistema ou controlar recurso?
- `Decorator vs Proxy`: o objetivo e empilhar responsabilidade ou controlar acesso?
- `Composite vs Visitor`: a arvore precisa de interface comum ou de novas operacoes sobre tipos estaveis?

## Rodada 4: Implementacao
- A linguagem favorece composicao, callbacks, interfaces pequenas ou heranca?
- O pattern exigiria mais tipos do que o problema comporta?
- O plano de testes cobre os cenarios que justificam o pattern?
