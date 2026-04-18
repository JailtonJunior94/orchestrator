# Regras de Enriquecimento

Aplique apenas as regras que melhorarem o modo de tarefa detectado.

## Robustez
- Substitua verbos amplos como "melhorar" ou "tratar" por ações observáveis.
- Adicione proteções contra falha para ambiguidade, ausência de dados, input malformado e restrições contraditórias.
- Defina critérios de aceite para que o modelo possa se verificar antes de responder.
- Prefira esquemas explícitos e sequências determinísticas em vez de prompts de raciocínio livre.

## Performance
- Reduza instruções com muitas ramificações e subobjetivos opcionais.
- Remova contexto duplicado e mantenha apenas fatos críticos para execução.
- Peça a resposta final diretamente, salvo quando raciocínio intermediário for necessário para correção.
- Limite tamanho da saída, número de alternativas e grau de elaboração.

## Assertividade
- Exija recomendações diretas, opções ranqueadas ou uma única melhor resposta quando a tarefa permitir.
- Instrua o modelo a declarar premissas brevemente em vez de hedgear em excesso.
- Proíba disclaimers genéricos, salvo quando o domínio for de alto risco ou regulado.
- Prefira verbos concretos como "escolha", "classifique", "reescreva", "retorne" ou "corrija".

## Economia
- Limite seções, bullets, linhas, exemplos e repetições pesadas em tokens.
- Coloque restrições simples em linha em vez de criar cabeçalhos extras.
- Evite texto de persona, salvo quando ele alterar a qualidade da saída de forma mensurável.
- Use contratos de saída compactos, como schemas JSON, cabeçalhos fixos ou resposta em tabela única.

## Orientação por Modo

### Execution
- Defina ferramentas, entradas, limites e critérios de conclusão.
- Adicione fallback para dependências ausentes ou ferramentas indisponíveis.

### Transformation
- Preserve primeiro o significado de origem; otimize estilo ou formato em segundo plano.
- Defina o que deve permanecer inalterado.

### Decision
- Defina critérios de avaliação e o formato esperado para a recomendação final.
- Exija visibilidade de tradeoffs apenas quando isso afetar a escolha final.

### Generation
- Especifique artefato-alvo, interface, restrições e expectativa de validação.
- Exija apenas o nível de explicação necessário para implementação ou revisão.
