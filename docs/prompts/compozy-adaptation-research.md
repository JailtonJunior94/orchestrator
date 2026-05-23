# Prompt Enriquecido: Pesquisa e Adaptação para Padrões Compozy

## Prompt Original
> Eu quero saber efetivamente, como esse projeto: https://github.com/compozy/compozy conversa com os modelos (LLM) (mandatório)
> com base no meu codebase . o que falta pra eu fazer igual? crie um plano expecifico para adaptar o meu projeto  (mandatório)
> siga boas práticas implementadas em: https://github.com/compozy/compozy/tree/main  (mandatório)

---

## Prompt Enriquecido

**Atue como um Arquiteto de Software Sênior e Engenheiro de IA especialista em sistemas multi-agentes e orquestração de LLMs, focado em processos críticos de interoperabilidade de 2026.**

### 1. Contexto e Criticidade
Este é um **processo extremamente crítico**. O objetivo é garantir que o `ai-spec-harness` atinja paridade técnica com o estado da arte em orquestração de agentes, funcionando de forma impecável tanto **neste projeto quanto em qualquer outro projeto** onde for instalado.

### 2. Objetivo da Missão (Mandatório)
Realizar uma análise técnica profunda do repositório [Compozy](https://github.com/compozy/compozy) — **utilizando obrigatoriamente o GitHub CLI (`gh`) para explorar o código em `https://github.com/compozy/compozy/tree/main` em tempo real** — para identificar seus padrões de interação com LLMs e propor um plano de evolução.

**Restrição de Paridade (Invariante):** A solução deve funcionar de forma **IDÊNTICA** e portátil nos seguintes ambientes (May 2026 standard):
- `claude-code-cli`
- `codex-cli`
- `gemini-cli` / `antigravity`
- `copilot-cli`

### 3. Escopo da Investigação (Foco em Compozy e Portabilidade Global)
**Baseie-se exclusivamente em documentações oficiais e padrões de 2026.**

Analise detalhadamente no Compozy:
- **Abstração via ACP:** Como o protocolo garante consistência entre diferentes CLIs.
- **Padrões de Instalação e Bootstrap:** Como o `compozy setup` facilita a adoção em novos projetos.
- **Hierarquia de Configuração:** Uso de configuração global (`~/.compozy`) vs local (`.compozy`) para portabilidade absoluta.
- **Persistência de Estado:** Garantia de que fluxos são versionáveis e independentes de ambiente.

### 4. Análise de Gap e Ação Imediata
Compare os achados com o `ai-spec-harness` atual (`internal/runtime`, `internal/install`, `internal/config`).

**A entrega final deve ser um relatório técnico que acione IMEDIATAMENTE a skill `create-prd` para documentar e iniciar a implementação/correção das lacunas identificadas.**

### 5. Plano de Adaptação e Instalação (Mandatório)
Crie um plano estruturado em fases, com foco em:
- **Fase 1: Agnosticismo de Protocolo:** Resposta idêntica para as 4 CLIs alvo.
- **Fase 2: Motor de Instalação Portátil:** Refatoração para bootstrap em < 30s em qualquer codebase.
- **Fase 3: Camada de Configuração Universal:** Suporte a múltiplos projetos sem fricção.
- **Fase 4: Validação de Paridade Extrema:** Matriz de testes cross-CLI e cross-project.

### 6. Critérios de Saída
- **Relatório de Análise 2026 (Crítico):** Pontos técnicos extraídos via `gh cli`.
- **Draft de PRD (via `create-prd`):** Requisitos funcionais para sanar os gaps.
- **Guia de Instalação Universal:** Procedimento para portabilidade imediata.
- **Roadmap de Paridade Total.**

---

## Justificativa das Alterações

1.  **Definição de Persona**: Garante que a análise seja técnica e arquitetural, não apenas superficial.
2.  **Delimitação de Pesquisa**: Substitui "conversa com modelos" por termos técnicos precisos (abstração de provedores, composição de prompts, etc.), garantindo que o agente procure as partes corretas do código.
3.  **Contextualização Local**: Fornece ao agente os pontos de entrada do projeto atual (`internal/runtime`, `internal/skills`), economizando tokens de pesquisa e focando a comparação.
4.  **Estrutura de Saída (Fases)**: Transforma um pedido genérico de "plano" em um roadmap de engenharia acionável e modular.
5.  **Restrição de Stack**: Reforça a necessidade de manter a compatibilidade com Go e ACP, evitando sugestões de mudança total de linguagem ou framework base.
6.  **Uso de Ferramenta Específica (gh CLI)**: Obriga o uso do GitHub CLI para garantir que a investigação seja baseada no estado atual e real do código do Compozy (`main` branch), evitando alucinações.
7.  **Paridade Multi-CLI (2026)**: Mandato de funcionamento idêntico entre Claude, Codex, Gemini e Copilot, seguindo padrões oficiais de 2026 (ACP).
8.  **Portabilidade Universal**: Garante que a solução funcione perfeitamente tanto neste projeto quanto em novos projetos onde for instalada.
9.  **Gatilho PRD-First**: Obriga que o resultado da pesquisa acione imediatamente a skill `create-prd`, seguindo a governança do repositório para correção de lacunas.
10. **Criticidade**: Marca o processo como extremamente crítico para elevar o nível de prioridade e rigor na execução.
