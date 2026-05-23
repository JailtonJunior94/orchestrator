# Prompt Enriquecido: Alinhamento Arquitetural com Compozy (Foco Codex-CLI 2026)

## Prompt Original
> Eu quero saber efetivamente, como esse projeto: https://github.com/compozy/compozy conversa com os modelos (LLM) (mandatório)
> com base no meu codebase . o que falta pra eu fazer igual? crie um plano expecifico para adaptar o meu projeto  (mandatório)
> siga boas práticas implementadas em: https://github.com/compozy/compozy/tree/main  (mandatório)
> Foco no app codex-cli em 2026 (mandatório)

---

## Prompt Enriquecido

### Persona e Objetivo
**Atue como um Arquiteto de Software Sênior e Engenheiro de IA Especialista em Protocolos de Agentes (ACP/MCP).**

Seu objetivo é realizar uma análise técnica exaustiva do repositório [Compozy](https://github.com/compozy/compozy) para decifrar seu modelo de interação com LLMs e projetar um plano de transição para o `ai-spec-harness` (Orchestrator). O foco desta adaptação é o ecossistema **Codex-CLI** na visão de **2026**, priorizando eficiência, economia de tokens e confiabilidade via governança rigorosa.

### 1. Pesquisa e Engenharia Reversa (Compozy Deep Dive)
**Mandatário: Utilize o GitHub CLI (`gh`) para realizar buscas, listar arquivos e ler o conteúdo do repositório `compozy/compozy` diretamente do terminal.**

Investigue como o Compozy "conversa" com os modelos, focando em:
- **Camada de Transporte ACP:** Como ele utiliza o Agent Control Protocol para orquestrar múltiplos turnos e subagentes?
- **Integração MCP (Model Context Protocol):** De que forma ele expõe ferramentas (tools) e recursos para o modelo de forma dinâmica?
- **Arquitetura de Mensagens:** Analise o schema de comunicação (JSON-RPC) e como o contexto é mantido entre execuções de comandos CLI.
- **Gestão de Prompt & Templates:** Como os sistemas de "instructions" (ex: `.compozy/instructions.md`) são compilados e injetados na janela de contexto.

### 2. Análise de Gap (Harness vs. Compozy)
Compare a implementação atual do `ai-spec-harness` com os padrões do Compozy:
- **Runtime:** Nosso `internal/runtime` (Go) vs. o sistema de execução do Compozy.
- **Memória:** Nosso sistema de `MEMORY.md` vs. o modelo de memória de dois níveis (compactação de contexto) do Compozy.
- **Hooks:** Nossa automação via shell scripts vs. o sistema formal de lifecycle hooks do Compozy.
- **Codex Integration:** Como o `codex-cli` interage hoje vs. como ele poderia se tornar um runtime nativo ACP.

### 3. Plano de Adaptação para Codex-CLI (2026 Vision)
Crie um roadmap técnico específico para transformar o Orchestrator em um sistema paritário ao Compozy, focado no `codex-cli`:

- **Fase 1: Fundação Codex-ACP:** Implementação de um adapter em Go que permita ao `codex-cli` atuar como um provedor de runtime compatível com as skills atuais.
- **Fase 2: Orquestração de Contexto Evolutiva:** Implementação de "Context Window Management" inspirado no Compozy, permitindo que o `codex-cli` processe grandes codebases sem estourar limites de tokens.
- **Fase 3: Protocolo de Ferramentas (MCP):** Criação de um servidor MCP interno para expor nossas skills (`create-prd`, `execute-task`) como ferramentas nativas para o Codex.
- **Fase 4: Governança & Spec-Hash:** Integração do sistema de `spec-hash` do Orchestrator com a interface de feedback do Codex, garantindo que "nenhuma linha de código é escrita sem contrato".

### 4. Critérios de Aceitação (Saída Esperada)
- **Relatório de Mecânica LLM:** Descrição técnica de como o Compozy lida com `system prompts`, `tool calls` e `event loops`.
- **Tabela de Paridade:** Comparação funcional entre as duas arquiteturas.
- **Especificação de Arquitetura (Draft):** Mudanças sugeridas em `internal/runtime/` e `internal/skills/` para suportar o novo modelo.
- **Configuração Codex-native:** Exemplo de como um arquivo `CODEX.md` e `.codex/config.toml` deve ser estruturado para suportar esta visão.

---

## Justificativa do Enriquecimento

1.  **Terminologia Técnica (ACP/MCP):** Substitui termos vagos por protocolos de mercado (Agent Control Protocol / Model Context Protocol) que são o padrão para 2026, garantindo que o agente procure as implementações corretas.
2.  **Foco em Codex-CLI:** Direciona a adaptação especificamente para a stack de ferramentas Codex, diferenciando-a das implementações de Gemini ou Copilot.
3.  **Gestão de Contexto:** Adiciona a necessidade de "economia e eficiência" (mencionada na instrução do usuário) através de estratégias de compactação e janelas de contexto.
4.  **Invariantes de Projeto:** Mantém o foco na governança (`spec-hash`, `PRD-first`) que é a marca registrada do `ai-spec-harness`, garantindo que a adaptação não perca a essência do projeto original.
5.  **Estrutura de Fases:** Transforma um "plano" em um roadmap de engenharia executável.
