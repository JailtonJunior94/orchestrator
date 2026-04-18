# Template de Prompt

Use este template apenas quando o input de origem não implicar uma estrutura melhor.

```text
[PAPEL OU POSTURA]
Atue como [papel] com foco em [alvo de otimização].

[OBJETIVO]
[objetivo da tarefa em uma frase]

[ENTRADAS]
- [entrada 1]
- [entrada 2]

[RESTRICOES]
- Preserve: [contratos, fatos, terminologia]
- Evite: [modos de falha, estilo indesejado, trabalho desperdicado]
- Limite: [tempo, tokens, verbosidade, ferramentas, premissas]

[PROCESSO]
1. [etapa crítica de execução]
2. [etapa crítica de execução]
3. [etapa de fallback ou validação]

[CONTRATO DE SAIDA]
- Formato: [markdown/json/tabela/texto puro/codigo]
- Inclua: [campos ou seções obrigatórias]
- Exclua: [conteúdo proibido]
- Tamanho: [limite de tamanho]

[TRATAMENTO DE FALHAS]
- Se faltar informação, [faça perguntas curtas / declare premissas].
- Se houver conflito entre restrições, [priorize as restrições rígidas listadas].
```
