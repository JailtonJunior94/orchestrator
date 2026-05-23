# Prompt Enriquecido: Alinhamento Arquitetural com Compozy (Foco Claude-CLI 2026)

## Prompt Original
> Eu quero saber efetivamente, como esse projeto: https://github.com/compozy/compozy conversa com os modelos (LLM) (mandatório)
> com base no meu codebase . o que falta pra eu fazer igual? crie um plano expecifico para adaptar o meu projeto  (mandatório)
> siga boas práticas implementadas em: https://github.com/compozy/compozy/tree/main  (mandatório)
> Foco no app claude-cli em 2026 (mandatório)

---

## Prompt Enriquecido

### Persona e Objetivo
**Atue como um Arquiteto de Software Sênior e Engenheiro de IA Especialista em Protocolos de Agentes (ACP/MCP) e Agentes Autónomos de Codificação.**

Seu objetivo é realizar uma análise técnica exaustiva do repositório [Compozy](https://github.com/compozy/compozy) — **utilizando obrigatoriamente o GitHub CLI (`gh`) como ferramenta primária de investigação** — para decifrar seu modelo de interação com LLMs e projetar um plano de transição para o `ai-spec-harness` (Orchestrator). O foco desta adaptação é o ecossistema **Claude-CLI (Claude Code)** na visão de **2026**, alavancando a natureza "agent-first" do Claude, sua alta precisão em ferramentas e a governança rigorosa do Orchestrator baseada em `spec-hash`.

### 1. Pesquisa e Engenharia Reversa (Compozy Deep Dive)
**Utilize obrigatoriamente o GitHub CLI (`gh`) para explorar, listar e ler os arquivos do repositório Compozy diretamente, garantindo que a análise seja baseada no código-fonte real e atualizado.**

Investigue como o Compozy "conversa" com os modelos, focando em:
- **Orquestração de Eventos ACP:** Como o Compozy utiliza o Agent Control Protocol para gerenciar o loop de raciocínio-ação-observação?
- **Ecossistema de Ferramentas MCP:** De que forma ele expõe o sistema de arquivos, ferramentas de build e skills de governança através do Model Context Protocol?
- **Injeção de "Agent Rules":** Como as instruções procedurais (equivalentes ao nosso `CLAUDE.md`) são compiladas e injetadas para garantir que o modelo não ignore restrições de segurança e arquitetura.
- **Protocolo de Mensagens JSON-RPC:** Analise o schema de comunicação e como o contexto é "podado" ou compactado para manter a eficiência em sessões longas.

### 2. Análise de Gap (Harness vs. Compozy)
Compare a implementação atual do `ai-spec-harness` com os padrões de excelência do Compozy:
- **Runtime Agentic:** Nosso `internal/runtime` (Go) vs. o sistema de execução baseado em eventos do Compozy.
- **Gestão de Contexto Local:** Como tratamos o `CLAUDE.md` e os arquivos de regras hoje vs. a integração nativa via protocolos dinâmicos do Compozy.
- **Validação de Evidências:** Nosso sistema de `execution_report.md` manual vs. o loop de feedback automatizado e integrado do Compozy.
- **Interação com Claude Code:** Como o Orchestrator invoca o `claude` hoje vs. como ele poderia ser um "servidor de habilidades" para o Claude.

### 3. Plano de Adaptação para Claude-CLI (Vision 2026)
Crie um roadmap técnico específico para transformar o `claude-cli` em um orquestrador de elite, mantendo as invariantes de `spec-hash` e `PRD-first`:

- **Fase 1: Claude-ACP Bridge:** Evoluir o `runtime` para suportar o protocolo ACP de forma bidirecional, permitindo que o Claude Code chame skills do Orchestrator como ferramentas nativas.
- **Fase 2: Governança via MCP Server:** Implementar um servidor MCP interno que exponha o `ai-spec-harness` como um recurso, garantindo que o Claude valide o `spec-hash` e o `drift` em tempo real antes de cada edição.
- **Fase 3: Loop de Refinamento e Revisão:** Adotar o padrão de "Auto-Review" do Compozy, onde o Claude Code invoca a skill `review` internamente antes de sinalizar o término de uma task.
- **Fase 4: Memória de Projeto Evolutiva:** Migrar de um `MEMORY.md` estático para um sistema de indexação de contexto inspirado no Compozy, otimizando o consumo de tokens para grandes refatorações.

### 4. Critérios de Aceitação (Saída Esperada)
- **Relatório de Mecânica Claude-native:** Descrição técnica de como o Compozy otimiza chamadas de ferramentas e gestão de contexto para modelos Anthropic.
- **Tabela de Gaps e Paridade:** Comparação funcional direta entre o Orchestrator atual e o estado da arte do Compozy.
- **Draft de Especificação Técnica (TechSpec):** Mudanças estruturais sugeridas em `internal/runtime/` e `.agents/skills/` para suportar o novo modelo.
- **Configuração Claude-2026:** Exemplo de como um arquivo `CLAUDE.md` e a estrutura em `.claude/` devem evoluir para esta visão de 2026.

---

## Justificativa do Enriquecimento

1.  **Foco na Natureza Agêntica do Claude**: Claude Code é desenhado para ser um agente que toma decisões. O prompt foca em como o Compozy potencializa essa autonomia mantendo o controle (ACP/MCP).
2.  **Integração de Governança**: Diferente de uma pesquisa genérica, este prompt exige que a adaptação mantenha o `spec-hash` e `PRD-first`, que são o diferencial competitivo do Orchestrator.
3.  **Tecnologias de 2026**: Foca explicitamente em MCP e ACP, que são os padrões de interoperabilidade de agentes para o futuro próximo.
4.  **Estrutura de Fases**: Transforma um pedido de "plano" em um roadmap de engenharia de software real, pronto para ser decomposto em tarefas.
