## Prompt Enriquecido

```text
Escreva um documento em Markdown, em PT-BR, para definir a evolucao do `task-loop` deste repositorio com foco em uma entrega incremental e segura.

Contexto:
- hoje o task-loop executa tasks de forma sequencial
- quero manter esse fluxo simples como padrao
- tambem quero abrir espaco para escolher configuracoes diferentes por papel, por exemplo:
  - execucao com Claude Sonnet 4.6
  - revisao com GPT-5.4
- a solucao deve considerar Claude Code, Codex e GitHub Copilot CLI
- avalie o uso do `acp-go-sdk` como mecanismo de abstracao, mas sem forcar reescrita ampla na primeira fase
- use `tasks/prd-task-loop-sequential-execution` como base para cenarios de teste e exemplos de execucao

Preserve no contexto da proposta o prompt operacional abaixo:

Use a skill execute-task para implementar a <tasks>

Contexto obrigatorio:
- Leia o arquivo de task antes de iniciar qualquer alteracao
- Arquitetura: [descreva a camada e os contratos relevantes]
- Referencias a carregar: [go-implementation, ddd, tests - conforme a task]

Criterios de execucao nao negociaveis:
- preservar contratos publicos existentes (nenhuma assinatura publica muda sem ADR)
- nenhuma interface nova sem fronteira real justificada
- context.Context em todas as operacoes de IO
- testes table-driven para todos os cenarios do criterio de pronto
- registrar evidencia de conclusao no arquivo de task (output do teste, lint)
- nao fechar a task sem evidencia de validacao

Gere a resposta com estas secoes:
1. Problema
2. Objetivo
3. Nao objetivo
4. Requisitos funcionais
5. Requisitos nao funcionais
6. Riscos iniciais

Restricoes:
- incluir um modo simples como default
- incluir uma evolucao opcional para configuracao por papel
- dizer como validar a feature com testes usando o PRD `tasks/prd-task-loop-sequential-execution`
- nao propor mudancas amplas sem necessidade comprovada
```

## Justificativas das Adicoes

- Estruturei a saida em secoes obrigatorias para transformar uma intencao ampla em um artefato imediatamente utilizavel como insumo de PRD ou tech spec.
- Mantive a separacao entre modo simples e evolucao opcional para respeitar o pedido de preservar o fluxo atual.
- Tornei `tasks/prd-task-loop-sequential-execution` uma fixture explicita de teste, em vez de apenas uma referencia informal.
- Mantive o prompt operacional da skill `execute-task` dentro do contexto para preservar o contrato atual do loop e evitar perda de governanca durante a evolucao.
- Delimitei o papel do `acp-go-sdk` como avaliacao incremental para evitar reescrita ampla na primeira entrega.
