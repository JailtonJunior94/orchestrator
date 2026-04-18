# Regras do Fluxo

Use este arquivo somente quando houver duvida sobre a ordem do processo.

Mantenha sempre esta sequencia:

1. validar origem e destino
2. pedir aprovacao para analisar
3. gerar changelog
4. pedir aprovacao do conteudo
5. resolver o destino final
6. pedir aprovacao para publicar
7. publicar
8. retornar rastreabilidade

Nao pule aprovacoes.

Nao publique quando:

- o usuario aprovou a analise, mas nao aprovou o conteudo
- o destino ainda esta ambiguo
- a origem do GitHub ainda nao foi resolvida com confianca

Use a skill de GitHub para a parte de leitura e sintese do diff.

Use a skill de Confluence somente quando o destino final for `confluence`.
