# Regras de Contexto

Mantenha apenas o contexto que altere correção, segurança, conformidade ou formato de saída.

## Manter
- Material-fonte obrigatório que o modelo precise transformar ou analisar.
- Restrições inegociáveis como prazo, orçamento, APIs, políticas, guias de estilo e contratos de interface.
- Exemplos concretos apenas quando definirem o formato esperado ou a fronteira de decisão.
- Prioridades nomeadas quando o usuário trocar explicitamente uma qualidade por outra, como latência por profundidade.

## Comprimir
- Repetições do mesmo objetivo.
- Contexto narrativo que possa ser substituído por um resumo de uma linha.
- Requisitos longos em prosa que possam virar restrições em lista.
- Múltiplos exemplos que expressem o mesmo padrão.

## Remover
- Frases motivacionais, moldura retórica e texto de cortesia.
- Instruções abertas de brainstorming quando a tarefa exigir uma única saída direta.
- Fatos de fundo que não sejam referenciados por nenhum critério de aceite.
- Casos de borda opcionais quando o usuário priorizar explicitamente baixo custo ou baixa latência.

## Política para Informação Ausente
- Preserve entradas obrigatórias ausentes como placeholders quando o preenchimento for esperado do usuário.
- Converta lacunas realmente bloqueantes em uma pergunta curta de esclarecimento.
- Use premissas explícitas apenas quando a tarefa puder prosseguir com segurança sem perguntar antes.
