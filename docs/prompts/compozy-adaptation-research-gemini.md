# Prompt Enriquecido: Alinhamento Arquitetural com Compozy (Foco Gemini-CLI 2026)

## Prompt Original
> Eu quero saber efetivamente, como esse projeto: https://github.com/compozy/compozy conversa com os modelos (LLM) (mandatório)
> com base no meu codebase . o que falta pra eu fazer igual? crie um plano expecifico para adaptar o meu projeto  (mandatório)
> siga boas práticas implementadas em: https://github.com/compozy/compozy/tree/main  (mandatório)
> Foco no app gemini-cli em 2026 (mandatório)

---

## Prompt Enriquecido

### Persona e Objetivo
**Atue como um Arquiteto de Software Sênior e Engenheiro de IA Especialista em Protocolos de Agentes (ACP/MCP) e LLMs de Longo Contexto.**

Seu objetivo é realizar uma análise técnica exaustiva do repositório [Compozy](https://github.com/compozy/compozy) utilizando obrigatoriamente o **GitHub CLI (`gh`)** para explorar o código, decifrar seu modelo de interação com LLMs e projetar um plano de transição para o `ai-spec-harness` (Orchestrator). O foco desta adaptação é o ecossistema **Gemini-CLI** na visão de **2026**, alavancando as capacidades de raciocínio e a imensa janela de contexto do Gemini com a governança rigorosa do Orchestrator.

### 1. Pesquisa e Engenharia Reversa (Compozy Deep Dive)
Utilize comandos como `gh repo view compozy/compozy --web` (para contexto visual se necessário) e principalmente `gh api /repos/compozy/compozy/contents/path` ou ferramentas de busca de código para investigar como o Compozy "conversa" com os modelos, focando em:
- **Padrão de Orquestração ACP:** Como ele utiliza o Agent Control Protocol para gerenciar o loop de eventos entre o modelo e o sistema de arquivos?
- **Exposição de Ferramentas via MCP:** De que forma ele expõe skills e recursos dinâmicos através do Model Context Protocol?
- **Compilação de Instruções Procedurais:** Como os arquivos de instrução (equivalentes ao nosso `GEMINI.md`) são processados e injetados no System Prompt para garantir aderência a fluxos complexos.
- **Gestão de Sessão e Estado:** Como o Compozy mantém o estado entre diferentes invocações de subagentes.

### 2. Análise de Gap (Harness vs. Compozy)
Compare a implementação atual do `ai-spec-harness` com os padrões de excelência do Compozy:
- **Runtime de Eventos:** Nosso `internal/runtime` atual (Go) vs. o sistema de execução assíncrono do Compozy.
- **Injeção de Contexto:** Como tratamos o `GEMINI.md` hoje (manual/carregamento por prompt) vs. a integração nativa via protocolo do Compozy.
- **Hooks de Ciclo de Vida:** Nossa automação atual baseada em shell scripts em `.gemini/hooks/` vs. o sistema formal de ganchos de execução do Compozy.
- **Autonomia vs. Governança:** Como o Compozy equilibra a autonomia do agente com as travas de segurança.

### 3. Plano de Adaptação para Gemini-CLI (Vision 2026)
Crie um roadmap técnico específico para transformar o `gemini-cli` em um orquestrador paritário ao Compozy, mantendo as invariantes de `spec-hash` e `PRD-first`:

- **Fase 1: Gemini-ACP Native Bridge:** Implementação de um adapter avançado em Go que transforme o `gemini-cli` em um cliente ACP de primeira classe, permitindo streaming de eventos e chamadas de ferramentas granulares.
- **Fase 2: Orquestração Baseada em GEMINI.md:** Transformar as orientações de `GEMINI.md` em um servidor de contexto dinâmico que o Gemini consome via MCP, garantindo que o agente nunca desvie do procedimento operacional padrão.
- **Fase 3: Loop de Auto-Validação Proativa:** Implementar o padrão de "Refining Loop" do Compozy, onde o Gemini-CLI valida sua própria saída contra o `execution_report.md` antes de sinalizar a conclusão da tarefa.
- **Fase 4: Gestão de Contexto de Longa Duração:** Otimizar o uso da janela de 1M+ de tokens do Gemini para manter o grafo de conhecimento do projeto (CODEX.md) sempre quente, reduzindo a latência de "cold starts" de subagentes.

### 4. Critérios de Aceitação (Saída Esperada)
- **Documento de Arquitetura Comparativa:** Detalhando como o Compozy resolve o problema de "conversa com modelos" e como o Orchestrator deve evoluir.
- **Draft de Especificação Técnica:** Sugestão de mudanças em `internal/runtime/` e `internal/invocation/` para suportar o novo modelo.
- **Modelo de Configuração Gemini-native:** Exemplo de como o arquivo `GEMINI.md` e a estrutura em `.gemini/` devem evoluir para esta visão de 2026.
- **Avaliação de Confiabilidade:** Nota técnica sobre a viabilidade desta integração mantendo o rigor do SDD (Software Development Design).

---

## Justificativa do Enriquecimento

1.  **Contexto Específico do Gemini-CLI**: Diferente do Codex ou Copilot, o Gemini foca em janelas de contexto massivas e instruções procedurais fortes (`GEMINI.md`). O prompt foi enriquecido para destacar essa vantagem competitiva.
2.  **Tecnologias de 2026 (ACP/MCP)**: Garante que a pesquisa foque nos protocolos de mercado mais avançados para interação entre agentes e ferramentas.
3.  **Equilíbrio Governança/Autonomia**: Mantém o foco no `spec-hash` e `PRD-first` (invariantes do projeto) enquanto busca a sofisticação de interação do Compozy.
4.  **Estrutura de Fases**: Transforma um desejo de "plano" em um roadmap de engenharia claro e acionável para o time de desenvolvimento.
