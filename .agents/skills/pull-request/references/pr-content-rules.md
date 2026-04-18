# Regras de Conteúdo da PR

Use estas regras quando o diff e os commits não bastarem para decidir como escrever o título e o body.

## Título

### Quando usar formato tipo Conventional Commit
Usar `type(scope-opcional): descrição` somente quando ao menos uma destas condições for verdadeira:
- o repositório já usa esse padrão em títulos de PR ou commits recentes
- o usuário pedir explicitamente
- a equipe tiver convenção conhecida e consistente

### Quando não forçar Conventional Commits
Não forçar esse formato quando:
- a convenção do repositório não estiver clara
- a PR representar uma mudança ampla demais para um único `type`
- o padrão do time for mais descritivo do que tipado

### Boas características do título
- descreve o objetivo principal em uma linha
- evita nomes de arquivo
- evita frases vagas
- não promete mais do que o diff entrega

## Body

### Seções recomendadas
- `## Resumo`
- `## O que mudou`
- `## Validação`
- `## Riscos e observações`

### Seções opcionais
- `## Breaking changes`
- `## Pendências`
- `## Ticket ou contexto adicional`

### Regras de preenchimento
- preencher apenas seções sustentadas por evidência
- usar bullets curtos para mudanças concretas
- separar `o que mudou` de `como foi validado`
- marcar pendências reais sem transformar hipótese em fato

## Checklist
O checklist deve refletir somente itens verificáveis, por exemplo:
- testes automatizados executados
- documentação atualizada
- migração necessária
- monitoramento ou rollout pendente

Se a evidência não existir, deixar o item desmarcado ou omitir a seção.

## Breaking Changes
Declarar breaking change apenas quando houver quebra concreta de contrato, compatibilidade ou fluxo consumido externamente. Se houver risco mas não certeza, descrever como risco, não como fato consumado.
