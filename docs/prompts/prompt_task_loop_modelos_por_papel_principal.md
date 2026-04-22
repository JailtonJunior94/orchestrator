# Prompt Enriquecido - Variante Principal

```text
Voce vai produzir um documento de requisitos em Markdown para evoluir o comando `task-loop` deste repositorio.

Objetivo macro da iniciativa:
Permitir que o task-loop escolha configuracoes de execucao por papel no fluxo de automacao de tasks, separando pelo menos:
1. modelo/agente de implementacao
2. modelo/agente de revisao

Exemplo de configuracao alvo:
- implementacao: Claude Sonnet 4.6
- revisao: GPT-5.4

O desenho precisa considerar operacao com os agentes e CLIs ja suportados no repositorio, incluindo:
- Claude Code
- Codex
- GitHub Copilot CLI

Direcao tecnica obrigatoria:
- considerar o uso de `https://github.com/coder/acp-go-sdk` como base para padronizar a invocacao e a conversa com provedores/agentes quando isso reduzir acoplamento entre ferramentas
- explicitar no documento quando o ACP deve ser adotado desde a primeira versao e quando deve ficar como evolucao posterior

Compatibilidade obrigatoria:
- a proposta deve preservar uma versao simples do task-loop equivalente ao comportamento atual, para uso sem configuracao avancada
- essa versao simples deve continuar permitindo execucao sequencial como hoje, com configuracao minima e menor risco operacional

Contexto obrigatorio do repositorio:
- linguagem principal: Go 1.26+
- CLI framework: cobra
- harness atual suporta Claude Code, Codex, Gemini CLI e GitHub Copilot
- a referencia funcional mais proxima esta em `tasks/prd-task-loop-sequential-execution`
- use esse PRD como base para exemplos de fluxo, cenarios de teste e comparacao de comportamento

Prompt base a ser preservado para execucao das tasks pelo loop:

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

O documento deve responder ao seguinte problema:
Hoje o task-loop escolhe a ferramenta de forma mais global e linear. A evolucao desejada e permitir configuracao por papel no workflow, por exemplo usar um agente/modelo para executar a task e outro para revisar, com possibilidade de operar isso por Claude Code, Codex e Copilot CLI, sem perder a simplicidade do fluxo atual.

Produza a saida em Markdown com as secoes abaixo, nesta ordem:
1. Problema
2. Objetivo
3. Nao objetivo
4. Requisitos funcionais
5. Requisitos nao funcionais
6. Riscos iniciais

Regras de elaboracao:
- escreva em PT-BR
- seja especifico sobre o que significa "modelo por papel": diferencie ferramenta CLI, provider, modelo e etapa do fluxo
- inclua requisitos para pelo menos dois modos de operacao:
  1. modo simples: comportamento proximo do task-loop atual
  2. modo avancado: configuracao por papel (execucao e revisao) com possibilidade de diferentes modelos/agentes
- inclua requisitos para fallback quando uma combinacao de ferramenta + modelo nao for suportada
- inclua requisitos para validacao e testes usando `tasks/prd-task-loop-sequential-execution` como fixture de execucao
- inclua requisitos para observabilidade e evidencias de qual combinacao foi usada em cada iteracao
- preserve compatibilidade com contratos publicos atuais, salvo quando o proprio documento sinalizar a necessidade de ADR
- nao invente dependencias adicionais alem do `acp-go-sdk` sem justificativa clara

Detalhamento minimo esperado em cada secao:

Problema:
- descreva as limitacoes atuais do task-loop para selecionar agente/modelo por etapa
- explique por que o fluxo atual dificulta combinar implementacao e revisao com ferramentas/modelos distintos

Objetivo:
- deixar claro o resultado esperado para o operador do CLI
- explicitar o ganho de flexibilidade sem degradar o fluxo simples

Nao objetivo:
- listar explicitamente itens fora de escopo, como paralelismo, UI grafica, suporte irrestrito a qualquer provider, ou reescrita completa do task-loop

Requisitos funcionais:
- configuracao do executor por papel
- configuracao do revisor por papel
- modo simples backward compatible
- validacao de compatibilidade entre ferramenta, provider e modelo
- resolucao de defaults
- persistencia/relatorio da configuracao usada na execucao
- integracao ou abstracao via ACP quando aplicavel
- estrategia de fallback quando ACP ou a ferramenta nao suportar uma combinacao desejada
- cobertura de testes de execucao baseada no PRD `tasks/prd-task-loop-sequential-execution`

Requisitos nao funcionais:
- manter arquitetura atual do repositorio
- preservar baixo acoplamento e evitar interfaces sem fronteira real
- garantir `context.Context` em IO
- manter testabilidade com FakeFileSystem e testes table-driven quando aplicavel
- impacto operacional previsivel
- logs e evidencias suficientes para troubleshooting

Riscos iniciais:
- diferencas de capacidade entre CLIs
- mapeamento incorreto entre ferramenta e modelo
- aumento de complexidade de configuracao
- risco de regressao no modo simples
- acoplamento excessivo ao ACP ou a um fornecedor especifico

Criterios de qualidade da resposta:
- cada requisito funcional deve ser observavel e testavel
- a secao de nao objetivo deve reduzir ambiguidades de escopo
- a proposta deve deixar claro se a primeira entrega usa ACP de forma obrigatoria ou incremental
- a saida deve ser suficientemente precisa para virar PRD ou tech spec sem reescrita total
```

## Justificativas das Adicoes

- Estruturei a saida em secoes obrigatorias para transformar uma intencao ampla em um artefato imediatamente utilizavel como insumo de PRD ou tech spec.
- Separei "ferramenta", "provider", "modelo" e "papel" para reduzir a principal ambiguidade do pedido.
- Tornei `tasks/prd-task-loop-sequential-execution` uma fixture explicita de teste, em vez de apenas uma referencia informal.
- Mantive o prompt operacional da skill `execute-task` dentro do contexto para preservar o contrato atual do loop e evitar perda de governanca durante a evolucao.
- Delimitei o papel do `acp-go-sdk` para evitar que ele vire premissa obrigatoria sem avaliacao de custo, ao mesmo tempo em que ele continua como direcao tecnica clara.
