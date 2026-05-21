# Prompt Enriquecido: Pesquisa e Adaptação para Padrões Compozy

## Prompt Original
> Eu quero saber efetivamente, como esse projeto: https://github.com/compozy/compozy conversa com os modelos (LLM) (mandatório)
> com base no meu codebase . o que falta pra eu fazer igual? crie um plano expecifico para adaptar o meu projeto  (mandatório)
> siga boas práticas implementadas em: https://github.com/compozy/compozy/tree/main  (mandatório)

---

## Prompt Enriquecido

**Atue como um Arquiteto de Software Sênior e Engenheiro de IA especialista em sistemas multi-agentes e orquestração de LLMs.**

### 1. Contexto do Projeto Atual (ai-spec-harness)
O projeto `orchestrator` é um harness para CLIs de IA (Claude, Gemini, Codex) construído em **Go**. Ele utiliza o protocolo **ACP (Agent Control Protocol)** para comunicação, possui uma arquitetura baseada em **Skills** com validação de schema JSON e gerencia o ciclo de vida de execução através de um `ACPRunner` em `internal/runtime`.

### 2. Objetivo da Missão
Realizar uma análise técnica profunda do repositório [Compozy](https://github.com/compozy/compozy) — **utilizando obrigatoriamente o GitHub CLI (`gh`) para explorar o código em tempo real** — para identificar seus padrões de interação com LLMs e propor um plano de evolução para o `ai-spec-harness` que adote essas melhores práticas, mantendo a compatibilidade com o stack atual.

### 3. Escopo da Investigação (Foco em Compozy)
**Utilize obrigatoriamente o GitHub CLI (`gh`) para explorar e ler os arquivos do repositório Compozy diretamente, garantindo análise em tempo real do código.**

Analise detalhadamente os seguintes pontos no repositório Compozy:
- **Abstração de Provedores:** Como o sistema lida com diferentes LLMs (OpenAI, Anthropic, etc.)? Existe uma interface unificada ou camada de transporte?
- **Composição de Prompts:** Como os prompts são construídos, versionados e injetados com contexto?
- **Tool/Function Calling:** Qual o mecanismo para o modelo invocar ferramentas e como os resultados são processados?
- **Gerenciamento de Estado e Memória:** Como o histórico da conversa e o estado do sistema são preservados entre turnos?
- **Tratamento de Erros e Resiliência:** Como falhas de API, rate limits e alucinações de formato são tratadas?

### 4. Análise de Gap (Harness vs. Compozy)
Compare os achados acima com a implementação atual em:
- `internal/runtime/` (Runner, Client, Events)
- `internal/skills/` (Schema, Frontmatter, Discovery)
- `internal/invocation/` (Recursion limits)

Identifique o que falta no `ai-spec-harness` para atingir o nível de sofisticação e modularidade do Compozy.

### 5. Plano de Adaptação (Mandatório)
Crie um plano de ação técnico estruturado em fases:
- **Fase 1: Alinhamento Arquitetural:** Mudanças estruturais necessárias em `internal/`.
- **Fase 2: Camada de Abstração:** Definição de interfaces para provedores e prompts.
- **Fase 3: Implementação Piloto:** Sugestão de uma skill ou package para validar o novo padrão.
- **Fase 4: Validação e Paridade:** Como garantir que a nova abordagem não quebra as invariantes de governança atuais.

### 6. Critérios de Saída
- **Relatório de Análise:** Documento markdown com os pontos técnicos extraídos do Compozy.
- **Gap Map:** Tabela comparativa entre Compozy e ai-spec-harness.
- **Roadmap Técnico:** Lista de tarefas (estilo PRD/TechSpec) para a adaptação.

---

## Justificativa das Alterações

1.  **Definição de Persona**: Garante que a análise seja técnica e arquitetural, não apenas superficial.
2.  **Delimitação de Pesquisa**: Substitui "conversa com modelos" por termos técnicos precisos (abstração de provedores, composição de prompts, etc.), garantindo que o agente procure as partes corretas do código.
3.  **Contextualização Local**: Fornece ao agente os pontos de entrada do projeto atual (`internal/runtime`, `internal/skills`), economizando tokens de pesquisa e focando a comparação.
4.  **Estrutura de Saída (Fases)**: Transforma um pedido genérico de "plano" em um roadmap de engenharia acionável e modular.
5.  **Restrição de Stack**: Reforça a necessidade de manter a compatibilidade com Go e ACP, evitando sugestões de mudança total de linguagem ou framework base.
6.  **Uso de Ferramenta Específica (gh CLI)**: Obriga o uso do GitHub CLI para garantir que a investigação seja baseada no estado atual e real do código do Compozy, evitando alucinações baseadas em dados de treinamento datados.
