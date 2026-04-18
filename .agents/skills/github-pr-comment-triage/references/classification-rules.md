# Regras de Classificação

Usar estas regras apenas quando a heurística inicial do script não for suficiente.

## Classes

### `bug`
Usar quando o comentário aponta comportamento incorreto, falha lógica, regressão, condição quebrada ou ausência clara de tratamento esperado.

### `melhoria`
Usar quando o comentário pede refino de implementação, simplificação, legibilidade, desempenho ou manutenção sem afirmar falha direta.

### `sugestao`
Usar quando o comentário propõe alternativa, abordagem opcional ou ajuste discutível que pode ou não ser aplicado.

### `duvida`
Usar quando o comentário pede explicação, contexto, racional técnico ou esclarecimento de intenção.

### `nit`
Usar para detalhe pequeno de estilo, nomenclatura, formatação ou consistência local sem impacto funcional relevante.

### `documentacao`
Usar para pedidos de comentários, README, docstring, explicação de contrato, exemplos ou ajuste textual.

### `teste`
Usar quando o comentário pede teste novo, ajuste de cenário, cobertura ou validação automatizada.

### `risco`
Usar quando o comentário destaca segurança, concorrência, rollback, edge case, compatibilidade ou impacto operacional.

### `outro`
Usar apenas quando nenhuma classe acima representar o comentário sem forçar interpretação.

## Critérios de Resumo

1. Reduzir o comentário a uma intenção principal.
2. Preservar o verbo de ação implícito quando houver.
3. Não repetir saudações, agradecimentos ou texto de contexto já implícito.
4. Não copiar blocos longos do comentário bruto.

## Critérios de Ação Recomendada

1. Escrever uma ação verificável.
2. Preferir a menor mudança útil.
3. Referenciar arquivo e linha quando o comentário for inline.
4. Quando a resposta exigir apenas contexto, explicitar que a ação é responder e não editar código.
